---
id: TASK-7
title: Support nullable fields and is_null representations
status: Done
assignee:
  - '@claude'
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T10:56:00Z'
labels:
  - config
  - parser
  - avro
dependencies: []
priority: high
type: feature
ordinal: 7000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
There is no way to express null values. Producers commonly write nulls as placeholders like "-", "N/A" or an empty string, and today those values either fail validation or have to be matched as strings. `Representation.IsNull` is already parsed from config but never used. The original design had a per-field `nullable` flag and `is_null: true` on representations.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A field with nullable: true is written to the Avro schema as a union with null
- [x] #2 A value matching a representation with is_null: true is written as null
- [x] #3 An is_null representation on a field that is not nullable is rejected at config load
- [x] #4 README and tests updated
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
1. Config: add FieldConfig.Nullable (yaml 'nullable', default false). In checkRepresentations reject an is_null rep on a non-nullable field (AC#3) and reject static args / named capture groups on an is_null rep (it produces no value args).
2. Parser: carry isNull on compiled reps; a matching is_null rep resolves the cell to nil. Add a nil-cell path in ParseData (missing column -> null) for TASK-8 to reuse. Empty cells get no special treatment: unmatched -> data failure.
3. Avro writer: nullable field type becomes ["null", <type>] with default null; wrap non-null values for goavro's union encoding using the branch name per logical type (long, string, double, bytes.decimal, int.date, long.time-micros, long.timestamp-micros, enum fullname incl. namespace).
4. Tests: pytest fixtures for every logical type nullable (null + non-null, read with fastavro, schema union check), namespaced enum, config rejections, empty-cell-unmatched, dropped rows/report with nulls; Go unit tests where useful.
5. README: move nullable/is_null out of Planned into Fields/Representations/Avro output docs. Record decisions with backlog decision create.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Decisions: decision-3 (empty cell on nullable field without a matching rep is a data failure; nulls only from is_null reps), decision-4 (is_null reps reject static args and named capture groups), decision-5 (nullable default false; union [null, T] default null; goavro union wrapping lives in the avro writer).
For TASK-8: ParseData already treats a nil cell (missing non-required column from the reader) as null; TASK-8 only needs the required option + config check + RequiredFieldNames change.
Fixture rep_args_all_sets gained nullable: true on field d, since its is_null rep is now only allowed on nullable fields.
Tests: AC#1 tests/test_logical_types.py::test_nullable_fields_are_unions_with_null (+ test_nullable_enum_without_namespace); AC#2 test_nullable_fields_write_null_and_non_null_values, test_nullable_enum_without_namespace, tests/test_error_handling.py::test_dropped_rows_and_report_with_nullable_fields; AC#3 tests/test_config_validation.py::test_is_null_on_non_nullable_field_fails_parse and test_validate_config_rejects_bad_is_null_representations; AC#4 README Fields/Representations/Avro table updated, Planned bullet removed.
Verification: python3 -m pytest tests -> 140 passed; go vet ./... clean.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added per-field nullable: true (default false). Nullable fields are written to Avro as ["null", <type>] with default null; the avro writer wraps non-null values in goavro's union form using the right branch name for each logical type (long, string, double, bytes.decimal, int.date, long.time-micros, long.timestamp-micros, enum fullname incl. config namespace). Representations with is_null: true resolve a matching cell to null; at config load they're rejected on non-nullable fields (naming the field) and when they carry static args or named groups. Empty cells get no special meaning (decision-3). The parser also treats a nil cell as null, ready for TASK-8. README documents nullable/is_null and the Planned bullet is gone. Tests cover every logical type nullable with null and non-null values (read with fastavro), namespaced and plain enums, config rejections via parse and validate_config, and dropped rows/report with nulls. pytest 140 passed, go vet clean.
<!-- SECTION:FINAL_SUMMARY:END -->
