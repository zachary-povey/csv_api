---
id: TASK-10
title: Support layered config files
status: Done
assignee:
  - '@claude'
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T11:14:00Z'
labels:
  - config
  - cli
dependencies: []
priority: medium
type: feature
ordinal: 10000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
One data contract is often received from several producers with small differences (a header name, an extra date format). Rather than copy the whole config per producer, a producer config extends a shared base and only states its differences.

Decisions (agreed in design discussion):
- A config can name its base with a top-level `extends: <path>`, resolved relative to the extending file. Chains are allowed (a extends b extends c). The file at the end of the chain, with no `extends`, is the root.
- The root defines the contract and the output schema. Files that extend it may only change input-side settings: `representations`, `header_patterns`, per-field error messages and `allow_extra_fields` (whichever of these exist when this is implemented; tasks adding new input-side settings should make them overridable). They may not add or remove fields, reorder fields, or change a field's `logical_type` or the schema `name`, so every producer's output has the same schema.
- Fields in an extending file are matched to the base by `name`.
- `representations` in an extending file replaces the base list. `prepend_representations` / `append_representations` add to it instead (first match wins, so prepending makes producer-specific formats take priority). Other lists replace.
- Removing individual items is not supported; replacing the list covers it.
- `parse` and `validate_config` take a single `-c`, the leaf of the chain.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Given a config with extends pointing at a base, when parse runs, then the merged config is used and the output schema matches the base's
- [x] #2 Given a chain a extends b extends c, when a is loaded, then settings merge in order c, b, a with later files winning
- [x] #3 Given an extends path, when it is relative, then it is resolved relative to the file containing it
- [x] #4 Given an extending file that sets a field's logical_type or the schema name, when it is loaded, then config load fails naming the file and the setting
- [x] #5 Given an extending file that names a field not in the base, when it is loaded, then config load fails naming the file and the field
- [x] #6 Given representations in an extending file, when merged, then they replace the base list; given prepend_representations or append_representations, then they are added before or after the base list
- [x] #7 Given a cycle in extends, or an extends path that does not exist, when the config is loaded, then it fails with an error naming the files involved
- [x] #8 Given validate_config with --print-merged, when it runs, then it validates the merged config and prints it as YAML
- [x] #9 README documents extends, the input-side-only rule and the list merge rules
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
1. config: LoadConfig resolves the extends chain leaf->root (relative to containing file, cycle + missing-file errors naming files), strict-decodes the root into Config and each extending file into an override struct (input-side keys only: representations/prepend_representations/append_representations, header_patterns, error_message, required, allow_extra_fields; schema-side keys name/nullable/logical_type decoded only to be rejected with file+setting; unknown field -> error naming file+field), merges root->leaf, then runs the existing checks on the merged Config.
2. main: validate_config --print-merged marshals the merged config to YAML on stdout (status message to stderr). omitempty tags so the output is a clean root config.
3. tests/test_layered_config.py + fixtures (subdirectories for relative resolution), one test per AC incl. print-merged round-trip and checks on the merged config.
4. README: new Layered configs section, remove Planned bullet (fixing -c base -c partner wording); CLAUDE.md.
5. pytest + go vet.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implementation: internal/config/layered.go (readChain, loadMerged, overrideConfig/fieldOverride, applyOverride); LoadConfig now = loadMerged + Config.check() (the existing checks, unchanged, run on the merged config; errors there are prefixed 'merged config for <leaf>'). Errors from an extending file are prefixed "config '<file>' (extends '<base>')".
Decisions: decision-13 (required is overridable: it only says whether the producer's file may omit a column; required: false still needs nullable, checked on the merged config), decision-15 (--print-merged: YAML only on stdout, status message on stderr; output is a root config with no extends), decision-16 (any schema-side key name/nullable/logical_type in an extending file fails, even if equal to the base).
Other choices: an extending file may omit fields entirely; a field listed twice in one extending file fails; if a file sets representations and prepend/append together, the replacement is applied first; header_patterns: [] in an extending file returns to exact name matching. omitempty yaml tags added (and logical_type / allow_extra_fields moved in the structs) so --print-merged output is clean; decoding is unaffected.
README Planned bullet (-c base.yaml -c partner.yaml) removed; the new Layered configs section documents the single -c + extends design.
Tests: tests/test_layered_config.py (AC#1-#9) with tests/fixtures/layered_*; invalid layered fixtures added to INVALID_CONFIG_FIXTURES in test_config_validation.py. Full suite 207 passed, go vet clean.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added layered configs: a config can 'extends: <path>' a base (relative to the containing file, chains allowed). The root defines the schema; extending files may only override input-side settings (allow_extra_fields; per field by name: representations (replace), prepend_/append_representations, header_patterns, error_message, required). Schema-side keys (name, nullable, logical_type), unknown fields and unknown keys fail naming the file; cycles and missing files fail naming the files. All existing load checks run on the merged config. validate_config --print-merged prints the merged config as YAML (a valid root config, round-trip tested). README gains a Layered configs section (Planned bullet removed), CLAUDE.md updated. Verified: python3 -m pytest tests (207 passed), go vet ./... clean. Decisions 13, 15, 16.
<!-- SECTION:FINAL_SUMMARY:END -->
