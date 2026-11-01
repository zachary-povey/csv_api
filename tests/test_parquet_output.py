"""Tests for Parquet output with --output-format parquet (TASK-13)."""

import datetime
import json
import subprocess
from decimal import Decimal
from pathlib import Path

import pyarrow.parquet as pq
import pytest

from tests.utils import FIXTURES_DIR, read_avro_file, run_csv_api

ALL_TYPES = FIXTURES_DIR / "parquet_all_types"
DATA_FAILURE = FIXTURES_DIR / "parquet_data_failure"
UTC = datetime.timezone.utc


def run_format(build_path, fixture_dir, output_path, output_format, extra_args=()):
    return run_csv_api(
        build_path,
        fixture_dir / "config.yaml",
        fixture_dir / "data.csv",
        output_path,
        ["--output-format", output_format, *extra_args],
    )


@pytest.fixture
def parquet_output(build_path, tmp_path) -> Path:
    output_path = tmp_path / "output.parquet"
    proc = run_format(build_path, ALL_TYPES, output_path, "parquet")
    assert proc.returncode == 0, proc.stderr
    return output_path


def logical_type(column) -> dict:
    """The Parquet logical type of a column, as a dict ({} if it has none)."""
    if column.logical_type.type == "NONE":
        return {}
    return json.loads(column.logical_type.to_json())


def test_parquet_schema_maps_logical_types(parquet_output):
    """TASK-13 AC#1: every logical type is written with the Parquet physical
    and logical type equivalent to the Avro output, in config order, and
    nullable fields are OPTIONAL while the rest are REQUIRED."""
    schema = pq.ParquetFile(parquet_output).schema
    time_type = {"Type": "Time", "isAdjustedToUTC": False, "timeUnit": "microseconds"}
    timestamp_type = {"Type": "Timestamp", "isAdjustedToUTC": True, "timeUnit": "microseconds"}
    expected = {
        "name": ("BYTE_ARRAY", {"Type": "String"}),
        "count": ("INT64", {"Type": "Int", "bitWidth": 64, "isSigned": True}),
        "amount": ("BYTE_ARRAY", {"Type": "Decimal", "precision": 10, "scale": 2}),
        "ratio": ("DOUBLE", {}),
        "status": ("BYTE_ARRAY", {"Type": "Enum"}),
        "day": ("INT32", {"Type": "Date"}),
        "at_time": ("INT64", time_type),
        "ts": ("INT64", timestamp_type),
        "flag": ("BOOLEAN", {}),
    }
    names = [schema.column(i).name for i in range(len(schema))]
    assert names == list(expected) + [f"opt_{name}" for name in expected]
    for i in range(len(schema)):
        column = schema.column(i)
        physical, logical = expected[column.name.removeprefix("opt_")]
        assert column.physical_type == physical, column.name
        actual = logical_type(column)
        # pyarrow adds reader-side flags to timestamps; compare only the spec keys
        assert {k: actual.get(k) for k in logical} == logical, column.name
        # REQUIRED columns have no definition level, OPTIONAL ones have one
        assert column.max_definition_level == (1 if column.name.startswith("opt_") else 0), column.name
    assert schema.to_arrow_schema().field("ts").type.tz == "UTC"


def test_parquet_values_for_every_logical_type(parquet_output):
    """TASK-13 AC#3: the Parquet output is read back and every logical type
    has the expected values, including nulls in nullable fields, exact
    decimals (12345678.90 at precision 10 scale 2, leading zeros, negatives),
    times to the microsecond and timestamps converted to UTC."""
    rows = pq.read_table(parquet_output).to_pylist()
    for row in rows:
        # pyarrow reads Parquet ENUM columns as binary
        for name in ("status", "opt_status"):
            if row[name] is not None:
                row[name] = row[name].decode()
    assert rows == [
        {
            "name": "Alice",
            "count": 9007199254740993,  # beyond float64's exact range
            "amount": Decimal("12345678.90"),
            "ratio": 1.5,
            "status": "active",
            "day": datetime.date(2024, 2, 29),
            "at_time": datetime.time(13, 45, 30, 123456),
            "ts": datetime.datetime(2024, 3, 15, 7, 30, tzinfo=UTC),
            "flag": True,
            "opt_name": "Al",
            "opt_count": 7,
            "opt_amount": Decimal("99999999.99"),
            "opt_ratio": 0.25,
            "opt_status": "inactive",
            "opt_day": datetime.date(1969, 12, 31),
            "opt_at_time": datetime.time(0, 0, 0, 1),
            "opt_ts": datetime.datetime(1970, 1, 1, tzinfo=UTC),
            "opt_flag": False,
        },
        {
            "name": "Bob",
            "count": -42,
            "amount": Decimal("0.05"),
            "ratio": -2.75,
            "status": "inactive",
            "day": datetime.date(1970, 1, 1),
            "at_time": datetime.time(23, 59, 59, 999999),
            "ts": datetime.datetime(1970, 1, 1, 4, 59, 59, tzinfo=UTC),
            "flag": False,
            "opt_name": None,
            "opt_count": None,
            "opt_amount": None,
            "opt_ratio": None,
            "opt_status": None,
            "opt_day": None,
            "opt_at_time": None,
            "opt_ts": None,
            "opt_flag": None,
        },
        {
            "name": "Carol",
            "count": 0,
            "amount": Decimal("-1.50"),
            "ratio": 0.1,
            "status": "active",
            "day": datetime.date(2000, 1, 1),
            "at_time": datetime.time(0, 0),
            "ts": datetime.datetime(2038, 1, 19, 3, 14, 8, tzinfo=UTC),
            "flag": True,
            "opt_name": "",  # an empty string, not null
            "opt_count": 0,
            "opt_amount": Decimal("-0.01"),
            "opt_ratio": -0.5,
            "opt_status": "active",
            "opt_day": datetime.date(2024, 1, 1),
            "opt_at_time": datetime.time(12, 0, 0, 500000),
            "opt_ts": datetime.datetime(2024, 7, 1, 1, 0, tzinfo=UTC),
            "opt_flag": True,
        },
    ]
    # decimals keep their scale exactly (no float rounding)
    assert [str(row["amount"]) for row in rows] == ["12345678.90", "0.05", "-1.50"]


def test_parquet_and_avro_hold_the_same_values(build_path, tmp_path, parquet_output):
    """TASK-13 AC#1, AC#3: the same file converted to Avro and to Parquet
    gives the same values for every logical type."""
    avro_path = tmp_path / "output.avro"
    proc = run_format(build_path, ALL_TYPES, avro_path, "avro")
    assert proc.returncode == 0, proc.stderr
    parquet_rows = pq.read_table(parquet_output).to_pylist()
    for row in parquet_rows:
        for name in ("status", "opt_status"):
            if row[name] is not None:
                row[name] = row[name].decode()
    assert parquet_rows == read_avro_file(avro_path)


def test_avro_is_the_default_format(build_path, tmp_path):
    """TASK-13 AC#2: without --output-format the output is Avro, as before."""
    output_path = tmp_path / "output"
    proc = run_csv_api(build_path, ALL_TYPES / "config.yaml", ALL_TYPES / "data.csv", output_path)
    assert proc.returncode == 0, proc.stderr
    assert output_path.read_bytes()[:4] == b"Obj\x01"
    assert len(read_avro_file(output_path)) == 3


def test_parquet_with_no_rows_is_valid(build_path, tmp_path):
    """TASK-13 AC#1: a file with a header and no rows gives a valid Parquet
    file with the schema and no rows."""
    data_path = tmp_path / "data.csv"
    data_path.write_text((ALL_TYPES / "data.csv").read_text().splitlines()[0] + "\n")
    output_path = tmp_path / "output.parquet"
    proc = run_csv_api(
        build_path, ALL_TYPES / "config.yaml", data_path, output_path, ["--output-format", "parquet"]
    )
    assert proc.returncode == 0, proc.stderr
    table = pq.read_table(output_path)
    assert table.num_rows == 0
    assert table.column_names[:2] == ["name", "count"]


def test_parquet_drops_invalid_rows_when_asked(build_path, tmp_path):
    """TASK-13 AC#1: with --error-on-data-failures=false, the valid rows are
    written to Parquet and the invalid one is dropped."""
    output_path = tmp_path / "output.parquet"
    proc = run_format(
        build_path, DATA_FAILURE, output_path, "parquet", ["--error-on-data-failures=false"]
    )
    assert proc.returncode == 0, proc.stderr
    assert "dropped 1 row(s)" in proc.stderr
    assert [row["name"] for row in pq.read_table(output_path).to_pylist()] == ["Alice", "Bob"]


def test_failed_parquet_run_leaves_no_output(build_path, tmp_path):
    """TASK-13 AC#1: a failed Parquet run leaves no output behind (design
    principle 4), not even a temporary file."""
    output_path = tmp_path / "output.parquet"
    proc = run_format(build_path, DATA_FAILURE, output_path, "parquet")
    assert proc.returncode != 0
    assert "not-a-number" in proc.stderr
    assert list(tmp_path.iterdir()) == []


def test_stream_is_rejected_with_parquet(build_path, tmp_path):
    """TASK-13 AC#2: --stream can't be used with Parquet, whose footer is only
    written at the end; the run is rejected before anything is written."""
    for args in (["--stream"], ["--stream", "-o", str(tmp_path / "output.parquet")]):
        cmd = [
            str(build_path), "parse",
            "-c", str(ALL_TYPES / "config.yaml"),
            "-i", str(ALL_TYPES / "data.csv"),
            "--output-format", "parquet",
            *args,
        ]
        proc = subprocess.run(cmd, capture_output=True, timeout=60)
        assert proc.returncode != 0
        assert b"--stream can't be used with --output-format=parquet" in proc.stderr
        assert proc.stdout == b""
        assert list(tmp_path.iterdir()) == []


def test_unknown_output_format_is_rejected(build_path, tmp_path):
    """TASK-13 AC#2: an unsupported --output-format is rejected, naming the
    supported formats."""
    output_path = tmp_path / "output.csv"
    proc = run_format(build_path, ALL_TYPES, output_path, "csv")
    assert proc.returncode != 0
    assert "unsupported --output-format 'csv', supported values are 'avro' and 'parquet'" in proc.stderr
    assert not output_path.exists()


def test_output_format_is_documented():
    """TASK-13 AC#2: the --output-format flag and the Parquet type mapping are
    documented in the README."""
    readme = (Path(__file__).parent.parent / "README.md").read_text()
    assert "--output-format" in readme
    assert "| Type | Type args | Avro output | Parquet output |" in readme
