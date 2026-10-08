from tests.utils import run_fixture

DATA_ROWS = 40
DROP_DATA_FAILURES = ["--error-on-data-failures=false"]
COLLECT_ERRORS = ["--fail-fast=false"]


def test_fails_fast_on_first_data_failure_by_default(build_path):
    result = run_fixture(build_path, "data_failures")
    assert result.returncode != 0, "Command should have failed"
    assert "row 2, column 'id'" in result.stderr
    assert "row 31" not in result.stderr, "Only the first error should be reported"
    assert "fail-fast" in result.stderr
    assert not result.output_exists, "A failed run should leave no output"


def test_collects_all_data_failures_without_fail_fast(build_path):
    result = run_fixture(build_path, "data_failures", COLLECT_ERRORS)
    assert result.returncode != 0, "Command should have failed"
    assert "row 2, column 'id': value 'x'" in result.stderr
    assert "row 31, column 'id': value 'y'" in result.stderr
    assert "row 31, column 'status': failed to convert 'gone'" in result.stderr
    assert "stopped" not in result.stderr.lower()
    assert not result.output_exists, "A failed run should leave no output"


def test_max_errors_limits_collected_errors(build_path):
    result = run_fixture(
        build_path, "data_failures", COLLECT_ERRORS + ["--max-errors", "2"]
    )
    assert result.returncode != 0, "Command should have failed"
    assert "row 2, column 'id'" in result.stderr
    assert "row 31, column 'id'" in result.stderr
    assert "row 31, column 'status'" not in result.stderr
    assert "--max-errors (2)" in result.stderr


def test_drops_rows_with_data_failures(build_path):
    result = run_fixture(build_path, "data_failures", DROP_DATA_FAILURES)
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == DATA_ROWS - 2
    assert all(r["id"] not in ("x", "y") for r in result.records)
    assert "dropped 2 row(s)" in result.stderr


def test_data_failure_report_contains_dropped_rows(build_path):
    result = run_fixture(
        build_path, "data_failures", DROP_DATA_FAILURES, data_failure_report=True
    )
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == DATA_ROWS - 2

    header, *rows = result.data_failure_rows
    assert header == ["id", "status", "name", "__csv_api_error__"]
    assert [row[:3] for row in rows] == [
        ["x", "active", "first"],
        ["y", "gone", "second"],
    ]
    assert "column 'id': value 'x' did not match any pattern" in rows[0][3]
    # every failed cell in the row is reported
    assert "column 'id'" in rows[1][3]
    assert "column 'status'" in rows[1][3]


def test_malformed_rows_fail_even_when_dropping_data_failures(build_path):
    result = run_fixture(
        build_path, "malformed_rows", DROP_DATA_FAILURES, data_failure_report=True
    )
    assert result.returncode != 0, "Command should have failed"
    assert "row 3 has 2 values but the header has 3" in result.stderr
    assert not result.output_exists, "A failed run should leave no output"
    assert result.data_failure_rows == [], "A failed run should leave no report"


def test_collects_all_malformed_rows_without_fail_fast(build_path):
    result = run_fixture(build_path, "malformed_rows", COLLECT_ERRORS)
    assert result.returncode != 0, "Command should have failed"
    assert "row 3 has 2 values" in result.stderr
    assert "row 5 has 4 values" in result.stderr
    # data failures are still reported alongside format errors
    assert "row 4, column 'id': value 'x'" in result.stderr


def test_broken_quote_stops_processing(build_path):
    result = run_fixture(build_path, "broken_quote", COLLECT_ERRORS)
    assert result.returncode != 0, "Command should have failed"
    assert "quoted-field" in result.stderr
    assert "could not be parsed past this point" in result.stderr


def test_max_errors_requires_fail_fast_off(build_path):
    result = run_fixture(build_path, "data_failures", ["--max-errors", "2"])
    assert result.returncode != 0, "Command should have failed"
    assert "--max-errors can only be set with --fail-fast=false" in result.stderr


def test_data_failure_report_requires_dropping_data_failures(build_path):
    result = run_fixture(build_path, "data_failures", data_failure_report=True)
    assert result.returncode != 0, "Command should have failed"
    assert (
        "--data-failure-report can only be set with --error-on-data-failures=false"
        in result.stderr
    )


def test_unsupported_error_report(build_path):
    result = run_fixture(build_path, "data_failures", ["--error-report", "json"])
    assert result.returncode != 0, "Command should have failed"
    assert "unsupported --error-report 'json'" in result.stderr
