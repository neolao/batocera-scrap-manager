package site

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// writeMediaFile creates a dummy media file on disk at
// <registryFolder>/<system>/<relPath>, as if it had already been copied
// there by a previous update, so existence checks in Generate find it.
func writeMediaFile(t *testing.T, registryFolder, system, relPath string) {
	t.Helper()
	fullPath := filepath.Join(registryFolder, system, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to set up test media file: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("dummy"), 0o644); err != nil {
		t.Fatalf("failed to set up test media file: %v", err)
	}
}

func TestGenerate_GamesGroupedBySystem_ListsNameDescriptionAndJaquette(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{
				Path:  "Sonic.zip",
				Name:  "Sonic the Hedgehog",
				Desc:  "A blue hedgehog runs fast.",
				Image: "images/sonic.png",
			}},
			{System: "mastersystem", Game: gamelist.Game{
				Path:  "Alex.zip",
				Name:  "Alex Kidd",
				Desc:  "A kid with miracle powers.",
				Image: "images/alex.png",
			}},
		},
	}
	writeMediaFile(t, registryFolder, "megadrive", "images/sonic.png")
	writeMediaFile(t, registryFolder, "mastersystem", "images/alex.png")

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	for _, want := range []string{
		"megadrive", "mastersystem",
		"Sonic the Hedgehog", "A blue hedgehog runs fast.", "megadrive/images/sonic.png",
		"Alex Kidd", "A kid with miracle powers.", "mastersystem/images/alex.png",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("index.html does not contain %q", want)
		}
	}
}

func TestGenerate_MultipleSystems_NavigationBarLinksToEachSystemAnchor(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic the Hedgehog"}},
			{System: "mastersystem", Game: gamelist.Game{Path: "Alex.zip", Name: "Alex Kidd"}},
		},
	}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	nav := extractTag(t, html, "nav")
	for _, want := range []string{`href="#megadrive"`, `href="#mastersystem"`} {
		if !strings.Contains(nav, want) {
			t.Errorf("navigation bar does not link to %q, got: %s", want, nav)
		}
	}

	for _, wantID := range []string{`id="megadrive"`, `id="mastersystem"`} {
		if !strings.Contains(html, wantID) {
			t.Errorf("index.html does not contain a section anchor %q", wantID)
		}
	}
}

func TestGenerate_SingleSystem_NavigationBarStillRendersLink(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic the Hedgehog"}},
		},
	}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	nav := extractTag(t, html, "nav")
	if !strings.Contains(nav, `href="#megadrive"`) {
		t.Errorf("navigation bar does not link to the only system, got: %s", nav)
	}
}

func TestGenerate_EmbeddedStylesheet_IncludesResponsiveRules(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic the Hedgehog"}},
		},
	}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	if !strings.Contains(html, "<style") {
		t.Error("index.html does not contain an embedded stylesheet")
	}
	if !strings.Contains(html, "@media") {
		t.Error("embedded stylesheet does not contain a responsive (@media) rule for small screens")
	}
}

func TestGenerate_EachSystemSection_HasBackToTopLink(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic the Hedgehog"}},
			{System: "mastersystem", Game: gamelist.Game{Path: "Alex.zip", Name: "Alex Kidd"}},
		},
	}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	if strings.Count(html, `href="#top"`) < 2 {
		t.Errorf("expected a back-to-top link in each of the 2 system sections, got: %s", html)
	}
}

func TestGenerate_CardDescription_IsClampedInTheGrid(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic the Hedgehog"}},
		},
	}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	if !strings.Contains(html, "-webkit-line-clamp") {
		t.Error("card description is not clamped, so it can grow to take all the card space")
	}
}

func TestGenerate_GameCard_LinksToADetailModalWithTheFullDescription(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{
				Path: "Sonic.zip",
				Name: "Sonic the Hedgehog",
				Desc: "A blue hedgehog runs very fast through Green Hill Zone, collecting rings along the way.",
			}},
		},
	}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	if !strings.Contains(html, `href="#modal-megadrive-0"`) {
		t.Errorf("game card does not link to its detail modal, got: %s", html)
	}
	if !strings.Contains(html, `id="modal-megadrive-0"`) {
		t.Errorf("no modal found for the game, got: %s", html)
	}
	modal := extractByID(t, html, "modal-megadrive-0")
	if !strings.Contains(modal, "A blue hedgehog runs very fast through Green Hill Zone, collecting rings along the way.") {
		t.Errorf("modal does not contain the game's full description, got: %s", modal)
	}
}

func TestGenerate_ModalClose_DoesNotLinkToAnAnchorThatWouldScrollThePage(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic the Hedgehog"}},
		},
	}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	modal := extractByID(t, html, "modal-megadrive-0")
	if strings.Contains(modal, `href="#megadrive"`) || strings.Contains(modal, `href="#top"`) {
		t.Errorf("modal close/backdrop links to a real page anchor, which would scroll the page on close, got: %s", modal)
	}
	if strings.Count(modal, `href="#_modal-close"`) < 2 {
		t.Errorf("expected both the close button and the backdrop to link to a non-existent anchor so closing does not scroll the page, got: %s", modal)
	}
}

func TestGenerate_Modal_ShowsGameMetadata(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{
				Path:        "Sonic.zip",
				Name:        "Sonic the Hedgehog",
				Rating:      "0.8",
				ReleaseDate: "19910623T000000",
				Developer:   "Sonic Team",
				Publisher:   "Sega",
				Genre:       "Platform",
				Players:     "1",
			}},
		},
	}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)
	modal := extractByID(t, html, "modal-megadrive-0")

	for _, want := range []string{"Sonic Team", "Sega", "Platform", "1991", "★★★★☆"} {
		if !strings.Contains(modal, want) {
			t.Errorf("modal does not contain %q, got: %s", want, modal)
		}
	}
}

func TestGenerate_Modal_OmitsMissingMetadata_WithoutCrashing(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic the Hedgehog"}},
		},
	}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)
	modal := extractByID(t, html, "modal-megadrive-0")

	if strings.Contains(modal, "★") {
		t.Errorf("modal shows a star rating for a game with no rating, got: %s", modal)
	}
}

func TestGenerate_Modal_ShowsVideoPlayer_WhenVideoAvailable(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic the Hedgehog", Video: "videos/sonic.mp4"}},
		},
	}
	writeMediaFile(t, registryFolder, "megadrive", "videos/sonic.mp4")

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)
	modal := extractByID(t, html, "modal-megadrive-0")

	if !strings.Contains(modal, "<video") {
		t.Errorf("modal does not contain a video player, got: %s", modal)
	}
	if !strings.Contains(modal, "megadrive/videos/sonic.mp4") {
		t.Errorf("modal video does not reference the game's video file, got: %s", modal)
	}
}

func TestGenerate_Modal_VideoPlayer_DoesNotPreloadUntilOpened(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic the Hedgehog", Video: "videos/sonic.mp4"}},
		},
	}
	writeMediaFile(t, registryFolder, "megadrive", "videos/sonic.mp4")

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)
	modal := extractByID(t, html, "modal-megadrive-0")

	if !strings.Contains(modal, `preload="none"`) {
		t.Errorf("video player does not disable preloading, so the browser fetches every game's video on page load, got: %s", modal)
	}
}

func TestGenerate_CardImage_LoadsLazily(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{
				Path:  "Sonic.zip",
				Name:  "Sonic the Hedgehog",
				Image: "images/sonic.png",
			}},
		},
	}
	writeMediaFile(t, registryFolder, "megadrive", "images/sonic.png")

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	if !strings.Contains(html, `<img src="megadrive/images/sonic.png" alt="Sonic the Hedgehog" loading="lazy">`) {
		t.Errorf("card image does not load lazily, so the browser fetches every game's jaquette on page load, got: %s", html)
	}
}

func TestGenerate_Modal_OmitsVideoPlayer_WhenNoVideo(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic the Hedgehog"}},
		},
	}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)
	modal := extractByID(t, html, "modal-megadrive-0")

	if strings.Contains(modal, "<video") {
		t.Errorf("modal contains a video player for a game with no video, got: %s", modal)
	}
}

func TestGenerate_ImagePath_PercentEncodesReservedCharacters(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "switch", Game: gamelist.Game{
				Path:  "CatQuest3.nsp",
				Name:  "Cat Quest III",
				Image: "images/Cat Quest III [010088501B8F2000] (1G+1U)-image.jpg",
			}},
		},
	}
	writeMediaFile(t, registryFolder, "switch", "images/Cat Quest III [010088501B8F2000] (1G+1U)-image.jpg")

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	if strings.Contains(html, "[010088501B8F2000]") {
		t.Errorf("image path contains un-encoded square brackets, which some HTTP servers mishandle, got: %s", html)
	}
	if !strings.Contains(html, "%5B010088501B8F2000%5D") {
		t.Errorf("image path does not percent-encode square brackets, got: %s", html)
	}
}

func TestGenerate_ImageFileMissingOnDisk_RendersPlaceholderInsteadOfBrokenLink(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{
				Path:  "Sonic.zip",
				Name:  "Sonic the Hedgehog",
				Image: "images/sonic.png",
			}},
		},
	}
	// Note: no file is actually written at registryFolder/megadrive/images/sonic.png.

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	if strings.Contains(html, "sonic.png") {
		t.Errorf("index.html references a jaquette file that does not exist on disk, got: %s", html)
	}
	if !strings.Contains(html, "card__art--empty") {
		t.Errorf("index.html does not render a placeholder for the missing jaquette, got: %s", html)
	}
}

func TestGenerate_VideoFileMissingOnDisk_OmitsVideoPlayer(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{
				Path:  "Sonic.zip",
				Name:  "Sonic the Hedgehog",
				Video: "videos/sonic.mp4",
			}},
		},
	}
	// Note: no file is actually written at registryFolder/megadrive/videos/sonic.mp4.

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)
	modal := extractByID(t, html, "modal-megadrive-0")

	if strings.Contains(modal, "<video") {
		t.Errorf("modal contains a video player referencing a video file that does not exist on disk, got: %s", modal)
	}
}

func TestGenerate_NavigationSystems_ScrollHorizontallyInsteadOfWrapping(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic the Hedgehog"}},
		},
	}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	if !strings.Contains(html, "overflow-x: auto") {
		t.Error("navigation systems list does not scroll horizontally when there are many systems")
	}
}

func TestGenerate_CardArtwork_UsesFourByThreeAspectRatio(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic the Hedgehog"}},
		},
	}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	if !strings.Contains(html, "aspect-ratio: 4 / 3") {
		t.Errorf("card artwork does not use a 4:3 aspect ratio, got: %s", html)
	}
}

func TestGenerate_GameWithoutJaquette_RendersWithoutBrokenImage(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{
				Path: "Streets.zip",
				Name: "Streets of Rage",
				Desc: "Beat 'em up.",
				// Image intentionally left empty.
			}},
		},
	}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	if !strings.Contains(html, "Streets of Rage") {
		t.Errorf("index.html does not contain the game name")
	}
	if strings.Contains(html, `src=""`) || strings.Contains(html, `<img src="megadrive/">`) {
		t.Errorf("index.html contains a broken image reference: %s", html)
	}
}

func TestGenerate_EmptyRegistry_ProducesValidSiteWithNoGamesMessage(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	if !strings.Contains(html, "No games") {
		t.Errorf("index.html does not contain a no-games message, got: %s", html)
	}
}

func TestGenerate_WritesIndexHTMLDirectlyAtRegistryRoot(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(registryFolder, "index.html")); err != nil {
		t.Errorf("index.html not found at the registry root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(registryFolder, "site", "index.html")); err == nil {
		t.Error("index.html was written under a site/ subfolder, want it directly at the registry root")
	}
}

func TestGenerate_LeftoverSiteSubfolder_IsLeftUntouched(t *testing.T) {
	registryFolder := t.TempDir()
	staleSiteFolder := filepath.Join(registryFolder, "site")
	if err := os.MkdirAll(staleSiteFolder, 0o755); err != nil {
		t.Fatalf("failed to set up test: %v", err)
	}
	stalePath := filepath.Join(staleSiteFolder, "index.html")
	if err := os.WriteFile(stalePath, []byte("stale content from a previous version"), 0o644); err != nil {
		t.Fatalf("failed to set up test: %v", err)
	}

	reg := &registry.Registry{}
	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := os.ReadFile(stalePath)
	if err != nil {
		t.Fatalf("leftover site/index.html was removed, want it left untouched: %v", err)
	}
	if string(data) != "stale content from a previous version" {
		t.Errorf("leftover site/index.html content changed, want it left untouched")
	}
}

func TestGenerate_IndexHTMLBlockedByExistingDirectory_ReturnsError(t *testing.T) {
	registryFolder := t.TempDir()
	// Create a directory where the index.html file needs to go, so writing fails.
	if err := os.MkdirAll(filepath.Join(registryFolder, "index.html"), 0o755); err != nil {
		t.Fatalf("failed to set up test: %v", err)
	}

	reg := &registry.Registry{}
	if err := Generate(reg, registryFolder); err == nil {
		t.Fatal("Generate() expected an error, got nil")
	}
}

func readIndex(t *testing.T, registryFolder string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(registryFolder, "index.html"))
	if err != nil {
		t.Fatalf("failed to read generated index.html: %v", err)
	}
	return string(data)
}

// extractTag returns the content of the first occurrence of the given tag
// name in html (e.g. "nav" for the first <nav ...>...</nav> block).
func extractTag(t *testing.T, html, tagName string) string {
	t.Helper()
	re := regexp.MustCompile(`(?s)<` + tagName + `[^>]*>.*?</` + tagName + `>`)
	match := re.FindString(html)
	if match == "" {
		t.Fatalf("index.html does not contain a <%s> element, got: %s", tagName, html)
	}
	return match
}

// extractByID returns the content of the element with the given id
// attribute, up to its matching closing div tag.
func extractByID(t *testing.T, html, id string) string {
	t.Helper()
	re := regexp.MustCompile(`(?s)<div[^>]*id="` + regexp.QuoteMeta(id) + `"[^>]*>.*?</div>\s*</div>`)
	match := re.FindString(html)
	if match == "" {
		t.Fatalf("index.html does not contain an element with id=%q, got: %s", id, html)
	}
	return match
}
