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
**Definition:** The action of populating the registry from the `gamelist.xml` files already present in the ROMs folders, without duplicating already-known entries (deduplication key: system + ROM filename, ignoring any subfolder prefix — see [`decisions/005`](decisions/005-match-registry-entries-by-rom-filename-not-full-path.md)), while also detecting metadata that changed since the last import. A game with neither a description nor a jaquette locally — meaning it has not actually been scraped yet — is not imported (see [`decisions/007`](decisions/007-skip-empty-games-only-on-import-not-retroactively.md)). A Hand-edited field is the one thing an import never overwrites.
**Code:** `(*Registry).Import`, `registry.ImportFromRomsFolder` in `internal/registry/registry.go`
**Do not confuse with:** the `update` command (`internal/cli/update.go`), which is the CLI command exposing this import mechanism to the user.

## Completion
**Definition:** The reverse of Import: filling gaps left in a ROMs folder's own `gamelist.xml` (missing name, description, media, rating, genre, etc.) using the matching entry already known in the registry, without ever overwriting metadata already present locally. The registry is read-only in this flow; the ROMs folder is what gets written to. It is the one operation of the tool that writes outside the registry, into files under no version control, which is why both of its entry points confirm before writing and why a game sheet is swapped in atomically rather than rewritten in place. A folder that cannot be read stops it, leaving the folders already completed counted and the failing one named.
**Code:** `registry.CompleteRomsFolder` in `internal/registry/registry.go`; wording shared through `registry.CompletionSummaryFormat`
**Do not confuse with:** Import, which flows in the opposite direction (ROMs folder → registry); or the `scrape` command (`internal/cli/scrape.go`) and the web UI's `/complete` page (`internal/webui/complete.go`), which are the two entry points exposing this one mechanism to the user — the second running it in the background, since it can take minutes.

## Game ID
**Definition:** A game's identifier inside the registry: its ROM file's base name, without directory prefix or extension (e.g. `Sonic.zip` in a subfolder → `Sonic`). One and the same key names the game's metadata file on disk, deduplicates registry entries, and addresses a game in the web UI's URLs — deliberately never re-derived by a second rule. It is derived, never stored: correcting a game's ROM Path from the web UI changes it, which re-files the game and changes the address of its own page.
**Code:** `registry.GameID`, `(*Registry).FindByID` in `internal/registry/registry.go`
**Do not confuse with:** a ROM filename (which still carries its extension, and possibly a subfolder prefix) — several ROM filenames can share one game ID, and the registry then treats them as the same game (see [`decisions/014`](decisions/014-dedupe-by-extension-stripped-filename-too.md)).

## ROM path
**Definition:** Where a game's ROM file sits relative to its Batocera system folder — the filename plus any subfolder — as `gamelist.xml` writes it and as the registry stores it, byte for byte. It is what a game *is* to Batocera, not one of the values describing it: the Game ID derives from it, so it decides the game's file on disk, its dedup key and its web UI address. It is therefore the one thing a correction can change that moves the entry, and the one field no Hand-edited field mark and no Protected game ever covers — a later Import from a ROMs folder still holding the old path will undo the correction or add the game a second time.
**Code:** `registry.ChangePath`, `registry.ValidatePath`, `registry.RemoveGameFile` in `internal/registry/rompath.go`; `pathError`, `pathConfirmation` in `internal/webui/rompath.go`
**Do not confuse with:** the Game ID it derives (which drops the subfolder and the extension, so several ROM paths yield one Game ID), or a media reference (also relative to the system folder, but stored in its own right and never derived from this one — a path correction never renames a medium).

## Consultation site
**Definition:** The static HTML site (re)generated from the registry's content, letting a user browse games grouped by system (name, description, jaquette) in a web browser instead of opening individual metadata files.
**Code:** `site.Generate` in `internal/site/site.go`
**Do not confuse with:** the registry itself, which is the underlying data source — the consultation site is a read-only rendering of it, regenerated on every `update`.

## Web UI
**Definition:** The registry served live over HTTP by the `serve` command: a page listing every game grouped by system, and one page per game — addressed by its Game ID — showing its full metadata and every medium available for it, the form correcting that metadata, the control declaring the game right as a whole (see Protected game), and the confirmed Removal of the game. Managing a game's media is still to be built on the same per-game URLs.
**Code:** `webui.Handler` in `internal/webui/webui.go`, `runServe` in `internal/cli/serve.go`
**Do not confuse with:** the Consultation site, which is a static file regenerated on every update and browsable without any server running — the web UI renders the registry on demand and gives each game its own address (see [`decisions/015`](decisions/015-real-per-game-pages-in-the-served-web-ui.md)).

## Hand-edited field
**Definition:** A metadata field of a registry entry that an Import must not overwrite: the registry records which fields these are, per game, and puts their stored value back instead of letting the ROMs folder's own — still badly scraped — value win, so the user's judgement is not undone by the next `update`. Only the eight editable text fields can be one (never the ROM path nor a media reference). A field becomes one either when a correction actually changes its value, or when the whole game is declared right (see Protected game) — the mark states "this value is mine", not necessarily "I typed this value" — and it stops being one when the user hands it back to the scraper.
**Code:** `registry.Entry.ManualFields`, `editableFields`, `keepHandEditedFields`, `UpdateMetadata` in `internal/registry/metadata.go` and `internal/registry/registry.go`
**Do not confuse with:** Completion, which also protects existing values — but locally, in the ROMs folder, and by never overwriting any non-empty field rather than by remembering who set it.

## Protected game
**Definition:** A game whose eight editable fields are *all* Hand-edited fields, declared right in one go rather than field by field — so no `update` refreshes any of its metadata. It is the per-field marking at its limit, not a second mechanism: the state is derived from the same table of editable fields rather than stored as a flag, so no part of the tool can hold a different idea of what "protected" means. Protecting writes no value, lifting the protection clears every mark on the game (including those left by earlier corrections), and neither ever reaches the game's media or its ROM path.
**Code:** `registry.Protect`, `registry.Unprotect`, `(Entry).FullyProtected` in `internal/registry/metadata.go`; `runProtect`/`runUnprotect` in `internal/cli/protect.go`; `protectionOf`, `setProtection` in `internal/webui/protect.go`
**Do not confuse with:** a partly protected game, which has some Hand-edited fields but not all — the Web UI offers to protect it, but never to lift in bulk, since that would discard which fields the user had corrected (see [`decisions/021`](decisions/021-whole-game-protection-is-every-field-marked-at-once.md)).

## Removal
**Definition:** Dropping one game from the registry: its metadata file and every medium the registry holds for it are erased from the registry folder, and its entry leaves the registry. It reaches nothing outside the registry — the ROM and the media in the ROMs folders are untouched, so a later Import brings the game back from there. A Protected game is removable like any other: protection shields metadata from being *refreshed*, not from being dropped. The erasure of the metadata file is what makes a removal real — writing the registry only ever writes files, never deletes one that left it (see [`decisions/022`](decisions/022-a-deletion-is-committed-when-the-game-file-is-gone.md)).
**Code:** `registry.Remove`, `registry.RemoveByID` in `internal/registry/registry.go`; `runRemove` in `internal/cli/remove.go`; `deleteGame` in `internal/webui/delete.go`
**Do not confuse with:** Completion or Import, which never drop an entry; or deleting a ROM, which this never does.
