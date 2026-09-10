# Module: gamelist
**Role:** Parses and writes `gamelist.xml` files in the EmulationStation/Batocera format.
**Files:** `internal/gamelist/gamelist.go`
**Exports:**
- `gamelist.Game` — a game (path, name, desc, image, video, marquee, thumbnail, rating, release_date, developer, publisher, genre, players), `xml` and `json` tags; all `xml` tags except `path` use `omitempty`, so writing omits unset fields instead of emitting empty tags
- `gamelist.Parse(r io.Reader) ([]Game, error)`
- `gamelist.ParseFile(path string) ([]Game, error)`
- `gamelist.ParseWithVisibility(path string) (games []Game, hidden []bool, err error)` — like `ParseFile`, plus `hidden[i]` reporting whether `games[i]` is marked `<hidden>true</hidden>` — EmulationStation's own way of excluding a ROM from Batocera's list without touching the file. `Hidden` is not one of the fields `Game` models (see below); this is the one entry point that reads it, used by [`modules/registry.md`](registry.md)'s ROMs folder status report to leave a hidden game out of what it counts entirely
- `gamelist.Write(w io.Writer, games []Game) error` — encodes games as a `gameList` XML document (with XML header), indented. It writes exactly what `Game` models and nothing else — `UpdateFile` is what rewrites an existing document without discarding the rest of it
- `gamelist.UpdateFile(path string, games []Game) error` — rewrites the document at path with games, **preserving** everything it already held that `Game` does not model (see below). It writes to a temporary file **in the target's own folder** (`os.CreateTemp`, named `.<basename>.*`), syncs it, gives it the target's existing permissions (or `0644` when there was no target), and renames it over the target. Any failure removes the temporary file and leaves the previous file byte-identical — see [`decisions/026`](../decisions/026-a-gamelist-is-written-beside-then-swapped-in.md). It replaced the former `WriteFile`, whose name no longer described what it does: it now reads the target before writing it

**Depends on:** (no internal module)

An empty `gameList`, or one with missing optional fields, does not produce an error (zero values).

## Writing is all-or-nothing (backlog item 023)
The file writer used to be `os.Create` + `Write`, which truncates the target before a single byte of the new document exists. That file is the user's only copy and is under no version control, so an interruption cost data no rollback could bring back. Writing beside and renaming makes the swap atomic; the temporary file must live in the same folder, since a rename is only atomic within one filesystem and a system folder may be its own mount.

Two consequences are deliberate. What refuses a write is now the **folder**, not the file: a `gamelist.xml` left read-only is replaced all the same, since the rename only needs the folder — the registry's own `ManualFields` is what shields values, and it acts before the file is written. And a new file gets `0644` rather than whatever the process umask would have produced through `os.Create`, matching how `internal/config` writes its own file; an existing file keeps its permissions exactly.

## Preserving what Game does not model (backlog item 024)
A `<game>` holds more than the thirteen fields `Game` declares: Batocera writes `<favorite>`, `<playcount>` and `<lastplayed>` there as the user plays, and scrapers leave attributes on the element itself (`id`, `source`). Re-encoding the document from `Game` alone erased every one of them — the user asked for descriptions to be filled and lost their play history in exchange.

`UpdateFile` therefore calls `preservedOf(path)` first, which parses the document already in place into `documentGame` — `Game` embedded, plus `Hidden string` (`xml:"hidden,omitempty"`), `Attrs []xml.Attr` (`,any,attr`) and `Unmodelled []unmodelledElement` (`,any`) — and indexes by ROM path every game carrying any of the three. `documentGames` then re-attaches that remainder, `Hidden` included, to the games being written. An `unmodelledElement` keeps the element's name, its attributes and its raw `,innerxml`, so nested markup comes back byte for byte.

`Hidden` is modelled on `documentGame` rather than left to fall into `Unmodelled` (see [`decisions/039`](../decisions/039-hidden-is-modelled-on-documentgame-not-on-game.md)) so `ParseWithVisibility` can read it at all — a generic `,any` capture is opaque to typed code. It stays off the exported `Game` for the exact reason a play count does: `Game` embeds into `registry.Entry` and is stored flat in the registry's JSON, so a field added there is a field every registry file gains. `preservedOf`'s own "does this game carry anything worth indexing" check was updated to include a non-empty `Hidden` alongside non-empty `Attrs`/`Unmodelled` — a game whose only extra content is `<hidden>` would otherwise never be indexed at all, and `documentGames` would silently drop it on the next rewrite.

The type is **unexported on purpose**: `Game` must stay a plain comparable struct — `mergeGameEntry` decides "unchanged" with `r.Entries[i].Game == g`, which a slice field would break outright — and, more to the point, a play count belongs to the ROMs folder, not to an index of scraped metadata. Keeping the remainder inside this package is what makes "the registry never carries it" structural rather than a discipline to remember. `Parse` and `ParseFile` still return plain `[]Game`; only a rewrite has any use for the rest.

Preserved elements are re-emitted **after** the modelled ones (struct field order), not at their original position — EmulationStation looks children up by name, so the order is immaterial to it. A ROM path listed twice keeps the last entry's remainder; a game dropped from the written list takes its own with it. Matching on `Game.Path` is what ties a remainder to its game — the same string that identifies a game everywhere else in a game sheet.
