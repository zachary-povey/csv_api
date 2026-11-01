import datetime
from decimal import Decimal

import pytest

from tests.utils import FIXTURES_DIR, run_fixture, run_validate_config


def test_unknown_logical_type_key_fails_parse(build_path):
    """TASK-3 AC#1: an unknown key under logical_type fails config load, naming the key."""
    result = run_fixture(build_path, "config_unknown_logical_type_key")
    assert result.returncode != 0, "Command should have failed"
    assert not result.output_exists
    assert "max_length" in result.stderr
    assert "field 'label'" in result.stderr


def test_unknown_logical_type_arg_fails_parse(build_path):
    """TASK-3 AC#1: an unknown key under logical_type.args fails config load, naming the key."""
    result = run_fixture(build_path, "config_unknown_type_arg")
    assert result.returncode != 0, "Command should have failed"
    assert not result.output_exists
    assert "'logical_type.args' has invalid keys: case_insensitive" in result.stderr
    assert "field 'status'" in result.stderr


def test_invalid_logical_type_args_fail_parse(build_path):
    """TASK-3 AC#2: validation errors on the narrowed type config are not ignored."""
    result = run_fixture(build_path, "config_invalid_type_args")
    assert result.returncode != 0, "Command should have failed"
    assert not result.output_exists
    assert "field 'amount'" in result.stderr
    assert "Scale" in result.stderr


@pytest.mark.parametrize(
    "fixture_name, expected",
    [
        ("config_unknown_logical_type_key", "max_length"),
        ("config_unknown_type_arg", "case_insensitive"),
        ("config_invalid_type_args", "Scale"),
    ],
)
def test_validate_config_reports_logical_type_errors(build_path, fixture_name, expected):
    """TASK-3 AC#3: validate_config reports unknown keys and type config validation errors."""
    result = run_validate_config(build_path, fixture_name)
    assert result.returncode != 0, "validate_config should have failed"
    assert expected in result.stderr
    assert "config file is not valid" in result.stderr


def test_unknown_key_outside_logical_type_fails(build_path):
    """Unknown keys at the top, field and representation levels are rejected
    by the strict YAML decode (regression guard alongside TASK-3)."""
    result = run_validate_config(build_path, "config_unknown_field_key")
    assert result.returncode != 0, "validate_config should have failed"
    assert "field is_nul not found" in result.stderr


def test_valid_config_passes_validate_config(build_path):
    """TASK-3 AC#3: a valid config still passes validate_config."""
    result = run_validate_config(build_path, "enum_basic")
    assert result.returncode == 0, result.stderr


# --- TASK-9: representation args are checked against the logical type at load ---

REP_ARG_ERRORS = [
    # (fixture, field, pattern, expected message part)
    (
        "config_rep_args_unknown",
        "d",
        "^(?P<yr>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})$",
        "unknown value args [yr] for date",
    ),
    (
        "config_rep_args_missing",
        "d",
        "^(?P<year>[0-9]{4})-(?P<month>[0-9]{2})$",
        "value args [month, year] (from capture groups and static args) are not a valid set for date",
    ),
    (
        "config_rep_args_conflicting",
        "ts",
        "^epoch$",
        "value args [day, month, value, year]",
    ),
    (
        "config_rep_static_integer_fraction",
        "n",
        "^half$",
        "static arg 'value': must be an integer, got 1.5",
    ),
    (
        "config_rep_static_month_not_numeric",
        "d",
        "^(?P<year>[0-9]{4})-(?P<day>[0-9]{2})$",
        "static arg 'month': must be an integer, got jan",
    ),
    (
        "config_rep_static_enum_not_permitted",
        "status",
        "^live$",
        "'enabled' is not in permitted_values [active, inactive]",
    ),
    (
        "config_rep_static_bad_precision",
        "ts",
        "^(?P<value>[0-9]+)$",
        "'minutes' must be 'seconds', 'milliseconds' or 'microseconds'",
    ),
]


@pytest.mark.parametrize("fixture_name, field, pattern, expected", REP_ARG_ERRORS)
def test_invalid_representation_args_fail_parse(build_path, fixture_name, field, pattern, expected):
    """TASK-9 AC#1: a representation whose capture groups plus static args are not a
    valid arg set for its type (unknown, missing or conflicting args, or an unusable
    static arg value) fails config load, naming the field and pattern."""
    result = run_fixture(build_path, fixture_name)
    assert result.returncode != 0, "Command should have failed"
    assert not result.output_exists
    assert f"field '{field}', pattern '{pattern}'" in result.stderr
    assert expected in result.stderr
    # the config fails at load, before any row is read
    assert "row " not in result.stderr


def test_every_documented_arg_set_is_accepted(build_path):
    """TASK-9 AC#2: each type accepts all of its documented arg sets (decimal value vs
    integer_part+decimal_part, timestamp epoch vs components, time with no args, a
    fixed date pattern with static args, is_null representations skipped)."""
    result = run_validate_config(build_path, "rep_args_all_sets")
    assert result.returncode == 0, result.stderr

    result = run_fixture(build_path, "rep_args_all_sets")
    assert result.returncode == 0, result.stderr
    first, second = result.records
    assert first["d"] == datetime.date(2024, 3, 15)
    assert second["d"] == datetime.date(2020, 1, 1)
    assert first["dec_v"] == Decimal("1.50")
    assert first["dec_p"] == Decimal("2.05")
    assert first["t"] == datetime.time(12, 30)
    assert second["t"] == datetime.time(0, 0)
    assert first["ts_epoch"] == datetime.datetime(2023, 11, 14, 22, 13, 20, tzinfo=datetime.timezone.utc)
    assert second["ts_epoch"] == datetime.datetime(1970, 1, 1, tzinfo=datetime.timezone.utc)
    assert first["ts_comp"] == datetime.datetime(2024, 3, 15, 9, 0, tzinfo=datetime.timezone.utc)
    assert (first["i"], second["i"]) == (42, 0)
    assert (first["e"], second["e"]) == ("green", "red")


@pytest.mark.parametrize("fixture_name, field, pattern, expected", REP_ARG_ERRORS)
def test_validate_config_reports_representation_arg_errors(build_path, fixture_name, field, pattern, expected):
    """TASK-9 AC#3: validate_config reports representation arg errors."""
    result = run_validate_config(build_path, fixture_name)
    assert result.returncode != 0, "validate_config should have failed"
    assert f"field '{field}', pattern '{pattern}'" in result.stderr
    assert expected in result.stderr
    assert "config file is not valid" in result.stderr


# fixtures whose config is meant to be invalid
INVALID_CONFIG_FIXTURES = {
    "config_invalid_type_args",
    "config_unknown_field_key",
    "config_unknown_logical_type_key",
    "config_unknown_type_arg",
    "decimal_conflicting_args",
    *(name for name, *_ in REP_ARG_ERRORS),
}


@pytest.mark.parametrize(
    "fixture_name",
    sorted(
        p.parent.name
        for p in FIXTURES_DIR.glob("*/config.yaml")
        if p.parent.name not in INVALID_CONFIG_FIXTURES
    ),
)
def test_existing_fixtures_still_load(build_path, fixture_name):
    """TASK-9 AC#4: every existing (valid) test fixture still loads with the arg check."""
    result = run_validate_config(build_path, fixture_name)
    assert result.returncode == 0, result.stderr
