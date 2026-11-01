---
id: TASK-15
title: Add boolean logical type
status: Done
assignee:
  - '@claude'
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T11:20:00Z'
labels:
  - types
dependencies: []
priority: low
type: feature
ordinal: 15000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The original planned type list included bool, which was never built. Producers write booleans in many ways (Y/N, true/false, 1/0), which representations with static args can map.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A boolean logical type converts a value arg to true/false and maps to Avro boolean
- [x] #2 Static args on representations can map arbitrary strings to true/false
- [x] #3 README type tables and tests updated (see CLAUDE.md "Adding New Logical Types")
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
1. config: add Boolean LogicalType + BooleanTypeConfig, UnmarshalTypeConfigs case; add BoolArg helper (bool or "true"/"false" only).
2. representations.go: valueArgSets Boolean {value}; checkStaticArg requires a bool (or "true"/"false") static value at config load.
3. converters.go: convert_boolean via BoolArg.
4. avro_writer: map_type_json "boolean", unionBranch "boolean".
5. README: type table, value-args table, static-arg check text, example of producer spellings mapping.
6. Tests in tests/test_logical_types.py with fixtures: canonical values, static-arg mapping (Y/N, yes/no, 1/0), nullable boolean, invalid value data failure, bad static arg config-load failure.
7. Run pytest + go vet.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Decision (decision-20): the boolean converter accepts only the canonical strings "true"/"false" (and YAML bools from static args). Producer spellings like Y/N, yes/no, 1/0 are mapped with representations carrying static args. This follows design principle 7 (converters never see raw input) and principle 3 (strict by default), and keeps the producer contract explicit in config. Static 'value' args must be a bool or "true"/"false" and are checked at config load (principle 6). The shared config.BoolArg helper is used by both the converter and checkStaticArg.
Tests: AC#1 test_boolean_canonical_values, test_boolean_nullable, test_boolean_invalid_value_is_data_failure; AC#2 test_boolean_static_args_map_producer_spellings, test_boolean_bad_static_arg_fails_config_load (all in tests/test_logical_types.py); AC#3 is covered by those tests plus the README type/value-arg tables. Full pytest: 229 passed; go vet clean.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added a boolean logical type. In config it is the Boolean LogicalType with a BooleanTypeConfig, value arg {value}, and a static 'value' checked at load to be a bool or "true"/"false" (via the new config.BoolArg helper). In the parser, convert_boolean accepts only true/false. In Avro it is written as "boolean" and as the union branch "boolean" when nullable. The README type table, the value-args table, the static-arg check text, and an example of mapping Y/N, yes/no, 1/0 are updated. Decision-20 records why the converter is strict and producer spellings go in representations. Five fixtures and tests were added in tests/test_logical_types.py. The bad-config fixture was added to INVALID_CONFIG_FIXTURES. pytest: 229 passed; go vet clean.
<!-- SECTION:FINAL_SUMMARY:END -->
