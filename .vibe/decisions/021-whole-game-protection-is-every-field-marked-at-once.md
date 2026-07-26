---
date: 2026-07-26
status: accepted
---
# Whole-game protection is every field marked at once

**Context:** Backlog item 018 asks for a way to state "this game is good, leave it alone" from the CLI
and from the web UI. The per-field protection of [`decision 017`](017-protect-hand-edited-fields-from-later-imports.md)
already exists, but a field is only marked when an edit actually changes its value, so protecting a
correct game meant retyping each of its fields into a different value and back.

**Decision:** Protecting a game sets `Entry.ManualFields` to the whole `editableFields` table; lifting
the protection clears it. It is the existing mechanism at its limit, not a second one — no `Protected`
flag is added to the entry, and the on-disk shape does not change. `Entry.FullyProtected` derives the
state from that same table, so the CLI and the web UI cannot disagree on what "protected" means, and
a mark naming something uneditable is tolerated rather than counted.

Lifting therefore also drops the marks left by earlier hand corrections. The CLI does it
unconditionally and says so in `unprotect --help`; the web UI only offers the lift on a *fully*
protected game, since lifting from the partial state would silently discard which fields the user had
corrected — information that cannot be reconstructed. Selective lifting stays where it already is: the
per-field hand-back checkboxes on the edit form.

**Reason:** A separate flag would create two sources of truth for one question an import has to answer
once (`keepHandEditedFields`), and every consumer would have to remember to consult both. Deriving the
state from the field table instead of storing it also means adding an editable field later cannot leave
a game half-protected without anyone noticing.

**Rejected alternatives:**
- *A `Protected bool` on the entry* — reads better on disk, but forces the import to merge two
  protection rules and lets the two drift; the flag and the marks would eventually disagree.
- *Making the web UI's bulk lift available in the partial state too* — symmetric with the CLI, but it
  destroys the record of which fields were hand-corrected, with no undo and no warning.
- *Marking only the fields the user can see on the game page* — the six displayed rows exclude name and
  description, so a game could read as "protected" while two of its fields were still refreshable.
