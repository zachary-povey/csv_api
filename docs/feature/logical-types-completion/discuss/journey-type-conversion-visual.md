# Journey: Converting Messy CSV Data Through Flexible Representations to Typed Avro

## Architecture Reminder

```
CSV cell string
    |
    v
Representation regex match (first match wins)
    |
    v
Extract named capture groups as args
    |
    v
Merge with static args (regex wins on overlap)
    |
    v
Converter receives args dict (never sees raw input)
    |
    v
Typed output value
    |
    v
Avro schema validates + writes
```

The key insight: **representations handle input flexibility, converters handle output correctness**. The converter for enum does not know whether the input was "active", "Active", or "live" -- it only sees `{value: "active"}`.

## Journey Flow

```
[Define Config]       [Validate Config]       [Run Parse]              [Verify Output]
     |                      |                      |                        |
     v                      v                      v                        v
Write YAML with         ./build validate_config  ./build parse            Read Avro output
logical_type +          --config_path cfg.yaml   --config/data/output     and inspect schema
representations
     |                      |                      |                        |
Feels: Thoughtful       Feels: Hopeful          Feels: Anxious          Feels: Confident
"Which formats do I     "Will it accept my       "Will my messy input    "The output has
 need to accept?"        type and args?"          convert correctly?"     clean typed values"
```

## Emotional Arc

- **Start**: Thoughtful -- "I know what output type I need, but I need to think about what input formats to expect and how to capture the right named groups."
- **Middle**: Anxious at parse time -- "Will the regex extract the right components? Will the converter validate them correctly?"
- **End**: Confident -- "Messy input produced clean, typed Avro. My pipeline handles format variation declaratively."

Pattern: **Confidence Building** -- small wins at each validation step reduce anxiety progressively.

## Step Details

### Step 1: Define Config (YAML Authoring)

The user's main creative work: choosing logical type, configuring type-specific args, and writing representations that capture the right named groups from their specific input formats.

**Enum example -- accepting variant inputs, mapping to canonical values:**

```
+-- Config YAML ---------------------------------------------------+
| fields:                                                          |
|   - name: status                                                 |
|     logical_type:                                                |
|       name: enum                                                 |
|       args:                                                      |
|         permitted_values: [active, inactive]  <-- output vocab   |
|     representations:                                             |
|       - pattern: "(?i)(?P<value>active|inactive)"                |
|           ^-- case-insensitive regex, extracts value             |
|       - pattern: "live"                                          |
|         args:                                                    |
|           value: "active"    <-- static arg remaps "live"        |
|       - pattern: "disabled"                                      |
|         args:                                                    |
|           value: "inactive"  <-- static arg remaps "disabled"    |
+------------------------------------------------------------------+
```

**Date example -- multiple input formats via different representations:**

```
+-- Config YAML ---------------------------------------------------+
| fields:                                                          |
|   - name: transaction_date                                       |
|     logical_type:                                                |
|       name: date                                                 |
|     representations:                                             |
|       - pattern: "^(?P<year>\\d{4})-(?P<month>\\d{2})-(?P<day>\\d{2})$"
|           ^-- ISO format: 2024-03-15                             |
|       - pattern: "^(?P<day>\\d{2})/(?P<month>\\d{2})/(?P<year>\\d{4})$"
|           ^-- European: 15/03/2024                               |
|       - pattern: "^(?P<month>\\d{2})-(?P<day>\\d{2})-(?P<year>\\d{4})$"
|           ^-- US: 03-15-2024                                     |
+------------------------------------------------------------------+
```

**Timestamp example -- epoch with static precision arg:**

```
+-- Config YAML ---------------------------------------------------+
| fields:                                                          |
|   - name: created_at                                             |
|     logical_type:                                                |
|       name: timestamp                                            |
|     representations:                                             |
|       - pattern: "^(?P<value>\\d+)$"                             |
|         args:                                                    |
|           precision: "seconds"  <-- static arg, not in regex     |
+------------------------------------------------------------------+
```

Shared artifacts: config YAML file (consumed by validate_config, parse subcommands)

### Step 2: Validate Config

```
$ ./build validate_config --config_path config.yaml

+-- Success Output ------------------------------------------------+
| Config valid.                                                    |
+------------------------------------------------------------------+

+-- Error Output (type-specific arg missing) ----------------------+
| Error: permitted_values is required for enum type                |
+------------------------------------------------------------------+

+-- Error Output (unknown type name) ------------------------------+
| Error: unknown logical type in config: 'timestamps'              |
+------------------------------------------------------------------+
```

Integration checkpoint: Config parses. Type-specific args validated. Unknown types caught. Regex compilation not checked here (happens at parse time).

### Step 3: Run Parse

The pipeline processes each CSV cell through representations, then converters.

```
$ ./build parse --config_path config.yaml --data_path data.csv --output_path output.avro

+-- Success (exit code 0) -----------------------------------------+
| (output.avro written)                                            |
+------------------------------------------------------------------+

+-- Representation Failure (no regex matched) ----------------------+
| value 'unknown_status' did not match any pattern in              |
| column 'status'                                                  |
+------------------------------------------------------------------+

+-- Converter Failure (args invalid for type) ---------------------+
| Failed to convert 'actve' to type 'enum'                        |
| Resolved args: map[value:actve]                                  |
| Exception:                                                       |
|  value 'actve' is not in permitted values [active inactive]      |
+------------------------------------------------------------------+

+-- Converter Failure (invalid temporal component) ----------------+
| Failed to convert '2024-13-15' to type 'date'                   |
| Resolved args: map[year:2024 month:13 day:15]                   |
| Exception:                                                       |
|  invalid month: 13 (must be 1-12)                                |
+------------------------------------------------------------------+
```

Key: two distinct failure modes. Representation failure = no regex matched the raw input. Converter failure = regex matched and extracted args, but args are invalid for the type (enum value not permitted, date component out of range).

### Step 4: Verify Output

```
$ python -c "import fastavro; r=fastavro.reader(open('output.avro','rb')); \
  print(r.writer_schema); [print(rec) for rec in r]"

+-- Schema Fragment -----------------------------------------------+
| {"name":"status","type":{"type":"enum","name":"status",          |
|   "symbols":["active","inactive"]}}                              |
| {"name":"transaction_date","type":{"type":"int",                 |
|   "logicalType":"date"}}                                         |
| {"name":"event_time","type":{"type":"long",                      |
|   "logicalType":"time-micros"}}                                  |
| {"name":"created_at","type":{"type":"long",                      |
|   "logicalType":"timestamp-micros"}}                             |
+------------------------------------------------------------------+

+-- Records -------------------------------------------------------+
| {'status': 'active', 'transaction_date': 19797,                  |
|  'event_time': 34200000000, 'created_at': 1710512345000000}      |
+------------------------------------------------------------------+
```

Integration checkpoint: Avro records contain correctly typed values. Schema includes logical type metadata for downstream tool recognition.

## Error Paths

| Error Category | When | User Sees | Recovery |
|----------------|------|-----------|----------|
| Unknown type name | Config load | "unknown logical type: 'timestamps'" | Fix typo in config |
| Missing required args | Config validation | "permitted_values is required for enum" | Add missing args to config |
| No representation matched | Parse time | "value 'X' did not match any pattern in column 'Y'" | Add representation for this input format or fix source data |
| Extracted value not in enum | Parse time (converter) | "value 'X' is not in permitted values [...]" | Fix source data, add representation with static arg to remap, or expand permitted_values |
| Invalid date component | Parse time (converter) | "invalid month: 13" with resolved args shown | Fix source data or adjust regex to reject before converter |
| Invalid time component | Parse time (converter) | "invalid hour: 25" with resolved args shown | Fix source data or adjust regex |
| Timezone parse failure | Parse time (converter) | "unknown timezone: 'XYZ'" | Fix source data or use offset format instead of named timezone |
