---
id: TASK-8
title: Allow optional columns with a required field option
status: To Do
assignee: []
created_date: '2026-10-08 18:05'
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
- [ ] #1 A field with required: false may be missing from the file; its values are written as null
- [ ] #2 required: false is only allowed on nullable fields (rejected at config load otherwise)
- [ ] #3 Missing required fields are still a file-level error
- [ ] #4 README and tests updated
<!-- AC:END -->
