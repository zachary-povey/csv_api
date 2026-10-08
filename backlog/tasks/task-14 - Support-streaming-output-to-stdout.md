---
id: TASK-14
title: Support streaming output to stdout
status: To Do
assignee: []
created_date: '2026-10-08 18:05'
labels:
  - output
  - cli
dependencies: []
priority: low
type: feature
ordinal: 14000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The old notes say output should default to streaming to stdout, so the tool can sit in a shell pipeline. Today --output-path is required. Note that streaming conflicts with "a failed run leaves no output", since data is emitted before the whole file is validated; that trade-off needs deciding.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 parse can write output to stdout
- [ ] #2 Behaviour on failure when streaming is decided and documented
- [ ] #3 Errors and warnings go to stderr only so stdout stays clean
<!-- AC:END -->
