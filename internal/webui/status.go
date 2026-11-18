package webui

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/site"
)

// romsFolderStatusURL reports what one configured ROMs folder is still
// missing, compared to what a Completion could already fill from the
// registry: a summary per system, or one system's paginated games when the
// query names it. It is read-only: opening it never writes anything.
const romsFolderStatusURL = "/roms-folder"

// statusFolderParam names, in the query, which configured ROMs folder the
// report is about.
const statusFolderParam = "folder"

// romsFolderStatusURLFor builds the URL of one configured folder's status
// page.
func romsFolderStatusURLFor(folder string) string {
	return romsFolderStatusURL + "?" + url.Values{statusFolderParam: {folder}}.Encode()
}

// romsFolderLink is one entry of the home page's list of configured ROMs
// folders, each leading to its own status page.
type romsFolderLink struct {
	Folder string
	URL    string
}

// romsFolderLinksOf builds the home page's list of ROMs folders, in the
// configured order.
func romsFolderLinksOf(romsFolders []string) []romsFolderLink {
	links := make([]romsFolderLink, len(romsFolders))
	for i, folder := range romsFolders {
		links[i] = romsFolderLink{Folder: folder, URL: romsFolderStatusURLFor(folder)}
	}
	return links
}

// statusSystemParam names, in the query, which system of the folder a
// system status page is about. Its absence means the folder's summary.
const statusSystemParam = "system"

// romsFolderSystemStatusURLFor builds the URL of one page of one system's
// status within a configured folder. The first page is addressed without a
// page parameter, so a system keeps one canonical URL.
func romsFolderSystemStatusURLFor(folder, system string, page int) string {
	query := url.Values{statusFolderParam: {folder}, statusSystemParam: {system}}
	if page > 1 {
		query.Set(pageParam, strconv.Itoa(page))
	}
	return romsFolderStatusURL + "?" + query.Encode()
}

// romsFolderStatusView is the folder's summary page: which folder it is
// about, and one row per system leading to that system's own page.
type romsFolderStatusView struct {
	Folder  string
	Systems []systemSummaryView
}

// systemSummaryView is one system's row of the summary.
type systemSummaryView struct {
	Name            string
	URL             string
	ROMCount        int
	CompleteCount   int
	IncompleteCount int
	NotListedCount  int
	FillableCount   int
	Problem         string
}

// systemStatusPage is one page of one system's games within a folder, with
// the navigation to the folder's other systems and the pager.
type systemStatusPage struct {
	Folder     string
	FolderURL  string
	Name       string
	Systems    []folderSystemLink
	Changed    string
	Problem    string
	ROMCount   int
	Complete   int
	Incomplete int
	NotListed  int
	Fillable   int
	Games      []gameStatusView
	Page       int
	PageCount  int
	PrevURL    string
	NextURL    string
	First      int
	Last       int
	Total      int
}

// folderSystemLink is one entry of the navigation between a folder's systems.
type folderSystemLink struct {
	Name string
	URL  string
}

// gameStatusView is one game of a system page, its missing and fillable
// fields already turned into the labels a user knows them by — the registry
// only ever names them by their internal identifiers.
type gameStatusView struct {
	// System is this game's Batocera system — every other identifier here is
	// only unique within it.
	System      string
	ROMFilename string
	Name        string
	Listed      bool
	Complete    bool
	Missing     string
	Fillable    string
	// RegistryURL leads to this game's own page in the registry. Empty when
	// the registry does not know this game at all — nothing to link to.
	RegistryURL string
	// ToRegistryURL and ToFolderURL open the review copying chosen values
	// one way or the other. Each is empty when its source holds nothing: no
	// local entry, or no registry entry.
	ToRegistryURL string
	ToFolderURL   string
	// FillURL leads to the form creating this game in the registry by hand.
	// Empty exactly when RegistryURL is not: only a game the registry does
	// not know can be filled in.
	FillURL string
	// LocalImageURL and RegistryImageURL show the cover art the ROMs folder
	// and the registry each hold, side by side; empty when that file is
	// absent.
	LocalImageURL    string
	RegistryImageURL string
}

// serveRomsFolderStatus renders, read-only, either the summary of one
// configured ROMs folder or, when the query names a system, one page of that
// system's games. Nothing is written by opening either.
func (ui *webUI) serveRomsFolderStatus(w http.ResponseWriter, r *http.Request) {
	folder := r.URL.Query().Get(statusFolderParam)
	if !slices.Contains(ui.romsFolders, folder) {
		renderProblem(w, http.StatusBadRequest, "Bad request",
			"This request named a ROMs folder this server does not offer.")
		return
	}

	ui.mu.RLock()
	reg := ui.reg
	ui.mu.RUnlock()

	if r.URL.Query().Has(statusSystemParam) {
		ui.serveRomsFolderSystemStatus(w, r, reg, folder)
		return
	}

	status, err := registry.RomsFolderStatus(reg, folder)
	if err != nil {
		renderProblem(w, http.StatusInternalServerError, "Could not read this folder",
			"This ROMs folder could not be read.")
		return
	}
	render(w, http.StatusOK, romsFolderStatusTemplate, romsFolderStatusViewOf(status))
}

// romsFolderStatusViewOf turns a registry.FolderStatus into the summary rows,
// keeping its order (most problems first).
func romsFolderStatusViewOf(status registry.FolderStatus) romsFolderStatusView {
	view := romsFolderStatusView{Folder: status.RomsFolder}
	for _, sys := range status.Systems {
		view.Systems = append(view.Systems, systemSummaryView{
			Name:            sys.System,
			URL:             romsFolderSystemStatusURLFor(status.RomsFolder, sys.System, 1),
			ROMCount:        sys.ROMCount,
			CompleteCount:   sys.CompleteCount,
			IncompleteCount: sys.IncompleteCount,
			NotListedCount:  sys.NotListedCount,
			FillableCount:   sys.FillableCount,
			Problem:         sys.Problem,
		})
	}
	return view
}

// serveRomsFolderSystemStatus renders one page of one system's games. Only
// that system is read; an unknown system and a page out of bounds both answer
// the not-found page, as the registry's own system pages do.
func (ui *webUI) serveRomsFolderSystemStatus(w http.ResponseWriter, r *http.Request, reg *registry.Registry, folder string) {
	query := r.URL.Query()
	system := query.Get(statusSystemParam)

	st, err := registry.RomsFolderSystemStatus(reg, folder, system)
	if errors.Is(err, os.ErrNotExist) {
		renderProblem(w, http.StatusNotFound, "Not found", "No system named "+system+" in this ROMs folder.")
		return
	}
	if err != nil {
		renderProblem(w, http.StatusInternalServerError, "Could not read this folder",
			"This ROMs folder could not be read.")
		return
	}

	pageCount := max(1, (len(st.Games)+gamesPerPage-1)/gamesPerPage)
	page, ok := requestedPage(query, pageCount)
	if !ok {
		renderProblem(w, http.StatusNotFound, "Not found",
			"There is no page "+query.Get(pageParam)+" in system "+system+".")
		return
	}
	first := (page - 1) * gamesPerPage
	last := min(first+gamesPerPage, len(st.Games))

	systems, err := registry.RomsFolderSystems(folder)
	if err != nil {
		renderProblem(w, http.StatusInternalServerError, "Could not read this folder",
			"This ROMs folder could not be read.")
		return
	}
	links := make([]folderSystemLink, len(systems))
	for i, name := range systems {
		links[i] = folderSystemLink{Name: name, URL: romsFolderSystemStatusURLFor(folder, name, 1)}
	}

	view := systemStatusPage{
		Folder:     folder,
		FolderURL:  romsFolderStatusURLFor(folder),
		Name:       system,
		Systems:    links,
		Changed:    statusConfirmation(query),
		Problem:    st.Problem,
		ROMCount:   st.ROMCount,
		Complete:   st.CompleteCount,
		Incomplete: st.IncompleteCount,
		NotListed:  st.NotListedCount,
		Fillable:   st.FillableCount,
		Games:      ui.gameStatusViews(folder, system, st.Games[first:last]),
		Page:       page,
		PageCount:  pageCount,
		PrevURL:    statusPageURL(folder, system, page-1, pageCount),
		NextURL:    statusPageURL(folder, system, page+1, pageCount),
		First:      first + 1,
		Last:       last,
		Total:      len(st.Games),
	}
	render(w, http.StatusOK, romsFolderSystemStatusTemplate, view)
}

// statusPageURL builds the URL of one page of a system's status, or an empty
// string when page falls outside [1, pageCount] — how the pager knows the
// first page has no previous link and the last no next one.
func statusPageURL(folder, system string, page, pageCount int) string {
	if page < 1 || page > pageCount {
		return ""
	}
	return romsFolderSystemStatusURLFor(folder, system, page)
}

// gameStatusViews turns one page of a system's games into what the template
// renders. Only this page's cover art files are looked up on disk.
func (ui *webUI) gameStatusViews(folder, system string, games []registry.GameStatus) []gameStatusView {
	views := make([]gameStatusView, len(games))
	for i, game := range games {
		views[i] = gameStatusView{
			System:        system,
			ROMFilename:   game.ROMFilename,
			Name:          game.Name,
			Listed:        game.Listed,
			Complete:      game.Complete(),
			Missing:       fieldLabels(game.Missing),
			Fillable:      fieldLabels(game.Fillable),
			RegistryURL:   registryURLFor(system, game.RegistryID),
			LocalImageURL: localImageURL(folder, system, game.LocalImage),
		}
		if game.RegistryID == "" {
			views[i].FillURL = fillURLFor(folder, system, game.ROMFilename)
		} else {
			views[i].ToFolderURL = transferURLFor(folder, system, game.ROMFilename, directionToFolder)
		}
		if game.Listed {
			views[i].ToRegistryURL = transferURLFor(folder, system, game.ROMFilename, directionToRegistry)
		}
		if game.RegistryImage != "" && site.MediaFileExists(ui.registryFolder, system, game.RegistryImage) {
			views[i].RegistryImageURL = mediaURL(site.EscapeMediaPath(system, game.RegistryImage))
		}
	}
	return views
}

// localImageURL builds the URL serving a cover art file of folder's system,
// or nothing when the reference is empty, is not relative to the system's
// folder, or names no file on disk.
func localImageURL(folder, system, relPath string) string {
	if relPath == "" || path.IsAbs(relPath) {
		return ""
	}
	if !site.MediaFileExists(folder, system, relPath) {
		return ""
	}
	return romsFolderMediaURL(folder, system, relPath)
}

// romsFolderMediaPrefix is the URL space serving a ROMs folder's own cover
// art, as /roms-folder/media/{system}/{path}?folder=F.
const romsFolderMediaPrefix = romsFolderStatusURL + "/media/"

// romsFolderMediaExtensions are the only files the ROMs folder media route
// serves: cover art, never a ROM or a gamelist.xml.
var romsFolderMediaExtensions = []string{".png", ".jpg", ".jpeg", ".gif", ".webp"}

// romsFolderMediaURL builds the URL serving relPath, relative to folder's
// system folder as gamelist.xml writes it ("./images/x.png").
func romsFolderMediaURL(folder, system, relPath string) string {
	escaped := path.Clean(romsFolderMediaPrefix + site.EscapeMediaPath(system, relPath))
	return escaped + "?" + url.Values{statusFolderParam: {folder}}.Encode()
}

// serveRomsFolderMedium serves one cover art file of a configured ROMs
// folder's system. The file is opened through an os.Root confined to the
// system's folder, so no path can reach outside it, and only image
// extensions are served.
func (ui *webUI) serveRomsFolderMedium(w http.ResponseWriter, r *http.Request) {
	folder := r.URL.Query().Get(statusFolderParam)
	system := r.PathValue("system")
	name := r.PathValue("path")
	if !slices.Contains(ui.romsFolders, folder) {
		renderProblem(w, http.StatusBadRequest, "Bad request",
			"This request named a ROMs folder this server does not offer.")
		return
	}
	if !slices.Contains(romsFolderMediaExtensions, strings.ToLower(path.Ext(name))) {
		renderProblem(w, http.StatusNotFound, "Not found", "No such cover art in this ROMs folder.")
		return
	}

	if system == "." || system == ".." || system != filepath.Base(system) {
		renderProblem(w, http.StatusNotFound, "Not found", "No such cover art in this ROMs folder.")
		return
	}
	root, err := os.OpenRoot(filepath.Join(folder, system))
	if err != nil {
		renderProblem(w, http.StatusNotFound, "Not found", "No such cover art in this ROMs folder.")
		return
	}
	defer root.Close()

	http.ServeFileFS(w, r, root.FS(), name)
}

// registryURLFor builds the link to a gap's own registry page, when it has
// one — an empty registryID (no matching entry) yields no link at all,
// which is what the template branches on to offer none.
func registryURLFor(system, registryID string) string {
	if registryID == "" {
		return ""
	}
	return gameURL(system, registryID)
}

// fieldLabels turns a list of registry.Field*/registry.Medium identifiers
// into the comma-separated labels a user already knows them by from the
// edit form and the media section — rather than a table of its own repeating
// the same twelve spellings.
func fieldLabels(ids []string) string {
	labels := make([]string, len(ids))
	for i, id := range ids {
		labels[i] = fieldLabel(id)
	}
	return strings.Join(labels, ", ")
}

// fieldLabel resolves one field or medium identifier to its label, trying
// the media table first (registry.Medium is also a plain string) and falling
// back to the metadata one; an identifier belonging to neither — which
// gapFields never actually produces — falls back to itself.
func fieldLabel(id string) string {
	if m, ok := registry.LookupMedium(id); ok {
		return mediaLabels[m]
	}
	for _, f := range editableFields {
		if f.Manual == id {
			return f.Label
		}
	}
	return id
}

// romsFolderStatusTemplate renders one ROMs folder's summary: one row per
// system, worst first, each leading to that system's own paginated page.
var romsFolderStatusTemplate = newPage("roms-folder-status", `
{{define "title"}}Status of {{.Folder}} - Registry{{end}}
{{define "body"}}
<nav class="crumbs" aria-label="Breadcrumb">
<a href="/">Registry</a><span class="crumbs__sep">/</span><span class="crumbs__current">ROMs folder status</span>
</nav>
<main>
<h2 class="game__title">Status of <code>{{.Folder}}</code></h2>
<p class="home__note">A read-only report: opening it changes nothing, here or in the registry. Open a system to see its games.</p>
{{if not .Systems}}
<p class="empty-state">Nothing to report: no ROM file and no gamelist.xml entry were found in this folder.</p>
{{else}}
<div class="status__table-wrap">
<table class="status__table">
<thead><tr><th scope="col">System</th><th scope="col">ROM files</th><th scope="col">Complete</th><th scope="col">Incomplete</th><th scope="col">Not listed</th><th scope="col">Fillable now</th></tr></thead>
<tbody>
{{range .Systems}}
<tr>
<th scope="row"><a href="{{.URL}}">{{.Name}}</a></th>
{{if .Problem}}<td colspan="5" class="errors__title">{{.Problem}}</td>
{{else}}<td>{{.ROMCount}}</td><td>{{.CompleteCount}}</td><td>{{.IncompleteCount}}</td><td>{{.NotListedCount}}</td><td>{{.FillableCount}}</td>{{end}}
</tr>
{{end}}
</tbody>
</table>
</div>
{{end}}
</main>
{{end}}
`)

// romsFolderSystemStatusTemplate renders one page of one system's games
// within a ROMs folder: every game, complete ones included, each with the
// cover art the folder and the registry hold side by side, what it still
// misses, and the controls copying chosen values between the folder and
// the registry, either way.
var romsFolderSystemStatusTemplate = newPage("roms-folder-system-status", `
{{define "title"}}{{.Name}} in {{.Folder}} - Registry{{end}}
{{define "body"}}
<nav class="crumbs" aria-label="Breadcrumb">
<a href="/">Registry</a><span class="crumbs__sep">/</span><a href="{{.FolderURL}}">ROMs folder status</a><span class="crumbs__sep">/</span><span class="crumbs__current">{{.Name}}</span>
</nav>
<nav class="console" aria-label="Systems of this ROMs folder">
<a class="console__brand" href="{{.FolderURL}}">Folder</a>
<div class="console__systems">
{{range .Systems}}<a href="{{.URL}}"{{if eq .Name $.Name}} class="is-current" aria-current="page"{{end}}>{{.Name}}</a>
{{end}}
</div>
</nav>
<main>
{{if .Changed}}<p class="banner" id="changed" role="status" tabindex="-1">{{.Changed}}</p>{{end}}
<section class="system">
<h2 class="system__title">{{.Name}} <span class="system__count">{{.Total}} games</span></h2>
<p class="home__note">In <code>{{.Folder}}</code>. Read-only: copying one game below changes that game alone.</p>
{{if .Problem}}
<p class="errors__title">{{.Problem}}</p>
{{else}}
<dl class="status__counts">
<div><dt>ROM files</dt><dd>{{.ROMCount}}</dd></div>
<div><dt>Complete</dt><dd>{{.Complete}}</dd></div>
<div><dt>Incomplete</dt><dd>{{.Incomplete}}</dd></div>
<div><dt>Not listed</dt><dd>{{.NotListed}}</dd></div>
<div><dt>Fillable now</dt><dd>{{.Fillable}}</dd></div>
</dl>
{{if not .Games}}
<p class="empty-state">No game in this system.</p>
{{else}}
{{template "pager" .}}
<ul class="status__gaps">
{{range .Games}}
<li class="status__gap">
<div class="status__arts">
<figure class="status__art{{if not .LocalImageURL}} status__art--empty{{end}}">
{{if .LocalImageURL}}<img src="{{.LocalImageURL}}" alt="Cover art of {{.ROMFilename}} in this ROMs folder" loading="lazy">{{end}}
<figcaption>ROMs folder</figcaption>
</figure>
<figure class="status__art{{if not .RegistryImageURL}} status__art--empty{{end}}">
{{if .RegistryImageURL}}<img src="{{.RegistryImageURL}}" alt="Cover art of {{.ROMFilename}} in the registry" loading="lazy">{{end}}
<figcaption>Registry</figcaption>
</figure>
</div>
<div class="status__gap-body">
<p><span class="status__gap-name">{{if .Name}}{{.Name}}{{else}}{{.ROMFilename}}{{end}}</span>
<code class="status__gap-file">{{.ROMFilename}}</code></p>
{{if .Complete}}<p class="status__gap-complete">Complete.</p>
{{else}}
{{if not .Listed}}<p class="status__gap-state">Not scraped locally yet.</p>
{{else}}<p class="status__gap-state">Missing: {{.Missing}}.</p>{{end}}
{{if .Fillable}}<p class="status__gap-fillable">A Completion would already fill: {{.Fillable}}.</p>
{{else}}<p class="status__gap-stuck">Needs real scraping — the registry does not know this either.</p>{{end}}
{{end}}
{{if .RegistryURL}}<p class="status__gap-link"><a href="{{.RegistryURL}}">Open in the registry &rarr;</a></p>{{end}}
<div class="status__gap-actions">
{{if .ToRegistryURL}}<a class="button button--quiet" href="{{.ToRegistryURL}}">Folder → Registry</a>{{end}}
{{if .ToFolderURL}}<a class="button button--quiet" href="{{.ToFolderURL}}">Registry → Folder</a>{{end}}
{{if .FillURL}}<a class="button button--quiet" href="{{.FillURL}}">Fill in by hand</a>{{end}}
</div>
</div>
</li>
{{end}}
</ul>
{{template "pager" .}}
{{end}}
{{end}}
<a class="back-to-top" href="#top">&#9650; Back to top</a>
</section>
</main>
{{end}}
`+pagerTemplate)
