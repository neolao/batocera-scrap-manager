package webui

import (
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// writeMediaFile creates a dummy media file at
// <registryFolder>/<system>/<relPath>, as if a previous update had copied it
// there, so the handler's existence checks find it.
func writeMediaFile(t *testing.T, registryFolder, system, relPath string) {
	t.Helper()
	fullPath := filepath.Join(registryFolder, system, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to set up test media file: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("dummy media"), 0o644); err != nil {
		t.Fatalf("failed to set up test media file: %v", err)
	}
}

// fullyScrapedRegistry builds a registry folder holding one thoroughly
// scraped game (every metadata field, every media file present on disk) plus
// a second system, and returns the registry and its folder.
func fullyScrapedRegistry(t *testing.T) (*registry.Registry, string) {
	t.Helper()
	registryFolder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{
			Path:        "./Sonic the Hedgehog.zip",
			Name:        "Sonic the Hedgehog",
			Desc:        "A blue hedgehog runs very fast through Green Hill Zone.",
			Image:       "images/sonic.png",
			Video:       "videos/sonic.mp4",
			Marquee:     "images/sonic-marquee.png",
			Thumbnail:   "images/sonic-thumb.png",
			Rating:      "0.8",
			ReleaseDate: "19910623T000000",
			Developer:   "Sonic Team",
			Publisher:   "Sega",
			Genre:       "Platform",
			Players:     "1",
		}},
		{System: "mastersystem", Game: gamelist.Game{
			Path:  "./Alex Kidd.zip",
			Name:  "Alex Kidd in Miracle World",
			Desc:  "A kid with miracle powers.",
			Image: "images/alex.png",
		}},
	}}
	for _, media := range []struct{ system, relPath string }{
		{"megadrive", "images/sonic.png"},
		{"megadrive", "videos/sonic.mp4"},
		{"megadrive", "images/sonic-marquee.png"},
		{"megadrive", "images/sonic-thumb.png"},
		{"mastersystem", "images/alex.png"},
	} {
		writeMediaFile(t, registryFolder, media.system, media.relPath)
	}
	return reg, registryFolder
}

// get runs one GET request against the handler and returns the response
// recorder.
func get(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

var hrefPattern = regexp.MustCompile(`href="([^"]*)"`)

// cardLinks extracts the game links (those pointing at a game page) from an
// HTML document, unescaping them back into requestable URLs.
func cardLinks(body string) []string {
	var links []string
	for _, match := range hrefPattern.FindAllStringSubmatch(body, -1) {
		link := html.UnescapeString(match[1])
		if strings.HasPrefix(link, "/game/") {
			links = append(links, link)
		}
	}
	return links
}

// systemLinks extracts the links to system pages from an HTML document,
// unescaping them back into requestable URLs.
func systemLinks(body string) []string {
	var links []string
	for _, match := range hrefPattern.FindAllStringSubmatch(body, -1) {
		link := html.UnescapeString(match[1])
		if strings.HasPrefix(link, "/system/") {
			links = append(links, link)
		}
	}
	return links
}

// systemLink returns the first link to a system page of an HTML document, or
// an empty string when it holds none.
func systemLink(body string) string {
	links := systemLinks(body)
	if len(links) == 0 {
		return ""
	}
	return links[0]
}

var pagerPattern = regexp.MustCompile(`rel="(?:prev|next)" href="([^"]*)"`)

// pagerLinks extracts the previous/next links of a paginated list, unescaping
// them back into requestable URLs.
func pagerLinks(body string) []string {
	var links []string
	for _, match := range pagerPattern.FindAllStringSubmatch(body, -1) {
		links = append(links, html.UnescapeString(match[1]))
	}
	return links
}

var srcPattern = regexp.MustCompile(`src="([^"]*)"`)

// mediaSources extracts every src attribute of an HTML document, unescaping
// them back into requestable URLs.
func mediaSources(body string) []string {
	var sources []string
	for _, match := range srcPattern.FindAllStringSubmatch(body, -1) {
		sources = append(sources, html.UnescapeString(match[1]))
	}
	return sources
}

func TestHandler_HomePage_ListsEachSystemWithItsGameCount(t *testing.T) {
	reg, registryFolder := crowdedRegistry(t, "megadrive", 3)
	reg.Entries = append(reg.Entries, registry.Entry{System: "snes", Game: gamelist.Game{
		Path: "./Zelda.zip", Name: "A Link to the Past",
	}})

	rec := get(t, Handler(reg, registryFolder), "/")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"megadrive", "snes", ">3<", ">1<"} {
		if !strings.Contains(body, want) {
			t.Errorf("home page does not contain %q, got: %s", want, body)
		}
	}
}

func TestHandler_HomePage_DoesNotRenderIndividualGames(t *testing.T) {
	// The whole point of the summary: a registry of thousands of games must
	// not be serialized into the page a phone opens first.
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder), "/").Body.String()

	for _, unwanted := range []string{"Sonic the Hedgehog", "Alex Kidd in Miracle World", "<img"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("home page renders %q, want a systems summary only, got: %s", unwanted, body)
		}
	}
}

func TestHandler_HomePage_EverySystemLinkLeadsToItsPage(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	h := Handler(reg, registryFolder)

	links := systemLinks(get(t, h, "/").Body.String())

	if len(links) != 2 {
		t.Fatalf("found %d system links on the home page, want 2 (links: %v)", len(links), links)
	}
	for _, link := range links {
		if rec := get(t, h, link); rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", link, rec.Code)
		}
	}
}

func TestHandler_HomePage_EmptyRegistry_ShowsAnEmptyStateNotABlankPage(t *testing.T) {
	rec := get(t, Handler(&registry.Registry{}, t.TempDir()), "/")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "No games in the registry yet") {
		t.Errorf("empty registry home page does not explain there is nothing to show, got: %s", rec.Body.String())
	}
}

func TestHandler_GamePage_FullyScrapedGame_ShowsEveryMetadataAndMedium(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, registryFolder), "/game/megadrive/Sonic%20the%20Hedgehog")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Sonic the Hedgehog",
		"A blue hedgehog runs very fast through Green Hill Zone.",
		"★★★★☆", "4/5", "1991", "Sonic Team", "Sega", "Platform",
		"/media/megadrive/images/sonic.png",
		"/media/megadrive/videos/sonic.mp4",
		"/media/megadrive/images/sonic-marquee.png",
		"/media/megadrive/images/sonic-thumb.png",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("game page does not contain %q", want)
		}
	}
}

func TestHandler_GamePage_LinksBackToTheListAndToItsSystem(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder), "/game/megadrive/Sonic%20the%20Hedgehog").Body.String()

	if !strings.Contains(body, `href="/"`) {
		t.Errorf("game page has no link back to the game list, got: %s", body)
	}
	if !strings.Contains(body, `href="/system/megadrive"`) {
		t.Errorf("game page has no link back to its system's page, got: %s", body)
	}
}

func TestHandler_GamePage_GameWithoutMetadataOrMedia_KeepsEveryFieldLabel(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "Bare Bones.zip", Name: "Bare Bones"}},
	}}

	rec := get(t, Handler(reg, registryFolder), "/game/megadrive/Bare%20Bones")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Rating", "Year", "Developer", "Publisher", "Genre", "Players"} {
		if !strings.Contains(body, want) {
			t.Errorf("game page drops the %q label for a game missing that field", want)
		}
	}
	if !strings.Contains(body, "No description available") {
		t.Errorf("game page does not say the description is missing, got: %s", body)
	}
	if !strings.Contains(body, "No cover art") {
		t.Errorf("game page does not say the cover art is missing, got: %s", body)
	}
	if strings.Contains(body, "<video") || strings.Contains(body, "<img") {
		t.Errorf("game page renders a media element for a game with no media, got: %s", body)
	}
	if strings.Contains(body, `src=""`) {
		t.Errorf("game page renders an empty src attribute, got: %s", body)
	}
}

func TestHandler_GamePage_MediaReferencedButMissingOnDisk_IsNotLinked(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{
			Path:  "Sonic.zip",
			Name:  "Sonic",
			Image: "images/sonic.png",
			Video: "videos/gone.mp4",
		}},
	}}
	writeMediaFile(t, registryFolder, "megadrive", "images/sonic.png")

	body := get(t, Handler(reg, registryFolder), "/game/megadrive/Sonic").Body.String()

	if !strings.Contains(body, "/media/megadrive/images/sonic.png") {
		t.Errorf("game page does not link the jaquette that exists on disk, got: %s", body)
	}
	if strings.Contains(body, "<video") {
		t.Errorf("game page renders a video player for a video file missing from disk, got: %s", body)
	}
}

func TestHandler_GamePage_UnknownGameSystemOrMalformedPath_Returns404(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	h := Handler(reg, registryFolder)

	targets := []string{
		"/game/megadrive/Does%20Not%20Exist", // unknown game
		"/game/nosuchsystem/Sonic",           // unknown system
		"/game/mastersystem/Sonic",           // known game, wrong system
		"/game/",                             // no system, no game
		"/game/megadrive",                    // no game
		"/game/megadrive/",                   // empty game id
		"/game/megadrive/Sonic/extra",        // trailing junk
		"/nope",                              // unknown page
	}
	for _, target := range targets {
		if rec := get(t, h, target); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", target, rec.Code)
		}
	}
}

func TestHandler_NotFoundPage_IsStyledAndOffersAWayBack(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, registryFolder), "/game/megadrive/Does%20Not%20Exist")

	body := rec.Body.String()
	if !strings.Contains(body, "<style") {
		t.Errorf("404 page is not styled like the rest of the site, got: %s", body)
	}
	if !strings.Contains(body, `href="/"`) {
		t.Errorf("404 page offers no link back to the game list, got: %s", body)
	}
	if !strings.Contains(body, "Does Not Exist") {
		t.Errorf("404 page does not name what was not found, got: %s", body)
	}
}

func TestHandler_Media_ExistingFile_IsServed(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, registryFolder), "/media/megadrive/images/sonic.png")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "dummy media" {
		t.Errorf("body = %q, want the media file's content", rec.Body.String())
	}
}

func TestHandler_Media_PathOutsideTheRegistryFolder_IsRefused(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	secret := filepath.Join(filepath.Dir(registryFolder), "secret.txt")
	if err := os.WriteFile(secret, []byte("top secret"), 0o644); err != nil {
		t.Fatalf("failed to set up the file outside the registry: %v", err)
	}
	h := Handler(reg, registryFolder)

	for _, target := range []string{
		"/media/../secret.txt",
		"/media/..%2Fsecret.txt",
		"/media/megadrive/../../secret.txt",
		"/media/%2e%2e%2fsecret.txt",
	} {
		rec := get(t, h, target)
		if strings.Contains(rec.Body.String(), "top secret") {
			t.Errorf("GET %s served a file outside the registry folder", target)
		}
		if rec.Code == http.StatusOK {
			t.Errorf("GET %s = 200, want an error status", target)
		}
	}
}

func TestHandler_Media_Directory_IsNotListed(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	h := Handler(reg, registryFolder)

	for _, target := range []string{"/media/", "/media/megadrive/", "/media/megadrive/images/"} {
		rec := get(t, h, target)
		if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), "sonic") {
			t.Errorf("GET %s lists the registry folder's content", target)
		}
	}
}

func TestHandler_GameWithSpecialCharacters_LinksAndMediaRoundTrip(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "gb", Game: gamelist.Game{
			Path:  "./Pokémon Red & Blue #1 (USA) [!].zip",
			Name:  "Pokémon Red & Blue",
			Desc:  "Catch them all.",
			Image: "images/Pokémon Red & Blue #1 [!].png",
		}},
	}}
	writeMediaFile(t, registryFolder, "gb", "images/Pokémon Red & Blue #1 [!].png")
	h := Handler(reg, registryFolder)

	links := cardLinks(get(t, h, systemLink(get(t, h, "/").Body.String())).Body.String())

	if len(links) != 1 {
		t.Fatalf("found %d game links, want 1 (links: %v)", len(links), links)
	}
	rec := get(t, h, links[0])
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", links[0], rec.Code)
	}
	var mediaRequested int
	for _, src := range mediaSources(rec.Body.String()) {
		if !strings.HasPrefix(src, "/media/") {
			continue
		}
		mediaRequested++
		if mediaRec := get(t, h, src); mediaRec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", src, mediaRec.Code)
		}
	}
	if mediaRequested != 1 {
		t.Errorf("game page references %d media files, want 1", mediaRequested)
	}
}

func TestHandler_Pages_AreNotServedWithNoStoreCaching(t *testing.T) {
	// no-store defeats the browser's scroll restoration, so going back to
	// the list after opening a game would jump back to the top of the page.
	reg, registryFolder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, registryFolder), "/")

	if strings.Contains(rec.Header().Get("Cache-Control"), "no-store") {
		t.Errorf("Cache-Control = %q, want no no-store directive", rec.Header().Get("Cache-Control"))
	}
}

func TestHandler_DotPrefixedMediaPath_IsServedWithoutARedirect(t *testing.T) {
	// gamelist.xml (as written by EmulationStation/Batocera) references media
	// as "./images/foo.png". Served as-is, that non-canonical URL triggers a
	// redirect whose Location re-escapes the already escaped path, so the
	// browser ends up requesting a file that does not exist.
	registryFolder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{
			Path:  "./Sonic.zip",
			Name:  "Sonic",
			Image: "./images/Sonic [!].png",
		}},
	}}
	writeMediaFile(t, registryFolder, "megadrive", "images/Sonic [!].png")
	h := Handler(reg, registryFolder)

	body := get(t, h, "/game/megadrive/Sonic").Body.String()

	var checked int
	for _, src := range mediaSources(body) {
		if !strings.HasPrefix(src, "/media/") {
			continue
		}
		checked++
		if strings.Contains(src, "/./") {
			t.Errorf("media URL %q is not canonical", src)
		}
		if rec := get(t, h, src); rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200 without any redirect", src, rec.Code)
		}
	}
	if checked != 1 {
		t.Fatalf("game page references %d media files, want 1", checked)
	}
}
