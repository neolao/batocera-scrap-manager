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
	savedFully       = "1"
	savedNoSite      = "stale-site"
	savedProtected   = "protected"
	savedUnprotected = "unprotected"
	savedAnchor      = "#saved"

	// staleSuffix turns an outcome into the variant reporting that the
	// registry was written but the site derived from it was not.
	staleSuffix = "-stale"

	// staleCaveat is appended to an outcome's sentence in that case, rather
	// than replacing it: what was asked for did happen.
	staleCaveat = " The consultation site could not be regenerated — run 'update' to rebuild it."
)

// savedConfirmations maps the outcome a change redirected with to the sentence
// the game page confirms it with. The sentences state the resulting state
// rather than the action, so they stay true even when the submission turned out
// to be a no-op — a stale page asking again for what already holds.
var savedConfirmations = map[string]string{
	savedFully:       "Metadata saved to the registry.",
	savedNoSite:      "Metadata saved to the registry, but the consultation site could not be regenerated — run 'update' to rebuild it.",
	savedProtected:   "This game is protected — updates will leave its metadata alone.",
	savedUnprotected: "This game is no longer protected — updates can overwrite its metadata again.",
}

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
	errs := validate(values, pathError(ui.reg, system, id, values.Path))
	if len(errs) > 0 {
		render(w, http.StatusUnprocessableEntity, editTemplate, editForm(entry, values, errs))
		return
	}

	candidate := ui.reg.Clone()
	metadata := storedMetadata(values, entry.Game)
	if err := registry.UpdateMetadata(candidate, system, id, metadata, r.Form[handBackParam]); err != nil {
		renderGameNotFound(w, system, id)
		return
	}

	renamed, moved, ok := ui.applyPathChange(w, candidate, entry, values, system, id)
	if !ok {
		return
	}
	newID := registry.GameID(values.Path)

	siteErr := store.Save(candidate, ui.registryFolder)
	if siteErr != nil && !errors.Is(siteErr, store.ErrSiteNotRegenerated) {
		page := editForm(entry, values, nil)
		page.Problem = "The registry folder could not be written to, so this correction was not saved."
		render(w, http.StatusInternalServerError, editTemplate, page)
		return
	}
	ui.reg = candidate

	// From here the change is committed: the registry on disk holds it, so the
	// served snapshot must too. Erasing the file the game was stored under is
	// what keeps it from loading twice on the next start — but a failure there
	// is a caveat on a change that did happen, never a refusal (decisions/024).
	leftBehind := ""
	if moved {
		if err := registry.RemoveGameFile(ui.registryFolder, system, id); err != nil {
			leftBehind = registry.EntryFileName(id)
		}
	}

	query := savedRedirectQuery(renamed, moved, leftBehind, values.Path, siteErr)
	http.Redirect(w, r, gameURL(system, newID)+"?"+query.Encode()+savedAnchor, http.StatusSeeOther)
}

// applyPathChange moves candidate's entry to values.Path when it differs from
// the game's current one — applied after the metadata, since correcting the
// path moves the entry under a new identifier and UpdateMetadata addresses it
// by the old one. The domain refuses on its own terms even though pathError
// already checked the same rules on the submission: it owns them, and a rule
// added there must not slip through here silently. moved reports whether the
// change also lands the entry under a different identifier, which is what
// decides whether its previous file is left for the caller to erase. ok is
// false once the refusal has already been rendered as the form's own error.
func (ui *webUI) applyPathChange(
	w http.ResponseWriter, candidate *registry.Registry, entry registry.Entry, values editValues, system, id string,
) (renamed, moved, ok bool) {
	renamed = values.Path != entry.Game.Path
	if !renamed {
		return false, false, true
	}
	if err := registry.ChangePath(candidate, system, id, values.Path); err != nil {
		render(w, http.StatusUnprocessableEntity, editTemplate,
			editForm(entry, values, map[string]string{pathKey: pathRefusal(err)}))
		return renamed, false, false
	}
	return renamed, registry.GameID(values.Path) != id, true
}

// savedRedirectQuery builds the query the game page reads to word its own
// confirmation. A renamed game names the path it resulted in, which no table
// keyed by the outcome alone could hold, so it composes its own query instead
// of merely picking an outcome — see savedConfirmation.
func savedRedirectQuery(renamed, moved bool, leftBehind, path string, siteErr error) url.Values {
	outcome := savedFully
	if siteErr != nil {
		outcome = savedNoSite
	}
	query := url.Values{}
	if renamed {
		query = renameQuery(path, moved, leftBehind)
		outcome = savedPath
		if siteErr != nil {
			outcome = savedPath + staleSuffix
		}
	}
	query.Set(savedParam, outcome)
	return query
}

// savedConfirmation turns what a change redirected with into the sentence the
// game page confirms it with — nothing at all when the page was simply browsed
// to. An outcome carrying the stale suffix confirms what did happen, then adds
// what did not. It reads the whole query rather than the outcome alone,
// because a path change names the path it resulted in, which no table holds.
func savedConfirmation(query url.Values) string {
	outcome := query.Get(savedParam)
	if message, found := savedConfirmations[outcome]; found {
		return message
	}
	// A send names the folder it wrote to, and a media change the medium it
	// concerned — neither of which a table keyed by the outcome alone can hold.
	// Both compose their own caveats too, so they are asked before the stale
	// suffix is looked for.
	if message := sentConfirmation(query); message != "" {
		return message
	}
	if message := mediaConfirmation(query); message != "" {
		return message
	}

	base, stale := strings.CutSuffix(outcome, staleSuffix)
	message := ""
	if base == savedPath {
		message = pathConfirmation(query)
	} else if known, found := savedConfirmations[base]; found && stale {
		message = known
	}
	if message == "" {
		return ""
	}
	if stale {
		message += staleCaveat
	}
	return message
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
		Path:      line(pathKey),
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
// free to ignore them — and returns one message per refused field. The ROM
// path's own refusal is passed in rather than computed here: it is the only
// one needing the registry, to tell which game already holds an identifier.
func validate(values editValues, pathRefused string) map[string]string {
	errs := map[string]string{}

	if pathRefused != "" {
		errs[pathKey] = pathRefused
	}
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
