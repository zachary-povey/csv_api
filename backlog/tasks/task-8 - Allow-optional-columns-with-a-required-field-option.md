---
id: TASK-8
title: Allow optional columns with a required field option
status: Done
assignee:
  - '@claude'
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T10:58:00Z'
labels:
  - config
  - reader
dependencies:
  - TASK-7
references:
  - internal/reader/reader.go
priority: medium
type: feature
ordinal: 8000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Every configured field must be present in the file (`RequiredFieldNames` returns all fields). Some columns are optional in practice. The reader already has a path for a missing non-required column (it passes a nil value), but the parser would dereference that nil and crash. Depends on nullable support, since a missing column has to produce null.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A field with required: false may be missing from the file; its values are written as null
- [x] #2 required: false is only allowed on nullable fields (rejected at config load otherwise)
- [x] #3 Missing required fields are still a file-level error
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
1. Config: add optional 'required' key to FieldConfig (*bool, default true); IsRequired() helper; RequiredFieldNames returns only required fields.
2. Config load: reject required: false on a field without nullable: true, naming the field (field-level check in config.go).
3. Reader already passes nil for a missing non-required column and the parser writes nil as null; Avro writer emits null for nullable fields. Verify; no code change expected there.
4. Tests: fixtures for optional column missing (with extra columns and reordered columns), optional column present, data failure report with missing optional column, required:false on non-nullable (parse + validate_config), missing required field still a file-level error alongside an optional one. Each names its AC.
5. README: document 'required' in Fields, remove the Planned bullet; CLAUDE.md Configuration Format lists 'required' (and nullable).
6. Run pytest + go vet.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Added 'required' (*bool, default true) to FieldConfig with IsRequired(); RequiredFieldNames now only lists required fields; Config.checkRequired rejects required: false without nullable: true at load, naming the field. Reader/parser/avro writer needed no change: the reader already maps present columns by header name and passes nil for a missing optional column, the parser writes nil as null.
Tests: AC#1 test_missing_optional_column_is_written_as_null (reordered + extra columns), test_optional_column_present_is_parsed_normally, test_data_failure_report_with_missing_optional_column; AC#2 test_required_false_on_non_nullable_field_fails_parse, test_validate_config_rejects/accepts_required_false...; AC#3 test_missing_required_column_is_file_level_error_alongside_optional, test_missing_required_field. pytest 151 passed, go vet clean.
Out of scope, noticed: a header with the same column name twice silently maps the field to the last occurrence (reader getFieldPositions); may deserve a follow-up.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Fields can now set required: false to allow their column to be missing from the file; the field is then null in every row (Avro union with null). Fields stay required by default, a missing required field is still a file-level error, and required: false is only accepted on nullable fields (rejected at config load naming the field). README documents the option in Fields (Planned bullet removed); CLAUDE.md lists the key. Covered by new pytest fixtures for missing/present optional columns (including column reordering, extra columns and the data failure report) and config rejection. Full suite (151) and go vet pass. Follow-up idea: duplicate header names silently use the last column.
<!-- SECTION:FINAL_SUMMARY:END -->
