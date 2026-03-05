# Four Forces Analysis: Logical Types Completion

## Overall Feature Context

The tool's value proposition is: define a strict output schema (Avro) while accepting arbitrarily messy input formats via flexible regex representations. Representations extract named groups as args, merge with static args, and pass to converters that produce typed output. The converter never sees the raw input string.

---

## Forces Analysis: Enum Type

### Demand-Generating

- **Push**: Partner CSVs contain categorical columns with inconsistent labels -- "active", "Active", "live", "enabled" all meaning the same status. Currently Ravi writes Python pre-processing scripts to normalize these before feeding to csv_api. These scripts are fragile, per-partner, and break when upstream systems change their labels.
- **Pull**: Define one field with `permitted_values: [active, inactive]`, then write representations: a case-insensitive regex `(?i)(?P<value>active|inactive)`, a remapping pattern `"live"` with `args: {value: "active"}`, and `"disabled"` with `args: {value: "inactive"}`. Input flexibility in the config, strict output vocabulary in Avro.

### Demand-Reducing

- **Anxiety**: Will the converter correctly reject values that pass the regex but are not in permitted_values? If the regex extracts "cancelled" but it is not a permitted value, is that caught? What about subtle whitespace issues in extracted values?
- **Habit**: Using string type with no validation and cleaning up downstream. It works, it is familiar, and adding enum constraints feels like it could cause more pipeline failures initially.

### Assessment

- Switch likelihood: High
- Key blocker: Anxiety about the boundary between representation matching and converter validation
- Key enabler: Strong push from repeated per-partner normalization script maintenance
- Design implication: Converter must clearly reject extracted values not in permitted_values with an error naming the value and the full permitted set. The regex-then-validate two-step must be well-understood.

---

## Forces Analysis: Date Type

### Demand-Generating

- **Push**: Date columns arrive in DD/MM/YYYY from European partners, MM-DD-YYYY from US systems, YYYYMMDD compact from APIs. Ravi writes per-format parsing logic in Python to extract components and compute days-since-epoch. Each new partner format means new code.
- **Pull**: Write a representation regex per format -- `(?P<day>\d{2})/(?P<month>\d{2})/(?P<year>\d{4})` for European, `(?P<month>\d{2})-(?P<day>\d{2})-(?P<year>\d{4})` for US. The converter receives the same (year, month, day) args regardless of input format and produces Avro date. New format = new regex, no code changes.

### Demand-Reducing

- **Anxiety**: Invalid date components (month=13, day=32, Feb 30 in non-leap year) could produce corrupt Avro output if the converter does not validate the assembled date. The regex only validates string format, not calendar validity.
- **Habit**: Storing dates as strings and deferring parsing to downstream consumers avoids the problem entirely. Spark and Presto can parse date strings at query time.

### Assessment

- Switch likelihood: High
- Key blocker: Anxiety about invalid date assembly from regex-extracted components
- Key enabler: Push from maintaining per-format parsing scripts
- Design implication: Converter must validate that extracted year/month/day form a real calendar date, rejecting impossible combinations with clear errors

---

## Forces Analysis: Time Type

### Demand-Generating

- **Push**: Time columns use 12-hour ("3:05 PM"), 24-hour ("15:05:00"), and fractional-second ("15:05:00.123456") formats. No current path to normalize these into a typed Avro time field. Ravi either stores as string or converts to timestamp (carrying unnecessary date).
- **Pull**: Regex extracts hour/minute/second/fraction components. A 12-hour format representation could use static args or regex logic to convert PM hours. Converter produces microseconds-since-midnight. Handles any time format through representation flexibility.

### Demand-Reducing

- **Anxiety**: Microsecond precision handling -- will fractional seconds of varying digit lengths (3 digits = ms, 6 = us, 9 = ns) convert correctly? Will truncation be silent or reported?
- **Habit**: Embedding time-of-day in full timestamps or storing as strings. Most analysis tools handle timestamp-to-time extraction.

### Assessment

- Switch likelihood: Medium-High
- Key blocker: Anxiety about fractional second precision behavior
- Key enabler: Push from scheduling domains where time-without-date is semantically important
- Design implication: Converter must handle fractional seconds of any digit length, padding or truncating to microsecond precision predictably. Behavior for >6 fractional digits should be explicit.

---

## Forces Analysis: Timestamp Type

### Demand-Generating

- **Push**: Timestamp data arrives as ISO strings with timezone offsets, epoch seconds from APIs, epoch milliseconds from JavaScript clients, and custom formats like "15-Mar-2024 2:30pm EST". The most complex temporal type with the most format variation. Ravi maintains the most code for timestamp handling.
- **Pull**: Two converter parameter sets handle both patterns: epoch-based (value + precision + offset) and component-based (year/month/day/hour/minute/second/timezone). Representations extract whatever components exist from whatever format. A static arg `precision: "seconds"` on an epoch representation eliminates the need to encode precision in the regex.

### Demand-Reducing

- **Anxiety**: Timezone handling is notoriously subtle. Will UTC offsets, named timezones ("America/New_York"), and implicit-UTC (no timezone specified) all produce correct microsecond-epoch values? Will DST transitions be handled? If timezone handling is wrong, financial calculations downstream could be incorrect and the error would be invisible until someone audits the numbers.
- **Habit**: Treating timestamps as raw epoch longs or ISO strings and pushing timezone interpretation to consumers. This is the path of least resistance and avoids the tool making timezone decisions.

### Assessment

- Switch likelihood: Medium
- Key blocker: Anxiety about timezone correctness -- this is the strongest anxiety across all four types
- Key enabler: Push from timestamp format chaos across data sources (strongest push across all four types)
- Design implication: Default behavior for missing timezone must be documented and predictable (UTC). Both epoch and component paths must produce identical output for equivalent inputs. Named timezone support scope should be explicitly defined.
