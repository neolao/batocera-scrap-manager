---
status: in_progress
depends_on: [015]
---
# View And Edit The ROM Path

## Description
In a Batocera `gamelist.xml`, a game is identified by its ROM path — the filename and its optional subfolder — not by its displayed name. That path appears nowhere in the web UI today: a game's page shows metadata only, and the metadata correction deliberately excludes the `Path` field. The path must be readable on a game's page and correctable from the edit form, with the registry reorganizing itself accordingly.

## Acceptance Criteria
- [ ] A game's page displays the ROM path as stored (relative to the system folder, subfolder included), and the edit form offers it as an editable control pre-filled with that same value
- [ ] After a new path is saved, the entry is stored under the JSON file derived from that path (the old file no longer exists), and the game's page stays reachable at the URL matching the new identifier
- [ ] A rejected path — empty, absolute, or escaping the system folder through `..` — produces an explicit error message and leaves the registry strictly unchanged
- [ ] A path whose derived identifier already belongs to another game of the same system is rejected with an explicit message, without overwriting the existing entry

## Notes
The path is not one more metadata field: `registry.GameID` (`filepath.Base` minus the extension) derives from it the storage JSON filename, the dedup key, the web UI's URL key, and the matching criterion against the games of `gamelist.xml` (`internal/registry/registry.go`). Changing it therefore means moving the game's file inside the registry folder and redirecting to the new URL, while honoring the write-then-swap rule: replace the served snapshot only once the write succeeded. The field is absent from `editableFields` (`internal/registry/metadata.go`) by design — adding it there as-is would also give it the `ManualFields` mechanics, which do not mean the same thing for an identity as for a value; open question to settle at implementation time. Another open question: a corrected path will no longer match the file actually present in the ROMs folder, and a later `update` may recreate an entry under the old name — decide whether the operation should rename the associated media and/or warn the user. Like every registry change, persist through `internal/store` so the static site gets regenerated.
