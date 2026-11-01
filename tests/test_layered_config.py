import datetime
import subprocess
import tempfile
from pathlib import Path

import pytest
import yaml

from tests.utils import (
    FIXTURES_DIR,
    read_avro_file_with_schema,
    run_csv_api,
    run_fixture,
    run_validate_config,
)

DROP_DATA_FAILURES = ["--error-on-data-failures=false"]


def print_merged(build_path, config_path) -> subprocess.CompletedProcess:
    """Run `validate_config --print-merged` on a config file."""
    cmd = [str(build_path), "validate_config", "-c", str(config_path), "--print-merged"]
    return subprocess.run(cmd, capture_output=True, text=True, timeout=60)


def merged_config(build_path, fixture_name) -> dict:
    """The merged config of a fixture's config.yaml, as printed by --print-merged."""
    proc = print_merged(build_path, FIXTURES_DIR / fixture_name / "config.yaml")
    assert proc.returncode == 0, proc.stderr
    return yaml.safe_load(proc.stdout)


def field(config: dict, name: str) -> dict:
    return next(f for f in config["fields"] if f["name"] == name)


def patterns(config: dict, name: str) -> list[str]:
    return [rep["pattern"] for rep in field(config, name)["representations"]]


def test_extending_config_is_merged_and_keeps_base_schema(build_path):
    """TASK-10 AC#1: a config with extends is merged with its base for parse
    (the producer's header_patterns and extra date format are used), and the
    output schema is exactly the one the base config alone produces.
    TASK-10 AC#3: extends: base/base.yaml is resolved relative to the
    extending file (the tests run from the repo root, not the fixture dir)."""
    result = run_fixture(build_path, "layered_basic")
    assert result.returncode == 0, result.stderr
    assert result.records == [
        {"id": 1, "order_date": datetime.date(2024, 3, 15), "notes": None},
        {"id": 2, "order_date": datetime.date(2024, 3, 16), "notes": None},
    ]

    base_dir = FIXTURES_DIR / "layered_basic" / "base"
    with tempfile.TemporaryDirectory() as tmpdir:
        output_path = Path(tmpdir) / "base.avro"
        proc = run_csv_api(build_path, base_dir / "base.yaml", base_dir / "data.csv", output_path)
        assert proc.returncode == 0, proc.stderr
        _, base_schema = read_avro_file_with_schema(output_path)
    assert result.writer_schema == base_schema
    assert base_schema["name"] == "com.example.orders"


def test_chain_merges_root_first_later_files_winning(build_path):
    """TASK-10 AC#2: config.yaml (a) extends shared/producer.yaml (b) extends
    shared/root.yaml (c); settings merge in order c, b, a with later files
    winning: header_patterns come from a, error_message from b (a doesn't set
    it), allow_extra_fields: false from b is undone by a, and the
    representations are a's prepended, then c's, then b's appended.
    TASK-10 AC#3: b's 'extends: root.yaml' is resolved relative to b, not to a."""
    merged = merged_config(build_path, "layered_chain")
    assert merged["allow_extra_fields"] is True
    assert field(merged, "day")["header_patterns"] == ["^Day$"]
    assert field(merged, "day")["error_message"] == "from b"
    assert field(merged, "id")["error_message"] == "from c"
    assert patterns(merged, "day") == [
        "^start$",
        "^(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})$",
        "^(?P<day>[0-9]{2})/(?P<month>[0-9]{2})/(?P<year>[0-9]{4})$",
    ]

    # and parse uses the same merged config
    result = run_fixture(
        build_path, "layered_chain", extra_args=DROP_DATA_FAILURES, data_failure_report=True
    )
    assert result.returncode == 0, result.stderr
    assert result.records == [
        {"id": 1, "day": datetime.date(2020, 1, 1)},
        {"id": 2, "day": datetime.date(2024, 3, 15)},
        {"id": 3, "day": datetime.date(2024, 3, 16)},
    ]
    assert result.data_failure_rows[1][:3] == ["4", "16.03.2024", "x"]
    assert "(note: from b)" in result.data_failure_rows[1][3]


def test_prepend_and_append_add_to_the_base_list(build_path):
    """TASK-10 AC#6: prepend_representations go before the base list (so the
    producer's format takes priority) and append_representations after it."""
    merged = merged_config(build_path, "layered_basic")
    assert patterns(merged, "order_date") == [
        "^(?P<day>[0-9]{2})/(?P<month>[0-9]{2})/(?P<year>[0-9]{4})$",
        "^(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})$",
    ]
    # the chain fixture appends in b and prepends in a (see AC#2 test)
    chain = merged_config(build_path, "layered_chain")
    assert patterns(chain, "day")[-1] == (
        "^(?P<day>[0-9]{2})/(?P<month>[0-9]{2})/(?P<year>[0-9]{4})$"
    )


def test_representations_replace_the_base_list(build_path):
    """TASK-10 AC#6: representations in an extending file replace the base
    list, so the base's ISO format no longer matches."""
    merged = merged_config(build_path, "layered_replace")
    assert patterns(merged, "order_date") == [
        "^(?P<year>[0-9]{4})(?P<month>[0-9]{2})(?P<day>[0-9]{2})$"
    ]
    result = run_fixture(build_path, "layered_replace", extra_args=DROP_DATA_FAILURES)
    assert result.returncode == 0, result.stderr
    assert result.records == [{"id": 1, "order_date": datetime.date(2024, 3, 15), "notes": None}]
    assert "warning: dropped 1 row(s) with invalid data" in result.stderr


def test_extending_file_without_fields(build_path):
    """TASK-10: an extending file may leave out fields and only set a
    top-level input-side setting (allow_extra_fields: false here)."""
    result = run_fixture(build_path, "layered_no_fields")
    assert result.returncode != 0
    assert not result.output_exists
    assert "header 'source' (column 3) matches no field" in result.stderr


def test_required_can_be_overridden(build_path):
    """TASK-10 (decision-13): required is input-side, so a producer may make a
    nullable base field optional; its missing column is null."""
    result = run_fixture(build_path, "layered_required")
    assert result.returncode == 0, result.stderr
    assert result.records == [{"id": 1, "notes": None}]


@pytest.mark.parametrize(
    "fixture_name, expected",
    [
        ("layered_override_logical_type", "field 'id': can't set 'logical_type'"),
        ("layered_override_name", "can't set the schema 'name'"),
        ("layered_override_nullable", "field 'id': can't set 'nullable'"),
    ],
)
def test_schema_side_setting_fails_naming_file_and_setting(build_path, fixture_name, expected):
    """TASK-10 AC#4: an extending file that sets a field's logical_type (or
    nullable), or the schema name, fails the load naming the file and setting,
    for both parse and validate_config."""
    config_path = str(FIXTURES_DIR / fixture_name / "config.yaml")
    result = run_fixture(build_path, fixture_name)
    assert result.returncode != 0
    assert not result.output_exists
    assert f"config '{config_path}'" in result.stderr
    assert expected in result.stderr

    proc = run_validate_config(build_path, fixture_name)
    assert proc.returncode != 0
    assert f"config '{config_path}'" in proc.stderr
    assert expected in proc.stderr


def test_field_not_in_base_fails_naming_file_and_field(build_path):
    """TASK-10 AC#5: an extending file that names a field not in the base
    fails the load naming the file and the field."""
    config_path = FIXTURES_DIR / "layered_unknown_field" / "config.yaml"
    result = run_fixture(build_path, "layered_unknown_field")
    assert result.returncode != 0
    assert not result.output_exists
    assert f"config '{config_path}'" in result.stderr
    assert "field 'discount' is not in the base config" in result.stderr


def test_unknown_key_in_extending_file_fails_naming_file(build_path):
    """TASK-10: extending files are decoded strictly too: a typo fails the
    load naming the file and the key."""
    config_path = FIXTURES_DIR / "layered_unknown_key" / "config.yaml"
    proc = run_validate_config(build_path, "layered_unknown_key")
    assert proc.returncode != 0
    assert f"config '{config_path}'" in proc.stderr
    assert "field header_pattern not found" in proc.stderr


@pytest.mark.parametrize(
    "fixture_name, expected",
    [
        (
            "layered_is_null_not_nullable",
            "field 'id', pattern '^-$': is_null representation on a field that is not nullable",
        ),
        (
            "layered_required_not_nullable",
            "field 'id': required: false is only allowed on a nullable field",
        ),
    ],
)
def test_checks_run_on_the_merged_config(build_path, fixture_name, expected):
    """TASK-10: every config check runs on the merged config, so an override
    that is fine on its own but breaks the base field's rules fails the load."""
    config_path = FIXTURES_DIR / fixture_name / "config.yaml"
    proc = run_validate_config(build_path, fixture_name)
    assert proc.returncode != 0
    assert f"merged config for '{config_path}'" in proc.stderr
    assert expected in proc.stderr


def test_extends_cycle_fails_naming_files(build_path):
    """TASK-10 AC#7: a cycle in extends fails the load, naming the files in
    the cycle."""
    fixture_dir = FIXTURES_DIR / "layered_cycle"
    proc = run_validate_config(build_path, "layered_cycle")
    assert proc.returncode != 0
    assert (
        f"extends cycle: {fixture_dir / 'config.yaml'} -> {fixture_dir / 'sub' / 'middle.yaml'}"
        f" -> {fixture_dir / 'config.yaml'}"
    ) in proc.stderr

    result = run_fixture(build_path, "layered_cycle")
    assert result.returncode != 0
    assert "extends cycle" in result.stderr


def test_missing_extends_fails_naming_files(build_path):
    """TASK-10 AC#7: an extends path that does not exist fails the load,
    naming the extending file and the missing one."""
    fixture_dir = FIXTURES_DIR / "layered_missing"
    proc = run_validate_config(build_path, "layered_missing")
    assert proc.returncode != 0
    assert (
        f"config '{fixture_dir / 'config.yaml'}' extends "
        f"'{fixture_dir / 'sub' / 'does_not_exist.yaml'}', which can't be read"
    ) in proc.stderr


def test_print_merged_round_trips(build_path, tmp_path):
    """TASK-10 AC#8: validate_config --print-merged validates the merged
    config and prints it as YAML; stdout carries only the YAML (it parses as a
    root config with no extends), and the printed config is itself a valid
    root config that prints back unchanged."""
    proc = print_merged(build_path, FIXTURES_DIR / "layered_chain" / "config.yaml")
    assert proc.returncode == 0, proc.stderr
    assert "config file is valid" in proc.stderr
    printed = yaml.safe_load(proc.stdout)
    assert "extends" not in printed
    assert printed["name"] == "chain"
    assert [f["name"] for f in printed["fields"]] == ["id", "day"]

    root_path = tmp_path / "merged.yaml"
    root_path.write_text(proc.stdout)
    again = print_merged(build_path, root_path)
    assert again.returncode == 0, again.stderr
    assert again.stdout == proc.stdout

    # the plain check still passes on the printed file
    check = subprocess.run(
        [str(build_path), "validate_config", "-c", str(root_path)],
        capture_output=True,
        text=True,
        timeout=60,
    )
    assert check.returncode == 0, check.stderr


def test_print_merged_on_invalid_config_prints_nothing(build_path):
    """TASK-10 AC#8: an invalid merged config fails validation and prints no
    YAML to stdout."""
    proc = print_merged(build_path, FIXTURES_DIR / "layered_is_null_not_nullable" / "config.yaml")
    assert proc.returncode != 0
    assert proc.stdout == ""
    assert "config file is not valid" in proc.stderr


def test_print_merged_on_root_config(build_path):
    """TASK-10 AC#8: --print-merged also works on a config without extends."""
    proc = print_merged(build_path, FIXTURES_DIR / "date_iso" / "config.yaml")
    assert proc.returncode == 0, proc.stderr
    assert yaml.safe_load(proc.stdout) == yaml.safe_load(
        (FIXTURES_DIR / "date_iso" / "config.yaml").read_text()
    )


def test_readme_documents_layered_configs():
    """TASK-10 AC#9: the README documents extends, the input-side-only rule
    and the list merge rules, and no longer lists layered configs as planned."""
    readme = (FIXTURES_DIR.parent.parent / "README.md").read_text()
    section = readme.split("### Layered configs", 1)[1].split("\n## ", 1)[0]
    assert "`extends` names the base config, resolved relative to the file that contains it" in section
    assert "may only change input-side settings" in section
    for setting in [
        "`representations` | field | replaces the base list",
        "`prepend_representations` | field | added before the base list",
        "`append_representations` | field | added after the base list",
        "`header_patterns` | field | replaces the base list",
    ]:
        assert setting in section
    assert "-c base.yaml -c partner.yaml" not in readme
    assert "--print-merged" in readme
