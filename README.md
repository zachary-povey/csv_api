# csv_api

The csv validator I wish I didn't need.

## Background

A tool for a specific use case: when csv data is the data format used to transfer data across application boundaries (within, or between, organisations). In this scenario, CSV is a bad format because, lacking types and schema, it offers no way to define a formal set of expectations on the data being transferred and therefore no method with which to verify the data is as expected.

In lieu of this, informal, poorly defined expectations of the data are often used, with work often scoped out using "example" files which may not capture edge cases and often do not accurately reflect the final data format after go-live. Furthermore, in the scenario where CSV is the only available format (usually due to a lack of knowledge or resourcing on the data producer side), the technologies used on the producer side (usually an off-the-shelf reporting tools, excel, sql queries or a combination of these) can often be a source of ongoing drift in the data format.

Because of the lack of a formal format, and the ability of the format to drift, the import and export side of the transfer process frequently do not align, meaning the process errors frequently. Without a formal validation step, misinterpreted data may only cause errors in downstream processes that use the data, or even worse silently pass through all processes but invoke unintended behaviour or display invalid data to report users. These downstream issues can typically be hard to debug. In it's worst incarnation, this can lead to an "automatic" data transfer process that requires so much ongoing maintenance it actually uses more dev time than manually importing the data would.

The solution to this problem, as I see it is:

**If you are the data producer**, use a better format. Avro or Parquet preferably, but json with an agreed json schema can also do the job. The schema can then be agreed with the consumers in the scoping stage of the project and used by the process writing and the process reading the data.

**If you are the consumer**, ask, demand or beg that the producer uses a better format. Flat out refuse to ingest excel files and pull every organisational lever you can to try and agree a new format with the producers. If you are truly powerless to get this changed then:

- csv format expectations should be defined formally during project scoping
- the expectation should be validated against as the first step in the process
- the validation step should also convert the data to a well-typed format for downstream processes to use
- validation failures should be reported in good detail

### Scope

- to formally specify the bespoke physical representation of logical types agreed by the csv API
- to validate input files match this specification
- to convert input files to a file type with a standard physical representation of the same logical types
- command line config overrides
- partial conversion (bad rows dropped)
- error reports
- custom errors

### Out of scope

- validation of logical values
- combinations of fields etc
- capturing contextual information (e.g when a file was received, allow use cases to be sorted via config overrides)

### Notes

At some point, the data will have to be converted, do this first formally. If the expectations are not met, errors would have occurred anyway

### Design principles

1. The config is the contract agreed with the data producer. Command-line flags are the consumer's run-time choices. Keep contract in config and run-time behaviour in flags.
2. The output schema is defined by the root config only. Producer-specific overrides (layered configs) may change input-side settings only, so every producer's output has the same schema.
3. Strict by default; anything more lenient is an explicit opt-in.
4. A failed run leaves no output behind. The only exception is the explicit `--stream` opt-in, where the non-zero exit code signals incomplete output.
5. Never lose data silently: dropped rows always produce a warning and can be written to a report.
6. Catch config mistakes when the config loads, not per cell at run time.
7. Converters never see raw input. Input flexibility belongs in representations (regexes and static args); converters only check that their args form a valid value.
8. Errors are specific: row, column, value and reason. Stdout carries only output data; errors and warnings go to stderr.
9. Out of scope: validating logical values, cross-field rules, capturing contextual information (see [Out of scope](#out-of-scope)).

## Usage

Build the binary (written to `build/csv-api`):

```sh
./scripts/build.sh
```

Validate a csv file against a config and convert it to Avro:

```sh
./build/csv-api parse --config-path config.yaml --input-path data.csv --output-path data.avro
# short form
./build/csv-api parse -c config.yaml -i data.csv -o data.avro
```

The command exits non-zero and prints an error report if the file does not match the config. A failed run never leaves an output file behind.

Check a config file on its own:

```sh
./build/csv-api validate_config -c config.yaml
```

### Error handling

Errors fall into three groups:

| Group | Examples | Behaviour |
| --- | --- | --- |
| Execution errors | input can't be opened, output can't be written | always fail the run immediately |
| Format errors | unreadable header, missing column, row with the wrong number of values | always fail the run |
| Data failures | a value that matches no representation, or fails conversion (e.g. 30 February) | fail the run, unless `--error-on-data-failures=false` |

These flags control how errors are handled. They are choices for whoever runs the tool, not part of the config agreed with the data producer.

| Flag | Default | Description |
| --- | --- | --- |
| `--error-on-data-failures` | `true` | Fail the run on data failures. If `false`, rows containing invalid values are dropped, the rest are written, and a warning gives the number dropped. |
| `--fail-fast` | `true` | Stop at the first error. If `false`, keep going and report every error found. |
| `--max-errors` | no limit | With `--fail-fast=false`, stop after collecting this many errors. Dropped rows don't count as errors. |
| `--data-failure-report` | none | With `--error-on-data-failures=false`, write the dropped rows to this csv file: the original header and values, plus a `__csv_api_error__` column listing every failed value in the row (with the field's `error_message`, if it has one). |
| `--error-report` | `console` | Where errors are reported. Only `console` is supported for now. |

Boolean flags are turned off with `=false`, e.g. `--fail-fast=false`. Setting `--max-errors` with fail-fast on, or `--data-failure-report` while erroring on data failures, is rejected.

```sh
# report every problem in the file
./build/csv-api parse -c config.yaml -i data.csv -o data.avro --fail-fast=false

# convert what's valid, keep the rejected rows for follow-up
./build/csv-api parse -c config.yaml -i data.csv -o data.avro \
  --error-on-data-failures=false --data-failure-report rejected.csv
```

A broken quote stops processing even with `--fail-fast=false`, since the rows after it can't be located reliably.

## Configuration

A config lists the fields expected in the file. Each field has a logical type (what the value means) and one or more representations (how that value is written in the csv).

```yaml
name: com.example.orders # the Avro schema name (required)
fields:
  - name: some_date # matched to the csv column with this exact header
    logical_type:
      name: date
    representations:
      # a fixed value: the literal "start" means 2020-01-01
      - pattern: "^start$"
        args:
          year: 2020
          month: 1
          day: 1
      # named capture groups supply the type's args
      - pattern: "^(?P<year>[0-9]{4})-(?P<month>[0-9]{2})-(?P<day>[0-9]{2})$"

  - name: amount
    logical_type:
      name: decimal
      args: # type args apply to the whole column
        precision: 10
        scale: 2
    representations:
      - pattern: "^£?(?P<value>[0-9]+\\.[0-9]{2})$"

  - name: status
    logical_type:
      name: enum
      args:
        permitted_values: [active, inactive]
    representations:
      - pattern: "^(active|inactive)$"
      # remap a variant label onto a permitted value
      - pattern: "^live$"
        args:
          value: active
```

### Name

`name` is required and becomes the name of the Avro record in the output schema, so downstream consumers see a meaningful schema name. It must be a valid Avro name: it starts with a letter or underscore and contains only letters, digits and underscores. It may be a dotted fullname such as `com.example.orders`, in which case the parts before the last dot are the Avro namespace (enum types in the schema, which are named after their field, share that namespace). Because of that, an enum field can't have the same name as the last part of `name`. A missing or invalid name fails the config load.

### Fields

`fields` is a list, and its order sets the order of fields in the output schema. Each field's `name` is its Avro field name (and, for an enum field, its Avro enum type name), so it must be a valid Avro name: it starts with a letter or underscore and contains only letters, digits and underscores (no dots). Field names must be unique. An invalid or duplicate name fails the config load, naming the field. If a csv header isn't a valid Avro name (say `first name`), give the field a valid name and match the header with `header_patterns` (below). A field is matched to a csv column by an exact header match on `name`, unless it has `header_patterns`. The order of columns doesn't matter. Columns in the file that match no field are ignored by default; see `allow_extra_fields` below.

`header_patterns` is a list of Go regular expressions for a field's column header, for producers whose header text varies (case, spacing, suffixes). A column whose header matches any of them is mapped to the field. When a field has `header_patterns`, they replace the match on `name`: the name is then only the field's name in the output, so it can be a clean Avro name while the header is something like `First Name `. Add an exact-name pattern (`^first_name$`) if the name should match too. Like representation patterns, header patterns are not anchored, so use `^` and `$` to match the whole header. An invalid pattern fails the config load, naming the field and pattern.

```yaml
  - name: first_name
    header_patterns:
      - "(?i)^\\s*first[ _]?name\\s*$" # "First Name", "first_name ", "FIRSTNAME", ...
      - "^Given Name$"
    logical_type:
      name: string
    representations:
      - pattern: "^(.+)$"
```

Each field must match at most one column, and each column at most one field. A header that matches more than one field, or a field that matches more than one header (including a header name repeated in the file), is a file-level error naming the headers and fields involved. All problems with the header are reported together, even with fail-fast on. A required field that matches no header is a missing column (see below); the error lists the field's `header_patterns` if it has them. Errors about a value name its column by the header in the file, adding the field name when the two differ (e.g. `column 'First Name ' (field 'first_name')`), and the data failure report keeps the file's own header row.

`allow_extra_fields` is an optional top-level setting (default `true`). By default, columns in the file that match no field are ignored: they never reach the output. Set `allow_extra_fields: false` to treat an unexpected column as a sign the producer changed the format: every column that matches no field is then a file-level error naming its header and column number (e.g. `header 'source' (column 4) matches no field`), reported together with any other header problems. This default is an exception to strict-by-default, since an ignored column can't change the output.

```yaml
name: com.example.orders
allow_extra_fields: false # every column in the file must match a field
fields:
  ...
```

Fields are not nullable by default: every value must resolve to a value of the field's type. Set `nullable: true` on a field to allow nulls. Nulls only come from representations marked `is_null: true` (see [Representations](#representations)), so the producer's null placeholders (`-`, `N/A`, an empty string, ...) are spelled out in the config. An empty cell on a nullable field is not null unless an `is_null` representation matches it; otherwise it fails like any other unmatched value.

```yaml
  - name: discount
    nullable: true
    logical_type:
      name: integer
    representations:
      - pattern: "^(-|N/A|)$" # "-", "N/A" or an empty cell
        is_null: true
      - pattern: "^[0-9]+$"
```

Fields are required by default: the file must have a column for every field, and a missing one is a file-level error. Set `required: false` on a field whose column may be left out of the file. When the column is missing, the field is null in every row; when it is present, its values are parsed as usual. Since a missing column means null, `required: false` is only allowed on a field with `nullable: true`; on any other field it fails the config load, naming the field.

```yaml
  - name: notes
    required: false # the column may be missing from the file
    nullable: true
    logical_type:
      name: string
    representations:
      - pattern: "^(.*)$"
```

`error_message` is an optional plain-language message for whoever has to fix the data, shown whenever the field fails: with each of its data failures (a value that matches no representation or fails conversion), in both the console report and the data failure report's `__csv_api_error__` column, and when its column is missing from the file. It is added after the technical reason, which is still reported, as `(note: <message>)`. Fields without one report their errors as usual.

```yaml
  - name: amount
    error_message: "amount must be pounds with 2 decimal places, e.g. 12.50 - contact the finance team"
    logical_type:
      name: decimal
      args:
        precision: 10
        scale: 2
    representations:
      - pattern: "^£?(?P<value>[0-9]+\\.[0-9]{2})$"
```

A failing value is then reported as:

```
row 3, column 'amount': value '12.5' did not match any pattern (note: amount must be pounds with 2 decimal places, e.g. 12.50 - contact the finance team)
```

### Logical types and type args

`logical_type.name` is one of the types below. Some types take `args` under `logical_type`. These are settings that must be the same for every value in the column (such as decimal precision or the permitted values of an enum), so they live on the type rather than on a representation.

The config is checked strictly when it loads: an unknown key anywhere in it (including under `logical_type` or its `args`, e.g. a typo like `permited_values`) or a missing or conflicting type arg fails the run, and `validate_config` reports it with the field name. A representation's `args` are value args handed to the converter, and are checked against the logical type's value args (see [Representations](#representations)).

| Type | Type args | Avro output |
| --- | --- | --- |
| `string` | | `string` |
| `integer` | | `long` |
| `decimal` | `precision` and `scale`, or `as_float: true` | `bytes` with `decimal` logical type, or `double` if `as_float` |
| `enum` | `permitted_values` (required) | `enum` named after the field |
| `date` | | `int` with `date` logical type |
| `time` | | `long` with `time-micros` logical type |
| `timestamp` | | `long` with `timestamp-micros` logical type (UTC) |

A field with `nullable: true` is written as a union of `null` and the type above (e.g. `["null", "long"]`), with a default of `null`.

### Representations

Each value is tested against the field's representations in order, and the first one whose `pattern` matches is used. If none match, the cell is reported as an error. Patterns are Go regular expressions and are not anchored, so use `^` and `$` to match the whole value.

A representation produces a set of value args, which are handed to the type's converter:

- **Named capture groups** become args with the group's name, e.g. `(?P<year>[0-9]{4})` gives `year`.
- **A single unnamed capture group** is treated as `value`. A pattern with no capture groups at all makes the whole match `value`; types with no `value` arg (`date`, `time`) ignore it, so a fixed pattern like `^start$` can supply all its args statically.
- **Static `args`** on the representation add fixed values. If a static arg and a capture group share a name, the static arg wins. Integer-valued args (`integer`'s `value`, and the numeric date/time/timestamp parts) may be given as YAML numbers or strings, e.g. `args: {value: 0}`; a number with a fractional part is rejected.

The value args each type accepts:

| Type | Value args |
| --- | --- |
| `string`, `integer`, `enum` | `value` |
| `decimal` | `value`, or `integer_part` and `decimal_part` |
| `date` | `year`, `month`, `day` |
| `time` | `hour`, `minute`, `second`, `millisecond`, `microsecond` (all optional, default 0) |
| `timestamp` | `value` and `precision` (`seconds`, `milliseconds` or `microseconds`), with optional `offset`; or `year`, `month`, `day` with optional `hour`, `minute`, `second`, `millisecond`, `microsecond` and `timezone` (e.g. `+01:00`) |

When the config loads, each representation is checked against this table: the args from its capture groups and static `args` must form one of the type's arg sets, with nothing missing and no unknown names (e.g. a `date` pattern with a `yr` group fails). Static arg values are checked too where they can be on their own: integer-valued args must be whole numbers, an `enum` `value` must be in `permitted_values`, and a timestamp `precision` or `timezone` must be valid. A failed check names the field and pattern, and `validate_config` reports it. Representations with `is_null: true` are checked separately (below), since they provide no value args.

A representation with `is_null: true` makes a matching value null instead of handing args to a converter. It is only allowed on a field with `nullable: true`, and since it provides no value args it can't have static `args` or named capture groups (unnamed groups such as `^(-|N/A)$` are fine). Breaking either rule fails the config load, naming the field. Like any representation it is tried in order, so put it before a broader pattern that would also match the placeholder.

At run time the converters check that the args make a valid value (e.g. a real calendar date, or a captured enum value in `permitted_values`), and report a cell error if not.

## Planned

These are intended but not built yet. Work is tracked as tasks with [Backlog.md](https://github.com/MrLesk/Backlog.md) in `backlog/` (run `backlog board` or `backlog task list`).

- layered config files: a shared base config with per-producer overrides merged on top (`-c base.yaml -c partner.yaml`)
- streaming output to stdout, Parquet output, and a structured (json) `--error-report`
