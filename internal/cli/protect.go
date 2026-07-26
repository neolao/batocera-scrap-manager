package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

const protectUsage = `Usage:
  batocera-scrap-manager protect <system> <rom-filename>

Marks every editable field of the game as set by hand, so 'update' stops
overwriting them. No field value is changed: protecting states that the
values already stored are the right ones. Media files are not concerned.
`

const unprotectUsage = `Usage:
  batocera-scrap-manager unprotect <system> <rom-filename>

Clears every mark on the game — including the ones left by corrections made
in the web UI — so 'update' refreshes its fields again. No field value is
changed. To hand a single field back instead, use the checkbox offered under
it in the web UI's edit form.
`

func runProtect(args []string, out io.Writer) int {
	return applyProtection(args, out, protectUsage, "protected", registry.Protect)
}

func runUnprotect(args []string, out io.Writer) int {
	return applyProtection(args, out, unprotectUsage, "unprotected", registry.Unprotect)
}

// applyProtection runs the half of the protection pair described by usage and
// applied by apply, which the two commands differ by — everything else, from
// reading the two arguments to persisting the registry, is one and the same
// sequence. verb names the outcome in the confirmation line.
func applyProtection(args []string, out io.Writer, usage, verb string,
	apply func(*registry.Registry, string, string) error) int {

	if len(args) > 0 && args[0] == "--help" {
		fmt.Fprint(out, usage)
		return 0
	}
	system, romFilename, ok := parseProtectionArgs(args, out, usage)
	if !ok {
		return 1
	}

	cfg, reg, ok := loadConfigAndRegistry(out)
	if !ok {
		return 1
	}

	// The registry addresses a game by its ID, which the web UI already holds
	// but a command line does not: it is derived here through the one helper
	// that owns that rule, never by a second one (see decisions/014).
	if err := apply(reg, system, registry.GameID(romFilename)); err != nil {
		if errors.Is(err, registry.ErrGameNotFound) {
			fmt.Fprintf(out, "error: no game found for system %q and filename %q\n", system, romFilename)
			return 1
		}
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	if !saveAndGenerateSite(cfg, reg, out) {
		return 1
	}

	fmt.Fprintf(out, "%s %s (system: %s)\n", verb, romFilename, system)
	return 0
}

// parseProtectionArgs reads the system and ROM filename both commands take.
// It is stricter than remove's own reading, which silently ignores anything
// past the second argument: a mistyped command must be refused rather than
// applied to whatever its first two words happen to name.
func parseProtectionArgs(args []string, out io.Writer, usage string) (system, romFilename string, ok bool) {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			return failProtectionArgs(out, usage, "unknown option: %s", arg)
		}
	}
	if len(args) != 2 {
		return failProtectionArgs(out, usage, "expected a system and a ROM filename, got %d argument(s)", len(args))
	}
	return args[0], args[1], true
}

// failProtectionArgs reports a malformed command line: the problem first, then
// the usage, so the user sees what to fix and how.
func failProtectionArgs(out io.Writer, usage, format string, args ...any) (string, string, bool) {
	fmt.Fprintf(out, "error: "+format+"\n", args...)
	fmt.Fprint(out, usage)
	return "", "", false
}
