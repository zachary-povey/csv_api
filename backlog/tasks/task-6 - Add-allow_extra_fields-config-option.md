---
id: TASK-6
title: Add allow_extra_fields config option
status: To Do
assignee: []
created_date: '2026-10-08 18:05'
labels:
  - config
  - reader
dependencies: []
priority: low
type: feature
ordinal: 6000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Columns not in the config are always ignored. Some contracts want to treat an unexpected column as a sign the producer changed the format. The original design had a top-level `allow_extra_fields` setting.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 With allow_extra_fields: false, a column not mapped to any field is a file-level error naming the column
- [ ] #2 The default keeps the current behaviour (extra columns ignored)
- [ ] #3 README and tests updated
<!-- AC:END -->
