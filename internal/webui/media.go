package webui

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/store"
)

// maxUploadBytes caps one uploaded medium. A scraped video is the largest thing
// this flow ever carries, and 64 MiB holds a generous one — while keeping a
// stray or hostile upload from filling the registry folder. maxUploadMemory is
// what is held in memory before the rest spills to a temporary file, so the cap
// costs no memory of its own.
const (
	maxUploadBytes  = 64 << 20
	maxUploadMemory = 4 << 20
)

// mediaFileParam is the name of the upload form's single file control.
const mediaFileParam = "file"

// mediumParam carries which medium a change concerned to the game page that
// confirms it — the sentence names it, which no table of outcomes could hold.
const mediumParam = "medium"

// The outcome a media change redirects to the game page with.
const (
	mediaStored  = "media-stored"
	mediaCleared = "media-cleared"
)

// mediaLabels is how each medium is named to a user — the registry's own names
// are what URLs and submissions carry, and "image" is not what a player calls a
// cover art.
var mediaLabels = map[registry.Medium]string{
	registry.MediumImage:     "Cover art",
	registry.MediumVideo:     "Video",
	registry.MediumMarquee:   "Marquee",
	registry.MediumThumbnail: "Thumbnail",
}

// mediumControl is one medium on a game's page: what it currently holds, and
// the two things that can be done about it. URL is empty when no file is
// actually in the registry folder, Reference being what the entry names — the
// two differ exactly when a reference points at a file that is not there, which
// the page says rather than showing a broken image. DeleteURL is empty when
// there is nothing to remove.
type mediumControl struct {
	Label     string
	Medium    string
	URL       string
	Reference string
	Video     bool
	UploadURL string
	DeleteURL string
	Accept    string
}

// mediaControlsOf describes how a game's page offers to manage its four media,
// in the order the registry lists them. It reads the resolved view for what is
// really on disk and the entry for what is merely referenced, so a medium whose
// file went missing is told apart from one the game never had.
func mediaControlsOf(entry registry.Entry, system, id string, present map[registry.Medium]string) []mediumControl {
	controls := make([]mediumControl, 0, len(registry.Media()))
	for _, m := range registry.Media() {
		reference := referenceOf(entry, m)
		control := mediumControl{
			Label:     mediaLabels[m],
			Medium:    string(m),
			URL:       mediaURL(present[m]),
			Reference: reference,
			Video:     m == registry.MediumVideo,
			UploadURL: mediumURL(system, id, m),
			Accept:    strings.Join(m.Extensions(), ","),
		}
		if reference != "" {
			control.DeleteURL = mediumURL(system, id, m) + "/delete"
		}
		controls = append(controls, control)
	}
	return controls
}

// referenceOf reads what an entry stores for one medium. It goes through the
// registry's own lookup rather than a second switch here, so the web UI cannot
// hold its own idea of which field a medium is.
func referenceOf(entry registry.Entry, m registry.Medium) string {
	switch m {
	case registry.MediumImage:
		return entry.Game.Image
	case registry.MediumVideo:
		return entry.Game.Video
	case registry.MediumMarquee:
		return entry.Game.Marquee
	case registry.MediumThumbnail:
		return entry.Game.Thumbnail
	}
	return ""
}

// mediumURL builds the URL one medium of one game is managed at, percent-
// encoding the system and the identifier exactly as gameURL does.
func mediumURL(system, id string, m registry.Medium) string {
	return gameURL(system, id) + "/media/" + url.PathEscape(string(m))
}

// requestedMedium resolves the medium a URL names, answering the themed
// not-found page when it names none: a medium is part of the address here, so
// an unknown one is a URL pointing at nothing, not a refused submission.
func requestedMedium(w http.ResponseWriter, r *http.Request) (registry.Medium, bool) {
	name := r.PathValue("medium")
	m, known := registry.LookupMedium(name)
	if !known {
		renderProblem(w, http.StatusNotFound, "Not found", "There is no medium called "+name+".")
		return "", false
	}
	return m, true
}

// uploadMedium stores a file submitted for one of a game's media, then
// redirects back to the game's page, which shows the medium and confirms what
// happened — a reload of that page then shows the game, never a second upload
// of the same file.
//
// The submission is read before the write lock is taken: a body of up to
// maxUploadBytes can take a while to arrive, and holding the snapshot lock for
// that long would queue every page of the site behind one slow upload.
func (ui *webUI) uploadMedium(w http.ResponseWriter, r *http.Request) {
	if crossSite(r) {
		renderProblem(w, http.StatusForbidden, "Refused",
			"This upload did not come from the registry's own pages.")
		return
	}
	m, known := requestedMedium(w, r)
	if !known {
		return
	}

	system, id := r.PathValue("system"), r.PathValue("id")
	ui.mu.RLock()
	entry, found := ui.reg.FindByID(system, id)
	ui.mu.RUnlock()
	if !found {
		renderGameNotFound(w, system, id)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadMemory); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			ui.renderGameProblem(w, entry, http.StatusRequestEntityTooLarge,
				"This file is larger than the "+megabytes(maxUploadBytes)+" an upload may carry, so nothing was stored.")
			return
		}
		ui.renderGameProblem(w, entry, http.StatusBadRequest, "This upload could not be read.")
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := r.FormFile(mediaFileParam)
	if err != nil {
		ui.renderGameProblem(w, entry, http.StatusBadRequest, "Choose a file to upload first.")
		return
	}
	defer file.Close()
	if header.Size == 0 {
		ui.renderGameProblem(w, entry, http.StatusBadRequest,
			"This file is empty, so it was not stored: it would show as a broken "+
				strings.ToLower(mediaLabels[m])+" everywhere.")
		return
	}

	ui.mu.Lock()
	defer ui.mu.Unlock()

	if _, stillThere := ui.reg.FindByID(system, id); !stillThere {
		renderGameNotFound(w, system, id)
		return
	}
	candidate := ui.reg.Clone()
	replaced, err := registry.WriteMedium(candidate, ui.registryFolder, system, id, m, header.Filename, file)
	switch {
	case errors.Is(err, registry.ErrGameNotFound):
		renderGameNotFound(w, system, id)
		return
	case errors.Is(err, registry.ErrUnsupportedMediaFile):
		ui.renderGameProblem(w, entry, http.StatusUnprocessableEntity,
			"This file is not one a "+strings.ToLower(mediaLabels[m])+" can be stored as. Accepted here: "+
				strings.Join(m.Extensions(), ", ")+".")
		return
	case err != nil:
		ui.renderGameProblem(w, entry, http.StatusInternalServerError,
			"This file could not be written into the registry, so nothing was stored: "+err.Error())
		return
	}

	saveErr := store.Save(candidate, ui.registryFolder)
	if saveErr != nil && !errors.Is(saveErr, store.ErrSiteNotRegenerated) {
		ui.renderGameProblem(w, entry, http.StatusInternalServerError,
			"The registry could not be written, so this medium was not stored.")
		return
	}
	ui.reg = candidate

	// From here the change is committed. Erasing the file the medium was stored
	// under before is what keeps a replaced medium from lingering unreferenced —
	// but a failure there is a caveat on a change that did happen, never a
	// refusal (decisions/031, following decisions/024).
	warnings := mediaWarnings(saveErr)
	if replaced != "" {
		if err := registry.RemoveMediumFile(ui.registryFolder, system, replaced); err != nil {
			warnings = append(warnings, warningMediaLeft)
		}
	}
	http.Redirect(w, r, mediaChangedURL(system, id, m, mediaStored, warnings), http.StatusSeeOther)
}

// serveMediumDeleteConfirmation renders the page asking to confirm one medium's
// deletion. Erasing is what the submission does; opening this page changes
// nothing.
func (ui *webUI) serveMediumDeleteConfirmation(w http.ResponseWriter, r *http.Request) {
	m, known := requestedMedium(w, r)
	if !known {
		return
	}
	system, id := r.PathValue("system"), r.PathValue("id")

	ui.mu.RLock()
	entry, found := ui.reg.FindByID(system, id)
	ui.mu.RUnlock()
	if !found {
		renderGameGone(w, system, id)
		return
	}

	// Revalidated rather than served from the browser's cache, so going back to
	// this page after the deletion does not offer to erase the medium again.
	w.Header().Set("Cache-Control", "no-cache")
	render(w, http.StatusOK, mediumDeleteTemplate, mediumDeleteForm(entry, system, id, m, ""))
}

// removeMedium erases one medium of a game and empties the reference to it,
// then redirects back to the game's page.
//
// Like a game's deletion, and unlike every other change, the served snapshot is
// swapped in as soon as the file is gone rather than once the whole write
// succeeded: the erasure is what a deletion means, so it is already persisted
// at that point (decisions/022).
func (ui *webUI) removeMedium(w http.ResponseWriter, r *http.Request) {
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
	m, known := requestedMedium(w, r)
	if !known {
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
	_, err := registry.ClearMedium(candidate, ui.registryFolder, system, id, m)
	switch {
	case errors.Is(err, registry.ErrMediaLeftBehind):
		warnings = append(warnings, warningMediaLeft)
	case err != nil:
		render(w, http.StatusInternalServerError, mediumDeleteTemplate,
			mediumDeleteForm(entry, system, id, m,
				"This file could not be erased from the registry, so the "+
					strings.ToLower(mediaLabels[m])+" is still there."))
		return
	}
	ui.reg = candidate

	warnings = append(warnings, mediaWarnings(store.Save(candidate, ui.registryFolder))...)
	http.Redirect(w, r, mediaChangedURL(system, id, m, mediaCleared, warnings), http.StatusSeeOther)
}

// mediaWarnings turns what a store.Save left unfinished into the caveats a
// media change carries. Only a stale consultation site can come from here: a
// registry that could not be written at all is the caller's to refuse.
func mediaWarnings(err error) []string {
	if err != nil {
		return []string{warningStaleSite}
	}
	return nil
}

// mediumDeleteForm builds the confirmation's view: the medium, the very file
// about to be erased, and where the two ways out lead.
func mediumDeleteForm(entry registry.Entry, system, id string, m registry.Medium, problem string) mediumDeleteConfirmation {
	return mediumDeleteConfirmation{
		Name:      entry.Game.Name,
		System:    system,
		SystemURL: systemURL(system),
		GameURL:   gameURL(system, id),
		Action:    mediumURL(system, id, m) + "/delete",
		Label:     mediaLabels[m],
		File:      referenceOf(entry, m),
		Problem:   problem,
	}
}

// mediumDeleteConfirmation is the page asking whether to really erase one
// medium: which one, what file that is, and the two ways out. File is empty
// when the entry refers to nothing — the page then says the medium is already
// gone rather than promising to erase a file it cannot name.
type mediumDeleteConfirmation struct {
	Name      string
	System    string
	SystemURL string
	GameURL   string
	Action    string
	Label     string
	File      string
	// Problem states why the medium was not erased, when the confirmation is
	// shown again after a failed attempt rather than for the first time.
	Problem string
}

// mediaChangedURL builds the game page URL confirming a media change: the
// outcome, the medium it concerned, and whatever it left unfinished.
func mediaChangedURL(system, id string, m registry.Medium, outcome string, warnings []string) string {
	query := url.Values{savedParam: {outcome}, mediumParam: {string(m)}}
	for _, warning := range warnings {
		query.Add(warningParam, warning)
	}
	return gameURL(system, id) + "?" + query.Encode() + savedAnchor
}

// mediaConfirmation turns what a media change redirected with into the sentence
// the game page confirms it with, or nothing at all when the query is not one
// of its own. The medium is always named: four of them sit on the page, and a
// confirmation that does not say which one changed confirms nothing.
func mediaConfirmation(query url.Values) string {
	m, known := registry.LookupMedium(query.Get(mediumParam))
	if !known {
		return ""
	}

	message := ""
	switch query.Get(savedParam) {
	case mediaStored:
		message = mediaLabels[m] + " saved to the registry."
	case mediaCleared:
		message = mediaLabels[m] + " removed from the registry."
	default:
		return ""
	}

	warnings := query[warningParam]
	if slices.Contains(warnings, warningMediaLeft) {
		message += " One file could not be erased and is still in the registry, referred to by nothing."
	}
	if slices.Contains(warnings, warningStaleSite) {
		message += staleCaveat
	}
	return message
}

// megabytes words a byte cap the way a user reads it.
func megabytes(bytes int64) string {
	return strconv.FormatInt(bytes/(1<<20), 10) + " MB"
}

// renderGameProblem shows a game's own page again, carrying the sentence saying
// why the change it was asked for did not happen. A refused upload has nothing
// to redirect to — nothing changed — and a dead-end error page would take the
// user away from the very controls they need to try again.
func (ui *webUI) renderGameProblem(w http.ResponseWriter, entry registry.Entry, status int, problem string) {
	detail := ui.gameDetail(entry)
	detail.Problem = problem
	render(w, status, gameTemplate, detail)
}

// mediumDeleteTemplate renders the confirmation asking whether to really erase
// one of a game's media: which one, the file it is, and the two ways out.
var mediumDeleteTemplate = newPage("medium-delete", `
{{define "title"}}Remove the {{.Label}} of {{.Name}} - Registry{{end}}
{{define "body"}}
<nav class="crumbs" aria-label="Breadcrumb">
<a href="/">Registry</a><span class="crumbs__sep">/</span><a href="{{.SystemURL}}">{{.System}}</a><span class="crumbs__sep">/</span><a href="{{.GameURL}}">{{.Name}}</a><span class="crumbs__sep">/</span><span class="crumbs__current">Remove the {{.Label}}</span>
</nav>
<main>
<article class="game">
<h2 class="game__title">Remove the {{.Label}} of {{.Name}}?</h2>
{{if .Problem}}
<div class="errors" role="alert" tabindex="-1" id="errors">
<p class="errors__title">Not removed</p>
<p>{{.Problem}}</p>
</div>
{{end}}
<div class="confirm">
{{if .File}}
<p class="confirm__lead">This erases the {{.Label}} of {{.Name}} from the registry of {{.System}}. This file will be deleted:</p>
<ul class="confirm__files">
<li><code>{{.System}}/{{.File}}</code></li>
</ul>
{{else}}
<p class="confirm__lead">{{.Name}} already has no {{.Label}} in the registry &mdash; there is nothing left to erase.</p>
{{end}}
<p class="confirm__note">Only this medium is touched. The ROMs folders are left alone, so a later import can bring it back from what Batocera holds.</p>
<p class="confirm__warning">This cannot be undone.</p>
<div class="form__actions">
<form method="post" action="{{.Action}}">
<button class="button button--danger" type="submit">Remove this {{.Label}}</button>
</form>
<a class="button button--quiet" href="{{.GameURL}}">Cancel</a>
</div>
</div>
</article>
</main>
{{end}}
`)
