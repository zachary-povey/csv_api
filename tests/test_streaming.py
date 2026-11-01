"""Tests for streaming output with --stream (TASK-14)."""

import io
import os
import subprocess
import threading

import fastavro

from tests.utils import FIXTURES_DIR, read_avro_file

SIMPLE = FIXTURES_DIR / "simple_integers"
DATA_FAILURES = FIXTURES_DIR / "data_failures"
INCOMPLETE = "the output written to stdout is incomplete"


def run_parse(build_path, config_path, data_path, extra_args=()):
    """Run parse and return the process, with stdout kept as bytes."""
    cmd = [
        str(build_path),
        "parse",
        "--config-path",
        str(config_path),
        "--input-path",
        str(data_path),
        *extra_args,
    ]
    return subprocess.run(cmd, capture_output=True, timeout=60)


def read_avro_bytes(data: bytes) -> list[dict]:
    """Read every record from Avro OCF bytes; fails on anything that isn't Avro."""
    return list(fastavro.reader(io.BytesIO(data)))


def write_rows(path, rows, header="id,age"):
    path.write_text(header + "\n" + "".join(f"{row}\n" for row in rows))


def test_stream_writes_avro_to_stdout(build_path):
    """TASK-14 AC#1: with --stream and no --output-path, the output is written
    to stdout."""
    proc = run_parse(build_path, SIMPLE / "config.yaml", SIMPLE / "data.csv", ["--stream"])
    assert proc.returncode == 0, proc.stderr.decode()
    records = read_avro_bytes(proc.stdout)
    assert records[:2] == [{"id": 1, "age": 25}, {"id": 2, "age": 30}]
    assert len(records) == len((SIMPLE / "data.csv").read_text().splitlines()) - 1


def test_stream_writes_header_when_no_rows(build_path, tmp_path):
    """TASK-14 AC#1: the Avro header is written even when no rows are output,
    so stdout is always readable Avro."""
    data_path = tmp_path / "data.csv"
    write_rows(data_path, [])
    proc = run_parse(build_path, SIMPLE / "config.yaml", data_path, ["--stream"])
    assert proc.returncode == 0, proc.stderr.decode()
    reader = fastavro.reader(io.BytesIO(proc.stdout))
    assert list(reader) == []
    assert reader.writer_schema["name"] == "simple_integers"


def test_stream_emits_rows_before_the_input_is_finished(build_path, tmp_path):
    """TASK-14 AC#1: rows are written to stdout as they are processed. The
    input is a pipe that is held open: records must be readable from stdout
    before the rest of the input (ending in an invalid row) is written.
    Also TASK-21 AC#2: batching rows into blocks must not hold a row back."""
    fifo = tmp_path / "data.csv"
    os.mkfifo(fifo)
    proc = subprocess.Popen(
        [
            str(build_path), "parse", "--stream",
            "--config-path", str(SIMPLE / "config.yaml"),
            "--input-path", str(fifo),
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    try:
        with open(fifo, "w") as data:
            data.write("id,age\n" + "".join(f"{i},{i}\n" for i in range(1, 101)))
            data.flush()

            # read the first record while the input is still open
            first = []
            reader_thread = threading.Thread(
                target=lambda: first.append(next(fastavro.reader(proc.stdout))),
                daemon=True,
            )
            reader_thread.start()
            reader_thread.join(timeout=20)
            assert first == [{"id": 1, "age": 1}], "no record was streamed before the input ended"

            data.write("x,1\n")
        _, stderr = proc.communicate(timeout=20)
    finally:
        proc.kill()
    assert proc.returncode != 0
    assert INCOMPLETE in stderr.decode()


def test_stream_failure_keeps_rows_written_before_the_error(build_path, tmp_path):
    """TASK-14 AC#1 and AC#2: with a long valid prefix and a late error, the
    rows validated before the error are on stdout (readable Avro), the run
    exits non-zero and stderr says the output is incomplete."""
    rows = 5000
    data_path = tmp_path / "data.csv"
    write_rows(data_path, [f"{i},{i}" for i in range(1, rows + 1)] + ["x,1"])
    proc = run_parse(build_path, SIMPLE / "config.yaml", data_path, ["--stream"])
    assert proc.returncode != 0
    stderr = proc.stderr.decode()
    assert "row 5002, column 'id'" in stderr
    assert INCOMPLETE in stderr
    records = read_avro_bytes(proc.stdout)
    assert 0 < len(records) <= rows
    assert records == [{"id": i, "age": i} for i in range(1, len(records) + 1)]


def test_stream_invalid_file_fails_with_incomplete_output_message(build_path):
    """TASK-14 AC#2: with --stream and an invalid file, parse exits non-zero
    and stderr says the output written is incomplete."""
    proc = run_parse(
        build_path, DATA_FAILURES / "config.yaml", DATA_FAILURES / "data.csv", ["--stream"]
    )
    assert proc.returncode != 0
    stderr = proc.stderr.decode()
    assert "row 2, column 'id'" in stderr
    assert INCOMPLETE in stderr


def test_stream_header_failure_fails_with_incomplete_output_message(build_path):
    """TASK-14 AC#2: a failure before any row is written (a bad header) is
    also reported as incomplete output."""
    proc = run_parse(
        build_path,
        FIXTURES_DIR / "missing_required_field" / "config.yaml",
        FIXTURES_DIR / "missing_required_field" / "data.csv",
        ["--stream"],
    )
    assert proc.returncode != 0
    assert INCOMPLETE in proc.stderr.decode()


def test_no_stream_failure_leaves_no_output(build_path, tmp_path):
    """TASK-14 AC#3: without --stream, a failed run leaves no output behind
    and nothing on stdout."""
    output_path = tmp_path / "output.avro"
    proc = run_parse(
        build_path,
        DATA_FAILURES / "config.yaml",
        DATA_FAILURES / "data.csv",
        ["--output-path", str(output_path)],
    )
    assert proc.returncode != 0
    assert proc.stdout == b""
    assert "incomplete" not in proc.stderr.decode()
    assert list(tmp_path.iterdir()) == [], "no output or temporary file is left"


def test_output_path_required_without_stream(build_path, tmp_path):
    """TASK-14 AC#3: without --stream, --output-path is still required; output
    is never streamed to stdout."""
    proc = run_parse(build_path, SIMPLE / "config.yaml", SIMPLE / "data.csv")
    assert proc.returncode != 0
    assert proc.stdout == b""
    assert "--output-path is required" in proc.stderr.decode()


def test_stream_stdout_carries_only_output_data(build_path):
    """TASK-14 AC#4: with --stream, warnings (dropped rows) go to stderr and
    stdout carries only the Avro output."""
    proc = run_parse(
        build_path,
        DATA_FAILURES / "config.yaml",
        DATA_FAILURES / "data.csv",
        ["--stream", "--error-on-data-failures=false"],
    )
    assert proc.returncode == 0, proc.stderr.decode()
    assert "dropped 2 row(s)" in proc.stderr.decode()
    assert b"dropped" not in proc.stdout
    # the whole of stdout is one Avro file: fastavro fails on trailing data
    assert len(read_avro_bytes(proc.stdout)) == 38


def test_stream_errors_go_to_stderr_only(build_path):
    """TASK-14 AC#4: with --stream, errors go to stderr only; stdout is
    nothing but Avro."""
    proc = run_parse(
        build_path,
        DATA_FAILURES / "config.yaml",
        DATA_FAILURES / "data.csv",
        ["--stream", "--fail-fast=false"],
    )
    assert proc.returncode != 0
    assert b"row 2" not in proc.stdout
    assert b"incomplete" not in proc.stdout
    assert "row 31, column 'status'" in proc.stderr.decode()
    read_avro_bytes(proc.stdout)


def test_stream_with_output_path_writes_the_path_directly(build_path, tmp_path):
    """TASK-14: with --stream and --output-path, the file is written in place
    as rows are processed: it is left (incomplete) on failure, and nothing
    goes to stdout."""
    output_path = tmp_path / "output.avro"
    data_path = tmp_path / "data.csv"
    write_rows(data_path, ["1,1", "2,2", "x,3"])
    proc = run_parse(
        build_path,
        SIMPLE / "config.yaml",
        data_path,
        ["--stream", "--output-path", str(output_path)],
    )
    assert proc.returncode != 0
    assert proc.stdout == b""
    assert f"the output written to '{output_path}' is incomplete" in proc.stderr.decode()
    assert sorted(p.name for p in tmp_path.iterdir()) == ["data.csv", "output.avro"]
    assert read_avro_file(output_path) == [{"id": 1, "age": 1}, {"id": 2, "age": 2}]

    write_rows(data_path, ["1,1", "2,2"])
    proc = run_parse(
        build_path,
        SIMPLE / "config.yaml",
        data_path,
        ["--stream", "--output-path", str(output_path)],
    )
    assert proc.returncode == 0, proc.stderr.decode()
    assert read_avro_file(output_path) == [{"id": 1, "age": 1}, {"id": 2, "age": 2}]


def test_stream_side_reports_stay_atomic(build_path, tmp_path):
    """TASK-14: with --stream, the data failure report and json error report
    are still only moved into place when complete (failed run: no data failure
    report, json report written whole)."""
    data_path = tmp_path / "data.csv"
    write_rows(data_path, ["1,1", "y,2", "3,3", "4"])  # a malformed last row fails the run
    report_path = tmp_path / "rejected.csv"
    json_path = tmp_path / "errors.json"
    proc = run_parse(
        build_path,
        SIMPLE / "config.yaml",
        data_path,
        [
            "--stream", "--error-on-data-failures=false",
            "--data-failure-report", str(report_path),
            "--error-report", "json", "--error-report-path", str(json_path),
        ],
    )
    assert proc.returncode != 0
    assert INCOMPLETE in proc.stderr.decode()
    assert not report_path.exists()
    assert sorted(p.name for p in tmp_path.iterdir()) == ["data.csv", "errors.json"]
    assert '"valid": false' in json_path.read_text()


def test_readme_documents_stream():
    """TASK-14 AC#5: the README documents --stream and the incomplete-output
    trade-off, and streaming is no longer listed as planned."""
    readme = (FIXTURES_DIR.parent.parent / "README.md").read_text()
    section = readme.split("#### Streaming output", 1)[1].split("\n### ", 1)[0]
    assert "--stream" in section
    assert "incomplete" in section
    assert "exit code" in section
    # the Planned section is gone once nothing is left in it (TASK-13)
    planned = readme.split("## Planned", 1)[1] if "## Planned" in readme else ""
    assert "stream" not in planned.lower()


def test_usage_error_unknown_flag_writes_nothing_to_stdout(build_path):
    """TASK-18 AC#1: an unknown flag fails without writing to stdout."""
    result = subprocess.run(
        [str(build_path), "parse", "--stream", "--bogus"], capture_output=True, timeout=60
    )
    assert result.returncode != 0
    assert result.stdout == b""
    assert b"bogus" in result.stderr


def test_usage_error_missing_required_flag_writes_nothing_to_stdout(build_path):
    """TASK-18 AC#2: a missing required flag fails without writing to stdout."""
    config = FIXTURES_DIR / "simple_strings" / "config.yaml"
    result = subprocess.run(
        [str(build_path), "parse", "--stream", "-c", str(config)], capture_output=True, timeout=60
    )
    assert result.returncode != 0
    assert result.stdout == b""
    assert b"input-path" in result.stderr
