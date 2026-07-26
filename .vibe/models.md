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
| ManualFields | []string | names of the metadata fields an import must not overwrite (`name`, `desc`, `rating`, `release_date`, `developer`, `publisher`, `genre`, `players`); an import puts these values back instead of overwriting them. Empty for a game nobody corrected, all eight for a game protected as a whole — `Entry.FullyProtected()` derives that state from the same `editableFields` table rather than storing a flag |
Defined in: `internal/registry/registry.go`

## storedGame (on-disk shape of an Entry's game)
| Field | Type | Notes |
|---|---|---|
| *(embedded)* Game | Game | the game's own fields, flat at the root of its JSON file, unchanged from earlier versions |
| ManualFields | []string | `json:"manual_fields,omitempty"` — omitted entirely when nothing was corrected, so a file written before this feature and a file written now for an untouched game are identical |
Defined in: `internal/registry/registry.go` (written by `Save`, read by `Load`)

## Metadata (a correction)
| Field | Type | Notes |
|---|---|---|
| Name, Desc, Rating, ReleaseDate, Developer, Publisher, Genre, Players | string | the eight editable fields at once; an empty one clears the stored value. The ROM path and the media references are deliberately absent, so no correction can reach them |
Defined in: `internal/registry/metadata.go`

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
