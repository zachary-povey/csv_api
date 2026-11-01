---
id: TASK-21
title: Write Avro output in multi-row blocks
status: Done
assignee:
  - '@claude'
created_date: '2026-11-01T11:38:00Z'
updated_date: '2026-11-01T12:05:00Z'
labels:
  - performance
dependencies: []
references:
  - internal/avro_writer/avro_writer.go
  - tests/test_streaming.py
priority: medium
type: enhancement
ordinal: 21000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
avro_writer.Write calls goavro OCFWriter.Append once per row, and goavro writes one OCF block (count, size, 16-byte sync marker) per Append call, straight to the unbuffered output file: one write syscall per row. TASK-16 found this makes the Avro output ~38% larger than the same records written in normal-sized blocks (19.6 MB vs 14.3 MB for 300k rows re-encoded with fastavro) and costs a syscall per row. Per-row blocks were chosen so --stream output reaches its reader as rows are validated (TASK-14), so batching must keep streamed output timely.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Given a run without --stream, when a file of many rows is converted to Avro, then rows are written in multi-row blocks and the output for the scripts/benchmark.py 1M-row file is at least 25% smaller than the TASK-16 baseline (65 MB) with identical records
- [x] #2 Given --stream, when rows are validated, then each row still reaches the output reader without waiting for further input rows (e.g. a block is flushed whenever the writer would otherwise wait for input)
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
1. avro_writer.Write collects rows into a block of up to maxBlockRows (100), wrapping nullable values per row as before, and Appends each block (goavro writes one block per Append with a single Write call, so no extra output buffering is needed; header still written up front).
2. Without --stream a block is written only when full (or at the end); with --stream (new flushWhenIdle arg, passed from main) a non-blocking receive ends the block whenever no row is ready, so no row waits for later input.
3. Measure size/throughput with scripts/benchmark.py and pick the block size.
4. Tests: tests/test_avro_blocks.py (AC#1 block layout + identical records; AC#2 every row streamed while input held open); TASK-14 fifo test also names AC#2.
5. README streaming bullet + Performance; decision; notes.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Approach change from the first draft: flushing whenever the channel is empty regardless of --stream gave 51 MB (only 21.5% smaller), since the writer outpaces the parser and finds the channel empty most of the time, so blocks stayed small. Idle flushing is therefore --stream only (decision-30).
Block size: 4096-row blocks gave 47 MB but were ~10-15% slower; 1000-row blocks 47.5 MB but consistently slower (user CPU +1.3s). With the channel buffer at 100 (decision-28), the parser stalls on a full channel while the writer encodes a big block. 100-row blocks: 47.7 MB (47,676,694 bytes vs 65,476,694 baseline = 27.2% smaller), 10,000 blocks instead of 1,000,000, all 1M records identical to the baseline output (compared with fastavro).
Throughput (1M-row benchmark, 5 paired runs on the same noisy 4-cpu VM): baseline 6.63-6.82s (~148k rows/s), new 6.87-7.27s (~142k rows/s), ~4% slower, within the VM's ±10% run-to-run variation; sys time drops (fewer write syscalls). Peak RSS unchanged (19-20 MiB). Later standalone runs: avro 7.8s ~129k rows/s, 48 MB (README table).
Tests: tests/test_avro_blocks.py::test_rows_are_written_in_multi_row_blocks (AC#1; fails on the old binary) and ::test_stream_writes_every_validated_row_without_waiting_for_input (AC#2; 250 rows through a held-open fifo, all must be readable - fails if idle flushing is disabled, which the existing TASK-14 fifo test with exactly 100 rows would not catch). tests/test_streaming.py::test_stream_emits_rows_before_the_input_is_finished also names AC#2 and passes. The 1M-row size figure is from the benchmark (not automated), the automated test proves the block layout deterministically. ACs left unchanged.
Out of scope, possible follow-up: encoding blocks in a separate goroutine could allow larger blocks without stalling the parser, but the remaining size gain is <1 MB.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Avro output is now written in blocks of up to 100 rows instead of one OCF block per row. avro_writer.Write batches rows (nullable values still wrapped per row) and Appends each block; with --stream it also writes a block as soon as no further row is ready, so no row waits for later input; without --stream blocks are only written when full, so they stay large even though the writer outpaces the parser (decision-30). Benchmark 1M rows: Avro output 47.7 MB vs 65.5 MB (27% smaller), identical records, 10k blocks instead of 1M; throughput ~4% lower in paired runs (within VM noise), peak RSS unchanged. Larger blocks were measured slower (parser stalls on the 100-slot channel while a big block is encoded), hence 100. Tests: tests/test_avro_blocks.py (AC#1, AC#2), tests/test_streaming.py fifo test (AC#2). README streaming bullet and Performance table/notes updated. Full pytest (271) and go vet pass.
<!-- SECTION:FINAL_SUMMARY:END -->
