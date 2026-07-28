---
date: 2026-07-28
status: accepted
---
# An import stands down before a correction saved while it ran

**Context:** The import started from the web UI (backlog item 022) captures the
served registry snapshot, works off a `Clone()` of it for minutes, then persists
that clone and swaps it in. Every other change to the registry does the same —
but instantaneously, inside its own request. The import is the first one whose
"apply to a clone, swap it in" spans long enough for another change to land in
between: a correction saved from the edit form, a deletion, a protection.

**Decision:** The commit takes the write lock, compares the served snapshot with
the one the run captured, and gives up when they differ — writing nothing,
swapping nothing, and reporting "Nothing was written: the registry was corrected
from these pages while the import was running… Run it again."

**Reason:** The clone predates the correction, so persisting it would erase a
value the user had just typed by hand, silently and irrecoverably — the exact
loss `Entry.ManualFields` exists to prevent (decisions/017). Losing the import
instead loses nothing: it read files that are still there, and rerunning it costs
time only. The comparison is a pointer identity check because the swap discipline
already guarantees what that means — a changed registry is always a *new*
`*Registry`, never the same one mutated.

**Rejected alternatives:**
- *Merging the correction into the candidate.* Requires knowing which entries
  changed and why; the registry keeps no such journal, and guessing would
  reintroduce the overwrite the whole rule exists to prevent.
- *Refusing corrections while an import runs.* Punishes the fast, common action
  for the sake of the slow, rare one, and leaves the edit form dead for minutes.
- *Letting the last writer win.* That is the silent data loss itself.
- *Rerunning the import automatically against the new snapshot.* Another few
  minutes of work started without asking, which could lose the race again.
