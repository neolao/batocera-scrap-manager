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
**Definition:** The action of populating the registry from the `gamelist.xml` files already present in the ROMs folders, without duplicating already-known entries (deduplication key: system + ROM path), while also detecting metadata that changed since the last import.
**Code:** `(*Registry).Import`, `registry.ImportFromRomsFolder` in `internal/registry/registry.go`
**Do not confuse with:** the `update` command (`internal/cli/update.go`), which is the CLI command exposing this import mechanism to the user.
