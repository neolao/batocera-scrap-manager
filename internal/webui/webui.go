// Package webui serves the registry's content over HTTP: a page listing
// every game grouped by system, and one page per game showing its full
// metadata and media. It renders with the presentation layer shared with the
// static consultation site (see internal/site), so both look alike, but
// navigates to real per-game URLs instead of the static site's in-page
// overlays — see decisions/015.
package webui

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path"
	"slices"
	"sort"
	"sync"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/site"
)

// mediaURLPrefix is the URL space under which the registry folder's media
// files are served.
const mediaURLPrefix = "/media/"

// Handler returns the HTTP handler serving reg's content, reading media
// files from registryFolder. The registry is a snapshot: the handler
// renders reg as it was given, it never reloads it from disk. romsFolders is
// the configured list of Batocera ROMs folders, which the completion writes
// back to; none configured is a valid state, not an error.
func Handler(reg *registry.Registry, registryFolder string, romsFolders []string) http.Handler {
	ui := &webUI{reg: reg, registryFolder: registryFolder, romsFolders: romsFolders}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", ui.serveHome)
	mux.HandleFunc("GET /system/{system}", ui.serveSystem)
	mux.HandleFunc("GET /game/{system}/{id}", ui.serveGame)
	mux.HandleFunc("GET /game/{system}/{id}/edit", ui.serveEditForm)
	mux.HandleFunc("POST /game/{system}/{id}/edit", ui.saveGame)
	// The methodless pattern is what answers 405: the catch-all below matches
	// every method, so the mux's own method-not-allowed handling would never
	// fire for this URL.
	mux.HandleFunc("/game/{system}/{id}/edit", ui.serveWrongMethod)
	mux.HandleFunc("POST /game/{system}/{id}/protect", ui.setProtection)
	mux.HandleFunc("/game/{system}/{id}/protect", ui.serveWrongProtectMethod)
	mux.HandleFunc("GET /game/{system}/{id}/delete", ui.serveDeleteConfirmation)
	mux.HandleFunc("POST /game/{system}/{id}/delete", ui.deleteGame)
	mux.HandleFunc("/game/{system}/{id}/delete", ui.serveWrongDeleteMethod)
	mux.HandleFunc("GET "+completeURL, ui.serveComplete)
	mux.HandleFunc("POST "+completeURL, ui.startCompletion)
	mux.HandleFunc(completeURL, ui.serveWrongCompleteMethod)
	mux.Handle("GET "+mediaURLPrefix, http.StripPrefix(mediaURLPrefix,
		http.FileServer(fileOnlyFS{http.Dir(registryFolder)})))
	mux.HandleFunc("/", ui.serveUnknownPage)
	return mux
}

// webUI holds what every page needs: the registry snapshot to render, the
// folder its media files are read from, and the ROMs folders the completion
// writes back to. Requests are served concurrently and a correction replaces
// the snapshot, so mu guards every access to reg — the readers as much as the
// writer. romsFolders is fixed for the process's lifetime (the configuration
// is read once, when serve starts), so it needs no guarding.
type webUI struct {
	mu             sync.RWMutex
	reg            *registry.Registry
	registryFolder string
	romsFolders    []string
	// completion tracks the one completion of the ROMs folders the server runs
	// at a time. It guards itself: a run outlives its request and must not hold
	// mu, which every served page needs.
	completion completionState
}

// homeView is the summary of the registry: what systems it holds and how many
// games each has, plus the confirmation of a deletion that redirected here.
// The home page is where a deletion lands when it emptied its system — the
// deleted game's own page is gone, and so is its system's.
type homeView struct {
	Systems []systemSummary
	Deleted string
}

// systemSummary is one system on the home page, and one entry of the system
// navigation every list page carries.
type systemSummary struct {
	Name  string
	Count int
	URL   string
}

// gameCard is one game as shown on a system's list: enough to recognize it,
// plus the link to its own page.
type gameCard struct {
	Name     string
	Desc     string
	Year     string
	URL      string
	ImageURL string
}

// gameDetail is one game's own page. Every media URL is empty when the file
// is not actually present in the registry folder, and Fields always carries
// the same labels in the same order, whether or not the game has a value for
// them.
type gameDetail struct {
	Name      string
	System    string
	SystemURL string
	EditURL   string
	DeleteURL string
	Saved     string
	// RomPath is the game's ROM path as stored, subfolder included. It sits
	// outside Fields because it is not scraped metadata: it is what identifies
	// the entry, and nothing shields it from a later update.
	RomPath  string
	Desc     string
	CoverURL string
	VideoURL string
	Extras   []extraMedium
	Fields   []metadataField
	// Protection states, in words, whether updates may refresh this game, and
	// offers the one control that state allows.
	Protection protectionControl
}

// extraMedium is a labelled still image of a game beyond its cover art
// (marquee, thumbnail), listed only when its file exists.
type extraMedium struct {
	Label string
	URL   string
}

// metadataField is one labelled metadata row of a game's page. An empty
// Value renders as a placeholder rather than dropping the row, so the set of
// labels stays the same from one game to the next. HandEdited marks a value
// corrected by hand, which later updates leave alone.
type metadataField struct {
	Label      string
	Value      string
	HandEdited bool
}

// serveHome renders the registry's summary: one entry per system, naming how
// many games it holds and linking to its own list.
//
// It deliberately does not go through site.GroupBySystem: that resolves every
// media reference of every entry against the disk, which is thousands of
// os.Stat calls for a page that shows no media at all.
func (ui *webUI) serveHome(w http.ResponseWriter, r *http.Request) {
	ui.mu.RLock()
	summaries := systemSummaries(ui.reg.Entries)
	ui.mu.RUnlock()

	query := r.URL.Query()
	render(w, http.StatusOK, homeTemplate, homeView{
		Systems: summaries,
		Deleted: deletedConfirmation(query.Get(deletedParam), query.Get(systemParam), query[warningParam]),
	})
}

// systemSummaries counts the games of each system, sorted by system name —
// the same order site.GroupBySystem renders them in, so the summary and the
// lists agree.
func systemSummaries(entries []registry.Entry) []systemSummary {
	counts := map[string]int{}
	for _, entry := range entries {
		counts[entry.System]++
	}

	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)

	summaries := make([]systemSummary, len(names))
	for i, name := range names {
		summaries[i] = systemSummary{Name: name, Count: counts[name], URL: systemURL(name)}
	}
	return summaries
}

// serveGame renders one game's full metadata and media, or the not-found
// page when the URL designates a system or a game the registry does not
// know.
func (ui *webUI) serveGame(w http.ResponseWriter, r *http.Request) {
	system, id := r.PathValue("system"), r.PathValue("id")

	ui.mu.RLock()
	entry, found := ui.reg.FindByID(system, id)
	ui.mu.RUnlock()
	if !found {
		renderGameNotFound(w, system, id)
		return
	}

	detail := ui.gameDetail(entry)
	detail.Saved = savedConfirmation(r.URL.Query())
	render(w, http.StatusOK, gameTemplate, detail)
}

// serveUnknownPage renders the not-found page for any URL the mux has no
// route for.
func (ui *webUI) serveUnknownPage(w http.ResponseWriter, r *http.Request) {
	renderProblem(w, http.StatusNotFound, "Not found", "There is nothing at "+r.URL.Path+".")
}

// renderGameNotFound renders the not-found page for a URL naming a game the
// registry does not know — an unknown system, an unknown identifier, or a
// game requested under the wrong system.
func renderGameNotFound(w http.ResponseWriter, system, id string) {
	renderProblem(w, http.StatusNotFound, "Not found", "No game named "+id+" in system "+system+".")
}

// gameDetail builds the rendering view of one registry entry, reusing the
// static site's grouping so media presence, rating and year are resolved
// exactly the same way on both.
func (ui *webUI) gameDetail(entry registry.Entry) gameDetail {
	view := site.GroupBySystem([]registry.Entry{entry}, ui.registryFolder)[0].Games[0]

	// A fully protected game says so once, at the game level: lighting the mark
	// of every single field under that sentence reads as noise, or as a
	// contradiction.
	fullyProtected := entry.FullyProtected()
	handEdited := func(field string) bool {
		return !fullyProtected && slices.Contains(entry.ManualFields, field)
	}
	detail := gameDetail{
		Name:       view.Name,
		System:     view.System,
		SystemURL:  systemURL(view.System),
		EditURL:    gameURL(view.System, view.ID) + "/edit",
		DeleteURL:  gameURL(view.System, view.ID) + "/delete",
		Protection: protectionOf(entry, view.System, view.ID),
		RomPath:    entry.Game.Path,
		Desc:       view.Desc,
		CoverURL:   mediaURL(view.ImagePath),
		VideoURL:   mediaURL(view.VideoPath),
		Fields: []metadataField{
			{Label: "Rating", Value: ratingValue(view), HandEdited: handEdited("rating")},
			{Label: "Year", Value: view.Year, HandEdited: handEdited("release_date")},
			{Label: "Developer", Value: view.Developer, HandEdited: handEdited("developer")},
			{Label: "Publisher", Value: view.Publisher, HandEdited: handEdited("publisher")},
			{Label: "Genre", Value: view.Genre, HandEdited: handEdited("genre")},
			{Label: "Players", Value: view.Players, HandEdited: handEdited("players")},
		},
	}

	for _, medium := range []struct{ label, path string }{
		{"Marquee", view.MarqueePath},
		{"Thumbnail", view.ThumbnailPath},
	} {
		if medium.path != "" {
			detail.Extras = append(detail.Extras, extraMedium{Label: medium.label, URL: mediaURL(medium.path)})
		}
	}
	return detail
}

// ratingValue renders a game's rating as stars followed by the same value in
// words, so it is not conveyed by star glyphs alone — which a screen reader
// announces poorly or not at all.
func ratingValue(view site.GameView) string {
	if view.Stars == "" {
		return ""
	}
	return view.Stars + " " + view.RatingLabel
}

// gameURL builds the URL of a game's page from its system and registry
// identifier, percent-encoding each of them so names containing spaces,
// brackets or non-ASCII characters still produce a valid link that routes
// back to the same game.
func gameURL(system, id string) string {
	return "/game/" + url.PathEscape(system) + "/" + url.PathEscape(id)
}

// systemURL builds the URL of one system's paginated game list, percent-
// encoding its name so a system whose folder holds spaces or non-ASCII
// characters still produces a valid link routing back to it.
func systemURL(system string) string {
	return "/system/" + url.PathEscape(system)
}

// mediaURL turns an already percent-encoded registry-relative media path (as
// computed by site.GroupBySystem) into the URL serving it, leaving an empty
// path empty — that is how a missing medium is signalled to the templates.
// The path is cleaned because gamelist.xml writes media references in the
// "./images/foo.png" form: served over HTTP, such a non-canonical URL makes
// http.FileServer answer a redirect whose Location re-escapes the already
// escaped path, and the browser then follows it to a file that does not
// exist. The static site links the very same media relatively, where the
// extra "./" is harmless, so this is a transport concern rather than a
// presentation difference.
func mediaURL(escapedPath string) string {
	if escapedPath == "" {
		return ""
	}
	return path.Clean(mediaURLPrefix + escapedPath)
}

// problem is what the themed error page shows: the status code, a short
// title for the browser tab, and a sentence naming what went wrong.
type problem struct {
	Code    int
	Title   string
	Message string
}

// renderProblem renders the themed error page — every refusal the web UI
// answers (a game that does not exist, a method it does not accept, a
// submission it cannot read) is shown in-theme rather than as a bare error
// string.
func renderProblem(w http.ResponseWriter, status int, title, message string) {
	// A 404 is cacheable by heuristic, and the pages that answer one here are
	// URLs whose game may well come back — a later import recreating what a
	// deletion removed. Only the refusals are kept out of the cache; the pages
	// that do render content stay cacheable, so going back to the list still
	// restores its scroll position.
	w.Header().Set("Cache-Control", "no-store")
	render(w, status, problemTemplate, problem{Code: status, Title: title, Message: message})
}

// render writes one HTML page, rendering it fully before sending anything so
// a template failure surfaces as a clean error instead of a truncated page.
func render(w http.ResponseWriter, status int, page *template.Template, data any) {
	var buf bytes.Buffer
	if err := page.Execute(&buf, data); err != nil {
		http.Error(w, "failed to render the page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// fileOnlyFS serves the registry folder's files while refusing its
// directories, so the registry's arborescence is never listed. http.Dir
// already confines every lookup to that folder, neutralizing path traversal.
type fileOnlyFS struct {
	dir http.Dir
}

func (fs fileOnlyFS) Open(name string) (http.File, error) {
	f, err := fs.dir.Open(name)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if info.IsDir() {
		f.Close()
		return nil, os.ErrNotExist
	}
	return f, nil
}

//go:embed page.css
var pageCSS string

// layout is the chrome every served page shares: the theme (the same
// stylesheet as the static site, plus the rules specific to served pages),
// the marquee header, and the closing tags. Each page defines the "title"
// and "body" templates it fills it with.
var layout = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{block "title" .}}Registry{{end}}</title>
{{block "head" .}}{{end}}
<style>{{stylesheet}}</style>
</head>
<body id="top">
<header class="marquee">
<h1><a class="marquee__link" href="/">Registry</a></h1>
</header>
{{template "body" .}}
</body>
</html>
`

// newPage parses one page's body (and optional title) on top of the shared
// layout.
func newPage(name, body string) *template.Template {
	page := template.New(name).Funcs(template.FuncMap{
		"stylesheet": func() template.CSS { return site.StyleSheet + template.CSS(pageCSS) },
	})
	return template.Must(template.Must(page.Parse(layout)).Parse(body))
}

// homeTemplate summarizes the registry system by system, each one linking to
// its own paginated list. It names no game on purpose: a registry of several
// thousand games would otherwise be serialized into the very first page a
// browser opens.
var homeTemplate = newPage("home", `
{{define "body"}}
<main>
{{if .Deleted}}<p class="banner" id="deleted" role="status" tabindex="-1">{{.Deleted}}</p>{{end}}
{{if not .Systems}}
<p class="empty-state">No games in the registry yet.</p>
{{else}}
<div class="home__actions">
<a class="button" href="`+completeURL+`">Complete the ROMs folders</a>
</div>
<h2 class="system__title">Systems</h2>
<ul class="systems">
{{range .Systems}}
<li><a class="systems__item" href="{{.URL}}">
<span class="systems__name">{{.Name}}</span>
<span class="systems__count">{{.Count}}</span>
</a></li>
{{end}}
</ul>
{{end}}
</main>
{{end}}
`)

// gameTemplate renders one game's own page: cover art, description, the full
// set of metadata labels, and every medium available for it.
var gameTemplate = newPage("game", `
{{define "title"}}{{.Name}} - Registry{{end}}
{{define "body"}}
<nav class="crumbs" aria-label="Breadcrumb">
<a href="/">Registry</a><span class="crumbs__sep">/</span><a href="{{.SystemURL}}">{{.System}}</a><span class="crumbs__sep">/</span><span class="crumbs__current">{{.Name}}</span>
</nav>
<main>
{{if .Saved}}<p class="banner" id="saved" role="status" tabindex="-1">{{.Saved}}</p>{{end}}
<article class="game">
<h2 class="game__title">{{.Name}}</h2>
<p class="game__file"><span class="game__file-label">`+pathLabel+`</span> <code>{{.RomPath}}</code></p>
<div class="game__layout">
<div class="game__art{{if not .CoverURL}} game__art--empty{{end}}">
{{if .CoverURL}}<img src="{{.CoverURL}}" alt="Cover art of {{.Name}}">{{else}}No cover art{{end}}
</div>
<div>
{{if .Desc}}<p class="game__desc">{{.Desc}}</p>{{else}}<p class="game__desc game__desc--empty">No description available.</p>{{end}}
<dl class="meta">
{{range .Fields}}
<div>
<dt class="meta__label">{{.Label}}{{if .HandEdited}} <span class="meta__manual" title="Corrected by hand: updates leave this value alone">hand-edited</span>{{end}}</dt>
{{if .Value}}<dd class="meta__value">{{.Value}}</dd>{{else}}<dd class="meta__value meta__value--empty">&mdash;</dd>{{end}}
</div>
{{end}}
</dl>
<p class="meta__state">{{.Protection.Summary}}</p>
<div class="meta__actions">
<a class="button" href="{{.EditURL}}">Edit metadata</a>
{{if .Protection.Label}}
<form class="meta__protect" method="post" action="{{.Protection.Action}}">
<input type="hidden" name="protected" value="{{.Protection.Value}}">
<button class="button button--quiet" type="submit">{{.Protection.Label}}</button>
</form>
{{end}}
<a class="button button--danger meta__delete" href="{{.DeleteURL}}">Delete from the registry</a>
</div>
</div>
</div>
{{if or .VideoURL .Extras}}
<section class="media">
<h3 class="media__title">Media</h3>
<div class="media__grid">
{{if .VideoURL}}
<div class="media__item">
<span class="media__label">Video</span>
<video src="{{.VideoURL}}" controls muted loop playsinline preload="none"></video>
</div>
{{end}}
{{range .Extras}}
<div class="media__item">
<span class="media__label">{{.Label}}</span>
<img src="{{.URL}}" alt="{{.Label}} of {{$.Name}}" loading="lazy">
</div>
{{end}}
</div>
</section>
{{end}}
<a class="back-to-top" href="{{.SystemURL}}">&#9650; Back to {{.System}}</a>
</article>
</main>
{{end}}
`)

// problemTemplate renders a themed page naming what went wrong, rather than
// a bare error string.
var problemTemplate = newPage("problem", `
{{define "title"}}{{.Title}} - Registry{{end}}
{{define "body"}}
<main>
<div class="notfound">
<p class="notfound__code">{{.Code}}</p>
<p class="notfound__message">{{.Message}}</p>
<a class="notfound__back" href="/">Back to the game list</a>
</div>
</main>
{{end}}
`)
