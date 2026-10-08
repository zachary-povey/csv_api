---
id: TASK-13
title: Add Parquet output
status: To Do
assignee: []
created_date: '2026-10-08 18:05'
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
