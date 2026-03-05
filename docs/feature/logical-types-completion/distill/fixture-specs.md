# Fixture Specifications: logical-types-completion

Each fixture is a directory under `tests/fixtures/<name>/` containing `config.yaml` and `data.csv`.

---

## Enum Fixtures

### `enum_basic`

**config.yaml:**

```yaml
fields:
  - name: status
    logical_type:
      name: enum
      args:
        permitted_values:
          - active
          - inactive
          - pending
    representations:
      - pattern: "(?P<value>.*)"
```

**data.csv:**

```csv
status
active
inactive
pending
```

---

### `enum_case_insensitive`

**config.yaml (final):**

```yaml
fields:
  - name: status
    logical_type:
      name: enum
      args:
        permitted_values:
          - active
          - inactive
    representations:
      - pattern: "^(?i)(?:active)$"
        args:
          value: active
      - pattern: "^(?i)(?:inactive)$"
        args:
          value: inactive
```

**data.csv:**

```csv
status
ACTIVE
Inactive
active
```

---

### `enum_value_remap`

**config.yaml:**

```yaml
fields:
  - name: status
    logical_type:
      name: enum
      args:
        permitted_values:
          - active
          - inactive
    representations:
      - pattern: "^(?:live)$"
        args:
          value: active
      - pattern: "^(?:disabled)$"
        args:
          value: inactive
      - pattern: "(?P<value>active|inactive)"
```

**data.csv:**

```csv
status
live
disabled
active
```

---

### `enum_multiple_representations`

**config.yaml:**

```yaml
fields:
  - name: priority
    logical_type:
      name: enum
      args:
        permitted_values:
          - low
          - medium
          - high
    representations:
      - pattern: "(?P<value>low|medium|high)"
      - pattern: "^(?:1)$"
        args:
          value: low
      - pattern: "^(?:2)$"
        args:
          value: medium
      - pattern: "^(?:3)$"
        args:
          value: high
```

**data.csv:**

```csv
priority
low
3
medium
1
high
2
```

---

### `enum_invalid_value`

**config.yaml:**

```yaml
fields:
  - name: status
    logical_type:
      name: enum
      args:
        permitted_values:
          - active
          - inactive
    representations:
      - pattern: "(?P<value>.*)"
```

**data.csv:**

```csv
status
active
unknown
```

---

### `enum_mixed_types`

**config.yaml:**

```yaml
fields:
  - name: name
    logical_type:
      name: string
    representations:
      - pattern: "(?P<value>.*)"
  - name: age
    logical_type:
      name: integer
    representations:
      - pattern: "(?P<value>[0-9]+)"
  - name: status
    logical_type:
      name: enum
      args:
        permitted_values:
          - active
          - inactive
    representations:
      - pattern: "(?P<value>active|inactive)"
```

**data.csv:**

```csv
name,age,status
Alice,30,active
Bob,25,inactive
```

---

## Date Fixtures

### `date_iso`

**config.yaml:**

```yaml
fields:
  - name: event_date
    logical_type:
      name: date
    representations:
      - pattern: "(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})"
```

**data.csv:**

```csv
event_date
2024-03-15
2000-01-01
1999-12-31
```

---

### `date_custom_format`

**config.yaml:**

```yaml
fields:
  - name: event_date
    logical_type:
      name: date
    representations:
      - pattern: "(?P<day>[0-9]{2})/(?P<month>[0-9]{2})/(?P<year>[0-9]{4})"
```

**data.csv:**

```csv
event_date
15/03/2024
01/01/2000
31/12/1999
```

---

### `date_invalid_components`

**config.yaml:**

```yaml
fields:
  - name: event_date
    logical_type:
      name: date
    representations:
      - pattern: "(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})"
```

**data.csv:**

```csv
event_date
2024-03-15
2024-13-01
```

---

### `date_mixed_types`

**config.yaml:**

```yaml
fields:
  - name: event_name
    logical_type:
      name: string
    representations:
      - pattern: "(?P<value>.*)"
  - name: event_date
    logical_type:
      name: date
    representations:
      - pattern: "(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})"
  - name: attendees
    logical_type:
      name: integer
    representations:
      - pattern: "(?P<value>[0-9]+)"
```

**data.csv:**

```csv
event_name,event_date,attendees
Conference,2024-03-15,250
Workshop,2024-06-20,50
```

---

## Time Fixtures

### `time_iso`

**config.yaml:**

```yaml
fields:
  - name: event_time
    logical_type:
      name: time
    representations:
      - pattern: "(?P<hour>[0-9]{2}):(?P<minute>[0-9]{2}):(?P<second>[0-9]{2})"
```

**data.csv:**

```csv
event_time
09:30:00
14:15:45
23:00:00
```

---

### `time_fractional_seconds`

**config.yaml:**

```yaml
fields:
  - name: event_time
    logical_type:
      name: time
    representations:
      - pattern: "(?P<hour>[0-9]{2}):(?P<minute>[0-9]{2}):(?P<second>[0-9]{2})\\.(?P<microsecond>[0-9]+)"
```

**data.csv:**

```csv
event_time
09:30:00.123456
14:15:45.500000
00:00:00.000001
```

---

### `time_custom_format`

**config.yaml:**

```yaml
fields:
  - name: event_time
    logical_type:
      name: time
    representations:
      - pattern: "(?P<hour>[0-9]{1,2})h(?P<minute>[0-9]{2})m"
```

**data.csv:**

```csv
event_time
9h30m
14h00m
0h00m
```

---

### `time_edge_cases`

**config.yaml:**

```yaml
fields:
  - name: event_time
    logical_type:
      name: time
    representations:
      - pattern: "(?P<hour>[0-9]{2}):(?P<minute>[0-9]{2}):(?P<second>[0-9]{2})"
```

**data.csv:**

```csv
event_time
00:00:00
23:59:59
12:00:00
```

---

## Timestamp Fixtures

### `timestamp_iso_offset`

**config.yaml:**

```yaml
fields:
  - name: created_at
    logical_type:
      name: timestamp
    representations:
      - pattern: "(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})T(?P<hour>[0-9]{2}):(?P<minute>[0-9]{2}):(?P<second>[0-9]{2})(?P<timezone>[+-][0-9]{2}:[0-9]{2})"
```

**data.csv:**

```csv
created_at
2024-03-15T10:30:00+05:00
2024-01-01T00:00:00-08:00
```

---

### `timestamp_iso_utc`

**config.yaml:**

```yaml
fields:
  - name: created_at
    logical_type:
      name: timestamp
    representations:
      - pattern: "(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})T(?P<hour>[0-9]{2}):(?P<minute>[0-9]{2}):(?P<second>[0-9]{2})(?:Z)"
```

**data.csv:**

```csv
created_at
2024-03-15T10:30:00Z
2024-01-01T00:00:00Z
```

Note: The `(?:Z)` is a non-capturing group so EnsureValueName does not wrap the pattern (total_caps > 0). No `timezone` arg means UTC assumed.

---

### `timestamp_custom_format`

**config.yaml:**

```yaml
fields:
  - name: created_at
    logical_type:
      name: timestamp
    representations:
      - pattern: "(?P<day>[0-9]{2})/(?P<month>[0-9]{2})/(?P<year>[0-9]{4}) (?P<hour>[0-9]{2}):(?P<minute>[0-9]{2}):(?P<second>[0-9]{2})"
```

**data.csv:**

```csv
created_at
15/03/2024 10:30:00
01/01/2000 00:00:00
```

---

### `timestamp_epoch`

**config.yaml:**

```yaml
fields:
  - name: created_at
    logical_type:
      name: timestamp
    representations:
      - pattern: "(?P<value>[0-9]+)"
        args:
          precision: seconds
```

**data.csv:**

```csv
created_at
1710499800
946684800
```

Note: 1710499800 = 2024-03-15T13:30:00Z, 946684800 = 2000-01-01T00:00:00Z.

---

### `timestamp_timezone_conversion`

**config.yaml:**

```yaml
fields:
  - name: created_at
    logical_type:
      name: timestamp
    representations:
      - pattern: "(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})T(?P<hour>[0-9]{2}):(?P<minute>[0-9]{2}):(?P<second>[0-9]{2})(?P<timezone>[+-][0-9]{2}:[0-9]{2})"
```

**data.csv:**

```csv
created_at
2024-03-15T10:00:00+05:00
2024-07-01T20:00:00-04:00
```

Expected UTC values: 2024-03-15T05:00:00Z, 2024-07-02T00:00:00Z.

---

### `timestamp_invalid`

**config.yaml:**

```yaml
fields:
  - name: created_at
    logical_type:
      name: timestamp
    representations:
      - pattern: "(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})T(?P<hour>[0-9]{2}):(?P<minute>[0-9]{2}):(?P<second>[0-9]{2})(?:Z)"
```

**data.csv:**

```csv
created_at
2024-03-15T10:30:00Z
2024-03-15T25:00:00Z
```

---

## Walking Skeleton Fixture

### `all_new_types_mixed`

**config.yaml:**

```yaml
fields:
  - name: employee_name
    logical_type:
      name: string
    representations:
      - pattern: "(?P<value>.*)"
  - name: employee_id
    logical_type:
      name: integer
    representations:
      - pattern: "(?P<value>[0-9]+)"
  - name: department
    logical_type:
      name: enum
      args:
        permitted_values:
          - engineering
          - marketing
          - sales
    representations:
      - pattern: "(?P<value>engineering|marketing|sales)"
  - name: hire_date
    logical_type:
      name: date
    representations:
      - pattern: "(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})"
  - name: shift_start
    logical_type:
      name: time
    representations:
      - pattern: "(?P<hour>[0-9]{2}):(?P<minute>[0-9]{2}):(?P<second>[0-9]{2})"
  - name: last_login
    logical_type:
      name: timestamp
    representations:
      - pattern: "(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})T(?P<hour>[0-9]{2}):(?P<minute>[0-9]{2}):(?P<second>[0-9]{2})(?:Z)"
```

**data.csv:**

```csv
employee_name,employee_id,department,hire_date,shift_start,last_login
Alice,101,engineering,2020-06-15,09:00:00,2024-03-15T08:30:00Z
Bob,102,marketing,2019-01-10,08:30:00,2024-03-14T17:45:00Z
```
