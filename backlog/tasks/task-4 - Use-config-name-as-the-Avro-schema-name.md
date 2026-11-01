---
id: TASK-4
title: Use config name as the Avro schema name
status: Done
assignee:
  - '@claude'
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T10:52:00Z'
labels:
  - config
  - avro
dependencies: []
priority: medium
type: feature
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The Avro schema name is hard-coded to `test_schema`. The original design had a top-level `name` in the config so the output schema carries a meaningful name for downstream consumers.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A top-level `name` in the config is used as the Avro record name
- [x] #2 Config load rejects a name that is not a valid Avro name
- [x] #3 Behaviour when name is omitted is decided and documented (default or required)
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
1. Add required top-level `name` to Config; validate at load against Avro fullname rules (dot-separated segments, each [A-Za-z_][A-Za-z0-9_]*), clear error messages for missing/invalid.
2. Avro writer uses config name as the record name (replace hard-coded test_schema); pass Config to the template.
3. Add `name` to every fixture config and examples/ config mechanically (fixture dir name as the schema name).
4. Tests: pytest reading writer_schema name (AC#1), invalid names rejected by parse and validate_config (AC#2), missing name rejected (AC#3); Go unit test for the name checker.
5. README: document `name` in Configuration, remove Planned bullet, update example config.
6. Record decisions (required name; dotted fullname allowed) with backlog decision create.
7. Check field names producing goavro errors; note follow-up if so.

8. Reject an enum field named like the record (found during testing: produces a file readers reject).
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Decision decision-2 (backlog decision create has no body flag, so context/decision/consequences recorded here):
- Context: AC#3 left default-vs-required open. Name is part of the output contract downstream consumers rely on; a silent default (like test_schema) is the meaningless name this task removes.
- Decision: `name` is required (design principle 3, strict by default). Validated at load against Avro fullname rules: one or more dot-separated segments, each [A-Za-z_][A-Za-z0-9_]*. Dotted fullnames are allowed (Avro spec permits them; leading parts become the namespace, useful for registries/codegen).
- Consequences: every existing config needs a `name`; all tests/fixtures/*/config.yaml got name = fixture dir name, examples/*.yaml got name = file stem. Enum types (named after their field) inherit the record namespace.

Extra check added (small, introduced by this task): an enum field with the same name as the last segment of the config name redefines the record's named type. goavro writes it without complaint but fastavro rejects the file ('redefined named type'), so it is rejected at config load (Config.checkEnumNames, fixture config_name_enum_clash).

Follow-up (not fixed, outside ACs): field names are not checked against Avro name rules at load. A field like 'first name' passes validate_config but parse fails at writer setup with a goavro error ('Record "t" field 1 ought to have valid name'). Also duplicate field names are not checked. Both belong in a load-time check (design principle 6). Note header matching is on `name`, so fixing this needs a decision on how non-Avro header names map to Avro field names (possibly with header_patterns / an alias).
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
The config now has a required top-level `name` used as the Avro record name (replacing hard-coded test_schema). It is validated at load (parse and validate_config) against Avro fullname rules; dotted names set the namespace. An enum field named like the record is rejected because it yields a file readers reject. All fixtures and examples got a name. README documents `name` under Configuration (Planned bullet removed, example updated). Tests: test_config_name_is_avro_schema_name / test_dotted_config_name_sets_avro_namespace (AC#1, read fastavro writer_schema), test_invalid_config_name_fails_parse / test_validate_config_reports_invalid_name / test_enum_field_named_like_config_fails / Go TestCheckAvroName (AC#2), test_missing_config_name_fails (AC#3, decision-2). Full pytest (127 passed) and go vet clean. Follow-up: field names aren't validated as Avro names at load (goavro error at writer setup).
<!-- SECTION:FINAL_SUMMARY:END -->
