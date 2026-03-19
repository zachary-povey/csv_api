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

Scope:

- to formally specify the bespoke physical representation of logical types agreed by the csv API
- to validate input files match this specification
- to convert input files to a file type with a standard physical representation of the same logical types
- command line config overrides
- partial conversion (bad rows dropped)
- error reports
- custom errors
  Out of scope
- validation of logical values
- combinations of fields etc
- capturing contextual information (e.g when a file was received, allow use cases to be sorted via config overrides)

### Notes

At some point, the data will have to be converted, do this first formally. If the expectations are not met, errors would have occurred anyway

### Example config

```yaml
name: MyDesciptiveName # used as avro schema name
allow_extra_fields: true
fields:
  some_date:
    header_patterns:
      - ^SomeDate$
    logical_type: date
    required: true # if it should be present in the file
    nullable: true # if the value of the reolved logical type can be null
    physical_representations:
      - pattern: "-'"
        is_null: true
      - pattern: "start"
        args:
          year: 2020
          month: 1
          day: 1
      - pattern: "[0-9]{4}-[0-9]{2}-[0-9]{2}"
```
