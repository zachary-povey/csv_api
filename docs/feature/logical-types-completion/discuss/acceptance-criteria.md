# Acceptance Criteria: Logical Types Completion

All scenarios use concrete test data. Each type section covers: valid conversion, representation-driven input flexibility, error handling, Avro schema correctness, and type-specific edge cases.

Architecture reminder: representations (regex + static args) handle input flexibility. Converters receive extracted args and handle output correctness. Converters never see the raw CSV string.

---

## Enum Type

### Scenario: Valid enum values pass through as strings

```gherkin
Given a config with field "status" of type enum
  And permitted_values are ["active", "inactive", "pending"]
  And a representation with pattern "(?P<value>.+)"
  And a CSV with rows:
    | status   |
    | active   |
    | inactive |
    | pending  |
When the parse command runs
Then the output Avro contains 3 records
  And record 0 has status "active"
  And record 1 has status "inactive"
  And record 2 has status "pending"
```

### Scenario: Case-insensitive input accepted via regex representation

```gherkin
Given a config with field "status" of type enum
  And permitted_values are ["active", "inactive"]
  And representations:
    | pattern                              | args          |
    | (?i)active                           | {value: "active"}   |
    | (?i)inactive                         | {value: "inactive"} |
  And a CSV with rows:
    | status   |
    | Active   |
    | INACTIVE |
When the parse command runs
Then the output Avro contains 2 records
  And record 0 has status "active"
  And record 1 has status "inactive"
```

### Scenario: Value remapping via static args on representations

```gherkin
Given a config with field "status" of type enum
  And permitted_values are ["active", "inactive"]
  And representations:
    | pattern  | args                  |
    | live     | {value: "active"}     |
    | disabled | {value: "inactive"}   |
  And a CSV with rows:
    | status   |
    | live     |
    | disabled |
When the parse command runs
Then the output Avro contains 2 records
  And record 0 has status "active"
  And record 1 has status "inactive"
```

### Scenario: Extracted value not in permitted set produces error

```gherkin
Given a config with field "status" of type enum
  And permitted_values are ["active", "inactive", "pending"]
  And a representation with pattern "(?P<value>.+)"
  And a CSV with rows:
    | status    |
    | active    |
    | cancelled |
When the parse command runs
Then the command exits with non-zero status
  And the error output contains "cancelled"
  And the error output references the permitted values
```

### Scenario: Avro schema contains enum type with symbols

```gherkin
Given a config with field "priority" of type enum
  And permitted_values are ["low", "medium", "high"]
  And a CSV with valid enum values
When the parse command runs
Then the Avro schema defines "priority" as type "enum"
  And the schema symbols are ["low", "medium", "high"]
  And the schema name for the enum is "priority"
```

### Scenario: Enum config without permitted_values is rejected

```gherkin
Given a config with field "status" of type enum
  And no permitted_values arg is specified
When the config is loaded
Then the config validation fails
  And the error mentions "permitted_values" is required
```

---

## Date Type

### Scenario: ISO date converts to days since epoch

```gherkin
Given a config with field "transaction_date" of type date
  And the default ISO representation ^(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})$
  And a CSV with rows:
    | transaction_date |
    | 2024-03-15       |
    | 2024-01-01       |
When the parse command runs
Then the output Avro contains integer values for days since 1970-01-01
  And the Avro schema defines transaction_date with logicalType "date"
  And the schema base type is "int"
```

### Scenario: European format via custom representation produces same output as ISO

```gherkin
Given a config with field "transaction_date" of type date
  And a representation ^(?P<day>\d{2})/(?P<month>\d{2})/(?P<year>\d{4})$
  And a CSV with rows:
    | transaction_date |
    | 15/03/2024       |
When the parse command runs
Then the output Avro contains the same days-since-epoch value as for ISO "2024-03-15"
```

### Scenario: Pre-epoch date produces negative value

```gherkin
Given a config with field "birth_date" of type date
  And a CSV with rows:
    | birth_date |
    | 1965-06-20 |
When the parse command runs
Then the output Avro contains a negative integer for birth_date
```

### Scenario: Leap year Feb 29 is accepted

```gherkin
Given a config with field "event_date" of type date
  And a CSV with rows:
    | event_date |
    | 2024-02-29 |
When the parse command runs
Then the command exits with zero status
  And the output Avro contains the correct days value for 2024-02-29
```

### Scenario: Non-leap year Feb 29 is rejected

```gherkin
Given a config with field "event_date" of type date
  And a CSV with rows:
    | event_date |
    | 2023-02-29 |
When the parse command runs
Then the command exits with non-zero status
  And the error output mentions the invalid date
```

### Scenario: Invalid month is rejected

```gherkin
Given a config with field "event_date" of type date
  And a CSV where the regex captures month "13" and day "1" and year "2024"
When the parse command runs
Then the command exits with non-zero status
  And the error output mentions the invalid month
```

### Scenario: Invalid day for month is rejected

```gherkin
Given a config with field "event_date" of type date
  And a CSV with rows:
    | event_date |
    | 2024-04-31 |
When the parse command runs
Then the command exits with non-zero status
  And the error output mentions the invalid date
```

### Scenario: Unix epoch date produces zero

```gherkin
Given a config with field "event_date" of type date
  And a CSV with rows:
    | event_date |
    | 1970-01-01 |
When the parse command runs
Then the output Avro contains value 0 for event_date
```

---

## Time Type

### Scenario: Standard time converts to microseconds since midnight

```gherkin
Given a config with field "opening_time" of type time
  And the default representation ^(?P<hour>\d{2}):(?P<minute>\d{2}):(?P<second>\d{2})$
  And a CSV with rows:
    | opening_time |
    | 09:30:00     |
    | 17:00:00     |
When the parse command runs
Then record 0 opening_time equals 34200000000 microseconds
  And record 1 opening_time equals 61200000000 microseconds
  And the Avro schema defines opening_time with logicalType "time-micros"
  And the schema base type is "long"
```

### Scenario: Fractional seconds with 6-digit precision

```gherkin
Given a config with field "event_time" of type time
  And a representation capturing hour, minute, second, and fraction group
  And a CSV with rows:
    | event_time       |
    | 14:30:15.123456  |
When the parse command runs
Then the output includes the 123456 microsecond fractional component
```

### Scenario: Fractional seconds with 3-digit precision are right-padded

```gherkin
Given a config with field "event_time" of type time
  And a representation capturing hour, minute, second, and fraction group
  And a CSV with rows:
    | event_time    |
    | 14:30:15.123  |
When the parse command runs
Then the fractional component is treated as 123000 microseconds (padded to 6 digits)
```

### Scenario: Custom time format via representation

```gherkin
Given a config with field "appointment_time" of type time
  And a representation ^(?P<hour>\d{1,2}):(?P<minute>\d{2})$ (no seconds)
  And a CSV with rows:
    | appointment_time |
    | 9:30             |
When the parse command runs
Then the output equals microseconds for 09:30:00.000000
  And second defaults to 0
```

### Scenario: Midnight produces zero

```gherkin
Given a config with field "closing_time" of type time
  And a CSV with rows:
    | closing_time |
    | 00:00:00     |
When the parse command runs
Then the output Avro contains value 0 for closing_time
```

### Scenario: One second before midnight

```gherkin
Given a config with field "last_call" of type time
  And a CSV with rows:
    | last_call |
    | 23:59:59  |
When the parse command runs
Then the output Avro contains value 86399000000 for last_call
```

### Scenario: Invalid hour is rejected

```gherkin
Given a config with field "shift_start" of type time
  And a CSV where row 2 has hour component "25"
When the parse command runs
Then the command exits with non-zero status
  And the error output mentions the invalid time component
```

### Scenario: Invalid minute is rejected

```gherkin
Given a config with field "shift_start" of type time
  And a CSV where row 2 has minute component "60"
When the parse command runs
Then the command exits with non-zero status
  And the error output mentions the invalid time component
```

---

## Timestamp Type

### Scenario: ISO timestamp with Z offset converts to UTC microseconds

```gherkin
Given a config with field "event_timestamp" of type timestamp
  And a representation for ISO format extracting year/month/day/hour/minute/second
  And a CSV with rows:
    | event_timestamp          |
    | 2024-03-15T14:30:00Z     |
When the parse command runs
Then the output Avro contains the correct int64 microsecond value for 2024-03-15 14:30:00 UTC
  And the Avro schema defines event_timestamp with logicalType "timestamp-micros"
  And the schema base type is "long"
```

### Scenario: Timestamp with positive timezone offset converts to UTC

```gherkin
Given a config with field "event_timestamp" of type timestamp
  And a representation extracting components and timezone offset
  And a CSV with rows:
    | event_timestamp              |
    | 2024-03-15T14:30:00+05:30    |
When the parse command runs
Then the output Avro contains the UTC-equivalent microsecond value
  And the value equals microseconds for 2024-03-15T09:00:00Z
```

### Scenario: Timestamp with negative timezone offset converts to UTC

```gherkin
Given a config with field "event_timestamp" of type timestamp
  And a CSV with rows:
    | event_timestamp              |
    | 2024-03-15T14:30:00-04:00    |
When the parse command runs
Then the output Avro contains the UTC-equivalent microsecond value
  And the value equals microseconds for 2024-03-15T18:30:00Z
```

### Scenario: Timestamp with fractional seconds preserves precision

```gherkin
Given a config with field "event_timestamp" of type timestamp
  And a CSV with rows:
    | event_timestamp                  |
    | 2024-03-15T14:30:00.123456Z      |
When the parse command runs
Then the output Avro preserves microsecond precision
```

### Scenario: Component-based timestamp with defaults for missing time parts

```gherkin
Given a config with a representation capturing only year, month, day for a timestamp field
  And a CSV with date-only value "2024-03-15"
When the parse command runs
Then the output equals microseconds for 2024-03-15T00:00:00Z
  And hour, minute, second default to 0
  And timezone defaults to UTC
```

### Scenario: Epoch seconds with static precision arg

```gherkin
Given a config with a representation pattern "^(?P<value>\d+)$"
  And static args {precision: "seconds"} on the representation
  And a CSV with rows:
    | epoch_time  |
    | 1710512345  |
When the parse command runs
Then the output Avro contains value 1710512345000000
```

### Scenario: Epoch milliseconds with static precision arg

```gherkin
Given a config with a representation pattern "^(?P<value>\d+)$"
  And static args {precision: "milliseconds"} on the representation
  And a CSV with rows:
    | epoch_time     |
    | 1710512345000  |
When the parse command runs
Then the output Avro contains value 1710512345000000
```

### Scenario: Invalid timestamp month is rejected

```gherkin
Given a config with field "event_timestamp" of type timestamp
  And a CSV with timestamp containing month "13"
When the parse command runs
Then the command exits with non-zero status
  And the error output mentions the invalid timestamp component
```

---

## Cross-Type Scenarios

### Scenario: All four new types in same config file

```gherkin
Given a config with:
  | field       | type      |
  | status      | enum      |
  | event_date  | date      |
  | event_time  | time      |
  | created_at  | timestamp |
  And a CSV with valid data for all fields
When the parse command runs
Then all four fields convert with correct types and values
  And the Avro schema contains correct logical type annotations for each
```

### Scenario: New types mixed with existing types in same config

```gherkin
Given a config with:
  | field       | type      |
  | id          | integer   |
  | name        | string    |
  | price       | decimal   |
  | status      | enum      |
  | event_date  | date      |
  | event_time  | time      |
  | created_at  | timestamp |
  And a CSV with valid data for all fields
When the parse command runs
Then all seven logical types work correctly in the same pipeline run
  And the Avro schema contains appropriate type definitions for each field
```

### Scenario: Multiple representations per field with first-match-wins behavior

```gherkin
Given a config with field "transaction_date" of type date
  And representations in order:
    | pattern                                              |
    | ^(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})$   |
    | ^(?P<day>\d{2})/(?P<month>\d{2})/(?P<year>\d{4})$   |
  And a CSV with rows:
    | transaction_date |
    | 2024-03-15       |
    | 15/03/2024       |
When the parse command runs
Then row 1 matches the ISO representation
  And row 2 matches the European representation
  And both produce the same days-since-epoch output value
```
