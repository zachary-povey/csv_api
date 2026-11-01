---
id: TASK-5
title: Match csv headers to fields with header_patterns
status: Done
assignee:
  - '@claude'
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T11:03:00Z'
labels:
  - config
  - reader
dependencies: []
priority: medium
type: feature
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Fields are matched to columns by an exact header match on `name`. Producers often vary header text (case, spacing, suffixes) between exports, which is a common source of drift. The original design had per-field `header_patterns` (regexes); it is also on the old notes to-do list.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A field can list header_patterns; a column whose header matches any of them is mapped to the field
- [x] #2 Without header_patterns, the exact match on name still applies
- [x] #3 A header matching more than one field, or a field matching more than one header, is a file-level error
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
1. Config: add header_patterns (list of Go regexes, unanchored) to FieldConfig; compile at LoadConfig, failing with field + pattern named. Add FieldConfig.MatchesHeader: any header_pattern matches, or exact name match when there are no header_patterns (header_patterns replace the name match).
2. Reader: rewrite getFieldPositions to match every header against every field; collect all file-level problems: a header matching >1 field, a field matching >1 header (also duplicate exact-name headers, which previously silently used the last column), and missing required fields (mentioning header_patterns where set). Open returns a list of file errors so all are reported.
3. Column naming: reader exposes the matched header text per field; parser cell errors say column '<header>' and add (field '<name>') when the header differs from the field name. Data failure report keeps the original header row.
4. Record decisions with backlog decision create (patterns replace name match; column naming in messages).
5. Tests: Go unit tests for matching/ambiguity, pytest fixtures for AC#1-3, config validation test for invalid pattern; README Fields section + remove Planned bullet; CLAUDE.md config format list.
6. Run pytest + go vet; finalize.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Decisions: decision-6 (header_patterns replace the exact name match), decision-7 (errors name a column by its header text plus (field '<name>') when they differ; data failure report keeps the original header row), decision-8 (all header problems reported at once, even with fail-fast; main.go appends them straight to ErrorReport.FileErrors since AddReportError would stop at one).
Duplicate exact-name headers (previously the last column silently won) are now a file-level error under AC#3.
reader.Open now returns []error file errors; parser.NewParser takes the reader's ColumnNames(). Removed now-unused Config.AllFieldNames/RequiredFieldNames/FieldMap.
Tests: tests/test_header_patterns.py (AC#1-3), internal/reader/reader_test.go (AC#1-3), README + CLAUDE.md updated (AC#4).
Not done (later tasks): TASK-17 field-name Avro checks, TASK-10 overridable header_patterns.

Orchestrator review: decision-8 (all header errors reported together even with fail-fast) accepted - the header is checked as one unit before any rows are read.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Fields can list header_patterns (unanchored Go regexes, compiled at config load with errors naming field and pattern). A column whose header matches any pattern maps to the field; header_patterns replace the exact name match (decision-6); fields without them keep the exact match. The reader now checks every header against every field and reports, all at once (decision-8): a header matching >1 field, a field matching >1 header (including duplicate exact-name headers, which used to silently use the last column), and missing required fields (listing the header_patterns tried). Cell errors name the column by its header text plus the field name when they differ (decision-7); the data failure report keeps the original header. Verified by tests/test_header_patterns.py and internal/reader/reader_test.go; full pytest (165 passed) and go vet clean. README Fields section and CLAUDE.md updated; Planned bullet removed.
<!-- SECTION:FINAL_SUMMARY:END -->
