# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run Commands

- **Build**: `./scripts/build.sh` (creates binary at `./build/csv_api`)
- **Run**: `./build parse --config_path <config> --data_path <csv> --output_path <output>`
- **Validate config**: `./build validate_config --config_path <config>`

## Architecture Overview

This is a CSV validation and Avro conversion tool with a concurrent pipeline architecture:

1. **Reader** (`internal/reader`) - Reads CSV files and validates headers against config
2. **Parser** (`internal/parser`) - Validates data using regex patterns and converts types
3. **Avro Writer** (`internal/avro_writer`) - Generates Avro schema and writes output

The main processing flow uses Go channels for concurrent execution:

```
CSV → Reader Channel → Parser Channel → Avro Writer → Avro File
```

### Key Components

- **Config System** (`internal/config`) - YAML-based field definitions with logical types (string, integer, decimal, enum, timestamp, date, time) and regex validation patterns
- **Error Tracking** (`internal/error_tracking`) - Centralized error collection with file/row/cell level reporting
- **Type System** - Each field has representations (regex patterns) that map to logical types with optional arguments

### Configuration Format

Fields are defined in YAML with:

- `name`: Field identifier
- `logical_type`: Type definition (name + optional args for decimals/enums)
- `representations`: Array of regex patterns for validation, with optional named groups and null handling

The pipeline processes data concurrently using goroutines and channels, with error tracking that can halt processing on fatal errors or collect validation errors for reporting.

## Adding New Logical Types

To add a new logical type to the system, you need to update several components:

### 1. Config System (`internal/config/config.go`)
- Add new type constant to `LogicalType` enum
- Create corresponding `*TypeConfig` struct implementing `LogicalTypeConfig` interface
- Update `ParseLogicalType()` function to handle the new type
- Add type mapping in `logical_type_mappings`

### 2. Parser (`internal/parser/converters.go`)
- Add new case in `Convert()` function switch statement
- Implement `convert_<type>()` function following the pattern:
  - Extract parameters from `args map[string]any`
  - Validate required parameters exist and have correct types
  - Perform type conversion with proper error handling
  - Return converted value or error

### 3. Avro Writer (`internal/avro_writer/avro_writer.go`)
- Update `map_type_json()` function to handle the new type config
- Return appropriate Avro type mapping:
  - Simple types: return JSON string like `"long"`, `"string"`
  - Complex types: return JSON object with type and logical type info
  - For decimal types: distinguish between float (`"double"`) and precise decimal with bytes+logicalType

### 4. Testing
- Add comprehensive tests in `tests/test_logical_types.py` covering:
  - Valid conversions with different parameter sets
  - Mixed types in same dataset
  - Validation failures and error handling
  - Edge cases specific to the type

### Type Implementation Notes
- **Parameter Extraction**: Use type assertions with proper error handling for `args` map
- **Error Messages**: Include context about which parameter failed and why
- **Avro Compatibility**: Ensure converted values match expected Avro schema types
- **Logical Types**: For Avro logical types (decimal, timestamp, etc.), implement proper encoding (e.g., decimal as scaled integer bytes)
