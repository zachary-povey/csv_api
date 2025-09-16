# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run Commands

- **Build**: `./scripts/build.sh` (creates binary at `./build`)
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