package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/neolao/batocera-scrap-manager/internal/config"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

const scrapeUsage = `Usage:
  batocera-scrap-manager scrape
  batocera-scrap-manager scrape <path>

Without a path, completes missing metadata and media for every game in
every configured ROMs folder, using the registry as the source of
already-known information.

With the path to a specific ROM file, only that game is completed.
`

func runScrape(args []string, out io.Writer) int {
	if len(args) > 0 && args[0] == "--help" {
		fmt.Fprint(out, scrapeUsage)
		return 0
	}

	cfg, reg, ok := loadConfigAndRegistry(out)
	if !ok {
		return 1
	}

	if len(args) > 0 {
		return runScrapeTargeted(reg, cfg, args[0], out)
	}

	var processed, completed, failed int
	for _, romsFolder := range cfg.RomsFolders {
		onProgress := newCompletionProgressReporter(out, romsFolder)

		folderProcessed, folderCompleted, folderFailed, err := registry.CompleteRomsFolder(reg, romsFolder, cfg.RegistryFolder, onProgress)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return 1
		}
		processed += folderProcessed
		completed += folderCompleted
		failed += folderFailed
	}

	fmt.Fprintf(out, "%d processed, %d completed, %d failed\n", processed, completed, failed)
	return 0
}

// runScrapeTargeted completes a single game, identified by its real path on
// disk, instead of every game in every configured ROMs folder. See
// decisions/013 for why the path is a real disk path rather than one
// relative to a configured ROMs folder.
func runScrapeTargeted(reg *registry.Registry, cfg config.Config, path string, out io.Writer) int {
	romsFolder, system, romFilename, err := resolveGamePath(cfg, path)
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	onProgress := newCompletionProgressReporter(out, romsFolder)

	completedGame, failedGame, err := registry.CompleteGame(reg, romsFolder, cfg.RegistryFolder, system, romFilename, onProgress)
	if err != nil {
		if errors.Is(err, registry.ErrGameNotFound) {
			fmt.Fprintf(out, "error: no game found in the registry for %q (system: %s)\n", path, system)
			return 1
		}
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	completed, failed := 0, 0
	if completedGame {
		completed = 1
	}
	if failedGame {
		failed = 1
	}
	fmt.Fprintf(out, "%d processed, %d completed, %d failed\n", 1, completed, failed)
	return 0
}

// resolveGamePath finds which configured ROMs folder path falls under, and
// derives the Batocera system (the ROMs folder's immediate subfolder) and
// the ROM filename from it.
func resolveGamePath(cfg config.Config, path string) (romsFolder, system, romFilename string, err error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", "", err
	}

	for _, folder := range cfg.RomsFolders {
		rel, err := filepath.Rel(folder, absPath)
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			continue
		}

		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) < 2 {
			continue
		}

		return folder, parts[0], filepath.Base(absPath), nil
	}

	return "", "", "", fmt.Errorf("%q is not inside any configured ROMs folder", path)
}
