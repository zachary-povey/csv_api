"""Tests for streaming Parquet output with --stream --output-format parquet (TASK-24)."""

import io
import os
import subprocess
import threading
from pathlib import Path

import pyarrow as pa
import pyarrow.parquet as pq
import pytest

from tests.utils import FIXTURES_DIR, run_csv_api

ALL_TYPES = FIXTURES_DIR / "parquet_all_types"
DATA_FAILURE = FIXTURES_DIR / "parquet_data_failure"
SIMPLE = FIXTURES_DIR / "simple_integers"
PARQUET = ["--output-format", "parquet"]
INCOMPLETE = "the output written to stdout is incomplete"
# rows per Parquet row group (rowsPerRowGroup in internal/parquet_writer)
ROW_GROUP = 100_000


def run_parse(build_path, fixture_dir, extra_args=(), data_path=None):
    """Run parse with --output-format parquet, stdout kept as bytes."""
    cmd = [
        str(build_path), "parse",
        "-c", str(fixture_dir / "config.yaml"),
        "-i", str(data_path or fixture_dir / "data.csv"),
        *PARQUET,
        *extra_args,
    ]
    return subprocess.run(cmd, capture_output=True, timeout=60)


def read_parquet_bytes(data: bytes) -> pa.Table:
    """Read a whole Parquet file from bytes; fails on anything that isn't one
    complete Parquet file (the footer is read from the very end)."""
    return pq.read_table(io.BytesIO(data))


def assert_not_parquet(data: bytes):
    """The bytes have no Parquet footer and readers reject them."""
    assert not data.endswith(b"PAR1"), "a footer was written"
    with pytest.raises(pa.ArrowInvalid):
        read_parquet_bytes(data)


def write_rows(path: Path, rows: int, bad_row: str | None = None):
    with open(path, "w") as data:
        data.write("id,age\n")
        data.writelines(f"{i},{i % 100}\n" for i in range(1, rows + 1))
        if bad_row:
            data.write(bad_row + "\n")


def test_stream_parquet_to_stdout_matches_non_streamed_run(build_path, tmp_path):
    """TASK-24 AC#1: with --stream --output-format parquet and no
    --output-path, a complete, readable Parquet file is written to stdout,
    with the same schema and values as a non-streamed run."""
    proc = run_parse(build_path, ALL_TYPES, ["--stream"])
    assert proc.returncode == 0, proc.stderr.decode()
    streamed = read_parquet_bytes(proc.stdout)

    output_path = tmp_path / "output.parquet"
    file_proc = run_csv_api(
        build_path, ALL_TYPES / "config.yaml", ALL_TYPES / "data.csv", output_path, PARQUET
    )
    assert file_proc.returncode == 0, file_proc.stderr
    expected = pq.read_table(output_path)
    assert streamed.num_rows > 0
    assert streamed.schema == expected.schema
    assert streamed.to_pylist() == expected.to_pylist()


def test_stream_parquet_multiple_row_groups_to_stdout(build_path, tmp_path):
    """TASK-24 AC#1: an input larger than one row group streams to stdout as
    one readable file holding every row, in order."""
    data_path = tmp_path / "data.csv"
    write_rows(data_path, ROW_GROUP + 5)
    proc = run_parse(build_path, SIMPLE, ["--stream"], data_path=data_path)
    assert proc.returncode == 0, proc.stderr.decode()
    parquet_file = pq.ParquetFile(io.BytesIO(proc.stdout))
    assert parquet_file.metadata.num_row_groups == 2
    assert parquet_file.read().column("id").to_pylist() == list(range(1, ROW_GROUP + 6))


def test_stream_parquet_invalid_file_writes_no_footer(build_path):
    """TASK-24 AC#2: with an invalid file, parse exits non-zero, stderr says
    the output is incomplete, and no footer is written: what reached stdout
    (nothing here, the run failed inside the first row group) isn't Parquet."""
    proc = run_parse(build_path, DATA_FAILURE, ["--stream"])
    assert proc.returncode != 0
    stderr = proc.stderr.decode()
    assert "not-a-number" in stderr
    assert INCOMPLETE in stderr
    assert_not_parquet(proc.stdout)


def test_stream_parquet_failure_after_a_row_group_writes_no_footer(build_path, tmp_path):
    """TASK-24 AC#2: when the run fails after a full row group has been
    written, stdout holds that row group but no footer, so Parquet readers
    reject it; the run exits non-zero and says the output is incomplete."""
    data_path = tmp_path / "data.csv"
    write_rows(data_path, ROW_GROUP + 10, bad_row="x,1")
    proc = run_parse(build_path, SIMPLE, ["--stream"], data_path=data_path)
    assert proc.returncode != 0
    stderr = proc.stderr.decode()
    assert f"row {ROW_GROUP + 12}, column 'id'" in stderr
    assert INCOMPLETE in stderr
    assert proc.stdout.startswith(b"PAR1") and len(proc.stdout) > 100_000
    assert_not_parquet(proc.stdout)


def test_stream_parquet_with_output_path_writes_the_path_directly(build_path, tmp_path):
    """TASK-24 AC#3: with --stream and --output-path, the Parquet file is
    written in place (no temporary file, as with Avro streaming, decision-21):
    left without a footer on failure, complete on success; nothing goes to
    stdout."""
    output_path = tmp_path / "out" / "output.parquet"
    output_path.parent.mkdir()
    proc = run_parse(build_path, DATA_FAILURE, ["--stream", "-o", str(output_path)])
    assert proc.returncode != 0
    assert proc.stdout == b""
    assert f"the output written to '{output_path}' is incomplete" in proc.stderr.decode()
    assert [p.name for p in output_path.parent.iterdir()] == ["output.parquet"]
    assert_not_parquet(output_path.read_bytes())

    proc = run_parse(build_path, ALL_TYPES, ["--stream", "-o", str(output_path)])
    assert proc.returncode == 0, proc.stderr.decode()
    assert proc.stdout == b""
    assert [p.name for p in output_path.parent.iterdir()] == ["output.parquet"]
    assert pq.read_table(output_path).num_rows > 0


def test_stream_parquet_writes_row_groups_before_the_input_is_finished(build_path, tmp_path):
    """TASK-24 AC#4: with an input larger than one row group, the completed
    row group is written to stdout before the input has finished. The input
    is a pipe held open after one row group's worth of rows; output must
    arrive before the rest of the input (ending in an invalid row) is written.
    (internal/parquet_writer TestRowGroupsReachOutputBeforeInputEnds checks
    the whole row group arrives.)"""
    fifo = tmp_path / "data.csv"
    os.mkfifo(fifo)
    proc = subprocess.Popen(
        [
            str(build_path), "parse", "--stream", *PARQUET,
            "--config-path", str(SIMPLE / "config.yaml"),
            "--input-path", str(fifo),
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    try:
        with open(fifo, "w") as data:
            data.write("id,age\n")
            data.writelines(f"{i},{i % 100}\n" for i in range(1, ROW_GROUP + 2))
            data.flush()

            # Parquet output is buffered until a row group is complete, so
            # any bytes at all mean the first row group was written
            first = []
            reader_thread = threading.Thread(
                target=lambda: first.append(proc.stdout.read1(65536)), daemon=True
            )
            reader_thread.start()
            reader_thread.join(timeout=30)
            assert first and first[0].startswith(b"PAR1"), "no row group was written before the input ended"

            data.write("x,1\n")
        rest, stderr = proc.communicate(timeout=30)
    finally:
        proc.kill()
    assert proc.returncode != 0
    assert INCOMPLETE in stderr.decode()
    assert_not_parquet(first[0] + rest)


def test_stream_parquet_stdout_carries_only_parquet(build_path):
    """TASK-24 AC#5: with --stream --output-format parquet, warnings (dropped
    rows) go to stderr and stdout is nothing but one Parquet file (pyarrow
    reads the footer from the very end, so trailing text would fail it)."""
    proc = run_parse(build_path, DATA_FAILURE, ["--stream", "--error-on-data-failures=false"])
    assert proc.returncode == 0, proc.stderr.decode()
    assert "dropped 1 row(s)" in proc.stderr.decode()
    assert b"dropped" not in proc.stdout
    assert proc.stdout.startswith(b"PAR1")
    assert [row["name"] for row in read_parquet_bytes(proc.stdout).to_pylist()] == ["Alice", "Bob"]


def test_stream_parquet_errors_go_to_stderr_only(build_path):
    """TASK-24 AC#5: errors and the incomplete-output message go to stderr
    only, never to stdout."""
    proc = run_parse(build_path, DATA_FAILURE, ["--stream", "--fail-fast=false"])
    assert proc.returncode != 0
    assert b"not-a-number" not in proc.stdout
    assert b"incomplete" not in proc.stdout
    assert "not-a-number" in proc.stderr.decode()


def test_readme_documents_parquet_streaming():
    """TASK-24 AC#6: the README's streaming and output format sections
    document Parquet streaming, including that a Parquet file is only
    readable once complete, and no longer say it is rejected."""
    readme = (FIXTURES_DIR.parent.parent / "README.md").read_text()
    streaming = readme.split("#### Streaming output", 1)[1].split("\n### ", 1)[0]
    assert "--stream --output-format parquet" in streaming
    assert "only readable once it is complete" in streaming
    assert "rejected before the run starts" not in streaming
    output_format = readme.split("#### Output format", 1)[1].split("#### Streaming output", 1)[0]
    assert "--stream" in output_format
