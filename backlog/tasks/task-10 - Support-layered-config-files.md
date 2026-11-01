---
id: TASK-10
title: Support layered config files
status: To Do
assignee: []
created_date: '2026-10-08 18:05'
updated_date: '2026-11-01 10:21'
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
- [ ] #1 Given a config with extends pointing at a base, when parse runs, then the merged config is used and the output schema matches the base's
- [ ] #2 Given a chain a extends b extends c, when a is loaded, then settings merge in order c, b, a with later files winning
- [ ] #3 Given an extends path, when it is relative, then it is resolved relative to the file containing it
- [ ] #4 Given an extending file that sets a field's logical_type or the schema name, when it is loaded, then config load fails naming the file and the setting
- [ ] #5 Given an extending file that names a field not in the base, when it is loaded, then config load fails naming the file and the field
- [ ] #6 Given representations in an extending file, when merged, then they replace the base list; given prepend_representations or append_representations, then they are added before or after the base list
- [ ] #7 Given a cycle in extends, or an extends path that does not exist, when the config is loaded, then it fails with an error naming the files involved
- [ ] #8 Given validate_config with --print-merged, when it runs, then it validates the merged config and prints it as YAML
- [ ] #9 README documents extends, the input-side-only rule and the list merge rules
<!-- AC:END -->
