---
id: TASK-14
title: Support streaming output to stdout
status: Done
assignee:
  - '@claude'
created_date: 2026-10-08 18:05
updated_date: '2026-11-01T11:24:00Z'
labels:
  - output
  - cli
dependencies: []
priority: low
type: feature
ordinal: 14000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The old notes say output should be able to stream to stdout, so the tool can sit in a shell pipeline. By default a run writes output to a temporary file and only moves it into place on success, so a failed run leaves no output. Streaming cannot work that way, because data is emitted before the whole file has been validated.

Decision: support both behaviours, chosen with a `--stream` flag. Without it, behaviour stays as it is now (all or nothing). With it, output is written as rows are validated, and on failure the output is incomplete; the non-zero exit code is the signal consumers must check.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Given --stream and no --output-path, when parse runs, then the output is written to stdout as rows are processed
- [x] #2 Given --stream and an invalid file, when parse runs, then it exits non-zero and stderr says the output written is incomplete
- [x] #3 Given no --stream, when the run fails, then no output is left behind (current behaviour unchanged)
- [x] #4 Given --stream, when parse runs, then errors and warnings go to stderr only, so stdout carries only output data
- [x] #5 README documents --stream and the incomplete-output trade-off
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
1. Add Stream and OutputPath to options.Options; Validate: --output-path required unless --stream.
2. main.go: --stream flag, --output-path no longer Required at cli level. parse(): without --stream unchanged (temp+rename). With --stream: write the OCF straight to stdout (no -o) or straight to the -o path (no temp/rename). Side files (data failure report, json report) stay atomic.
3. avro_writer.WriteFile takes an io.Writer instead of a path; stdout wrapped in a plain io.Writer (no bufio) so goavro writes the header at once and a block per Append, and doesn't try to read an existing header from a stdout redirected to a non-empty file.
4. On failure with --stream, append an 'output written to <stdout|path> is incomplete' line to the error (stderr).
5. Audit stdout use in parse (none expected); everything else stays on stderr.
6. tests/test_streaming.py: AC#1 (stdout readable with fastavro, incremental emission via a FIFO input read while the process still runs), AC#2, AC#3, AC#4, header with zero rows, -o with --stream, -o required without --stream.
7. README (--stream docs, trade-off, Planned bullet) and CLAUDE.md.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Decisions: decision-21 (--stream with -o writes the path in place, incomplete on failure, consistent with stream semantics and simpler than rejecting it; stdout selected by leaving out -o, '-o -' not special to keep it simple), decision-22 (-o required without --stream; incomplete-output line on stderr for every --stream failure, including ones before any row is written, e.g. bad header; side reports keep temp+rename).
Buffering: avro_writer.Write now takes an io.Writer; goavro writes the OCF header in NewOCFWriter (so zero rows still gives valid Avro) and one block per Append. Stdout is passed unbuffered, wrapped in struct{io.Writer} so goavro doesn't treat it as an *os.File and try to append to an existing OCF when stdout is redirected to a non-empty file. Proved by a FIFO-input test that reads a record from stdout while the input is still open; a mutation with bufio-wrapped stdout makes it fail.
Stdout audit (parse, with and without --stream): nothing in main.go, error_tracking, parser, reader or avro_writer writes to stdout; the console report, dropped-rows warning and errors go to stderr (via ExitErrHandler / Fprintln(os.Stderr)). Outside scope: urfave/cli usage errors (unknown flag, missing -c/-i) print 'Incorrect Usage' + command help to stdout before parse runs, which would land in a pipeline's consumer. Not changed here; possible follow-up (e.g. OnUsageError / help to stderr on error).
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added --stream to parse. Without it behaviour is unchanged (temp file + rename, -o required). With it the Avro output is written as rows are validated: to stdout when --output-path is left out, otherwise to the path in place; on failure the exit is non-zero and stderr says the output written is incomplete. avro_writer.Write takes an io.Writer (stdout unbuffered); options gained OutputPath/Stream with validation. Side reports stay atomic. Tests: tests/test_streaming.py (AC#1 stdout Avro, header with zero rows, FIFO incremental-emission test, late-error prefix; AC#2; AC#3; AC#4; AC#5 README). README (Streaming output section, Planned now only Parquet) and CLAUDE.md updated. 242 pytest passed, go vet clean. Follow-up: urfave/cli usage errors print help to stdout.
<!-- SECTION:FINAL_SUMMARY:END -->
