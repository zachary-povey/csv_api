---
id: TASK-13
title: Add Parquet output
status: To Do
assignee: []
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T10:41:00Z'
labels:
  - output
dependencies: []
priority: low
type: feature
ordinal: 13000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Output is Avro only. The original design and notes targeted Parquet, which many downstream analytics tools prefer. Avro became the first output format during implementation.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 parse can write Parquet with logical types mapped equivalently to the Avro output
- [ ] #2 The output format is selectable and documented
- [ ] #3 Tests read the Parquet output and check values for every logical type
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Full test suite passes (python3 -m pytest tests)
- [ ] #2 go vet ./... is clean
- [ ] #3 Every acceptance criterion has a test that names it (e.g. TASK-12 AC#2)
- [ ] #4 README updated for any user-facing change
<!-- DOD:END -->
