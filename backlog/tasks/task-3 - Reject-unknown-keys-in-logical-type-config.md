---
id: TASK-3
title: Reject unknown keys in logical type config
status: To Do
assignee: []
created_date: '2026-10-08 18:05'
labels:
  - config
dependencies: []
references:
  - internal/config/config.go
priority: medium
type: bug
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Unknown keys under `logical_type` (e.g. a typo like `permited_values`) are silently ignored, because the mapstructure decode in `narrowType` allows extras (there is a "todo: validate no extras" comment). A typo in the agreed contract should be caught at config load. Also, `LoadConfig` calls `validate.Struct` on each narrowed type config but ignores the result.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A config with an unknown key under logical_type or its args fails to load with an error naming the key
- [ ] #2 Validation errors on narrowed type configs are no longer ignored
- [ ] #3 `validate_config` reports these errors
<!-- AC:END -->
