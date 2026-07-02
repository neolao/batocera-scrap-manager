package cli

import (
	"fmt"
	"io"

	"github.com/neolao/batocera-scrap-manager/internal/config"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/site"
)

// loadConfigAndRegistry loads the persisted configuration and registry,
// writing an error message to out and returning ok=false if either step
// fails or the registry folder is not configured yet.
func loadConfigAndRegistry(out io.Writer) (cfg config.Config, reg *registry.Registry, ok bool) {
	configPath, err := config.DefaultPath()
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return config.Config{}, nil, false
	}

	cfg, err = config.Load(configPath)
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return config.Config{}, nil, false
	}
	if cfg.RegistryFolder == "" {
		fmt.Fprintln(out, "error: registry not configured, run 'config set-registry' first")
		return config.Config{}, nil, false
	}

	reg, err = registry.Load(cfg.RegistryFolder)
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return config.Config{}, nil, false
	}

	return cfg, reg, true
}

// saveAndGenerateSite persists reg to cfg's registry folder and regenerates
// the static site from it, writing an error message to out and returning
// false if either step fails.
func saveAndGenerateSite(cfg config.Config, reg *registry.Registry, out io.Writer) bool {
	if err := registry.Save(cfg.RegistryFolder, reg); err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return false
	}
	if err := site.Generate(reg, cfg.RegistryFolder); err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return false
	}
	return true
}

// newCompletionProgressReporter builds a registry.CompletionEvent handler
// that prints a header line the first time a system is seen, then one line
// per game, identifying romsFolder as the folder being processed.
func newCompletionProgressReporter(out io.Writer, romsFolder string) func(registry.CompletionEvent) {
	lastSystem := ""
	return func(e registry.CompletionEvent) {
		if e.System != lastSystem {
			fmt.Fprintf(out, "%s: %d game(s)\n", e.System, e.GameCount)
			lastSystem = e.System
		}
		fmt.Fprintf(out, "  [%d/%d] %s: %s\n", e.GameIndex, e.GameCount, romsFolder, e.GameName)
	}
}
