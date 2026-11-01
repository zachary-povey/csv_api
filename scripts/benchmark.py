#!/usr/bin/env python3
"""Benchmark csv-api on a large generated csv file.

Generates a csv (1,000,000 rows by default) with a realistic mix of types
into a temporary directory, runs the built binary on it for each output
format and reports wall time, rows/sec and peak memory (max RSS).

Usage (build first with ./scripts/build.sh):

    python3 scripts/benchmark.py [--rows N] [--formats avro,parquet] [--build-path PATH]

Stdlib only; tests/test_performance.py reuses the generator.
"""

import argparse
import os
import platform
import subprocess
import sys
import tempfile
import time
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent

CONFIG = r"""name: com.example.benchmark
fields:
  - name: id
    logical_type:
      name: integer
    representations:
      - pattern: "^[0-9]+$"
  - name: name
    logical_type:
      name: string
    representations:
      - pattern: "^(.+)$"
  - name: amount
    logical_type:
      name: decimal
      args:
        precision: 12
        scale: 2
    representations:
      - pattern: "^(-?[0-9]+\\.[0-9]{2})$"
  - name: ratio
    logical_type:
      name: decimal
      args:
        as_float: true
    representations:
      - pattern: "^([0-9]+\\.[0-9]+)$"
  - name: status
    logical_type:
      name: enum
      args:
        permitted_values: [open, closed, pending]
    representations:
      - pattern: "^(open|closed|pending)$"
  - name: created_on
    logical_type:
      name: date
    representations:
      - pattern: "^(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})$"
  - name: updated_at
    logical_type:
      name: timestamp
    representations:
      - pattern: "^(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})T(?P<hour>[0-9]{2}):(?P<minute>[0-9]{2}):(?P<second>[0-9]{2})Z$"
  - name: active
    logical_type:
      name: boolean
    representations:
      - pattern: "^(true|false)$"
  - name: note
    nullable: true
    logical_type:
      name: string
    representations:
      - pattern: "^$"
        is_null: true
      - pattern: "^(.+)$"
"""

HEADER = "id,name,amount,ratio,status,created_on,updated_at,active,note\n"
BLOCK_ROWS = 1000
STATUSES = ("open", "closed", "pending")


def _row(i: int) -> str:
    note = "" if i % 4 == 0 else f"note {i}"
    return (
        f"{i},customer {i},{(i * 37) % 100000 - 500}.{i % 100:02d},"
        f"0.{i:04d},{STATUSES[i % 3]},20{i % 25:02d}-{i % 12 + 1:02d}-{i % 28 + 1:02d},"
        f"2024-{i % 12 + 1:02d}-{i % 28 + 1:02d}T{i % 24:02d}:{i % 60:02d}:{(i * 7) % 60:02d}Z,"
        f"{'true' if i % 2 else 'false'},{note}\n"
    )


def write_config(path: Path) -> None:
    path.write_text(CONFIG)


def write_csv(path: Path, rows: int, first_rows: str = "") -> None:
    """Write a header and `rows` valid data rows to path, after `first_rows`
    (raw csv lines, e.g. an invalid row). A block of rows is built once and
    repeated, so generation is fast."""
    block = "".join(_row(i) for i in range(BLOCK_ROWS))
    with open(path, "w") as f:
        f.write(HEADER)
        f.write(first_rows)
        for _ in range(rows // BLOCK_ROWS):
            f.write(block)
        f.write("".join(_row(i) for i in range(rows % BLOCK_ROWS)))


def run(cmd: list[str]) -> tuple[int, float, int]:
    """Run cmd, returning its exit code, wall time (s) and peak RSS (KiB)."""
    start = time.perf_counter()
    proc = subprocess.Popen(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
    # wait4 gives the child's own resource usage; ru_maxrss is in KiB on Linux
    _, status, usage = os.wait4(proc.pid, 0)
    elapsed = time.perf_counter() - start
    proc.returncode = os.waitstatus_to_exitcode(status)
    if proc.returncode != 0:
        sys.stderr.write(proc.stderr.read().decode())
    proc.stderr.close()
    return proc.returncode, elapsed, usage.ru_maxrss


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    parser.add_argument("--rows", type=int, default=1_000_000)
    parser.add_argument("--formats", default="avro,parquet")
    parser.add_argument("--build-path", default=str(REPO_ROOT / "build" / "csv-api"))
    args = parser.parse_args()

    print(f"machine: {platform.machine()}, {os.cpu_count()} cpus, python {platform.python_version()}")
    with tempfile.TemporaryDirectory() as tmp:
        tmpdir = Path(tmp)
        config, data = tmpdir / "config.yaml", tmpdir / "data.csv"
        write_config(config)
        write_csv(data, args.rows)
        print(f"input: {args.rows:,} rows, {data.stat().st_size / 1e6:.0f} MB")
        for fmt in args.formats.split(","):
            output = tmpdir / f"output.{fmt}"
            code, elapsed, rss_kib = run([
                args.build_path, "parse", "-c", str(config), "-i", str(data),
                "-o", str(output), "--output-format", fmt,
            ])
            if code != 0:
                print(f"{fmt}: failed with exit code {code}")
                return 1
            print(
                f"{fmt}: {elapsed:.2f}s, {args.rows / elapsed:,.0f} rows/s, "
                f"peak RSS {rss_kib / 1024:.0f} MiB, output {output.stat().st_size / 1e6:.0f} MB"
            )
    return 0


if __name__ == "__main__":
    sys.exit(main())
