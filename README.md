# batocera-scrap-manager

A command-line tool for managing game scraping data (metadata, box art, etc.) on Batocera. It centralizes into a registry the information already scraped on your ROMs folders, so you can browse it and keep it up to date.

<!-- vibe:begin:features -->
## Features

- Configure the registry location and one or more Batocera ROMs folders to watch.
- Browse the configured registry and the list of watched ROMs folders at any time.
- Update the registry in one step from the configured ROMs folders: new games are added, games whose metadata changed are refreshed, and a summary (added / updated / unchanged) is displayed. Games with no scraped data (no description and no jaquette) are skipped, so the registry only holds games worth keeping. A single game can also be targeted by its path, to import or update just that one.
- See live progress (current system and a per-game counter) while the registry is being updated, instead of waiting silently for the final summary.
- The registry keeps a copy of each game's cover art, video, marquee, and thumbnail alongside its metadata, organized by system just like on Batocera itself.
- Complete a ROMs folder's missing metadata and media (description, jaquette, rating, genre, etc.) using the registry as the source of already-known information, with a summary of processed / completed / failed entries. A single game can also be targeted by its path, to complete just that one.
- Remove a specific game's entry (metadata and media) from the registry.
- Browse the registry's content in a web browser: updating the registry generates a styled static HTML site listing every game grouped by system, with its name, a short description, and jaquette, a navigation bar (scrollable when many systems are configured) to jump between systems, and a layout that stays readable on small screens.
- Each game on the consultation site opens a detail view showing its full description, rating, release year, developer, publisher, genre, number of players, and gameplay video when available.
- Get detailed, command-specific help with `--help` on any command (e.g. `update --help`), instead of just the generic top-level help.
<!-- vibe:end:features -->

<!-- vibe:begin:install -->
## Installation

Prerequisite: Go 1.26 or later.

With `go install`:

```sh
go install github.com/neolao/batocera-scrap-manager@latest
```

Or by building from source:

```sh
git clone https://github.com/neolao/batocera-scrap-manager.git
cd batocera-scrap-manager
go build -o batocera-scrap-manager .
```
<!-- vibe:end:install -->

<!-- vibe:begin:usage -->
## Usage

Show help or version:

```sh
batocera-scrap-manager --help
batocera-scrap-manager --version
```

Each command also has its own detailed help:

```sh
batocera-scrap-manager config --help
batocera-scrap-manager update --help
batocera-scrap-manager scrape --help
batocera-scrap-manager remove --help
```

Configure the registry location and the ROMs folders to watch:

```sh
batocera-scrap-manager config set-registry /userdata/saves/scrap-registry
batocera-scrap-manager config add-roms-folder /userdata/roms
batocera-scrap-manager config list
```

Update the registry from the configured ROMs folders (this also (re)generates a browsable HTML site at the root of the registry folder, at `index.html`):

```sh
batocera-scrap-manager update
```

Update a single game instead, by giving its path in a configured ROMs folder:

```sh
batocera-scrap-manager update /userdata/roms/megadrive/Sonic.zip
```

Complete a ROMs folder's missing metadata and media from the registry:

```sh
batocera-scrap-manager scrape
```

Complete a single game instead, by giving its path in a configured ROMs folder:

```sh
batocera-scrap-manager scrape /userdata/roms/megadrive/Sonic.zip
```

Remove a game's entry from the registry:

```sh
batocera-scrap-manager remove megadrive Sonic.zip
```
<!-- vibe:end:usage -->
