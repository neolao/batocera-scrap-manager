package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/neolao/batocera-scrap-manager/internal/config"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

const removeUsage = `Usage:
  batocera-scrap-manager remove <system> <rom-path>
`

func runRemove(args []string, out io.Writer) int {
	if len(args) < 2 {
		fmt.Fprint(out, removeUsage)
		return 1
	}
	system, romPath := args[0], args[1]

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

	if err := registry.Remove(reg, cfg.RegistryFolder, system, romPath); err != nil {
		if errors.Is(err, registry.ErrGameNotFound) {
			fmt.Fprintf(out, "error: no game found for system %q and path %q\n", system, romPath)
			return 1
		}
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	fmt.Fprintf(out, "removed %s (system: %s)\n", romPath, system)
	return 0
}
