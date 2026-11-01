---
id: TASK-17
title: Check field names are valid, unique Avro names at config load
status: Done
assignee:
  - '@claude'
created_date: '2026-11-01T10:53:00Z'
updated_date: '2026-11-01T11:04:00Z'
labels:
  - config
  - avro
dependencies:
  - TASK-5
priority: medium
type: bug
ordinal: 17000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Found while implementing TASK-4. A field `name` is used as the Avro field name (and as the Avro enum type name for enum fields), but it is not checked at load. A field named `first name` passes `validate_config`, then `parse` fails when the Avro writer is created with goavro's "ought to have valid name" error. Duplicate field names are not checked either. Design principle 6 says config mistakes should be caught at load. Today a field must be named exactly as its csv header, so rejecting invalid Avro names only becomes reasonable once header_patterns (TASK-5) lets a field with a valid name be matched to a header like `first name`.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Given a field whose name is not a valid Avro name, when the config is loaded, then it fails naming the field and pointing at header_patterns
- [x] #2 Given two fields with the same name, when the config is loaded, then it fails naming the duplicate
- [x] #3 Given validate_config on such a config, when it runs, then it reports these errors
- [x] #4 Given the README, when a user reads the Fields section, then it states that field names must be valid, unique Avro names
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
1. Add config.checkFieldNames (called in LoadConfig right after struct validation): each field name must match the existing avroNameSegment regex (no dots; field names are not fullnames), else fail naming the field and suggesting header_patterns to match the csv header; empty name gets a 'name is required' error; second occurrence of a name fails naming the duplicate.
2. Go unit test for the name rule; pytest fixtures config_field_name_invalid / config_field_name_duplicate with parse + validate_config tests naming TASK-17 ACs.
3. README Fields section states the rule; README test for AC#4.
4. Check existing fixtures/examples (all already valid). Run pytest + go vet.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Tests: internal/config/config_test.go TestCheckFieldName (AC#1), tests/test_config_validation.py test_invalid_field_name_fails_parse (AC#1), test_duplicate_field_name_fails_parse (AC#2), test_validate_config_reports_field_name_errors (AC#3), test_readme_states_field_name_rule (AC#4). test_existing_fixtures_still_load confirms no existing fixture relied on an invalid/duplicate name; examples/*.yaml validate_config results unchanged. Check runs right after struct validation, so it fails before type/representation checks. First error only (like the other load checks). Duplicate check is case-sensitive, matching Avro. Two enum fields with the same name are already caught as duplicates. pytest 170 passed, go vet clean.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Config load now checks every field name (config.checkFieldNames / CheckFieldName in internal/config/config.go): it must match Avro's name rule [A-Za-z_][A-Za-z0-9_]* (reusing avroNameSegment; no dots), with an error naming the field and suggesting header_patterns to match a header like 'first name'; an empty name gets a 'missing name' error; a repeated name fails as 'duplicate field name'. Both parse and validate_config report these before any run. README Fields section states the rule. New fixtures config_field_name_invalid / config_field_name_duplicate; Go unit test + pytest cover AC#1-4. Full pytest (170 passed) and go vet clean.
<!-- SECTION:FINAL_SUMMARY:END -->
