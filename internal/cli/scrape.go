package cli

import (
	"fmt"
	"io"

	"github.com/neolao/batocera-scrap-manager/internal/config"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

func runScrape(out io.Writer) int {
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

	onProgress := func(e registry.CompletionEvent) {
		if e.GameIndex == 1 {
			fmt.Fprintf(out, "%s: %d game(s)\n", e.System, e.GameCount)
		}
		fmt.Fprintf(out, "  [%d/%d] %s\n", e.GameIndex, e.GameCount, e.GameName)
	}

	var processed, completed, failed int
	for _, romsFolder := range cfg.RomsFolders {
		p, c, f, err := registry.CompleteRomsFolder(reg, romsFolder, cfg.RegistryFolder, onProgress)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return 1
		}
		processed += p
		completed += c
		failed += f
	}

	fmt.Fprintf(out, "%d processed, %d completed, %d failed\n", processed, completed, failed)
	return 0
}
