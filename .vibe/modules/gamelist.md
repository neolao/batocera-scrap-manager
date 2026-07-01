# Module: gamelist
**Role:** Parses `gamelist.xml` files in the EmulationStation/Batocera format.
**Files:** `internal/gamelist/gamelist.go`
**Exports:**
- `gamelist.Game` — a game (path, name, desc, image, video, marquee, thumbnail, rating, release_date, developer, publisher, genre, players), `xml` and `json` tags
- `gamelist.Parse(r io.Reader) ([]Game, error)`
- `gamelist.ParseFile(path string) ([]Game, error)`

**Depends on:** (no internal module)

An empty `gameList`, or one with missing optional fields, does not produce an error (zero values).
