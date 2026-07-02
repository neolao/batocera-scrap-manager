---
date: 2026-07-02
status: accepted
---
# Auto-regenerate the HTML consultation site on `update`, not a dedicated command

**Context:** Backlog item 007 asks for a static HTML site to browse the registry's content (games grouped by system, with name, description, and jaquette). Two designs were considered for when the site gets (re)generated.

**Decision:** The site is regenerated automatically at the end of the `update` command, right after the registry is saved. There is no dedicated `site`/`generate` command.

**Reason:** `update` is the only command that actually mutates the registry (`scrape` only reads it to complete ROMs folders, per [`decisions/004`](004-registry-as-source-to-complete-roms-folders.md)). Tying regeneration to the one command that changes the source of truth keeps the site always in sync without requiring the user to remember an extra step.

**Rejected alternatives:** A dedicated command (e.g. `site`) invoked manually — rejected because it adds a step the user must remember, and the site can silently go stale between an `update` and the next manual regeneration.
