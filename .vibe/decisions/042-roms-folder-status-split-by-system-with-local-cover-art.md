---
date: 2026-11-18
status: accepted
---
# ROMs folder status is split by system, and serves the folder's own cover art

**Context:** The ROMs folder status page rendered every system's problem games on one page, with no pagination and no images. On a real folder spanning many systems, this made it impractical to use.

**Decision:** `/roms-folder?folder=F` becomes a summary table with one row per system. `&system=S` renders that system alone: every game, complete ones included, at 60 games a page, with the folder's cover art and the registry's shown side by side. The folder's cover art is served by `GET /roms-folder/media/{system}/{path...}?folder=F`. That route accepts only a configured folder and image extensions, and opens files through an `os.Root` confined to the system's folder.

**Reason:** This mirrors the registry's own split into system pages (decisions/023). A system page reads only its own system. Serving the folder's images is the only way to show what Batocera displays today. Confining that route to images under the system folder keeps it from turning into a way to read arbitrary files, ROMs included.

**Rejected alternatives:** Showing only the registry's image, which does not show what the folder really holds. Keeping the problem games only, which the user declined because they want every game listed. A path-style URL `/roms-folder/{system}`, which conflicts with the existing `/roms-folder/retrieve` route.
