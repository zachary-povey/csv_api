---
id: TASK-2
title: Accept numeric static args for integer fields
status: Done
assignee:
  - '@claude'
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T10:42:00Z'
labels:
  - parser
dependencies: []
references:
  - internal/parser/converters.go
priority: medium
type: bug
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Static `args` in YAML can be numbers, but `convert_int` only accepts strings, so a representation like pattern `^none$` with `args: {value: 0}` fails every matching cell with "value in args is not of type string". Other converters (date, time, timestamp) already accept both via `argToInt`.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 An integer field with a numeric static value arg converts correctly
- [x] #2 String values from capture groups still convert as before
- [x] #3 A fixture test covers a numeric static arg on an integer field
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
1. Extend argToInt to accept int64 and whole-number float64 (yaml.v2 decodes static numbers as int, or float64 for e.g. 5.0).
2. Make convert_int use argToInt(args, "value") so string capture groups and numeric static args both work.
3. Add fixture integer_static_numeric_arg (capture group + static numeric arg) and pytest naming TASK-2 AC#1-3; add a non-integer float static arg check via Go unit test.
4. README: note static args may be YAML numbers for integer-valued args.
5. Run pytest + go vet.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
convert_int now uses argToInt; argToInt/argToIntDefault share new numberToInt helper accepting int, int64, uint64 (in range) and whole-number float64 (yaml.v2 decodes 12.0 as float64). Fractional/NaN/Inf floats rejected per cell. Tests: tests/test_logical_types.py::test_integer_static_numeric_arg (AC#1-3, fixture integer_static_numeric_arg) and internal/parser/converters_test.go (AC#1-2 plus rejection cases). Follow-up (out of scope): per design principle 6, a non-integer static arg on an integer field is still only caught per cell at run time, not at config load.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Integer fields now accept numeric static args (e.g. args: {value: 0}). convert_int delegates to argToInt, which together with argToIntDefault now accepts int/int64/uint64 and whole-number float64 via a shared numberToInt helper, so date/time/timestamp parts also gain int64/float handling. String capture-group values convert as before. Added fixture test and Go unit tests; README notes numeric static args. pytest (52 passed) and go vet clean. Follow-up: validate static args at config load (principle 6).
<!-- SECTION:FINAL_SUMMARY:END -->
