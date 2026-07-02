# Module: gamelist
**Role:** Parses and writes `gamelist.xml` files in the EmulationStation/Batocera format.
**Files:** `internal/gamelist/gamelist.go`
**Exports:**
- `gamelist.Game` — a game (path, name, desc, image, video, marquee, thumbnail, rating, release_date, developer, publisher, genre, players), `xml` and `json` tags; all `xml` tags except `path` use `omitempty`, so writing omits unset fields instead of emitting empty tags
- `gamelist.Parse(r io.Reader) ([]Game, error)`
- `gamelist.ParseFile(path string) ([]Game, error)`
- `gamelist.Write(w io.Writer, games []Game) error` — encodes games as a `gameList` XML document (with XML header), indented
- `gamelist.WriteFile(path string, games []Game) error` — writes/truncates the file at path via `Write`

**Depends on:** (no internal module)

An empty `gameList`, or one with missing optional fields, does not produce an error (zero values).
