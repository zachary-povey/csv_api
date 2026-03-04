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
