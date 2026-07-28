---
status: done
---
# Update A Single Game By Its Path

## Description
The `update` command currently imports into the registry every game of every configured ROMs folder. An option must make it possible to target a single game by giving its path, so only that game gets imported or updated in the registry, without processing the whole folder.

## Acceptance Criteria
- [ ] The user can run `update` giving the path of one game, and only that game is imported or updated in the registry
- [ ] If the given game does not exist, or has no matching entry in its system's `gamelist.xml`, a clear error message is displayed and the command returns an error code
- [ ] The displayed summary stays consistent with that targeted mode (e.g. 1 added, or 1 updated, or 0 unchanged depending on the case)
- [ ] Without that option, the existing behavior (updating every game of every configured ROMs folder) stays unchanged

## Notes
Symmetrical to item 011 (targeted scrape of one game by path), but in the opposite direction (ROMs folder → registry). To settle: is the given path relative to the configured ROMs folder, or an absolute path on disk? How to derive the targeted game's system (its parent subfolder) also needs to be determined.
