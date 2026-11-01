---
id: TASK-19
title: Reject decimal values that exceed precision or scale
status: Done
assignee:
  - '@claude'
created_date: '2026-11-01T11:32:00Z'
updated_date: '2026-11-01T11:34:00Z'
labels:
  - parser
dependencies: []
priority: high
type: bug
ordinal: 19000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Found while reviewing TASK-13. Decimal values are not checked against the field's precision and scale. With precision 4, scale 2, the value 1.234 is silently written as 1.23 and 123456.7 is written as 1.235E+5 (a corrupted value), and the run exits 0. This breaks design principle 5 (never lose data silently): the converter must check that its args form a valid value for the type (principle 7), so a value that does not fit is a data failure like any other.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Given a decimal field with precision p and scale s, when a value has more fractional digits than s, then it is a data failure naming the row, column, value and scale
- [x] #2 Given a decimal field with precision p and scale s, when a value has more than p-s integer digits, then it is a data failure naming the row, column, value and precision
- [x] #3 Given values that fit (including fewer fractional digits than s, negatives, and leading zeros), when parsed, then they are written exactly in both Avro and Parquet output
- [x] #4 Given a static decimal value arg that does not fit precision/scale, when the config loads, then it fails naming the field and pattern
- [x] #5 Given an as_float decimal, when parsed, then behaviour is unchanged (no precision/scale check)
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
1. Add config.DecimalArg(s, precision, scale): parse decimal text exactly to *big.Rat (ParseFloat as syntax gate as before) and fail if it has more fractional digits than scale or more integer digits than precision-scale (judged on the exact value, so trailing zeros beyond scale and leading zeros are fine).
2. convert_decimal uses DecimalArg for both the value and integer_part+decimal_part paths when not as_float; as_float path unchanged.
3. checkStaticArg DecimalTypeConfig: a static 'value' on a precise decimal goes through DecimalArg so a value that does not fit fails config load (field + pattern named by checkRepresentations).
4. Writers: values now always fit, so goavro and parquet decimalBytes encode exactly; update the parquet comment. Go unit tests for DecimalArg.
5. Fixtures + pytest for AC#1-#5 (AC#3 checks both Avro and Parquet), README decimal docs, decision record for trailing zeros.
6. Run pytest + go vet.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
decision-27 (accepted): the precision/scale check is on the exact value, not the text. '1.230' at scale 2 is exactly 1.23, so it is accepted and written as 1.23; no data is lost (principle 5) and the converter only checks that its args form a valid value (principle 7). Likewise leading zeros (0001.50) don't count as integer digits. Alternative (reject any text with more than scale fractional digits) would be stricter on formatting, but formatting strictness belongs in the representation's regex, e.g. [0-9]{2}.

Implementation: config.DecimalArg(s, precision, scale) parses exactly to *big.Rat (ParseFloat kept as syntax gate, so accepted syntax is unchanged and huge exponents are rejected) and is shared by convert_decimal (value and integer_part+decimal_part paths) and checkStaticArg (static 'value' on a precise decimal). Since values always fit, goavro and parquet decimalBytes encode exactly; parquet comment updated. Static integer_part/decimal_part args are not checked at load (they combine with a captured part; left to the converter, as for other multi-arg checks).

Tests: AC#1 test_decimal_too_many_fractional_digits_is_data_failure, AC#2 test_decimal_too_many_integer_digits_is_data_failure, AC#3 test_decimal_values_that_fit_are_written_exactly (Avro + Parquet), AC#4 test_decimal_static_value_must_fit_precision_scale (all in tests/test_logical_types.py), AC#5 test_decimal_as_float_has_no_precision_scale_check; Go unit test TestDecimalArgPrecisionScale. New AC#4 fixtures added to INVALID_CONFIG_FIXTURES in test_config_validation.py. pytest: 265 passed; go vet clean.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Precise decimal values are now checked against the field's precision and scale instead of being silently rounded (1.234 -> 1.23) or corrupted (123456.7 -> 1.235E+5). New config.DecimalArg parses the value exactly to *big.Rat and rejects more than scale fractional digits or more than precision-scale integer digits, with a message naming the scale or precision; the parser reports it as a normal cell data failure (row, column, value). The same helper checks static decimal 'value' args at config load, so a bad one fails naming field and pattern. as_float decimals are unchanged. The check is on the value, so trailing fractional zeros (1.230) and leading zeros are accepted (decision-27). Values written to Avro and Parquet are therefore always exact. README documents the check. Verified with pytest (265 passed) and go vet.
<!-- SECTION:FINAL_SUMMARY:END -->
