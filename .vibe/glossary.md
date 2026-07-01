# Ubiquitous Language

## Registry (Registre)
**Definition:** Dossier centralisant les données de scraping déjà collectées (métadonnées de jeux, médias) — la source de vérité que les commandes CLI lisent et mettent à jour.
**Code:** `registry.Registry`, `registry.Load`, `registry.Save` in `internal/registry/registry.go`
**Do not confuse with:** un dossier de ROMs (source des données brutes) — le registry est la destination centralisée.

## Dossier de ROMs (ROMs folder)
**Definition:** Dossier Batocera surveillé, contenant un ou plusieurs sous-dossiers de systèmes, chacun avec ses ROMs, son `gamelist.xml` et ses médias déjà scrapés.
**Code:** `config.Config.RomsFolders`, `registry.ImportFromRomsFolder` in `internal/config/config.go`, `internal/registry/registry.go`

## Système (System)
**Definition:** Plateforme de jeu Batocera (ex. `megadrive`, `mastersystem`) — chaque système correspond à un sous-dossier d'un dossier de ROMs.
**Code:** `registry.Entry.System` in `internal/registry/registry.go`

## Gamelist
**Definition:** Fichier `gamelist.xml` (convention EmulationStation/Batocera) listant les jeux d'un système avec leurs métadonnées et médias déjà scrapés.
**Code:** `gamelist.Game`, `gamelist.Parse` in `internal/gamelist/gamelist.go`

## Import
**Definition:** Action de peupler le registry à partir des `gamelist.xml` déjà présents dans les dossiers de ROMs, sans dupliquer les entrées déjà connues (clé de dédoublonnage : système + chemin du ROM), en détectant aussi les métadonnées qui ont changé depuis le dernier import.
**Code:** `(*Registry).Import`, `registry.ImportFromRomsFolder` in `internal/registry/registry.go`
**Do not confuse with:** la commande `update` (`internal/cli/update.go`), qui est la commande CLI exposant ce mécanisme d'import à l'utilisateur.
