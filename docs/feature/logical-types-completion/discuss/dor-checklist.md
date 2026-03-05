# Definition of Ready Checklist: Logical Types Completion

## US-01: Enum Field Validation with Flexible Input Mapping

| DoR Item | Status | Evidence |
|----------|--------|----------|
| Problem statement clear | PASS | "Ravi finds it painful to maintain per-partner Python scripts that normalize inconsistent labels ('active', 'Active', 'live', 'enabled')" -- specific pain with concrete examples in domain language |
| User/persona identified | PASS | Ravi Krishnan, data engineer ingesting partner CSV exports with inconsistent categorical labels |
| 3+ domain examples | PASS | 4 examples: direct match, case-insensitive via regex representation, value remapping via static args, extracted value not in permitted set |
| UAT scenarios (3-7) | PASS | 5 scenarios in user-stories.md + 6 scenarios in acceptance-criteria.md covering: valid values, remapping via static args, case-insensitive via regex, extracted value rejection, schema correctness, missing config args |
| AC derived from UAT | PASS | 6 acceptance criteria derived from scenarios: permitted value check, error reporting, case-sensitive comparison, static arg remapping, Avro schema, mixed types |
| Right-sized | PASS | ~1 day effort, 5 UAT scenarios, single converter + Avro mapping + tests |
| Technical notes | PASS | Existing config struct, goavro enum expectations, field name as Avro enum name, EnsureValueName behavior documented |
| Dependencies tracked | PASS | Config parsing already implemented. No external dependencies. |

### DoR Status: PASSED

---

## US-02: Date Field Conversion from Multiple Input Formats

| DoR Item | Status | Evidence |
|----------|--------|----------|
| Problem statement clear | PASS | "Ravi receives date columns in ISO, European, and US formats from different partners. He finds it tedious to maintain per-format Python conversion scripts and worries about DD/MM vs MM/DD confusion" |
| User/persona identified | PASS | Ravi Krishnan, data engineer producing partition-compatible Avro from multi-format date sources |
| 3+ domain examples | PASS | 3 examples: ISO format with default representation, European format via custom representation, invalid date components after extraction |
| UAT scenarios (3-7) | PASS | 6 scenarios in user-stories.md + 8 scenarios in acceptance-criteria.md covering: ISO conversion, custom format representation, pre-epoch, leap year valid/invalid, invalid month, invalid day, epoch zero |
| AC derived from UAT | PASS | 6 acceptance criteria: component conversion, format-independent output, pre-epoch negatives, invalid component errors, leap year validation, Avro schema |
| Right-sized | PASS | ~1 day effort, 6 UAT scenarios, single converter + Avro mapping + tests |
| Technical notes | PASS | Go time.Date() normalization caveat (must verify constructed date matches input), days calculation method, string-to-int parsing |
| Dependencies tracked | PASS | Config parsing already implemented. No external dependencies. |

### DoR Status: PASSED

---

## US-03: Time Field Conversion with Fractional Second Precision

| DoR Item | Status | Evidence |
|----------|--------|----------|
| Problem statement clear | PASS | "Ravi finds it wasteful to embed time-of-day values in full timestamps, which carry unnecessary date components and confuse downstream scheduling queries" |
| User/persona identified | PASS | Ravi Krishnan, data engineer ingesting scheduling and time-of-day data from varied formats |
| 3+ domain examples | PASS | 4 examples: standard 24h time, fractional seconds with varying precision, custom format via representation, out-of-range component |
| UAT scenarios (3-7) | PASS | 6 scenarios in user-stories.md + 8 scenarios in acceptance-criteria.md covering: standard conversion, 6-digit fraction, 3-digit fraction padding, custom format (no seconds), midnight, max time, invalid hour, invalid minute |
| AC derived from UAT | PASS | 6 acceptance criteria: component conversion, fraction handling, defaults, range validation, edge cases, Avro schema |
| Right-sized | PASS | ~1 day effort, 6 UAT scenarios, single converter + Avro mapping + tests |
| Technical notes | PASS | Microsecond calculation formula, fraction group padding/truncation logic, string-to-int parsing |
| Dependencies tracked | PASS | Config parsing already implemented. No external dependencies. |

### DoR Status: PASSED

---

## US-04: Timestamp Field Conversion with Timezone Handling

| DoR Item | Status | Evidence |
|----------|--------|----------|
| Problem statement clear | PASS | "Ravi receives timestamps as ISO strings with timezone offsets, epoch seconds from APIs, and epoch milliseconds from JavaScript clients. Getting format/precision/timezone handling wrong means silent data corruption in financial calculations" |
| User/persona identified | PASS | Ravi Krishnan, data engineer building event pipelines from multi-format CSV sources needing canonical UTC timestamps |
| 3+ domain examples | PASS | 4 examples: ISO with timezone offset (component-based), epoch seconds with static precision arg, component-based with defaults, invalid month component |
| UAT scenarios (3-7) | PASS | 7 scenarios in user-stories.md + 8 scenarios in acceptance-criteria.md covering: ISO/Z, positive offset, negative offset, fractional seconds, component defaults, epoch seconds, epoch milliseconds, invalid component |
| AC derived from UAT | PASS | 7 acceptance criteria: component-based conversion, epoch-based conversion, timezone handling, validation, fractional precision, Avro schema, parameter set equivalence |
| Right-sized | PASS | ~2-3 days effort (most complex type, two parameter sets), 7 UAT scenarios |
| Technical notes | PASS | Go time.Date() with timezone, epoch multiplication, parameter set dispatch pattern (same as decimal), shared validation with date/time |
| Dependencies tracked | PASS | Config parsing already implemented. Benefits from date (US-02) and time (US-03) validation logic. Implementation order: enum -> date -> time -> timestamp. |

### DoR Status: PASSED

---

## Summary

| Story | DoR Status | Estimated Effort |
|-------|-----------|-----------------|
| US-01: Enum | PASSED | 1 day |
| US-02: Date | PASSED | 1 day |
| US-03: Time | PASSED | 1 day |
| US-04: Timestamp | PASSED | 2-3 days |

All four stories pass DoR and are ready for handoff to DESIGN wave.

### Recommended Implementation Order

1. **US-01: Enum** -- simplest converter, quick win, validates the converter pattern
2. **US-02: Date** -- establishes temporal component validation pattern
3. **US-03: Time** -- establishes microsecond precision pattern
4. **US-04: Timestamp** -- most complex, benefits from date+time validation already implemented
