# Module: registry
**Role:** Index centralisé des jeux déjà connus (métadonnées scrapées), peuplé et tenu à jour en important les `gamelist.xml` des dossiers de ROMs sans dupliquer les entrées déjà connues, tout en détectant les métadonnées qui ont changé.
**Files:** `internal/registry/registry.go`
**Exports:**
- `registry.Entry{System, Game}`
- `registry.Registry{Entries}`
- `registry.Load(path string) (*Registry, error)` — fichier absent = registry vide, pas d'erreur
- `registry.Save(path string, reg *Registry) error`
- `(*Registry).Import(system string, games []gamelist.Game) (added, updated, unchanged int)` — dédoublonnage par clé (système, chemin du ROM) ; si l'entrée existe déjà, compare l'intégralité des métadonnées et remplace + compte "updated" en cas de différence, sinon "unchanged"
- `registry.ImportFromRomsFolder(reg *Registry, romsFolder string) (added, updated, unchanged int, err error)` — scanne les sous-dossiers (un par système Batocera) d'un dossier de ROMs, lit `gamelist.xml` s'il existe (sinon ignore silencieusement le système), et importe

**Depends on:** [`modules/gamelist.md`](gamelist.md)

**Note d'architecture :** `ImportFromRomsFolder` est exposée via la commande CLI `update` (voir [`modules/cli.md`](cli.md)), implémentée pour l'item de backlog 002.
