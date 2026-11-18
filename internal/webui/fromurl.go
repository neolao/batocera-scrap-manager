package webui

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/scrape"
	"github.com/neolao/batocera-scrap-manager/internal/site"
	"github.com/neolao/batocera-scrap-manager/internal/store"
)

// fromURLParam carries the address of the page to import from; takeParam
// names each value the user ticked on the review page — a field identifier
// (registry.Field*) or a medium name.
const (
	fromURLParam = "url"
	takeParam    = "take"
)

// fillFromURL reviews, then applies, an import creating a game the registry
// does not know yet; an existing game's own review lives under its page.
const fillFromURL = fillURL + "/from-url"

// supportedSitesNote names the sites a page may be imported from, for the
// lookup form and the refusal of any other address.
const supportedSitesNote = "Supported site: retrogaming.neolao.com (game pages, over https)."

// gameFromURL is the review and apply URL of an import into an existing game.
func gameFromURL(system, id string) string {
	return gameURL(system, id) + "/from-url"
}

// importedFields lists, in form order, the values a page may propose, with
// where each sits in a scrape.Result and in a game.
var importedFields = []struct {
	Key   string
	Label string
	from  func(scrape.Result) string
	in    func(*gamelist.Game) *string
}{
	{registry.FieldName, "Name", func(r scrape.Result) string { return r.Name }, func(g *gamelist.Game) *string { return &g.Name }},
	{registry.FieldDesc, "Description", func(r scrape.Result) string { return r.Desc }, func(g *gamelist.Game) *string { return &g.Desc }},
	{registry.FieldReleaseDate, "Release date", func(r scrape.Result) string { return r.ReleaseDate }, func(g *gamelist.Game) *string { return &g.ReleaseDate }},
	{registry.FieldPublisher, "Publisher", func(r scrape.Result) string { return r.Publisher }, func(g *gamelist.Game) *string { return &g.Publisher }},
	{registry.FieldGenre, "Genre", func(r scrape.Result) string { return r.Genre }, func(g *gamelist.Game) *string { return &g.Genre }},
}

// importedMedia lists the media a page may propose, with where each sits in a
// scrape.Result.
var importedMedia = []struct {
	Medium registry.Medium
	from   func(scrape.Result) string
}{
	{registry.MediumImage, func(r scrape.Result) string { return r.ImageURL }},
	{registry.MediumThumbnail, func(r scrape.Result) string { return r.ThumbnailURL }},
}

// lookupForm is the small form asking for a page address, shown on the edit
// and fill-in forms and at the top of the review page.
type lookupForm struct {
	Action string
	Hidden []hiddenField
	URL    string
}

// hiddenField is one value a form carries along unchanged.
type hiddenField struct {
	Name  string
	Value string
}

// fieldChoice is one value of the review: the game's current value, the
// page's, and whether it is offered ticked.
type fieldChoice struct {
	Key      string
	Label    string
	Current  string
	Proposed string
	// Same is a value the game already holds as is: shown, not offered.
	Same    bool
	Checked bool
}

// mediumChoice is one medium of the review: the game's current file, if any,
// and the page's image.
type mediumChoice struct {
	Key         string
	Label       string
	CurrentURL  string
	ProposedURL string
	Checked     bool
}

// fromURLPage is the review of an import: what the page proposes against
// what the game holds, one tick box per value.
type fromURLPage struct {
	Title   string
	BackURL string
	Lookup  lookupForm
	Action  string
	Hidden  []hiddenField
	URL     string
	Fields  []fieldChoice
	Media   []mediumChoice
	Problem string
}

// importTarget is the game an import is about: an existing entry, or a ROM of
// a ROMs folder the registry does not know yet.
type importTarget struct {
	system string
	// id is empty for a game still to be created.
	id      string
	current gamelist.Game
	// For a game still to be created: where its ROM sits.
	folder string
	rom    string
}

// reviewPage builds the review of result against t, ticking a value only where the
// game holds nothing yet.
func (ui *webUI) reviewPage(t importTarget, raw string, result scrape.Result) fromURLPage {
	page := ui.emptyReview(t, raw)
	for _, f := range importedFields {
		proposed := f.from(result)
		if proposed == "" {
			continue
		}
		current := *f.in(&t.current)
		page.Fields = append(page.Fields, fieldChoice{
			Key:      f.Key,
			Label:    f.Label,
			Current:  displayed(f.Key, current),
			Proposed: displayed(f.Key, proposed),
			Same:     current == proposed,
			Checked:  current == "",
		})
	}
	for _, m := range importedMedia {
		proposed := m.from(result)
		if proposed == "" {
			continue
		}
		choice := mediumChoice{Key: string(m.Medium), Label: mediaLabels[m.Medium], ProposedURL: proposed}
		ref := registry.Reference(&t.current, m.Medium)
		if ref != "" && site.MediaFileExists(ui.registryFolder, t.system, ref) {
			choice.CurrentURL = mediaURL(site.EscapeMediaPath(t.system, ref))
		}
		choice.Checked = choice.CurrentURL == ""
		page.Media = append(page.Media, choice)
	}
	return page
}

// emptyReview is the review page of t with no value yet: what a refused
// address or an unreachable site shows, with the lookup form to try again.
func (ui *webUI) emptyReview(t importTarget, raw string) fromURLPage {
	page := fromURLPage{URL: raw}
	if t.id != "" {
		page.Title = t.current.Name
		page.BackURL = gameURL(t.system, t.id) + "/edit"
		page.Action = gameFromURL(t.system, t.id)
	} else {
		page.Title = t.rom
		page.BackURL = fillURLFor(t.folder, t.system, t.rom)
		page.Action = fillFromURL
		page.Hidden = romFields(t.folder, t.system, t.rom)
	}
	page.Lookup = lookupForm{Action: page.Action, Hidden: page.Hidden, URL: raw}
	return page
}

// romFields are the hidden values naming a ROM of a ROMs folder.
func romFields(folder, system, rom string) []hiddenField {
	return []hiddenField{
		{statusFolderParam, folder},
		{retrieveSystemParam, system},
		{retrieveROMParam, rom},
	}
}

// displayed renders a stored value the way the review shows it: a release
// date as YYYY-MM-DD rather than the registry's timestamp.
func displayed(key, value string) string {
	if key != registry.FieldReleaseDate || len(value) < 8 {
		return value
	}
	return value[0:4] + "-" + value[4:6] + "-" + value[6:8]
}

// lookupProblem words a failed lookup or download for the user, with the
// status it answers.
func lookupProblem(err error) (int, string) {
	switch {
	case errors.Is(err, scrape.ErrUnsupportedSite):
		return http.StatusUnprocessableEntity, "This address is not one this tool can read. " + supportedSitesNote
	case errors.Is(err, scrape.ErrTooLarge):
		return http.StatusBadGateway, "The site answered with more than a game page or an image may weigh, so nothing was imported."
	case errors.Is(err, scrape.ErrNotAnImage):
		return http.StatusBadGateway, "An image the page points to is not an image, so nothing was imported."
	default:
		return http.StatusBadGateway, "The site could not be read — check the address, or try again later. Nothing was imported."
	}
}

// renderReview answers the review of t for the page at raw, looked up now.
func (ui *webUI) renderReview(w http.ResponseWriter, r *http.Request, t importTarget, raw string) {
	result, err := ui.scraper.Lookup(r.Context(), raw)
	if err != nil {
		status, message := lookupProblem(err)
		page := ui.emptyReview(t, raw)
		page.Problem = message
		render(w, status, fromURLTemplate, page)
		return
	}
	render(w, http.StatusOK, fromURLTemplate, ui.reviewPage(t, raw, result))
}

// serveGameFromURL reviews an import into an existing game.
func (ui *webUI) serveGameFromURL(w http.ResponseWriter, r *http.Request) {
	system, id := r.PathValue("system"), r.PathValue("id")
	ui.mu.RLock()
	entry, found := ui.reg.FindByID(system, id)
	ui.mu.RUnlock()
	if !found {
		renderGameNotFound(w, system, id)
		return
	}
	ui.renderReview(w, r, importTarget{system: system, id: id, current: entry.Game}, r.URL.Query().Get(fromURLParam))
}

// serveFillFromURL reviews an import creating a game the registry does not
// know yet.
func (ui *webUI) serveFillFromURL(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	folder := query.Get(statusFolderParam)
	system := query.Get(retrieveSystemParam)
	rom := query.Get(retrieveROMParam)

	ui.mu.RLock()
	fillable := fillableGame(ui.reg, ui.romsFolders, folder, system, rom)
	ui.mu.RUnlock()
	if !fillable {
		renderBadFill(w)
		return
	}
	ui.renderReview(w, r, importTarget{system: system, folder: folder, rom: rom}, query.Get(fromURLParam))
}

// download is one image fetched for an import, ready to store.
type download struct {
	medium   registry.Medium
	filename string
	data     []byte
}

// fetchImport reads the page again — the review's values are never trusted
// from the browser — and downloads every ticked medium, before any lock is
// taken: the network may take seconds every other page must not wait for.
func (ui *webUI) fetchImport(ctx context.Context, raw string, take []string) (scrape.Result, []download, error) {
	result, err := ui.scraper.Lookup(ctx, raw)
	if err != nil {
		return scrape.Result{}, nil, err
	}
	var downloads []download
	for _, m := range importedMedia {
		address := m.from(result)
		if address == "" || !slices.Contains(take, string(m.Medium)) {
			continue
		}
		data, filename, err := ui.scraper.Download(ctx, address)
		if err != nil {
			return scrape.Result{}, nil, err
		}
		downloads = append(downloads, download{medium: m.Medium, filename: filename, data: data})
	}
	return result, downloads, nil
}

// applyImport writes the ticked values of result and the downloaded media
// into candidate's entry system/id, every value marked as corrected by hand.
// It returns the media files a replaced medium left behind, for the caller
// to erase once the registry is written.
func (ui *webUI) applyImport(candidate *registry.Registry, system, id string, current gamelist.Game, result scrape.Result, take []string, downloads []download) ([]string, error) {
	updated := current
	for _, f := range importedFields {
		proposed := f.from(result)
		if proposed != "" && slices.Contains(take, f.Key) {
			*f.in(&updated) = proposed
		}
	}
	if err := registry.UpdateMetadata(candidate, system, id, metadataOf(updated), nil); err != nil {
		return nil, err
	}

	var replaced []string
	for _, d := range downloads {
		old, err := registry.WriteMedium(candidate, ui.registryFolder, system, id, d.medium, d.filename, bytes.NewReader(d.data))
		if err != nil {
			return nil, err
		}
		if old != "" {
			replaced = append(replaced, old)
		}
	}
	return replaced, nil
}

// metadataOf reads a game's editable metadata.
func metadataOf(g gamelist.Game) registry.Metadata {
	return registry.Metadata{
		Name:        g.Name,
		Desc:        g.Desc,
		Rating:      g.Rating,
		ReleaseDate: g.ReleaseDate,
		Developer:   g.Developer,
		Publisher:   g.Publisher,
		Genre:       g.Genre,
		Players:     g.Players,
	}
}

// readImportSubmission refuses a cross-site or unreadable submission and
// returns the page address and the ticked values. ok is false once the
// refusal has been rendered.
func readImportSubmission(w http.ResponseWriter, r *http.Request) (raw string, take []string, ok bool) {
	if crossSite(r) {
		renderProblem(w, http.StatusForbidden, "Refused",
			"This request did not come from the registry's own pages.")
		return "", nil, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		renderProblem(w, http.StatusBadRequest, "Bad request", "This request could not be read.")
		return "", nil, false
	}
	return r.PostFormValue(fromURLParam), r.PostForm[takeParam], true
}

// renderImportProblem shows the review of t again, without values, saying
// what went wrong.
func (ui *webUI) renderImportProblem(w http.ResponseWriter, t importTarget, raw string, status int, message string) {
	page := ui.emptyReview(t, raw)
	page.Problem = message
	render(w, status, fromURLTemplate, page)
}

// importGameFromURL applies the ticked values of a page to an existing game,
// then redirects to that game's page.
func (ui *webUI) importGameFromURL(w http.ResponseWriter, r *http.Request) {
	raw, take, ok := readImportSubmission(w, r)
	if !ok {
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
	target := importTarget{system: system, id: id, current: entry.Game}
	if len(take) == 0 {
		ui.renderImportProblem(w, target, raw, http.StatusUnprocessableEntity, "Tick at least one value to import.")
		return
	}

	result, downloads, err := ui.fetchImport(r.Context(), raw, take)
	if err != nil {
		status, message := lookupProblem(err)
		ui.renderImportProblem(w, target, raw, status, message)
		return
	}

	ui.mu.Lock()
	defer ui.mu.Unlock()
	entry, found = ui.reg.FindByID(system, id)
	if !found {
		renderGameNotFound(w, system, id)
		return
	}
	candidate := ui.reg.Clone()
	replaced, err := ui.applyImport(candidate, system, id, entry.Game, result, take, downloads)
	if err != nil {
		ui.renderImportProblem(w, target, raw, http.StatusUnprocessableEntity, importWriteProblem(err))
		return
	}
	siteErr, ok := ui.commitURLImport(w, target, raw, candidate, replaced)
	if !ok {
		return
	}
	outcome := savedImported
	if siteErr != nil {
		outcome += staleSuffix
	}
	http.Redirect(w, r, gameURL(system, id)+"?"+savedParam+"="+outcome+savedAnchor, http.StatusSeeOther)
}

// fillGameFromURL creates a game the registry does not know from the ticked
// values of a page, then redirects to its system's status page, where it can
// be sent to the folder.
func (ui *webUI) fillGameFromURL(w http.ResponseWriter, r *http.Request) {
	raw, take, ok := readImportSubmission(w, r)
	if !ok {
		return
	}
	folder := r.PostFormValue(statusFolderParam)
	system := r.PostFormValue(retrieveSystemParam)
	rom := r.PostFormValue(retrieveROMParam)
	target := importTarget{system: system, folder: folder, rom: rom}

	ui.mu.RLock()
	fillable := fillableGame(ui.reg, ui.romsFolders, folder, system, rom)
	ui.mu.RUnlock()
	if !fillable {
		renderBadFill(w)
		return
	}
	if !slices.Contains(take, registry.FieldName) {
		ui.renderImportProblem(w, target, raw, http.StatusUnprocessableEntity, "Tick the name: a game needs one to be created.")
		return
	}

	result, downloads, err := ui.fetchImport(r.Context(), raw, take)
	if err != nil {
		status, message := lookupProblem(err)
		ui.renderImportProblem(w, target, raw, status, message)
		return
	}
	if result.Name == "" {
		ui.renderImportProblem(w, target, raw, http.StatusUnprocessableEntity, "This page names no game, so nothing was created.")
		return
	}

	ui.mu.Lock()
	defer ui.mu.Unlock()
	if !fillableGame(ui.reg, ui.romsFolders, folder, system, rom) {
		renderBadFill(w)
		return
	}
	romPath := "./" + rom
	id := registry.GameID(romPath)
	candidate := ui.reg.Clone()
	candidate.Import(system, []gamelist.Game{{Path: romPath}})
	replaced, err := ui.applyImport(candidate, system, id, gamelist.Game{Path: romPath}, result, take, downloads)
	if err != nil {
		ui.renderImportProblem(w, target, raw, http.StatusUnprocessableEntity, importWriteProblem(err))
		return
	}
	if _, ok := ui.commitURLImport(w, target, raw, candidate, replaced); !ok {
		return
	}
	http.Redirect(w, r, retrievedURL(folder, system, result.Name, retrievedFilled), http.StatusSeeOther)
}

// commitURLImport saves candidate and makes it the served registry, then erases
// the media files the import replaced. ok is false once the failure has been
// rendered; siteErr reports a registry written without its site.
func (ui *webUI) commitURLImport(w http.ResponseWriter, t importTarget, raw string, candidate *registry.Registry, replaced []string) (siteErr error, ok bool) {
	saveErr := store.Save(candidate, ui.registryFolder)
	if saveErr != nil && !errors.Is(saveErr, store.ErrSiteNotRegenerated) {
		ui.renderImportProblem(w, t, raw, http.StatusInternalServerError,
			"The registry folder could not be written to, so nothing was imported.")
		return nil, false
	}
	ui.reg = candidate
	for _, old := range replaced {
		// A file left behind is harmless: the registry no longer references it
		// (decisions/031).
		_ = registry.RemoveMediumFile(ui.registryFolder, t.system, old)
	}
	return saveErr, true
}

// importWriteProblem words a failure to write an imported value or medium.
func importWriteProblem(err error) string {
	if errors.Is(err, registry.ErrUnsupportedMediaFile) {
		return "An image the page points to is in a format the registry does not store, so nothing was imported."
	}
	return "The import could not be written into the registry, so nothing was imported."
}

// lookupFormTemplate renders the form asking for a page address.
const lookupFormTemplate = `
{{define "lookup-form"}}
<form class="form lookup" method="get" action="{{.Action}}">
{{range .Hidden}}<input type="hidden" name="{{.Name}}" value="{{.Value}}">
{{end}}
<label class="field__label" for="field-from-url">Import from a URL</label>
<p class="field__hint" id="hint-from-url">` + supportedSitesNote + ` You choose what to keep before anything is written.</p>
<div class="lookup__row">
<input class="field__control" type="url" id="field-from-url" name="` + fromURLParam + `" value="{{.URL}}" required spellcheck="false" autocapitalize="none" aria-describedby="hint-from-url" placeholder="https://retrogaming.neolao.com/…">
<button class="button button--quiet" type="submit">Look up</button>
</div>
</form>
{{end}}
`

// fromURLTemplate renders the review of an import: per value, the current
// and the proposed one, ticked only where nothing would be overwritten.
var fromURLTemplate = newPage("from-url", `
{{define "title"}}Import into {{.Title}} - Registry{{end}}
{{define "body"}}
<nav class="crumbs" aria-label="Breadcrumb">
<a href="/">Registry</a><span class="crumbs__sep">/</span><a href="{{.BackURL}}">{{.Title}}</a><span class="crumbs__sep">/</span><span class="crumbs__current">Import from a URL</span>
</nav>
<main>
<article class="game">
<h2 class="game__title">Import into {{.Title}}</h2>
{{template "lookup-form" .Lookup}}
{{if .Problem}}
<div class="errors" role="alert" tabindex="-1" id="errors">
<p class="errors__title">Nothing imported</p>
<p>{{.Problem}}</p>
</div>
{{end}}
{{if or .Fields .Media}}
<form class="form" method="post" action="{{.Action}}">
{{range .Hidden}}<input type="hidden" name="{{.Name}}" value="{{.Value}}">
{{end}}
<input type="hidden" name="`+fromURLParam+`" value="{{.URL}}">
<fieldset class="form__fields">
<legend>Values found on the page</legend>
<p class="field__hint">Ticked values are written to the registry and protected from later updates. A value already set is only overwritten if you tick it.</p>
<table class="import__table">
<thead><tr><th scope="col">Take</th><th scope="col">Value</th><th scope="col">Now</th><th scope="col">On the page</th></tr></thead>
<tbody>
{{range .Fields}}
<tr>
<td>{{if .Same}}<span class="import__same">Same</span>{{else}}<input type="checkbox" id="take-{{.Key}}" name="`+takeParam+`" value="{{.Key}}"{{if .Checked}} checked{{end}}>{{end}}</td>
<th scope="row">{{if .Same}}{{.Label}}{{else}}<label for="take-{{.Key}}">{{.Label}}</label>{{end}}</th>
<td class="import__value">{{if .Current}}{{.Current}}{{else}}<span class="import__empty">empty</span>{{end}}</td>
<td class="import__value">{{.Proposed}}</td>
</tr>
{{end}}
{{range .Media}}
<tr>
<td><input type="checkbox" id="take-{{.Key}}" name="`+takeParam+`" value="{{.Key}}"{{if .Checked}} checked{{end}}></td>
<th scope="row"><label for="take-{{.Key}}">{{.Label}}</label></th>
<td>{{if .CurrentURL}}<img class="import__art" src="{{.CurrentURL}}" alt="Current {{.Label}}">{{else}}<span class="import__empty">none</span>{{end}}</td>
<td><img class="import__art" src="{{.ProposedURL}}" alt="{{.Label}} on the page" referrerpolicy="no-referrer"></td>
</tr>
{{end}}
</tbody>
</table>
</fieldset>
<div class="form__actions">
<button class="button" type="submit">Import the ticked values</button>
<a class="button button--quiet" href="{{.BackURL}}">Cancel</a>
</div>
</form>
{{end}}
</article>
</main>
{{end}}
`+lookupFormTemplate)
