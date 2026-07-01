# Data models

## Config
| Field | Type | Notes |
|---|---|---|
| RegistryPath | string | chemin absolu, `json:"registry_path"` |
| RomsFolders | []string | chemins absolus, dédoublonnés, `json:"roms_folders"` |
Defined in: `internal/config/config.go`

## Game
| Field | Type | Notes |
|---|---|---|
| Path | string | chemin du ROM relatif au dossier système, clé de dédoublonnage |
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
Defined in: `internal/gamelist/gamelist.go` (issu du parsing de `gamelist.xml`, format EmulationStation/Batocera)

## Entry
| Field | Type | Notes |
|---|---|---|
| System | string | nom du système Batocera (nom du sous-dossier, ex. `megadrive`) |
| Game | Game | |
Defined in: `internal/registry/registry.go`

## Registry
| Field | Type | Notes |
|---|---|---|
| Entries | []Entry | index centralisé, persisté en JSON |
Defined in: `internal/registry/registry.go`
