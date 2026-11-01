---
id: TASK-18
title: Send CLI usage errors and help to stderr
status: Done
assignee:
  - '@claude'
created_date: '2026-11-01T11:24:00Z'
updated_date: '2026-11-01T11:25:00Z'
labels:
  - cli
dependencies: []
priority: low
type: bug
ordinal: 18000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Found while implementing TASK-14. A usage error (unknown flag, missing -c/-i) makes urfave/cli print "Incorrect Usage" and the command help to stdout before parse runs. With --stream in a shell pipeline that text reaches the consumer as if it were output data, breaking design principle 8 (stdout carries only output data).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Given parse with an unknown flag, when it runs, then it exits non-zero and nothing is written to stdout
- [x] #2 Given parse with a missing required flag, when it runs, then it exits non-zero and nothing is written to stdout
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Full test suite passes (python3 -m pytest tests)
- [x] #2 go vet ./... is clean
- [x] #3 Every acceptance criterion has a test that names it (e.g. TASK-12 AC#2)
- [x] #4 README updated for any user-facing change
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Set the cli.App Writer to os.Stderr so urfave/cli's help and usage-error text never reach stdout (stdout is reserved for output data; validate_config --print-merged writes to os.Stdout directly so is unaffected).
2. Add tests in tests/test_streaming.py for an unknown flag and a missing required flag.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Tests: tests/test_streaming.py::test_usage_error_unknown_flag_writes_nothing_to_stdout (AC#1), ::test_usage_error_missing_required_flag_writes_nothing_to_stdout (AC#2). README streaming section notes help/usage errors go to stderr.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Set cli.App Writer to os.Stderr so urfave/cli help and usage-error text go to stderr, keeping stdout for output data only (principle 8, matters for --stream pipelines). --help output also goes to stderr now, as is common for tools whose stdout carries data. Two tests cover unknown and missing required flags; full suite (244) passes, go vet clean.
<!-- SECTION:FINAL_SUMMARY:END -->
