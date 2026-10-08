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
    result = run_fixture(build_path, "missing_required_field")
    assert result.returncode != 0, "Command should have failed"
    assert (
        "age" in result.stderr
        or "age" in result.stdout
        or "missing" in result.stderr.lower()
        or "required" in result.stderr.lower()
    ), f"Error should mention missing required field 'age': {result.stderr} {result.stdout}"


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
