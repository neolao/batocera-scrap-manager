---
date: 2026-09-10
status: accepted
---
# `<hidden>` is modelled on `documentGame`, not on `Game`

**Context:** The ROMs folder status report needed to ignore a game EmulationStation itself marks `<hidden>true</hidden>` — until now an unmodelled element, invisible to any typed code, preserved on rewrite only through the generic "everything this package does not understand" mechanism.

**Decision:** A `Hidden` field was added to the package-private `documentGame` type (`internal/gamelist/gamelist.go`), read by a new `ParseWithVisibility`, rather than to the exported `Game` struct every `ParseFile`/`Write` caller already uses.

**Reason:** `Game` is embedded, unchanged, into `registry.Entry` and stored flat in the registry's own JSON files. Adding `Hidden` there would silently grow the registry's on-disk shape and hand every existing caller (Import, Completion, the edit form) a field they have no reason to carry — the same boundary that already keeps a favourite mark or a play count out of `Game` for the exact same reason. `ParseWithVisibility` is the one entry point that reads it, used only by the status report.

**Rejected alternatives:** Adding `Hidden bool` to `Game` directly — simpler call sites, but it would leak local ROMs-folder curation into the registry's data model, and every unrelated caller of `ParseFile`/`Write` would start seeing (and needing to round-trip) a field it never asked about.
