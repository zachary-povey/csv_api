---
id: TASK-9
title: Check representation args against the logical type at config load
status: Done
assignee:
  - '@claude'
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T10:50:00Z'
labels:
  - config
dependencies: []
priority: medium
type: feature
ordinal: 9000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
If a representation cannot supply the args its logical type needs (e.g. a date pattern with no year/month/day groups or static args), the mistake only shows up at runtime as a conversion error on every cell. Since named groups and static args are known statically, the config can be checked up front. This was an unfinished item in the original "Add type validation" notes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A representation whose named groups plus static args do not form a valid arg set for its logical type fails config load, naming the field and pattern
- [x] #2 Each type accepts all of its documented arg sets (e.g. decimal value vs integer_part+decimal_part, timestamp epoch vs components)
- [x] #3 `validate_config` reports these errors
- [x] #4 Existing test fixtures still load
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
1. Add internal/config/representations.go: CompilePattern (moved from parser.EnsureValueName so config and parser derive the same capture-group args), a table of each type's documented value arg sets, and Config.checkRepresentations called at the end of LoadConfig.
2. For each non-is_null representation: provided args = named groups (incl. implicit value) + static arg keys; reject unknown arg names and sets matching no documented set; errors name field and pattern.
3. Check static arg values where cheap (integer-valued parts, enum value in permitted_values, timestamp precision/timezone, decimal value, string-ness), sharing the integer conversion (config.IntArg, moved from parser numberToInt) with the converters.
4. Fixtures + tests in tests/test_config_validation.py naming TASK-9 AC#1-4.
5. README: document the load-time check, remove the Planned bullet.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implementation: internal/config/representations.go holds the check (Config.checkRepresentations, called at the end of LoadConfig so later layered/nullable work still runs it on the final config). Pattern compilation (EnsureValueName) moved from parser into config.CompilePattern so config and parser derive identical capture-group args; numberToInt moved to config.IntArg and shared by the converters and the static-value check. Invalid regexes now also fail at load, naming field and pattern.

Static arg value checks (within the spirit of AC#1, TASK-2 follow-up): integer-valued args (integer value, date/time/timestamp parts, timestamp value/offset) must be whole numbers; string/enum/decimal value and precision/timezone must be strings; enum value must be in permitted_values; timestamp precision in seconds/milliseconds/microseconds; timezone parseable; decimal value parseable. integer_part/decimal_part content is left to the converter. convert_timestamp's epoch value now uses config.IntArg too (accepts whole YAML floats/int64 like the other integer args) so the load check and runtime agree.

Decision decision-1: the whole-match 'value' of a pattern with no capture groups is ignored by the check for types without a value arg (date, time), matching runtime (those converters ignore it); README's documented '^start$' + static date args keeps working. For timestamp/decimal the implicit value still counts, so e.g. a '^epoch$' timestamp rep with static year/month/day is rejected (it always failed at runtime: the converter took the epoch branch). is_null representations are compiled but skip the arg-set check.

Follow-up (out of scope, not changed): EnsureValueName counts non-capturing groups and flag groups like (?:x) or (?i) as capture groups, so '^(?:live)$' supplies no 'value' at all, contrary to README ('no capture groups makes the whole match value'). The new check now reports such reps without a static value as missing 'value' at load.

Orchestrator review: replaced the hand-rolled capture-group scanner in CompilePattern with regexp/syntax, so non-capturing groups and flags like (?i) are no longer counted as captures (fixes the follow-up noted above; pattern with only (?:...) groups now makes the whole match 'value' as the README says). Go unit test TestCompilePatternValueGroup covers it; tests/test_go_unit.py runs go test ./... inside the pytest suite.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Representations are now checked against their logical type when the config loads (principle 6). Each representation's capture groups (incl. the implicit value) plus static arg keys must form one of the type's documented arg sets; unknown names (e.g. a date 'yr' group), missing args and conflicting sets fail with 'field X, pattern P: ...'. Static arg values are also checked where cheap (whole numbers, enum value in permitted_values, timestamp precision/timezone, decimal value). is_null reps are skipped. Code: internal/config/representations.go (CompilePattern and IntArg moved there from the parser and shared). Tests in tests/test_config_validation.py: AC#1 test_invalid_representation_args_fail_parse, AC#2 test_every_documented_arg_set_is_accepted, AC#3 test_validate_config_reports_representation_arg_errors, AC#4 test_existing_fixtures_still_load. README documents the check; Planned bullet removed. pytest 119 passed, go vet clean. Follow-up: (?:...) / (?i) groups suppress the implicit whole-match value (see notes).
<!-- SECTION:FINAL_SUMMARY:END -->
