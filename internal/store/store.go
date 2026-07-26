// Package store commits a change made to the registry: it writes the
// registry itself and regenerates the static consultation site derived from
// it, so the two never drift apart. Every caller that modifies the registry —
// the CLI commands and the web UI alike — goes through it rather than
// repeating the sequence (see decisions/020).
package store

import (
	"errors"
	"fmt"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/site"
)

// ErrSiteNotRegenerated reports the half-failure worth telling apart: the
// registry — the source of truth — was written, but the consultation site
// derived from it could not be regenerated and is now stale. A caller that
// treated it as a plain failure would tell the user nothing was saved, and
// have them redo a change that already applied.
var ErrSiteNotRegenerated = errors.New("store: registry saved but the consultation site was not regenerated")

// Save writes reg into the registry folder at registryFolder and regenerates
// the consultation site from it. When only the site regeneration fails, the
// returned error wraps ErrSiteNotRegenerated.
func Save(reg *registry.Registry, registryFolder string) error {
	if err := registry.Save(registryFolder, reg); err != nil {
		return err
	}
	if err := site.Generate(reg, registryFolder); err != nil {
		return fmt.Errorf("%w: %v", ErrSiteNotRegenerated, err)
	}
	return nil
}
