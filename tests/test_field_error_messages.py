"""TASK-11: per-field error messages (error_message in the config).

TASK-11 AC#4: these tests, with the README 'error_message' docs, cover the
feature; every test below names the criterion it proves."""
import subprocess
from decimal import Decimal

from tests.utils import run_fixture, run_validate_config

AMOUNT_NOTE = (
    "(note: amount must be pounds with 2 decimal places, e.g. 12.50"
    " - contact the finance team)"
)
STATUS_NOTE = "(note: status must be active or inactive)"


def test_field_error_message_is_accepted_in_config(build_path):
    """TASK-11 AC#1: a field can have an optional error_message in the config
    (fields with and without one load, and valid rows convert as usual)."""
    result = run_validate_config(build_path, "field_error_message")
    assert result.returncode == 0, f"Config should be valid: {result.stderr}"

    result = run_fixture(
        build_path, "field_error_message", ["--error-on-data-failures=false"]
    )
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert result.records == [
        {"id": 1, "amount": Decimal("12.50"), "status": "active"},
        {"id": 4, "amount": Decimal("7.25"), "status": "inactive"},
    ]


def test_field_error_message_in_console_report(build_path):
    """TASK-11 AC#2: the field's message follows the technical reason of its
    data failures in the console report (no match, and failed conversion).
    TASK-11 AC#3: a field without a message reports as before."""
    result = run_fixture(build_path, "field_error_message", ["--fail-fast=false"])
    assert result.returncode != 0, "Command should have failed"
    assert (
        f"row 3, column 'amount': value '12.5' did not match any pattern {AMOUNT_NOTE}"
        in result.stderr
    )
    assert "row 4, column 'Status' (field 'status'): failed to convert 'gone' to enum" in result.stderr
    assert f"(resolved args: map[value:gone]) {STATUS_NOTE}" in result.stderr
    # id has no error_message
    assert "row 3, column 'id': value 'x' did not match any pattern\n" in result.stderr


def test_field_error_message_in_data_failure_report(build_path):
    """TASK-11 AC#2: the field's message is included with its failures in the
    data failure report's __csv_api_error__ column.
    TASK-11 AC#3: a field without a message reports as before."""
    result = run_fixture(
        build_path,
        "field_error_message",
        ["--error-on-data-failures=false"],
        data_failure_report=True,
    )
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    header, *rows = result.data_failure_rows
    assert header == ["id", "amount", "Status", "__csv_api_error__"]
    assert rows[0][3] == (
        "column 'id': value 'x' did not match any pattern; "
        f"column 'amount': value '12.5' did not match any pattern {AMOUNT_NOTE}"
    )
    assert rows[1][3].startswith("column 'Status' (field 'status'): failed to convert 'gone'")
    assert rows[1][3].endswith(STATUS_NOTE)


def test_field_error_message_on_missing_required_column(build_path):
    """TASK-11 AC#2: a required field's message is also shown when its column
    is missing from the file. TASK-11 AC#3: fields without a message are
    listed as before."""
    result = run_fixture(build_path, "missing_required_with_error_message")
    assert result.returncode != 0, "Command should have failed"
    assert (
        "missing some required fields: [age (note: the age column must be included"
        " - contact the HR team), team]" in result.stderr
    )


def test_field_error_message_must_be_a_string(build_path, tmp_path):
    """TASK-11 AC#1: error_message is a plain string; anything else fails the
    config load."""
    config = tmp_path / "config.yaml"
    config.write_text(
        "name: bad_error_message\n"
        "fields:\n"
        "  - name: id\n"
        "    error_message: [not, a, string]\n"
        "    logical_type:\n"
        "      name: integer\n"
        "    representations:\n"
        "      - pattern: \"^([0-9]+)$\"\n"
    )
    result = subprocess.run(
        [str(build_path), "validate_config", "--config-path", str(config)],
        capture_output=True,
        text=True,
        timeout=60,
    )
    assert result.returncode != 0, "Config should be invalid"
    assert "cannot unmarshal !!seq into string" in result.stderr
