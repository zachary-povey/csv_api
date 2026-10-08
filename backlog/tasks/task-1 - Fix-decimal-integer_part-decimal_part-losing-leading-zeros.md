---
id: TASK-1
title: Fix decimal integer_part/decimal_part losing leading zeros
status: To Do
assignee: []
created_date: '2026-10-08 18:05'
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
- [ ] #1 integer_part=12, decimal_part=05 converts to 12.05 for both as_float and precision/scale decimals
- [ ] #2 Precise decimals built from parts do not go through float64
- [ ] #3 A fixture test covers decimal parts with leading zeros
<!-- AC:END -->
