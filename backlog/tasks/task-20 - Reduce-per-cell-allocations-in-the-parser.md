---
id: TASK-20
title: Reduce per-cell allocations in the parser
status: To Do
assignee: []
created_date: '2026-11-01T11:38:00Z'
labels:
  - performance
dependencies: []
references:
  - scripts/benchmark.py
  - internal/parser/parser.go
priority: medium
type: enhancement
ordinal: 20000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
TASK-16 measured ~55k rows/s on a 1M-row, 9-column file (scripts/benchmark.py) with a 4-cpu machine. A CPU profile shows the single parser goroutine is the bottleneck (~94% of wall time) and most of its time is allocation and GC rather than real work: per cell, extract_args builds a new map[string]string (calling SubexpNames/SubexpIndex each time) and FindStringSubmatch allocates a slice; parseValue then copies it into a second map[string]any and merges the static args; ParseData builds a new map per row. extract_args + map work was ~70% of parser time and ~0.9 GB/0.3M rows allocated. Converting decimals via big.Rat (config.DecimalArg, pow10) is the next largest cost. Hot paths only; behaviour and error messages must not change.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Given the 1M-row file from scripts/benchmark.py, when it is converted to Avro on the same machine as the TASK-16 baseline (~55k rows/s), then throughput is at least 2x the baseline, with the new figure recorded in task notes and README "Performance"
- [ ] #2 Given the existing test suite, when the parser changes are made, then every test passes unchanged (same output values and error messages)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Full test suite passes (python3 -m pytest tests)
- [ ] #2 go vet ./... is clean
- [ ] #3 Every acceptance criterion has a test that names it (e.g. TASK-12 AC#2)
- [ ] #4 README updated for any user-facing change
<!-- DOD:END -->
