---
id: TASK-4
title: Use config name as the Avro schema name
status: To Do
assignee: []
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T10:41:00Z'
labels:
  - config
  - avro
dependencies: []
priority: medium
type: feature
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The Avro schema name is hard-coded to `test_schema`. The original design had a top-level `name` in the config so the output schema carries a meaningful name for downstream consumers.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A top-level `name` in the config is used as the Avro record name
- [ ] #2 Config load rejects a name that is not a valid Avro name
- [ ] #3 Behaviour when name is omitted is decided and documented (default or required)
- [ ] #4 README and tests updated
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Full test suite passes (python3 -m pytest tests)
- [ ] #2 go vet ./... is clean
- [ ] #3 Every acceptance criterion has a test that names it (e.g. TASK-12 AC#2)
- [ ] #4 README updated for any user-facing change
<!-- DOD:END -->
