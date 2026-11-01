---
id: TASK-22
title: Parse rows on multiple cores
status: Done
assignee:
  - '@claude'
created_date: '2026-11-01T11:38:00Z'
updated_date: '2026-11-01T12:16:00Z'
labels:
  - performance
dependencies:
  - TASK-20
references:
  - main.go
  - internal/parser/parser.go
priority: low
type: enhancement
ordinal: 22000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
TASK-16 found the pipeline is limited by one parser goroutine (reader and writer each use well under one core while the parser is saturated), so on a multi-core machine most cores sit idle. Spreading validation and conversion over several workers could multiply throughput, but rows must keep their input order in the output and error reporting (row numbers, fail-fast, max-errors, the data failure report) must behave as today. Best done after the per-cell allocation work so the gain is measured on an efficient single-worker baseline.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Given a machine with 4 or more cpus and the scripts/benchmark.py 1M-row file, when it is converted, then throughput is at least 2x the single-worker throughput measured just before this change
- [x] #2 Given any input, when it is converted, then output rows are in input order and the error report, fail-fast, --max-errors and data failure report behave exactly as before (existing tests pass unchanged)
- [x] #3 Given several parser workers and a file whose rows have data failures interleaved with valid rows, when it is converted with or without --error-on-data-failures, --fail-fast or --max-errors, then the output rows and their order, the errors (order and row numbers) and the data failure report are identical to a single-worker run
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
1. Run scripts/benchmark.py before the change for the single-worker baseline.
2. Parser: split ParseData into pure per-row work (parseRow, unchanged) run by N workers, each on its own copy of the compiled representations (decision-29), and a single in-order stage that keeps all side effects as today (error tracking, dropped rows, data failure report, send to writer).
3. A dispatcher batches rows from the reader (up to 256, or fewer when no row is ready so --stream stays responsive), sends each batch to the workers and, in input order, to the in-order stage, which waits for each batch to finish. Everything keeps draining on KillCh so nothing blocks.
4. Worker count = runtime.GOMAXPROCS(0) (a Parser field so tests can set it); no new flag.
5. Add deterministic AC#3 (several workers vs one: same output order, errors and data failure report); prove with a Go test comparing worker counts 1 vs N and a pytest comparing GOMAXPROCS=1 vs default on a few-thousand-row file.
6. Benchmark after; record before/after in notes and README Performance. Run full suite 3x and go test -race ./...
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Added AC#3 (deterministic): AC#1 is machine-dependent (needs >=4 cpus and timing), so AC#3 pins down the behaviour parallelism could break (ordering, error order/row numbers, data failure report) independently of the machine.

Design (decision-31): ParseData now runs a dispatcher (batches of up to 256 rows; a partial batch is sent as soon as no further row is ready, so --stream still streams), runtime.GOMAXPROCS(0) workers each with its own copy of the compiled representations (Parser.worker(), decision-29) doing only parseRow, and the existing loop as the single in-order stage: it waits for each batch in input order and does all side effects (handleRow: error tracking, dropped rows, data failure report, send to the writer) exactly as before. Workers never wait on the in-order stage, so it and the dispatcher always drain the input when killed. Worker count is Parser.Workers (default GOMAXPROCS) so tests can vary it; GOMAXPROCS env limits it; no flag.

Profiling (pprof, 4 cpus): with workers alone Avro reached only ~1.8x (~222-231k rows/s): ~3.1 of 4 cores busy, workers ~6.3s cpu, avro writer ~2.0s, reader ~1.1s, GC mark >1.5s plus STW/assist; the tiny live heap made GOGC=100 collect constantly. GOGC sweep with workers: 100 -> 4.1s, 200 -> 3.0s, 400 -> 2.6s, 800 -> 2.5s. decision-32: main sets debug.SetGCPercent(400) unless GOGC is set. (GOGC=400 doesn't help the old single-worker binary: 134k rows/s.)

Benchmark (scripts/benchmark.py, 1M rows, 4-cpu VM, paired runs before/after): before Avro 131k/131k/127k rows/s (~130k), Parquet 150k/139k/128k; after Avro 391k/397k/368k (~385k, ~3.0x), Parquet 332k/357k/312k (~334k, ~2.4x). Peak RSS Avro 20->37 MiB, Parquet 36->87 MiB. After with GOMAXPROCS=1: Avro 151k. After with GOGC=100: Avro 231k, Parquet 292k.

Tests: internal/parser TestWorkersGiveSingleWorkerResults (AC#3, workers 1 vs 2/4/8, 5000 rows, 4 flag combos), TestWorkersDrainInputWhenKilled (AC#2, no deadlock when killed with an unread output); tests/test_parallel_parsing.py (AC#3/AC#2, GOMAXPROCS=1 vs 4 end to end, 3000 rows). Full suite 3x: 275 passed each time. go test -race -count=3 ./...: pass. TestParseRowAllocations is now skipped under -race (race_test.go/norace_test.go build tags): it failed under -race before this change too because the race detector adds allocations.

Orchestrator review: decision-32 (default GOGC=400 unless GOGC is set) accepted - memory stays flat and well under 100 MiB; CLAUDE.md architecture updated to describe the parser workers.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Rows are now validated and converted on all cores. ParseData dispatches batches of up to 256 rows to runtime.GOMAXPROCS workers (each with its own compiled representations) that only run parseRow; a single in-order stage keeps every side effect (errors, fail-fast/--max-errors, dropped rows, data failure report, output) exactly as before, so results don't depend on the worker count (decision-31). Profiling showed GC became the limit once parallel, so the binary sets GOGC=400 unless GOGC is set (decision-32). Benchmark on 4 cpus: Avro ~130k -> ~385k rows/s (~3.0x), Parquet ~139k -> ~334k rows/s (~2.4x); without the GC change only ~1.8x for Avro. Peak RSS rises to ~37 MiB (Avro) / ~87 MiB (Parquet), still flat with file size. Added deterministic AC#3 proven by Go tests (workers 1 vs 2/4/8) and tests/test_parallel_parsing.py (GOMAXPROCS=1 vs 4); kill/drain covered by a Go test and the existing TASK-16 prompt-stop tests. Full suite passes 3x, go vet clean, go test -race passes. README Performance updated. Risk: AC#1 figures are machine-dependent.
<!-- SECTION:FINAL_SUMMARY:END -->
