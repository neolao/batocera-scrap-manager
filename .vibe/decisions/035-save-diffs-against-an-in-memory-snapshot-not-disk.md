---
date: 2026-07-30
status: accepted
---
# Save diffs against an in-memory snapshot, not disk

**Context:** `Save` rewrote every entry's JSON file on every call, even when a
single game changed — flagged by a `/vibe:review` pass (2026-07-30) as a
performance concern on registries holding several thousand games (backlog
item 027).

**Decision:** `Registry` carries an unexported snapshot of the entries as they
were last known to match the registry folder on disk, set by `Load` and
carried across by `Clone`. `Save` diffs the current entries against that
snapshot (same-value comparison, `Game` via `==`, `ManualFields` via
`slices.Equal`) and writes only the entries that are new or differ from it,
then refreshes the snapshot to the entries it just wrote. A `Registry` built
directly (most existing tests, and any future caller with no known baseline)
has no snapshot and falls back to writing every entry — the same behaviour as
before this change, so it stays a safe default rather than a special case to
maintain.

**Reason:** Every call site already follows the same shape: apply a change to
a `Clone()`, persist it, swap it in on success (`(*Registry).Clone()`'s own
doc). That means the object handed to `Save` is always either a fresh `Load`
or a `Clone` of a registry whose entries are trusted to already match disk —
exactly the reference point a diff needs, without re-reading or `os.Stat`-ing
anything (a separate concern, backlog item 028). Carrying the snapshot on the
`Registry` itself, instead of threading a second registry parameter through
`Save`, keeps every existing call site — CLI and web UI alike — unchanged.

**Rejected alternatives:**
- *Diff against what's actually on disk* (re-read or stat each file at save
  time): correct but reintroduces the same per-entry disk cost this item
  exists to remove, and conflates two separate concerns (item 028 already
  covers the blocking-stat problem).
- *Have every mutating function (`UpdateMetadata`, `Protect`, `WriteMedium`,
  …) report which entries it touched, and thread that set through to `Save`*:
  more precise, but it would make every current and future mutator respons­ible
  for accurately reporting its own blast radius — a much larger surface to get
  wrong than one value-equality check in `Save` itself, for the same result.
