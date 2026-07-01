---
date: 2026-07-02
status: accepted
---
# Reporting import progress via a callback

**Context:** The `update` command was silent during its entire execution, only printing a final summary once every ROMs folder had been fully processed (backlog item 004). For a large collection with potentially heavy media files, this gives no feedback while the command is actually working.

**Decision:** `registry.ImportFromRomsFolder` takes an additional `onProgress func(ProgressEvent)` parameter, invoked once per game as it is processed (with the system name, the game's 1-based index and the total game count for that system, and the game's name). It is nil-safe: callers that do not care about progress (most existing tests) pass `nil` and nothing is invoked. The `update` CLI command supplies a callback that prints one line per system started and one line per game processed, using plain sequential output (no carriage-return overwrites, no ANSI cursor codes), so redirecting the output to a file or a pipe stays readable.

**Reason:** A callback keeps the progress-reporting concern fully optional and out of the registry package's core responsibility (it does not know how to print), while requiring the smallest possible change to the existing `Load`/`Import`/`Save` flow. Plain sequential lines (rather than an animated progress bar) satisfy the requirement to remain usable in non-interactive environments without any extra logic to detect whether output is a terminal.

**Rejected alternatives:** A channel-based progress stream — rejected as unnecessary concurrency complexity for a single-threaded, synchronous import loop. An animated single-line progress bar (carriage-return overwrites) — rejected because it would clutter output when redirected to a file, which acceptance criteria explicitly rule out.
