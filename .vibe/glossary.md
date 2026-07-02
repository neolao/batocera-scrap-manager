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
**Definition:** The action of populating the registry from the `gamelist.xml` files already present in the ROMs folders, without duplicating already-known entries (deduplication key: system + ROM filename, ignoring any subfolder prefix — see [`decisions/005`](decisions/005-match-registry-entries-by-rom-filename-not-full-path.md)), while also detecting metadata that changed since the last import.
**Code:** `(*Registry).Import`, `registry.ImportFromRomsFolder` in `internal/registry/registry.go`
**Do not confuse with:** the `update` command (`internal/cli/update.go`), which is the CLI command exposing this import mechanism to the user.

## Completion
**Definition:** The reverse of Import: filling gaps left in a ROMs folder's own `gamelist.xml` (missing name, description, media, rating, genre, etc.) using the matching entry already known in the registry, without ever overwriting metadata already present locally. The registry is read-only in this flow; the ROMs folder is what gets written to.
**Code:** `registry.CompleteRomsFolder` in `internal/registry/registry.go`
**Do not confuse with:** Import, which flows in the opposite direction (ROMs folder → registry); or the `scrape` command (`internal/cli/scrape.go`), which is the CLI command exposing this completion mechanism to the user.
