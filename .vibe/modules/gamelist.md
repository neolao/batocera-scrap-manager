# Module: gamelist
**Role:** Parse les fichiers `gamelist.xml` au format EmulationStation/Batocera.
**Files:** `internal/gamelist/gamelist.go`
**Exports:**
- `gamelist.Game` — un jeu (path, name, desc, image, video, marquee, thumbnail, rating, release_date, developer, publisher, genre, players), tags `xml` et `json`
- `gamelist.Parse(r io.Reader) ([]Game, error)`
- `gamelist.ParseFile(path string) ([]Game, error)`

**Depends on:** (aucun module interne)

Un `gameList` vide ou avec des champs optionnels absents ne produit pas d'erreur (valeurs zéro).
