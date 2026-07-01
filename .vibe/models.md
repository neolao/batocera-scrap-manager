# Data models

## Config
| Field | Type | Notes |
|---|---|---|
| RegistryPath | string | absolute path, `json:"registry_path"` |
| RomsFolders | []string | absolute paths, deduplicated, `json:"roms_folders"` |
Defined in: `internal/config/config.go`

## Game
| Field | Type | Notes |
|---|---|---|
| Path | string | ROM path relative to the system folder, deduplication key |
| Name | string | |
| Desc | string | |
| Image | string | |
| Video | string | |
| Marquee | string | |
| Thumbnail | string | |
| Rating | string | |
| ReleaseDate | string | `json:"release_date"` |
| Developer | string | |
| Publisher | string | |
| Genre | string | |
| Players | string | |
Defined in: `internal/gamelist/gamelist.go` (parsed from `gamelist.xml`, EmulationStation/Batocera format)

## Entry
| Field | Type | Notes |
|---|---|---|
| System | string | Batocera system name (subfolder name, e.g. `megadrive`) |
| Game | Game | |
Defined in: `internal/registry/registry.go`

## Registry
| Field | Type | Notes |
|---|---|---|
| Entries | []Entry | centralized index, persisted as JSON |
Defined in: `internal/registry/registry.go`
