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

// readAndSubmit is what most of this site's URLs allow: opening a page, and
// submitting the form on it. The exceptions are the URLs that only ever take a
// submission, which name their own single method.
const readAndSubmit = http.MethodGet + ", " + http.MethodPost

// Handler returns the HTTP handler serving reg's content, reading media
// files from registryFolder. The registry is a snapshot: the handler renders
// reg as it was given and never reloads it from disk, but a change made
// through the web UI — a correction, a deletion, an import — replaces it, so
// the result shows straight away. romsFolders is the configured list of
// Batocera ROMs folders the two long operations read from and write back to;
// none configured is a valid state, not an error.
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
	mux.HandleFunc("/game/{system}/{id}/edit", allowOnly(readAndSubmit))
	mux.HandleFunc("POST /game/{system}/{id}/protect", ui.setProtection)
	mux.HandleFunc("/game/{system}/{id}/protect", allowOnly(http.MethodPost))
	mux.HandleFunc("GET /game/{system}/{id}/send", ui.serveSendConfirmation)
	mux.HandleFunc("POST /game/{system}/{id}/send", ui.sendGame)
	mux.HandleFunc("/game/{system}/{id}/send", allowOnly(readAndSubmit))
	mux.HandleFunc("POST /game/{system}/{id}/media/{medium}", ui.uploadMedium)
	mux.HandleFunc("/game/{system}/{id}/media/{medium}", allowOnly(http.MethodPost))
	mux.HandleFunc("GET /game/{system}/{id}/media/{medium}/delete", ui.serveMediumDeleteConfirmation)
	mux.HandleFunc("POST /game/{system}/{id}/media/{medium}/delete", ui.removeMedium)
	mux.HandleFunc("/game/{system}/{id}/media/{medium}/delete", allowOnly(readAndSubmit))
	mux.HandleFunc("GET /game/{system}/{id}/delete", ui.serveDeleteConfirmation)
	mux.HandleFunc("POST /game/{system}/{id}/delete", ui.deleteGame)
	mux.HandleFunc("/game/{system}/{id}/delete", allowOnly(readAndSubmit))
	mux.HandleFunc("GET "+importURL, ui.serveImport)
	mux.HandleFunc("POST "+importURL, ui.startImport)
	mux.HandleFunc(importURL, allowOnly(readAndSubmit))
	mux.HandleFunc("GET "+completeURL, ui.serveComplete)
	mux.HandleFunc("POST "+completeURL, ui.startCompletion)
	mux.HandleFunc(completeURL, allowOnly(readAndSubmit))
	mux.Handle("GET "+mediaURLPrefix, http.StripPrefix(mediaURLPrefix,
		http.FileServer(fileOnlyFS{http.Dir(registryFolder)})))
	mux.HandleFunc("/", ui.serveUnknownPage)
	return mux
}

// webUI holds what every page needs: the registry snapshot to render, the
// folder its media files are read from, and the ROMs folders the import reads
// and the completion writes back to. Requests are served concurrently and a
// change replaces the snapshot, so mu guards every access to reg — the readers
// as much as the writers. romsFolders is fixed for the process's lifetime (the
// configuration is read once, when serve starts), so it needs no guarding.
type webUI struct {
	mu             sync.RWMutex
	reg            *registry.Registry
	registryFolder string
	romsFolders    []string
	// jobs tracks the one long operation the server runs at a time, whichever
	// direction it goes. It guards itself: a run outlives its request and must
	// not hold mu, which every served page needs.
	jobs jobs
}

// homeView is the summary of the registry: what systems it holds and how many
// games each has, plus the confirmation of a deletion that redirected here.
// The home page is where a deletion lands when it emptied its system — the
// deleted game's own page is gone, and so is its system's.
//
// Configured reports whether there is any ROMs folder at all, which is what
// decides between offering the two maintenance flows and naming the command
// that configures one: a button leading to a dead end is worse than a
// sentence.
type homeView struct {
	Systems    []systemSummary
	Deleted    string
	Configured bool
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
	Fields   []metadataField
	// Media is the four media the game may hold, each with what it currently
	// holds and the controls managing it. All four are always rendered: a page
	// showing only what exists could not offer to add what does not.
	Media []mediumControl
	// Problem states why a change asked for from this page did not happen — a
	// refused upload has nothing to redirect to, since nothing changed.
	Problem string
	// Protection states, in words, whether updates may refresh this game, and
	// offers the one control that state allows.
	Protection protectionControl
	// Send offers to write this game into one of the configured ROMs folders,
	// under one of the two rules a send may follow.
	Send sendControl
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
		Systems:    summaries,
		Deleted:    deletedConfirmation(query.Get(deletedParam), query.Get(systemParam), query[warningParam]),
		Configured: len(ui.romsFolders) > 0,
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
		Send:       sendControlOf(view.System, view.ID, ui.romsFolders),
		RomPath:    entry.Game.Path,
		Desc:       view.Desc,
		CoverURL:   mediaURL(view.ImagePath),
		Fields: []metadataField{
			{Label: "Rating", Value: ratingValue(view), HandEdited: handEdited("rating")},
			{Label: "Year", Value: view.Year, HandEdited: handEdited("release_date")},
			{Label: "Developer", Value: view.Developer, HandEdited: handEdited("developer")},
			{Label: "Publisher", Value: view.Publisher, HandEdited: handEdited("publisher")},
			{Label: "Genre", Value: view.Genre, HandEdited: handEdited("genre")},
			{Label: "Players", Value: view.Players, HandEdited: handEdited("players")},
		},
	}

	detail.Media = mediaControlsOf(entry, view.System, view.ID, map[registry.Medium]string{
		registry.MediumImage:     view.ImagePath,
		registry.MediumVideo:     view.VideoPath,
		registry.MediumMarquee:   view.MarqueePath,
		registry.MediumThumbnail: view.ThumbnailPath,
	})
	return detail
}

// allowOnly answers the themed 405 page for a method a URL does not support,
// naming the ones it does. Every such URL needs it registered under its own
// methodless pattern: the catch-all route matches every method, so the mux's
// own method-not-allowed handling would never fire for them.
func allowOnly(methods string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", methods)
		renderProblem(w, http.StatusMethodNotAllowed, "Not allowed",
			r.Method+" is not something this page accepts.")
	}
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
//
// The two maintenance flows are offered above that summary, and outside the
// "is the registry empty" branch: an empty registry is exactly when importing
// matters most. Each is named by the direction it goes rather than by its verb
// alone — "import" and "complete" are equally opaque, and only one of the two
// writes into the user's own Batocera files.
var homeTemplate = newPage("home", `
{{define "body"}}
<main>
{{if .Deleted}}<p class="banner" id="deleted" role="status" tabindex="-1">{{.Deleted}}</p>{{end}}
<h2 class="system__title">Maintenance</h2>
{{if .Configured}}
<div class="home__actions">
<div class="home__action">
<a class="button" href="`+importURL+`">Import from the ROMs folders</a>
<p class="home__note">Brings what Batocera already scraped into the registry.</p>
</div>
<div class="home__action">
<a class="button" href="`+completeURL+`">Complete the ROMs folders</a>
<p class="home__note">Writes what the registry knows back into Batocera.</p>
</div>
</div>
{{else}}
<p class="home__note">No ROMs folder is configured yet. Add one with <code>batocera-scrap-manager config add-roms-folder &lt;path&gt;</code>, then restart the server.</p>
{{end}}
{{if not .Systems}}
<p class="empty-state">No games in the registry yet.</p>
{{else}}
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
{{if .Problem}}
<div class="errors" role="alert" tabindex="-1" id="errors">
<p class="errors__title">Nothing was changed</p>
<p>{{.Problem}}</p>
</div>
{{end}}
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
<section class="send">
<h3 class="send__title">Send to a ROMs folder</h3>
{{if .Send.Folders}}
<p class="send__lead">Writes what the registry knows about this game into one Batocera folder, this game alone.</p>
<form class="send__form" method="get" action="{{.Send.Action}}">
<div class="field">
<label class="field__label" for="send-folder">ROMs folder</label>
<select class="field__control field__control--path" id="send-folder" name="`+sendFolderParam+`">
{{range .Send.Folders}}<option value="{{.}}">{{.}}</option>
{{end}}
</select>
</div>
<fieldset class="field">
<legend class="field__label">What to write</legend>
{{range $index, $mode := .Send.Modes}}
<label class="send__mode">
<input type="radio" name="`+sendModeParam+`" value="{{$mode.Value}}"{{if not $index}} checked{{end}}>
<span class="send__mode-label">{{$mode.Label}}</span>
<span class="send__mode-note">{{$mode.Note}}</span>
</label>
{{end}}
</fieldset>
<div class="form__actions">
<button class="button" type="submit">Choose and confirm</button>
</div>
</form>
{{else}}
<p class="send__lead">No ROMs folder is configured yet. Add one with <code>batocera-scrap-manager config add-roms-folder &lt;path&gt;</code>, then restart the server.</p>
{{end}}
</section>
<section class="media">
<h3 class="media__title">Media</h3>
<p class="media__lead">Stored in the registry. Sending this game to a ROMs folder with the replace rule is what carries them to Batocera.</p>
<div class="media__grid">
{{range .Media}}
<div class="media__item">
<span class="media__label">{{.Label}}</span>
{{if .URL}}
{{if .Video}}<video src="{{.URL}}" controls muted loop playsinline preload="none"></video>
{{else}}<img src="{{.URL}}" alt="{{.Label}} of {{$.Name}}" loading="lazy">{{end}}
{{else if .Reference}}
<p class="media__missing">Referred to as <code>{{.Reference}}</code>, but that file is not in the registry.</p>
{{else}}
<p class="media__missing">None yet.</p>
{{end}}
<form class="media__upload" method="post" action="{{.UploadURL}}" enctype="multipart/form-data">
<label class="media__file" for="media-{{.Medium}}">Choose a {{.Label}} file<span class="media__accept">Accepted: {{.Accept}}</span></label>
<input class="field__control" type="file" id="media-{{.Medium}}" name="`+mediaFileParam+`" accept="{{.Accept}}" required>
<div class="media__actions">
<button class="button" type="submit">Upload</button>
{{if .DeleteURL}}<a class="button button--quiet" href="{{.DeleteURL}}">Remove</a>{{end}}
</div>
</form>
</div>
{{end}}
</div>
</section>
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
