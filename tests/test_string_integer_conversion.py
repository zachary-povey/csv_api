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
        "--config_path", str(config_path),
        "--data_path", str(data_path),
        "--output_path", str(output_path)
    ]
    result = subprocess.run(cmd, capture_output=True, text=True)
    return result


def read_avro_file(file_path):
    """Read and return records from an Avro file"""
    records = []
    with open(file_path, 'rb') as f:
        reader = fastavro.reader(f)
        for record in reader:
            records.append(record)
    return records


def test_simple_string_fields(build_path):
    """Test basic string field parsing"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            'fields': [
                {
                    'name': 'name',
                    'logical_type': {'name': 'string'},
                    'representations': [{'pattern': '(?P<value>.*)'}]
                },
                {
                    'name': 'description',
                    'logical_type': {'name': 'string'},
                    'representations': [{'pattern': '(?P<value>.*)'}]
                }
            ]
        }
        config_path = os.path.join(tmpdir, 'config.yaml')
        with open(config_path, 'w') as f:
            yaml.dump(config, f)

        # Create CSV file
        csv_content = "name,description\nJohn,Software Engineer\nJane,Data Scientist\n"
        data_path = os.path.join(tmpdir, 'data.csv')
        with open(data_path, 'w') as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, 'output.avro')
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode == 0, f"Command failed: {result.stderr}"

        # Verify output
        records = read_avro_file(output_path)
        assert len(records) == 2
        assert records[0]['name'] == 'John'
        assert records[0]['description'] == 'Software Engineer'
        assert records[1]['name'] == 'Jane'
        assert records[1]['description'] == 'Data Scientist'


def test_simple_integer_fields(build_path):
    """Test basic integer field parsing"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            'fields': [
                {
                    'name': 'id',
                    'logical_type': {'name': 'integer'},
                    'representations': [{'pattern': '(?P<value>[0-9]+)'}]
                },
                {
                    'name': 'age',
                    'logical_type': {'name': 'integer'},
                    'representations': [{'pattern': '(?P<value>[0-9]+)'}]
                }
            ]
        }
        config_path = os.path.join(tmpdir, 'config.yaml')
        with open(config_path, 'w') as f:
            yaml.dump(config, f)

        # Create CSV file
        csv_content = "id,age\n1,25\n2,30\n"
        data_path = os.path.join(tmpdir, 'data.csv')
        with open(data_path, 'w') as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, 'output.avro')
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode == 0, f"Command failed: {result.stderr}"

        # Verify output
        records = read_avro_file(output_path)
        assert len(records) == 2
        assert records[0]['id'] == 1
        assert records[0]['age'] == 25
        assert records[1]['id'] == 2
        assert records[1]['age'] == 30


def test_mixed_string_integer_fields(build_path):
    """Test mixed string and integer fields"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            'fields': [
                {
                    'name': 'name',
                    'logical_type': {'name': 'string'},
                    'representations': [{'pattern': '(?P<value>.*)'}]
                },
                {
                    'name': 'score',
                    'logical_type': {'name': 'integer'},
                    'representations': [{'pattern': '(?P<value>[0-9]+)'}]
                },
                {
                    'name': 'category',
                    'logical_type': {'name': 'string'},
                    'representations': [{'pattern': '(?P<value>.*)'}]
                }
            ]
        }
        config_path = os.path.join(tmpdir, 'config.yaml')
        with open(config_path, 'w') as f:
            yaml.dump(config, f)

        # Create CSV file
        csv_content = "name,score,category\nAlice,95,A\nBob,87,B\n"
        data_path = os.path.join(tmpdir, 'data.csv')
        with open(data_path, 'w') as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, 'output.avro')
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode == 0, f"Command failed: {result.stderr}"

        # Verify output
        records = read_avro_file(output_path)
        assert len(records) == 2
        assert records[0]['name'] == 'Alice'
        assert records[0]['score'] == 95
        assert records[0]['category'] == 'A'
        assert records[1]['name'] == 'Bob'
        assert records[1]['score'] == 87
        assert records[1]['category'] == 'B'


def test_complex_integer_patterns(build_path):
    """Test integer fields with complex regex patterns"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file - similar to existing test_config.yaml
        config = {
            'fields': [
                {
                    'name': 'product',
                    'logical_type': {'name': 'string'},
                    'representations': [{'pattern': '(?P<value>.*)'}]
                },
                {
                    'name': 'price',
                    'logical_type': {'name': 'integer'},
                    'representations': [{'pattern': ' *(?P<value>[0-9]+) *gbp *'}]
                }
            ]
        }
        config_path = os.path.join(tmpdir, 'config.yaml')
        with open(config_path, 'w') as f:
            yaml.dump(config, f)

        # Create CSV file
        csv_content = "product,price\nWidget,10 gbp\nGadget, 25 gbp \n"
        data_path = os.path.join(tmpdir, 'data.csv')
        with open(data_path, 'w') as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, 'output.avro')
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode == 0, f"Command failed: {result.stderr}"

        # Verify output
        records = read_avro_file(output_path)
        assert len(records) == 2
        assert records[0]['product'] == 'Widget'
        assert records[0]['price'] == 10
        assert records[1]['product'] == 'Gadget'
        assert records[1]['price'] == 25


def test_string_with_special_characters(build_path):
    """Test string fields with special characters"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            'fields': [
                {
                    'name': 'text',
                    'logical_type': {'name': 'string'},
                    'representations': [{'pattern': '(?P<value>.*)'}]
                }
            ]
        }
        config_path = os.path.join(tmpdir, 'config.yaml')
        with open(config_path, 'w') as f:
            yaml.dump(config, f)

        # Create CSV file with special characters
        csv_content = 'text\n"Hello, World!"\n"Test with ""quotes"""\n'
        data_path = os.path.join(tmpdir, 'data.csv')
        with open(data_path, 'w') as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, 'output.avro')
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode == 0, f"Command failed: {result.stderr}"

        # Verify output
        records = read_avro_file(output_path)
        assert len(records) == 2
        assert records[0]['text'] == 'Hello, World!'
        assert records[1]['text'] == 'Test with "quotes"'


def test_integer_validation_failure(build_path):
    """Test that non-integer values fail validation"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            'fields': [
                {
                    'name': 'id',
                    'logical_type': {'name': 'integer'},
                    'representations': [{'pattern': '(?P<value>[0-9]+)'}]
                }
            ]
        }
        config_path = os.path.join(tmpdir, 'config.yaml')
        with open(config_path, 'w') as f:
            yaml.dump(config, f)

        # Create CSV file with invalid integer
        csv_content = "id\n123\nabc\n"
        data_path = os.path.join(tmpdir, 'data.csv')
        with open(data_path, 'w') as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, 'output.avro')
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode != 0, "Command should have failed"
        # Should have graceful error message about invalid value
        assert "abc" in result.stderr or "abc" in result.stdout, f"Error should mention invalid value 'abc': {result.stderr} {result.stdout}"


def test_regex_pattern_mismatch(build_path):
    """Test that values not matching regex pattern fail"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file with strict integer pattern (no letters allowed)
        config = {
            'fields': [
                {
                    'name': 'price',
                    'logical_type': {'name': 'integer'},
                    'representations': [{'pattern': '(?P<value>[0-9]+) gbp'}]  # Must end with " gbp"
                }
            ]
        }
        config_path = os.path.join(tmpdir, 'config.yaml')
        with open(config_path, 'w') as f:
            yaml.dump(config, f)

        # Create CSV file with value that doesn't match pattern
        csv_content = "price\n10 gbp\n15 dollars\n"  # "15 dollars" doesn't match " gbp" pattern
        data_path = os.path.join(tmpdir, 'data.csv')
        with open(data_path, 'w') as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, 'output.avro')
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode != 0, "Command should have failed"
        # Should have graceful error message about pattern mismatch
        assert ("dollars" in result.stderr or "dollars" in result.stdout or
                "pattern" in result.stderr.lower() or "does not match" in result.stderr.lower()), f"Error should mention pattern mismatch with 'dollars': {result.stderr} {result.stdout}"


def test_missing_required_field(build_path):
    """Test that missing required fields cause validation failure"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file with two required fields
        config = {
            'fields': [
                {
                    'name': 'name',
                    'logical_type': {'name': 'string'},
                    'representations': [{'pattern': '(?P<value>.*)'}]
                },
                {
                    'name': 'age',
                    'logical_type': {'name': 'integer'},
                    'representations': [{'pattern': '(?P<value>[0-9]+)'}]
                }
            ]
        }
        config_path = os.path.join(tmpdir, 'config.yaml')
        with open(config_path, 'w') as f:
            yaml.dump(config, f)

        # Create CSV file with missing 'age' column
        csv_content = "name\nJohn\nJane\n"
        data_path = os.path.join(tmpdir, 'data.csv')
        with open(data_path, 'w') as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, 'output.avro')
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode != 0, "Command should have failed"
        # Should have graceful error message about missing required field
        assert ("age" in result.stderr or "age" in result.stdout or
                "missing" in result.stderr.lower() or "required" in result.stderr.lower()), f"Error should mention missing required field 'age': {result.stderr} {result.stdout}"


def test_integer_overflow_or_invalid_format(build_path):
    """Test that invalid integer formats fail conversion"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            'fields': [
                {
                    'name': 'number',
                    'logical_type': {'name': 'integer'},
                    'representations': [{'pattern': '(?P<value>[0-9.]+)'}]  # Pattern allows decimals but logical type is integer
                }
            ]
        }
        config_path = os.path.join(tmpdir, 'config.yaml')
        with open(config_path, 'w') as f:
            yaml.dump(config, f)

        # Create CSV file with decimal value
        csv_content = "number\n123\n45.67\n"  # 45.67 matches pattern but isn't valid integer
        data_path = os.path.join(tmpdir, 'data.csv')
        with open(data_path, 'w') as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, 'output.avro')
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode != 0, "Command should have failed"
        # Should have graceful error message about invalid integer format
        assert ("45.67" in result.stderr or "45.67" in result.stdout or
                "invalid integer" in result.stderr.lower() or "not a valid integer" in result.stderr.lower()), f"Error should mention invalid integer '45.67': {result.stderr} {result.stdout}"


def test_empty_required_field(build_path):
    """Test that empty values in required fields fail validation"""
    with tempfile.TemporaryDirectory() as tmpdir:
        # Create config file
        config = {
            'fields': [
                {
                    'name': 'name',
                    'logical_type': {'name': 'string'},
                    'representations': [{'pattern': '(?P<value>.+)'}]  # Pattern requires at least one character
                }
            ]
        }
        config_path = os.path.join(tmpdir, 'config.yaml')
        with open(config_path, 'w') as f:
            yaml.dump(config, f)

        # Create CSV file with empty value
        csv_content = 'name\nJohn\n""\nJane\n'  # Empty string in second row
        data_path = os.path.join(tmpdir, 'data.csv')
        with open(data_path, 'w') as f:
            f.write(csv_content)

        # Run conversion
        output_path = os.path.join(tmpdir, 'output.avro')
        result = run_csv_api(build_path, config_path, data_path, output_path)

        assert result.returncode != 0, "Command should have failed"
        # Should have graceful error message about empty value or pattern mismatch
        assert ("did not match" in result.stderr.lower() or "pattern" in result.stderr.lower() or
                "value ''" in result.stderr), f"Error should mention pattern mismatch with empty value: {result.stderr} {result.stdout}"