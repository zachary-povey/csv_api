---
id: TASK-2
title: Accept numeric static args for integer fields
status: To Do
assignee: []
created_date: '2026-10-08 18:05'
labels:
  - parser
dependencies: []
references:
  - internal/parser/converters.go
priority: medium
type: bug
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Static `args` in YAML can be numbers, but `convert_int` only accepts strings, so a representation like pattern `^none$` with `args: {value: 0}` fails every matching cell with "value in args is not of type string". Other converters (date, time, timestamp) already accept both via `argToInt`.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 An integer field with a numeric static value arg converts correctly
- [ ] #2 String values from capture groups still convert as before
- [ ] #3 A fixture test covers a numeric static arg on an integer field
<!-- AC:END -->
