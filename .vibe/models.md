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

## folderReport
| Field | Type | Notes |
|---|---|---|
| Folder | string | the configured ROMs folder this line accounts for |
| Counts | [3]int | the three counts of whichever job produced it — `processed/completed/failed` for a completion, `added/updated/unchanged` for an import |
| Problem | string | empty when the folder was gone through to the end; otherwise the sentence naming what stopped it — the counts reached before the failure are kept |

The counts are positional because the two jobs count different things under the same shape; each job's `jobDescription.Format` (`registry.CompletionSummaryFormat` or `registry.ImportSummaryFormat`) is what words them, so no line of a page can drift from what the CLI prints.
Defined in: `internal/webui/job.go`

## jobReport
| Field | Type | Notes |
|---|---|---|
| Running | bool | a run is in flight; the only state emitting the page's auto-reload |
| StartedAt | string | `15:04:05` of the run's start |
| Elapsed | string | time since the start while running, total duration once over |
| Current | string | the folder being processed, plus `— system: game` once the domain reports one |
| Folders | []folderLine | one line per folder (`Folder`, `Summary`, `Problem`), its counts already worded |
| Totals | [3]int | the folders' counts added up, for a page that has to word its own verdict |
| Summary | string | those same totals, already worded |
| Totalled | bool | whether the totals say anything the per-folder lines do not — they do not with a single folder |
| Failed | bool | the run itself failed, or any folder carries a `Problem` |
| Problem | string | a failure of the run as a whole, beyond any one folder (a registry that could not be written) |
| Caveat | string | a reservation on a run that did apply (a consultation site left stale) — never counted as a failure |

A copy taken under `jobs`' own mutex, so a template never reads a field the run is writing.
Defined in: `internal/webui/job.go`

## jobView
| Field | Type | Notes |
|---|---|---|
| Report | *jobReport | this job kind's own last (or current) run, nil until one happened |
| Other | jobKind | the opposite kind when it is holding the shared slot, empty otherwise |

Both come from a single lock, or a run ending between two reads would leave the page contradicting itself. One slot is shared by the import and the completion — see [`decisions/027`](decisions/027-one-background-job-at-a-time-whichever-direction-it-goes.md).
Defined in: `internal/webui/job.go`

## sendMode
| Field | Type | Notes |
|---|---|---|
| Value | string | what the form submits and what the query carries: `fill` or `replace` |
| Label | string | how the game page and the confirmation both name the rule |
| Note | string | the one sentence saying what the rule does to a value the folder already holds — the only difference that matters between the two, never left to be inferred from the verb |

`sendModes` is the table of the two, in the order the game page offers them: filling first, since it cannot lose anything. It is what maps a submitted value to `registry.CompleteGame` or `registry.ReplaceGame`, and what a request's rule is validated against.
Defined in: `internal/webui/send.go`

## mediumKind (the description of one medium)
| Field | Type | Notes |
|---|---|---|
| name | Medium | `image`, `video`, `marquee` or `thumbnail` — what URLs and submissions carry |
| field | func(*gamelist.Game) *string | the reference this medium is stored in on a game |
| folder | string | `images` or `videos` — the subfolder of the system folder it is stored under |
| suffix | string | `-image`, `-video`, `-marquee`, `-thumb`, following what Batocera's own scrapers write beside a ROM |
| extensions | []string | the dotted lowercase extensions it accepts, and the list a caller words its refusal from |

`mediaKinds` is the single table of the four, in the order a page offering them renders. It is the enriched successor of the old accessor-only `mediaFields`, and the same table `RemoveByID`, `copyGameMedia`, `copyFilledMedia` and `copyEveryMedium` walk — so a medium added later cannot be honoured by one and forgotten by another. The stored file's name is `<folder>/<gameID><suffix><ext>`, composed here and never taken from a submitted filename (see [`decisions/031`](decisions/031-an-uploaded-medium-is-named-by-the-registry-never-by-the-client.md)).
Defined in: `internal/registry/media.go`

## mediumControl
| Field | Type | Notes |
|---|---|---|
| Label | string | how the medium is named to a user (`Cover art`, not `image`) |
| Medium | string | the registry's own name, for the form action and the element ids |
| URL | string | the served URL, **empty when no file is really in the registry folder** |
| Reference | string | what the entry stores; non-empty with an empty `URL` means a reference pointing at a file that is not there, which the page names rather than showing broken |
| Video | bool | whether the preview is a `<video>` rather than an `<img>` |
| UploadURL | string | where the file control submits |
| DeleteURL | string | empty when there is nothing to remove — gated on `Reference`, not on `URL`, so a dangling reference can still be cleared |
| Accept | string | the medium's extensions joined for the `accept` attribute and shown under the control |

All four are always built, whether or not the game holds them: a section showing only what exists could not offer to add what does not.
Defined in: `internal/webui/media.go`

## mediumDeleteConfirmation
| Field | Type | Notes |
|---|---|---|
| Name, System, SystemURL, GameURL, Action | string | the game whose medium is being erased and where the two ways out lead |
| Label | string | the medium, named as the user reads it |
| File | string | the reference about to be erased; empty when the entry holds none, which the page words instead of promising to erase a file it cannot name |
| Problem | string | empty on the first attempt; otherwise why the file could not be erased, the page being shown again rather than a dead-end error |
Defined in: `internal/webui/media.go`

## sendConfirmation
| Field | Type | Notes |
|---|---|---|
| Name, System, SystemURL, GameURL, Action | string | the game being sent and where the page leads |
| Folder | string | the one chosen ROMs folder, already checked against the configured list |
| Mode | sendMode | the chosen rule, carried whole so the page words it from the same table the control did |
| Replaces | bool | `Mode.Value == "replace"`; what the template branches on for the warning wording and the danger styling |
| Problem | string | empty on the first attempt; otherwise why the folder could not be written to, the page being shown again rather than a dead-end error |
Defined in: `internal/webui/send.go`

## FolderStatus / SystemStatus / GameGap
| Field | Type | Notes |
|---|---|---|
| FolderStatus.RomsFolder | string | the folder the report is about |
| FolderStatus.Systems | []SystemStatus | ordered worst first (most `IncompleteCount + NotListedCount`), tie-broken by name; a system with nothing to report is left out entirely |
| SystemStatus.ROMCount | int | files sitting directly in the system's folder, `gamelist.xml`, an `_info.txt` note and any subfolder excluded — no extension check (see [`decisions/038`](decisions/038-roms-folder-status-detects-orphan-roms-by-flat-file-heuristic.md)) — and a hidden game's file dropped too (see `Hidden`, below) |
| SystemStatus.CompleteCount / IncompleteCount / NotListedCount | int | local entries with every field present / missing at least one / ROM files with no local entry at all — a hidden game counts as none of the three |
| SystemStatus.FillableCount | int | how many of the incomplete-or-not-listed games the registry already holds enough to fill, in whole or in part |
| SystemStatus.Problem | string | non-empty when this system's own folder or `gamelist.xml` could not be read — every other count then stays zero, and the system is still listed rather than dropped |
| SystemStatus.Gaps | []GameGap | sorted by best-known name |
| GameGap.ROMFilename / Name | string | the file's base name; the best name known, local entry's first then the registry's |
| GameGap.Listed | bool | false means the ROM has no local `gamelist.xml` entry at all |
| GameGap.Missing | []string | `registry.Field*`/`registry.Medium` identifiers still empty locally |
| GameGap.Fillable | []string | the subset of `Missing` the registry already holds a value for |
| GameGap.RegistryID | string | the matching registry entry's own identifier, when one exists — what a caller links to that entry's page with; empty when the registry does not know this game |

`gapFields` computes `Missing`/`Fillable` by walking the same enriched `gameFields` table `mergeGame`/`overwriteGame` already walk — the same test a real Completion applies, without writing anything. A game marked `<hidden>` in the local `gamelist.xml` (`gamelist.ParseWithVisibility`) never reaches `addGap` at all — see [`decisions/039`](decisions/039-hidden-is-modelled-on-documentgame-not-on-game.md).
Defined in: `internal/registry/status.go`

## romsFolderStatusView / systemStatusView / gapView
| Field | Type | Notes |
|---|---|---|
| romsFolderStatusView.Folder / Systems | string / []systemStatusView | the rendering counterpart of `registry.FolderStatus` |
| systemStatusView.Anchor | string | `"system-" + name`, what the page's table of contents jumps to on the one long page the whole report renders on |
| gapView.Missing / Fillable | string | comma-joined labels, resolved from `registry.GameGap`'s identifiers through the existing `mediaLabels`/`editableFields` tables rather than a table of its own |
| gapView.RegistryURL | string | the gap's own page in the registry (`gameURL`), built from `GameGap.RegistryID`; empty — no link rendered — when the registry does not know the game |
Defined in: `internal/webui/status.go` (built by `romsFolderStatusViewOf` from `registry.FolderStatus`)

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
