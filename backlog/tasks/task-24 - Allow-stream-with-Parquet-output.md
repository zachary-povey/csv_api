---
id: TASK-24
title: Allow --stream with Parquet output
status: To Do
assignee: []
created_date: '2026-11-01T17:54:00Z'
labels:
  - output
  - cli
dependencies:
  - TASK-23
priority: medium
type: feature
ordinal: 24000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Agreed with the user, replacing decision-26 (--stream rejected with --output-format parquet). The user's main use case is piping Parquet output to another tool (e.g. `| aws s3 cp - s3://...`). Parquet writes row groups sequentially and its footer last, and parquet-go can write to a non-seekable io.Writer, so streaming to stdout works: the consumer gets a complete file when the run succeeds. Readers cannot use a Parquet file before its footer arrives, so streaming does not give early access to rows; its value is pipe-friendliness and bounded memory. On failure the output has no footer, so Parquet readers reject it, which is a stronger signal than truncated Avro; the non-zero exit code remains the signal consumers must check (TASK-14).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Given --stream --output-format parquet and no --output-path, when parse runs on a valid file, then a complete, readable Parquet file is written to stdout with the same values as a non-streamed run
- [ ] #2 Given --stream --output-format parquet and an invalid file, when parse runs, then it exits non-zero, stderr says the output is incomplete, and nothing after the failure is written as a footer
- [ ] #3 Given --stream --output-format parquet with --output-path, when parse runs, then that path is written directly as with Avro streaming (decision-21)
- [ ] #4 Given --stream --output-format parquet and an input larger than one row group, when parse runs, then completed row groups are written to the output before the input has finished
- [ ] #5 Given --stream --output-format parquet, when parse runs, then stdout carries only Parquet bytes (errors and warnings go to stderr)
- [ ] #6 Given the README, when a user reads the streaming and output format docs, then Parquet streaming is documented, including that a Parquet file is only readable once complete
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Full test suite passes (python3 -m pytest tests)
- [ ] #2 go vet ./... is clean
- [ ] #3 Every acceptance criterion has a test that names it (e.g. TASK-12 AC#2)
- [ ] #4 README updated for any user-facing change
<!-- DOD:END -->
