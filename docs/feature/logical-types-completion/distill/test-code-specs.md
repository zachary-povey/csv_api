# Test Code Specifications: logical-types-completion

All tests are added to `tests/test_logical_types.py`. They follow the existing pattern exactly:
`run_fixture(build_path, "<fixture_name>")` with assertions on `result.returncode`, `result.records`, and error output.

New imports needed at top of file:

```python
import datetime
```

(`decimal` is already imported.)

---

## Implementation Order

Tests should be enabled one at a time in this order. All tests except the first should be marked with `@pytest.mark.skip(reason="not yet implemented")` initially.

1. `test_enum_basic` -- first enum test, proves enum converter + avro writer work
2. `test_enum_case_insensitive` -- static args remapping via representation
3. `test_enum_value_remap` -- non-capturing group + static args pattern
4. `test_enum_multiple_representations` -- multiple representations per field
5. `test_enum_invalid_value` -- enum error path
6. `test_enum_mixed_types` -- enum alongside other types
7. `test_date_iso` -- first date test, proves date converter + avro writer work
8. `test_date_custom_format` -- custom representation for dates
9. `test_date_invalid_components` -- date error path
10. `test_date_mixed_types` -- date alongside other types
11. `test_time_iso` -- first time test, proves time converter + avro writer work
12. `test_time_fractional_seconds` -- microsecond precision
13. `test_time_custom_format` -- custom representation for times
14. `test_time_edge_cases` -- boundary values
15. `test_timestamp_iso_offset` -- first timestamp test, proves converter + avro writer work
16. `test_timestamp_iso_utc` -- Z suffix handling
17. `test_timestamp_custom_format` -- custom representation for timestamps
18. `test_timestamp_epoch` -- epoch-based conversion with precision static arg
19. `test_timestamp_timezone_conversion` -- timezone offset to UTC
20. `test_timestamp_invalid` -- timestamp error path
21. `test_all_new_types_mixed` -- walking skeleton, all types together

---

## Test Functions

### Enum Tests

```python
@pytest.mark.skip(reason="not yet implemented")
def test_enum_basic(build_path):
    result = run_fixture(build_path, "enum_basic")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["status"] == "active"
    assert result.records[1]["status"] == "inactive"
    assert result.records[2]["status"] == "pending"


@pytest.mark.skip(reason="not yet implemented")
def test_enum_case_insensitive(build_path):
    result = run_fixture(build_path, "enum_case_insensitive")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["status"] == "active"
    assert result.records[1]["status"] == "inactive"
    assert result.records[2]["status"] == "active"


@pytest.mark.skip(reason="not yet implemented")
def test_enum_value_remap(build_path):
    result = run_fixture(build_path, "enum_value_remap")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["status"] == "active"
    assert result.records[1]["status"] == "inactive"
    assert result.records[2]["status"] == "active"


@pytest.mark.skip(reason="not yet implemented")
def test_enum_multiple_representations(build_path):
    result = run_fixture(build_path, "enum_multiple_representations")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 6
    assert result.records[0]["priority"] == "low"
    assert result.records[1]["priority"] == "high"
    assert result.records[2]["priority"] == "medium"
    assert result.records[3]["priority"] == "low"
    assert result.records[4]["priority"] == "high"
    assert result.records[5]["priority"] == "medium"


@pytest.mark.skip(reason="not yet implemented")
def test_enum_invalid_value(build_path):
    result = run_fixture(build_path, "enum_invalid_value")
    assert result.returncode != 0, "Command should have failed"
    assert (
        "unknown" in result.stderr.lower()
        or "permitted" in result.stderr.lower()
        or "not a valid" in result.stderr.lower()
        or "invalid" in result.stderr.lower()
    ), f"Error should mention invalid enum value: {result.stderr} {result.stdout}"


@pytest.mark.skip(reason="not yet implemented")
def test_enum_mixed_types(build_path):
    result = run_fixture(build_path, "enum_mixed_types")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    assert result.records[0]["name"] == "Alice"
    assert result.records[0]["age"] == 30
    assert result.records[0]["status"] == "active"
    assert result.records[1]["name"] == "Bob"
    assert result.records[1]["age"] == 25
    assert result.records[1]["status"] == "inactive"
```

### Date Tests

Note on fastavro date behavior: Avro date logical type (`{"type":"int","logicalType":"date"}`) stores days since Unix epoch. fastavro automatically converts these to `datetime.date` objects when reading. goavro writes the int value; fastavro reads and converts.

```python
@pytest.mark.skip(reason="not yet implemented")
def test_date_iso(build_path):
    result = run_fixture(build_path, "date_iso")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["event_date"] == datetime.date(2024, 3, 15)
    assert result.records[1]["event_date"] == datetime.date(2000, 1, 1)
    assert result.records[2]["event_date"] == datetime.date(1999, 12, 31)


@pytest.mark.skip(reason="not yet implemented")
def test_date_custom_format(build_path):
    result = run_fixture(build_path, "date_custom_format")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["event_date"] == datetime.date(2024, 3, 15)
    assert result.records[1]["event_date"] == datetime.date(2000, 1, 1)
    assert result.records[2]["event_date"] == datetime.date(1999, 12, 31)


@pytest.mark.skip(reason="not yet implemented")
def test_date_invalid_components(build_path):
    result = run_fixture(build_path, "date_invalid_components")
    assert result.returncode != 0, "Command should have failed"
    assert (
        "13" in result.stderr
        or "invalid" in result.stderr.lower()
        or "date" in result.stderr.lower()
    ), f"Error should mention invalid date: {result.stderr} {result.stdout}"


@pytest.mark.skip(reason="not yet implemented")
def test_date_mixed_types(build_path):
    result = run_fixture(build_path, "date_mixed_types")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    assert result.records[0]["event_name"] == "Conference"
    assert result.records[0]["event_date"] == datetime.date(2024, 3, 15)
    assert result.records[0]["attendees"] == 250
    assert result.records[1]["event_name"] == "Workshop"
    assert result.records[1]["event_date"] == datetime.date(2024, 6, 20)
    assert result.records[1]["attendees"] == 50
```

### Time Tests

Note on fastavro time-micros behavior: Avro time-micros logical type (`{"type":"long","logicalType":"time-micros"}`) stores microseconds since midnight. fastavro converts these to `datetime.time` objects. If goavro writes raw microsecond longs, fastavro will handle the conversion.

```python
@pytest.mark.skip(reason="not yet implemented")
def test_time_iso(build_path):
    result = run_fixture(build_path, "time_iso")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["event_time"] == datetime.time(9, 30, 0)
    assert result.records[1]["event_time"] == datetime.time(14, 15, 45)
    assert result.records[2]["event_time"] == datetime.time(23, 0, 0)


@pytest.mark.skip(reason="not yet implemented")
def test_time_fractional_seconds(build_path):
    result = run_fixture(build_path, "time_fractional_seconds")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["event_time"] == datetime.time(9, 30, 0, 123456)
    assert result.records[1]["event_time"] == datetime.time(14, 15, 45, 500000)
    assert result.records[2]["event_time"] == datetime.time(0, 0, 0, 1)


@pytest.mark.skip(reason="not yet implemented")
def test_time_custom_format(build_path):
    result = run_fixture(build_path, "time_custom_format")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["event_time"] == datetime.time(9, 30, 0)
    assert result.records[1]["event_time"] == datetime.time(14, 0, 0)
    assert result.records[2]["event_time"] == datetime.time(0, 0, 0)


@pytest.mark.skip(reason="not yet implemented")
def test_time_edge_cases(build_path):
    result = run_fixture(build_path, "time_edge_cases")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["event_time"] == datetime.time(0, 0, 0)
    assert result.records[1]["event_time"] == datetime.time(23, 59, 59)
    assert result.records[2]["event_time"] == datetime.time(12, 0, 0)
```

### Timestamp Tests

Note on fastavro timestamp-micros behavior: Avro timestamp-micros logical type (`{"type":"long","logicalType":"timestamp-micros"}`) stores microseconds since epoch UTC. fastavro converts these to `datetime.datetime` objects in UTC. The datetime objects will be timezone-aware (with `tzinfo=datetime.timezone.utc`) or naive depending on fastavro version. Tests should handle both possibilities.

```python
@pytest.mark.skip(reason="not yet implemented")
def test_timestamp_iso_offset(build_path):
    result = run_fixture(build_path, "timestamp_iso_offset")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    # 2024-03-15T10:30:00+05:00 = 2024-03-15T05:30:00Z
    ts0 = result.records[0]["created_at"]
    assert ts0.replace(tzinfo=None) == datetime.datetime(2024, 3, 15, 5, 30, 0)
    # 2024-01-01T00:00:00-08:00 = 2024-01-01T08:00:00Z
    ts1 = result.records[1]["created_at"]
    assert ts1.replace(tzinfo=None) == datetime.datetime(2024, 1, 1, 8, 0, 0)


@pytest.mark.skip(reason="not yet implemented")
def test_timestamp_iso_utc(build_path):
    result = run_fixture(build_path, "timestamp_iso_utc")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    ts0 = result.records[0]["created_at"]
    assert ts0.replace(tzinfo=None) == datetime.datetime(2024, 3, 15, 10, 30, 0)
    ts1 = result.records[1]["created_at"]
    assert ts1.replace(tzinfo=None) == datetime.datetime(2024, 1, 1, 0, 0, 0)


@pytest.mark.skip(reason="not yet implemented")
def test_timestamp_custom_format(build_path):
    result = run_fixture(build_path, "timestamp_custom_format")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    ts0 = result.records[0]["created_at"]
    assert ts0.replace(tzinfo=None) == datetime.datetime(2024, 3, 15, 10, 30, 0)
    ts1 = result.records[1]["created_at"]
    assert ts1.replace(tzinfo=None) == datetime.datetime(2000, 1, 1, 0, 0, 0)


@pytest.mark.skip(reason="not yet implemented")
def test_timestamp_epoch(build_path):
    result = run_fixture(build_path, "timestamp_epoch")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    # 1710499800 seconds = 2024-03-15T13:30:00Z
    ts0 = result.records[0]["created_at"]
    assert ts0.replace(tzinfo=None) == datetime.datetime(2024, 3, 15, 13, 30, 0)
    # 946684800 seconds = 2000-01-01T00:00:00Z
    ts1 = result.records[1]["created_at"]
    assert ts1.replace(tzinfo=None) == datetime.datetime(2000, 1, 1, 0, 0, 0)


@pytest.mark.skip(reason="not yet implemented")
def test_timestamp_timezone_conversion(build_path):
    result = run_fixture(build_path, "timestamp_timezone_conversion")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    # 2024-03-15T10:00:00+05:00 = 2024-03-15T05:00:00Z
    ts0 = result.records[0]["created_at"]
    assert ts0.replace(tzinfo=None) == datetime.datetime(2024, 3, 15, 5, 0, 0)
    # 2024-07-01T20:00:00-04:00 = 2024-07-02T00:00:00Z
    ts1 = result.records[1]["created_at"]
    assert ts1.replace(tzinfo=None) == datetime.datetime(2024, 7, 2, 0, 0, 0)


@pytest.mark.skip(reason="not yet implemented")
def test_timestamp_invalid(build_path):
    result = run_fixture(build_path, "timestamp_invalid")
    assert result.returncode != 0, "Command should have failed"
    assert (
        "25" in result.stderr
        or "invalid" in result.stderr.lower()
        or "hour" in result.stderr.lower()
        or "timestamp" in result.stderr.lower()
    ), f"Error should mention invalid timestamp: {result.stderr} {result.stdout}"
```

### Walking Skeleton Test

```python
@pytest.mark.skip(reason="not yet implemented")
def test_all_new_types_mixed(build_path):
    result = run_fixture(build_path, "all_new_types_mixed")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2

    alice = result.records[0]
    assert alice["employee_name"] == "Alice"
    assert alice["employee_id"] == 101
    assert alice["department"] == "engineering"
    assert alice["hire_date"] == datetime.date(2020, 6, 15)
    assert alice["shift_start"] == datetime.time(9, 0, 0)
    ts_alice = alice["last_login"]
    assert ts_alice.replace(tzinfo=None) == datetime.datetime(2024, 3, 15, 8, 30, 0)

    bob = result.records[1]
    assert bob["employee_name"] == "Bob"
    assert bob["employee_id"] == 102
    assert bob["department"] == "marketing"
    assert bob["hire_date"] == datetime.date(2019, 1, 10)
    assert bob["shift_start"] == datetime.time(8, 30, 0)
    ts_bob = bob["last_login"]
    assert ts_bob.replace(tzinfo=None) == datetime.datetime(2024, 3, 14, 17, 45, 0)
```

---

## Avro Type Assertion Notes

These are the expected Python types returned by fastavro for each Avro logical type. If goavro does not set the logical type metadata correctly, fastavro may return raw integers instead. In that case, tests need adjustment:

| Avro Type | Expected Python Type | Fallback (raw) |
|-----------|---------------------|-----------------|
| `{"type":"int","logicalType":"date"}` | `datetime.date` | `int` (days since epoch) |
| `{"type":"long","logicalType":"time-micros"}` | `datetime.time` | `int` (microseconds) |
| `{"type":"long","logicalType":"timestamp-micros"}` | `datetime.datetime` | `int` (microseconds since epoch) |
| `{"type":"enum","name":"...","symbols":[...]}` | `str` | N/A |

If during implementation fastavro returns raw integers for date/time/timestamp, the converter should store the raw int value and assertions should use epoch arithmetic. The test-code-specs above assume fastavro handles the logical type conversion, which is the standard behavior.
