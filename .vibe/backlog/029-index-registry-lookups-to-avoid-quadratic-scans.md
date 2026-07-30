---
status: todo
---
# Index Registry Lookups To Avoid Quadratic Scans

## Description
`Registry.indexOfID` does a linear scan of the whole `Entries` slice. `Import`/`CompleteRomsFolder` (and the single-game `mergeGameEntry`/`fillGameFromRegistry` they call) each call it once per game processed, so importing or completing a ROMs folder against a registry that already holds thousands of entries costs `O(games scanned × registry size)` instead of near-linear. Flagged by a `/vibe:review` pass (2026-07-30) as quadratic on registries described elsewhere as holding "several thousand games".

## Acceptance Criteria
- [ ] Looking up a game by `(system, id)` during import or completion no longer scans the whole registry linearly for every game processed
- [ ] Import and completion still produce the exact same added/updated/unchanged (or processed/completed/failed) counts as before on every existing test fixture
- [ ] Any index introduced stays correct across every mutation site of `Registry` (an entry appended mid-import, `RemoveByID`, a path change re-keying an entry, `Clone()`) — no stale or missing entries after any of them

## Notes
This was deliberately *not* attempted in the same review pass: a maintained index has to stay in sync across every place `registry.go` mutates `Entries` directly, which is a change to the package's core invariants — worth designing deliberately (e.g. a `map[string]int` keyed by `system + "\x00" + id`, built once per `Import`/`CompleteRomsFolder` run and kept current as entries are appended) rather than retrofitting defensively. `FindByID`, used on every web UI request, goes through the same `indexOfID` and would benefit too, but is lower urgency since it's one lookup per request rather than one per game of a batch run.
