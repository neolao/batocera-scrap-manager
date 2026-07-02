// Package cli implements the batocera-scrap-manager command-line interface.
package cli

import (
	"fmt"
	"io"
)

const version = "0.2.0"

const usage = `batocera-scrap-manager - manage game scraping data for Batocera

Usage:
  batocera-scrap-manager [command]

Commands:
  --version   Print the version and exit
  --help      Print this help message and exit
  config      Configure the registry path and ROMs folders
  update      Update the registry from the configured ROMs folders
  scrape      Complete missing ROMs metadata and media from the registry
  remove      Remove a game's entry (metadata and media) from the registry
`

// Execute runs the CLI with the given arguments and writes output to out.
// It returns the process exit code.
func Execute(args []string, out io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(out, usage)
		return 0
	}

	switch args[0] {
	case "--version":
		fmt.Fprintln(out, version)
		return 0
	case "--help":
		fmt.Fprint(out, usage)
		return 0
	case "config":
		return runConfig(args[1:], out)
	case "update":
		return runUpdate(out)
	case "scrape":
		return runScrape(out)
	case "remove":
		return runRemove(args[1:], out)
	default:
		fmt.Fprintf(out, "unknown command: %s\n", args[0])
		return 1
	}
}
