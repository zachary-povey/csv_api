---
id: TASK-15
title: Add boolean logical type
status: To Do
assignee: []
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T10:41:00Z'
labels:
  - types
dependencies: []
priority: low
type: feature
ordinal: 15000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The original planned type list included bool, which was never built. Producers write booleans in many ways (Y/N, true/false, 1/0), which representations with static args can map.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A boolean logical type converts a value arg to true/false and maps to Avro boolean
- [ ] #2 Static args on representations can map arbitrary strings to true/false
- [ ] #3 README type tables and tests updated (see CLAUDE.md "Adding New Logical Types")
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Full test suite passes (python3 -m pytest tests)
- [ ] #2 go vet ./... is clean
- [ ] #3 Every acceptance criterion has a test that names it (e.g. TASK-12 AC#2)
- [ ] #4 README updated for any user-facing change
<!-- DOD:END -->
