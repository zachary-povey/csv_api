---
id: TASK-22
title: Parse rows on multiple cores
status: To Do
assignee: []
created_date: '2026-11-01T11:38:00Z'
updated_date: '2026-11-01T11:38:00Z'
labels:
  - performance
dependencies:
  - TASK-20
references:
  - main.go
  - internal/parser/parser.go
priority: low
type: enhancement
ordinal: 22000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
TASK-16 found the pipeline is limited by one parser goroutine (reader and writer each use well under one core while the parser is saturated), so on a multi-core machine most cores sit idle. Spreading validation and conversion over several workers could multiply throughput, but rows must keep their input order in the output and error reporting (row numbers, fail-fast, max-errors, the data failure report) must behave as today. Best done after the per-cell allocation work so the gain is measured on an efficient single-worker baseline.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Given a machine with 4 or more cpus and the scripts/benchmark.py 1M-row file, when it is converted, then throughput is at least 2x the single-worker throughput measured just before this change
- [ ] #2 Given any input, when it is converted, then output rows are in input order and the error report, fail-fast, --max-errors and data failure report behave exactly as before (existing tests pass unchanged)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Full test suite passes (python3 -m pytest tests)
- [ ] #2 go vet ./... is clean
- [ ] #3 Every acceptance criterion has a test that names it (e.g. TASK-12 AC#2)
- [ ] #4 README updated for any user-facing change
<!-- DOD:END -->
