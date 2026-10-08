---
id: TASK-16
title: Test performance on a large file
status: To Do
assignee: []
created_date: '2026-10-08 18:05'
labels:
  - performance
dependencies: []
priority: low
type: task
ordinal: 16000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The tool is meant to sit at the start of production data pipelines, but has only been tested on tiny fixtures. "Test large file" was on the old notes to-do list. Worth checking throughput, memory use, and that error handling (fail-fast, max-errors) stops promptly on a large bad file.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Throughput and peak memory are measured on a file of at least 1 million rows and recorded in the task notes
- [ ] #2 A large file with an early error stops promptly with default flags
- [ ] #3 Any significant bottlenecks found have follow-up tasks
<!-- AC:END -->
