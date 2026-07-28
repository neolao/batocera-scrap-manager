---
status: done
---
# Show The ROMs Folder During Scrape

## Description
While the `scrape` command runs, the user currently sees which system and which game are being completed, but not which ROMs folder (among the configured ones) the change is applied to. When several ROMs folders are configured, that piece of information is missing to locate where each change happens. The command must therefore also state, in its live output, which ROMs folder is being updated.

## Acceptance Criteria
- [ ] With a single configured ROMs folder, the user still sees clearly which folder is being updated during the scrape
- [ ] With several configured ROMs folders, the live output states unambiguously which folder each displayed change belongs to
- [ ] The final summary (processed / completed / failed) keeps its format unchanged

## Notes
Builds on the completion feature already in place (the `scrape` command, see item 003 and the revision of the filtered output). Left to be defined: is the folder printed once per folder (like the per-system header), or repeated on every game line.
