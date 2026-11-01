---
id: TASK-17
title: Check field names are valid, unique Avro names at config load
status: To Do
assignee: []
created_date: '2026-11-01T10:53:00Z'
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
- [ ] #1 Given a field whose name is not a valid Avro name, when the config is loaded, then it fails naming the field and pointing at header_patterns
- [ ] #2 Given two fields with the same name, when the config is loaded, then it fails naming the duplicate
- [ ] #3 Given validate_config on such a config, when it runs, then it reports these errors
- [ ] #4 Given the README, when a user reads the Fields section, then it states that field names must be valid, unique Avro names
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Full test suite passes (python3 -m pytest tests)
- [ ] #2 go vet ./... is clean
- [ ] #3 Every acceptance criterion has a test that names it (e.g. TASK-12 AC#2)
- [ ] #4 README updated for any user-facing change
<!-- DOD:END -->
