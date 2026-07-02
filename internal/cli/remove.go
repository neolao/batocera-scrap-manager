package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

const removeUsage = `Usage:
  batocera-scrap-manager remove <system> <rom-filename>
`

func runRemove(args []string, out io.Writer) int {
	if len(args) > 0 && args[0] == "--help" {
		fmt.Fprint(out, removeUsage)
		return 0
	}
	if len(args) < 2 {
		fmt.Fprint(out, removeUsage)
		return 1
	}
	system, romFilename := args[0], args[1]

	cfg, reg, ok := loadConfigAndRegistry(out)
	if !ok {
		return 1
	}

	if err := registry.Remove(reg, cfg.RegistryFolder, system, romFilename); err != nil {
		if errors.Is(err, registry.ErrGameNotFound) {
			fmt.Fprintf(out, "error: no game found for system %q and filename %q\n", system, romFilename)
			return 1
		}
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	fmt.Fprintf(out, "removed %s (system: %s)\n", romFilename, system)
	return 0
}
