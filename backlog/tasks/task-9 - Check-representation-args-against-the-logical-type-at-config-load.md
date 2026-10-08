---
id: TASK-9
title: Check representation args against the logical type at config load
status: To Do
assignee: []
created_date: '2026-10-08 18:05'
labels:
  - config
dependencies: []
priority: medium
type: feature
ordinal: 9000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
If a representation cannot supply the args its logical type needs (e.g. a date pattern with no year/month/day groups or static args), the mistake only shows up at runtime as a conversion error on every cell. Since named groups and static args are known statically, the config can be checked up front. This was an unfinished item in the original "Add type validation" notes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A representation whose named groups plus static args do not form a valid arg set for its logical type fails config load, naming the field and pattern
- [ ] #2 Each type accepts all of its documented arg sets (e.g. decimal value vs integer_part+decimal_part, timestamp epoch vs components)
- [ ] #3 `validate_config` reports these errors
- [ ] #4 Existing test fixtures still load
<!-- AC:END -->
