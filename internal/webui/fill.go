package webui

import (
	"errors"
	"net/http"
	"net/url"
	"slices"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/store"
)

// fillURL is the form creating, in the registry, a game of a ROMs folder the
// registry does not know, from values typed by hand. It writes only the
// registry: the game reaches the ROMs folder later, through the usual send
// confirmation.
const fillURL = romsFolderStatusURL + "/fill"

// fillURLFor builds the URL of the fill-in form for rom of folder's system.
func fillURLFor(folder, system, rom string) string {
	query := url.Values{statusFolderParam: {folder}, gameSystemParam: {system}, gameROMParam: {rom}}
	return fillURL + "?" + query.Encode()
}

// fillPage is the fill-in form's view: which ROM of which folder it creates,
// the metadata controls, and the errors a refused submission must show.
type fillPage struct {
	Folder    string
	System    string
	ROM       string
	StatusURL string
	Fields    []formControl
	Errors    []formError
	// Problem states what went wrong when the values were fine but could not
	// be stored.
	Problem string
	// Lookup offers to import this game's values from a web page instead.
	Lookup lookupForm
}

// fillForm builds the fill-in form for rom, filled with values and the
// messages of the fields errs refused.
func fillForm(folder, system, rom string, values editValues, errs map[string]string) fillPage {
	fields, summary := metadataControls(values, errs, nil)
	return fillPage{
		Folder:    folder,
		System:    system,
		ROM:       rom,
		StatusURL: romsFolderSystemStatusURLFor(folder, system, 1),
		Fields:    fields,
		Errors:    summary,
		Lookup:    lookupForm{Action: fillFromURL, Hidden: romFields(folder, system, rom)},
	}
}

// fillableGame reports whether rom is, right now, a game of folder's system
// the registry does not know — the only kind the fill-in form may create.
// Recomputed from disk and the registry rather than trusted from the request.
func fillableGame(reg *registry.Registry, romsFolders []string, folder, system, rom string) bool {
	if !slices.Contains(romsFolders, folder) {
		return false
	}
	st, err := registry.RomsFolderSystemStatus(reg, folder, system)
	if err != nil {
		return false
	}
	for _, game := range st.Games {
		if game.ROMFilename == rom && game.RegistryID == "" {
			return true
		}
	}
	return false
}

// renderBadFill refuses a fill-in naming a folder, system or ROM that is not,
// right now, a game the registry does not know.
func renderBadFill(w http.ResponseWriter) {
	renderProblem(w, http.StatusBadRequest, "Bad request",
		"This request named a ROMs folder or a game this server does not offer to fill in — it may already be in the registry.")
}

// serveFillForm renders the empty fill-in form for one game unknown to the
// registry.
func (ui *webUI) serveFillForm(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	folder := query.Get(statusFolderParam)
	system := query.Get(gameSystemParam)
	rom := query.Get(gameROMParam)

	ui.mu.RLock()
	fillable := fillableGame(ui.reg, ui.romsFolders, folder, system, rom)
	ui.mu.RUnlock()
	if !fillable {
		renderBadFill(w)
		return
	}

	render(w, http.StatusOK, fillTemplate, fillForm(folder, system, rom, editValues{}, nil))
}

// fillGame creates one game in the registry from the values typed by hand,
// every one of them marked as corrected by hand so a later update leaves it
// alone, then redirects back to its system's status page, where it can now be
// sent to the folder. Nothing is committed to the served registry until the
// write to disk succeeded.
func (ui *webUI) fillGame(w http.ResponseWriter, r *http.Request) {
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
	system := r.PostFormValue(gameSystemParam)
	rom := r.PostFormValue(gameROMParam)

	ui.mu.Lock()
	defer ui.mu.Unlock()

	if !fillableGame(ui.reg, ui.romsFolders, folder, system, rom) {
		renderBadFill(w)
		return
	}

	values := submittedValues(r)
	if errs := validate(values, ""); len(errs) > 0 {
		render(w, http.StatusUnprocessableEntity, fillTemplate, fillForm(folder, system, rom, values, errs))
		return
	}

	romPath := "./" + rom
	id := registry.GameID(romPath)
	candidate := ui.reg.Clone()
	candidate.Import(system, []gamelist.Game{{Path: romPath}})
	if err := registry.UpdateMetadata(candidate, system, id, storedMetadata(values, gamelist.Game{}), nil); err != nil {
		renderBadFill(w)
		return
	}

	if err := store.Save(candidate, ui.registryFolder); err != nil && !errors.Is(err, store.ErrSiteNotRegenerated) {
		page := fillForm(folder, system, rom, values, nil)
		page.Problem = "The registry folder could not be written to, so this game was not created."
		render(w, http.StatusInternalServerError, fillTemplate, page)
		return
	}
	ui.reg = candidate

	http.Redirect(w, r, statusConfirmationURL(folder, system, values.Name, outcomeCreated), http.StatusSeeOther)
}

// fillTemplate renders the fill-in form: the ROM it creates a game for, and
// the same metadata controls the edit form offers.
var fillTemplate = newPage("fill", `
{{define "title"}}Fill in {{.ROM}} - Registry{{end}}
{{define "body"}}
<nav class="crumbs" aria-label="Breadcrumb">
<a href="/">Registry</a><span class="crumbs__sep">/</span><a href="{{.StatusURL}}">{{.System}}</a><span class="crumbs__sep">/</span><span class="crumbs__current">Fill in</span>
</nav>
<main>
<article class="game">
<h2 class="game__title">Fill in <code>{{.ROM}}</code></h2>
<p class="home__note">Creates this game in the registry with what you type. Nothing is written to <code>{{.Folder}}</code> until you send it there. Cover art and other media can be added afterwards from the game's page in the registry.</p>
{{template "lookup-form" .Lookup}}
{{if .Problem}}
<div class="errors" role="alert" tabindex="-1" id="errors">
<p class="errors__title">Not saved</p>
<p>{{.Problem}}</p>
</div>
{{end}}
{{if .Errors}}
<div class="errors" role="alert" tabindex="-1" id="errors">
<p class="errors__title">This was not saved:</p>
<ul>
{{range .Errors}}<li><a href="#field-{{.Key}}">{{.Label}}: {{.Message}}</a></li>
{{end}}
</ul>
</div>
{{end}}
<form class="form" method="post" action="`+fillURL+`">
<input type="hidden" name="`+statusFolderParam+`" value="{{.Folder}}">
<input type="hidden" name="`+gameSystemParam+`" value="{{.System}}">
<input type="hidden" name="`+gameROMParam+`" value="{{.ROM}}">
{{template "metadata-fields" .}}
<div class="form__actions">
<button class="button" type="submit">Create in the registry</button>
<a class="button button--quiet" href="{{.StatusURL}}">Cancel</a>
</div>
</form>
</article>
</main>
{{end}}
`+metadataFieldsTemplate+lookupFormTemplate)
