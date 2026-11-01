"""Rows are parsed on several cores (TASK-22).

The number of parser workers is runtime.GOMAXPROCS, so setting the GOMAXPROCS
environment variable runs the binary with one worker or several. Throughput
(TASK-22 AC#1) depends on the machine, so it is measured with
scripts/benchmark.py rather than here; see README "Performance" and the task
notes for the before/after figures.
"""

import csv

import fastavro
import pytest

from scripts.benchmark import _row, write_config
from tests.utils import run_csv_api

ROWS = 3000


def interleaved_rows(path):
    """A csv of ROWS rows where every 7th row has an invalid id and every 11th
    an invalid status and date, so failures fall throughout every batch."""
    lines = []
    for i in range(ROWS):
        values = _row(i).rstrip("\n").split(",")
        if i % 7 == 3:
            values[0] = f"x{i}"
        if i % 11 == 5:
            values[4], values[5] = "unknown", "2024-13-45"
        lines.append(",".join(values) + "\n")
    path.write_text("id,name,amount,ratio,status,created_on,updated_at,active,note\n" + "".join(lines))


def run_with_workers(build_path, tmp_path, monkeypatch, workers, extra_args):
    """Run on the interleaved file with GOMAXPROCS=workers and return the
    exit code, stderr, output records and data failure report rows."""
    run_dir = tmp_path / f"workers_{workers}"
    run_dir.mkdir()
    config, data = run_dir / "config.yaml", run_dir / "data.csv"
    output, report = run_dir / "output.avro", run_dir / "failures.csv"
    write_config(config)
    interleaved_rows(data)
    args = list(extra_args)
    if "--error-on-data-failures=false" in args:
        args += ["--data-failure-report", str(report)]
    monkeypatch.setenv("GOMAXPROCS", str(workers))
    proc = run_csv_api(build_path, config, data, output, args)
    records = []
    if output.exists():
        with open(output, "rb") as f:
            records = list(fastavro.reader(f))
    report_rows = []
    if report.exists():
        with open(report, newline="") as f:
            report_rows = list(csv.reader(f))
    # temp paths in messages differ between runs
    stderr = proc.stderr.replace(str(run_dir), "<dir>")
    return proc.returncode, stderr, records, report_rows


@pytest.mark.parametrize(
    "extra_args",
    [
        ["--error-on-data-failures=false"],
        ["--fail-fast=false"],
        ["--fail-fast=false", "--max-errors", "100"],
        [],
    ],
    ids=["dropped-rows", "all-errors", "max-errors", "fail-fast"],
)
def test_several_workers_match_one_worker(build_path, tmp_path, monkeypatch, extra_args):
    """TASK-22 AC#3: given several workers and a file whose rows have data
    failures interleaved with valid rows, the output order, the errors (order
    and row numbers) and the data failure report are identical to a
    single-worker run. Also TASK-22 AC#2 (input order, error reporting,
    fail-fast, --max-errors and the data failure report behave as before)."""
    one = run_with_workers(build_path, tmp_path, monkeypatch, 1, extra_args)
    several = run_with_workers(build_path, tmp_path, monkeypatch, 4, extra_args)
    assert several == one

    returncode, stderr, records, report_rows = one
    if extra_args == ["--error-on-data-failures=false"]:
        assert returncode == 0, stderr
        dropped = len(report_rows) - 1
        assert dropped > 500 and len(records) == ROWS - dropped
        assert [r["id"] for r in records] == sorted(r["id"] for r in records)
    else:
        assert returncode != 0
        assert "row 5, column 'id'" in stderr
