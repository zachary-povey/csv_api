---
id: TASK-13
title: Add Parquet output
status: Done
assignee:
  - '@claude'
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T11:31:00Z'
labels:
  - output
dependencies: []
priority: low
type: feature
ordinal: 13000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Output is Avro only. The original design and notes targeted Parquet, which many downstream analytics tools prefer. Avro became the first output format during implementation.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 parse can write Parquet with logical types mapped equivalently to the Avro output
- [x] #2 The output format is selectable and documented
- [x] #3 Tests read the Parquet output and check values for every logical type
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Full test suite passes (python3 -m pytest tests)
- [x] #2 go vet ./... is clean
- [x] #3 Every acceptance criterion has a test that names it (e.g. TASK-12 AC#2)
- [x] #4 README updated for any user-facing change
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add --output-format avro|parquet flag (default avro) to parse; validate in options (reject unknown values and --stream with parquet).
2. New package internal/parquet_writer with Write(io.Writer, config, channel, wg, errTracker) mirroring avro_writer; schema built from config in field order (ordered group node), logical types mapped like Avro; nullable -> OPTIONAL else REQUIRED.
3. main.go picks the writer by format; temp file + rename unchanged so failed runs leave nothing.
4. Use github.com/parquet-go/parquet-go (pure Go, maintained; already in go.mod but unused) at latest version.
5. Tests (pyarrow): every logical type values + Parquet logical types, nullable nulls, decimal exactness, same fixture through both formats compared, failed run leaves no output, --stream rejection, flag docs.
6. README: Parquet column in type table, flag docs, remove Planned section; CLAUDE.md architecture + new-type steps.
7. Record decisions (flag vs extension, --stream rejected, time isAdjustedToUTC=false, enum values not enforced).
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Decisions:
- decision-23: format chosen with --output-format avro|parquet (default avro). Output format is the consumer's run-time choice (principle 1), so a flag; not inferred from the extension to keep it explicit and keep existing runs (any extension) on Avro. Unknown values rejected at option validation.
- decision-26: --stream --output-format parquet rejected at option validation. Parquet's footer (schema + row group index) is written at the end, so a partial file is unreadable; streaming has no value.
- decision-24: mapping string->BYTE_ARRAY STRING; integer->INT64 INT(64,signed); decimal->BYTE_ARRAY DECIMAL(p,s) (two's complement unscaled, same as Avro bytes; digits beyond scale dropped the same way as goavro), as_float->DOUBLE; enum->BYTE_ARRAY ENUM (as parquet-avro maps Avro enums; Parquet can't record permitted values; pyarrow reads ENUM as binary - documented); date->INT32 DATE; time->INT64 TIME(MICROS, isAdjustedToUTC=false) = local time like Avro time-micros; timestamp->INT64 TIMESTAMP(MICROS, isAdjustedToUTC=true); boolean->BOOLEAN; nullable->OPTIONAL else REQUIRED. Columns keep config order (custom ordered group node; parquet.Group sorts by name).
- decision-25: github.com/parquet-go/parquet-go v0.32.0 (pure Go, maintained; v0.24.0 was already an unused require). Raises go directive 1.23.3 -> 1.24.9 (GOTOOLCHAIN=auto downloads it). Row groups capped at 100k rows to bound memory.
Tests: tests/test_parquet_output.py (fixtures parquet_all_types, parquet_data_failure). Adjusted tests/test_streaming.py::test_readme_documents_stream to allow the Planned section to be gone. README: Output format section, Parquet column in type table, Planned replaced with Roadmap. CLAUDE.md updated.
Follow-ups (out of scope): decimal precision isn't enforced by the parser and extra scale digits are silently truncated in both Avro and Parquet output (pre-existing goavro behaviour, mirrored).
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added Parquet output: parse --output-format parquet (default avro) writes Parquet via new internal/parquet_writer (same Write signature as avro_writer; main.go picks one). Logical types map to the Parquet equivalents of the Avro output (README type table has a Parquet column); nullable -> OPTIONAL, columns in config order. --stream with parquet and unknown formats are rejected at option validation. Temp file + rename unchanged, so failed parquet runs leave nothing. Tests (tests/test_parquet_output.py, pyarrow) check schema logical types and values for every type incl. nulls, exact decimals, micro times and UTC timestamps, compare Avro vs Parquet values, failed-run cleanup, --stream rejection, docs. Full pytest (256) and go vet pass. Risk: go directive raised to 1.24.9 by parquet-go v0.32.0; decimal precision/scale overflow not enforced (pre-existing).
<!-- SECTION:FINAL_SUMMARY:END -->
