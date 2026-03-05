# Technical Requirements: Logical Types Completion

## Architecture Context

The tool has a three-layer architecture for each field:

1. **Representation layer** (input flexibility): Regex patterns match raw CSV strings. Named capture groups extract parameters. Static args on representations provide fixed values. The first matching representation wins. Regex-extracted args override static args on overlap.
2. **Converter layer** (output correctness): Receives the merged args dict and the logical type config. Validates args are present and valid. Converts to the output type. **Never sees the raw input string.**
3. **Avro writer layer** (output schema): Maps logical type configs to Avro schema fragments. Defines the output schema, not the input format.

This means: converters do NOT validate input format -- that is the representation layer's job. Converters validate their args are present, have correct types, and represent valid values (e.g., month is 1-12). The tool's value proposition is: define a strict output schema (Avro), but accept arbitrarily messy input formats via flexible regex representations.

## Scope

Add converter functions, Avro writer schema mappings, and integration tests for four logical types: Enum, Timestamp, Date, Time. Config structs already exist in `internal/config/config.go`. The converter dispatch in `internal/parser/converters.go` needs new cases. The Avro schema template in `internal/avro_writer/avro_writer.go` needs new type mappings.

## Functional Requirements

### FR-01: Enum Type Conversion

- Converter receives `args["value"]` (string) and `EnumTypeConfig.Args.PermittedValues` ([]string).
- If value is in permitted_values list, output the string value unchanged.
- If value is NOT in permitted_values, return an error identifying the invalid value and the permitted set.
- Comparison is case-sensitive (exact match on the extracted value, not the raw input).
- Input flexibility (case-insensitive matching, value remapping) is handled entirely by representations -- e.g., a case-insensitive regex `(?i)(?P<value>active|inactive)` or a remapping pattern `"live"` with `args: {value: "active"}`.

### FR-02: Date Type Conversion

- Converter receives `args["year"]`, `args["month"]`, `args["day"]` (all required, string or int).
- Validates components form a real calendar date: month 1-12, day valid for given month/year (including leap year rules).
- Output: int32 representing days since 1970-01-01 (Unix epoch).
- Dates before epoch (e.g., 1965-06-20) produce negative values.
- Input flexibility (ISO, European DD/MM/YYYY, US MM-DD-YYYY, compact YYYYMMDD) is handled entirely by representations that extract year/month/day named groups.

### FR-03: Time Type Conversion

- Converter receives optional `args["hour"]` (default 0), `args["minute"]` (default 0), `args["second"]` (default 0), `args["millisecond"]` (default 0), `args["microsecond"]` (default 0). All string or int.
- If a `args["fraction"]` group is present instead of separate millisecond/microsecond, parse it as fractional seconds: pad or truncate to 6 digits, split into millisecond (first 3) and microsecond (last 3).
- Validates components: hour 0-23, minute 0-59, second 0-59, millisecond 0-999, microsecond 0-999.
- Output: int64 representing microseconds since midnight.
- Calculation: `(hour*3600 + minute*60 + second) * 1_000_000 + millisecond*1000 + microsecond`
- Input flexibility (24h, 12h with AM/PM, fractional seconds of varying precision) is handled entirely by representations.

### FR-04: Timestamp Type Conversion

Two parameter sets, distinguished by which args are present (same dispatch pattern as decimal's value vs integer_part/decimal_part):

- **Parameter set 1 (epoch-based)**: `args["value"]` (string or int, epoch value), optional `args["offset"]` (string or int, default 0), `args["precision"]` (string or int: "seconds", "milliseconds", or "microseconds"). Output: int64 microseconds since Unix epoch, adjusted by offset.
- **Parameter set 2 (component-based)**: `args["year"]`, `args["month"]`, `args["day"]` (required), optional `args["hour"]`, `args["minute"]`, `args["second"]`, `args["millisecond"]`, `args["microsecond"]`, `args["timezone"]` (all string or int except timezone which is string). Output: int64 microseconds since Unix epoch (UTC).
- Date component validation: same rules as FR-02 (month 1-12, day valid for month/year).
- Time component validation: same rules as FR-03 (hour 0-23, minute 0-59, second 0-59).
- Timezone handling: if timezone provided, interpret components in that timezone and convert to UTC; if absent, interpret as UTC.
- Input flexibility (ISO with offset, epoch strings, custom formats) is handled entirely by representations. A representation for epoch seconds would be `pattern: "^(?P<value>\d+)$"` with `args: {precision: "seconds"}`.

### FR-05: Avro Schema Generation

Update `map_type_json` in avro_writer to handle new type configs:

- Enum: `{"type":"enum","name":"<field_name>","symbols":["val1","val2",...]}`
- Date: `{"type":"int","logicalType":"date"}`
- Time: `{"type":"long","logicalType":"time-micros"}`
- Timestamp: `{"type":"long","logicalType":"timestamp-micros"}`

### FR-06: Error Reporting

- All conversion errors propagate through the existing ErrorTracker.
- Error messages include: raw value, field name, resolved args dict, and specific problem description.
- This follows the existing pattern in `parser.go` line 145: `fmt.Sprintf("Failed to convert '%s' to type '%s'\nResolved args: %v\nException:\n %s", ...)`.
- Two distinct failure modes are already handled by the pipeline:
  - Representation failure (no regex matched): "value 'X' did not match any pattern in column 'Y'"
  - Converter failure (args invalid): "Failed to convert ... Resolved args: ... Exception: ..."

## Non-Functional Requirements

### NFR-01: Pipeline Compatibility

- New converters operate within the existing concurrent pipeline (Reader channel -> Parser channel -> Avro Writer).
- No changes to pipeline architecture, channel types, or goroutine structure.

### NFR-02: Config Pattern Consistency

- New types follow the existing config YAML pattern established by string, integer, and decimal.
- Config validation catches missing required args at config load time, before CSV processing.
- Args from regex named groups arrive as strings; static args from config may arrive as strings or ints. Converters must handle both (same pattern as existing convert_decimal).

### NFR-03: Test Coverage

- Each type has integration test fixtures following the existing pattern in `tests/fixtures/` (config.yaml + data.csv per fixture).
- Tests in `tests/test_logical_types.py` following the existing pattern.
- Coverage must include: valid conversions, representation-based input flexibility, validation failures, and type-specific edge cases.

## Constraints

- Go standard library for date/time parsing (time package).
- goavro/v2 library for Avro output (already in use).
- Args from regex named groups arrive as strings; hardcoded args from config may arrive as strings or ints. Converters must handle both.

## Dependencies

- Config structs (EnumTypeConfig, TimestampTypeConfig, DateTypeConfig, TimeTypeConfig) already exist in `internal/config/config.go`.
- Config parsing and validation already handles these types in `UnmarshalTypeConfigs()`.
- Parser dispatch in `Convert()` needs new cases added to the switch statement.
- Avro writer `map_type_json` needs new cases added to the switch statement.
- Only converter functions, Avro schema mappings, and tests need to be added.
