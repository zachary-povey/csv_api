---
id: TASK-12
title: Add JSON error report
status: Done
assignee:
  - '@claude'
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T11:18:00Z'
labels:
  - errors
  - cli
dependencies: []
references:
  - internal/options/options.go
priority: medium
type: feature
ordinal: 12000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Errors are only reported to the console as text, which is hard for other systems to consume (e.g. raising a ticket or alerting the producer automatically). A structured JSON report was on the original notes to-do list; `--error-report` exists with only `console` supported so this can be added as a new value.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 `--error-report` accepts a JSON option that writes errors to a file
- [x] #2 Each error records its group (file, row, data), row number, column, raw value and reason where applicable
- [x] #3 The report records whether processing stopped early and why
- [x] #4 README and tests updated
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
1. error_tracking: structure row errors as RowError{Row, Reason} (row from FieldPos / csv.ParseError.StartLine); add DroppedRows to ErrorReport.
2. error_tracking/json_format.go: build the JSON report {valid, stopped_early, stop_reason, dropped_rows, errors[{group,row,column,field,value,reason,message}]} from the tracker, groups file/row/data/execution.
3. options: --error-report accepts console|json; new --error-report-path, required with json and rejected otherwise.
4. main.go: create the tracker in the action so every outcome (config/open failures included) lands in it; console report to stderr as before; with json, write the report atomically (temp + rename) on success and failure.
5. tests/test_json_error_report.py covering each group, fail-fast/max-errors early stop, success, dropped rows, flag validation; update the old unsupported-json test.
6. README error handling docs + JSON layout, remove json from Planned bullet; CLAUDE.md options line.
7. Decisions: CLI shape, report written on success, JSON layout.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Decisions:
- decision-17 (CLI shape): --error-report json + required --error-report-path, validated like --data-failure-report. Console summary still goes to stderr (principle 8); JSON goes to the file.
- decision-18 (written on success too): consumers always find a report; it's a report, not output, so principle 4 doesn't apply - it's most needed on failure. Written atomically (temp + rename) like the other outputs. Flag validation errors happen before the run and write no report.
- decision-19 (layout): flat, every key always present (null when not applicable) so consumers don't need key checks. column is the header as in the file (not the console label); field is the config field.

Implementation notes:
- Row errors are now RowError{Row, Reason}; Reason keeps the full console text (console output unchanged). Broken-quote row number comes from csv.ParseError.StartLine.
- CellError gained Header (raw column header) for the JSON column.
- The tracker is created in the parse action so the report covers failures outside the pipeline (config can't load, input can't open, output can't be renamed): any untracked run error is added as an execution error when writing the JSON.
- JSON stopped_early is also true for file errors (no rows checked) and execution errors, with a reason; the console report is unchanged.
- ErrorReport.DroppedRows set from the parser after the pipeline finishes.
- tests/test_error_handling.py::test_unsupported_error_report now uses 'xml' since json is supported.

Out of scope (not changed): internal/error_tracking/console_format.go has unused constants defaultCacheTTL/maxCacheTTL (passphrase cache) that look stray.

Orchestrator review: removed the stray unused defaultCacheTTL/maxCacheTTL constants from console_format.go. Row reasons repeating the row number and unstructured header errors accepted as-is (AC#2 says 'where applicable').
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added --error-report json with a required --error-report-path: the console report still goes to stderr, and a JSON report is written atomically to the file on every run (success included). Layout: {valid, stopped_early, stop_reason, dropped_rows, errors[{group: execution|file|row|data, row, column, field, value, reason, message}]}. Row errors are now structured (RowError with row number); cell errors carry the raw header. Execution errors outside the pipeline (config/input/output failures) are included. Verified by tests/test_json_error_report.py (13 tests: each group, fail-fast/max-errors/broken-quote early stop, success, dropped rows, flag validation); full suite 220 passed, go vet clean. README documents the flag and layout; json removed from Planned; CLAUDE.md options line updated. Decisions 17-19.
<!-- SECTION:FINAL_SUMMARY:END -->
