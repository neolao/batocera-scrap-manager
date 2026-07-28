package webui

import (
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/site"
	"github.com/neolao/batocera-scrap-manager/internal/store"
)

// The deletion's outcome travels to the game list, which is where a deletion
// lands: the game's own page is gone, so it cannot confirm anything. The name
// is carried along because the list cannot show what disappeared from it.
const (
	deletedParam = "deleted"
	systemParam  = "system"
	warningParam = "warning"

	deletedAnchor = "#deleted"
)

// The caveats a deletion may carry: what was asked for happened either way,
// so they are added to the confirmation rather than replacing it.
const (
	warningStaleSite = "stale-site"
	warningMediaLeft = "media-left"
)

// deleteConfirmation is the page asking whether to really delete a game: what
// is about to be deleted, named file by file, and the one control confirming
// it. Protected states that this game is protected against updates — which
// does not stand in the way of deleting it, and is worth saying before the
// user finds out.
type deleteConfirmation struct {
	Name      string
	System    string
	SystemURL string
	GameURL   string
	Action    string
	Files     []string
	Protected bool
	// Problem states why the deletion did not happen, when the confirmation is
	// shown again after a failed attempt rather than for the first time.
	Problem string
}

// serveDeleteConfirmation renders the page asking to confirm a game's
// deletion. Deleting is what the submission does; opening this page never
// changes anything.
func (ui *webUI) serveDeleteConfirmation(w http.ResponseWriter, r *http.Request) {
	system, id := r.PathValue("system"), r.PathValue("id")

	ui.mu.RLock()
	entry, found := ui.reg.FindByID(system, id)
	registryFolder := ui.registryFolder
	ui.mu.RUnlock()
	if !found {
		renderGameGone(w, system, id)
		return
	}

	// Revalidated rather than served from the browser's cache, so going back to
	// this page after the deletion does not offer to delete the game again. It
	// stays storable: no-store would also defeat the list's scroll restoration.
	w.Header().Set("Cache-Control", "no-cache")
	render(w, http.StatusOK, deleteTemplate, deleteForm(entry, registryFolder, ""))
}

// serveWrongDeleteMethod refuses a method the delete URL does not support,
// naming the ones it does. It needs its own handler because serveWrongMethod
// is worded for the edit URL.
func (ui *webUI) serveWrongDeleteMethod(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
	renderProblem(w, http.StatusMethodNotAllowed, "Not allowed",
		r.Method+" is not something this page accepts.")
}

// deleteGame deletes a game from the registry, writes the consultation site
// without it, then redirects to the game list — the game's own page no longer
// exists to redirect back to.
//
// Unlike every other change the web UI makes, the served snapshot is swapped
// in as soon as the game's file is gone rather than once the whole write
// succeeded: the deletion is already persisted at that point, and a snapshot
// still holding the game would have the next write of any kind recreate its
// file (see decisions/022).
func (ui *webUI) deleteGame(w http.ResponseWriter, r *http.Request) {
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

	system, id := r.PathValue("system"), r.PathValue("id")

	ui.mu.Lock()
	defer ui.mu.Unlock()

	entry, found := ui.reg.FindByID(system, id)
	if !found {
		renderGameGone(w, system, id)
		return
	}

	var warnings []string
	candidate := ui.reg.Clone()
	err := registry.RemoveByID(candidate, ui.registryFolder, system, id)
	switch {
	case errors.Is(err, registry.ErrMediaLeftBehind):
		warnings = append(warnings, warningMediaLeft)
	case err != nil:
		page := deleteForm(entry, ui.registryFolder, "The registry folder could not be written to, so this game was not deleted.")
		render(w, http.StatusInternalServerError, deleteTemplate, page)
		return
	}
	ui.reg = candidate

	if err := store.Save(candidate, ui.registryFolder); err != nil {
		warnings = append(warnings, warningStaleSite)
	}
	systemSurvives := len(entriesOfSystem(candidate.Entries, system)) > 0
	http.Redirect(w, r, deletedURL(entry.Game.Name, system, systemSurvives, warnings), http.StatusSeeOther)
}

// deletedURL builds the URL of the list confirming the deletion of a game:
// the game's own system, or the home page when that game was the last of its
// system — landing on a system that no longer exists would answer a 404 to a
// deletion that succeeded.
func deletedURL(name, system string, systemSurvives bool, warnings []string) string {
	query := url.Values{deletedParam: {name}, systemParam: {system}}
	for _, warning := range warnings {
		query.Add(warningParam, warning)
	}

	landing := "/"
	if systemSurvives {
		landing = systemURL(system)
	}
	return landing + "?" + query.Encode() + deletedAnchor
}

// renderGameGone renders the not-found page for a game the deletion flow could
// not find. It suggests the deletion already happened rather than reading as a
// failure: reaching this from a stale page — a replayed submission, a
// bookmarked confirmation — is exactly what it usually means.
func renderGameGone(w http.ResponseWriter, system, id string) {
	renderProblem(w, http.StatusNotFound, "Not found",
		"No game named "+id+" in system "+system+" — it may already have been deleted.")
}

// deleteForm builds the confirmation's view for entry, listing what is really
// on disk: promising to delete four media a game never had would misrepresent
// what is at stake, and the media files are the part a user does not expect to
// lose.
func deleteForm(entry registry.Entry, registryFolder, problem string) deleteConfirmation {
	id := registry.GameID(entry.Game.Path)
	return deleteConfirmation{
		Name:      entry.Game.Name,
		System:    entry.System,
		SystemURL: systemURL(entry.System),
		GameURL:   gameURL(entry.System, id),
		Action:    gameURL(entry.System, id) + "/delete",
		Files:     deletedFiles(entry, registryFolder),
		Protected: len(entry.ManualFields) > 0,
		Problem:   problem,
	}
}

// deletedFiles names every file the deletion of entry erases: its metadata
// file, then each medium it references that is actually present in the
// registry folder. Every path is relative to the registry folder — the media
// ones already are, and the metadata file is prefixed to match, or the list
// would mix two different origins and read as if they sat in two places.
func deletedFiles(entry registry.Entry, registryFolder string) []string {
	files := []string{entry.System + "/" + registry.GameID(entry.Game.Path) + ".json"}

	view := site.GroupBySystem([]registry.Entry{entry}, registryFolder)[0].Games[0]
	for _, escapedPath := range []string{view.ImagePath, view.VideoPath, view.MarqueePath, view.ThumbnailPath} {
		// GroupBySystem leaves the path empty when the file is not on disk, and
		// percent-encodes the rest for use in a URL — undone here, since this is
		// a file name shown to a human rather than a link.
		if escapedPath == "" {
			continue
		}
		relPath, err := url.PathUnescape(escapedPath)
		if err != nil {
			relPath = escapedPath
		}
		relPath = filepath.ToSlash(filepath.Clean(relPath))
		if !slices.Contains(files, relPath) {
			files = append(files, relPath)
		}
	}
	return files
}

// deletedConfirmation turns the outcome a deletion redirected with into the
// sentence the game list confirms it with — nothing at all when the list was
// simply browsed to. The game is named because the list is the one page that
// cannot show what left it.
func deletedConfirmation(name, system string, warnings []string) string {
	if name == "" {
		return ""
	}

	message := "Deleted \"" + name + "\" from " + system + " — its metadata and media files are gone."
	if slices.Contains(warnings, warningMediaLeft) {
		message += " Some of its media files could not be deleted and are still in the " + system + " folder of the registry."
	}
	if slices.Contains(warnings, warningStaleSite) {
		message += staleCaveat
	}
	return message
}

// deleteTemplate renders the confirmation asking whether to really delete a
// game: what is about to be erased, named file by file, and the two ways out.
var deleteTemplate = newPage("delete", `
{{define "title"}}Delete {{.Name}} - Registry{{end}}
{{define "body"}}
<nav class="crumbs" aria-label="Breadcrumb">
<a href="/">Registry</a><span class="crumbs__sep">/</span><a href="{{.SystemURL}}">{{.System}}</a><span class="crumbs__sep">/</span><a href="{{.GameURL}}">{{.Name}}</a><span class="crumbs__sep">/</span><span class="crumbs__current">Delete</span>
</nav>
<main>
<article class="game">
<h2 class="game__title">Delete {{.Name}}?</h2>
{{if .Problem}}
<div class="errors" role="alert" tabindex="-1" id="errors">
<p class="errors__title">Not deleted</p>
<p>{{.Problem}}</p>
</div>
{{end}}
<div class="confirm">
<p class="confirm__lead">This removes {{.Name}} from the registry of {{.System}}. These files will be deleted:</p>
<ul class="confirm__files">
{{range .Files}}<li><code>{{.}}</code></li>
{{end}}
</ul>
{{if .Protected}}
<p class="confirm__note">This game is protected against updates — deleting it removes it anyway, hand-edited values included.</p>
{{end}}
<p class="confirm__warning">This cannot be undone.</p>
<div class="form__actions">
<form method="post" action="{{.Action}}">
<button class="button button--danger" type="submit">Delete this game</button>
</form>
<a class="button button--quiet" href="{{.GameURL}}">Cancel</a>
</div>
</div>
</article>
</main>
{{end}}
`)
