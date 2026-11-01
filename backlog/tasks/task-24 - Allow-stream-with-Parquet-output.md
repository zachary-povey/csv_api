---
id: TASK-24
title: Allow --stream with Parquet output
status: Done
assignee:
  - '@claude'
created_date: '2026-11-01T17:54:00Z'
updated_date: '2026-11-01T18:04:00Z'
labels:
  - output
  - cli
dependencies:
  - TASK-23
priority: medium
type: feature
ordinal: 24000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Agreed with the user, replacing decision-26 (--stream rejected with --output-format parquet). The user's main use case is piping Parquet output to another tool (e.g. `| aws s3 cp - s3://...`). Parquet writes row groups sequentially and its footer last, and parquet-go can write to a non-seekable io.Writer, so streaming to stdout works: the consumer gets a complete file when the run succeeds. Readers cannot use a Parquet file before its footer arrives, so streaming does not give early access to rows; its value is pipe-friendliness and bounded memory. On failure the output has no footer, so Parquet readers reject it, which is a stronger signal than truncated Avro; the non-zero exit code remains the signal consumers must check (TASK-14).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Given --stream --output-format parquet and no --output-path, when parse runs on a valid file, then a complete, readable Parquet file is written to stdout with the same values as a non-streamed run
- [x] #2 Given --stream --output-format parquet and an invalid file, when parse runs, then it exits non-zero, stderr says the output is incomplete, and nothing after the failure is written as a footer
- [x] #3 Given --stream --output-format parquet with --output-path, when parse runs, then that path is written directly as with Avro streaming (decision-21)
- [x] #4 Given --stream --output-format parquet and an input larger than one row group, when parse runs, then completed row groups are written to the output before the input has finished
- [x] #5 Given --stream --output-format parquet, when parse runs, then stdout carries only Parquet bytes (errors and warnings go to stderr)
- [x] #6 Given the README, when a user reads the streaming and output format docs, then Parquet streaming is documented, including that a Parquet file is only readable once complete
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
1. options.Validate: drop the rule rejecting --stream with parquet; replace test_stream_is_rejected_with_parquet with tests of the new behaviour.
2. error_tracking: add ErrorTracker.Failed() (any execution or report error recorded, under the lock).
3. parquet_writer.Write: write through our own bufio.Writer (parquet-go's internal 32KiB buffer off via WriteBufferSize(0)); flush a row group (writer.Flush + our buffer) every rowsPerRowGroup rows so completed row groups reach the output as they are written; rowsPerRowGroup becomes a package var so Go tests can use a small size (no CLI flag). After the channel closes, if errTracker.Failed() return without Close/flush: no footer, so the partial output is unreadable as Parquet. Today the failure path closes the writer and writes a valid footer over the partial rows (harmless only because the temp file is removed).
4. Go tests in internal/parquet_writer: row groups reach a non-seekable writer before the channel closes (AC#4); a failed run writes no footer (AC#2).
5. pytest (tests/test_parquet_streaming.py): stdout streaming equals non-streamed values (AC#1); invalid file -> non-zero, incomplete message, no footer, unreadable, incl. one failing after a full 100k row group (AC#2); --stream -o written directly (AC#3); fifo input held open -> bytes on stdout before input ends (AC#4); stdout only Parquet with warnings/errors (AC#5); README test (AC#6).
6. Audit stdout writes on the parquet path (AC#5).
7. Docs: README streaming + output format sections, CLAUDE.md. Run pytest, go vet, go test -race. Mark decision-26 superseded.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
- parquet-go v0.32.0 writes to any io.Writer (offsets tracked internally, no Seek) and flushes a row group when full, but through its own 32 KiB bufio, which could hold the tail of a completed row group back. Write now turns that buffer off (WriteBufferSize(0)), writes through its own 64 KiB bufio and flushes row group + buffer itself every rowsPerRowGroup rows (MaxRowsPerRowGroup option dropped: one mechanism).
- Failure path before this change: the parser closes the channel after a kill, and Write called Close(), writing a valid footer over the partial rows (harmless only because the temp file was removed). Now Write checks the new ErrorTracker.Failed() once the channel is closed (every pipeline error is recorded before the parser closes it) and returns without Close/flush, so no footer is written.
- rowsPerRowGroup is a package var (tests set it to 2); no CLI flag.
- AC#5 audit: on the parse path only openOutput's stdout writer writes to stdout; parquet-go only logs (stderr) or prints from internal/debug wrappers used in its own tests.
- Residual edge (not in ACs): an error after the writer has finished (closing the output file, flushing the data failure report) fails the run with the footer already written; the exit code still signals it.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
--stream now works with --output-format parquet (replacing decision-26). options.Validate no longer rejects it. parquet_writer.Write writes each completed row group (100k rows) straight through to the output via its own buffer, works with non-seekable stdout, and writes no footer when the run has failed (new ErrorTracker.Failed()), so incomplete streamed Parquet is rejected by readers. Tests: tests/test_parquet_streaming.py (AC#1-6, incl. a fifo test and a failure after a full row group) and internal/parquet_writer/parquet_writer_test.go (AC#4 row group bytes reach a non-seekable writer before the channel closes; AC#2 no footer). README streaming/output format sections and CLAUDE.md updated. pytest 301 passed, go vet clean, go test -race clean.
<!-- SECTION:FINAL_SUMMARY:END -->
