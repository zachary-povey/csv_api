---
id: TASK-11
title: Add per-field error messages to config
status: Done
assignee:
  - '@claude'
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T11:09:00Z'
labels:
  - config
  - errors
dependencies: []
priority: medium
type: feature
ordinal: 11000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Error reports describe failures in terms of patterns and converters, which means little to the producer who has to fix the data. A config author should be able to attach a plain-language message to a field (e.g. "amount must be pounds with 2 decimal places, e.g. 12.50 - contact the finance team") that is shown when that field fails. This is the "custom errors" item from the original scope.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A field can have an optional error message in the config
- [x] #2 The message is included with that field's errors in the console report and in the data failure report column
- [x] #3 Fields without a message report as they do now
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
1. Add optional error_message (string) to FieldConfig (plain input-side field, overridable later by TASK-10).
2. error_tracking: add a CellError struct (row, column label, field, raw value, reason, message) with a Text() formatter; ErrorReport.CellErrors holds structs (groundwork for TASK-12 JSON report); AddCellError on the tracker.
3. parser: build CellErrors; the text appends ' (note: <message>)' after the technical reason when the field has error_message; used for both the console report and the __csv_api_error__ column.
4. reader: missing required column file error also appends the field's note.
5. Record decisions (key name; applies to missing-column error; format keeps technical reason).
6. Tests: fixtures field_error_message + missing_required_with_error_message; pytest tests naming TASK-11 AC#1-4; config load test.
7. README (config docs, report column, Error handling), remove Planned bullet; CLAUDE.md Configuration Format.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Decisions: decision-10 (key name error_message, plain optional string on FieldConfig so TASK-10 can override it per producer), decision-11 (also shown on the missing required column file error - it is an error about the field, and cheap to include), decision-12 (format: technical reason kept, message appended as ' (note: <message>)' on one line so it works inside a csv cell too).
Reasoning for decision-12: principle 8 (errors are specific) means the custom message supplements, not replaces, the reason; one-line format keeps the console and __csv_api_error__ texts identical apart from the row prefix.
Structure: cell errors are now error_tracking.CellError structs (Row, Column label, Field, Value, Reason, Message) with Text()/String() formatters; ErrorReport.CellErrors is []CellError and the tracker has AddCellError (Cell report error type removed; limit logic factored into underLimit/checkLimit). This gives TASK-12 (JSON report) the structured data it needs. File/row errors are still strings.
Not done (out of scope): error_message is not applied to row-level errors (wrong value count), which are not about a single field.
Tests: tests/test_field_error_messages.py with fixtures field_error_message and missing_required_with_error_message.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added an optional per-field error_message to the config. It is shown, after the unchanged technical reason, as ' (note: <message>)' with the field's data failures in the console report and in the data failure report's __csv_api_error__ column, and with the missing required column file error. Fields without a message report exactly as before. Cell errors are now stored as error_tracking.CellError structs (row, column, field, value, reason, message), groundwork for TASK-12's JSON report. README (field docs, report column, Planned bullet removed) and CLAUDE.md updated. Decisions 10-12. Verified: python3 -m pytest tests (all pass, incl. tests/test_field_error_messages.py naming TASK-11 AC#1-4) and go vet ./... clean.
<!-- SECTION:FINAL_SUMMARY:END -->
