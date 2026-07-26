package webui

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/site"
	"github.com/neolao/batocera-scrap-manager/internal/store"
)

// maxFormBytes caps a submitted correction. The description is the only field
// of any length, and 64 KiB holds a very long one — while keeping a stray or
// hostile upload from being read into memory whole.
const maxFormBytes = 64 << 10

// savedParam carries the outcome of a save to the game page that the browser
// is redirected to, which renders the matching confirmation.
const savedParam = "saved"

const (
	savedFully   = "1"
	savedNoSite  = "stale-site"
	savedAnchor  = "#saved"
	savedMessage = "Metadata saved to the registry."
	staleMessage = "Metadata saved to the registry, but the consultation site could not be regenerated — run 'update' to rebuild it."
)

// saveGame applies a submitted correction to one game's metadata, writes the
// registry and the consultation site, then redirects back to the game's page:
// a reload of that page then shows the game, never a second submission of the
// same correction. Nothing is committed to the served snapshot until the
// write to disk succeeded.
func (ui *webUI) saveGame(w http.ResponseWriter, r *http.Request) {
	if crossSite(r) {
		renderProblem(w, http.StatusForbidden, "Refused",
			"This correction did not come from the registry's own pages.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		renderProblem(w, http.StatusBadRequest, "Bad request",
			"This correction could not be read — it may be larger than what the form accepts.")
		return
	}

	system, id := r.PathValue("system"), r.PathValue("id")

	ui.mu.Lock()
	defer ui.mu.Unlock()

	entry, found := ui.reg.FindByID(system, id)
	if !found {
		renderGameNotFound(w, system, id)
		return
	}

	values := submittedValues(r)
	if errs := validate(values); len(errs) > 0 {
		render(w, http.StatusUnprocessableEntity, editTemplate, editForm(entry, values, errs))
		return
	}

	candidate := ui.reg.Clone()
	metadata := storedMetadata(values, entry.Game)
	if err := registry.UpdateMetadata(candidate, system, id, metadata, r.Form[handBackParam]); err != nil {
		renderGameNotFound(w, system, id)
		return
	}

	err := store.Save(candidate, ui.registryFolder)
	if err != nil && !errors.Is(err, store.ErrSiteNotRegenerated) {
		page := editForm(entry, values, nil)
		page.Problem = "The registry folder could not be written to, so this correction was not saved."
		render(w, http.StatusInternalServerError, editTemplate, page)
		return
	}
	ui.reg = candidate

	outcome := savedFully
	if err != nil {
		outcome = savedNoSite
	}
	http.Redirect(w, r, gameURL(system, id)+"?"+savedParam+"="+outcome+savedAnchor, http.StatusSeeOther)
}

// savedConfirmation turns the outcome a save redirected with into the
// sentence the game page confirms it with — nothing at all when the page was
// simply browsed to.
func savedConfirmation(outcome string) string {
	switch outcome {
	case savedFully:
		return savedMessage
	case savedNoSite:
		return staleMessage
	default:
		return ""
	}
}

// crossSite reports whether a submission came from somewhere other than the
// registry's own pages. The web UI has no accounts to authenticate and is
// served on a home network, so this is what keeps another page open in the
// same browser from silently rewriting the registry.
func crossSite(r *http.Request) bool {
	switch r.Header.Get("Sec-Fetch-Site") {
	case "", "same-origin", "none":
	default:
		return true
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	parsed, err := url.Parse(origin)
	return err != nil || parsed.Host != r.Host
}

// submittedValues reads the eight editable values out of a submission. A
// field left out is read as empty — the form always sends all of them, so a
// missing one means the value was cleared, never "leave it as it was".
// Browsers submit a textarea's line breaks as CRLF: they are normalized here,
// or a correction that changed nothing would still rewrite the description
// and make every later import report the game as updated.
func submittedValues(r *http.Request) editValues {
	line := func(key string) string { return strings.TrimSpace(r.PostFormValue(key)) }
	text := func(key string) string {
		return strings.TrimSpace(strings.ReplaceAll(r.PostFormValue(key), "\r\n", "\n"))
	}

	return editValues{
		Name:      line("name"),
		Desc:      text("desc"),
		Rating:    line("rating"),
		Year:      line("year"),
		Developer: line("developer"),
		Publisher: line("publisher"),
		Genre:     line("genre"),
		Players:   line("players"),
	}
}

// validate checks what the form's own constraints only suggest — a client is
// free to ignore them — and returns one message per refused field.
func validate(values editValues) map[string]string {
	errs := map[string]string{}

	if values.Name == "" {
		errs["name"] = "Give the game a name: it is how it appears everywhere else."
	}
	if _, ok := submittedStars(values.Rating); !ok {
		errs["rating"] = "Pick a rating from the list."
	}
	if values.Year != "" {
		year, err := strconv.Atoi(values.Year)
		if err != nil || year < minYear || year > maxYear {
			errs["year"] = "Use a year between " + strconv.Itoa(minYear) + " and " + strconv.Itoa(maxYear) + ", or leave it empty."
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

// submittedStars reads the rating control's value as a star count, reporting
// whether it is one of the choices offered. No rating at all is valid and is
// reported as -1, which is not a star count.
func submittedStars(rating string) (int, bool) {
	if rating == "" {
		return -1, true
	}
	filled, err := strconv.Atoi(rating)
	if err != nil || filled < 0 || filled > 5 {
		return 0, false
	}
	return filled, true
}

// storedMetadata turns the submitted values into the metadata to store,
// against the game as it currently is. The rating and the release date are
// displayed in a lossier form than they are stored in, so an untouched one
// keeps its stored value byte for byte: only a value the user actually
// changed is rewritten (see decisions/019).
func storedMetadata(values editValues, current gamelist.Game) registry.Metadata {
	stored := registry.Metadata{
		Name:        values.Name,
		Desc:        values.Desc,
		Rating:      current.Rating,
		ReleaseDate: current.ReleaseDate,
		Developer:   values.Developer,
		Publisher:   values.Publisher,
		Genre:       values.Genre,
		Players:     values.Players,
	}

	displayed := displayedValues(current)
	if values.Rating != displayed.Rating {
		stored.Rating = ""
		if filled, ok := submittedStars(values.Rating); ok && filled >= 0 {
			stored.Rating = site.RatingFromStars(filled)
		}
	}
	if values.Year != displayed.Year {
		stored.ReleaseDate = site.ReleaseDateFromYear(values.Year)
	}
	return stored
}
