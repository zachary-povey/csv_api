import pytest

from tests.utils import run_fixture, run_validate_config


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
