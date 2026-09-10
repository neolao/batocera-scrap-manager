package webui

import (
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// romsFolderStatusURL is the one page reporting what one configured ROMs
// folder is still missing, compared to what a Completion could already fill
// from the registry. It is read-only: opening it never writes anything.
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

// romsFolderStatusView is the report's own page: which folder it is about,
// and each of its systems.
type romsFolderStatusView struct {
	Folder  string
	Systems []systemStatusView
}

// systemStatusView is one system's share of the report, plus what a template
// needs beyond registry.SystemStatus itself: an anchor to jump to it from the
// table of contents at the top, since the whole report renders on one page.
type systemStatusView struct {
	Anchor          string
	Name            string
	ROMCount        int
	CompleteCount   int
	IncompleteCount int
	NotListedCount  int
	FillableCount   int
	Problem         string
	Gaps            []gapView
}

// gapView is one problem game, with its missing and fillable fields already
// turned into the labels a user knows them by — the registry only ever names
// them by their internal identifiers.
type gapView struct {
	ROMFilename string
	Name        string
	Listed      bool
	Missing     string
	Fillable    string
	// RegistryURL leads to this game's own page in the registry, so its scrape
	// can be corrected or completed by hand right there. Empty when the
	// registry does not know this game at all — nothing to link to.
	RegistryURL string
}

// serveRomsFolderStatus renders the read-only status report of one
// configured ROMs folder. Nothing is written by opening it: the registry is
// only read, and the ROMs folder itself is only scanned.
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

	status, err := registry.RomsFolderStatus(reg, folder)
	if err != nil {
		renderProblem(w, http.StatusInternalServerError, "Could not read this folder",
			"This ROMs folder could not be read.")
		return
	}

	render(w, http.StatusOK, romsFolderStatusTemplate, romsFolderStatusViewOf(status))
}

// romsFolderStatusViewOf turns a registry.FolderStatus into what the template
// renders, resolving every field and medium identifier to its label.
func romsFolderStatusViewOf(status registry.FolderStatus) romsFolderStatusView {
	view := romsFolderStatusView{Folder: status.RomsFolder}
	for _, sys := range status.Systems {
		sv := systemStatusView{
			Anchor:          "system-" + sys.System,
			Name:            sys.System,
			ROMCount:        sys.ROMCount,
			CompleteCount:   sys.CompleteCount,
			IncompleteCount: sys.IncompleteCount,
			NotListedCount:  sys.NotListedCount,
			FillableCount:   sys.FillableCount,
			Problem:         sys.Problem,
		}
		for _, gap := range sys.Gaps {
			sv.Gaps = append(sv.Gaps, gapView{
				ROMFilename: gap.ROMFilename,
				Name:        gap.Name,
				Listed:      gap.Listed,
				Missing:     fieldLabels(gap.Missing),
				Fillable:    fieldLabels(gap.Fillable),
				RegistryURL: registryURLFor(sys.System, gap.RegistryID),
			})
		}
		view.Systems = append(view.Systems, sv)
	}
	return view
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

// romsFolderStatusTemplate renders one ROMs folder's status: per system, how
// much of it already matches what the registry knows, and the list of the
// games that do not — each telling apart a gap a Completion would already
// fill from one needing real scraping. Systems come from the registry
// already ordered by how many problems they hold, worst first; a table of
// contents at the top is what makes that order easy to skip to on the one
// long page the whole report renders on.
var romsFolderStatusTemplate = newPage("roms-folder-status", `
{{define "title"}}Status of {{.Folder}} - Registry{{end}}
{{define "body"}}
<nav class="crumbs" aria-label="Breadcrumb">
<a href="/">Registry</a><span class="crumbs__sep">/</span><span class="crumbs__current">ROMs folder status</span>
</nav>
<main>
<h2 class="game__title">Status of <code>{{.Folder}}</code></h2>
<p class="home__note">A read-only report: opening it changes nothing, here or in the registry.</p>
{{if not .Systems}}
<p class="empty-state">Nothing to report: no ROM file and no gamelist.xml entry were found in this folder.</p>
{{else}}
<ul class="status__toc">
{{range .Systems}}<li><a href="#{{.Anchor}}">{{.Name}}</a></li>{{end}}
</ul>
{{range .Systems}}
<section id="{{.Anchor}}" class="status__system">
<h3 class="system__title">{{.Name}}</h3>
{{if .Problem}}
<p class="errors__title">{{.Problem}}</p>
{{else}}
<dl class="status__counts">
<div><dt>ROM files</dt><dd>{{.ROMCount}}</dd></div>
<div><dt>Complete</dt><dd>{{.CompleteCount}}</dd></div>
<div><dt>Incomplete</dt><dd>{{.IncompleteCount}}</dd></div>
<div><dt>Not listed</dt><dd>{{.NotListedCount}}</dd></div>
<div><dt>Fillable now</dt><dd>{{.FillableCount}}</dd></div>
</dl>
{{if not .Gaps}}
<p class="empty-state">Nothing missing here.</p>
{{else}}
<ul class="status__gaps">
{{range .Gaps}}
<li class="status__gap">
<p><span class="status__gap-name">{{if .Name}}{{.Name}}{{else}}{{.ROMFilename}}{{end}}</span>
<code class="status__gap-file">{{.ROMFilename}}</code></p>
{{if not .Listed}}<p class="status__gap-state">Not scraped locally yet.</p>
{{else if .Missing}}<p class="status__gap-state">Missing: {{.Missing}}.</p>{{end}}
{{if .Fillable}}<p class="status__gap-fillable">A Completion would already fill: {{.Fillable}}.</p>
{{else}}<p class="status__gap-stuck">Needs real scraping — the registry does not know this either.</p>{{end}}
{{if .RegistryURL}}<p class="status__gap-link"><a href="{{.RegistryURL}}">Complete the scrape in the registry &rarr;</a></p>{{end}}
</li>
{{end}}
</ul>
{{end}}
{{end}}
</section>
{{end}}
{{end}}
</main>
{{end}}
`)
