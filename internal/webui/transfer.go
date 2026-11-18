package webui

import (
	"errors"
	"net/http"
	"net/url"
	"slices"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/site"
	"github.com/neolao/batocera-scrap-manager/internal/store"
)

// gameSystemParam and gameROMParam name, in a request about one game of a
// ROMs folder, its system and ROM filename. The folder travels under
// statusFolderParam.
const (
	gameSystemParam = "system"
	gameROMParam    = "rom"
)

// transferURL reviews, then applies, a copy of chosen values of one game
// between a ROMs folder's gamelist.xml and the registry, in the direction
// directionParam names.
const (
	transferURL         = romsFolderStatusURL + "/transfer"
	directionParam      = "direction"
	directionToRegistry = "to-registry"
	directionToFolder   = "to-folder"
)

// transferURLFor builds the review URL of a transfer of rom in direction.
func transferURLFor(folder, system, rom, direction string) string {
	query := url.Values{
		statusFolderParam: {folder},
		gameSystemParam:   {system},
		gameROMParam:      {rom},
		directionParam:    {direction},
	}
	return transferURL + "?" + query.Encode()
}

// statusOutcomeParam and statusNameParam carry the outcome of a change made
// from a system status page back to it, which renders the matching banner.
const (
	statusOutcomeParam = "changed"
	statusNameParam    = "name"
	statusAnchor       = "#changed"
)

// The outcomes a change made from a system status page may redirect with.
const (
	outcomeCreated     = "created"
	outcomeToRegistry  = "to-registry"
	outcomeToFolder    = "to-folder"
	outcomeFolderAdded = "to-folder-added"
	outcomeUnchanged   = "unchanged"
)

// statusConfirmations maps an outcome to the sentence completing the game's
// own name — a page listing many games at once must always say which one a
// banner is about.
var statusConfirmations = map[string]string{
	outcomeCreated:     " was created in the registry — copy it to this folder to write it there.",
	outcomeToRegistry:  ": the ticked values were copied into the registry.",
	outcomeToFolder:    ": the ticked values were written to this folder.",
	outcomeFolderAdded: ": a new entry holding the ticked values was added to this folder's gamelist.xml.",
	outcomeUnchanged:   ": the ticked values already matched — nothing changed.",
}

// statusConfirmation turns what a change redirected with into its banner, or
// nothing when the page was simply browsed to.
func statusConfirmation(query url.Values) string {
	name := query.Get(statusNameParam)
	if name == "" {
		return ""
	}
	suffix, known := statusConfirmations[query.Get(statusOutcomeParam)]
	if !known {
		return ""
	}
	return name + suffix
}

// statusConfirmationURL builds the system status page URL confirming a
// change to the game it names.
func statusConfirmationURL(folder, system, filter, name, outcome string) string {
	query := url.Values{
		statusFolderParam:  {folder},
		statusSystemParam:  {system},
		statusNameParam:    {name},
		statusOutcomeParam: {outcome},
	}
	if filter != "" {
		query.Set(filterParam, filter)
	}
	return romsFolderStatusURL + "?" + query.Encode() + statusAnchor
}

// transferRow is one value of a transfer review: what the destination holds,
// what the source would copy, and whether it is offered ticked.
type transferRow struct {
	Key         string
	Label       string
	Current     string
	Proposed    string
	CurrentURL  string
	ProposedURL string
	// Same is a value the destination already holds as is: shown, not
	// offered.
	Same    bool
	Checked bool
}

// transferPage is the review of a transfer.
type transferPage struct {
	Name      string
	ROM       string
	Folder    string
	System    string
	Direction string
	Filter    string
	From      string
	To        string
	StatusURL string
	Rows      []transferRow
	Problem   string
}

// transferRequest is one transfer's coordinates, read from a request.
type transferRequest struct {
	folder, system, rom, direction string
	// filter is the system status page's filter the review was opened from,
	// carried along to land back on the same list.
	filter string
}

// transferSides reads the source and destination of t as they are now. ok is
// false when the request does not name a game of a configured folder that
// has a source to copy from in that direction: a local entry to copy into
// the registry, a registry entry to copy into the folder.
func (ui *webUI) transferSides(t transferRequest) (source, destination gamelist.Game, ok bool) {
	if !slices.Contains(ui.romsFolders, t.folder) {
		return gamelist.Game{}, gamelist.Game{}, false
	}
	st, err := registry.RomsFolderSystemStatus(ui.reg, t.folder, t.system)
	if err != nil {
		return gamelist.Game{}, gamelist.Game{}, false
	}
	var game *registry.GameStatus
	for i := range st.Games {
		if st.Games[i].ROMFilename == t.rom {
			game = &st.Games[i]
		}
	}
	if game == nil {
		return gamelist.Game{}, gamelist.Game{}, false
	}

	local, localErr := registry.LocalGame(t.folder, t.system, t.rom)
	entry, inRegistry := ui.reg.FindByID(t.system, game.RegistryID)
	if game.RegistryID == "" {
		inRegistry = false
	}
	switch t.direction {
	case directionToRegistry:
		if localErr != nil {
			return gamelist.Game{}, gamelist.Game{}, false
		}
		return local, entry.Game, true
	case directionToFolder:
		if !inRegistry {
			return gamelist.Game{}, gamelist.Game{}, false
		}
		return entry.Game, local, true
	}
	return gamelist.Game{}, gamelist.Game{}, false
}

// transferForm builds the review of t, from source to destination.
func (ui *webUI) transferForm(t transferRequest, source, destination gamelist.Game) transferPage {
	page := transferPage{
		Name:      source.Name,
		ROM:       t.rom,
		Folder:    t.folder,
		System:    t.system,
		Direction: t.direction,
		Filter:    t.filter,
		From:      "This ROMs folder",
		To:        "Registry",
		StatusURL: filteredStatusURL(t.folder, t.system, t.filter, 1),
	}
	if page.Name == "" {
		page.Name = t.rom
	}
	if t.direction == directionToFolder {
		page.From, page.To = "Registry", "This ROMs folder"
	}

	for _, id := range registry.TransferFields() {
		proposed := registry.FieldValue(source, id)
		if proposed == "" {
			continue
		}
		current := registry.FieldValue(destination, id)
		row := transferRow{
			Key:      id,
			Label:    fieldLabel(id),
			Current:  transferDisplayed(id, current),
			Proposed: transferDisplayed(id, proposed),
			Same:     current == proposed,
			Checked:  current == "",
		}
		if m, isMedium := registry.LookupMedium(id); isMedium && m != registry.MediumVideo {
			localURL := localImageURL(t.folder, t.system, registry.FieldValue(sideOf(t, true, source, destination), id))
			registryURL := ui.registryImageURL(t.system, registry.FieldValue(sideOf(t, false, source, destination), id))
			if t.direction == directionToRegistry {
				row.ProposedURL, row.CurrentURL = localURL, registryURL
			} else {
				row.ProposedURL, row.CurrentURL = registryURL, localURL
			}
			// The same reference may name two different files, one in each
			// folder: a medium is never "the same" on the reference alone.
			row.Same = false
			row.Checked = row.CurrentURL == ""
		}
		page.Rows = append(page.Rows, row)
	}
	return page
}

// sideOf returns, of source and destination, the folder's game when local is
// true and the registry's otherwise.
func sideOf(t transferRequest, local bool, source, destination gamelist.Game) gamelist.Game {
	sourceIsLocal := t.direction == directionToRegistry
	if local == sourceIsLocal {
		return source
	}
	return destination
}

// registryImageURL serves a registry media reference of system, or nothing
// when it names no file.
func (ui *webUI) registryImageURL(system, ref string) string {
	if ref == "" || !site.MediaFileExists(ui.registryFolder, system, ref) {
		return ""
	}
	return mediaURL(site.EscapeMediaPath(system, ref))
}

// transferDisplayed renders a stored value the way the review shows it.
func transferDisplayed(id, value string) string {
	if value == "" {
		return ""
	}
	switch id {
	case registry.FieldReleaseDate:
		return displayed(id, value)
	case registry.FieldRating:
		return site.FormatStars(value)
	}
	return value
}

// readTransferRequest reads a transfer's coordinates from a query or a form.
func readTransferRequest(values url.Values) transferRequest {
	return transferRequest{
		folder:    values.Get(statusFolderParam),
		system:    values.Get(gameSystemParam),
		rom:       values.Get(gameROMParam),
		direction: values.Get(directionParam),
		filter:    requestedFilter(values),
	}
}

// renderBadTransfer refuses a transfer naming no game with a source to copy
// from in that direction.
func renderBadTransfer(w http.ResponseWriter) {
	renderProblem(w, http.StatusBadRequest, "Bad request",
		"This request named a ROMs folder, a game or a direction this server cannot copy from.")
}

// serveTransfer renders the review of a transfer. Opening it changes nothing.
func (ui *webUI) serveTransfer(w http.ResponseWriter, r *http.Request) {
	t := readTransferRequest(r.URL.Query())
	ui.mu.RLock()
	source, destination, ok := ui.transferSides(t)
	var page transferPage
	if ok {
		page = ui.transferForm(t, source, destination)
	}
	ui.mu.RUnlock()
	if !ok {
		renderBadTransfer(w)
		return
	}
	render(w, http.StatusOK, transferTemplate, page)
}

// transfer copies the ticked values, then redirects back to the system
// status page with a banner naming the outcome. Into the registry, nothing
// is committed to the served snapshot until the write to disk succeeded.
func (ui *webUI) transfer(w http.ResponseWriter, r *http.Request) {
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
	t := readTransferRequest(r.PostForm)
	take := r.PostForm[takeParam]

	ui.mu.Lock()
	defer ui.mu.Unlock()

	source, destination, ok := ui.transferSides(t)
	if !ok {
		renderBadTransfer(w)
		return
	}
	page := ui.transferForm(t, source, destination)
	if len(take) == 0 {
		page.Problem = "Tick at least one value to copy."
		render(w, http.StatusUnprocessableEntity, transferTemplate, page)
		return
	}

	outcome, status, problem := ui.applyTransfer(t, take)
	if problem != "" {
		page.Problem = problem
		render(w, status, transferTemplate, page)
		return
	}
	http.Redirect(w, r, statusConfirmationURL(t.folder, t.system, t.filter, page.Name, outcome), http.StatusSeeOther)
}

// applyTransfer copies the ticked values of t, returning the outcome to
// confirm, or the status and sentence of a failure that wrote nothing worth
// keeping. Called under ui.mu's write lock.
func (ui *webUI) applyTransfer(t transferRequest, take []string) (outcome string, status int, problem string) {
	if t.direction == directionToFolder {
		changed, added, err := registry.SendFields(ui.reg, t.folder, ui.registryFolder, t.system, t.rom, take)
		switch {
		case errors.Is(err, registry.ErrMediaNotCopied):
			return "", http.StatusInternalServerError, "The values were written, but a medium could not be copied into this folder."
		case err != nil:
			return "", http.StatusInternalServerError, "This folder's gamelist.xml could not be written, so nothing was copied."
		case added:
			return outcomeFolderAdded, 0, ""
		case changed:
			return outcomeToFolder, 0, ""
		}
		return outcomeUnchanged, 0, ""
	}

	candidate := ui.reg.Clone()
	if _, err := registry.RetrieveFields(candidate, t.folder, ui.registryFolder, t.system, t.rom, take); err != nil {
		return "", http.StatusInternalServerError, "A value or a medium could not be copied into the registry, so nothing was saved."
	}
	saveErr := store.Save(candidate, ui.registryFolder)
	if saveErr != nil && !errors.Is(saveErr, store.ErrSiteNotRegenerated) {
		return "", http.StatusInternalServerError, "The registry folder could not be written to, so nothing was copied."
	}
	ui.reg = candidate
	return outcomeToRegistry, 0, ""
}

// transferTemplate renders the review of a transfer: one row per value the
// source holds, the destination's value beside it, ticked only where nothing
// would be overwritten.
var transferTemplate = newPage("transfer", `
{{define "title"}}Copy {{.Name}}: {{.From}} → {{.To}} - Registry{{end}}
{{define "body"}}
<nav class="crumbs" aria-label="Breadcrumb">
<a href="/">Registry</a><span class="crumbs__sep">/</span><a href="{{.StatusURL}}">{{.System}}</a><span class="crumbs__sep">/</span><span class="crumbs__current">{{.From}} → {{.To}}</span>
</nav>
<main>
<article class="game">
<h2 class="game__title">Copy {{.Name}}: {{.From}} → {{.To}}</h2>
<p class="home__note"><code>{{.ROM}}</code> in <code>{{.Folder}}</code>. Only the ticked values are copied; a value already set is overwritten only if you tick it.</p>
{{if .Problem}}
<div class="errors" role="alert" tabindex="-1" id="errors">
<p class="errors__title">Not copied</p>
<p>{{.Problem}}</p>
</div>
{{end}}
{{if not .Rows}}
<p class="empty-state">{{.From}} holds nothing for this game.</p>
{{else}}
<form class="form" method="post" action="`+transferURL+`">
<input type="hidden" name="`+statusFolderParam+`" value="{{.Folder}}">
<input type="hidden" name="`+gameSystemParam+`" value="{{.System}}">
<input type="hidden" name="`+gameROMParam+`" value="{{.ROM}}">
<input type="hidden" name="`+directionParam+`" value="{{.Direction}}">
{{if .Filter}}<input type="hidden" name="`+filterParam+`" value="{{.Filter}}">{{end}}
<table class="import__table">
<thead><tr><th scope="col">Copy</th><th scope="col">Value</th><th scope="col">{{.To}} now</th><th scope="col">{{.From}}</th></tr></thead>
<tbody>
{{range .Rows}}
<tr>
<td>{{if .Same}}<span class="import__same">Same</span>{{else}}<input type="checkbox" id="take-{{.Key}}" name="`+takeParam+`" value="{{.Key}}"{{if .Checked}} checked{{end}}>{{end}}</td>
<th scope="row">{{if .Same}}{{.Label}}{{else}}<label for="take-{{.Key}}">{{.Label}}</label>{{end}}</th>
<td class="import__value">{{if .CurrentURL}}<img class="import__art" src="{{.CurrentURL}}" alt="">{{end}}{{if .Current}}{{.Current}}{{else}}<span class="import__empty">empty</span>{{end}}</td>
<td class="import__value">{{if .ProposedURL}}<img class="import__art" src="{{.ProposedURL}}" alt="">{{end}}{{.Proposed}}</td>
</tr>
{{end}}
</tbody>
</table>
<div class="form__actions">
<button class="button" type="submit">Copy the ticked values to {{.To}}</button>
<a class="button button--quiet" href="{{.StatusURL}}">Cancel</a>
</div>
</form>
{{end}}
</article>
</main>
{{end}}
`)
