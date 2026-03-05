<!-- markdownlint-disable MD024 -->

# User Stories: Logical Types Completion

## US-01: Enum Field Validation with Flexible Input Mapping

### Problem

Ravi Krishnan is a data engineer who ingests CSV exports from partner systems containing categorical fields like transaction status and country codes. He finds it painful to maintain per-partner Python scripts that normalize inconsistent labels ("active", "Active", "live", "enabled") before feeding them to csv_api. When a partner changes their label format, the script breaks and invalid values silently enter the data warehouse.

### Who

- Data engineer | Ingesting partner CSV exports with inconsistent categorical labels | Needs strict output vocabulary with flexible input acceptance

### Solution

Validate extracted enum values against a configured list of permitted values and output them as Avro enum type fields. Input flexibility (case-insensitive matching, value remapping from variant labels) is handled by representations with regex patterns and static args. The converter only validates the final extracted value against permitted_values.

### Domain Examples

#### 1: Happy Path -- Direct match with default representation

Ravi configures a "status" field with `permitted_values: [active, inactive, pending]` and a single representation `(?P<value>.+)`. His CSV contains "active", "inactive", "pending". All values match the regex, extract as-is, pass the permitted_values check, and the output Avro schema contains an enum type with those three symbols.

#### 2: Edge Case -- Case-insensitive input via regex representation

Ravi's partner sends "Active" and "INACTIVE" in the CSV. Ravi writes a case-insensitive representation: `pattern: "(?i)(?P<value>active|inactive|pending)"`. The regex matches and extracts "Active" (preserving original case from the match). However, "Active" is not in permitted_values ["active", "inactive", "pending"] -- the enum comparison is case-sensitive on the extracted value. Ravi adjusts his regex to force lowercase output or adds separate representations with static args.

#### 3: Edge Case -- Value remapping via static args

Ravi's partner uses "live" to mean "active" and "disabled" to mean "inactive". Ravi adds representations: `pattern: "live"` with `args: {value: "active"}` and `pattern: "disabled"` with `args: {value: "inactive"}`. When the CSV contains "live", the regex matches, the static arg provides `value: "active"`, and the converter validates "active" is in permitted_values. The raw input "live" never reaches the converter.

#### 4: Error -- Extracted value not in permitted set

Ravi's CSV row 5 has status "cancelled". The regex `(?P<value>.+)` matches and extracts "cancelled". The converter finds "cancelled" is not in permitted_values [active, inactive, pending]. Error: "value 'cancelled' is not in permitted values [active inactive pending]".

### UAT Scenarios (BDD)

#### Scenario: Valid enum values convert to Avro enum type

Given Ravi has a config with field "status" of type enum with permitted_values [active, inactive, pending]
And a representation with pattern `(?P<value>.+)`
And a CSV file with rows: "active", "inactive", "pending"
When Ravi runs the parse command
Then the output Avro file contains 3 records with status values "active", "inactive", "pending"
And the Avro schema defines status as enum type with symbols [active, inactive, pending]

#### Scenario: Value remapping via static args produces valid enum output

Given Ravi has a config with field "status" of type enum with permitted_values [active, inactive]
And representations: pattern "live" with args {value: "active"}, pattern "disabled" with args {value: "inactive"}
And a CSV file with rows: "live", "disabled"
When Ravi runs the parse command
Then the output Avro contains status values "active" and "inactive"

#### Scenario: Extracted value not in permitted set produces clear error

Given Ravi has a config with field "status" of type enum with permitted_values [active, inactive, pending]
And a CSV file where row 3 contains status "cancelled"
When Ravi runs the parse command
Then the tool exits with a non-zero status
And the error output mentions "cancelled" and the permitted values

#### Scenario: Enum config without permitted_values is rejected at config load

Given Ravi has a config with field "status" of type enum but no permitted_values arg
When the config is loaded
Then the config validation fails
And the error mentions "permitted_values" is required

#### Scenario: Enum field alongside other types

Given Ravi has a config with enum field "status", string field "name", and integer field "id"
And a CSV file with valid data for all fields
When Ravi runs the parse command
Then all fields convert correctly with their respective types

### Acceptance Criteria

- [ ] Extracted value in permitted_values list outputs as string in Avro enum schema
- [ ] Extracted value not in permitted_values produces error identifying the value and permitted set
- [ ] Enum comparison is case-sensitive on the extracted value (input flexibility via representations)
- [ ] Static args on representations can remap input values to canonical enum values
- [ ] Avro schema uses enum type with symbols matching permitted_values and name matching field name
- [ ] Works alongside other field types in the same config

### Technical Notes

- Config struct `EnumTypeConfig` with `PermittedValues []string` already exists
- Avro enum schema requires a `name` field -- use the field name from config
- goavro expects enum values as plain strings; the schema defines the constraint
- The `EnsureValueName` function in parser.go will auto-wrap patterns without capture groups in `(?P<value>...)`, so simple patterns like `"live"` will have value extracted automatically

### Dependencies

- Config parsing for enum type: already implemented
- Traces to: JS-01 (Categorical Data Standardization)

---

## US-02: Date Field Conversion from Multiple Input Formats

### Problem

Ravi Krishnan is a data engineer who produces Avro files consumed by Spark and Presto for analytics. He receives date columns in ISO ("2024-03-15"), European ("15/03/2024"), and US ("03-15-2024") formats from different partners. He finds it tedious to maintain per-format Python conversion scripts, and worries about off-by-one errors and DD/MM vs MM/DD confusion that would place records in wrong partitions.

### Who

- Data engineer | Producing partition-compatible Avro from multi-format date sources | Needs native date type from any input format

### Solution

Parse CSV date fields by extracting year/month/day components via regex named groups (one representation per accepted format) and convert to Avro date logical type (int32 days since 1970-01-01). The converter validates calendar correctness. New input format = new representation regex, no code changes.

### Domain Examples

#### 1: Happy Path -- ISO date format with default representation

Ravi's CSV has "transaction_date" with value "2024-03-15". The default ISO representation `^(?P<year>\d{4})-(?P<month>0[1-9]|1[0-2])-(?P<day>0[1-9]|[12]\d|3[01])$` captures year=2024, month=03, day=15. The converter validates this is a real date and outputs 19797 (days from 1970-01-01 to 2024-03-15).

#### 2: Edge Case -- European format via custom representation

Ravi adds a representation `^(?P<day>\d{2})/(?P<month>\d{2})/(?P<year>\d{4})$` for European dates. His CSV contains "15/03/2024". The regex extracts day=15, month=03, year=2024 -- the same args as the ISO example, just extracted from a different format. Same converter, same output.

#### 3: Error -- Invalid date components after extraction

Ravi's CSV row 4 has date "2024-02-30". The regex captures year=2024, month=02, day=30. The converter validates: February 2024 has 29 days (leap year). Day 30 is invalid. Error: "invalid date: day 30 exceeds maximum 29 for month 2 in year 2024".

### UAT Scenarios (BDD)

#### Scenario: Valid ISO dates convert to days since epoch

Given Ravi has a config with field "transaction_date" of type date
And the default ISO representation
And a CSV file with dates "2024-03-15" and "2024-01-01"
When Ravi runs the parse command
Then the output Avro contains integer values representing days since 1970-01-01
And the Avro schema defines transaction_date with logicalType "date" and base type "int"

#### Scenario: Custom format representation extracts correct components

Given Ravi has a config with field "transaction_date" of type date
And a representation `^(?P<day>\d{2})/(?P<month>\d{2})/(?P<year>\d{4})$` for European dates
And a CSV file with date "15/03/2024"
When Ravi runs the parse command
Then the output Avro contains the same days-since-epoch value as for ISO "2024-03-15"

#### Scenario: Pre-epoch dates produce negative values

Given Ravi has a config with field "birth_date" of type date
And a CSV file with date "1965-06-20"
When Ravi runs the parse command
Then the output Avro contains a negative integer for birth_date

#### Scenario: Invalid date components produce clear error

Given Ravi has a config with field "event_date" of type date
And a CSV file where row 2 contains date "2024-02-30"
When Ravi runs the parse command
Then the tool exits with a non-zero status
And the error output mentions the invalid date

#### Scenario: Leap year Feb 29 is accepted for leap years

Given Ravi has a config with field "event_date" of type date
And a CSV file with date "2024-02-29"
When Ravi runs the parse command
Then the output Avro contains the correct days-since-epoch value

#### Scenario: Non-leap year Feb 29 is rejected

Given Ravi has a config with field "event_date" of type date
And a CSV file with date "2023-02-29"
When Ravi runs the parse command
Then the tool exits with a non-zero status
And the error output mentions the invalid date

### Acceptance Criteria

- [ ] Year/month/day components (string or int) convert to int32 days since 1970-01-01
- [ ] Different input formats produce identical output when they represent the same date (via different representations)
- [ ] Pre-epoch dates produce negative values
- [ ] Invalid month (0, 13+) or day (0, exceeding month length) produces error with specific component detail
- [ ] Leap year validation: Feb 29 accepted for leap years, rejected for non-leap years
- [ ] Avro schema uses `{"type":"int","logicalType":"date"}`

### Technical Notes

- Use Go `time.Date(year, month, day, 0, 0, 0, 0, time.UTC)` for construction; verify the constructed date matches input components (Go normalizes invalid dates silently -- e.g., Feb 30 becomes Mar 2)
- Days calculation: `int(date.Sub(epoch).Hours() / 24)` or `date.Unix() / 86400`
- Args arrive as strings from regex; must parse to int

### Dependencies

- Config parsing for date type: already implemented
- Traces to: JS-02 (Multi-Format Date Normalization)

---

## US-03: Time Field Conversion with Fractional Second Precision

### Problem

Ravi Krishnan is a data engineer ingesting scheduling data (store hours, appointment times, shift schedules) from CSV. He receives times in 24-hour ("15:05:00"), with fractional seconds ("14:30:15.123456"), and occasionally unusual formats. He finds it wasteful to embed time-of-day values in full timestamps, which carry unnecessary date components and confuse downstream scheduling queries.

### Who

- Data engineer | Ingesting scheduling and time-of-day data | Needs time-micros type without date baggage, from varied formats

### Solution

Parse CSV time fields by extracting hour/minute/second/fraction components via regex named groups and convert to Avro time-micros logical type (int64 microseconds since midnight). Fractional seconds of any digit length are handled by padding/truncating to microsecond precision.

### Domain Examples

#### 1: Happy Path -- Standard 24-hour time

Ravi's CSV has "opening_time" with value "09:30:00". The default representation extracts hour=09, minute=30, second=00. The converter produces 34200000000 microseconds (9*3600*1000000 + 30*60*1000000).

#### 2: Edge Case -- Fractional seconds with varying precision

Ravi's CSV has "event_time" with value "14:30:15.123456". The representation captures fraction="123456". The converter interprets this as 123 milliseconds + 456 microseconds. For a value "14:30:15.123" (3 digits), the converter pads to "123000" = 123ms + 0us.

#### 3: Edge Case -- Custom format via representation

Ravi has a 12-hour time source: "3:05 PM". He writes a representation that captures hour=3, minute=05, with a static arg or regex group for the period. The representation or a preceding transformation converts to 24h components. The converter receives hour=15, minute=5 and produces the correct microsecond value.

#### 4: Error -- Out-of-range time component

Ravi's CSV row 3 has time "25:30:00". The regex extracts hour=25. The converter validates: hour must be 0-23. Error: "invalid hour: 25 (must be 0-23)".

### UAT Scenarios (BDD)

#### Scenario: Valid time converts to microseconds since midnight

Given Ravi has a config with field "opening_time" of type time
And a CSV file with times "09:30:00" and "17:00:00"
When Ravi runs the parse command
Then record 0 opening_time equals 34200000000 microseconds
And record 1 opening_time equals 61200000000 microseconds
And the Avro schema defines opening_time with logicalType "time-micros" and base type "long"

#### Scenario: Fractional seconds are preserved to microsecond precision

Given Ravi has a config with field "event_time" of type time
And a representation capturing a "fraction" group
And a CSV file with time "14:30:15.123456"
When Ravi runs the parse command
Then the output includes the 123456 microsecond fractional component

#### Scenario: Short fractional seconds are right-padded

Given Ravi has a config with field "event_time" of type time
And a CSV file with time "14:30:15.123"
When Ravi runs the parse command
Then the fractional component is treated as 123000 microseconds (padded to 6 digits)

#### Scenario: Midnight produces zero

Given Ravi has a config with field "closing_time" of type time
And a CSV file with time "00:00:00"
When Ravi runs the parse command
Then the output Avro contains value 0 for closing_time

#### Scenario: Maximum valid time (one second before midnight)

Given Ravi has a config with field "last_call" of type time
And a CSV file with time "23:59:59"
When Ravi runs the parse command
Then the output Avro contains value 86399000000 for last_call

#### Scenario: Invalid time components produce clear error

Given Ravi has a config with field "shift_start" of type time
And a CSV file where row 2 has time with hour component "25"
When Ravi runs the parse command
Then the tool exits with a non-zero status
And the error output mentions the invalid time component

### Acceptance Criteria

- [ ] Hour/minute/second/millisecond/microsecond components convert to int64 microseconds since midnight
- [ ] "fraction" capture group parsed as fractional seconds, padded/truncated to 6 digits for microsecond precision
- [ ] Missing optional components default to 0
- [ ] Invalid hour (24+), minute (60+), second (60+) produce errors with specific component detail
- [ ] Midnight (00:00:00) produces value 0; 23:59:59 produces 86399000000
- [ ] Avro schema uses `{"type":"long","logicalType":"time-micros"}`

### Technical Notes

- Calculation: `(hour*3600 + minute*60 + second) * 1_000_000 + millisecond*1000 + microsecond`
- "fraction" group handling: right-pad with zeros to 6 chars, take first 6 chars, split into ms (first 3) and us (last 3)
- Args arrive as strings from regex; must parse to int

### Dependencies

- Config parsing for time type: already implemented
- Traces to: JS-03 (Time-of-Day Normalization)

---

## US-04: Timestamp Field Conversion with Timezone Handling

### Problem

Ravi Krishnan is a data engineer building event pipelines from CSV exports. He receives timestamps as ISO strings with timezone offsets from one partner, epoch seconds from an API, and epoch milliseconds from a JavaScript frontend. He finds it error-prone and time-consuming to manually handle the format/precision/timezone variation in Python scripts, and getting it wrong means silent data corruption in time-sensitive financial calculations.

### Who

- Data engineer | Building event pipelines from multi-format CSV sources | Needs canonical microsecond-precision UTC timestamps from any input format

### Solution

Parse CSV timestamp fields via two parameter paths -- epoch-based (value + precision + optional offset) and component-based (year/month/day/hour/minute/second/timezone) -- and convert to Avro timestamp-micros (int64 microseconds since Unix epoch UTC). Representations extract the appropriate args from any format. Static args encode precision or other fixed parameters.

### Domain Examples

#### 1: Happy Path -- ISO timestamp with timezone offset (component-based)

Ravi's CSV has "event_timestamp" with value "2024-03-15T14:30:00+05:30". The ISO representation extracts year=2024, month=3, day=15, hour=14, minute=30, second=0, and a timezone offset component. The converter interprets these in the +05:30 timezone and outputs the UTC-equivalent microseconds (equivalent to 2024-03-15T09:00:00Z).

#### 2: Edge Case -- Epoch seconds with static precision arg

Ravi's API sends epoch seconds. His representation: `pattern: "^(?P<value>\d+)$"` with `args: {precision: "seconds"}`. CSV contains "1710512345". The regex extracts value=1710512345, the static arg provides precision=seconds. The converter multiplies by 1,000,000 to produce 1710512345000000 microseconds.

#### 3: Edge Case -- Component-based with defaults (date-only timestamp)

Ravi's CSV has "report_date" as timestamp but the value is just "2024-03-15". The representation extracts year=2024, month=3, day=15. Hour, minute, second default to 0. Timezone defaults to UTC. Output: microseconds for 2024-03-15T00:00:00Z.

#### 4: Error -- Invalid month in component-based timestamp

Ravi's CSV row 7 contains a timestamp with month "13". The regex extracts it. The converter validates: month must be 1-12. Error: "invalid month: 13 (must be 1-12)".

### UAT Scenarios (BDD)

#### Scenario: ISO timestamp with Z offset converts to UTC microseconds

Given Ravi has a config with field "event_timestamp" of type timestamp
And a representation for ISO format
And a CSV file with timestamp "2024-03-15T14:30:00Z"
When Ravi runs the parse command
Then the output Avro contains the correct int64 microsecond value for 2024-03-15 14:30:00 UTC
And the Avro schema defines event_timestamp with logicalType "timestamp-micros" and base type "long"

#### Scenario: Timestamp with positive timezone offset converts to UTC

Given Ravi has a config with field "event_timestamp" of type timestamp
And a CSV file with timestamp "2024-03-15T14:30:00+05:30"
When Ravi runs the parse command
Then the output equals microseconds for 2024-03-15T09:00:00Z

#### Scenario: Epoch seconds with static precision arg converts correctly

Given Ravi has a config with a representation capturing "value" and static args {precision: "seconds"}
And a CSV file with value "1710512345"
When Ravi runs the parse command
Then the output Avro contains value 1710512345000000

#### Scenario: Epoch milliseconds converts correctly

Given Ravi has a config with a representation capturing "value" and static args {precision: "milliseconds"}
And a CSV file with value "1710512345000"
When Ravi runs the parse command
Then the output Avro contains value 1710512345000000

#### Scenario: Component-based timestamp with defaults for missing time parts

Given Ravi has a config capturing only year, month, day for a timestamp field
And a CSV file with value "2024-03-15"
When Ravi runs the parse command
Then the output equals microseconds for 2024-03-15T00:00:00Z

#### Scenario: Timestamp with fractional seconds preserves microsecond precision

Given Ravi has a config with field "event_timestamp" of type timestamp
And a CSV file with timestamp "2024-03-15T14:30:00.123456Z"
When Ravi runs the parse command
Then the output Avro preserves the 123456 microsecond fractional component

#### Scenario: Invalid timestamp components produce clear error

Given Ravi has a config with field "event_timestamp" of type timestamp
And a CSV file where row 3 has a timestamp with month "13"
When Ravi runs the parse command
Then the tool exits with a non-zero status
And the error output mentions the invalid timestamp component

### Acceptance Criteria

- [ ] Component-based: year/month/day required, hour/minute/second/ms/us default to 0, timezone defaults to UTC
- [ ] Epoch-based: value with precision (seconds/milliseconds/microseconds) and optional offset produces correct microseconds
- [ ] Timezone offsets correctly applied to produce UTC output
- [ ] Invalid date/time components produce errors (same validation as date and time converters)
- [ ] Fractional seconds preserved to microsecond precision
- [ ] Avro schema uses `{"type":"long","logicalType":"timestamp-micros"}`
- [ ] Both parameter sets produce identical output for equivalent inputs

### Technical Notes

- Go `time.Date()` with `time.UTC` or parsed timezone location for component-based
- Epoch conversion: manual multiplication based on precision value
- Parameter set dispatch: check for presence of "value" arg (same pattern as decimal's value vs integer_part/decimal_part)
- Shares date/time validation logic with US-02 and US-03 -- consider extracting common validation functions

### Dependencies

- Config parsing for timestamp type: already implemented
- Benefits from date (US-02) and time (US-03) validation logic already being implemented
- Traces to: JS-04 (Timestamp Conversion from Diverse Sources)
