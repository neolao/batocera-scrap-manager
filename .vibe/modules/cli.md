# Module: cli
**Role:** Point d'entrée et interface en ligne de commande de batocera-scrap-manager — parse les arguments et dispatche vers les commandes.
**Files:** `main.go`, `internal/cli/cli.go`, `internal/cli/config.go`
**Exports:** `cli.Execute(args []string, out io.Writer) int`
**Depends on:** [`modules/config.md`](config.md)

## Sous-commande `config`
`internal/cli/config.go` implémente `runConfig(args []string, out io.Writer) int`, dispatché par `Execute` sur `args[0] == "config"`.

- `config set-registry <path>` — définit le chemin du registry (converti en chemin absolu via `internal/config`), persiste dans le fichier de config.
- `config add-roms-folder <path>` — ajoute un dossier de ROMs Batocera à surveiller (dédoublonné par chemin absolu).
- `config list` — affiche le registry configuré (ou "(not set)") et la liste des dossiers de ROMs (ou "(none)").
- Toute sous-commande absente ou inconnue retourne le code de sortie 1.

Le chemin du fichier de config est résolu via `config.DefaultPath()` : variable d'environnement `BATOCERA_SCRAP_MANAGER_CONFIG` si définie, sinon `os.UserConfigDir()/batocera-scrap-manager/config.json`.
