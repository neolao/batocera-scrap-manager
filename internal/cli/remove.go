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

	err := registry.Remove(reg, cfg.RegistryFolder, system, romFilename)
	switch {
	case errors.Is(err, registry.ErrGameNotFound):
		fmt.Fprintf(out, "error: no game found for system %q and filename %q\n", system, romFilename)
		return 1
	case err != nil && !errors.Is(err, registry.ErrMediaLeftBehind):
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	// Media left behind is not a failed removal: the game itself is gone, and
	// only files nothing references anymore remain. Reporting it as an error
	// would have the user retry a removal that already happened.
	fmt.Fprintf(out, "removed %s (system: %s)\n", romFilename, system)
	if err != nil {
		fmt.Fprintf(out, "warning: %v\n", err)
	}
	return 0
}
