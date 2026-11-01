---
id: TASK-23
title: >-
  Treat empty cells as null on nullable fields with empty_is_null and
  quoted_empty_is_null
status: To Do
assignee: []
created_date: '2026-11-01T17:51:00Z'
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
- [ ] #1 Given a nullable field with default flags, when a cell is an unquoted empty cell (,,), then it is written as null
- [ ] #2 Given a nullable field with default flags, when a cell is a quoted empty cell (,"",), then it is written as null
- [ ] #3 Given a nullable string field with quoted_empty_is_null: false and a ^$ representation, when cells are ,, and ,"",, then the first is null and the second is an empty string
- [ ] #4 Given a nullable field with empty_is_null: false and no representation matching an empty value, when a cell is ,,, then it is a data failure
- [ ] #5 Given a nullable field with an is_null representation and default flags, when cells are empty, then they are still null (is_null does not change empty-cell behaviour)
- [ ] #6 Given either flag set explicitly on a field that is not nullable, when the config loads, then it fails naming the field and the flag
- [ ] #7 Given a non-nullable field, when a cell is empty (quoted or not), then behaviour is unchanged from before this task
- [ ] #8 Given top-level empty_is_null or quoted_empty_is_null, when a nullable field does not set them, then the top-level values apply, and a per-field value overrides the top-level one
- [ ] #9 Given an extending (layered) config, when it sets either flag at the top level or on a field, then the merged config uses its value
- [ ] #10 Given empty cells as the last field of a row, with CRLF line endings, with no trailing newline at end of file, and after a quoted field containing a newline, when parsed, then quoted and unquoted empty cells are still told apart correctly
- [ ] #11 Given the README, when a user reads the nullable docs, then it documents both flags, their defaults, the top-level/per-field rule, the Postgres-style example and that whitespace-only cells need an is_null pattern
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Full test suite passes (python3 -m pytest tests)
- [ ] #2 go vet ./... is clean
- [ ] #3 Every acceptance criterion has a test that names it (e.g. TASK-12 AC#2)
- [ ] #4 README updated for any user-facing change
<!-- DOD:END -->
