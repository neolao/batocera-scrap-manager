package webui

import (
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"syscall"

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
	// NothingToCopy is a review where every value the source holds is already
	// in the destination: shown as such, with no form.
	NothingToCopy bool
	// Problem is the sentence saying what went wrong, and Details the lines
	// saying, one per file or step, exactly what to fix.
	Problem string
	Details []string
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

	srcRoot, dstRoot := t.folder, ui.registryFolder
	if t.direction == directionToFolder {
		srcRoot, dstRoot = ui.registryFolder, t.folder
	}
	differ := registry.Differences(source, destination, srcRoot, dstRoot, t.system)
	page.NothingToCopy = len(differ) == 0

	for _, id := range registry.TransferFields() {
		proposed := registry.FieldValue(source, id)
		if proposed == "" {
			continue
		}
		current := registry.FieldValue(destination, id)
		same := !slices.Contains(differ, id)
		row := transferRow{
			Key:      id,
			Label:    fieldLabel(id),
			Current:  transferDisplayed(id, current),
			Proposed: transferDisplayed(id, proposed),
			Same:     same,
			Checked:  !same && current == "",
		}
		if m, isMedium := registry.LookupMedium(id); isMedium && m != registry.MediumVideo {
			localURL := localImageURL(t.folder, t.system, registry.FieldValue(sideOf(t, true, source, destination), id))
			registryURL := ui.registryImageURL(t.system, registry.FieldValue(sideOf(t, false, source, destination), id))
			if t.direction == directionToRegistry {
				row.ProposedURL, row.CurrentURL = localURL, registryURL
			} else {
				row.ProposedURL, row.CurrentURL = registryURL, localURL
			}
			row.Checked = !same && row.CurrentURL == ""
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

	outcome, failure := ui.applyTransfer(t, take)
	if failure != nil {
		page.Problem, page.Details = failure.summary, failure.details
		render(w, failure.status, transferTemplate, page)
		return
	}
	http.Redirect(w, r, statusConfirmationURL(t.folder, t.system, t.filter, page.Name, outcome), http.StatusSeeOther)
}

// transferFailure is a transfer that did not complete: the status to answer
// with, the sentence saying what went wrong, and the lines saying what to
// fix.
type transferFailure struct {
	status  int
	summary string
	details []string
}

// applyTransfer copies the ticked values of t, returning the outcome to
// confirm, or the failure that stopped it. Called under ui.mu's write lock.
func (ui *webUI) applyTransfer(t transferRequest, take []string) (outcome string, failure *transferFailure) {
	if t.direction == directionToFolder {
		changed, added, err := registry.SendFields(ui.reg, t.folder, ui.registryFolder, t.system, t.rom, take)
		var notCopied *registry.MediaNotCopiedError
		switch {
		case errors.As(err, &notCopied):
			return "", &transferFailure{
				status:  http.StatusInternalServerError,
				summary: "Some media could not be copied into this folder. " + writtenSentence(notCopied.Copied, changed, "Nothing else was written."),
				details: mediaFailureLines(notCopied.Failures),
			}
		case err != nil:
			return "", &transferFailure{
				status:  http.StatusInternalServerError,
				summary: "This folder's gamelist.xml could not be written, so nothing was copied.",
				details: []string{"Could not write " + filepath.Join(t.folder, t.system, "gamelist.xml") + ": " + describeCauseWith(err, "write")},
			}
		case added:
			return outcomeFolderAdded, nil
		case changed:
			return outcomeToFolder, nil
		}
		return outcomeUnchanged, nil
	}

	candidate := ui.reg.Clone()
	if _, err := registry.RetrieveFields(candidate, t.folder, ui.registryFolder, t.system, t.rom, take); err != nil {
		var notCopied *registry.MediaNotCopiedError
		if errors.As(err, &notCopied) {
			return "", &transferFailure{
				status:  http.StatusInternalServerError,
				summary: "Some media could not be copied into the registry. Nothing was saved.",
				details: mediaFailureLines(notCopied.Failures),
			}
		}
		return "", &transferFailure{
			status:  http.StatusInternalServerError,
			summary: "A value could not be copied into the registry. Nothing was saved.",
			details: []string{describeCause(err)},
		}
	}
	saveErr := store.Save(candidate, ui.registryFolder)
	if saveErr != nil && !errors.Is(saveErr, store.ErrSiteNotRegenerated) {
		return "", &transferFailure{
			status:  http.StatusInternalServerError,
			summary: "The registry folder could not be written to, so nothing was copied.",
			details: []string{"Could not write in " + ui.registryFolder + ": " + describeCauseWith(saveErr, "write")},
		}
	}
	ui.reg = candidate
	return outcomeToRegistry, nil
}

// writtenSentence says which values a partly failed transfer did write, or
// none says what to report when it wrote nothing at all.
func writtenSentence(copied []string, wrote bool, none string) string {
	if !wrote || len(copied) == 0 {
		return none
	}
	return "Written: " + fieldLabels(copied) + "."
}

// mediaFailureLines words each medium that could not be copied: which one,
// which file, at what step, and why.
func mediaFailureLines(failures []registry.MediaCopyError) []string {
	lines := make([]string, len(failures))
	for i, f := range failures {
		action, access := "write", "write"
		switch f.Step {
		case registry.StepRead:
			action, access = "read", "read"
		case registry.StepCreateFolder:
			action = "create the folder of"
		}
		lines[i] = mediaLabels[f.Medium] + ": could not " + action + " " + causePath(f.Err) + " — " + describeCauseWith(f.Err, access)
	}
	return lines
}

// causePath returns the file or folder an operating system error names, or
// nothing.
func causePath(err error) string {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Path
	}
	return "the file"
}

// describeCause words an error for the user: why it happened in plain words,
// and what to do about it — falling back to the error's own text for a cause
// it has no advice for.
func describeCause(err error) string {
	return describeCauseWith(err, "the needed")
}

// describeCauseWith is describeCause where a permission error names the
// access (read or write) the user running the tool is missing.
func describeCauseWith(err error, access string) string {
	switch {
	case errors.Is(err, fs.ErrPermission):
		return "permission denied. Give the user running this tool " + access + " access to it."
	case errors.Is(err, syscall.ENOSPC):
		return "no space left on the disk. Free some space there."
	case errors.Is(err, syscall.EROFS):
		return "the disk is read-only. Mount it with write access."
	case errors.Is(err, syscall.ENOTDIR):
		return "a name on the way is a file, not a folder. Rename or remove that file."
	case errors.Is(err, syscall.ENAMETOOLONG):
		return "the file name is too long for that disk. Shorten the name."
	case errors.Is(err, fs.ErrNotExist):
		return "it does not exist. It may have been moved since this page was opened."
	}
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err.Error() + "."
	}
	return err.Error() + "."
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
{{if .Details}}<ul>
{{range .Details}}<li>{{.}}</li>
{{end}}</ul>{{end}}
</div>
{{end}}
{{if not .Rows}}
<p class="empty-state">{{.From}} holds nothing for this game.</p>
{{else if .NothingToCopy}}
<p class="empty-state">Both sides already hold the same values — nothing to copy. What this game still misses is missing from {{.From}} too.</p>
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
