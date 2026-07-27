package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/store"
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

	// Deliberately not saveAndGenerateSite: its "print an error, fail the
	// command" semantics assume nothing was applied, which is never true here —
	// registry.Remove already erased the game's file. Every failure left is a
	// warning about persistence, not a denied removal (see decisions/023).
	if err := store.Save(reg, cfg.RegistryFolder); err != nil {
		fmt.Fprintf(out, "warning: %s\n", persistenceWarning(err))
	}
	return 0
}

// persistenceWarning words a store.Save failure that followed a removal already
// committed on disk. The two cases call for different recovery, so they are not
// folded into one message: a stale consultation site is rebuilt by 'update',
// whereas a registry that could not be rewritten is about the other entries and
// sending the user to 'update' would hide that.
func persistenceWarning(err error) string {
	if errors.Is(err, store.ErrSiteNotRegenerated) {
		return "the consultation site was not regenerated and no longer matches the registry — run 'batocera-scrap-manager update' to rebuild it"
	}
	return fmt.Sprintf("the game is deleted, but the registry could not be fully written, so its other entries may not have been saved: %v", err)
}
