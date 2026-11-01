---
id: TASK-11
title: Add per-field error messages to config
status: To Do
assignee: []
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T10:41:00Z'
labels:
  - config
  - errors
dependencies: []
priority: medium
type: feature
ordinal: 11000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Error reports describe failures in terms of patterns and converters, which means little to the producer who has to fix the data. A config author should be able to attach a plain-language message to a field (e.g. "amount must be pounds with 2 decimal places, e.g. 12.50 - contact the finance team") that is shown when that field fails. This is the "custom errors" item from the original scope.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A field can have an optional error message in the config
- [ ] #2 The message is included with that field's errors in the console report and in the data failure report column
- [ ] #3 Fields without a message report as they do now
- [ ] #4 README and tests updated
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Full test suite passes (python3 -m pytest tests)
- [ ] #2 go vet ./... is clean
- [ ] #3 Every acceptance criterion has a test that names it (e.g. TASK-12 AC#2)
- [ ] #4 README updated for any user-facing change
<!-- DOD:END -->
