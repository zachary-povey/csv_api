---
id: TASK-6
title: Add allow_extra_fields config option
status: Done
assignee:
  - '@claude'
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T11:06:00Z'
labels:
  - config
  - reader
dependencies: []
priority: low
type: feature
ordinal: 6000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Columns not in the config are always ignored. Some contracts want to treat an unexpected column as a sign the producer changed the format. The original design had a top-level `allow_extra_fields` setting.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 With allow_extra_fields: false, a column not mapped to any field is a file-level error naming the column
- [x] #2 The default keeps the current behaviour (extra columns ignored)
- [x] #3 README and tests updated
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
1. config: add top-level AllowExtraFields *bool (yaml allow_extra_fields) with ExtraFieldsAllowed() defaulting to true (AC#2 / decision).
2. reader.getFieldPositions: when extra fields are not allowed, every column matching no field is a file error naming it via describeColumn; reported with other header errors.
3. Tests: fixtures extra_fields_rejected / extra_fields_default; pytest naming TASK-6 AC#1/#2; unknown-key check still strict.
4. README (Fields section + remove Planned bullet), CLAUDE.md Configuration Format.
5. Run pytest + go vet; finalize.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
decision-9: allow_extra_fields defaults to true. Context: design principle 3 (strict by default) would suggest false, but AC#2 records the intent that the default keeps today's behaviour. Decision: follow the AC; extra columns never reach the output, so ignoring them can't corrupt or silently drop data. Consequences: strictness is an explicit opt-in (allow_extra_fields: false); stored as *bool (Config.AllowExtraFields, read via ExtraFieldsAllowed()) so 'unset' is distinguishable when TASK-10 makes it overridable in layered configs. (The decision CLI only takes a title, so this body lives here.)
Implementation: getFieldPositions adds one file error per unmapped column, 'header '<text>' (column N) matches no field (allow_extra_fields is false)', alongside the other header errors (decision-8: all reported at once). A non-bool value fails the config load via yaml.UnmarshalStrict.
Tests (tests/test_header_patterns.py): AC#1 test_extra_columns_rejected_when_not_allowed (fixture extra_fields_rejected), test_allow_extra_fields_must_be_a_bool; AC#2 test_extra_columns_ignored_by_default (fixtures extra_fields_default, extra_fields_allowed_explicit). pytest 176 passed; go vet clean.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added top-level allow_extra_fields config option (default true, keeping extra columns ignored; decision-9). With allow_extra_fields: false, every csv column that maps to no field is a file-level error naming its header and column number, reported together with other header errors. README (Fields section, removed Planned bullet) and CLAUDE.md updated. Tests in tests/test_header_patterns.py name TASK-6 AC#1/AC#2; full pytest (176) and go vet pass. Follow-up: TASK-10 to make it overridable in layered configs.
<!-- SECTION:FINAL_SUMMARY:END -->
