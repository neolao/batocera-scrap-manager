# Module: registry
**Role:** Centralized index of already-known games (scraped metadata and media), populated and kept up to date by importing the ROMs folders' `gamelist.xml` files without duplicating known entries, while detecting metadata that changed and copying the referenced media.
**Files:** `internal/registry/registry.go`
**Exports:**
- `registry.Entry{System, Game}`
- `registry.Registry{Entries}`
- `registry.Load(path string) (*Registry, error)` — path is the registry folder; reconstructs the registry by scanning its per-system subfolders for game JSON files; missing folder = empty registry, no error
- `registry.Save(path string, reg *Registry) error` — writes one JSON file per game (named after the ROM's base name) inside its system's subfolder, creating folders as needed
- `(*Registry).Import(system string, games []gamelist.Game) (added, updated, unchanged int)` — deduplicated by (system, ROM path) key; if the entry already exists, compares the full metadata and replaces + counts "updated" on a difference, otherwise "unchanged"
- `registry.ImportFromRomsFolder(reg *Registry, romsFolder, registryFolder string, onProgress func(ProgressEvent)) (added, updated, unchanged int, err error)` — scans the subfolders (one per Batocera system) of a ROMs folder, reads `gamelist.xml` if present (otherwise silently skips the system), imports each game, and for every added or updated game copies its referenced media (cover art, video, marquee, thumbnail) into `registryFolder`, mirroring the Batocera per-system arborescence; unchanged games are not recopied. If non-nil, `onProgress` is called once per game (added, updated, or unchanged) with its system, 1-based index and total count within that system, and name
- `registry.ProgressEvent{System, GameIndex, GameCount, GameName}` — one progress notification emitted by `ImportFromRomsFolder`

**Depends on:** [`modules/gamelist.md`](gamelist.md)

**Registry folder layout:** `<registryFolder>/<system>/<romBaseName>.json` holds one game's metadata, alongside `<registryFolder>/<system>/...` media subfolders (e.g. `images/`, `videos/`) mirroring the corresponding ROMs system folder — since media paths in `gamelist.xml` are relative to the system folder and are preserved unchanged when copied. There is no single index file for the whole registry: each game's entry is self-contained in its own file, so a corrupted or malformed entry only affects that one game on load.

**Architecture note:** `ImportFromRomsFolder` is exposed via the `update` CLI command (see [`modules/cli.md`](cli.md)), implemented for backlog item 002. Media copying was added afterward, reversing the initial decision to keep the registry metadata-only, once the need to keep the registry self-contained (independent of the original ROMs folders) was confirmed. The registry was then further split from a single index file into one file per game, for the same self-containment reason.
