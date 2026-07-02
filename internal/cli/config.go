package cli

import (
	"fmt"
	"io"

	"github.com/neolao/batocera-scrap-manager/internal/config"
)

const configUsage = `Usage:
  batocera-scrap-manager config set-registry <path>
  batocera-scrap-manager config add-roms-folder <path>
  batocera-scrap-manager config list
`

func runConfig(args []string, out io.Writer) int {
	if len(args) > 0 && args[0] == "--help" {
		fmt.Fprint(out, configUsage)
		return 0
	}
	if len(args) == 0 {
		fmt.Fprint(out, configUsage)
		return 1
	}

	path, err := config.DefaultPath()
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	switch args[0] {
	case "set-registry":
		if len(args) < 2 {
			fmt.Fprintln(out, "error: missing path argument")
			return 1
		}
		if err := cfg.SetRegistryFolder(args[1]); err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return 1
		}
		if err := config.Save(path, cfg); err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(out, "registry set to %s\n", cfg.RegistryFolder)
		return 0

	case "add-roms-folder":
		if len(args) < 2 {
			fmt.Fprintln(out, "error: missing path argument")
			return 1
		}
		added, err := cfg.AddRomsFolder(args[1])
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return 1
		}
		if !added {
			fmt.Fprintln(out, "ROMs folder already configured")
			return 0
		}
		if err := config.Save(path, cfg); err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(out, "ROMs folder added: %s\n", cfg.RomsFolders[len(cfg.RomsFolders)-1])
		return 0

	case "list":
		if cfg.RegistryFolder == "" {
			fmt.Fprintln(out, "registry: (not set)")
		} else {
			fmt.Fprintf(out, "registry: %s\n", cfg.RegistryFolder)
		}
		if len(cfg.RomsFolders) == 0 {
			fmt.Fprintln(out, "roms folders: (none)")
		} else {
			fmt.Fprintln(out, "roms folders:")
			for _, folder := range cfg.RomsFolders {
				fmt.Fprintf(out, "  - %s\n", folder)
			}
		}
		return 0

	default:
		fmt.Fprintf(out, "unknown config subcommand: %s\n", args[0])
		return 1
	}
}
