---
id: TASK-1
title: Fix decimal integer_part/decimal_part losing leading zeros
status: Done
assignee:
  - '@claude'
created_date: '2026-10-08 18:05'
updated_date: '2026-10-08 19:30'
labels:
  - parser
  - decimal
dependencies: []
references:
  - internal/parser/converters.go
priority: high
type: bug
ordinal: 1000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The decimal converter parses `decimal_part` as a number, so leading zeros are lost: a value split into integer_part=12 and decimal_part=05 converts to 12.5 instead of 12.05. This is silent data corruption, the worst failure mode for a validation tool. It also goes through float64, so precise (non as_float) decimals can lose precision.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 integer_part=12, decimal_part=05 converts to 12.05 for both as_float and precision/scale decimals
- [x] #2 Precise decimals built from parts do not go through float64
- [x] #3 A fixture test covers decimal parts with leading zeros
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. In convert_decimal's integer_part/decimal_part branch, stop parsing the parts as float64. Normalise each part to a digit string (string as-is, int via strconv.Itoa), validate integer_part as optional sign + digits and decimal_part as digits only, and join them into a single "<int>.<dec>" decimal string.
2. as_float: strconv.ParseFloat on the joined string. Precise: decimalToRat on the joined string directly, so no float64 is involved (this also fixes negative integer parts, which were previously added to a positive fraction, e.g. -12 + .05 = -11.95).
3. Add fixtures decimal_split_parts_leading_zeros (as_float) and decimal_precision_scale_split_parts_leading_zeros (precision/scale) with values like 12/05 and 0/007, plus pytest cases in tests/test_logical_types.py.
4. Run python3 -m pytest tests and go vet ./...
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Parts are now joined as text ("<integer_part>.<decimal_part>") and parsed once: ParseFloat for as_float, big.Rat for precision/scale, so precise decimals never touch float64. Parts are validated (integer_part: optional sign + digits, decimal_part: digits only). Side effect: negative integer parts are now correct (old code computed -3 + .007 = -2.993).
Evidence: the two new leading-zero tests fail on the old converter (Decimal('12.500') != Decimal('12.050')) and pass with the fix; python3 -m pytest tests: 51 passed; go vet ./... clean.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Fixed silent corruption in decimal integer_part/decimal_part conversion: decimal_part was parsed as a number, so 12 + 05 became 12.5. convert_decimal now joins the parts as a decimal string and parses it once (float for as_float, big.Rat for precision/scale, with no float64 step), and validates the parts. Added fixtures decimal_split_parts_leading_zeros and decimal_precision_scale_split_parts_leading_zeros (12.05, 0.07/0.005, negative values) with pytest cases; both fail on the old code and pass now. Full suite 51 passed, go vet clean.
<!-- SECTION:FINAL_SUMMARY:END -->
