# Module: cli
**Role:** Point d'entrée et interface en ligne de commande de batocera-scrap-manager — parse les arguments et dispatche vers les commandes.
**Files:** `main.go`, `internal/cli/cli.go`, `internal/cli/config.go`, `internal/cli/update.go`
**Exports:** `cli.Execute(args []string, out io.Writer) int`
**Depends on:** [`modules/config.md`](config.md), [`modules/registry.md`](registry.md)

## Sous-commande `config`
`internal/cli/config.go` implémente `runConfig(args []string, out io.Writer) int`, dispatché par `Execute` sur `args[0] == "config"`.

- `config set-registry <path>` — définit le chemin du registry (converti en chemin absolu via `internal/config`), persiste dans le fichier de config.
- `config add-roms-folder <path>` — ajoute un dossier de ROMs Batocera à surveiller (dédoublonné par chemin absolu).
- `config list` — affiche le registry configuré (ou "(not set)") et la liste des dossiers de ROMs (ou "(none)").
- Toute sous-commande absente ou inconnue retourne le code de sortie 1.

Le chemin du fichier de config est résolu via `config.DefaultPath()` : variable d'environnement `BATOCERA_SCRAP_MANAGER_CONFIG` si définie, sinon `os.UserConfigDir()/batocera-scrap-manager/config.json`.

## Sous-commande `update`
`internal/cli/update.go` implémente `runUpdate(out io.Writer) int`, dispatché par `Execute` sur `args[0] == "update"`.

- Charge la config, refuse avec le code de sortie 1 si `RegistryPath` n'est pas défini (message d'erreur explicite).
- Charge le registry, puis appelle `registry.ImportFromRomsFolder` pour chaque dossier de ROMs configuré ; s'arrête et retourne le code 1 dès qu'un dossier est introuvable.
- Sauvegarde le registry mis à jour, puis affiche un résumé `"%d added, %d updated, %d unchanged"`.
- Aucun dossier de ROMs configuré n'est un cas valide (pas une erreur) : affiche un résumé à zéro.
