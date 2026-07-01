# Module: config
**Role:** Persists user configuration — the registry path and the Batocera ROMs folders to watch.
**Files:** `internal/config/config.go`
**Exports:**
- `config.Config{RegistryPath, RomsFolders}`
- `config.DefaultPath() (string, error)` — resolves the config file path (`BATOCERA_SCRAP_MANAGER_CONFIG` environment variable first, otherwise `os.UserConfigDir()/batocera-scrap-manager/config.json`)
- `config.Load(path string) (Config, error)` — missing file = empty `Config{}`, no error
- `config.Save(path string, cfg Config) error` — creates parent directories if needed
- `(*Config).SetRegistryPath(path string) error` — converts to an absolute path
- `(*Config).AddRomsFolder(path string) (added bool, err error)` — deduplicates by absolute path

**Depends on:** (no internal module)

Stored as JSON (`registry_path`, `roms_folders`).
