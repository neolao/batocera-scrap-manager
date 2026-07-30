package webui

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// crowdedRegistry builds a registry holding count games in system, named
// "Game 001".."Game NNN" so their alphabetical order — the order the pages are
// cut in — is also their numbering.
func crowdedRegistry(t *testing.T, system string, count int) (*registry.Registry, string) {
	t.Helper()
	reg := &registry.Registry{}
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("Game %03d", i)
		reg.Entries = append(reg.Entries, registry.Entry{System: system, Game: gamelist.Game{
			Path: "./" + name + ".zip",
			Name: name,
		}})
	}
	return reg, t.TempDir()
}

func TestServeSystem_ListsOnlyThatSystemsGames(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, registryFolder, nil), "/system/megadrive")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Sonic the Hedgehog") {
		t.Errorf("system page does not list its own game, got: %s", body)
	}
	if strings.Contains(body, "Alex Kidd in Miracle World") {
		t.Errorf("system page lists a game of another system, got: %s", body)
	}
}

func TestServeSystem_GameCardsLinkToTheirOwnPage(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	h := Handler(reg, registryFolder, nil)

	links := cardLinks(get(t, h, "/system/megadrive").Body.String())

	if len(links) != 1 {
		t.Fatalf("found %d game links, want 1 (links: %v)", len(links), links)
	}
	if rec := get(t, h, links[0]); rec.Code != http.StatusOK {
		t.Errorf("GET %s = %d, want 200", links[0], rec.Code)
	}
}

func TestServeSystem_UnknownSystem_ReturnsNotFound(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, registryFolder, nil), "/system/nosuchsystem")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "nosuchsystem") {
		t.Errorf("404 page does not name the system that was not found, got: %s", rec.Body.String())
	}
}

func TestServeSystem_FirstPage_ShowsTheFirstGamesAndNoPreviousLink(t *testing.T) {
	reg, registryFolder := crowdedRegistry(t, "megadrive", gamesPerPage+5)

	body := get(t, Handler(reg, registryFolder, nil), "/system/megadrive").Body.String()

	if !strings.Contains(body, "Game 001") {
		t.Errorf("first page does not show the first game, got: %s", body)
	}
	if !strings.Contains(body, fmt.Sprintf("Game %03d", gamesPerPage)) {
		t.Errorf("first page does not show the last game of the page, got: %s", body)
	}
	if strings.Contains(body, fmt.Sprintf("Game %03d", gamesPerPage+1)) {
		t.Errorf("first page shows a game belonging to the next page, got: %s", body)
	}
	if strings.Contains(body, `rel="prev"`) {
		t.Errorf("first page offers a link to a previous page, got: %s", body)
	}
	if !strings.Contains(body, `rel="next"`) {
		t.Errorf("first page offers no link to the next page, got: %s", body)
	}
}

func TestServeSystem_SecondPage_ShowsTheNextGamesAndBothPagerLinks(t *testing.T) {
	reg, registryFolder := crowdedRegistry(t, "megadrive", 2*gamesPerPage+5)

	rec := get(t, Handler(reg, registryFolder, nil), "/system/megadrive?page=2")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "Game 001") {
		t.Errorf("second page still shows the first page's games, got: %s", body)
	}
	if !strings.Contains(body, fmt.Sprintf("Game %03d", gamesPerPage+1)) {
		t.Errorf("second page does not show the first game of the page, got: %s", body)
	}
	if !strings.Contains(body, `rel="prev"`) || !strings.Contains(body, `rel="next"`) {
		t.Errorf("second page of three does not offer both pager links, got: %s", body)
	}
}

func TestServeSystem_LastPage_HasNoNextLink(t *testing.T) {
	reg, registryFolder := crowdedRegistry(t, "megadrive", gamesPerPage+5)

	body := get(t, Handler(reg, registryFolder, nil), "/system/megadrive?page=2").Body.String()

	if !strings.Contains(body, fmt.Sprintf("Game %03d", gamesPerPage+5)) {
		t.Errorf("last page does not show the very last game, got: %s", body)
	}
	if strings.Contains(body, `rel="next"`) {
		t.Errorf("last page offers a link to a next page, got: %s", body)
	}
	if !strings.Contains(body, `rel="prev"`) {
		t.Errorf("last page offers no link back to the previous page, got: %s", body)
	}
}

func TestServeSystem_PagerLinksLeadToPagesThatExist(t *testing.T) {
	reg, registryFolder := crowdedRegistry(t, "megadrive", 2*gamesPerPage+5)
	h := Handler(reg, registryFolder, nil)

	// The pager is rendered above and below the list, so each target appears
	// twice: what matters is the set of pages it leads to.
	var targets []string
	for _, link := range pagerLinks(get(t, h, "/system/megadrive?page=2").Body.String()) {
		if slices.Contains(targets, link) {
			continue
		}
		targets = append(targets, link)
		if rec := get(t, h, link); rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", link, rec.Code)
		}
	}
	if len(targets) != 2 {
		t.Fatalf("the second page leads to %d distinct pages, want 2 (%v)", len(targets), targets)
	}
}

func TestServeSystem_SinglePage_RendersNoPager(t *testing.T) {
	reg, registryFolder := crowdedRegistry(t, "megadrive", 3)

	body := get(t, Handler(reg, registryFolder, nil), "/system/megadrive").Body.String()

	if strings.Contains(body, `<nav class="pager"`) {
		t.Errorf("a system fitting on one page still renders a pager, got: %s", body)
	}
}

func TestServeSystem_UnusablePageNumber_ReturnsNotFound(t *testing.T) {
	reg, registryFolder := crowdedRegistry(t, "megadrive", gamesPerPage+5)
	h := Handler(reg, registryFolder, nil)

	// A page out of bounds is a wrong URL, not an empty page: answering 200
	// with nothing on it would hide a broken link.
	for _, target := range []string{
		"/system/megadrive?page=0",
		"/system/megadrive?page=-1",
		"/system/megadrive?page=3",
		"/system/megadrive?page=999",
		"/system/megadrive?page=abc",
		"/system/megadrive?page=",
		"/system/megadrive?page=1.5",
	} {
		if rec := get(t, h, target); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", target, rec.Code)
		}
	}
}

func TestServeSystem_ExplicitFirstPage_IsAccepted(t *testing.T) {
	reg, registryFolder := crowdedRegistry(t, "megadrive", gamesPerPage+5)

	rec := get(t, Handler(reg, registryFolder, nil), "/system/megadrive?page=1")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Game 001") {
		t.Errorf("page=1 does not show the first page's games, got: %s", rec.Body.String())
	}
}

func TestServeSystem_SystemNameWithSpecialCharacters_LinksRoundTrip(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "pc engine cd", Game: gamelist.Game{Path: "./Ys.zip", Name: "Ys Book I & II"}},
	}}
	h := Handler(reg, registryFolder, nil)

	link := systemLink(get(t, h, "/").Body.String())

	if link == "" {
		t.Fatal("the home page offers no link to the system")
	}
	rec := get(t, h, link)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", link, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Ys Book I &amp; II") {
		t.Errorf("system page does not list its game, got: %s", rec.Body.String())
	}
}

func TestServeSystem_MarksTheCurrentSystemInTheNavigation(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder, nil), "/system/megadrive").Body.String()

	if !strings.Contains(body, `aria-current="page"`) {
		t.Errorf("system page does not mark the current system in its navigation, got: %s", body)
	}
	if !strings.Contains(body, `href="/system/mastersystem"`) {
		t.Errorf("system page does not offer to jump to the other system, got: %s", body)
	}
}

func TestServeSystem_DoesNotEmbedVideoPlayers(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder, nil), "/system/megadrive").Body.String()

	if strings.Contains(body, "<video") {
		t.Errorf("system page embeds a video player, got: %s", body)
	}
	if !strings.Contains(body, `loading="lazy"`) {
		t.Errorf("system page does not defer loading of its cover art, got: %s", body)
	}
}

func TestServeSystem_GameWithoutJaquette_RendersPlaceholderNotABrokenImage(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic", Image: "images/gone.png"}},
	}}

	body := get(t, Handler(reg, registryFolder, nil), "/system/megadrive").Body.String()

	if strings.Contains(body, "<img") {
		t.Errorf("system page renders an <img> for a jaquette missing from disk, got: %s", body)
	}
	if !strings.Contains(body, "card__art--empty") {
		t.Errorf("system page does not render the placeholder card art, got: %s", body)
	}
}

func TestServeSystem_GameCards_DoNotShowTheReleaseYear(t *testing.T) {
	// The list is browsed by name and cover art; the game's own page carries
	// the year for whoever wants it.
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder, nil), "/system/megadrive").Body.String()

	if strings.Contains(body, "1991") {
		t.Errorf("system page still shows the release year, got: %s", body)
	}
	if strings.Contains(body, "card__meta") {
		t.Errorf("system page still renders a year slot on its cards, got: %s", body)
	}
}

func TestServeSystem_GameCards_StillNameTheGamesTheyLinkTo(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder, nil), "/system/megadrive").Body.String()

	for _, want := range []string{"Sonic the Hedgehog", "A blue hedgehog runs very fast through Green Hill Zone."} {
		if !strings.Contains(body, want) {
			t.Errorf("system page does not contain %q, got: %s", want, body)
		}
	}
}

func TestServeSystem_IsNotServedWithNoStoreCaching(t *testing.T) {
	// no-store defeats the browser's scroll restoration, so going back to the
	// list after opening a game would jump back to the top of the page.
	reg, registryFolder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, registryFolder, nil), "/system/megadrive")

	if strings.Contains(rec.Header().Get("Cache-Control"), "no-store") {
		t.Errorf("Cache-Control = %q, want no no-store directive", rec.Header().Get("Cache-Control"))
	}
}

func TestServeSystem_FullyProtectedGame_ShowsProtectedBadge(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	if err := registry.Protect(reg, "megadrive", "Sonic the Hedgehog"); err != nil {
		t.Fatalf("Protect() error = %v", err)
	}

	body := get(t, Handler(reg, registryFolder, nil), "/system/megadrive").Body.String()

	if !strings.Contains(body, `<span class="card__protected"`) {
		t.Errorf("system page does not mark the fully protected game, got: %s", body)
	}
	if !strings.Contains(body, ">Protected<") {
		t.Errorf("system page does not carry an accessible \"Protected\" label, got: %s", body)
	}
}

func TestServeSystem_PartlyProtectedGame_DoesNotShowProtectedBadge(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic"}, ManualFields: []string{"name"}},
	}}

	body := get(t, Handler(reg, registryFolder, nil), "/system/megadrive").Body.String()

	if strings.Contains(body, `<span class="card__protected"`) {
		t.Errorf("system page marks a partly protected game as fully protected, got: %s", body)
	}
}

func TestServeSystem_UnprotectedGame_DoesNotShowProtectedBadge(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder, nil), "/system/megadrive").Body.String()

	if strings.Contains(body, `<span class="card__protected"`) {
		t.Errorf("system page marks an unprotected game as protected, got: %s", body)
	}
}
