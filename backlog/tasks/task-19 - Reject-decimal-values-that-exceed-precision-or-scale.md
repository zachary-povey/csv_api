---
id: TASK-19
title: Reject decimal values that exceed precision or scale
status: To Do
assignee: []
created_date: '2026-11-01T11:32:00Z'
labels:
  - parser
dependencies: []
priority: high
type: bug
ordinal: 19000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Found while reviewing TASK-13. Decimal values are not checked against the field's precision and scale. With precision 4, scale 2, the value 1.234 is silently written as 1.23 and 123456.7 is written as 1.235E+5 (a corrupted value), and the run exits 0. This breaks design principle 5 (never lose data silently): the converter must check that its args form a valid value for the type (principle 7), so a value that does not fit is a data failure like any other.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Given a decimal field with precision p and scale s, when a value has more fractional digits than s, then it is a data failure naming the row, column, value and scale
- [ ] #2 Given a decimal field with precision p and scale s, when a value has more than p-s integer digits, then it is a data failure naming the row, column, value and precision
- [ ] #3 Given values that fit (including fewer fractional digits than s, negatives, and leading zeros), when parsed, then they are written exactly in both Avro and Parquet output
- [ ] #4 Given a static decimal value arg that does not fit precision/scale, when the config loads, then it fails naming the field and pattern
- [ ] #5 Given an as_float decimal, when parsed, then behaviour is unchanged (no precision/scale check)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Full test suite passes (python3 -m pytest tests)
- [ ] #2 go vet ./... is clean
- [ ] #3 Every acceptance criterion has a test that names it (e.g. TASK-12 AC#2)
- [ ] #4 README updated for any user-facing change
<!-- DOD:END -->
