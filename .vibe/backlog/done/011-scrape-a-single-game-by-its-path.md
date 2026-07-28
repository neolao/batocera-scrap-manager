---
status: done
---
# Scrape A Single Game By Its Path

## Description
The `scrape` command currently completes every game of every configured ROMs folder. An option must make it possible to target a single game by giving its path, so only that game gets completed from the registry, without processing the whole folder.

## Acceptance Criteria
- [ ] The user can run `scrape` giving the path of one game, and only that game is completed from the registry
- [ ] If the given game has no matching entry in the registry, a clear error message is displayed and the command returns an error code
- [ ] The displayed summary stays consistent with that targeted mode (e.g. 1 processed, 1 completed or 0 completed depending on the case)
- [ ] Without that option, the existing behavior (completing every game of every configured ROMs folder) stays unchanged

## Notes
Builds on the completion mechanism already in place (`registry.CompleteRomsFolder`, item 003). To settle: is the given path relative to the configured ROMs folder, or an absolute path on disk? How to derive the targeted game's system (its parent subfolder) also needs to be determined.
