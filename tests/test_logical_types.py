import datetime
import decimal
from tests.utils import run_fixture


def test_simple_string_fields(build_path):
    result = run_fixture(build_path, "simple_strings")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    assert result.records[0]["name"] == "John"
    assert result.records[0]["description"] == "Software Engineer"
    assert result.records[1]["name"] == "Jane"
    assert result.records[1]["description"] == "Data Scientist"


def test_simple_integer_fields(build_path):
    result = run_fixture(build_path, "simple_integers")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    assert result.records[0]["id"] == 1
    assert result.records[0]["age"] == 25
    assert result.records[1]["id"] == 2
    assert result.records[1]["age"] == 30


def test_mixed_string_integer_fields(build_path):
    result = run_fixture(build_path, "mixed_string_integer")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    assert result.records[0]["name"] == "Alice"
    assert result.records[0]["score"] == 95
    assert result.records[0]["category"] == "A"
    assert result.records[1]["name"] == "Bob"
    assert result.records[1]["score"] == 87
    assert result.records[1]["category"] == "B"


def test_complex_integer_patterns(build_path):
    result = run_fixture(build_path, "complex_integer_patterns")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    assert result.records[0]["product"] == "Widget"
    assert result.records[0]["price"] == 10
    assert result.records[1]["product"] == "Gadget"
    assert result.records[1]["price"] == 25


def test_integer_static_numeric_arg(build_path):
    """TASK-2 AC#1, AC#2, AC#3: numeric static `value` args (YAML int and
    whole-number float) convert on an integer field, alongside string values
    from capture groups."""
    result = run_fixture(build_path, "integer_static_numeric_arg")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert [r["id"] for r in result.records] == [1, 2, 3, 4]
    # AC#1: numeric static args
    assert result.records[0]["quantity"] == 0
    assert result.records[1]["quantity"] == 12
    # AC#2: capture group strings still convert
    assert result.records[2]["quantity"] == 42
    assert result.records[3]["quantity"] == -7


def test_string_with_special_characters(build_path):
    result = run_fixture(build_path, "string_special_chars")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    assert result.records[0]["text"] == "Hello, World!"
    assert result.records[1]["text"] == 'Test with "quotes"'


def test_integer_validation_failure(build_path):
    result = run_fixture(build_path, "integer_invalid_value")
    assert result.returncode != 0, "Command should have failed"
    assert (
        "abc" in result.stderr or "abc" in result.stdout
    ), f"Error should mention invalid value 'abc': {result.stderr} {result.stdout}"


def test_regex_pattern_mismatch(build_path):
    result = run_fixture(build_path, "regex_pattern_mismatch")
    assert result.returncode != 0, "Command should have failed"
    assert (
        "dollars" in result.stderr
        or "dollars" in result.stdout
        or "pattern" in result.stderr.lower()
        or "does not match" in result.stderr.lower()
    ), f"Error should mention pattern mismatch with 'dollars': {result.stderr} {result.stdout}"


def test_missing_required_field(build_path):
    """TASK-8 AC#3: fields are required by default; a missing one is a
    file-level error."""
    result = run_fixture(build_path, "missing_required_field")
    assert result.returncode != 0, "Command should have failed"
    assert (
        "age" in result.stderr
        or "age" in result.stdout
        or "missing" in result.stderr.lower()
        or "required" in result.stderr.lower()
    ), f"Error should mention missing required field 'age': {result.stderr} {result.stdout}"
    assert "missing some required fields: [age]" in result.stderr
    assert "file-level errors" in result.stderr
    assert not result.output_exists


def test_integer_overflow_or_invalid_format(build_path):
    result = run_fixture(build_path, "integer_invalid_format")
    assert result.returncode != 0, "Command should have failed"
    assert (
        "45.67" in result.stderr
        or "45.67" in result.stdout
        or "invalid integer" in result.stderr.lower()
        or "not a valid integer" in result.stderr.lower()
    ), f"Error should mention invalid integer '45.67': {result.stderr} {result.stdout}"


def test_empty_required_field(build_path):
    result = run_fixture(build_path, "empty_required_field")
    assert result.returncode != 0, "Command should have failed"
    assert (
        "did not match" in result.stderr.lower()
        or "pattern" in result.stderr.lower()
        or "value ''" in result.stderr
    ), f"Error should mention pattern mismatch with empty value: {result.stderr} {result.stdout}"


def test_decimal_single_value_as_float(build_path):
    result = run_fixture(build_path, "decimal_as_float")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    assert result.records[0]["price"] == 19.99
    assert result.records[1]["price"] == 25.50


def test_decimal_single_value_with_precision_scale(build_path):
    result = run_fixture(build_path, "decimal_precision_scale")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    assert result.records[0]["amount"] == decimal.Decimal("123.45")
    assert result.records[1]["amount"] == decimal.Decimal("999.99")


def test_decimal_separate_integer_decimal_parts(build_path):
    result = run_fixture(build_path, "decimal_split_parts")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    assert result.records[0]["currency"] == 42.15
    assert result.records[1]["currency"] == 156.78


def test_decimal_precision_scale_with_integer_decimal_parts(build_path):
    result = run_fixture(build_path, "decimal_precision_scale_split_parts")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    assert result.records[0]["measurement"] == decimal.Decimal("25.125")
    assert result.records[1]["measurement"] == decimal.Decimal("99.999")


def test_decimal_split_parts_keep_leading_zeros(build_path):
    result = run_fixture(build_path, "decimal_split_parts_leading_zeros")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert [r["currency"] for r in result.records] == [12.05, 0.07, -3.01]


def test_decimal_precision_scale_split_parts_keep_leading_zeros(build_path):
    result = run_fixture(
        build_path, "decimal_precision_scale_split_parts_leading_zeros"
    )
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert [r["measurement"] for r in result.records] == [
        decimal.Decimal("12.050"),
        decimal.Decimal("0.005"),
        decimal.Decimal("-3.007"),
    ]


def test_decimal_mixed_with_other_types(build_path):
    result = run_fixture(build_path, "decimal_mixed_types")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    assert result.records[0]["product"] == "Apple"
    assert result.records[0]["price"] == 1.25
    assert result.records[0]["quantity"] == 10
    assert result.records[1]["product"] == "Banana"
    assert result.records[1]["price"] == 0.75
    assert result.records[1]["quantity"] == 15


def test_decimal_validation_failures(build_path):
    result = run_fixture(build_path, "decimal_invalid_format")
    assert result.returncode != 0, "Command should have failed"
    assert (
        "did not match" in result.stderr.lower()
        or "invalid_price" in result.stderr
        or "pattern" in result.stderr.lower()
    ), f"Error should mention validation failures: {result.stderr} {result.stdout}"


def test_decimal_precision_scale_validation_failure(build_path):
    result = run_fixture(build_path, "decimal_conflicting_args")
    assert result.returncode != 0, "Command should have failed due to invalid config"
    assert (
        "config" in result.stderr.lower()
        or "precision" in result.stderr
        or "as_float" in result.stderr
        or "mutually exclusive" in result.stderr.lower()
    ), f"Error should mention config validation issue: {result.stderr} {result.stdout}"


# --- Enum Tests ---


def test_enum_basic(build_path):
    result = run_fixture(build_path, "enum_basic")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["status"] == "active"
    assert result.records[1]["status"] == "inactive"
    assert result.records[2]["status"] == "pending"


def test_enum_case_insensitive(build_path):
    result = run_fixture(build_path, "enum_case_insensitive")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["status"] == "active"
    assert result.records[1]["status"] == "inactive"
    assert result.records[2]["status"] == "active"


def test_enum_value_remap(build_path):
    result = run_fixture(build_path, "enum_value_remap")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["status"] == "active"
    assert result.records[1]["status"] == "inactive"
    assert result.records[2]["status"] == "active"


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


def test_enum_invalid_value(build_path):
    result = run_fixture(build_path, "enum_invalid_value")
    assert result.returncode != 0, "Command should have failed"
    assert (
        "unknown" in result.stderr.lower()
        or "permitted" in result.stderr.lower()
        or "invalid" in result.stderr.lower()
    ), f"Error should mention invalid enum value: {result.stderr} {result.stdout}"


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


# --- Date Tests ---


def test_date_iso(build_path):
    result = run_fixture(build_path, "date_iso")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["event_date"] == datetime.date(2024, 3, 15)
    assert result.records[1]["event_date"] == datetime.date(2000, 1, 1)
    assert result.records[2]["event_date"] == datetime.date(1999, 12, 31)


def test_date_custom_format(build_path):
    result = run_fixture(build_path, "date_custom_format")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["event_date"] == datetime.date(2024, 3, 15)
    assert result.records[1]["event_date"] == datetime.date(2000, 1, 1)
    assert result.records[2]["event_date"] == datetime.date(1999, 12, 31)


def test_date_invalid_components(build_path):
    result = run_fixture(build_path, "date_invalid_components")
    assert result.returncode != 0, "Command should have failed"
    assert (
        "13" in result.stderr
        or "invalid" in result.stderr.lower()
        or "date" in result.stderr.lower()
    ), f"Error should mention invalid date: {result.stderr} {result.stdout}"


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


# --- Time Tests ---


def test_time_iso(build_path):
    result = run_fixture(build_path, "time_iso")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["event_time"] == datetime.time(9, 30, 0)
    assert result.records[1]["event_time"] == datetime.time(14, 15, 45)
    assert result.records[2]["event_time"] == datetime.time(23, 0, 0)


def test_time_fractional_seconds(build_path):
    result = run_fixture(build_path, "time_fractional_seconds")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["event_time"] == datetime.time(9, 30, 0, 123456)
    assert result.records[1]["event_time"] == datetime.time(14, 15, 45, 500000)
    assert result.records[2]["event_time"] == datetime.time(0, 0, 0, 1)


def test_time_custom_format(build_path):
    result = run_fixture(build_path, "time_custom_format")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["event_time"] == datetime.time(9, 30, 0)
    assert result.records[1]["event_time"] == datetime.time(14, 0, 0)
    assert result.records[2]["event_time"] == datetime.time(0, 0, 0)


def test_time_edge_cases(build_path):
    result = run_fixture(build_path, "time_edge_cases")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 3
    assert result.records[0]["event_time"] == datetime.time(0, 0, 0)
    assert result.records[1]["event_time"] == datetime.time(23, 59, 59)
    assert result.records[2]["event_time"] == datetime.time(12, 0, 0)


# --- Timestamp Tests ---


def test_timestamp_iso_offset(build_path):
    result = run_fixture(build_path, "timestamp_iso_offset")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    ts0 = result.records[0]["created_at"]
    assert ts0.replace(tzinfo=None) == datetime.datetime(2024, 3, 15, 5, 30, 0)
    ts1 = result.records[1]["created_at"]
    assert ts1.replace(tzinfo=None) == datetime.datetime(2024, 1, 1, 8, 0, 0)


def test_timestamp_iso_utc(build_path):
    result = run_fixture(build_path, "timestamp_iso_utc")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    ts0 = result.records[0]["created_at"]
    assert ts0.replace(tzinfo=None) == datetime.datetime(2024, 3, 15, 10, 30, 0)
    ts1 = result.records[1]["created_at"]
    assert ts1.replace(tzinfo=None) == datetime.datetime(2024, 1, 1, 0, 0, 0)


def test_timestamp_custom_format(build_path):
    result = run_fixture(build_path, "timestamp_custom_format")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    ts0 = result.records[0]["created_at"]
    assert ts0.replace(tzinfo=None) == datetime.datetime(2024, 3, 15, 10, 30, 0)
    ts1 = result.records[1]["created_at"]
    assert ts1.replace(tzinfo=None) == datetime.datetime(2000, 1, 1, 0, 0, 0)


def test_timestamp_epoch(build_path):
    result = run_fixture(build_path, "timestamp_epoch")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    ts0 = result.records[0]["created_at"]
    assert ts0.replace(tzinfo=None) == datetime.datetime(2024, 3, 15, 10, 50, 0)
    ts1 = result.records[1]["created_at"]
    assert ts1.replace(tzinfo=None) == datetime.datetime(2000, 1, 1, 0, 0, 0)


def test_timestamp_timezone_conversion(build_path):
    result = run_fixture(build_path, "timestamp_timezone_conversion")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert len(result.records) == 2
    ts0 = result.records[0]["created_at"]
    assert ts0.replace(tzinfo=None) == datetime.datetime(2024, 3, 15, 5, 0, 0)
    ts1 = result.records[1]["created_at"]
    assert ts1.replace(tzinfo=None) == datetime.datetime(2024, 7, 2, 0, 0, 0)


def test_timestamp_invalid(build_path):
    result = run_fixture(build_path, "timestamp_invalid")
    assert result.returncode != 0, "Command should have failed"
    assert (
        "25" in result.stderr
        or "invalid" in result.stderr.lower()
        or "hour" in result.stderr.lower()
        or "timestamp" in result.stderr.lower()
    ), f"Error should mention invalid timestamp: {result.stderr} {result.stdout}"


# --- Walking Skeleton ---


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


# --- TASK-7: nullable fields and is_null representations ---

NULLABLE_FIELDS = ["label", "count", "price", "ratio", "status", "day", "at", "seen"]


def test_nullable_fields_are_unions_with_null(build_path):
    """TASK-7 AC#1: a nullable field's Avro type is ["null", <type>] with a
    null default, for every logical type; a field without nullable stays a
    plain (non-union) type."""
    result = run_fixture(build_path, "nullable_all_types")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    fields = {f["name"]: f for f in result.writer_schema["fields"]}
    assert fields["id"]["type"] == "long"
    assert "default" not in fields["id"]
    for name in NULLABLE_FIELDS:
        field_type = fields[name]["type"]
        assert isinstance(field_type, list) and field_type[0] == "null", name
        assert len(field_type) == 2, name
        assert fields[name]["default"] is None, name
    assert fields["label"]["type"][1] == "string"
    assert fields["count"]["type"][1] == "long"
    assert fields["price"]["type"][1]["logicalType"] == "decimal"
    assert fields["ratio"]["type"][1] == "double"
    assert fields["status"]["type"][1]["name"] == "com.example.status"
    assert fields["day"]["type"][1]["logicalType"] == "date"
    assert fields["at"]["type"][1]["logicalType"] == "time-micros"
    assert fields["seen"]["type"][1]["logicalType"] == "timestamp-micros"


def test_nullable_fields_write_null_and_non_null_values(build_path):
    """TASK-7 AC#1, AC#2: every logical type, when nullable, writes non-null
    values through the union and writes null where an is_null representation
    matches (including an is_null pattern with an unnamed group, "^(-|N/A)$")."""
    result = run_fixture(build_path, "nullable_all_types")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    first, nulls, third = result.records
    assert first == {
        "id": 1,
        "label": "hello",
        "count": 5,
        "price": decimal.Decimal("12.34"),
        "ratio": 0.5,
        "status": "open",
        "day": datetime.date(2024, 3, 15),
        "at": datetime.time(9, 30),
        "seen": datetime.datetime(2023, 11, 14, 22, 13, 20, tzinfo=datetime.timezone.utc),
    }
    # AC#2: "-" matches each field's is_null representation
    assert nulls == {"id": 2, **{name: None for name in NULLABLE_FIELDS}}
    assert third["label"] is None  # "N/A" also matches the label's is_null pattern
    assert third["count"] == 7
    assert third["price"] == decimal.Decimal("1.00")
    assert third["ratio"] == 1.25
    assert third["status"] == "closed"
    assert third["day"] == datetime.date(1970, 1, 2)
    assert third["at"] == datetime.time(0, 1)
    assert third["seen"] == datetime.datetime(1970, 1, 1, tzinfo=datetime.timezone.utc)


def test_nullable_enum_without_namespace(build_path):
    """TASK-7 AC#1, AC#2: a nullable enum in a config with no namespace, with
    an is_null representation matching the empty string."""
    result = run_fixture(build_path, "nullable_unnamespaced_enum")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert result.writer_schema["fields"][0]["type"][1]["name"] == "status"
    assert [r["status"] for r in result.records] == ["open", None, "closed"]


def test_empty_cell_on_nullable_field_without_null_representation_fails(build_path):
    """TASK-7 (decision): nulls only come from is_null representations; an
    empty cell on a nullable field that matches no representation is a data
    failure like any other unmatched value."""
    result = run_fixture(build_path, "nullable_empty_cell_unmatched", ["--fail-fast=false"])
    assert result.returncode != 0, "Command should have failed"
    assert not result.output_exists
    assert "row 3, column 'count': value '' did not match any pattern" in result.stderr
    assert "row 4" not in result.stderr


# --- TASK-8: optional columns (required: false) ---


def test_missing_optional_column_is_written_as_null(build_path):
    """TASK-8 AC#1: a field with required: false may be missing from the file;
    its values are written as null. The file's columns are in a different
    order from the config and include an extra column, so present columns
    must still be mapped by header name."""
    result = run_fixture(build_path, "optional_column_missing")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert result.records == [
        {"id": 1, "note": None, "amount": decimal.Decimal("1.50")},
        {"id": 2, "note": None, "amount": None},
        {"id": 3, "note": None, "amount": decimal.Decimal("3.00")},
    ]
    fields = {f["name"]: f for f in result.writer_schema["fields"]}
    assert fields["note"]["type"] == ["null", "string"]
    assert fields["note"]["default"] is None


def test_optional_column_present_is_parsed_normally(build_path):
    """TASK-8 AC#1: when a required: false column is present, its values are
    parsed like any other column (including is_null representations)."""
    result = run_fixture(build_path, "optional_column_present")
    assert result.returncode == 0, f"Command failed: {result.stderr}"
    assert result.records == [
        {"id": 1, "note": "hello", "amount": decimal.Decimal("1.50")},
        {"id": 2, "note": None, "amount": None},
    ]


def test_missing_required_column_is_file_level_error_alongside_optional(build_path):
    """TASK-8 AC#3: a missing required field is still a file-level error when
    the config also has optional fields; only the required field is named."""
    result = run_fixture(build_path, "missing_required_with_optional")
    assert result.returncode != 0, "Command should have failed"
    assert not result.output_exists
    assert "file-level errors" in result.stderr
    assert "missing some required fields: [id]" in result.stderr
