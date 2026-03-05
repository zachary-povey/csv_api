# JTBD Job Stories: Logical Types Completion

## Persona: Ravi Krishnan, Data Engineer

Ravi works at a mid-size fintech company. He builds ETL pipelines that ingest CSV exports from partner systems and converts them into Avro for downstream analytics and data warehouse loading. He uses csv_api as the conversion layer between raw CSV and typed Avro output. His CSV sources are messy -- dates in DD/MM/YYYY from European partners, timestamps as epoch milliseconds from APIs, enum-like columns where "live" and "active" mean the same thing.

---

## JS-01: Categorical Data Standardization via Flexible Input Mapping

**When** I receive CSV exports from partner systems containing categorical fields that use inconsistent naming (e.g., "active", "Active", "live", "enabled" all meaning the same status),
**I want to** define representations that accept variant input strings and remap them to a canonical set of permitted enum values,
**so I can** enforce a strict output vocabulary while tolerating messy input without rejecting valid data or writing pre-processing scripts.

### Functional Job
Map variant categorical strings to a controlled vocabulary through regex matching and static arg remapping, then validate the extracted value against a permitted set.

### Emotional Job
Feel in control of data quality -- knowing that inconsistent labels from upstream systems are handled declaratively in the config, not with fragile pre-processing scripts.

### Social Job
Deliver data to downstream teams that uses a single, agreed-upon vocabulary for categorical fields, regardless of how inconsistent the source was.

---

## JS-02: Multi-Format Date Normalization

**When** I receive CSV exports containing date columns in inconsistent formats (ISO "2024-03-15", European "15/03/2024", US "03-15-2024", compact "20240315"),
**I want to** define a regex representation per format that extracts year/month/day named groups and maps them to Avro date type,
**so I can** produce date-typed Avro columns that downstream query engines recognize natively for partition pruning and date arithmetic, without manual format conversion.

### Functional Job
Extract year/month/day components from arbitrarily formatted date strings via regex named groups and convert to Avro date (int32 days since epoch).

### Emotional Job
Feel certain that date boundaries are correct regardless of input format, and that no off-by-one errors will cause records to land in the wrong partition.

### Social Job
Provide the data platform team with properly typed date columns from any partner format without manual post-processing.

---

## JS-03: Time-of-Day Normalization from Heterogeneous Formats

**When** I receive CSV data containing time-of-day fields in various formats (24-hour "15:05:00", 12-hour "3:05 PM", fractional "15:05:00.123456"),
**I want to** define representations that extract time components (hour, minute, second, fractional parts) from any format string,
**so I can** produce Avro time-micros values that preserve sub-second precision without carrying unnecessary date baggage.

### Functional Job
Extract hour/minute/second/fractional-second components from varied time format strings and convert to Avro time-micros (int64 microseconds since midnight).

### Emotional Job
Feel assured that sub-second precision is preserved and that midnight edge cases are handled correctly, regardless of input format quirks.

### Social Job
Deliver scheduling data that the application team can consume without custom parsing logic.

---

## JS-04: Timestamp Conversion from Diverse Sources

**When** I receive CSV event logs with timestamps in diverse formats (ISO 8601 with timezone offsets, epoch seconds from APIs, epoch milliseconds from JavaScript clients, custom formats with separate date and time components),
**I want to** define representations that extract temporal components or epoch values via regex named groups and static args (e.g., hardcoding precision="seconds"), converting them to canonical Avro timestamp-micros,
**so I can** produce time-series data that downstream systems can query, join, and aggregate without timezone or precision ambiguity.

### Functional Job
Parse timestamps via two parameter paths -- epoch-based (value + precision + optional offset) and component-based (year/month/day/hour/minute/second/timezone) -- and convert to Avro timestamp-micros (int64 microseconds since Unix epoch UTC).

### Emotional Job
Feel relieved that timezone and precision handling is correct, avoiding the anxiety of silent data corruption in time-sensitive financial calculations.

### Social Job
Demonstrate to the platform team that event data is ingested with full temporal fidelity from any source format.

---

## Cross-Cutting Concerns

All four job stories share these needs:

- **Representation-first architecture**: Input flexibility is handled entirely by regex representations. Converters only see extracted args -- they never touch raw CSV strings. This separation is the tool's core value proposition.
- **Error reporting**: When a value fails conversion, the error message must identify the row, field, raw value, resolved args, and the specific problem.
- **Config consistency**: Each type follows the existing config YAML pattern (logical_type name + optional args + representations with regex patterns and optional static args).
- **Pipeline integration**: Converted values flow through the existing Reader -> Parser -> Avro Writer channel pipeline without architectural changes.
