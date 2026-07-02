package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/neolao/batocera-scrap-manager/internal/config"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/site"
)

const updateUsage = `Usage:
  batocera-scrap-manager update
  batocera-scrap-manager update <path>

Without a path, imports or updates every game from every configured ROMs
folder into the registry.

With the path to a specific ROM file, only that game is imported or
updated.
`

func runUpdate(args []string, out io.Writer) int {
	if len(args) > 0 && args[0] == "--help" {
		fmt.Fprint(out, updateUsage)
		return 0
	}

	configPath, err := config.DefaultPath()
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}
	if cfg.RegistryFolder == "" {
		fmt.Fprintln(out, "error: registry not configured, run 'config set-registry' first")
		return 1
	}

	reg, err := registry.Load(cfg.RegistryFolder)
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	if len(args) > 0 {
		return runUpdateTargeted(reg, cfg, args[0], out)
	}

	onProgress := func(e registry.ProgressEvent) {
		if e.GameIndex == 1 {
			fmt.Fprintf(out, "%s: %d game(s)\n", e.System, e.GameCount)
		}
		fmt.Fprintf(out, "  [%d/%d] %s\n", e.GameIndex, e.GameCount, e.GameName)
	}

	var added, updated, unchanged int
	for _, romsFolder := range cfg.RomsFolders {
		a, u, unc, err := registry.ImportFromRomsFolder(reg, romsFolder, cfg.RegistryFolder, onProgress)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return 1
		}
		added += a
		updated += u
		unchanged += unc
	}

	if err := registry.Save(cfg.RegistryFolder, reg); err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	if err := site.Generate(reg, cfg.RegistryFolder); err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	fmt.Fprintf(out, "%d added, %d updated, %d unchanged\n", added, updated, unchanged)
	return 0
}

// runUpdateTargeted imports a single game, identified by its real path on
// disk, instead of every game in every configured ROMs folder. It reuses
// resolveGamePath (see scrape.go), the same path-resolution convention
// established for scrape's targeted mode — see decisions/013.
func runUpdateTargeted(reg *registry.Registry, cfg config.Config, path string, out io.Writer) int {
	romsFolder, system, romFilename, err := resolveGamePath(cfg, path)
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	onProgress := func(e registry.ProgressEvent) {
		fmt.Fprintf(out, "%s: %d game(s)\n", e.System, e.GameCount)
		fmt.Fprintf(out, "  [%d/%d] %s\n", e.GameIndex, e.GameCount, e.GameName)
	}

	added, updated, unchanged, err := registry.ImportGame(reg, romsFolder, cfg.RegistryFolder, system, romFilename, onProgress)
	if err != nil {
		if errors.Is(err, registry.ErrGameNotFound) {
			fmt.Fprintf(out, "error: no game found in the local gamelist for %q (system: %s)\n", path, system)
			return 1
		}
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	if err := registry.Save(cfg.RegistryFolder, reg); err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	if err := site.Generate(reg, cfg.RegistryFolder); err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	fmt.Fprintf(out, "%d added, %d updated, %d unchanged\n", added, updated, unchanged)
	return 0
}
