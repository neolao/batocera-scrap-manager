# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- Users can now complete a ROMs folder's missing metadata (description, jaquette, rating, genre, etc.) and media files with `batocera-scrap-manager scrape`, using the registry as the source of already-known information; only games that actually get new information appear in the live output, and a summary of processed/completed/failed entries is printed at the end.
- Users can now remove a specific game's entry (metadata and media) from the registry with `batocera-scrap-manager remove <system> <rom-filename>`.

### Fixed

- The registry now recognizes a game by its ROM filename alone, instead of its full path: two ROMs with the same filename in different subfolders of the same system are correctly treated as the same game, instead of silently colliding on disk.

## [0.2.0] - 2026-07-02

### Added

- Users can now update the registry from the configured ROMs folders with `batocera-scrap-manager update`, which adds new games, refreshes games whose metadata changed, and prints a summary of added/updated/unchanged entries.
- `batocera-scrap-manager update` now prints live progress (the system being processed and a counter for each game) as it runs, instead of staying silent until the final summary.

### Fixed

- The registry now also stores each game's cover art, video, marquee, and thumbnail, copied into a per-system folder mirroring Batocera's own ROMs layout, instead of keeping only text metadata.
- Each game's information is now stored in its own file inside its system folder, instead of one large file for the whole registry, so a corrupted entry no longer affects the rest of the registry.

## [0.1.0] - 2026-07-01

### Added

- Users can configure the registry folder and one or more Batocera ROMs folders to watch, via `batocera-scrap-manager config set-registry`, `config add-roms-folder`, and `config list`.

[Unreleased]: https://github.com/neolao/batocera-scrap-manager/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/neolao/batocera-scrap-manager/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/neolao/batocera-scrap-manager/releases/tag/v0.1.0
