---
id: TASK-23
title: >-
  Treat empty cells as null on nullable fields with empty_is_null and
  quoted_empty_is_null
status: Done
assignee:
  - '@claude'
created_date: '2026-11-01T17:51:00Z'
updated_date: '2026-11-01T17:59:00Z'
labels:
  - config
  - reader
  - parser
dependencies: []
priority: high
type: feature
ordinal: 23000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Agreed with the user, replacing decision-3 (nulls only from explicit is_null representations). Requiring an is_null pattern for the most common null spelling (an empty cell) is clunky, and the tool is meant for general FOSS use, so producer conventions must be covered: many tools write null as an empty cell, while some (e.g. Postgres COPY ... CSV) write null as an unquoted empty cell `,,` and an empty string as a quoted empty cell `,"",`. Go's encoding/csv returns "" for both, but quoting of an empty field can be inferred from Reader.FieldPos (field start columns) and Reader.InputOffset (record end), since an empty field is 0 bytes raw when unquoted and 2 bytes when quoted.

Agreed design:
- Two flags, both default true: `empty_is_null` (unquoted empty cell `,,` is null) and `quoted_empty_is_null` (quoted empty cell `,"",` is null). They are independent of is_null representations: adding an is_null pattern never changes how empty cells behave.
- The flags apply only to nullable fields. Setting either explicitly on a field that is not nullable fails at config load. A non-nullable field treats empty cells exactly as today.
- The flags are checked before representations. With a flag off, the empty cell goes through the representations like any other value (e.g. a string field with `^$` gets an empty string; otherwise it is a data failure).
- Whitespace-only cells are not covered by the flags; use an is_null pattern such as `^\s*$`.
- The flags can be set at the top level of the config as defaults for every nullable field, and per field to override the top-level setting.
- They describe how the producer writes nulls (input-side contract), so layered configs (TASK-10) may override them, both top-level and per field.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Given a nullable field with default flags, when a cell is an unquoted empty cell (,,), then it is written as null
- [x] #2 Given a nullable field with default flags, when a cell is a quoted empty cell (,"",), then it is written as null
- [x] #3 Given a nullable string field with quoted_empty_is_null: false and a ^$ representation, when cells are ,, and ,"",, then the first is null and the second is an empty string
- [x] #4 Given a nullable field with empty_is_null: false and no representation matching an empty value, when a cell is ,,, then it is a data failure
- [x] #5 Given a nullable field with an is_null representation and default flags, when cells are empty, then they are still null (is_null does not change empty-cell behaviour)
- [x] #6 Given either flag set explicitly on a field that is not nullable, when the config loads, then it fails naming the field and the flag
- [x] #7 Given a non-nullable field, when a cell is empty (quoted or not), then behaviour is unchanged from before this task
- [x] #8 Given top-level empty_is_null or quoted_empty_is_null, when a nullable field does not set them, then the top-level values apply, and a per-field value overrides the top-level one
- [x] #9 Given an extending (layered) config, when it sets either flag at the top level or on a field, then the merged config uses its value
- [x] #10 Given empty cells as the last field of a row, with CRLF line endings, with no trailing newline at end of file, and after a quoted field containing a newline, when parsed, then quoted and unquoted empty cells are still told apart correctly
- [x] #11 Given the README, when a user reads the nullable docs, then it documents both flags, their defaults, the top-level/per-field rule, the Postgres-style example and that whitespace-only cells need an is_null pattern
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
1. Config: *bool empty_is_null / quoted_empty_is_null on Config (top-level default) and FieldConfig (override), omitempty; Config.EmptyCellIsNull(field, quoted) resolves field > top-level > true, false for non-nullable; check() rejects an explicit per-field flag on a non-nullable field naming field and flag.
2. Layered: add both flags to overrideConfig and fieldOverride/applyOverride, inputSideOnly message.
3. Reader: only for fields whose two flags differ (otherwise quoting can't matter, so default configs pay nothing), classify empty cells as quoted: non-last field via FieldPos column gap (1 vs 3); last field via the raw byte just before the record's line terminator (InputOffset end, tail of recently read bytes kept by a small wrapping io.Reader; handles CRLF, EOF without newline, multi-line records, skipped blank lines). Record.QuotedEmpty []bool allocated only when a quoted empty cell is found.
4. Parser: in parseRow, an empty cell on a nullable field whose flag (quoted or not) is on is null before representations are tried.
5. Tests: Go unit tests for quote detection (AC#10) and config; pytest fixtures for AC#1-9; update test_empty_cell_on_nullable_field_without_null_representation_fails (decision-3 superseded).
6. README nullable docs + layered table; CLAUDE.md config format/layered note. Run pytest, go vet, go test -race.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented: Config/FieldConfig *bool empty_is_null + quoted_empty_is_null (omitempty), Config.EmptyCellIsNull resolves field > top-level > true (false for non-nullable); checkEmptyIsNull rejects explicit per-field flags on non-nullable fields. Layered: both flags in overrideConfig and fieldOverride/applyOverride. Parser: parseRow nulls an empty cell before representations when its flag is on.
Reader: decision-33 - empty-cell quoting is classified only for fields whose two flags differ (with defaults quoting can't matter, so no cost and no Record.QuotedEmpty allocation). Non-last field: FieldPos column gap (1 unquoted / 3 quoted). Last field: the raw byte before the record's line terminator (strip one \n then one \r; '"' = quoted), looked up by InputOffset in a tailReader keeping the last 64KiB+4 bytes read (bufio of 64KiB between it and csv). This avoids line-start offset tracking, and handles CRLF, EOF with no newline or a lone \r (csv drops it), multi-line records and skipped blank lines.
decision-34 - top-level flags never fail the load; only explicit per-field settings on non-nullable fields do.
Decision-3 superseded: test_empty_cell_on_nullable_field_without_null_representation_fails rewritten as ..._is_null; parquet_all_types opt_name now uses quoted_empty_is_null: false with a quoted "" cell so the empty-string vs null distinction stays tested.
Follow-up concern (out of scope): the data failure report re-writes raw rows with encoding/csv, which writes a quoted empty cell as unquoted, so re-feeding the report could turn an empty string into null when quoted_empty_is_null is false.
Tests: AC#1,#2 test_unquoted_and_quoted_empty_cells_are_null_by_default; AC#3 test_quoted_empty_is_null_false_postgres_style; AC#4 test_empty_is_null_false_without_matching_representation_fails; AC#5 test_is_null_representation_does_not_change_empty_cells; AC#6 test_flag_on_non_nullable_field_fails_config_load (+ layered); AC#7 test_empty_cells_on_non_nullable_fields_are_unchanged; AC#8 test_top_level_flags_are_defaults_and_fields_override_them; AC#9 test_layered_config_overrides_flags; AC#10 Go TestQuotedEmptyDetection/TestQuotedEmptyDetectionLongInput + test_quoted_and_unquoted_empty_cells_told_apart_in_any_position (CRLF fixture protected by .gitattributes -text); AC#11 README nullable section. pytest 293 passed, go vet clean, go test -race ok.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Nullable fields now treat empty cells as null via empty_is_null (unquoted ,,) and quoted_empty_is_null (quoted ,"",), both default true, settable top-level (defaults) and per field (override), overridable by layered configs; explicit per-field setting on a non-nullable field fails at load. Flags are checked before representations; is_null representations don't affect empty cells. The reader tells quoted from unquoted empty cells (FieldPos gap for non-last fields, raw byte before the line terminator for the last field) only for fields whose flags differ, so default configs pay nothing. Decision-3 superseded; decisions 33-34 recorded. README (nullable docs, Postgres example, layered table) and CLAUDE.md updated. Verified: pytest 293 passed, go vet clean, go test -race ok. Follow-up: data failure report loses empty-cell quoting.
<!-- SECTION:FINAL_SUMMARY:END -->
