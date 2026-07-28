# Module: gamelist
**Role:** Parses and writes `gamelist.xml` files in the EmulationStation/Batocera format.
**Files:** `internal/gamelist/gamelist.go`
**Exports:**
- `gamelist.Game` — a game (path, name, desc, image, video, marquee, thumbnail, rating, release_date, developer, publisher, genre, players), `xml` and `json` tags; all `xml` tags except `path` use `omitempty`, so writing omits unset fields instead of emitting empty tags
- `gamelist.Parse(r io.Reader) ([]Game, error)`
- `gamelist.ParseFile(path string) ([]Game, error)`
- `gamelist.Write(w io.Writer, games []Game) error` — encodes games as a `gameList` XML document (with XML header), indented
- `gamelist.WriteFile(path string, games []Game) error` — writes the document to a temporary file **in the target's own folder** (`os.CreateTemp`, named `.<basename>.*`), syncs it, gives it the target's existing permissions (or `0644` when there was no target), and renames it over the target. Any failure removes the temporary file and leaves the previous file byte-identical — see [`decisions/026`](../decisions/026-a-gamelist-is-written-beside-then-swapped-in.md)

**Depends on:** (no internal module)

An empty `gameList`, or one with missing optional fields, does not produce an error (zero values).

## Writing is all-or-nothing (backlog item 023)
`WriteFile` used to be `os.Create` + `Write`, which truncates the target before a single byte of the new document exists. That file is the user's only copy, is under no version control, and holds fields `Game` does not model at all (`favorite`, `lastplayed`, `playcount`) that the rewrite reproduces only from what was parsed — so an interruption cost the user data no rollback could bring back. Writing beside and renaming makes the swap atomic; the temporary file must live in the same folder, since a rename is only atomic within one filesystem and a system folder may be its own mount.

Two consequences are deliberate. What refuses a write is now the **folder**, not the file: a `gamelist.xml` left read-only is replaced all the same, since the rename only needs the folder — the registry's own `ManualFields` is what shields values, and it acts before the file is written. And a new file gets `0644` rather than whatever the process umask would have produced through `os.Create`, matching how `internal/config` writes its own file; an existing file keeps its permissions exactly.
