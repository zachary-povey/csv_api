---
id: TASK-10
title: Support layered config files
status: To Do
assignee: []
created_date: '2026-10-08 18:05'
labels:
  - config
  - cli
dependencies: []
priority: medium
type: feature
ordinal: 10000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
One data contract is often received from several producers with small differences (a header name, an extra date format). Rather than copy the whole config per producer, a shared base config should be combined with a small per-producer file, e.g. `-c base.yaml -c partner_a.yaml`, later files merged on top of earlier ones. Agreed design direction; merge semantics (fields merged by name, how representations lists combine) still need to be decided and documented. These are config files only; run-time error handling options stay CLI flags.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `-c` can be given more than once on parse and validate_config
- [ ] #2 Later files override earlier ones, with merge rules for fields and representations documented in the README
- [ ] #3 The merged config is validated as a whole
- [ ] #4 Tests cover overriding a field and adding a representation
<!-- AC:END -->
