---
date: 2026-07-28
status: accepted
---
# An interrupted import writes nothing

**Context:** The import started from the web UI (backlog item 022) walks every
configured ROMs folder, merging what it finds into a `Clone()` of the served
registry snapshot, then persists that clone once. A folder that cannot be read
stops it, as it stops the `update` command and as a missing folder stops the
completion.

**Decision:** When a folder fails, the run keeps and reports the counts the
folders before it reached, but the clone is dropped: the registry on disk and the
served snapshot are both left exactly as they were. The report says so in words —
"Nothing was written" — rather than leaving the counts to be read as saved.

**Reason:** `update` behaves this way already (it returns before saving), and two
entry points onto one operation may differ in what they offer, never in what they
mean. The counts still have a job to do: they say how far the run got before the
failure, which is what tells a user whether the folder they just unplugged is the
one at fault. Persisting them would make the report ambiguous instead — a
half-imported registry looks exactly like a complete one, and the next import
cannot tell which entries came from a run that failed.

**Consequence accepted:** "nothing" means nothing the registry *claims* — the
media files an interrupted run had already copied into the registry folder stay
there, referenced by no entry, exactly as `update` leaves them. Deleting them
would mean tracking what this run copied as opposed to what was already there,
and erasing a file the registry might legitimately own is the worse mistake. A
later successful import adopts them.

**Rejected alternatives:**
- *Persisting the folders that succeeded.* Makes the failure invisible in the
  data, and the acceptance criterion asks for the served snapshot to stay
  consistent with what is on disk, not for maximal salvage.
- *Persisting after each folder.* Same objection, plus it multiplies the site
  regenerations — the expensive half of a save — by the number of folders.
- *Carrying on to the next folder after a failure.* That is what the completion
  deliberately does not do (decisions/025); a missing folder is a configuration
  problem worth stopping on, not a per-item error to tally.
