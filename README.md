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
- Serve the registry over the local network and browse it from any device: the home page summarizes the systems and how many games each holds, every system has its own page listing its games 60 at a time, and each game gets its own page with all its metadata and every medium scraped for it (jaquette, video, marquee, thumbnail). The address and port can be chosen, and an unknown address — or a page number beyond the last — shows a clear "not found" page with a way back.
- Browse comfortably from a phone: on a small screen a game is listed as a compact row (thumbnail and name) instead of a full-width card, so about a dozen games fit on screen rather than one, and every link and button stays big enough to tap. This applies both to the served pages and to the generated `index.html`.
- A game is picked out of a list by its name and its cover art: the list stays uncluttered, and a game's release year is shown where it belongs — on the game's own page and in the consultation site's detail view.
- Correct a badly scraped game straight from the browser: each game's page offers a pre-filled form for its name, description, rating, release year, developer, publisher, genre and number of players. Saving updates the registry and regenerates the consultation site immediately, with no restart and no file to edit by hand — and a refused entry comes back with what you typed still in place.
- A value you corrected by hand is never overwritten by a later update, even though the ROMs folder still holds the badly scraped one. Each game's page shows which of its values were corrected that way, and any of them can be handed back to the scraper.
- Declare a whole game good in one go, so later updates leave all of its metadata alone — from the command line, or from the game's own page in the browser, which states whether updates may still refresh it. Protecting changes none of its values, and the protection can be lifted just as easily.
- A fully protected game is marked with a "Protected" badge right in its system's game list, so it can be told apart from the rest at a glance instead of having to open its own page.
- Delete a game from the registry straight from the browser, without knowing its exact ROM filename: a confirmation page names the game and lists, file by file, the metadata and media about to be erased. Once confirmed, the consultation site is regenerated without it and its system's list comes back with a banner naming what was deleted.
- Manage a game's media from the browser instead of taking whatever the last scrape copied in. Its page shows all four — cover art, video, marquee, thumbnail — present or not, each with a control to send a file from your computer and a link to remove the one already there. A stored file is named after the game itself, never after the name your computer gave it, so an upload can neither write outside the registry nor overwrite another game's media. Removing one asks for confirmation and names the very file about to be erased; the ROMs folders are left alone, so a later update can bring the medium back from what Batocera holds. A file of the wrong type, one over 64 MB, or an empty one is refused with an explanation and changes nothing.
- Read and correct the ROM file a game stands for — the filename and its subfolder — from the game's own page and from its edit form. It is offered apart from the described values, since it is what identifies the game rather than something describing it. Correcting it re-files the game and takes you to its new address; a filename already used by another game of the same system, or a path that names no file, is refused with an explanation and changes nothing.
- Feed the registry from the ROMs folders from the browser, without dropping to the command line: a page names every configured folder and states that they are only read, then the import runs in the background while the page follows it along and settles into a report — counts per folder, plus the same `added / updated / unchanged` summary the command line prints. The imported games are browsable immediately, with no restart. An import that finds nothing new says the registry was already up to date rather than showing a bare row of zeroes, a folder that cannot be found stops it without writing anything, and a correction made in the browser while it runs is never overwritten.
- Both maintenance actions sit together on the home page, each saying which way it goes, so the one that rewrites your Batocera files is not clicked for the one that only reads them. They are offered on an empty registry too, and only one of the two runs at a time — the other page says which one is going and offers to follow it.
- A page following a long operation reloads itself, says how often, and offers to stop reloading so a report can be read to the end.
- Send the registry's metadata back to Batocera from the browser, without dropping to the command line: a confirmation page names every ROMs folder about to be written to and spells out that this cannot be undone, then the completion runs in the background while the page follows it along and settles into a report — counts per folder, plus the same summary the command line prints. The report stays readable afterwards, only one completion runs at a time, and a folder that cannot be found is named without losing what the earlier folders did.
- Send a single game to a single ROMs folder from its own page, instead of running the whole completion over every configured folder or typing the ROM file's path on the command line. You pick the folder among those configured, and whether to fill that folder's gaps only — as a full completion does — or to replace what it holds with what the registry knows. A confirmation names the game, the folder and the chosen rule before anything is written, the result is reported on the game's page naming the folder, and nothing but that game in that folder is touched.
- A `gamelist.xml` is never left truncated: the new content is written beside the old one and swapped in one go, so a completion cut short by a power cut, a full disk or a stopped server leaves the previous file intact.
- Completing a ROMs folder leaves alone everything the tool does not manage in a `gamelist.xml` — favourites, play counts, last-played dates, and the attributes a scraper writes on a game. Those belong to Batocera and to you; they stay in the ROMs folder and never enter the registry.
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

To run the web browser interface without installing Go, build and run the provided Docker image instead:

```sh
docker build -t batocera-scrap-manager .
```

`config.json` is filled in with the `config` subcommand of the same image, run as a throwaway container (`--rm`, overriding the entrypoint) rather than through the long-running server — the server refuses to start before a registry is configured, so it cannot be up yet at this point:

```sh
docker run --rm \
  -e BATOCERA_SCRAP_MANAGER_CONFIG=/data/config/config.json \
  -v /path/to/config:/data/config \
  --entrypoint /batocera-scrap-manager \
  batocera-scrap-manager config set-registry /data/registry
docker run --rm \
  -e BATOCERA_SCRAP_MANAGER_CONFIG=/data/config/config.json \
  -v /path/to/config:/data/config \
  --entrypoint /batocera-scrap-manager \
  batocera-scrap-manager config add-roms-folder /data/roms/snes
```

`/path/to/config` is a *folder*, not the `config.json` file itself, and can be empty or not exist yet — Docker creates it, and the command above creates the file inside it. Mounting the file directly would backfire the moment it doesn't exist yet: Docker would bind-mount a *directory* in its place instead, and the server would never start. The same reasoning is why `registry_folder` and `roms_folders` inside `config.json` must be paths under the mounted volumes, e.g. `/data/registry` and `/data/roms`, not host paths.

Now start the server itself:

```sh
docker run -d \
  -p 8080:8080 \
  -e BATOCERA_SCRAP_MANAGER_CONFIG=/data/config/config.json \
  -v /path/to/registry:/data/registry \
  -v /path/to/roms:/data/roms \
  -v /path/to/config:/data/config \
  batocera-scrap-manager
```

The container runs as a non-root user (uid 65532): the host folders bind-mounted for the registry, the ROMs folder and the config folder must be readable and writable by that uid, or the server fails to start or to save.

`/path/to/roms` above is expected to be the *parent* of every system's ROMs folder (as Batocera itself lays them out, e.g. `roms/snes`, `roms/megadrive`), mounted once. Which of its subfolders are actually watched is controlled entirely by `roms_folders` in `config.json` — running `config add-roms-folder` as above and restarting the container is enough to watch a new one, with no new bind mount required. Folders that don't already share one parent directory (separate drives, network shares) can be gathered under one by creating a folder of symlinks on the host and mounting that instead.

A change to `config.json` only takes effect on the running server after a restart (`docker restart <container>`), since the configuration is read once at startup.

A `docker-compose.yml` is provided for the same setup. Copy `.env.example` to `.env`, fill in the host paths, then:

```sh
docker compose build
docker compose run --rm --entrypoint /batocera-scrap-manager \
  batocera-scrap-manager config set-registry /data/registry
docker compose run --rm --entrypoint /batocera-scrap-manager \
  batocera-scrap-manager config add-roms-folder /data/roms/snes
docker compose up -d
```

`docker compose run --rm` starts a throwaway container from the same service definition, overriding its entrypoint — no port is published by it, so it doesn't clash with the long-running server; it works the same way whether that server is up or not, which is why it's used for the first-time setup above. It stays the right way to change `config.json` afterwards too, e.g. to add another ROMs folder — just follow it with `docker compose restart` for the change to take effect.

`.env` is git-ignored since its paths are specific to your machine; `.env.example` documents each variable.

### Running on a NAS <!-- keep -->

Most NAS boxes that can run Docker at all (Synology with Container Manager / DSM 7.2+, or the older Docker package; QNAP with Container Station) expose it well enough to run this the same way as above, with a few NAS-specific points:

1. **Enable SSH** (Synology: *Control Panel → Terminal & SNMP*; QNAP: *Control Panel → Network & File Services → Telnet/SSH*) and log in as an admin user — the NAS's own Docker GUI can run a compose "project", but building the image and running `config` commands is far easier from a shell.
2. **Get the source onto the NAS.** Most NAS units don't ship `git`; clone the repository on your computer instead and copy the folder over (`scp -r batocera-scrap-manager admin@nas:/volume1/docker/`, or drag it in through File Station / a mapped SMB share).
3. **Check the CPU architecture** before building: recent Synology/QNAP models are `x86_64` (Intel/AMD), same as a typical dev machine, so `docker build`/`docker compose up --build` run directly on the NAS. Older or entry-level models are `arm64`/`armv7` — either build for that platform elsewhere with `docker buildx build --platform linux/arm64 -t batocera-scrap-manager .` and transfer the image (`docker save` / `docker load`, or a registry you control), or build it once directly over SSH on the NAS itself (slower, but architecture-correct by construction).
4. **Point paths at NAS shared folders**, not container-internal paths, in `.env` — e.g. `REGISTRY_FOLDER=/volume1/scrap-registry`, `ROMS_FOLDER=/volume1/roms` (the parent of every system's ROMs folder — see above), `CONFIG_FOLDER=/volume1/docker/batocera-scrap-manager/config` (a folder, not the `config.json` file — see `.env.example`). If Batocera itself lives on a different machine, share its `roms` folder over the network (NFS/SMB) and mount that share on the NAS first, so `ROMS_FOLDER` can point at a local path into it.
5. From the copied folder, over SSH:
   ```sh
   cp .env.example .env   # then edit .env with the paths above
   docker compose build
   docker compose run --rm --entrypoint /batocera-scrap-manager \
     batocera-scrap-manager config set-registry /data/registry
   docker compose run --rm --entrypoint /batocera-scrap-manager \
     batocera-scrap-manager config add-roms-folder /data/roms/snes
   docker compose up -d
   ```
6. The web UI is then reachable at `http://<nas-address>:8080` (or whatever `PORT` was set to in `.env`) from any machine on the network.

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

The same form also carries the game's ROM file — the filename and the subfolder it sits in, relative to its system's folder — which the game's page now shows under its title. It is offered on its own, above the described values, because it is what identifies the game rather than something describing it: nothing marks it as hand-edited, and protecting a game never covers it. Saving a new one files the game under its new name, removes the entry it was filed under, and takes you to the game's new address, which the confirmation names; correcting only the subfolder or the file extension leaves the game exactly where it was, and says so. A path that is empty, absolute, reaching outside its system's folder with `..`, or naming no file at all is refused with a message saying which — as is one whose filename already belongs to another game of the same system, which the refusal names. A refusal keeps everything else you typed and leaves the registry untouched. The game's media are never renamed or moved along with it: they are stored under paths of their own. Beware that the ROMs folder is not consulted here, so nothing checks that the file you name really exists — and a later `update` reading a `gamelist.xml` that still holds the old path will either undo the correction or add the game a second time under the old name.

Every value you correct this way is remembered as hand-edited: later `update` runs refresh everything else from the ROMs folders but leave those values alone, and the game's page marks them so you can tell them apart. Each of them can be handed back to the scraper from the form, with the checkbox offered under it. Note that the ROMs folder itself keeps its own value — Batocera will still display the badly scraped one until it is fixed there too. Completing the folder does not fix it, since completion only fills what is empty; sending that one game to the folder and asking to replace rather than to fill is what does (see below).

A game's page also carries a "Media" section holding all four of its media — cover art, video, marquee, thumbnail — whether or not the game has them, since a section showing only what exists could not offer to add what does not. Each one shows what the registry currently holds for it, or says "None yet"; a medium the registry refers to but whose file has gone missing says that instead, naming the file, rather than leaving a broken image on the page. Under each is a control to choose a file from your computer and upload it, and, when there is something to remove, a "Remove" link.

Uploading stores the file in the registry under a name built from the game itself and the file's type — `images/Sonic-image.png`, `videos/Sonic-video.mp4` and so on — never from the name your computer gave it. That is what makes an upload unable to write anywhere but where it should, or to land on another game's media, whatever the file is called on your side. Sending a medium the game already has replaces it: with a file of the same type it simply takes its place, and with a file of another type the old one is erased once the new reference is stored, so nothing is left behind unused. The browsable HTML site is regenerated straight away, as it is after a correction.

The accepted types are listed under each control: `.png`, `.jpg`, `.jpeg`, `.gif`, `.webp` and `.bmp` for the three images, `.mp4`, `.mkv`, `.webm`, `.avi`, `.mpg` and `.mpeg` for the video. A file that does not suit the medium is refused with a message listing the ones that do, as is a file larger than 64 MB or an empty one — in every case the registry is left exactly as it was, the medium you had is still there, and the page comes back with its controls so the upload can be retried.

"Remove" leads to a confirmation page naming the medium and the very file about to be erased, and saying it cannot be undone; cancelling changes nothing. Confirming erases that file, empties the game's reference to it, regenerates the browsable HTML site without it, and brings you back to the game's page. Only the registry is concerned: the media in your ROMs folders are untouched, so a later `update` brings the medium back from there. Removing a medium the game no longer has says so rather than failing.

Note that uploading a medium does not put it on Batocera. As with a corrected value, the registry is where it lands; sending that game to a ROMs folder with the "replace" rule is what carries it over — the section says so.

A game's page also says whether updates may still refresh it — not protected, partly protected when only some values were corrected by hand, or protected — and offers a button to protect it. Once a game is fully protected the button lifts the protection instead, and the per-value marks give way to that single sentence. The button is only offered the other way round on a fully protected game: lifting in bulk from the partly protected state would quietly discard which values you had corrected, so that is left to the edit form, one value at a time.

Whichever way you protect a game, Batocera keeps displaying what the ROMs folder holds until that is fixed there too.

A game's page also offers "Delete from the registry", which leads to a confirmation page rather than deleting on the spot: it names the game and lists every file about to be erased — its metadata and each medium actually stored for it — and warns you when the game is protected, since that does not prevent its deletion. Cancelling brings you back to the game untouched; confirming erases those files, regenerates the browsable HTML site without the game, and returns you to the list with a banner naming what was deleted. Only the registry is concerned: the ROM file and the media in your ROMs folders are left as they are, and a later `update` will simply import the game again from there. Deleting a game that is already gone says so instead of failing silently.

A game's page also offers "Send to a ROMs folder", which does for that one game what "Complete the ROMs folders" does for all of them at once — useful right after correcting it, when running the whole thing over every folder would be out of proportion. You pick the folder among those configured, and what to write:

- **Fill the gaps only** — a value the folder already holds is never overwritten, exactly as a full completion behaves.
- **Replace with what the registry knows** — every value and every medium the registry holds is written over the folder's, cover art file included. A value the registry does not know is left as the folder holds it: not knowing something is no reason to make you lose it. Nothing is ever deleted, so if the registry points at a differently named cover art the old file simply stays there, unused.

The button leads to a confirmation page naming the game, the folder and the chosen rule, spelling out what that rule does to what is already there and warning that it cannot be undone; cancelling changes nothing. Confirming writes straight away — a single game takes milliseconds, so there is no background run and no page to follow — and brings you back to the game's page with a banner naming the folder and saying what happened there: it was written to, it was already up to date, or it does not list that game at all. That last case is not a failure and claims no success: sending fills in what a folder already lists, it never adds a game to a folder that does not hold its ROM. The registry itself is only read — none of this rewrites your game files or the browsable HTML site — and no other game, and no other configured folder, is touched. With no ROMs folder configured, the section names the command that configures one instead of offering an empty choice.

The home page groups its two maintenance actions under a "Maintenance" heading, each with a line saying which way it goes: "Import from the ROMs folders" brings what Batocera already scraped into the registry, "Complete the ROMs folders" writes what the registry knows back into Batocera. Both are offered even when the registry is still empty — which is exactly when importing matters most — and with no ROMs folder configured neither is offered at all, the page naming the command that configures one instead.

"Import from the ROMs folders" does from the browser what `update` does from the command line. Its page names every configured folder and states what happens: the folders are only read and left as they are, the registry's game files and its browsable HTML site are rewritten, a game already known is refreshed from what its folder says except for the values you corrected by hand, and a game with neither description nor jaquette is not imported at all. Cancelling changes nothing.

Once confirmed, the import runs in the background and the page follows it along exactly as the completion's does, then settles into a report naming each folder with its own `added / updated / unchanged` counts, plus the total when several folders are configured. The imported games are then browsable straight away, with no restart. An import that had nothing new to bring says the registry was already up to date rather than leaving a row of zeroes to look like a failure, and folders that hold no game at all say that instead.

A configured folder that cannot be found stops the import and **nothing at all is written** — the counts the earlier folders reached are shown as an account of how far it got, not as a claim that they were saved, and running the import again once the folder is back does the whole job. Should the browsable HTML site fail to be regenerated, the games are in the registry all the same and the report says only the site is out of date. And if you correct a game from these pages while an import is running, the import stands down rather than overwriting your correction, telling you to run it again.

"Complete the ROMs folders" does from the browser what `scrape` does from the command line. It leads to a confirmation page first: it names every configured ROMs folder about to be written to, states that each system's `gamelist.xml` is rewritten in place and its media recopied, that the registry folder itself is left untouched, and that none of it can be undone. Cancelling changes nothing.

Once confirmed, the completion runs in the background and the page follows it along: it says when it started, how long it has been going and which folder and game it is on, reloading itself every five seconds until it is over — the page says so, and a "Stop reloading" link keeps it still if you would rather read it at your own pace. Leaving the page, or closing the browser, stops nothing. It then settles into a report naming each folder with its own `processed / completed / failed` counts, plus the total when several folders are configured. That report stays readable until the next completion replaces it, so a browser closed at the wrong moment does not cost you the account of what was written to your Batocera folders.

Only one of the two long operations runs at a time, whichever direction it goes: submitting again while one is going starts nothing and simply shows you the one already in progress, and the other one's page says which operation is holding things up and offers to follow it rather than a button that would do nothing. Browsing and correcting the registry stay responsive throughout. A configured folder that cannot be found — an SD card removed, a network share unmounted — stops the completion there, but the folders already done keep their counts and the report names the one that failed along with what went wrong. With no ROMs folder configured, the page names the command that configures one instead of offering a button.

Each `gamelist.xml` is written beside the old one and swapped in as a whole, so a completion interrupted halfway — a power cut, a full disk, a server stopped — leaves the previous file intact rather than truncated. This is also how `scrape` writes them. One consequence: marking a `gamelist.xml` read-only no longer stops it from being rewritten, since it is the folder's permissions that now decide.

Rewriting a `gamelist.xml` never costs you what the tool does not manage. Before writing, it reads the file already in place and keeps, game by game, everything it does not understand — `<favorite>`, `<playcount>`, `<lastplayed>`, whatever else Batocera or a scraper put there, along with the attributes carried by the game itself — then writes it back as it was. Those values stay in your ROMs folder: the registry neither imports nor stores them. A game removed from the list takes its own along with it, and the preserved elements are written after the ones the tool manages, which changes nothing for Batocera since it looks them up by name.

The registry is read when the server starts: after an `update` run from the command line, restart `serve` to see the changes. Everything done from the browser — corrections, protection changes, deletions, imports and completions — takes effect immediately, without restarting.
<!-- vibe:end:usage -->

<!-- vibe:begin:docs-index -->
## Documentation

- [docs/architecture.md](docs/architecture.md) — How the tool is put together: its main parts, how a ROMs folder's data flows into the registry and back out — a whole folder at a time or one chosen game, filling its gaps or replacing what it holds — how the registry is browsed, corrected and pruned, how a game's media are uploaded and erased, how the two long operations started from the browser run in the background alongside the served pages, and how a hand-made correction — or a whole protected game — is kept from being overwritten.
- [docs/configuration.md](docs/configuration.md) — Where the configuration lives, what it holds, and the environment variable that can relocate it.
<!-- vibe:end:docs-index -->
