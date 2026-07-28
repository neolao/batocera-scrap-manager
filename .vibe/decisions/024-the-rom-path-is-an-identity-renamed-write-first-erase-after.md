---
date: 2026-07-28
status: accepted
---
# The ROM path is an identity: renaming it writes the new file first and erases the old one after

**Context:** Backlog item 021 makes a game's ROM path readable on its page and correctable from
the edit form. The path is not one more metadata field: `GameID` (`filepath.Base` minus the
extension) derives from it the storage JSON filename, the dedup key, the web UI's URL key and the
matching criterion against `gamelist.xml`. Correcting it therefore moves the game's file inside the
registry folder and changes the address of its own page — while `registry.Save` only ever writes
files and never deletes one, so the file named after the old identifier would survive the rename
and resurrect the game as a duplicate on the next `Load`.

**Decision:**

1. The path stays **out of `editableFields`**. It gets its own domain function
   (`registry.ChangePath`, in-memory only) and its own control in its own fieldset on the form,
   with no `hand-edited` badge and no hand-back checkbox — `ManualFields` is never touched.
2. The rename is ordered **write the new file, then erase the old one**, the reverse of a deletion
   (see [`022`](022-a-deletion-is-committed-when-the-game-file-is-gone.md)): the commit point is
   `registry.Save` succeeding, not the erasure. The served snapshot is swapped in as soon as the
   registry was written, whatever the erasure then does; a leftover old file is reported as a
   caveat carried by the redirect's own outcome value, never as a failed save. No sentinel error
   is introduced for it: the erasure has exactly one call site, which already knows what it just
   tried to erase.
3. Erasing the old file lives in `internal/registry` (`RemoveGameFile`, metadata JSON only, never
   media), so the on-disk naming rule stays in the one package that owns it.
4. A rename that leaves the identifier unchanged (`Sonic.zip` → `sub/Sonic.iso`) skips the erasure
   entirely, and the confirmation does not claim the page moved.

**Reason:** Erasing first — the rule a deletion follows — would, if the write then failed, destroy
the only copy of the game's scraped metadata, which nothing can reconstruct. Erasing last can only
leave a duplicate, which the user can delete. The asymmetry is deliberate: a deletion's intent is
fulfilled by the erasure, a rename's intent is fulfilled by the write. Keeping the path out of
`editableFields` is what keeps the two mechanisms from contaminating each other: a `path` mark
would be silently dropped by `markedFields` (which only iterates `editableFields`), and adding the
row would flip every currently protected game to not-`FullyProtected` — a silent state migration
on disk.

**Rejected alternatives:**

- *Adding `path` to `editableFields`* — rejected: gives an identity the mechanics of a value, and
  silently unprotects every already-protected game.
- *Erasing the old file first, as a deletion does* — rejected: irrecoverable data loss on a failed
  write, against a recoverable duplicate the other way round.
- *Compensating by deleting the new file when `registry.Save` fails mid-iteration* — rejected: one
  more failure branch on the error path, for a case the next `Load` already surfaces as a visible
  duplicate.
- *Renaming the game's media alongside its ROM path* — rejected: media references are stored
  explicitly and relative to the system folder, never derived from `Game.Path`, so nothing breaks.
  Renaming them would mean four field rewrites plus file moves with no rollback, for a purely
  cosmetic gain.
- *Linking to the conflicting game from the collision message* — rejected against the UI/UX
  expert's advice: the message is rendered both in the field and inside the error summary, whose
  entries are themselves links to `#field-path`; an anchor nested in an anchor is invalid HTML and
  breaks keyboard navigation. The message names the conflicting file name and the game holding it
  instead.

**Known consequences, not addressed here:** a later `update` re-importing a ROMs folder whose
`gamelist.xml` still carries the old path will overwrite the correction (same identifier) or add a
second entry beside it (changed identifier) — `Path` is not protectable, by design. On a
case-insensitive registry folder (exFAT, SMB), a rename differing only by case passes the duplicate
check and overwrites another game's file; unguarded, judged low-likelihood on a Batocera setup.
