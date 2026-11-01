"""Avro output is written in multi-row blocks (TASK-21).

The output size and throughput for the 1M-row benchmark file are measured
with scripts/benchmark.py (see README "Performance"); these tests check the
block layout and the records on a smaller generated file.
"""

import io
import os
import subprocess
import threading

import fastavro

from scripts.benchmark import BLOCK_ROWS, write_config, write_csv
from tests.utils import FIXTURES_DIR, run_csv_api

ROWS = 5000
# the most rows the writer puts in one block (maxBlockRows in avro_writer.go)
MAX_BLOCK_ROWS = 100
SIMPLE = FIXTURES_DIR / "simple_integers"


def block_sizes(data: bytes) -> list[int]:
    return [block.num_records for block in fastavro.block_reader(io.BytesIO(data))]


def records(data: bytes) -> list[dict]:
    return list(fastavro.reader(io.BytesIO(data)))


def test_rows_are_written_in_multi_row_blocks(build_path, tmp_path):
    """TASK-21 AC#1: without --stream, rows are written in full blocks of
    MAX_BLOCK_ROWS rather than one block per row, with identical records
    (the benchmark config, so nullable values are still wrapped per row)."""
    config, data, output = tmp_path / "config.yaml", tmp_path / "data.csv", tmp_path / "out.avro"
    write_config(config)
    write_csv(data, ROWS)
    proc = run_csv_api(build_path, config, data, output)
    assert proc.returncode == 0, proc.stderr

    avro = output.read_bytes()
    assert block_sizes(avro) == [MAX_BLOCK_ROWS] * (ROWS // MAX_BLOCK_ROWS)

    rows = records(avro)
    assert len(rows) == ROWS
    # write_csv repeats the same BLOCK_ROWS rows, numbered from 0
    assert [r["id"] for r in rows] == [i % BLOCK_ROWS for i in range(ROWS)]
    assert [r["note"] for r in rows] == [
        None if i % 4 == 0 else f"note {i}" for i in (i % BLOCK_ROWS for i in range(ROWS))
    ]
    assert rows[1]["name"] == "customer 1" and rows[1]["status"] == "closed"

    # the same records as when streamed (which may use smaller blocks)
    streamed = subprocess.run(
        [str(build_path), "parse", "--stream", "-c", str(config), "-i", str(data)],
        capture_output=True,
        timeout=60,
    )
    assert streamed.returncode == 0, streamed.stderr.decode()
    assert records(streamed.stdout) == rows


def test_stream_writes_every_validated_row_without_waiting_for_input(build_path, tmp_path):
    """TASK-21 AC#2: with --stream, every row validated so far reaches the
    reader while the input is held open, including rows past the first
    block's worth: a block is written whenever the writer would wait for
    input, never held back until it is full."""
    rows = MAX_BLOCK_ROWS * 2 + 50
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
            data.write("id,age\n" + "".join(f"{i},{i}\n" for i in range(1, rows + 1)))
            data.flush()

            received = []

            def read_rows():
                reader = fastavro.reader(proc.stdout)
                for _ in range(rows):
                    received.append(next(reader))

            reader_thread = threading.Thread(target=read_rows, daemon=True)
            reader_thread.start()
            reader_thread.join(timeout=20)
            assert received == [{"id": i, "age": i} for i in range(1, rows + 1)], (
                "not every validated row was streamed before the input ended"
            )
        _, stderr = proc.communicate(timeout=20)
    finally:
        proc.kill()
    assert proc.returncode == 0, stderr.decode()
