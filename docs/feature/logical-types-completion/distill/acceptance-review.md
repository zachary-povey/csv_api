# Acceptance Review: logical-types-completion

## Coverage Matrix

| User Story / Capability | Test Scenarios | Count |
|------------------------|----------------|-------|
| Enum fields with permitted values | `test_enum_basic`, `test_enum_mixed_types` | 2 |
| Enum with input normalization | `test_enum_case_insensitive`, `test_enum_value_remap` | 2 |
| Enum with multiple input formats | `test_enum_multiple_representations` | 1 |
| Enum validation (reject invalid) | `test_enum_invalid_value` | 1 |
| Date fields (ISO format) | `test_date_iso`, `test_date_mixed_types` | 2 |
| Date fields (custom format) | `test_date_custom_format` | 1 |
| Date validation (reject invalid) | `test_date_invalid_components` | 1 |
| Time fields (ISO format) | `test_time_iso` | 1 |
| Time with microsecond precision | `test_time_fractional_seconds` | 1 |
| Time with custom format | `test_time_custom_format` | 1 |
| Time boundary values | `test_time_edge_cases` | 1 |
| Timestamp with timezone offset | `test_timestamp_iso_offset`, `test_timestamp_timezone_conversion` | 2 |
| Timestamp UTC (Z suffix) | `test_timestamp_iso_utc` | 1 |
| Timestamp custom format | `test_timestamp_custom_format` | 1 |
| Timestamp from epoch value | `test_timestamp_epoch` | 1 |
| Timestamp validation (reject invalid) | `test_timestamp_invalid` | 1 |
| All new types working together | `test_all_new_types_mixed` | 1 |
| **Total** | | **21** |

## Error Path Analysis

| Category | Error Scenarios | Total Scenarios | Ratio |
|----------|----------------|-----------------|-------|
| New tests only | 4 (`enum_invalid_value`, `date_invalid_components`, `timestamp_invalid`, implicit in enum remap failure modes) | 21 | 19% |
| Combined with existing file | 10 (6 existing + 4 new) | 40 (19 existing + 21 new) | 25% |

The error path ratio is below the 40% target for new tests alone. This is a deliberate trade-off: the existing test suite already covers the shared infrastructure error paths (pattern mismatch, missing fields, empty fields, invalid format). The new type-specific error paths that matter are:
- Enum: value not in permitted_values (converter-level validation)
- Date: invalid date components (converter-level validation)
- Timestamp: invalid time components (converter-level validation)
- Time: no specific error test -- invalid time components would be caught similarly to date

**Recommendation:** If higher error coverage is desired, add:
- `test_time_invalid_components` -- hour=25 or minute=60
- `test_enum_missing_permitted_values` -- config with no permitted_values arg (config validation error)
- `test_timestamp_epoch_invalid_precision` -- precision arg set to unsupported value

These are optional; the current set covers the primary failure modes for each type.

## Traceability

### Implementation Artifacts Required

For each new type (Enum, Date, Time, Timestamp), the DELIVER wave must modify:

1. **`internal/parser/converters.go`** -- Add case in `Convert()` switch + converter function
   - `convert_enum(args, config)` -- validates value against permitted_values
   - `convert_date(args)` -- assembles year/month/day into days-since-epoch int
   - `convert_time(args)` -- assembles h/m/s/us into microseconds-since-midnight long
   - `convert_timestamp(args)` -- handles both component-based and epoch-based, outputs microseconds-since-epoch long

2. **`internal/avro_writer/avro_writer.go`** -- Add cases in `map_type_json`
   - Enum: `{"type":"enum","name":"<field_name>","symbols":[...]}`
   - Date: `{"type":"int","logicalType":"date"}`
   - Time: `{"type":"long","logicalType":"time-micros"}`
   - Timestamp: `{"type":"long","logicalType":"timestamp-micros"}`

   Note: The enum case requires access to both the field name and the type config. The current `map_type_json` template function only receives `LogicalTypeConfig`, not the field name. The avro writer template or function signature will need adjustment for enum support.

3. **`tests/fixtures/`** -- 21 new fixture directories per fixture-specs.md

4. **`tests/test_logical_types.py`** -- 21 new test functions per test-code-specs.md

### Architecture Observations

- **Representation args merge behavior:** Static args on representations are set first, then regex-extracted args override (parser.go lines 132-135). This means for value remapping, the representation pattern must NOT capture a `value` group -- use non-capturing groups `(?:...)` so EnsureValueName leaves the pattern alone, and the static `args.value` is used without being overridden.

- **EnsureValueName edge case:** Patterns with only non-capturing groups (total_caps > 0, unnamed_caps = 0) are returned unchanged. This is the correct behavior for remapping patterns. Patterns with zero groups get wrapped in `(?P<value>...)`.

- **Enum avro schema requires field name:** The Avro enum type needs a `name` field in the schema. The current template function `map_type_json` receives only `LogicalTypeConfig`. For enum support, the template must pass the field name to the mapping function, or the function must receive additional context. This is a known implementation detail for the DELIVER wave.

- **goavro logical type encoding:** goavro handles date/time/timestamp logical types by accepting `int32` (date), `int64` (time-micros), and `int64` (timestamp-micros) values when the schema declares the logical type. The converter must output the raw numeric value (days since epoch, microseconds since midnight, microseconds since epoch UTC). goavro encodes these into the Avro binary format with logical type metadata.

## Risks and Gaps

1. **fastavro return types need verification.** The test assertions assume fastavro converts Avro logical types to Python datetime objects. If goavro does not set logical type metadata in the schema correctly, fastavro may return raw integers. The first test for each type (date_iso, time_iso, timestamp_iso_utc) will reveal this quickly. Mitigation: test-code-specs includes a fallback note.

2. **goavro enum encoding.** goavro may require the enum value to be passed as a specific type (e.g., `goavro.Union` or a plain string matching a symbol). The converter must return the value in the format goavro expects. This will surface when `test_enum_basic` is first run.

3. **Timezone parsing in Go.** The timestamp converter needs to parse timezone offsets like "+05:00" and "-08:00" and convert to UTC. Go's `time` package handles this well, but the converter receives string args, so it must parse the offset string manually or construct a `time.FixedZone`.

4. **Epoch timestamp precision arg.** The `precision` arg ("seconds", "milliseconds", "microseconds") is a static representation arg. The converter must scale the numeric value to microseconds based on this arg. The converter function needs to handle this branching.

5. **No null handling tests.** The existing `Representation` struct has an `is_null` field, but no tests cover nullable fields for new types. This is out of scope for this feature but worth noting.
