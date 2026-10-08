---
id: TASK-12
title: Add JSON error report
status: To Do
assignee: []
created_date: '2026-10-08 18:05'
labels:
  - errors
  - cli
dependencies: []
references:
  - internal/options/options.go
priority: medium
type: feature
ordinal: 12000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Errors are only reported to the console as text, which is hard for other systems to consume (e.g. raising a ticket or alerting the producer automatically). A structured JSON report was on the original notes to-do list; `--error-report` exists with only `console` supported so this can be added as a new value.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `--error-report` accepts a JSON option that writes errors to a file
- [ ] #2 Each error records its group (file, row, data), row number, column, raw value and reason where applicable
- [ ] #3 The report records whether processing stopped early and why
- [ ] #4 README and tests updated
<!-- AC:END -->
