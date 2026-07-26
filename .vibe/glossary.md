# Ubiquitous Language

## Registry
**Definition:** The store centralizing already-collected scraping data (game metadata, media) — the source of truth that CLI commands read and update.
**Code:** `registry.Registry`, `registry.Load`, `registry.Save` in `internal/registry/registry.go`
**Do not confuse with:** a ROMs folder (source of raw data) — the registry is the centralized destination.

## ROMs folder
**Definition:** A watched Batocera folder containing one or more system subfolders, each with its ROMs, its `gamelist.xml`, and its already-scraped media.
**Code:** `config.Config.RomsFolders`, `registry.ImportFromRomsFolder` in `internal/config/config.go`, `internal/registry/registry.go`

## System
**Definition:** A Batocera gaming platform (e.g. `megadrive`, `mastersystem`) — each system corresponds to a subfolder of a ROMs folder.
**Code:** `registry.Entry.System` in `internal/registry/registry.go`

## Gamelist
**Definition:** A `gamelist.xml` file (EmulationStation/Batocera convention) listing a system's games with their already-scraped metadata and media.
**Code:** `gamelist.Game`, `gamelist.Parse` in `internal/gamelist/gamelist.go`

## Import
**Definition:** The action of populating the registry from the `gamelist.xml` files already present in the ROMs folders, without duplicating already-known entries (deduplication key: system + ROM filename, ignoring any subfolder prefix — see [`decisions/005`](decisions/005-match-registry-entries-by-rom-filename-not-full-path.md)), while also detecting metadata that changed since the last import. A game with neither a description nor a jaquette locally — meaning it has not actually been scraped yet — is not imported (see [`decisions/007`](decisions/007-skip-empty-games-only-on-import-not-retroactively.md)).
**Code:** `(*Registry).Import`, `registry.ImportFromRomsFolder` in `internal/registry/registry.go`
**Do not confuse with:** the `update` command (`internal/cli/update.go`), which is the CLI command exposing this import mechanism to the user.

## Completion
**Definition:** The reverse of Import: filling gaps left in a ROMs folder's own `gamelist.xml` (missing name, description, media, rating, genre, etc.) using the matching entry already known in the registry, without ever overwriting metadata already present locally. The registry is read-only in this flow; the ROMs folder is what gets written to.
**Code:** `registry.CompleteRomsFolder` in `internal/registry/registry.go`
**Do not confuse with:** Import, which flows in the opposite direction (ROMs folder → registry); or the `scrape` command (`internal/cli/scrape.go`), which is the CLI command exposing this completion mechanism to the user.

## Game ID
**Definition:** A game's identifier inside the registry: its ROM file's base name, without directory prefix or extension (e.g. `Sonic.zip` in a subfolder → `Sonic`). One and the same key names the game's metadata file on disk, deduplicates registry entries, and addresses a game in the web UI's URLs — deliberately never re-derived by a second rule.
**Code:** `registry.GameID`, `(*Registry).FindByID` in `internal/registry/registry.go`
**Do not confuse with:** a ROM filename (which still carries its extension, and possibly a subfolder prefix) — several ROM filenames can share one game ID, and the registry then treats them as the same game (see [`decisions/014`](decisions/014-dedupe-by-extension-stripped-filename-too.md)).

## Consultation site
**Definition:** The static HTML site (re)generated from the registry's content, letting a user browse games grouped by system (name, description, jaquette) in a web browser instead of opening individual metadata files.
**Code:** `site.Generate` in `internal/site/site.go`
**Do not confuse with:** the registry itself, which is the underlying data source — the consultation site is a read-only rendering of it, regenerated on every `update`.

## Web UI
**Definition:** The registry served live over HTTP by the `serve` command: a page listing every game grouped by system, and one page per game — addressed by its Game ID — showing its full metadata and every medium available for it. Read-only at this stage; it is the surface on which editing a game, deleting it, and managing its media are to be built.
**Code:** `webui.Handler` in `internal/webui/webui.go`, `runServe` in `internal/cli/serve.go`
**Do not confuse with:** the Consultation site, which is a static file regenerated on every update and browsable without any server running — the web UI renders the registry on demand and gives each game its own address (see [`decisions/015`](decisions/015-real-per-game-pages-in-the-served-web-ui.md)).
