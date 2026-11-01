import decimal

from tests.utils import run_fixture, run_validate_config

DROP_DATA_FAILURES = ["--error-on-data-failures=false"]


def test_header_patterns_map_columns_to_fields(build_path):
    """TASK-5 AC#1: a column whose header matches any of a field's
    header_patterns is mapped to the field (patterns are not anchored, and any
    one of several patterns may match). TASK-5 AC#2: a field without
    header_patterns is still matched by an exact header match on its name
    ('id', not 'ID'). A non-required field whose patterns match no header is
    null."""
    result = run_fixture(build_path, "header_patterns_match")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert result.records == [
        {"id": 1, "first_name": "Ann", "amount": decimal.Decimal("1.50"), "notes": None},
        {"id": 2, "first_name": "Bob", "amount": decimal.Decimal("2.00"), "notes": None},
    ]
    # the output schema uses the field names, not the headers
    assert [f["name"] for f in result.writer_schema["fields"]] == [
        "id",
        "first_name",
        "amount",
        "notes",
    ]


def test_header_patterns_replace_the_name_match(build_path):
    """TASK-5 AC#1/AC#2: with header_patterns set, only the patterns are used
    (decision-6): a column named exactly like the field does not match, and the
    missing-field error names the patterns that were tried."""
    result = run_fixture(build_path, "header_patterns_replace_name")
    assert result.returncode != 0, "Command should have failed"
    assert not result.output_exists
    assert "file-level errors" in result.stderr
    assert (
        "missing some required fields: [first_name (no header matches "
        "header_patterns '^First Name$', '^Given Name$')]"
    ) in result.stderr


def test_header_matching_more_than_one_field_is_file_error(build_path):
    """TASK-5 AC#3: a header matching more than one field is a file-level
    error naming the header and the fields. Every header problem is reported,
    even with fail-fast on (the default)."""
    result = run_fixture(build_path, "header_patterns_header_ambiguous")
    assert result.returncode != 0, "Command should have failed"
    assert not result.output_exists
    assert "file-level errors" in result.stderr
    assert (
        "header 'Last Name' (column 2) matches more than one field: 'first_name', 'last_name'"
        in result.stderr
    )
    assert "missing some required fields: [email]" in result.stderr


def test_field_matching_more_than_one_header_is_file_error(build_path):
    """TASK-5 AC#3: a field whose header_patterns match more than one header is
    a file-level error naming the field and the headers."""
    result = run_fixture(build_path, "header_patterns_field_ambiguous")
    assert result.returncode != 0, "Command should have failed"
    assert not result.output_exists
    assert "file-level errors" in result.stderr
    assert (
        "field 'amount' matches more than one header: "
        "'Amount (GBP)' (column 2), 'Amount (EUR)' (column 3)"
    ) in result.stderr


def test_duplicate_header_is_file_error(build_path):
    """TASK-5 AC#3: without header_patterns, a header name that appears twice
    makes the field match more than one header, a file-level error (it used to
    silently read the last column)."""
    result = run_fixture(build_path, "duplicate_header")
    assert result.returncode != 0, "Command should have failed"
    assert not result.output_exists
    assert "file-level errors" in result.stderr
    assert (
        "field 'id' matches more than one header: 'id' (column 1), 'id' (column 3)"
        in result.stderr
    )


def test_data_failures_name_the_header_and_field(build_path):
    """TASK-5 AC#1: cell errors on a column mapped by header_patterns name the
    column by its header text, plus the field name (decision-7)."""
    result = run_fixture(build_path, "header_patterns_data_failure", ["--fail-fast=false"])
    assert result.returncode != 0, "Command should have failed"
    assert not result.output_exists
    assert "row 3, column 'id': value 'x' did not match any pattern" in result.stderr
    assert (
        "row 3, column 'Amount (GBP)' (field 'amount'): value 'abc' did not match any pattern"
        in result.stderr
    )


def test_data_failure_report_keeps_original_headers(build_path):
    """TASK-5 AC#1: the data failure report repeats the file's own header row,
    and its error column names the mapped column like the console does."""
    result = run_fixture(
        build_path, "header_patterns_data_failure", DROP_DATA_FAILURES, data_failure_report=True
    )
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert result.records == [
        {"id": 1, "amount": decimal.Decimal("1.50")},
        {"id": 3, "amount": decimal.Decimal("3.00")},
    ]
    header, *rows = result.data_failure_rows
    assert header == ["id", "Amount (GBP)", "__csv_api_error__"]
    assert [row[:2] for row in rows] == [["x", "abc"]]
    assert "column 'Amount (GBP)' (field 'amount'): value 'abc'" in rows[0][2]


def test_invalid_header_pattern_fails_config_load(build_path):
    """TASK-5 AC#1: header_patterns are compiled when the config loads; an
    invalid one fails parse and validate_config, naming the field and pattern."""
    for result in (
        run_fixture(build_path, "config_header_pattern_invalid"),
        run_validate_config(build_path, "config_header_pattern_invalid"),
    ):
        assert result.returncode != 0, "Command should have failed"
        assert "field 'amount', header pattern '^Amount (GBP$'" in result.stderr
