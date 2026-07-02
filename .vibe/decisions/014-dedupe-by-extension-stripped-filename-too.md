---
date: 2026-07-02
status: accepted
---
# Deduplicate registry entries by extension-stripped filename too

**Context:** A `/vibe:review` pass (review-tests) found that `gameFileName` (used by `Save` to name each game's on-disk JSON file) strips the ROM's file extension from its base name, while `indexOf` (the in-memory dedup/matching key backing `Import`, `ImportFromRomsFolder`, `CompleteRomsFolder`, `ImportGame`, `CompleteGame`) still compared the *full* base name, extension included. Two ROMs sharing a base name but differing only by extension (e.g. `Sonic.zip` and `Sonic.iso` in the same system) were therefore treated as two distinct entries in memory, while colliding on the same `Sonic.json` file on disk — the second `Save()` silently overwrote the first game's metadata, with no error and no counter reflecting the loss. This is the same class of bug already fixed once for subfolder prefixes (see `decisions/005`), just triggered by extension instead.

**Decision:** Both `gameFileName` and `indexOf` now derive their key from a new shared `gameID(path string) string` helper (base name, directory prefix and file extension both stripped). Two ROMs that would collide on the same on-disk JSON file are therefore now also recognized as the same registry entry at import time — the second one processed updates the first's metadata (counted as "updated"), rather than silently overwriting it later on `Save()`. `findGameByFilename` (used by `ImportGame`/`CompleteGame` to locate one specific ROM by its exact real disk path) was deliberately left unchanged — it still matches on the full filename including extension, since it identifies one exact physical file the user pointed at, not a dedup key.

**Reason:** Storage is the source of truth for identity here — the registry has always stored at most one JSON file per (system, extension-stripped basename) — so the in-memory dedup key must match that reality, exactly the same reasoning `decisions/005` already established for the subfolder case.

**Rejected alternatives:** Making `gameFileName` retain the extension (e.g. `Sonic.zip.json`) instead, keeping `indexOf`'s existing full-filename matching — rejected because the extension-stripped naming (`Sonic.json`) is an already-established, widely depended-on on-disk format (assumed by ~15 existing test assertions across the `registry` and `cli` packages, and potentially by real users' existing registry folders); changing it would be a bigger, more disruptive fix than aligning the in-memory key to match the existing storage format.
