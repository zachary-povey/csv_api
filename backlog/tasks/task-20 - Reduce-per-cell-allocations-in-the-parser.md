---
id: TASK-20
title: Reduce per-cell allocations in the parser
status: Done
assignee:
  - '@claude'
created_date: '2026-11-01T11:38:00Z'
updated_date: '2026-11-01T11:53:00Z'
labels:
  - performance
dependencies: []
references:
  - scripts/benchmark.py
  - internal/parser/parser.go
priority: medium
type: enhancement
ordinal: 20000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
TASK-16 measured ~55k rows/s on a 1M-row, 9-column file (scripts/benchmark.py) with a 4-cpu machine. A CPU profile shows the single parser goroutine is the bottleneck (~94% of wall time) and most of its time is allocation and GC rather than real work: per cell, extract_args builds a new map[string]string (calling SubexpNames/SubexpIndex each time) and FindStringSubmatch allocates a slice; parseValue then copies it into a second map[string]any and merges the static args; ParseData builds a new map per row. extract_args + map work was ~70% of parser time and ~0.9 GB/0.3M rows allocated. Converting decimals via big.Rat (config.DecimalArg, pow10) is the next largest cost. Hot paths only; behaviour and error messages must not change.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Given the 1M-row file from scripts/benchmark.py, when it is converted to Avro on the same machine as the TASK-16 baseline (~55k rows/s), then throughput is at least 2x the baseline, with the new figure recorded in task notes and README "Performance"
- [x] #2 Given the existing test suite, when the parser changes are made, then every test passes unchanged (same output values and error messages)
- [x] #3 Given a representative row of the benchmark's field types (scripts/benchmark.py), when the parser parses it, then it makes at most 45 allocations (95 before TASK-20), checked by a Go test using testing.AllocsPerRun
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
1. Measure the baseline with scripts/benchmark.py (same machine) and a Go benchmark/AllocsPerRun test on a benchmark row (parseRow extracted from ParseData).
2. Precompute each representation's named capture group indexes at NewParser time; match with FindStringSubmatchIndex and fill a per-representation args map whose keys never change (static args set once, winning over groups), so no map is built per cell.
3. DecimalArg: parse and check plain decimals (sign, digits, point) from their digits (SetFrac64 for <=18 digits, digit counts for scale/precision), falling back to the exact big.Rat path; Go test compares against the old implementation for values and error messages.
4. Size the row map up front.
5. Re-profile end to end; raise the pipeline queue buffer (main.go) if goroutine hand-over limits throughput.
6. Add deterministic AC (allocations per row) proven by a Go test; BenchmarkParseRow for AC#1; record before/after benchmark figures in notes and README Performance.
7. Full pytest + go vet.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Baseline (before any change, this machine, 4 cpus, Go 1.24.9, python3 scripts/benchmark.py): Avro 17.34 s = 57,671 rows/s; Parquet 15.05 s = 66,462 rows/s (TASK-16 recorded ~54k/63k). Parser alone (BenchmarkParseRow): 13.6 us/row, 95 allocs/row, 8.3 KB/row.
After (same machine, 3 runs): Avro 7.36-7.53 s = 132,784-135,919 rows/s (~2.3x today's baseline, ~2.5x TASK-16's 54k); Parquet 139k-153k rows/s. Parser alone: ~5.6 us/row, 40 allocs/row, 1.5 KB/row. Recorded in README Performance.
What changed: (1) parser precomputes each representation's named group indexes at NewParser time, matches with FindStringSubmatchIndex and fills a per-representation args map whose keys never change (static args set once and win over groups), replacing the two maps per cell plus SubexpNames/SubexpIndex lookups (decision-29: a Parser is therefore used by one goroutine; TASK-22 workers need their own). (2) config.DecimalArg parses/checks plain decimals (sign, digits, point; <=18 digits via SetFrac64) from their digits and falls back to the exact big.Rat path otherwise (exponents, long values, scale > precision). (3) row map sized up front; ParseData's per-row work moved to parseRow so it can be tested. (4) main.go queueBuffer 10 -> 100 (decision-28): profiling after (1)-(2) showed ~103k rows/s with the parser only ~82% busy, goroutines waking each other row by row (futex); 100 gives ~134k. Buffer alone on the old parser gave no gain (60k), so the parser work is the main driver.
AC#3 added: AC#1 is machine-dependent and can't be a reliable automated test, so a deterministic allocation budget (testing.AllocsPerRun, <=45 allocs/row vs 95 before) guards the improvement in CI. AC#1 evidence is the benchmark runs above plus BenchmarkParseRow (internal/parser/parser_test.go).
Tests: TestParseRowAllocations (AC#3), BenchmarkParseRow (AC#1), TestDecimalArgUnchanged (config; compares DecimalArg with the pre-TASK-20 implementation for values and error messages over plain/exponent/invalid inputs and sizes incl. scale > precision) and TestRepresentationMatchArgs (static args win, unmatched group gives '', no leftovers) for AC#2, plus the unchanged pytest suite (267 passed).
Found (out of scope, fixed incidentally): before this change a representation with two or more unnamed capture groups besides named ones (e.g. '^(?P<year>..)-(?P<month>..)-(?P<day>..)(Z|z)?(x)?$') passed config validation but panicked in extract_args (SubexpIndex("") = -1, index out of range). The parser now ignores unnamed groups, so such a pattern works; no test covers it yet.

Orchestrator review: decision-28 (channel buffer 10 -> 100) accepted. Added regression test tests/test_logical_types.py::test_date_pattern_with_extra_unnamed_groups for the pre-existing crash with named + multiple unnamed groups that the new parser fixes.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Cut the parser's per-cell allocations so the 1M-row benchmark converts to Avro at ~134k rows/s (was 57.7k on the same machine today, ~54k in TASK-16): 2.3x. Parser: named capture group indexes resolved at NewParser time, FindStringSubmatchIndex, and one reusable args map per representation with static args preset (decision-29: a Parser is single-goroutine); config.DecimalArg parses/checks plain decimals from their digits, falling back to big.Rat for anything else; row map pre-sized. main.go queue buffer 10 -> 100 (decision-28), needed once the parser was faster. Allocations per benchmark row 95 -> 40; new AC#3 enforces <=45 via testing.AllocsPerRun because AC#1 is machine-dependent. Verified: pytest 267 passed, go vet clean, Go tests compare DecimalArg against the old implementation (values and errors) and check match semantics. README Performance updated. Side effect: patterns with multiple unnamed groups no longer panic (previously index out of range); untested follow-up.
<!-- SECTION:FINAL_SUMMARY:END -->
