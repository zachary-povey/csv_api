---
id: TASK-7
title: Support nullable fields and is_null representations
status: To Do
assignee: []
created_date: '2026-10-08 18:05'
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
- [ ] #1 A field with nullable: true is written to the Avro schema as a union with null
- [ ] #2 A value matching a representation with is_null: true is written as null
- [ ] #3 An is_null representation on a field that is not nullable is rejected at config load
- [ ] #4 README and tests updated
<!-- AC:END -->
