package webui

import (
	"errors"
	"net/http"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/store"
)

// protectedParam carries the state the user asked for, rather than an
// instruction to flip the current one: a page opened before the game's
// protection changed elsewhere then submits a no-op instead of the opposite of
// what its button offered.
const protectedParam = "protected"

const (
	protectOn  = "on"
	protectOff = "off"
)

// The three states a game's page states in words — never by colour alone —
// and the label of the single control each of them offers.
const (
	notProtectedSummary    = "Not protected — updates can overwrite every field."
	partlyProtectedSummary = "Partly protected — updates leave the hand-edited fields alone."
	protectedSummary       = "Protected — updates leave every field alone."

	protectLabel   = "Protect this game"
	unprotectLabel = "Let updates overwrite this game"
)

// protectionControl is the game page's protection section: the state in words,
// and the one control that state offers. Label is empty when no control is
// offered — the partly protected state deliberately has no bulk lift, since it
// would discard which fields the user corrected by hand (see decisions/021).
type protectionControl struct {
	Summary string
	Action  string
	Label   string
	Value   string
}

// protectionOf describes how entry's page presents its protection.
func protectionOf(entry registry.Entry, system, id string) protectionControl {
	control := protectionControl{Action: gameURL(system, id) + "/protect"}

	switch {
	case entry.FullyProtected():
		control.Summary = protectedSummary
		control.Label, control.Value = unprotectLabel, protectOff
	case len(entry.ManualFields) > 0:
		// Lifting from here is offered field by field on the edit form instead:
		// in bulk it would silently drop which fields were corrected, and that
		// cannot be reconstructed.
		control.Summary = partlyProtectedSummary
		control.Label, control.Value = protectLabel, protectOn
	default:
		control.Summary = notProtectedSummary
		control.Label, control.Value = protectLabel, protectOn
	}
	return control
}

// setProtection applies the protection state a game's page asked for, writes
// the registry and the consultation site, then redirects back to that page.
// Nothing is committed to the served snapshot until the write to disk
// succeeded, exactly as a metadata correction is.
func (ui *webUI) setProtection(w http.ResponseWriter, r *http.Request) {
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

	apply, outcome, ok := requestedProtection(r.PostFormValue(protectedParam))
	if !ok {
		renderProblem(w, http.StatusBadRequest, "Bad request",
			"This request did not say whether to protect the game.")
		return
	}

	system, id := r.PathValue("system"), r.PathValue("id")

	ui.mu.Lock()
	defer ui.mu.Unlock()

	if _, found := ui.reg.FindByID(system, id); !found {
		renderGameNotFound(w, system, id)
		return
	}

	candidate := ui.reg.Clone()
	if err := apply(candidate, system, id); err != nil {
		renderGameNotFound(w, system, id)
		return
	}

	err := store.Save(candidate, ui.registryFolder)
	if err != nil && !errors.Is(err, store.ErrSiteNotRegenerated) {
		renderProblem(w, http.StatusInternalServerError, "Not changed",
			"The registry folder could not be written to, so this game's protection was not changed.")
		return
	}
	ui.reg = candidate

	if err != nil {
		outcome += staleSuffix
	}
	http.Redirect(w, r, gameURL(system, id)+"?"+savedParam+"="+outcome+savedAnchor, http.StatusSeeOther)
}

// requestedProtection reads the state a submission asked for, reporting
// whether it named one at all: a missing or unrecognized value is refused
// rather than silently defaulted to either state.
func requestedProtection(state string) (apply func(*registry.Registry, string, string) error, outcome string, ok bool) {
	switch state {
	case protectOn:
		return registry.Protect, savedProtected, true
	case protectOff:
		return registry.Unprotect, savedUnprotected, true
	default:
		return nil, "", false
	}
}
