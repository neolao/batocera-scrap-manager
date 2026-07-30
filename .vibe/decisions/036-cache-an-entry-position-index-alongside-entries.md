---
date: 2026-07-30
status: accepted
---
# Cache an entry position index alongside Entries

**Context:** `indexOfID` — the lookup every registry mutation (`FindByID`, `UpdateMetadata`, `Protect`/`Unprotect`, `ChangePath`, `WriteMedium`/`ClearMedium`, and each game merged by `Import`/`CompleteRomsFolder`) goes through — did a linear scan of `Registry.Entries`. `Import`/`CompleteRomsFolder` each call it once per game processed, so processing a ROMs folder against a registry holding thousands of entries cost `O(games processed × registry size)` (flagged by a `/vibe:review` pass, 2026-07-30, backlog item 029).

**Decision:** Add an unexported `index map[entryKey]int` field to `Registry`, mapping an entry's identity (the existing `entryKey{system, id}` `Save` already uses to diff against its saved baseline) to its position in `Entries`. It is built lazily by `ensureIndex()` on first use — first-match-wins, mirroring `indexOfID`'s own linear-scan semantics if `Entries` ever holds a duplicate key — so a `Registry` built directly by a struct literal (the ~168 existing test constructions, and `Load`) needs no explicit initialization step, the same pattern already used for the `saved` field.

Every mutation site of `Entries` or of a `Game.Path` keeps the index in sync:
- `mergeGameEntry`'s append inserts the new entry's key at its new position (the index is already built at that point, since the caller just resolved `indexOf` earlier in the same function).
- `ChangePath` deletes the old key and inserts the new one at the same position — its own duplicate check already forces `indexOfID` to run immediately before, so the map is guaranteed accurate at rekey time.
- `RemoveByID`'s slice splice shifts every later entry's position by one; rather than rewrite up to N map values, it invalidates the whole index (`reg.index = nil`) and lets the next lookup rebuild it. `RemoveByID` is not in the per-game batch loop `Import`/`CompleteRomsFolder` run, so this trade does not reintroduce the quadratic cost the index exists to remove.
- `Clone()` deep-copies the map key by key rather than assigning it, since `copy(clone.Entries, r.Entries)` already preserves every entry's position 1:1 — a bare assignment would alias the same map and let a change applied to a clone (the pattern every `webui` write already follows: apply to a `Clone()`, persist, swap in) leak back into the original.

**Reason:** A cached position map turns the hot lookup into O(1) without changing what any exported function returns, and reusing `entryKey` (rather than a second, ad-hoc string encoding of the same identity) keeps the package's notion of "what identifies an entry" defined in exactly one place.

**Rejected alternatives:** Incrementally shifting every affected map value on `RemoveByID` instead of invalidating — rejected as unnecessary complexity for a call that is not in the quadratic hot path. A `map[string]int` keyed by a `system + "\x00" + id` string — rejected in favor of the already-existing `entryKey` struct, which sidesteps any separator-collision question and avoids a second competing key encoding.
