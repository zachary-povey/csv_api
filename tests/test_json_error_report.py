"""Tests for the json error report (TASK-12; these tests and the README
"JSON error report" section are TASK-12 AC#4)."""

import json

from tests.utils import FIXTURES_DIR, run_csv_api

COLLECT_ERRORS = ["--fail-fast=false"]
KEYS = {"group", "row", "column", "field", "value", "reason", "message"}


def run_with_json_report(build_path, tmp_path, fixture_name, extra_args=(), data_path=None):
    """Run a fixture with --error-report json and return (process, report).

    The report is None when no report file was written.
    """
    fixture_dir = FIXTURES_DIR / fixture_name
    report_path = tmp_path / "errors.json"
    proc = run_csv_api(
        build_path,
        fixture_dir / "config.yaml",
        data_path or fixture_dir / "data.csv",
        tmp_path / "output.avro",
        ["--error-report", "json", "--error-report-path", str(report_path), *extra_args],
    )
    report = json.loads(report_path.read_text()) if report_path.exists() else None
    return proc, report


def test_json_report_written_to_file(build_path, tmp_path):
    """TASK-12 AC#1: --error-report json writes the errors to the given file,
    and the console report still goes to stderr."""
    proc, report = run_with_json_report(build_path, tmp_path, "data_failures")
    assert proc.returncode != 0, "Command should have failed"
    assert report is not None, "The json report should have been written"
    assert report["valid"] is False
    assert len(report["errors"]) == 1
    assert "row 2, column 'id'" in proc.stderr
    assert proc.stdout == ""
    assert not (tmp_path / "output.avro").exists()
    # written atomically: only the report itself is left beside the output
    assert sorted(p.name for p in tmp_path.iterdir()) == ["errors.json"]


def test_json_report_data_errors(build_path, tmp_path):
    """TASK-12 AC#2: data failures record group, row, column, field, raw value,
    reason and the field's error_message."""
    proc, report = run_with_json_report(
        build_path, tmp_path, "field_error_message", COLLECT_ERRORS
    )
    assert proc.returncode != 0
    errors = report["errors"]
    assert all(set(e) == KEYS for e in errors)
    assert [(e["group"], e["row"], e["column"], e["field"], e["value"]) for e in errors] == [
        ("data", 3, "id", "id", "x"),
        ("data", 3, "amount", "amount", "12.5"),
        ("data", 4, "Status", "status", "gone"),
    ]
    assert errors[0]["reason"] == "value 'x' did not match any pattern"
    assert errors[0]["message"] is None
    assert errors[1]["message"] == (
        "amount must be pounds with 2 decimal places, e.g. 12.50 - contact the finance team"
    )
    assert "gone" in errors[2]["reason"]
    assert errors[2]["message"] == "status must be active or inactive"


def test_json_report_row_errors(build_path, tmp_path):
    """TASK-12 AC#2: format errors in rows record group "row", the row number
    and the reason, alongside data failures."""
    proc, report = run_with_json_report(
        build_path, tmp_path, "malformed_rows", COLLECT_ERRORS
    )
    assert proc.returncode != 0
    rows = [e for e in report["errors"] if e["group"] == "row"]
    assert [(e["row"], e["reason"]) for e in rows] == [
        (3, "row 3 has 2 values but the header has 3"),
        (5, "row 5 has 4 values but the header has 3"),
    ]
    assert all(e["column"] is None and e["value"] is None for e in rows)
    data = [e for e in report["errors"] if e["group"] == "data"]
    assert [(e["row"], e["column"], e["value"]) for e in data] == [(4, "id", "x")]


def test_json_report_file_errors(build_path, tmp_path):
    """TASK-12 AC#2/AC#3: header problems are file errors; no rows were checked,
    which the report records as an early stop."""
    proc, report = run_with_json_report(build_path, tmp_path, "missing_required_field")
    assert proc.returncode != 0
    assert report["valid"] is False
    [error] = report["errors"]
    assert error["group"] == "file"
    assert "missing some required fields: [age]" in error["reason"]
    assert error["row"] is None
    assert report["stopped_early"] is True
    assert "header" in report["stop_reason"]


def test_json_report_execution_errors(build_path, tmp_path):
    """TASK-12 AC#2/AC#3: an execution error (input can't be opened) is
    reported in the "execution" group and stops the run."""
    proc, report = run_with_json_report(
        build_path, tmp_path, "data_failures", data_path=tmp_path / "missing.csv"
    )
    assert proc.returncode != 0
    assert "error opening data file" in proc.stderr
    [error] = report["errors"]
    assert error["group"] == "execution"
    assert "error opening data file" in error["reason"]
    assert report["stopped_early"] is True
    assert report["stop_reason"] == "an execution error stopped the run"


def test_json_report_stops_early_on_fail_fast(build_path, tmp_path):
    """TASK-12 AC#3: with fail-fast (the default) the report says processing
    stopped early and why."""
    proc, report = run_with_json_report(build_path, tmp_path, "data_failures")
    assert proc.returncode != 0
    assert report["stopped_early"] is True
    assert report["stop_reason"] == "fail-fast is on"
    assert [e["row"] for e in report["errors"]] == [2]


def test_json_report_stops_early_on_max_errors(build_path, tmp_path):
    """TASK-12 AC#3: reaching --max-errors is recorded as an early stop."""
    proc, report = run_with_json_report(
        build_path, tmp_path, "data_failures", COLLECT_ERRORS + ["--max-errors", "2"]
    )
    assert proc.returncode != 0
    assert report["stopped_early"] is True
    assert report["stop_reason"] == "--max-errors (2) was reached"
    assert len(report["errors"]) == 2


def test_json_report_stops_early_on_broken_quote(build_path, tmp_path):
    """TASK-12 AC#2/AC#3: a broken quote is a row error with its row number
    and stops processing."""
    proc, report = run_with_json_report(
        build_path, tmp_path, "broken_quote", COLLECT_ERRORS
    )
    assert proc.returncode != 0
    [error] = report["errors"]
    assert error["group"] == "row"
    assert error["row"] == 3
    assert "quoted-field" in error["reason"]
    assert report["stopped_early"] is True
    assert report["stop_reason"] == "the file could not be parsed past this point"


def test_json_report_not_stopped_when_all_errors_collected(build_path, tmp_path):
    """TASK-12 AC#3: a run that checked the whole file is not an early stop."""
    proc, report = run_with_json_report(
        build_path, tmp_path, "data_failures", COLLECT_ERRORS
    )
    assert proc.returncode != 0
    assert report["stopped_early"] is False
    assert report["stop_reason"] is None
    assert len(report["errors"]) == 3


def test_json_report_on_success(build_path, tmp_path):
    """TASK-12 AC#1: a successful run still writes a report, with no errors."""
    proc, report = run_with_json_report(build_path, tmp_path, "simple_integers")
    assert proc.returncode == 0, proc.stderr
    assert report == {
        "valid": True,
        "stopped_early": False,
        "stop_reason": None,
        "dropped_rows": 0,
        "errors": [],
    }
    assert (tmp_path / "output.avro").exists()


def test_json_report_counts_dropped_rows(build_path, tmp_path):
    """TASK-12 AC#1: rows dropped with --error-on-data-failures=false are
    counted in the report of a successful run."""
    proc, report = run_with_json_report(
        build_path, tmp_path, "data_failures", ["--error-on-data-failures=false"]
    )
    assert proc.returncode == 0, proc.stderr
    assert report["valid"] is True
    assert report["dropped_rows"] == 2
    assert report["errors"] == []


def test_json_report_requires_path(build_path, tmp_path):
    """TASK-12 AC#1: --error-report json without --error-report-path is rejected."""
    fixture_dir = FIXTURES_DIR / "simple_integers"
    proc = run_csv_api(
        build_path,
        fixture_dir / "config.yaml",
        fixture_dir / "data.csv",
        tmp_path / "output.avro",
        ["--error-report", "json"],
    )
    assert proc.returncode != 0
    assert "--error-report=json requires --error-report-path" in proc.stderr
    assert not (tmp_path / "output.avro").exists()


def test_json_report_path_requires_json(build_path, tmp_path):
    """TASK-12 AC#1: --error-report-path is rejected unless --error-report=json."""
    fixture_dir = FIXTURES_DIR / "simple_integers"
    proc = run_csv_api(
        build_path,
        fixture_dir / "config.yaml",
        fixture_dir / "data.csv",
        tmp_path / "output.avro",
        ["--error-report-path", str(tmp_path / "errors.json")],
    )
    assert proc.returncode != 0
    assert "--error-report-path can only be set with --error-report=json" in proc.stderr
    assert not (tmp_path / "errors.json").exists()
