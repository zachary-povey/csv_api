---
id: TASK-5
title: Match csv headers to fields with header_patterns
status: To Do
assignee: []
created_date: '2026-10-08 18:05'
labels:
  - config
  - reader
dependencies: []
priority: medium
type: feature
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fields are matched to columns by an exact header match on `name`. Producers often vary header text (case, spacing, suffixes) between exports, which is a common source of drift. The original design had per-field `header_patterns` (regexes); it is also on the old notes to-do list.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A field can list header_patterns; a column whose header matches any of them is mapped to the field
- [ ] #2 Without header_patterns, the exact match on name still applies
- [ ] #3 A header matching more than one field, or a field matching more than one header, is a file-level error
- [ ] #4 README and tests updated
<!-- AC:END -->
