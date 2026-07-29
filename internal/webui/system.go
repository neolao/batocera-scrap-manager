package webui

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/site"
)

// gamesPerPage is how many games one system page shows. A registry holds
// thousands of games and is browsed from a phone: serializing a whole system
// into one page is both unreadable and, since every card's media has to be
// looked up on disk, needlessly expensive.
const gamesPerPage = 60

// pageParam names the query parameter carrying which page of a system is
// asked for. Its absence means the first one.
const pageParam = "page"

// systemPage is one page of a system's games: the games themselves, where
// that page sits in the system, and the navigation to the other systems —
// which stays on every page, so hopping between systems never goes through
// the home page.
type systemPage struct {
	Name      string
	Systems   []systemSummary
	Games     []gameCard
	Page      int
	PageCount int
	// PrevURL and NextURL are empty on the first and last page respectively,
	// which is how the pager renders their controls as unavailable.
	PrevURL string
	NextURL string
	// First and Last are the ranks of this page's first and last game within
	// the whole system, so the pager can state what is being looked at.
	First int
	Last  int
	Total int
	// Deleted confirms a deletion that redirected here, the deleted game's own
	// page being gone.
	Deleted string
}

// serveSystem renders one page of a system's games. An unknown system, or a
// page number that does not designate an existing page, is a wrong URL rather
// than an empty list: both answer the not-found page, so a broken link is
// visible instead of looking like a system that lost its games.
func (ui *webUI) serveSystem(w http.ResponseWriter, r *http.Request) {
	system := r.PathValue("system")

	ui.mu.RLock()
	entries := entriesOfSystem(ui.reg.Entries, system)
	summaries := systemSummaries(ui.reg.Entries)
	registryFolder := ui.registryFolder
	ui.mu.RUnlock()

	if len(entries) == 0 {
		renderProblem(w, http.StatusNotFound, "Not found", "No system named "+system+" in the registry.")
		return
	}

	pageCount := (len(entries) + gamesPerPage - 1) / gamesPerPage
	page, ok := requestedPage(r.URL.Query(), pageCount)
	if !ok {
		renderProblem(w, http.StatusNotFound, "Not found",
			"There is no page "+r.URL.Query().Get(pageParam)+" in system "+system+".")
		return
	}

	first := (page - 1) * gamesPerPage
	last := min(first+gamesPerPage, len(entries))
	// Only this page's entries are grouped: GroupBySystem looks up every media
	// file of every entry it is given on disk, so grouping the whole system
	// would stat thousands of files to render sixty cards.
	games := site.GroupBySystem(entries[first:last], registryFolder)[0].Games

	query := r.URL.Query()
	render(w, http.StatusOK, systemTemplate, systemPage{
		Name:      system,
		Systems:   summaries,
		Games:     gameCards(games),
		Page:      page,
		PageCount: pageCount,
		PrevURL:   pageURL(system, page-1, 1, page),
		NextURL:   pageURL(system, page+1, page, pageCount),
		First:     first + 1,
		Last:      last,
		Total:     len(entries),
		Deleted:   deletedConfirmation(query.Get(deletedParam), query.Get(systemParam), query[warningParam]),
	})
}

// entriesOfSystem returns the entries belonging to system, in the order the
// shared presentation layer renders them (by game name) — so slicing a page
// out of them shows the same games site.GroupBySystem would have put there.
func entriesOfSystem(entries []registry.Entry, system string) []registry.Entry {
	var found []registry.Entry
	for _, entry := range entries {
		if entry.System == system {
			found = append(found, entry)
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].Game.Name < found[j].Game.Name })
	return found
}

// requestedPage reads which page of pageCount the query asks for, defaulting
// to the first when the parameter is absent. It reports false for anything
// that does not designate an existing page — a number out of bounds as much
// as a value that is not a number at all.
func requestedPage(query url.Values, pageCount int) (int, bool) {
	if !query.Has(pageParam) {
		return 1, true
	}

	page, err := strconv.Atoi(query.Get(pageParam))
	if err != nil || page < 1 || page > pageCount {
		return 0, false
	}
	return page, true
}

// pageURL builds the URL of one page of a system, or an empty string when
// page falls outside [low, high] — which is how the first page has no
// previous link and the last one no next link. The first page is addressed
// without a parameter, so a system has one canonical URL.
func pageURL(system string, page, low, high int) string {
	if page < low || page > high {
		return ""
	}
	if page == 1 {
		return systemURL(system)
	}
	return systemURL(system) + "?" + pageParam + "=" + strconv.Itoa(page)
}

// gameCards turns the views of one page's games into the cards the list
// renders, each linking to the game's own page.
func gameCards(games []site.GameView) []gameCard {
	cards := make([]gameCard, len(games))
	for i, game := range games {
		cards[i] = gameCard{
			Name:     game.Name,
			Desc:     game.Desc,
			URL:      gameURL(game.System, game.ID),
			ImageURL: mediaURL(game.ImagePath),
		}
	}
	return cards
}

// systemTemplate renders one page of a system's games: the navigation to the
// other systems, the pager framing the list above and below, and one card per
// game linking to its own page.
var systemTemplate = newPage("system", `
{{define "title"}}{{.Name}} - Registry{{end}}
{{define "body"}}
<nav class="crumbs" aria-label="Breadcrumb">
<a href="/">Registry</a><span class="crumbs__sep">/</span><span class="crumbs__current">{{.Name}}</span>
</nav>
<nav class="console" aria-label="Systems">
<a class="console__brand" href="/">Registry</a>
<div class="console__systems">
{{range .Systems}}<a href="{{.URL}}"{{if eq .Name $.Name}} class="is-current" aria-current="page"{{end}}>{{.Name}}</a>
{{end}}
</div>
</nav>
<main>
{{if .Deleted}}<p class="banner" id="deleted" role="status" tabindex="-1">{{.Deleted}}</p>{{end}}
<section class="system">
<h2 class="system__title">{{.Name}} <span class="system__count">{{.Total}} games</span></h2>
{{template "pager" .}}
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
{{template "pager" .}}
<a class="back-to-top" href="#top">&#9650; Back to top</a>
</section>
</main>
{{end}}

{{define "pager"}}
{{if gt .PageCount 1}}
<nav class="pager" aria-label="Pagination">
{{if .PrevURL}}<a class="button" rel="prev" href="{{.PrevURL}}">&#8249; Previous</a>{{else}}<span class="button button--disabled" aria-disabled="true">&#8249; Previous</span>{{end}}
<span class="pager__state">{{.First}}&ndash;{{.Last}} of {{.Total}} &middot; page {{.Page}} of {{.PageCount}}</span>
{{if .NextURL}}<a class="button" rel="next" href="{{.NextURL}}">Next &#8250;</a>{{else}}<span class="button button--disabled" aria-disabled="true">Next &#8250;</span>{{end}}
</nav>
{{end}}
{{end}}
`)
