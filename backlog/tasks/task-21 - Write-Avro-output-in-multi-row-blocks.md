---
id: TASK-21
title: Write Avro output in multi-row blocks
status: To Do
assignee: []
created_date: '2026-11-01T11:38:00Z'
labels:
  - performance
dependencies: []
references:
  - internal/avro_writer/avro_writer.go
  - tests/test_streaming.py
priority: medium
type: enhancement
ordinal: 21000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
avro_writer.Write calls goavro OCFWriter.Append once per row, and goavro writes one OCF block (count, size, 16-byte sync marker) per Append call, straight to the unbuffered output file: one write syscall per row. TASK-16 found this makes the Avro output ~38% larger than the same records written in normal-sized blocks (19.6 MB vs 14.3 MB for 300k rows re-encoded with fastavro) and costs a syscall per row. Per-row blocks were chosen so --stream output reaches its reader as rows are validated (TASK-14), so batching must keep streamed output timely.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Given a run without --stream, when a file of many rows is converted to Avro, then rows are written in multi-row blocks and the output for the scripts/benchmark.py 1M-row file is at least 25% smaller than the TASK-16 baseline (65 MB) with identical records
- [ ] #2 Given --stream, when rows are validated, then each row still reaches the output reader without waiting for further input rows (e.g. a block is flushed whenever the writer would otherwise wait for input)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Full test suite passes (python3 -m pytest tests)
- [ ] #2 go vet ./... is clean
- [ ] #3 Every acceptance criterion has a test that names it (e.g. TASK-12 AC#2)
- [ ] #4 README updated for any user-facing change
<!-- DOD:END -->
