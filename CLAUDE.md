# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run Commands

- **Build**: `./scripts/build.sh` (creates binary at `./build/csv-api`)
- **Run**: `./build/csv-api parse --config-path <config> --input-path <csv> --output-path <output>` (short flags: `-c`, `-i`, `-o`)
- **Validate config**: `./build/csv-api validate_config --config-path <config>`
- **Test**: `python3 -m pytest tests` (builds the binary first; set `BUILD_PATH` to test an existing binary; needs `pip install -r tests/requirements.txt`)

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
- **Error Tracking** (`internal/error_tracking`) - Centralized error collection with file/row/cell level reporting. Closes `KillCh` to stop the pipeline (execution errors, fail-fast, max-errors); the reader stops on it and the parser keeps draining its input so nothing blocks
- **Options** (`internal/options`) - Error handling flags for `parse` (`--error-on-data-failures`, `--fail-fast`, `--max-errors`, `--data-failure-report`, `--error-report`), documented in README.md
- **Type System** - Each field has representations (regex patterns) that map to logical types with optional arguments

### Configuration Format

Fields are defined in YAML with:

- `name`: Field identifier
- `logical_type`: Type definition (name + optional args for decimals/enums)
- `representations`: Array of regex patterns, tried in order (first match wins). Named capture groups and static `args` supply the type's value args; static args win on overlap. See README.md for the full reference.

The pipeline processes data concurrently using goroutines and channels. The reader checks the header synchronously before the goroutines start. Output (and the optional data failure report) is written to temporary files that are only renamed into place when the run succeeds.

## Adding New Logical Types

To add a new logical type to the system, you need to update several components:

### 1. Config System (`internal/config/config.go`)
- Add new type constant to `LogicalType` enum
- Create corresponding `*TypeConfig` struct implementing `LogicalTypeConfig` interface
- Add a case to the switch in `FieldConfig.UnmarshalTypeConfigs()`

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

<!-- BACKLOG.MD GUIDELINES START -->
<!-- backlog.md-instructions-version: 1.53.0 -->
<CRITICAL_INSTRUCTION>

## Backlog.md Workflow

This project uses Backlog.md for task and project management.

**At the beginning of each conversation in this project, run `backlog instructions overview` before answering or taking action. Re-read it only if you have not read it yet in the current conversation.**

Use the overview to decide whether to search, read, create, or update Backlog tasks.

Before task lifecycle actions, read the matching detailed guide:
- `backlog instructions task-creation` before creating or splitting tasks
- `backlog instructions task-execution` before planning, changing status or assignee, adding a plan or implementation notes, or implementing task work
- `backlog instructions task-finalization` before checking acceptance criteria, writing final summaries, or moving tasks to terminal statuses

Use `backlog <command> --help` before running unfamiliar commands. Help shows options, fields, and examples.

Do not edit Backlog task, draft, document, decision, or milestone markdown files directly. Use the `backlog` CLI so metadata, relationships, and history stay consistent.

</CRITICAL_INSTRUCTION>
<!-- BACKLOG.MD GUIDELINES END -->
