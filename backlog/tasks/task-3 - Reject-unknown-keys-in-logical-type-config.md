---
id: TASK-3
title: Reject unknown keys in logical type config
status: Done
assignee:
  - '@claude'
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T10:45:00Z'
labels:
  - config
dependencies: []
references:
  - internal/config/config.go
priority: medium
type: bug
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Unknown keys under `logical_type` (e.g. a typo like `permited_values`) are silently ignored, because the mapstructure decode in `narrowType` allows extras (there is a "todo: validate no extras" comment). A typo in the agreed contract should be caught at config load. Also, `LoadConfig` calls `validate.Struct` on each narrowed type config but ignores the result.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A config with an unknown key under logical_type or its args fails to load with an error naming the key
- [x] #2 Validation errors on narrowed type configs are no longer ignored
- [x] #3 `validate_config` reports these errors
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
1. In narrowType, decode logical_type with a mapstructure Decoder configured with ErrorUnused, so unknown keys under logical_type and logical_type.args fail with the key named.
2. Prefix logical type errors (decode, validation, unknown type name) with the field name so the user knows where to look.
3. Remove the redundant, ignored validate.Struct call in LoadConfig (narrowType already validates and returns the error) so no validation result is discarded.
4. Other config levels (top, field, representation) already use yaml.UnmarshalStrict, which is struct-field based; add a regression test, no code change. Representation args stay free-form value args (separate Planned item).
5. Fixtures + pytest tests (parse and validate_config) naming TASK-3 ACs; README note on strictness.
6. Run pytest and go vet.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
- narrowType now decodes with mapstructure ErrorUnused; Args fields tagged mapstructure:"args" so errors read "'args' has invalid keys: ...". Unknown keys directly under logical_type report as "'' has invalid keys: ..." (mapstructure's root name), prefixed with "field '<name>': invalid logical_type:".
- AC#2: the ignored validate.Struct in LoadConfig was redundant (narrowType already validates and returns the error); removed it, so no validation result is discarded. Validation errors now carry the field name too.
- Top/field/representation levels were already strict via yaml.UnmarshalStrict (struct-field based, so later tasks adding struct fields get new keys for free). No code change there, so no decision recorded; added a regression test (config_unknown_field_key).
- Not in scope: representation args remain free-form (checking them is the separate Planned item about representations providing their type's args). Validator messages (e.g. "Key: 'DecimalTypeConfig.Args.Scale' ... 'required_if' tag") are still Go-struct-ish; friendlier wording could be follow-up.
- Tests: tests/test_config_validation.py (AC#1: test_unknown_logical_type_key_fails_parse, test_unknown_logical_type_arg_fails_parse; AC#2: test_invalid_logical_type_args_fail_parse; AC#3: test_validate_config_reports_logical_type_errors, test_valid_config_passes_validate_config).

Orchestrator review: decoded via a wrapper struct so unknown-key errors read 'logical_type has invalid keys' / 'logical_type.args has invalid keys' instead of an empty root name.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Unknown keys under logical_type and logical_type.args now fail config load (mapstructure ErrorUnused) with the key and field named, for both parse and validate_config. Removed the redundant ignored validate.Struct call; type config validation errors propagate with the field name. Other config levels were already strict via yaml.UnmarshalStrict (regression test added). README documents config strictness. pytest (60 passed) and go vet clean. Follow-up idea: friendlier validator error wording.
<!-- SECTION:FINAL_SUMMARY:END -->
