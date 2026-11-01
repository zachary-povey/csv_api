# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run Commands

- **Build**: `./scripts/build.sh` (creates binary at `./build/csv-api`)
- **Run**: `./build/csv-api parse --config-path <config> --input-path <csv> --output-path <output>` (short flags: `-c`, `-i`, `-o`); add `--stream` and leave out `-o` to stream the output (Avro or Parquet) to stdout
- **Validate config**: `./build/csv-api validate_config --config-path <config>` (`--print-merged` also prints the merged config as YAML to stdout)
- **Test**: `python3 -m pytest tests` (builds the binary first; set `BUILD_PATH` to test an existing binary; needs `pip install -r tests/requirements.txt`)

## Architecture Overview

This is a CSV validation and Avro/Parquet conversion tool with a concurrent pipeline architecture:

1. **Reader** (`internal/reader`) - Reads CSV files and validates headers against config
2. **Parser** (`internal/parser`) - Validates data using regex patterns and converts types. Rows are dispatched in batches to one worker per GOMAXPROCS (each with its own compiled representations, since a Parser is single-goroutine); `ParseData` reassembles results in input order and does all side effects (error tracking, dropping rows, data failure report) in that single in-order stage
3. **Avro Writer** (`internal/avro_writer`) or **Parquet Writer** (`internal/parquet_writer`) - Generates the output schema and writes output. Both have a `Write(io.Writer, config, channel, wg, errTracker, ...)` function (the Avro one also takes `flushWhenIdle`, set by `--stream`); `main.go` picks one from `--output-format` (default `avro`). Parquet uses `github.com/parquet-go/parquet-go`, keeps config field order, and writes each row group (`rowsPerRowGroup`, 100k rows; a package var only so tests can shrink it) to the output as soon as it is complete. Its footer is written at the end, and not at all when the run has failed (`ErrorTracker.Failed()` once the channel closes), so a failed `--stream` run leaves output Parquet readers reject

The main processing flow uses Go channels for concurrent execution:

```
CSV → Reader Channel → Parser Channel → Avro/Parquet Writer → Avro/Parquet File
```

### Key Components

- **Config System** (`internal/config`) - YAML-based field definitions with logical types (string, integer, decimal, enum, timestamp, date, time) and regex validation patterns
- **Error Tracking** (`internal/error_tracking`) - Centralized error collection with file/row/cell level reporting. Closes `KillCh` to stop the pipeline (execution errors, fail-fast, max-errors); the reader stops on it and the parser keeps draining its input so nothing blocks
- **Options** (`internal/options`) - Output and error handling flags for `parse` (`--output-format` avro|parquet, `--stream`, `--error-on-data-failures`, `--fail-fast`, `--max-errors`, `--data-failure-report`, `--error-report` console|json, `--error-report-path`), documented in README.md
- **Type System** - Each field has representations (regex patterns) that map to logical types with optional arguments

### Configuration Format

A config has a required top-level `name` (a valid Avro name, used as the output record name), an optional top-level `allow_extra_fields` (default true: csv columns matching no field are ignored; false makes each such column a file-level error) and optional top-level `empty_is_null` / `quoted_empty_is_null` (defaults for every nullable field, see below). Fields are defined in YAML with:

- `name`: Field identifier
- `logical_type`: Type definition (name + optional args for decimals/enums)
- `nullable`: optional, default false; makes the Avro type a union with null
- `empty_is_null` / `quoted_empty_is_null`: optional `*bool`s, nullable fields only (an explicit setting on a non-nullable field fails at load); the field's value, else the top-level one, else true (`Config.EmptyCellIsNull`). An unquoted (`,,`) / quoted (`,"",`) empty cell is null when its flag is on, checked in `parseRow` before representations; with it off the empty cell goes through the representations. encoding/csv returns "" for both, so the reader classifies empty cells (`Record.QuotedEmpty`, nil when none) only for fields whose two flags differ: a non-last field from the `FieldPos` column gap, the last field from the raw byte before the record's line terminator (`tailReader` keeps the recently read bytes)
- `required`: optional, default true; `required: false` lets the column be missing from the file (the field is then null in every row); only allowed on nullable fields
- `header_patterns`: Optional array of (unanchored) regexes matched against csv headers to find the field's column; when set they replace the exact header match on `name`. A header matching more than one field, or a field matching more than one header, is a file-level error
- `representations`: Array of regex patterns, tried in order (first match wins). Named capture groups and static `args` supply the type's value args; static args win on overlap. A representation with `is_null: true` resolves to null (nullable fields only). See README.md for the full reference.
- `error_message`: optional plain-language message for the producer, appended as `(note: ...)` after the technical reason of the field's data failures (console report and `__csv_api_error__` column) and of its missing-column error. Cell errors are kept as `error_tracking.CellError` structs (row, column, field, value, reason, message).

Layered configs (`internal/config/layered.go`): a config may start with `extends: <path>` (relative to that file; chains allowed, cycles and missing files fail naming the files). The root (no `extends`) defines the schema. Extending files are strictly decoded into `overrideConfig`, which only holds input-side settings (`allow_extra_fields`, `empty_is_null`, `quoted_empty_is_null`, and per field matched by `name`: `representations` / `prepend_representations` / `append_representations`, `header_patterns`, `error_message`, `required`, `empty_is_null`, `quoted_empty_is_null`); schema-side keys (`name`, `nullable`, `logical_type`) and unknown fields fail naming the file. Files merge root to leaf, then `Config.check()` runs every load-time check on the merged config. A new input-side field setting must be added to `fieldOverride` and `applyOverride` (and the README table) to be overridable. `validate_config --print-merged` prints the merged `Config` as YAML (keep `omitempty` tags so it stays a clean root config).

The pipeline processes data concurrently using goroutines and channels. The reader checks the header synchronously before the goroutines start. Output (and the optional data failure report) is written to temporary files that are only renamed into place when the run succeeds. With `--stream` the output (Avro or Parquet) instead goes straight to stdout (no `--output-path`) or to the output path as rows are validated, and is incomplete on failure (`openOutput` in `main.go`); side reports stay atomic. In `parse`, stdout carries only output data: everything else goes to stderr.

## Adding New Logical Types

To add a new logical type to the system, you need to update several components:

### 1. Config System (`internal/config/config.go`)
- Add new type constant to `LogicalType` enum
- Create corresponding `*TypeConfig` struct implementing `LogicalTypeConfig` interface
- Add a case to the switch in `FieldConfig.UnmarshalTypeConfigs()`
- Add the type's value arg sets to `valueArgSets` in `internal/config/representations.go` (and any static arg value checks to `checkStaticArg`), so representations are checked at config load

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

### 4. Parquet Writer (`internal/parquet_writer/parquet_writer.go`)
- Add a case to `columnNode()` returning the Parquet leaf node equivalent to the Avro type (physical + logical type)
- Add a case to `columnValue()` converting the parser's value to a `parquet.Value`
- Add the type to `tests/fixtures/parquet_all_types` and the expectations in `tests/test_parquet_output.py`

### 5. Testing
- Add comprehensive tests in `tests/test_logical_types.py` covering:
  - Valid conversions with different parameter sets
  - Mixed types in same dataset
  - Validation failures and error handling
  - Edge cases specific to the type

### Type Implementation Notes
- **Parameter Extraction**: Use type assertions with proper error handling for `args` map
- **Error Messages**: Include context about which parameter failed and why
- **Avro/Parquet Compatibility**: Ensure converted values match expected Avro schema types, and that both writers hold the same values
- **Docs**: Add the type to the README type table (Avro and Parquet columns)
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
