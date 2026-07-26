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
	"sync"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/site"
)

// mediaURLPrefix is the URL space under which the registry folder's media
// files are served.
const mediaURLPrefix = "/media/"

// Handler returns the HTTP handler serving reg's content, reading media
// files from registryFolder. The registry is a snapshot: the handler
// renders reg as it was given, it never reloads it from disk.
func Handler(reg *registry.Registry, registryFolder string) http.Handler {
	ui := &webUI{reg: reg, registryFolder: registryFolder}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", ui.serveHome)
	mux.HandleFunc("GET /game/{system}/{id}", ui.serveGame)
	mux.HandleFunc("GET /game/{system}/{id}/edit", ui.serveEditForm)
	mux.HandleFunc("POST /game/{system}/{id}/edit", ui.saveGame)
	// The methodless pattern is what answers 405: the catch-all below matches
	// every method, so the mux's own method-not-allowed handling would never
	// fire for this URL.
	mux.HandleFunc("/game/{system}/{id}/edit", ui.serveWrongMethod)
	mux.Handle("GET "+mediaURLPrefix, http.StripPrefix(mediaURLPrefix,
		http.FileServer(fileOnlyFS{http.Dir(registryFolder)})))
	mux.HandleFunc("/", ui.serveUnknownPage)
	return mux
}

// webUI holds what every page needs: the registry snapshot to render, and
// the folder its media files are read from. Requests are served
// concurrently and a correction replaces the snapshot, so mu guards every
// access to reg — the readers as much as the writer.
type webUI struct {
	mu             sync.RWMutex
	reg            *registry.Registry
	registryFolder string
}

// systemListing is one system's section of the home page.
type systemListing struct {
	Name  string
	Games []gameCard
}

// gameCard is one game as shown on the home page: enough to recognize it,
// plus the link to its own page.
type gameCard struct {
	Name     string
	Desc     string
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
	Saved     string
	Desc      string
	CoverURL  string
	VideoURL  string
	Extras    []extraMedium
	Fields    []metadataField
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

// serveHome renders every game of the registry, grouped by system.
func (ui *webUI) serveHome(w http.ResponseWriter, _ *http.Request) {
	ui.mu.RLock()
	systems := site.GroupBySystem(ui.reg.Entries, ui.registryFolder)
	ui.mu.RUnlock()

	listings := make([]systemListing, len(systems))
	for i, system := range systems {
		cards := make([]gameCard, len(system.Games))
		for j, game := range system.Games {
			cards[j] = gameCard{
				Name:     game.Name,
				Desc:     game.Desc,
				URL:      gameURL(game.System, game.ID),
				ImageURL: mediaURL(game.ImagePath),
			}
		}
		listings[i] = systemListing{Name: system.Name, Games: cards}
	}

	render(w, http.StatusOK, homeTemplate, listings)
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
	detail.Saved = savedConfirmation(r.URL.Query().Get(savedParam))
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

	handEdited := func(field string) bool { return slices.Contains(entry.ManualFields, field) }
	detail := gameDetail{
		Name:      view.Name,
		System:    view.System,
		SystemURL: "/#" + view.System,
		EditURL:   gameURL(view.System, view.ID) + "/edit",
		Desc:      view.Desc,
		CoverURL:  mediaURL(view.ImagePath),
		VideoURL:  mediaURL(view.VideoPath),
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

// homeTemplate lists every game grouped by system, each card linking to the
// game's own page.
var homeTemplate = newPage("home", `
{{define "body"}}
{{if not .}}
<p class="empty-state">No games in the registry yet.</p>
{{else}}
<nav class="console" aria-label="Systems">
<a class="console__brand" href="#top">Registry</a>
<div class="console__systems">
{{range .}}<a href="#{{.Name}}">{{.Name}}</a>
{{end}}
</div>
</nav>
<main>
{{range .}}
<section id="{{.Name}}" class="system">
<h2 class="system__title">{{.Name}}</h2>
<div class="grid">
{{range .Games}}
<a class="card" href="{{.URL}}">
<div class="card__art{{if not .ImageURL}} card__art--empty{{end}}">
{{if .ImageURL}}<img src="{{.ImageURL}}" alt="Cover art of {{.Name}}" loading="lazy">{{end}}
</div>
<div class="card__body">
<h3 class="card__name">{{.Name}}</h3>
<p class="card__desc">{{.Desc}}</p>
</div>
</a>
{{end}}
</div>
<a class="back-to-top" href="#top">&#9650; Back to top</a>
</section>
{{end}}
</main>
{{end}}
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
<p class="meta__actions"><a class="button" href="{{.EditURL}}">Edit metadata</a></p>
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
