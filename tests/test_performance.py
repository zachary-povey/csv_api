"""Behaviour on a large file (TASK-16).

Throughput and memory are measured with scripts/benchmark.py, not here; see
README "Performance". These tests check that a large file with an early error
stops promptly instead of being read to the end.
"""

import time

import pytest

from scripts.benchmark import write_config, write_csv
from tests.utils import run_csv_api

ROWS = 1_000_000
# a full run over ROWS valid rows takes ~15-20s on a 4-cpu machine (see
# README "Performance"); a run stopped at an early error should take a
# fraction of a second, so this bound is generous yet well below a full run
PROMPT_STOP_SECONDS = 3.0
VALID_ROW = "1,first,1.00,0.5,open,2024-01-01,2024-01-01T00:00:00Z,true,\n"
INVALID_ROW = "x,bad,1.00,0.5,open,2024-01-01,2024-01-01T00:00:00Z,true,\n"


@pytest.fixture(scope="module")
def large_files(tmp_path_factory):
    """A config and a ROWS-row csv whose rows 3, 5 and 7 have an invalid id."""
    tmpdir = tmp_path_factory.mktemp("large")
    config, data = tmpdir / "config.yaml", tmpdir / "data.csv"
    write_config(config)
    write_csv(data, ROWS, first_rows=(VALID_ROW + INVALID_ROW) * 3)
    return config, data


def timed_run(build_path, config, data, output, extra_args=()):
    start = time.perf_counter()
    proc = run_csv_api(build_path, config, data, output, extra_args)
    return proc, time.perf_counter() - start


def test_early_error_stops_promptly_with_default_flags(build_path, large_files, tmp_path):
    """TASK-16 AC#2: a large file with an early error stops promptly with default flags."""
    config, data = large_files
    output = tmp_path / "output.avro"
    proc, elapsed = timed_run(build_path, config, data, output)
    assert proc.returncode != 0, "Command should have failed"
    assert "row 3, column 'id'" in proc.stderr
    assert "fail-fast" in proc.stderr
    assert elapsed < PROMPT_STOP_SECONDS, f"took {elapsed:.1f}s to stop"
    assert list(tmp_path.iterdir()) == [], "A failed run should leave no output"


def test_early_errors_stop_promptly_at_max_errors(build_path, large_files, tmp_path):
    """TASK-16 AC#2: with --fail-fast=false --max-errors N, a large file stops
    promptly once N early errors are collected."""
    config, data = large_files
    output = tmp_path / "output.avro"
    proc, elapsed = timed_run(
        build_path, config, data, output, ["--fail-fast=false", "--max-errors", "3"]
    )
    assert proc.returncode != 0, "Command should have failed"
    for row in (3, 5, 7):
        assert f"row {row}, column 'id'" in proc.stderr
    assert "--max-errors (3) was reached" in proc.stderr
    assert elapsed < PROMPT_STOP_SECONDS, f"took {elapsed:.1f}s to stop"
    assert list(tmp_path.iterdir()) == [], "A failed run should leave no output"
