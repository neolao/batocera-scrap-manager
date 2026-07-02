---
date: 2026-07-02
status: accepted
---
# Skip games with no scraped data only on import, not retroactively

**Context:** Backlog item 009 asks that games with neither a description nor any jaquette (no scraped data at all) no longer be added to the registry during `update`.

**Decision:** The exclusion only applies when a game is about to be newly added to the registry. A game already present in the registry is never removed as a side effect of `update`, even if its local `gamelist.xml` later loses its description and jaquette.

**Reason:** `update`'s job is to bring new/changed information into the registry, not to prune it — retroactive removal as a side effect of a routine import would be surprising and risks silently destroying already-collected data (e.g. if a local `gamelist.xml` is temporarily reset by an external tool). Removing an entry already has a dedicated, explicit command (`remove`, backlog item 005).

**Rejected alternatives:** Also removing existing registry entries that lose their last remaining data — rejected because it conflates "importing new information" with "pruning stale entries", and would make an already-destructive operation (data loss) implicit and hard to predict from `update`'s output.
