import pytest
import subprocess
import tempfile
import os
import yaml
import fastavro
from pathlib import Path


@pytest.fixture
def build_path():

    return Path(__file__).parent.parent / "build" / "csv_api"


def run_csv_api(build_path, config_path, data_path, output_path):
    """Run the csv_api binary with given parameters"""
    cmd = [
        str(build_path),
        "parse",
        "--config_path",
        str(config_path),
        "--data_path",
        str(data_path),
        "--output_path",
        str(output_path),
    ]
    result = subprocess.run(cmd, capture_output=True, text=True)
    return result


def read_avro_file(file_path):
    """Read and return records from an Avro file"""
    records = []
    with open(file_path, "rb") as f:
        reader = fastavro.reader(f)
        for record in reader:
            records.append(record)
    return records


def test_simple_string_fields(build_path):
    """Test basic string field parsing"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            "fields": [
                {
                    "name": "name",
                    "logical_type": {"name": "string"},
                    "representations": [{"pattern": "(?P<value>.*)"}],
                },
                {
                    "name": "description",
                    "logical_type": {"name": "string"},
                    "representations": [{"pattern": "(?P<value>.*)"}],
                },
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file
        csv_content = "name,description\nJohn,Software Engineer\nJane,Data Scientist\n"
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode == 0, f"Command failed: {result.stderr}"

        # Verify output
        records = read_avro_file(output_path)
        assert len(records) == 2
        assert records[0]["name"] == "John"
        assert records[0]["description"] == "Software Engineer"
        assert records[1]["name"] == "Jane"
        assert records[1]["description"] == "Data Scientist"


def test_simple_integer_fields(build_path):
    """Test basic integer field parsing"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            "fields": [
                {
                    "name": "id",
                    "logical_type": {"name": "integer"},
                    "representations": [{"pattern": "(?P<value>[0-9]+)"}],
                },
                {
                    "name": "age",
                    "logical_type": {"name": "integer"},
                    "representations": [{"pattern": "(?P<value>[0-9]+)"}],
                },
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file
        csv_content = "id,age\n1,25\n2,30\n"
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode == 0, f"Command failed: {result.stderr}"

        # Verify output
        records = read_avro_file(output_path)
        assert len(records) == 2
        assert records[0]["id"] == 1
        assert records[0]["age"] == 25
        assert records[1]["id"] == 2
        assert records[1]["age"] == 30


def test_mixed_string_integer_fields(build_path):
    """Test mixed string and integer fields"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            "fields": [
                {
                    "name": "name",
                    "logical_type": {"name": "string"},
                    "representations": [{"pattern": "(?P<value>.*)"}],
                },
                {
                    "name": "score",
                    "logical_type": {"name": "integer"},
                    "representations": [{"pattern": "(?P<value>[0-9]+)"}],
                },
                {
                    "name": "category",
                    "logical_type": {"name": "string"},
                    "representations": [{"pattern": "(?P<value>.*)"}],
                },
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file
        csv_content = "name,score,category\nAlice,95,A\nBob,87,B\n"
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode == 0, f"Command failed: {result.stderr}"

        # Verify output
        records = read_avro_file(output_path)
        assert len(records) == 2
        assert records[0]["name"] == "Alice"
        assert records[0]["score"] == 95
        assert records[0]["category"] == "A"
        assert records[1]["name"] == "Bob"
        assert records[1]["score"] == 87
        assert records[1]["category"] == "B"


def test_complex_integer_patterns(build_path):
    """Test integer fields with complex regex patterns"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file - similar to existing test_config.yaml
        config = {
            "fields": [
                {
                    "name": "product",
                    "logical_type": {"name": "string"},
                    "representations": [{"pattern": "(?P<value>.*)"}],
                },
                {
                    "name": "price",
                    "logical_type": {"name": "integer"},
                    "representations": [{"pattern": " *(?P<value>[0-9]+) *gbp *"}],
                },
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file
        csv_content = "product,price\nWidget,10 gbp\nGadget, 25 gbp \n"
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode == 0, f"Command failed: {result.stderr}"

        # Verify output
        records = read_avro_file(output_path)
        assert len(records) == 2
        assert records[0]["product"] == "Widget"
        assert records[0]["price"] == 10
        assert records[1]["product"] == "Gadget"
        assert records[1]["price"] == 25


def test_string_with_special_characters(build_path):
    """Test string fields with special characters"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            "fields": [
                {
                    "name": "text",
                    "logical_type": {"name": "string"},
                    "representations": [{"pattern": "(?P<value>.*)"}],
                }
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file with special characters
        csv_content = 'text\n"Hello, World!"\n"Test with ""quotes"""\n'
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode == 0, f"Command failed: {result.stderr}"

        # Verify output
        records = read_avro_file(output_path)
        assert len(records) == 2
        assert records[0]["text"] == "Hello, World!"
        assert records[1]["text"] == 'Test with "quotes"'


def test_integer_validation_failure(build_path):
    """Test that non-integer values fail validation"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            "fields": [
                {
                    "name": "id",
                    "logical_type": {"name": "integer"},
                    "representations": [{"pattern": "(?P<value>[0-9]+)"}],
                }
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file with invalid integer
        csv_content = "id\n123\nabc\n"
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode != 0, "Command should have failed"
        # Should have graceful error message about invalid value
        assert (
            "abc" in result.stderr or "abc" in result.stdout
        ), f"Error should mention invalid value 'abc': {result.stderr} {result.stdout}"


def test_regex_pattern_mismatch(build_path):
    """Test that values not matching regex pattern fail"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file with strict integer pattern (no letters allowed)
        config = {
            "fields": [
                {
                    "name": "price",
                    "logical_type": {"name": "integer"},
                    "representations": [
                        {"pattern": "(?P<value>[0-9]+) gbp"}
                    ],  # Must end with " gbp"
                }
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file with value that doesn't match pattern
        csv_content = (
            "price\n10 gbp\n15 dollars\n"  # "15 dollars" doesn't match " gbp" pattern
        )
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode != 0, "Command should have failed"
        # Should have graceful error message about pattern mismatch
        assert (
            "dollars" in result.stderr
            or "dollars" in result.stdout
            or "pattern" in result.stderr.lower()
            or "does not match" in result.stderr.lower()
        ), f"Error should mention pattern mismatch with 'dollars': {result.stderr} {result.stdout}"


def test_missing_required_field(build_path):
    """Test that missing required fields cause validation failure"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file with two required fields
        config = {
            "fields": [
                {
                    "name": "name",
                    "logical_type": {"name": "string"},
                    "representations": [{"pattern": "(?P<value>.*)"}],
                },
                {
                    "name": "age",
                    "logical_type": {"name": "integer"},
                    "representations": [{"pattern": "(?P<value>[0-9]+)"}],
                },
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file with missing 'age' column
        csv_content = "name\nJohn\nJane\n"
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode != 0, "Command should have failed"
        # Should have graceful error message about missing required field
        assert (
            "age" in result.stderr
            or "age" in result.stdout
            or "missing" in result.stderr.lower()
            or "required" in result.stderr.lower()
        ), f"Error should mention missing required field 'age': {result.stderr} {result.stdout}"


def test_integer_overflow_or_invalid_format(build_path):
    """Test that invalid integer formats fail conversion"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            "fields": [
                {
                    "name": "number",
                    "logical_type": {"name": "integer"},
                    "representations": [
                        {"pattern": "(?P<value>[0-9.]+)"}
                    ],  # Pattern allows decimals but logical type is integer
                }
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file with decimal value
        csv_content = (
            "number\n123\n45.67\n"  # 45.67 matches pattern but isn't valid integer
        )
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode != 0, "Command should have failed"
        # Should have graceful error message about invalid integer format
        assert (
            "45.67" in result.stderr
            or "45.67" in result.stdout
            or "invalid integer" in result.stderr.lower()
            or "not a valid integer" in result.stderr.lower()
        ), f"Error should mention invalid integer '45.67': {result.stderr} {result.stdout}"


def test_empty_required_field(build_path):
    """Test that empty values in required fields fail validation"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            "fields": [
                {
                    "name": "name",
                    "logical_type": {"name": "string"},
                    "representations": [
                        {"pattern": "(?P<value>.+)"}
                    ],  # Pattern requires at least one character
                }
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file with empty value
        csv_content = 'name\nJohn\n""\nJane\n'  # Empty string in second row
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode != 0, "Command should have failed"
        # Should have graceful error message about empty value or pattern mismatch
        assert (
            "did not match" in result.stderr.lower()
            or "pattern" in result.stderr.lower()
            or "value ''" in result.stderr
        ), f"Error should mention pattern mismatch with empty value: {result.stderr} {result.stdout}"


def test_decimal_single_value_as_float(build_path):
    """Test decimal type with single value representation and as_float=true"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            "fields": [
                {
                    "name": "price",
                    "logical_type": {"name": "decimal", "args": {"as_float": True}},
                    "representations": [{"pattern": "(?P<value>[0-9]+\\.[0-9]+)"}],
                }
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file
        csv_content = "price\n19.99\n25.50\n"
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode == 0, f"Command failed: {result.stderr}"

        # Verify output
        records = read_avro_file(output_path)
        assert len(records) == 2
        assert records[0]["price"] == 19.99
        assert records[1]["price"] == 25.50


def test_decimal_single_value_with_precision_scale(build_path):
    """Test decimal type with single value representation using precision and scale"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            "fields": [
                {
                    "name": "amount",
                    "logical_type": {
                        "name": "decimal",
                        "args": {"precision": 5, "scale": 2},
                    },
                    "representations": [{"pattern": "(?P<value>[0-9]+\\.[0-9]{2})"}],
                }
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file
        csv_content = "amount\n123.45\n999.99\n"
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode == 0, f"Command failed: {result.stderr}"

        # Verify output - should be precise decimal as bytes (Avro decimal logical type)
        records = read_avro_file(output_path)
        assert len(records) == 2
        # With precision/scale, values should be stored as Avro decimal logical type
        # The fastavro library should automatically decode these to Python Decimal objects
        import decimal

        assert records[0]["amount"] == decimal.Decimal("123.45")
        assert records[1]["amount"] == decimal.Decimal("999.99")


def test_decimal_separate_integer_decimal_parts(build_path):
    """Test decimal type with separate integer_part and decimal_part parameters"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            "fields": [
                {
                    "name": "currency",
                    "logical_type": {"name": "decimal", "args": {"as_float": True}},
                    "representations": [
                        {
                            "pattern": "\\$(?P<integer_part>[0-9]+)\\.(?P<decimal_part>[0-9]{2})"
                        }
                    ],
                }
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file with dollar format
        csv_content = "currency\n$42.15\n$156.78\n"
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode == 0, f"Command failed: {result.stderr}"

        # Verify output
        records = read_avro_file(output_path)
        assert len(records) == 2
        assert records[0]["currency"] == 42.15
        assert records[1]["currency"] == 156.78


def test_decimal_precision_scale_with_integer_decimal_parts(build_path):
    """Test decimal type with precision/scale using integer_part and decimal_part"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            "fields": [
                {
                    "name": "measurement",
                    "logical_type": {
                        "name": "decimal",
                        "args": {"precision": 6, "scale": 3},
                    },
                    "representations": [
                        {
                            "pattern": "(?P<integer_part>[0-9]+) and (?P<decimal_part>[0-9]{3})/1000"
                        }
                    ],
                }
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file with fractional representation
        csv_content = "measurement\n25 and 125/1000\n99 and 999/1000\n"
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode == 0, f"Command failed: {result.stderr}"

        # Verify output
        records = read_avro_file(output_path)
        assert len(records) == 2
        # Should reconstruct decimal from parts: 25.125 and 99.999 as Avro decimals
        import decimal

        assert records[0]["measurement"] == decimal.Decimal("25.125")
        assert records[1]["measurement"] == decimal.Decimal("99.999")


def test_decimal_mixed_with_other_types(build_path):
    """Test decimal fields mixed with string and integer fields"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            "fields": [
                {
                    "name": "product",
                    "logical_type": {"name": "string"},
                    "representations": [{"pattern": "(?P<value>.+)"}],
                },
                {
                    "name": "price",
                    "logical_type": {"name": "decimal", "args": {"as_float": True}},
                    "representations": [{"pattern": "(?P<value>[0-9]+\\.[0-9]+)"}],
                },
                {
                    "name": "quantity",
                    "logical_type": {"name": "integer"},
                    "representations": [{"pattern": "(?P<value>[0-9]+)"}],
                },
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file
        csv_content = "product,price,quantity\nApple,1.25,10\nBanana,0.75,15\n"
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode == 0, f"Command failed: {result.stderr}"

        # Verify output
        records = read_avro_file(output_path)
        assert len(records) == 2
        assert records[0]["product"] == "Apple"
        assert records[0]["price"] == 1.25
        assert records[0]["quantity"] == 10
        assert records[1]["product"] == "Banana"
        assert records[1]["price"] == 0.75
        assert records[1]["quantity"] == 15


def test_decimal_validation_failures(build_path):
    """Test decimal validation failure scenarios"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file expecting decimal format
        config = {
            "fields": [
                {
                    "name": "price",
                    "logical_type": {"name": "decimal", "args": {"as_float": True}},
                    "representations": [
                        {"pattern": "(?P<value>[0-9]+\\.[0-9]{2})"}
                    ],  # Requires exactly 2 decimal places
                }
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file with invalid decimal format
        csv_content = "price\n19.99\ninvalid_price\n25.5\n"  # Second value is non-numeric, third has wrong decimal places
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode != 0, "Command should have failed"
        # Should report pattern matching failures
        assert (
            "did not match" in result.stderr.lower()
            or "invalid_price" in result.stderr
            or "pattern" in result.stderr.lower()
        ), f"Error should mention validation failures: {result.stderr} {result.stdout}"


def test_decimal_precision_scale_validation_failure(build_path):
    """Test that invalid precision/scale arguments are caught"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file with conflicting decimal args (both as_float and precision)
        config = {
            "fields": [
                {
                    "name": "amount",
                    "logical_type": {
                        "name": "decimal",
                        "args": {
                            "as_float": True,
                            "precision": 5,  # Should be mutually exclusive with as_float
                            "scale": 2,
                        },
                    },
                    "representations": [{"pattern": "(?P<value>[0-9]+\\.[0-9]+)"}],
                }
            ]
        }
        config_path = os.path.join(tmpdir, "config.yaml")
        with open(config_path, "w") as f:
            yaml.dump(config, f)

        # Create CSV file
        csv_content = "amount\n123.45\n"
        data_path = os.path.join(tmpdir, "data.csv")
        with open(data_path, "w") as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, "output.avro")
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert (
            result.returncode != 0
        ), "Command should have failed due to invalid config"
        # Should report configuration error about conflicting decimal args
        assert (
            "config" in result.stderr.lower()
            or "precision" in result.stderr
            or "as_float" in result.stderr
            or "mutually exclusive" in result.stderr.lower()
        ), f"Error should mention config validation issue: {result.stderr} {result.stdout}"
