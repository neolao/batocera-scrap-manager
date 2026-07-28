---
status: done
---
# Remove An Entry From The Registry

## Description
The project must expose a CLI command removing one given game (one "scrap") from the registry, with all its associated data: its metadata record and the media files (cover art, video, marquee, thumbnail) already copied into the registry. This makes it possible to clean up an obsolete or incorrect entry, or one matching a game removed from the ROMs folders, without editing the registry by hand on disk.

## Acceptance Criteria
- [ ] The user can run a dedicated CLI command naming one given game (by system and ROM path) to remove from the registry
- [ ] The system deletes the metadata record and every media file associated with that game in the registry
- [ ] The system confirms the removal to the user
- [ ] Attempting to remove a game absent from the registry returns a clear error, without modifying the rest of the registry

## Notes
Left to be defined: how the user names the game to remove (system + exact ROM path, name, or interactive selection). No blocking dependency identified: the registry (items 001-002) is already in place.
