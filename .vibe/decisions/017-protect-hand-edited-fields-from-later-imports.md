---
date: 2026-07-26
status: accepted
---
# Protect hand-edited fields from later imports

**Context:** Editing a game's metadata from the web UI (backlog item 015) creates exactly the
difference `(*Registry).Import` resolves in favour of the ROMs folder: `mergeGameEntry` replaces a
registry entry whose metadata differs from the local `gamelist.xml`. A correction made in the
browser would therefore be silently reverted by the next `update`, counted as an ordinary "updated"
entry — the feature would look broken to the user who runs `update` regularly.

**Decision:** A registry entry now records which of its fields were set by hand (`Entry.Manual`,
persisted as `manual_fields` in the game's JSON file). An import applies the incoming game as
before, then restores every hand-edited field from the stored entry, so those values are never
overwritten. Marking is done by the edit flow, and only for fields whose value the save actually
changes; the edit form can also hand a field back to the scraper by unmarking it.

The on-disk shape stays backward compatible: the game's fields are still written flat at the root of
its JSON file, `manual_fields` is an additional optional key, and a file written by an earlier
version loads as an entry with nothing protected.

**Reason:** The protection has to live in the registry, because that is where the conflict is
resolved — any guard placed in the web UI would be bypassed by the CLI. Marking only the fields
that actually changed keeps the protection an explicit statement of intent ("this value is mine")
rather than a side effect of opening a form and pressing Save.

**Rejected alternatives:**
- *Document the limitation and do nothing* — the correction survives only until the next `update`,
  which makes the editing feature misleading rather than incomplete.
- *Write the correction back into the ROMs folder's `gamelist.xml`* — it does remove the conflict at
  its root and would benefit Batocera itself, but it makes the web UI write into the user's source
  folders, which so far only the `scrape` command does, and it needs the game to be located again in
  the configured ROMs folders. Kept as a possible follow-up.
- *Never overwrite an existing registry field* — breaks legitimate re-scrapes, which are the whole
  point of `update`.
