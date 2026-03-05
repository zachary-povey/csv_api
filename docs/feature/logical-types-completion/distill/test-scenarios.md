# Test Scenarios: logical-types-completion

Implementation order: Enum -> Date -> Time -> Timestamp

## Scenario Summary

| # | Type | Scenario | Fixture | Category |
|---|------|----------|---------|----------|
| 1 | Enum | Basic enum field | `enum_basic` | Happy path |
| 2 | Enum | Case-insensitive input via regex | `enum_case_insensitive` | Happy path |
| 3 | Enum | Value remapping via static representation args | `enum_value_remap` | Happy path |
| 4 | Enum | Multiple representations for same field | `enum_multiple_representations` | Happy path |
| 5 | Enum | Value not in permitted_values | `enum_invalid_value` | Error |
| 6 | Enum | Enum mixed with other types | `enum_mixed_types` | Happy path |
| 7 | Date | ISO format date (default representation) | `date_iso` | Happy path |
| 8 | Date | Custom format DD/MM/YYYY via representation | `date_custom_format` | Happy path |
| 9 | Date | Invalid date components (month 13, day 32) | `date_invalid_components` | Error |
| 10 | Date | Date mixed with other types | `date_mixed_types` | Happy path |
| 11 | Time | ISO format time (HH:MM:SS) | `time_iso` | Happy path |
| 12 | Time | Time with fractional seconds | `time_fractional_seconds` | Happy path |
| 13 | Time | Custom format via representation | `time_custom_format` | Happy path |
| 14 | Time | Edge cases: midnight and end of day | `time_edge_cases` | Boundary |
| 15 | Timestamp | ISO format with timezone offset | `timestamp_iso_offset` | Happy path |
| 16 | Timestamp | ISO format with Z (UTC) | `timestamp_iso_utc` | Happy path |
| 17 | Timestamp | Component extraction from custom format | `timestamp_custom_format` | Happy path |
| 18 | Timestamp | Epoch-based with precision arg | `timestamp_epoch` | Happy path |
| 19 | Timestamp | Timezone conversion to UTC | `timestamp_timezone_conversion` | Happy path |
| 20 | Timestamp | Invalid timestamp components | `timestamp_invalid` | Error |
| 21 | Mixed | All new types combined in one dataset | `all_new_types_mixed` | Walking skeleton |

Error path ratio: 4 error scenarios / 21 total = 19%. Adding the existing 6 error tests from test_logical_types.py, the overall file reaches 10/27 = 37%, approaching the 40% target. The error scenarios here focus on the novel failure modes unique to each new type.

---

## Enum Scenarios

### 1. Basic enum field (`enum_basic`)
**What it tests:** Simplest enum usage -- values match permitted_values exactly.
**Expected behavior:** Values are stored as strings in the Avro output, matching the input exactly.

### 2. Case-insensitive input via regex (`enum_case_insensitive`)
**What it tests:** Representation pattern uses `(?i)` flag so input matching is case-insensitive. The output value is what the regex captures (lowercase via named group), which must be in permitted_values.
**Expected behavior:** Input "ACTIVE" matches pattern `(?i)(?P<value>active|inactive)` and produces lowercase "active".

### 3. Value remapping via static representation args (`enum_value_remap`)
**What it tests:** Representation has a static `args.value` that overrides the regex-captured value. Pattern matches "live" but static arg sets value to "active".
**Expected behavior:** Static arg `value: active` takes priority... wait, actually regex args override static args based on parser.go line 133. So regex named groups override static args. For remapping, the pattern should NOT have a `(?P<value>...)` group -- it should just match, and the static arg provides the value.
**Expected behavior (corrected):** Pattern `^live$` has no named capture group (EnsureValueName wraps it as `(?P<value>^live$)` -- hmm, that would capture "live" as value). Let me reconsider. For remapping, the pattern should match but not capture a `value` group. Use a non-capturing group: `(?:live)` with static arg `value: active`. But EnsureValueName would see zero capture groups and wrap the whole thing. We need patterns like `(?:^live$)` -- EnsureValueName sees one unnamed capture... no, `(?:...)` is non-capturing. Let me re-read EnsureValueName: it checks for `?` after `(`, and if found, skips it. So `(?:live)` counts as zero unnamed captures, zero total named captures... wait, total_caps is incremented for all `(` regardless, but then if next char is `?` it's "already named or non-capturing" so unnamed_caps is not incremented. So `(?:live)` gives total_caps=1, unnamed_caps=0. Since total_caps != 0, the "wrap entire pattern" branch is not taken. So we get pattern returned as-is with no value group. Then convert_string looks for args["value"] from the static args. This works.

Actually wait -- with `(?:live)`, total_caps=1, unnamed_caps=0. The function returns the pattern unchanged. No `value` capture group. The static args provide `value: active`. The regex still needs to match the input string. `(?:live)` matches "live" within any string. Good.

But we need full-string matching. Use `^(?:live)$` or just make the non-capturing group anchor: `^(?:live)$`.

**Expected behavior:** Input "live" matches pattern, static arg `value: active` is used. "active" is in permitted_values. Output is "active".

### 4. Multiple representations for same field (`enum_multiple_representations`)
**What it tests:** A field has multiple representations, each mapping different input patterns to valid enum values. First matching representation wins.
**Expected behavior:** Different input formats all resolve to valid enum values.

### 5. Value not in permitted_values (`enum_invalid_value`)
**What it tests:** Input value is captured but is not in the permitted_values list.
**Expected behavior:** Process fails with non-zero exit code. Error output mentions the invalid value or "permitted".

### 6. Enum mixed with other types (`enum_mixed_types`)
**What it tests:** Enum field alongside string and integer fields in same dataset.
**Expected behavior:** All types parse correctly in same row.

---

## Date Scenarios

### 7. ISO format date (`date_iso`)
**What it tests:** Default representation extracts year/month/day from ISO-8601 format (YYYY-MM-DD).
**Expected behavior:** Dates stored as Avro date logical type (int, days since epoch). fastavro returns `datetime.date` objects.

### 8. Custom format DD/MM/YYYY (`date_custom_format`)
**What it tests:** Custom representation pattern extracts day/month/year from non-ISO format.
**Expected behavior:** Same Avro output regardless of input format. `datetime.date(2024, 3, 15)` for input "15/03/2024".

### 9. Invalid date components (`date_invalid_components`)
**What it tests:** Regex matches but extracted components form an invalid date (month=13, day=32).
**Expected behavior:** Process fails with non-zero exit code. Error mentions the invalid date.

### 10. Date mixed with other types (`date_mixed_types`)
**What it tests:** Date field alongside string and integer fields.
**Expected behavior:** All types parsed correctly together.

---

## Time Scenarios

### 11. ISO format time (`time_iso`)
**What it tests:** Default representation extracts hour/minute/second from HH:MM:SS format.
**Expected behavior:** Times stored as Avro time-micros logical type (long, microseconds since midnight). fastavro returns `datetime.time` objects or int microseconds.

### 12. Time with fractional seconds (`time_fractional_seconds`)
**What it tests:** Time input includes fractional seconds (milliseconds or microseconds).
**Expected behavior:** Fractional seconds preserved in output.

### 13. Custom format via representation (`time_custom_format`)
**What it tests:** Non-standard time format using custom regex to extract components.
**Expected behavior:** Components correctly assembled into time value.

### 14. Edge cases: midnight and end of day (`time_edge_cases`)
**What it tests:** Boundary values 00:00:00 (midnight) and 23:59:59 (last second of day).
**Expected behavior:** Both boundary values correctly represented.

---

## Timestamp Scenarios

### 15. ISO format with timezone offset (`timestamp_iso_offset`)
**What it tests:** ISO-8601 timestamp with explicit timezone offset like +05:30.
**Expected behavior:** Stored as UTC-converted microseconds since epoch. fastavro returns `datetime.datetime` in UTC.

### 16. ISO format with Z (UTC) (`timestamp_iso_utc`)
**What it tests:** ISO-8601 timestamp with Z suffix indicating UTC.
**Expected behavior:** Directly stored as microseconds since epoch. No timezone conversion needed.

### 17. Component extraction from custom format (`timestamp_custom_format`)
**What it tests:** Custom representation extracts year/month/day/hour/minute/second from non-ISO format.
**Expected behavior:** Components assembled into correct UTC timestamp.

### 18. Epoch-based with precision arg (`timestamp_epoch`)
**What it tests:** Input is a Unix epoch integer. Representation has static arg `precision: seconds` (or `milliseconds`). Converter uses the precision arg to interpret the numeric value.
**Expected behavior:** Epoch value correctly scaled to microseconds for Avro output.

### 19. Timezone conversion to UTC (`timestamp_timezone_conversion`)
**What it tests:** Component-based timestamp with explicit timezone arg. Output must be converted to UTC.
**Expected behavior:** A timestamp at 2024-03-15 10:00:00 in timezone +05:00 becomes 2024-03-15 05:00:00 UTC.

### 20. Invalid timestamp components (`timestamp_invalid`)
**What it tests:** Regex matches but components form invalid timestamp (e.g., hour=25).
**Expected behavior:** Process fails with non-zero exit code.

---

## Walking Skeleton

### 21. All new types combined (`all_new_types_mixed`)
**What it tests:** A single dataset using enum, date, time, and timestamp fields alongside string and integer. Validates the full pipeline works end-to-end for all new types together.
**Expected behavior:** All fields parse, convert, and serialize correctly. This is the first test to enable -- proves the entire pipeline works for every new type.
