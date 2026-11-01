---
id: TASK-14
title: Support streaming output to stdout
status: To Do
assignee: []
created_date: '2026-10-08 18:05'
updated_date: '2026-11-01 10:16'
labels:
  - output
  - cli
dependencies: []
priority: low
type: feature
ordinal: 14000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The old notes say output should be able to stream to stdout, so the tool can sit in a shell pipeline. By default a run writes output to a temporary file and only moves it into place on success, so a failed run leaves no output. Streaming cannot work that way, because data is emitted before the whole file has been validated.

Decision: support both behaviours, chosen with a `--stream` flag. Without it, behaviour stays as it is now (all or nothing). With it, output is written as rows are validated, and on failure the output is incomplete; the non-zero exit code is the signal consumers must check.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Given --stream and no --output-path, when parse runs, then the output is written to stdout as rows are processed
- [ ] #2 Given --stream and an invalid file, when parse runs, then it exits non-zero and stderr says the output written is incomplete
- [ ] #3 Given no --stream, when the run fails, then no output is left behind (current behaviour unchanged)
- [ ] #4 Given --stream, when parse runs, then errors and warnings go to stderr only, so stdout carries only output data
- [ ] #5 README documents --stream and the incomplete-output trade-off
<!-- AC:END -->
