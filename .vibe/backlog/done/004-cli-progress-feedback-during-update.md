---
status: done
---
# CLI Progress Feedback During Update

## Description
The `update` command currently stays silent for its whole run: it only speaks once everything has been processed, printing a single final summary ("X added, Y updated, Z unchanged"). On a large game collection, with potentially heavy media copies, that absence of feedback can make the tool look stuck. Progress must therefore be displayed during the run, not only at the end.

## Acceptance Criteria
- [ ] The user sees the name of the system being processed as `update` walks the configured ROMs folders
- [ ] The user sees a per-game progress indicator (e.g. a "game X of Y" counter, or the name of the game being processed) while a system is processed
- [ ] The final summary ("added/updated/unchanged") keeps being displayed at the end of the run, unchanged
- [ ] The output of `update` stays usable in a non-interactive environment (no rendering that breaks the output when redirected to a file or a pipe)

## Notes
The `update` command (`internal/cli/update.go`) and the import (`registry.ImportFromRomsFolder`) are already implemented (item 002); that is not a blocking dependency since they already exist, but this item will instrument them to emit progress as the run goes rather than all at once at the end. The exact level of detail (per system only, or per game as well) is to be refined with the user at implementation time.
