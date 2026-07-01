# Module: registry
**Role:** Index centralisé des jeux déjà connus (métadonnées scrapées), peuplé en important les `gamelist.xml` existants des dossiers de ROMs sans dupliquer les entrées déjà importées.
**Files:** `internal/registry/registry.go`
**Exports:**
- `registry.Entry{System, Game}`
- `registry.Registry{Entries}`
- `registry.Load(path string) (*Registry, error)` — fichier absent = registry vide, pas d'erreur
- `registry.Save(path string, reg *Registry) error`
- `(*Registry).Import(system string, games []gamelist.Game) (added, unchanged int)` — dédoublonnage par clé (système, chemin du ROM)
- `registry.ImportFromRomsFolder(reg *Registry, romsFolder string) (added, unchanged int, err error)` — scanne les sous-dossiers (un par système Batocera) d'un dossier de ROMs, lit `gamelist.xml` s'il existe (sinon ignore silencieusement le système), et importe

**Depends on:** [`modules/gamelist.md`](gamelist.md)

**Note d'architecture :** `ImportFromRomsFolder` n'est pour l'instant pas exposée via une commande CLI (décision consignée dans `.vibe/decisions.md`, 2026-07-01) — elle sera branchée par l'item de backlog 002 (commande `update`).
