---
status: done
---
# Regenerate The Site After A CLI Remove

## Description
The `remove <system> <rom-filename>` command does delete the registry entry and its media from the folder, but does not regenerate `index.html` — unlike `update`, and unlike the deletion from the web UI (item 016). The static site therefore keeps displaying the deleted game, with a cover art pointing at a file that no longer exists, until the next `update`. This is a pre-existing behavior, discovered while implementing item 016.

## Acceptance Criteria
- [ ] After a successful `remove`, the static `index.html` site no longer holds the deleted game, without having to run `update`
- [ ] If the registry was indeed modified but the site could not be regenerated, the command states what was deleted and reports the site as stale, rather than suggesting nothing happened
- [ ] A failing `remove` (unknown game) regenerates no site and changes nothing

## Notes
`internal/cli/common.go` already has `saveAndGenerateSite`, a thin wrapper over `store.Save` that `update` and `scrape` use — `runRemove` is the only one not going through it. Mind the ordering: `registry.Remove` already erases the game's files before `store.Save` comes in, so the deletion is committed from that moment on; a failed site regeneration must not be presented as a failed deletion (the same reasoning as [`decisions/022`](../../decisions/022-a-deletion-is-committed-when-the-game-file-is-gone.md), and the `store.ErrSiteNotRegenerated` sentinel already exists to tell that case apart). See also `registry.ErrMediaLeftBehind`, which `runRemove` already treats as a warning rather than an error.
