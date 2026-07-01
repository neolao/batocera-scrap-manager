# Module: config
**Role:** Persiste la configuration de l'utilisateur — chemin du registry et dossiers de ROMs Batocera à surveiller.
**Files:** `internal/config/config.go`
**Exports:**
- `config.Config{RegistryPath, RomsFolders}`
- `config.DefaultPath() (string, error)` — résout le chemin du fichier de config (variable d'env `BATOCERA_SCRAP_MANAGER_CONFIG` en priorité, sinon `os.UserConfigDir()/batocera-scrap-manager/config.json`)
- `config.Load(path string) (Config, error)` — fichier absent = `Config{}` vide, pas d'erreur
- `config.Save(path string, cfg Config) error` — crée les dossiers parents si besoin
- `(*Config).SetRegistryPath(path string) error` — convertit en chemin absolu
- `(*Config).AddRomsFolder(path string) (added bool, err error)` — dédoublonne par chemin absolu

**Depends on:** (aucun module interne)

Stocké en JSON (`registry_path`, `roms_folders`).
