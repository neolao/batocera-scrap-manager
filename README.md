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
- Remove a specific game's entry (metadata and media) from the registry, with the consultation site rebuilt straight away so it stops listing the game. Should the site fail to be rebuilt, the removal is still confirmed and the site is reported as out of date rather than left to look like a failure.
- Browse the registry's content in a web browser: updating the registry generates a styled static HTML site listing every game grouped by system, with its name, a short description, and jaquette, a navigation bar (scrollable when many systems are configured) to jump between systems, and a layout that stays readable on small screens.
- Each game on the consultation site opens a detail view showing its full description, rating, release year, developer, publisher, genre, number of players, and gameplay video when available.
- Serve the registry over the local network and browse it from any device: a page lists every game grouped by system, and each game gets its own page with all its metadata and every medium scraped for it (jaquette, video, marquee, thumbnail). The address and port can be chosen, and an unknown address shows a clear "not found" page with a way back to the list.
- Correct a badly scraped game straight from the browser: each game's page offers a pre-filled form for its name, description, rating, release year, developer, publisher, genre and number of players. Saving updates the registry and regenerates the consultation site immediately, with no restart and no file to edit by hand — and a refused entry comes back with what you typed still in place.
- A value you corrected by hand is never overwritten by a later update, even though the ROMs folder still holds the badly scraped one. Each game's page shows which of its values were corrected that way, and any of them can be handed back to the scraper.
- Declare a whole game good in one go, so later updates leave all of its metadata alone — from the command line, or from the game's own page in the browser, which states whether updates may still refresh it. Protecting changes none of its values, and the protection can be lifted just as easily.
- Delete a game from the registry straight from the browser, without knowing its exact ROM filename: a confirmation page names the game and lists, file by file, the metadata and media about to be erased. Once confirmed, the consultation site is regenerated without it and the game list comes back with a banner naming what was deleted.
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
batocera-scrap-manager serve --help
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

It names what it removed, and rebuilds the browsable HTML site without the game, so there is no need to run `update` afterwards just to stop seeing it. If the site could not be rebuilt, the removal is still confirmed — the game is gone either way — and a warning tells you the site is out of date and that `update` rebuilds it. Removing a game that is not in the registry changes nothing at all.

Declare a game good, so later `update` runs stop refreshing its metadata:

```sh
batocera-scrap-manager protect megadrive Sonic.zip
```

None of its values change: protecting only states that the ones already stored are the right ones. Hand it back to the scraper with the symmetric command:

```sh
batocera-scrap-manager unprotect megadrive Sonic.zip
```

Note that `unprotect` clears every mark on the game, including the ones left by corrections you made earlier in the web browser. To hand back a single value instead, use the checkbox offered under it in the edit form. Neither command touches the game's media or its ROM file.

Browse the registry live in a web browser, from this machine or from any device on the local network:

```sh
batocera-scrap-manager serve
```

It listens on `0.0.0.0:8080` by default and prints the address to open. Choose another address or port with `--addr`, and press `Ctrl+C` to stop it:

```sh
batocera-scrap-manager serve --addr 127.0.0.1:9000
```

From a game's page, the "Edit metadata" link opens a form pre-filled with its name, description, rating, release year, developer, publisher, genre and number of players. Saving writes the correction into the registry, regenerates the browsable HTML site, and brings you back to the game's page with a confirmation. The rating is picked as the stars it is shown with, and the release date as its year alone; anything you leave untouched keeps exactly the value that was stored. A game's ROM file and its media are never modified by this form, and a correction that cannot be accepted (an empty name, an implausible year) comes back on the form with what you typed and an explanation.

Every value you correct this way is remembered as hand-edited: later `update` runs refresh everything else from the ROMs folders but leave those values alone, and the game's page marks them so you can tell them apart. Each of them can be handed back to the scraper from the form, with the checkbox offered under it. Note that the ROMs folder itself keeps its own value — Batocera will still display the badly scraped one until it is fixed there too.

A game's page also says whether updates may still refresh it — not protected, partly protected when only some values were corrected by hand, or protected — and offers a button to protect it. Once a game is fully protected the button lifts the protection instead, and the per-value marks give way to that single sentence. The button is only offered the other way round on a fully protected game: lifting in bulk from the partly protected state would quietly discard which values you had corrected, so that is left to the edit form, one value at a time.

Whichever way you protect a game, Batocera keeps displaying what the ROMs folder holds until that is fixed there too.

A game's page also offers "Delete from the registry", which leads to a confirmation page rather than deleting on the spot: it names the game and lists every file about to be erased — its metadata and each medium actually stored for it — and warns you when the game is protected, since that does not prevent its deletion. Cancelling brings you back to the game untouched; confirming erases those files, regenerates the browsable HTML site without the game, and returns you to the list with a banner naming what was deleted. Only the registry is concerned: the ROM file and the media in your ROMs folders are left as they are, and a later `update` will simply import the game again from there. Deleting a game that is already gone says so instead of failing silently.

The registry is read when the server starts: after an `update`, restart `serve` to see the changes. Corrections and protection changes made from the browser take effect immediately, without restarting.
<!-- vibe:end:usage -->

<!-- vibe:begin:docs-index -->
## Documentation

- [docs/architecture.md](docs/architecture.md) — How the tool is put together: its main parts, how a ROMs folder's data flows into the registry and back out, how the registry is browsed, corrected and pruned, and how a hand-made correction — or a whole protected game — is kept from being overwritten.
- [docs/configuration.md](docs/configuration.md) — Where the configuration lives, what it holds, and the environment variable that can relocate it.
<!-- vibe:end:docs-index -->
