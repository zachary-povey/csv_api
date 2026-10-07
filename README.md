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

The command exits non-zero and prints an error report if the file does not match the config.

## Configuration

A config lists the fields expected in the file. Each field has a logical type (what the value means) and one or more representations (how that value is written in the csv).

```yaml
fields:
  - name: some_date # must match the csv column header exactly
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

### Fields

`fields` is a list, and its order sets the order of fields in the output schema. A field is matched to a csv column by an exact header match on `name`. Columns in the file that are not in the config are ignored.

### Logical types and type args

`logical_type.name` is one of the types below. Some types take `args` under `logical_type`. These are settings that must be the same for every value in the column (such as decimal precision or the permitted values of an enum), so they live on the type rather than on a representation.

| Type | Type args | Avro output |
| --- | --- | --- |
| `string` | | `string` |
| `integer` | | `long` |
| `decimal` | `precision` and `scale`, or `as_float: true` | `bytes` with `decimal` logical type, or `double` if `as_float` |
| `enum` | `permitted_values` (required) | `enum` named after the field |
| `date` | | `int` with `date` logical type |
| `time` | | `long` with `time-micros` logical type |
| `timestamp` | | `long` with `timestamp-micros` logical type (UTC) |

### Representations

Each value is tested against the field's representations in order, and the first one whose `pattern` matches is used. If none match, the cell is reported as an error. Patterns are Go regular expressions and are not anchored, so use `^` and `$` to match the whole value.

A representation produces a set of value args, which are handed to the type's converter:

- **Named capture groups** become args with the group's name, e.g. `(?P<year>[0-9]{4})` gives `year`.
- **A single unnamed capture group** is treated as `value`. A pattern with no capture groups at all makes the whole match `value`.
- **Static `args`** on the representation add fixed values. If a static arg and a capture group share a name, the static arg wins.

The value args each type accepts:

| Type | Value args |
| --- | --- |
| `string`, `integer`, `enum` | `value` |
| `decimal` | `value`, or `integer_part` and `decimal_part` |
| `date` | `year`, `month`, `day` |
| `time` | `hour`, `minute`, `second`, `millisecond`, `microsecond` (all optional, default 0) |
| `timestamp` | `value` and `precision` (`seconds`, `milliseconds` or `microseconds`), with optional `offset`; or `year`, `month`, `day` with optional `hour`, `minute`, `second`, `millisecond`, `microsecond` and `timezone` (e.g. `+01:00`) |

The converters check that the args make a valid value (e.g. a real calendar date, an enum value in `permitted_values`), and report a cell error if not.

## Planned

These are intended but not built yet:

- `name`: a config-level name for the Avro schema (currently always `test_schema`)
- `header_patterns`: regex matching of csv headers to fields, instead of an exact match on `name`
- `allow_extra_fields`: option to reject columns not in the config (currently they are always ignored)
- `required`: per-field option to allow a column to be missing (currently every field is required)
- `nullable` and `is_null`: allow null values, with representations that resolve to null (`is_null` is accepted in config but has no effect yet)
- partial conversion: drop bad rows and carry on (currently processing stops at the first bad cell)
- checking at config load that every representation provides the args its type needs (currently this is only caught per cell at runtime)
- streaming output to stdout, Parquet output, and a structured (json) error report
