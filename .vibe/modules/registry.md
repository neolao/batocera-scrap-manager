# Module: registry
**Role:** Centralized index of already-known games (scraped metadata), populated and kept up to date by importing the ROMs folders' `gamelist.xml` files without duplicating known entries, while detecting metadata that changed.
**Files:** `internal/registry/registry.go`
**Exports:**
- `registry.Entry{System, Game}`
- `registry.Registry{Entries}`
- `registry.Load(path string) (*Registry, error)` — missing file = empty registry, no error
- `registry.Save(path string, reg *Registry) error`
- `(*Registry).Import(system string, games []gamelist.Game) (added, updated, unchanged int)` — deduplicated by (system, ROM path) key; if the entry already exists, compares the full metadata and replaces + counts "updated" on a difference, otherwise "unchanged"
- `registry.ImportFromRomsFolder(reg *Registry, romsFolder string) (added, updated, unchanged int, err error)` — scans the subfolders (one per Batocera system) of a ROMs folder, reads `gamelist.xml` if present (otherwise silently skips the system), and imports

**Depends on:** [`modules/gamelist.md`](gamelist.md)

**Architecture note:** `ImportFromRomsFolder` is exposed via the `update` CLI command (see [`modules/cli.md`](cli.md)), implemented for backlog item 002.
