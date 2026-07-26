# Data models

## Config
| Field | Type | Notes |
|---|---|---|
| RegistryFolder | string | absolute path to the registry folder, `json:"registry_folder"` |
| RomsFolders | []string | absolute paths, deduplicated, `json:"roms_folders"` |
Defined in: `internal/config/config.go`

## Game
| Field | Type | Notes |
|---|---|---|
| Path | string | ROM path relative to the system folder, as found in `gamelist.xml`; the registry matches entries by its filename alone (`filepath.Base`), ignoring any subfolder prefix |
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
| Entries | []Entry | centralized index, reconstructed by scanning `<registryFolder>/<system>/*.json` (one file per game, no single index file); media files referenced by each `Game` are copied under `<registryFolder>/<system>/...`, mirroring the Batocera ROMs layout |
Defined in: `internal/registry/registry.go`

## ProgressEvent
| Field | Type | Notes |
|---|---|---|
| System | string | Batocera system name of the game being processed |
| GameIndex | int | 1-based index of this game within System's game list |
| GameCount | int | total number of games found for System |
| GameName | string | |
Defined in: `internal/registry/registry.go` (passed to the optional `onProgress` callback of `ImportFromRomsFolder`)

## CompletionEvent
| Field | Type | Notes |
|---|---|---|
| System | string | Batocera system name of the game being examined |
| GameIndex | int | 1-based index of this game within System's local game list |
| GameCount | int | total number of local games found for System |
| GameName | string | |
Defined in: `internal/registry/registry.go` (passed to the optional `onProgress` callback of `CompleteRomsFolder`)

## SystemView / GameView
| Field | Type | Notes |
|---|---|---|
| SystemView.Name | string | Batocera system name |
| SystemView.Games | []GameView | that system's games, sorted by name |
| GameView.Game | gamelist.Game | embedded — the raw metadata |
| GameView.ID | string | `registry.GameID(Game.Path)`; the key used in the web UI's per-game URLs |
| GameView.System | string | |
| GameView.ImagePath / VideoPath / MarqueePath / ThumbnailPath | string | percent-encoded `<system>/<relPath>`, relative to the registry folder; **empty when the referenced file is not on disk** — that emptiness is how every renderer detects a missing medium |
| GameView.Stars | string | rating as `★★★★☆`, empty if missing or invalid |
| GameView.RatingLabel | string | the same rating in words (`4/5`), so it is not conveyed by glyphs alone; empty if missing or invalid |
| GameView.Year | string | 4-digit year extracted from `ReleaseDate`, empty if missing or invalid |
Defined in: `internal/site/view.go` (produced by `site.GroupBySystem`, consumed by the static site and the web UI)
