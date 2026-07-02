---
date: 2026-07-02
status: accepted
---
# Generate the consultation site at the registry root, and never auto-clean the old location

**Context:** Backlog item 008 asks that the consultation site's `index.html` (introduced by item 007, generated under a `site/` subfolder — see [`decisions/006`](006-auto-regenerate-html-site-on-update.md)) be generated directly at the registry's root instead, to be easier to find and open.

**Decision:** `site.Generate` now writes `<registryFolder>/index.html` directly, instead of `<registryFolder>/site/index.html`. Jaquette references are adjusted from `../<system>/<image>` to `<system>/<image>` to match the new location. A leftover `site/` folder from a previous version is never touched automatically — it is left as-is if present.

**Reason:** The new root location is what was explicitly requested (easier to find/open). Leaving a pre-existing `site/` folder untouched avoids an automated command silently deleting a folder it did not create in the current run — a destructive action that was explicitly asked to be excluded from scope.

**Rejected alternatives:** Automatically deleting the old `site/` folder on the next `update` — rejected per explicit product decision, to keep `update` from performing unprompted destructive cleanup.
