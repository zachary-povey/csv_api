# Opportunity Scoring: Logical Types Completion

## Scoring Method

- Importance: Team estimate of how frequently this outcome is needed in real CSV-to-Avro workloads (proxy for user need, % rating 4+ on 5-point scale).
- Satisfaction: Current satisfaction with workarounds (lower = more painful workaround, % rating 4+ on 5-point scale).
- Score: Importance + max(0, Importance - Satisfaction)
- Source: Team estimates based on domain knowledge of CSV/Avro ETL patterns.
- Confidence: Medium (team estimates, not user survey data).

## Scores

| # | Outcome Statement | Imp. (%) | Sat. (%) | Score | Priority |
|---|-------------------|----------|----------|-------|----------|
| 1 | Minimize the time to normalize categorical data from inconsistent CSV sources into a strict output vocabulary | 85% | 20% | 15.5 | Extremely Underserved |
| 2 | Minimize the likelihood of invalid categorical values (typos, unmapped variants) reaching Avro output undetected | 85% | 15% | 15.5 | Extremely Underserved |
| 3 | Minimize the time to convert multi-format date columns into Avro date type without per-format scripts | 90% | 15% | 16.5 | Extremely Underserved |
| 4 | Minimize the likelihood of invalid temporal values (impossible dates, out-of-range times) reaching Avro output | 90% | 10% | 17.0 | Extremely Underserved |
| 5 | Minimize the time to normalize heterogeneous time-of-day formats to Avro time-micros | 75% | 15% | 13.5 | Underserved |
| 6 | Minimize the time to normalize timestamp data across epoch and string formats with timezone handling | 80% | 10% | 14.0 | Underserved |
| 7 | Minimize the likelihood of timezone conversion errors in timestamp processing | 80% | 10% | 14.0 | Underserved |
| 8 | Minimize the time to configure a pipeline handling all column types in a single pass | 80% | 50% | 11.0 | Appropriately Served |

## Top Opportunities (Score >= 12)

1. **Temporal value validation** -- Score: 17.0 -- Converters must validate assembled date/time components, not just accept whatever the regex extracts.
2. **Multi-format date conversion** -- Score: 16.5 -- Date is the most common temporal column. Representation-per-format is the tool's core value proposition.
3. **Categorical normalization** -- Score: 15.5 -- Enum with representation remapping (regex + static args) eliminates per-partner cleanup scripts.
4. **Categorical validation** -- Score: 15.5 -- Converter validates extracted value against permitted_values, catching typos post-regex.
5. **Timestamp normalization** -- Score: 14.0 -- Most complex type, two parameter sets, timezone handling.
6. **Timezone correctness** -- Score: 14.0 -- Highest anxiety outcome. Must be explicitly designed.
7. **Time normalization** -- Score: 13.5 -- Less common standalone but needed for type completeness.

## Implementation Order Recommendation

Based on opportunity scores, implementation complexity, and knowledge building:

1. **Enum** (Score 15.5) -- Simplest converter. Single parameter set (value). Config already has `EnumTypeConfig` with `permitted_values`. High value, low effort. Quick win.
2. **Date** (Score 16.5) -- Three-parameter converter (year, month, day). Straightforward Avro mapping (int, days since epoch). Establishes temporal component validation pattern.
3. **Time** (Score 13.5) -- Five-parameter converter (hour, minute, second, millisecond, microsecond). Avro mapping (long, microseconds since midnight). Establishes microsecond precision pattern.
4. **Timestamp** (Score 14.0) -- Most complex. Two parameter sets (epoch-based and component-based). Benefits from date+time validation patterns already implemented. Timezone handling adds risk.

This order builds knowledge incrementally: enum validates the converter pattern, date/time establish temporal component handling, and timestamp combines both with timezone complexity.

### Data Quality Notes

- Source: team estimates based on CSV processing patterns
- Sample size: internal usage analysis
- Confidence: Medium (team estimates, not user survey data)
