---
date: 2026-07-27
status: accepted
---
# A deletion is committed when the game file is gone, not when the site is regenerated

**Context:** Backlog item 016 exposes the registry's deletion over HTTP. Every other
change the web UI makes follows the same shape: clone the served snapshot, apply the
change to the clone, persist it through `store.Save`, and only swap the clone in once
the write succeeded — so a failed write never leaves the served pages claiming
something the disk does not hold.

Deletion does not fit that shape. `registry.Save` only ever *writes* the entries it is
given; it never deletes a file that left the registry. So the sequence "remove the
entry from the clone, then `store.Save`" deletes nothing at all: the game's JSON and
its media stay on disk and the game comes back on the next start. What actually
persists a deletion is `registry.Remove` erasing the files — which happens *before*
`store.Save` gets a chance to run.

**Decision:** The deletion is committed the moment the game's JSON file is gone.
Under the write lock: remove on the clone (erasing the JSON first, then the media),
swap the clone in as soon as the JSON is gone, then `store.Save`, then answer.

- The JSON could not be erased → nothing happened: `500`, snapshot untouched, and the
  confirmation page re-rendered with the reason so the user can retry.
- The JSON is gone but a media file resisted → the deletion holds: `303`, naming what
  was left behind. `Remove` therefore keeps going through the media instead of
  returning at the first failure, and reports it with `ErrMediaLeftBehind`.
- `store.Save` failed, whatever the reason → the deletion holds: `303` carrying the
  existing stale-site suffix. Answering `500` here would tell the user nothing was
  deleted while the game is already gone.

**Reason:** Swapping the snapshot in *before* `store.Save` is what makes the rest
safe. Were the snapshot left holding a game whose JSON is gone, the next write of any
kind — an edit or a protection change on *another* game — would have `registry.Save`
rewrite every entry of that stale snapshot and resurrect the deleted game's JSON, now
pointing at media that no longer exist.

The order JSON-then-media is kept for the same reason: the opposite failure mode
(a surviving JSON whose media were erased) leaves a game listed with broken images,
which is worse than an unreferenced file nobody can reach.

**Rejected alternatives:**
- *Remove the entry, `store.Save`, then erase the files* — leaves the deletion
  unpersisted until the very last step, and a failure there resurrects the game at the
  next start. Rejected as incorrect, not merely risky.
- *Treat any `store.Save` failure as a failed deletion (`500`)* — the honest report is
  impossible: the files are already gone.
- *Require `Sec-Fetch-Site` or `Origin` to be present on the deletion alone
  (fail-closed), since it is irreversible* — rejected: it would refuse legitimate
  deletions from browsers that send neither, while the edit form, which can overwrite
  a scraped description just as irrecoverably, stays lenient. One cross-site rule for
  every mutation, not one per perceived severity.
