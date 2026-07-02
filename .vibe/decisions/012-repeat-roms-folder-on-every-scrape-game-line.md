---
date: 2026-07-02
status: accepted
---
# Repeat the ROMs folder on every scrape game line, not as a per-folder header

**Context:** Backlog item 006 requires the `scrape` command's live output to show which ROMs folder each change belongs to, since the existing per-system header (`"<system>: <N> game(s)"`) resets per ROMs folder but never names the folder itself — ambiguous when two configured ROMs folders share a system name.

**Decision:** Include the ROMs folder path directly on every per-game progress line (`"  [<index>/<count>] <romsFolder>: <name>"`), rather than printing the folder path once as a header before each folder's block of systems/games.

**Reason:** The user explicitly preferred repetition per line over a per-folder header, for unambiguous attribution of every single printed change to its folder without relying on the reader tracking which header block they are currently under.

**Rejected alternatives:** A one-time header line per configured ROMs folder (printed before that folder's systems), mirroring the existing per-system header pattern — rejected by the user in favor of the per-line repetition.
