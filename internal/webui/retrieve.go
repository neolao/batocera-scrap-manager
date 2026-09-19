package webui

import (
	"errors"
	"net/http"
	"net/url"
	"slices"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/store"
)

// retrieveURL is the one endpoint pulling a single gap's already-scraped
// local data — read from its ROMs folder's own gamelist.xml — into the
// registry. Reached only from a Listed gap's row on that folder's status page
// (see decisions/041): a gap with no local entry has nothing to retrieve.
const retrieveURL = romsFolderStatusURL + "/retrieve"

// retrieveSystemParam and retrieveROMParam name, in a retrieve submission,
// the system and ROM filename of the one gap it targets. The folder travels
// under statusFolderParam, the same key the status page's own link already
// uses.
const (
	retrieveSystemParam = "system"
	retrieveROMParam    = "rom"
)

// retrievedParam and retrievedNameParam carry the outcome of a retrieve to
// the status page it redirects to, which renders the matching confirmation.
const (
	retrievedParam     = "retrieved"
	retrievedNameParam = "name"
	retrievedAnchor    = "#retrieved"
)

// The outcomes a retrieve may redirect with.
const (
	retrievedAdded     = "added"
	retrievedUpdated   = "updated"
	retrievedUnchanged = "unchanged"
	retrievedNothing   = "nothing"
	retrievedFailed    = "failed"
)

// retrievedConfirmations maps the outcome a retrieve redirected with to the
// sentence completing the game's own name — a page listing many gaps at once
// must always say which one a banner is about.
var retrievedConfirmations = map[string]string{
	retrievedAdded:     " was added to the registry.",
	retrievedUpdated:   "'s entry in the registry was refreshed from this folder.",
	retrievedUnchanged: " already matched the registry — nothing changed.",
	retrievedNothing:   " has no description or cover art locally yet — nothing to retrieve.",
	retrievedFailed:    " could not be retrieved — its local files may have changed since this page was opened.",
}

// retrievedConfirmation turns what a retrieve redirected with into the
// sentence the status page confirms it with, or nothing when the query is not
// a retrieve's.
func retrievedConfirmation(query url.Values) string {
	name := query.Get(retrievedNameParam)
	if name == "" {
		return ""
	}
	suffix, known := retrievedConfirmations[query.Get(retrievedParam)]
	if !known {
		return ""
	}
	return name + suffix
}

// retrievedURL builds the status page URL confirming a retrieve: the folder
// it stays on, the game it named, and how it turned out.
func retrievedURL(folder, name, outcome string) string {
	query := url.Values{
		statusFolderParam:  {folder},
		retrievedNameParam: {name},
		retrievedParam:     {outcome},
	}
	return romsFolderStatusURL + "?" + query.Encode() + retrievedAnchor
}

// retrievableGap reports the gap (system, rom) currently is on folder's
// status, when it is one a retrieve may act on — Listed, so there is a local
// entry to read from at all. Recomputed fresh rather than trusted from the
// request, the same rule a forged folder name is already refused by (see
// requestedSend in send.go): a page rendered earlier may no longer agree with
// what is on disk, or with the registry, right now.
func retrievableGap(reg *registry.Registry, folder, system, rom string) (registry.GameGap, bool) {
	status, err := registry.RomsFolderStatus(reg, folder)
	if err != nil {
		return registry.GameGap{}, false
	}
	for _, sys := range status.Systems {
		if sys.System != system {
			continue
		}
		for _, gap := range sys.Gaps {
			if gap.Listed && gap.ROMFilename == rom {
				return gap, true
			}
		}
	}
	return registry.GameGap{}, false
}

// gapDisplayName is the name a confirmation names a gap by — its own when
// known, the ROM filename otherwise, exactly as the status page's own list
// already falls back (see gapSortKey in internal/registry/status.go).
func gapDisplayName(gap registry.GameGap) string {
	if gap.Name != "" {
		return gap.Name
	}
	return gap.ROMFilename
}

// renderBadRetrieve refuses a retrieve naming a folder this server does not
// offer, or a (system, rom) pair that is not, right now, a gap this folder's
// status could pull from — a forged request as much as a page grown stale
// since it was rendered.
func renderBadRetrieve(w http.ResponseWriter) {
	renderProblem(w, http.StatusBadRequest, "Bad request",
		"This request named a ROMs folder or a game this server does not offer to retrieve.")
}

// retrieveGame pulls one gap's already-scraped local data into the registry,
// then redirects back to its folder's status page with a banner naming the
// outcome.
//
// It writes only the registry — never the ROMs folder — so, like a
// correction or a protection, it runs inside its own request with no
// confirmation page first (decisions/041), unlike Completion or Replacement.
func (ui *webUI) retrieveGame(w http.ResponseWriter, r *http.Request) {
	if crossSite(r) {
		renderProblem(w, http.StatusForbidden, "Refused",
			"This request did not come from the registry's own pages.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		renderProblem(w, http.StatusBadRequest, "Bad request", "This request could not be read.")
		return
	}

	folder := r.PostFormValue(statusFolderParam)
	system := r.PostFormValue(retrieveSystemParam)
	rom := r.PostFormValue(retrieveROMParam)
	if !slices.Contains(ui.romsFolders, folder) {
		renderBadRetrieve(w)
		return
	}

	ui.mu.Lock()
	defer ui.mu.Unlock()

	gap, ok := retrievableGap(ui.reg, folder, system, rom)
	if !ok {
		renderBadRetrieve(w)
		return
	}
	name := gapDisplayName(gap)

	candidate := ui.reg.Clone()
	added, updated, unchanged, err := registry.ImportGame(candidate, folder, ui.registryFolder, system, rom, nil)
	if err != nil {
		http.Redirect(w, r, retrievedURL(folder, name, retrievedFailed), http.StatusSeeOther)
		return
	}

	outcome := retrievedNothing
	switch {
	case added > 0:
		outcome = retrievedAdded
	case updated > 0:
		outcome = retrievedUpdated
	case unchanged > 0:
		outcome = retrievedUnchanged
	}

	if outcome != retrievedNothing {
		if saveErr := store.Save(candidate, ui.registryFolder); saveErr != nil && !errors.Is(saveErr, store.ErrSiteNotRegenerated) {
			http.Redirect(w, r, retrievedURL(folder, name, retrievedFailed), http.StatusSeeOther)
			return
		}
		ui.reg = candidate
	}

	http.Redirect(w, r, retrievedURL(folder, name, outcome), http.StatusSeeOther)
}
