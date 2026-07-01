# Module: registry
**Role:** Centralized index of already-known games (scraped metadata and media), populated and kept up to date by importing the ROMs folders' `gamelist.xml` files without duplicating known entries, while detecting metadata that changed and copying the referenced media.
**Files:** `internal/registry/registry.go`
**Exports:**
- `registry.Entry{System, Game}`
- `registry.Registry{Entries}`
- `registry.Load(path string) (*Registry, error)` — path is the registry folder; missing folder or index file = empty registry, no error
- `registry.Save(path string, reg *Registry) error` — writes the index file at `<path>/registry.json`, creating the folder as needed
- `(*Registry).Import(system string, games []gamelist.Game) (added, updated, unchanged int)` — deduplicated by (system, ROM path) key; if the entry already exists, compares the full metadata and replaces + counts "updated" on a difference, otherwise "unchanged"
- `registry.ImportFromRomsFolder(reg *Registry, romsFolder, registryFolder string) (added, updated, unchanged int, err error)` — scans the subfolders (one per Batocera system) of a ROMs folder, reads `gamelist.xml` if present (otherwise silently skips the system), imports each game, and for every added or updated game copies its referenced media (cover art, video, marquee, thumbnail) into `registryFolder`, mirroring the Batocera per-system arborescence; unchanged games are not recopied

**Depends on:** [`modules/gamelist.md`](gamelist.md)

**Registry folder layout:** `<registryFolder>/registry.json` holds the metadata index; `<registryFolder>/<system>/...` mirrors the media subfolders (e.g. `images/`, `videos/`) found under the corresponding ROMs system folder, since media paths in `gamelist.xml` are relative to the system folder and are preserved unchanged when copied.

**Architecture note:** `ImportFromRomsFolder` is exposed via the `update` CLI command (see [`modules/cli.md`](cli.md)), implemented for backlog item 002. Media copying was added afterward, reversing the initial decision to keep the registry metadata-only, once the need to keep the registry self-contained (independent of the original ROMs folders) was confirmed.
