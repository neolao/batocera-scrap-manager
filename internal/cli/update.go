package cli

import (
	"fmt"
	"io"

	"github.com/neolao/batocera-scrap-manager/internal/config"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

func runUpdate(out io.Writer) int {
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

	var added, updated, unchanged int
	for _, romsFolder := range cfg.RomsFolders {
		a, u, unc, err := registry.ImportFromRomsFolder(reg, romsFolder, cfg.RegistryFolder)
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

	fmt.Fprintf(out, "%d added, %d updated, %d unchanged\n", added, updated, unchanged)
	return 0
}
