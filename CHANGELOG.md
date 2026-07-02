# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- Users can now complete a ROMs folder's missing metadata (description, jaquette, rating, genre, etc.) and media files with `batocera-scrap-manager scrape`, using the registry as the source of already-known information; only games that actually get new information appear in the live output, and a summary of processed/completed/failed entries is printed at the end.
- `batocera-scrap-manager scrape` can now target a single game by giving its real path on disk, completing only that game instead of every game in every configured ROMs folder.
- `batocera-scrap-manager update` can now target a single game by giving its real path on disk, importing/updating only that game in the registry instead of every game in every configured ROMs folder.
- The live output of `batocera-scrap-manager scrape` now shows which ROMs folder each change belongs to, so it stays unambiguous when several ROMs folders are configured.
- Users can now remove a specific game's entry (metadata and media) from the registry with `batocera-scrap-manager remove <system> <rom-filename>`.
- `batocera-scrap-manager update` now (re)generates a static HTML site (`index.html` at the root of the registry folder) listing every game grouped by system, with its name, description, and jaquette when available, so the registry's content can be browsed in a web browser.
- `batocera-scrap-manager update` no longer adds a game to the registry if it has neither a description nor a jaquette, avoiding registry entries with no useful scraped data.
- The generated consultation site now has a styled, retro-arcade look with a sticky navigation bar linking to each system, games presented as consistent cards, a "back to top" link in every system section, and a layout that stays readable on small screens.
- On the consultation site, each game card now shows a truncated description and opens a detail view with the full description when clicked, instead of stretching the card to fit all the text.
- The consultation site's detail view now also shows the game's rating, release year, developer, publisher, genre, and number of players, and plays its gameplay video when one was scraped.

### Fixed

- The registry now recognizes a game by its ROM filename alone, instead of its full path: two ROMs with the same filename in different subfolders of the same system are correctly treated as the same game, instead of silently colliding on disk.
- Image and video links on the consultation site are now properly encoded, fixing broken artwork for games whose file names contain characters like `[`, `]`, spaces, or parentheses.
- Closing a game's detail view on the consultation site no longer scrolls the page away from where you were.
- The consultation site's navigation bar now scrolls horizontally instead of wrapping onto several lines when many systems are configured, so it no longer takes up excessive vertical space.
- The consultation site no longer links to a jaquette or video file that is missing from the registry folder, showing the usual placeholder instead of a broken image or an empty video player.
- The consultation site no longer feels slow to load on large registries: jaquettes now load only as they scroll into view, and gameplay videos are no longer fetched until their game's detail view is actually opened.

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
