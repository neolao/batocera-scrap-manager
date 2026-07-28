package webui

import (
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// sonicEditURL is the edit page of the fully scraped game built by
// fullyScrapedRegistry.
const sonicEditURL = "/game/megadrive/Sonic%20the%20Hedgehog/edit"

// inputValue returns the value attribute of the <input> or <textarea> named
// name, unescaped, reporting whether such a control was found at all.
func inputValue(t *testing.T, body, name string) (string, bool) {
	t.Helper()

	input := regexp.MustCompile(`<input[^>]*name="` + regexp.QuoteMeta(name) + `"[^>]*>`)
	if tag := input.FindString(body); tag != "" {
		value := regexp.MustCompile(`value="([^"]*)"`).FindStringSubmatch(tag)
		if value == nil {
			return "", true
		}
		return html.UnescapeString(value[1]), true
	}

	textarea := regexp.MustCompile(`<textarea[^>]*name="` + regexp.QuoteMeta(name) + `"[^>]*>([^<]*)</textarea>`)
	if match := textarea.FindStringSubmatch(body); match != nil {
		return html.UnescapeString(match[1]), true
	}
	return "", false
}

// selectedOption returns the value of the selected <option> of the <select>
// named name.
func selectedOption(t *testing.T, body, name string) string {
	t.Helper()

	block := regexp.MustCompile(`<select[^>]*name="` + regexp.QuoteMeta(name) + `"[^>]*>(.*?)</select>`)
	match := block.FindStringSubmatch(strings.ReplaceAll(body, "\n", ""))
	if match == nil {
		t.Fatalf("no <select> named %q in the page", name)
	}
	selected := regexp.MustCompile(`<option value="([^"]*)"[^>]*\bselected\b`).FindStringSubmatch(match[1])
	if selected == nil {
		return ""
	}
	return html.UnescapeString(selected[1])
}

// optionValues returns every option value of the <select> named name.
func optionValues(t *testing.T, body, name string) []string {
	t.Helper()

	block := regexp.MustCompile(`<select[^>]*name="` + regexp.QuoteMeta(name) + `"[^>]*>(.*?)</select>`)
	match := block.FindStringSubmatch(strings.ReplaceAll(body, "\n", ""))
	if match == nil {
		t.Fatalf("no <select> named %q in the page", name)
	}

	var values []string
	for _, option := range regexp.MustCompile(`<option value="([^"]*)"`).FindAllStringSubmatch(match[1], -1) {
		values = append(values, html.UnescapeString(option[1]))
	}
	return values
}

func TestHandler_EditPage_PrefillsEveryEditableField(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, registryFolder, nil), sonicEditURL)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []struct{ field, value string }{
		{"name", "Sonic the Hedgehog"},
		{"desc", "A blue hedgehog runs very fast through Green Hill Zone."},
		{"developer", "Sonic Team"},
		{"publisher", "Sega"},
		{"genre", "Platform"},
		{"players", "1"},
	} {
		got, found := inputValue(t, body, want.field)
		if !found {
			t.Errorf("the edit form has no control named %q", want.field)
			continue
		}
		if got != want.value {
			t.Errorf("field %q = %q, want %q", want.field, got, want.value)
		}
	}
}

func TestHandler_EditPage_Rating_PreselectsTheStarCountThePageDisplays(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder, nil), sonicEditURL).Body.String()

	if got := selectedOption(t, body, "rating"); got != "4" {
		t.Errorf("selected rating = %q, want %q: a stored 0.8 is displayed as 4 stars out of 5", got, "4")
	}
	for _, value := range optionValues(t, body, "rating") {
		if strings.Contains(value, ".") {
			t.Errorf("the rating control offers %q, want whole star counts, not the stored decimal", value)
		}
	}
}

func TestHandler_EditPage_ReleaseDate_PrefillsTheYearAlone(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder, nil), sonicEditURL).Body.String()

	got, found := inputValue(t, body, "year")
	if !found {
		t.Fatal("the edit form has no year control")
	}
	if got != "1991" {
		t.Errorf("year = %q, want %q", got, "1991")
	}
	if strings.Contains(body, "19910623T000000") {
		t.Error("the edit form exposes the stored release date, want the year it is displayed as")
	}
}

func TestHandler_EditPage_GameWithNoRatingNorGenre_RendersEmptyControlsNotPlaceholders(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic"}},
	}}

	body := get(t, Handler(reg, registryFolder, nil), "/game/megadrive/Sonic/edit").Body.String()

	if got, _ := inputValue(t, body, "genre"); got != "" {
		t.Errorf("genre = %q, want an empty control for a game that has no genre", got)
	}
	if got, _ := inputValue(t, body, "year"); got != "" {
		t.Errorf("year = %q, want an empty control for a game that has no release date", got)
	}
	if got := selectedOption(t, body, "rating"); got != "" {
		t.Errorf("selected rating = %q, want the not-rated option", got)
	}
	if strings.Contains(body, "&mdash;") {
		t.Error("the edit form renders the read-only em-dash placeholder inside a control")
	}
}

func TestHandler_EditPage_NeverExposesTheMedia(t *testing.T) {
	// The ROM path is editable since backlog item 021 — it identifies the
	// entry, and correcting it is a repair. Media are not: they are managed by
	// their own flows rather than typed in.
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder, nil), sonicEditURL).Body.String()

	for _, forbidden := range []string{"image", "video", "marquee", "thumbnail"} {
		if _, found := inputValue(t, body, forbidden); found {
			t.Errorf("the edit form carries a control named %q, want the media kept out of it", forbidden)
		}
	}
}

func TestHandler_EditPage_HandEditedField_OffersToHandItBackToTheScraper(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{{
		System:       "megadrive",
		Game:         gamelist.Game{Path: "./Sonic.zip", Name: "Sonic", Genre: "Platform"},
		ManualFields: []string{"genre"},
	}}}

	body := get(t, Handler(reg, registryFolder, nil), "/game/megadrive/Sonic/edit").Body.String()

	handBack := regexp.MustCompile(`<input[^>]*name="hand_back"[^>]*value="([^"]*)"`).FindAllStringSubmatch(body, -1)
	if len(handBack) != 1 {
		t.Fatalf("found %d hand-back controls, want exactly one (for the genre)", len(handBack))
	}
	if handBack[0][1] != "genre" {
		t.Errorf("hand-back control = %q, want it to target the genre", handBack[0][1])
	}
}

func TestHandler_EditPage_FieldNobodyEdited_HasNoHandBackControl(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder, nil), sonicEditURL).Body.String()

	if strings.Contains(body, `name="hand_back"`) {
		t.Error("the edit form offers to hand a field back to the scraper, want none: nothing was corrected by hand")
	}
}

func TestHandler_EditPage_CancellingLeadsBackToTheGamePage(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	h := Handler(reg, registryFolder, nil)

	body := get(t, h, sonicEditURL).Body.String()

	gameURL := "/game/megadrive/Sonic%20the%20Hedgehog"
	if !strings.Contains(body, fmt.Sprintf(`href="%s"`, gameURL)) {
		t.Fatalf("the edit page has no link back to %s, got: %s", gameURL, body)
	}
	if rec := get(t, h, gameURL); rec.Code != http.StatusOK {
		t.Errorf("GET %s = %d, want 200", gameURL, rec.Code)
	}
}

func TestHandler_EditPage_UnknownGame_RendersTheNotFoundPage(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, registryFolder, nil), "/game/megadrive/Golden%20Axe/edit")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Golden Axe") {
		t.Errorf("the not-found page does not name the game that was not found, got: %s", rec.Body.String())
	}
}

func TestHandler_EditPage_KnownGameOfAnotherSystem_RendersTheNotFoundPage(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, registryFolder, nil), "/game/mastersystem/Sonic%20the%20Hedgehog/edit")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandler_EditURL_UnsupportedMethod_Returns405NamingTheAllowedOnes(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	rec := httptest.NewRecorder()

	Handler(reg, registryFolder, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodPut, sonicEditURL, nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
	allow := rec.Header().Get("Allow")
	if !strings.Contains(allow, http.MethodGet) || !strings.Contains(allow, http.MethodPost) {
		t.Errorf("Allow = %q, want it to list GET and POST", allow)
	}
}

func TestHandler_GamePage_LinksToItsEditPage(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	h := Handler(reg, registryFolder, nil)

	body := get(t, h, "/game/megadrive/Sonic%20the%20Hedgehog").Body.String()

	if !strings.Contains(body, fmt.Sprintf(`href="%s"`, "/game/megadrive/Sonic%20the%20Hedgehog/edit")) {
		t.Fatalf("the game page has no link to its edit page, got: %s", body)
	}
	if rec := get(t, h, sonicEditURL); rec.Code != http.StatusOK {
		t.Errorf("GET the linked edit page = %d, want 200", rec.Code)
	}
}

func TestHandler_GamePage_HandEditedValue_IsMarkedAsSuch(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{{
		System:       "megadrive",
		Game:         gamelist.Game{Path: "./Sonic.zip", Name: "Sonic", Genre: "Platform", Publisher: "Sega"},
		ManualFields: []string{"genre"},
	}}}

	body := get(t, Handler(reg, registryFolder, nil), "/game/megadrive/Sonic").Body.String()

	marks := strings.Count(body, `class="meta__manual"`)
	if marks != 1 {
		t.Errorf("found %d hand-edited marks on the game page, want exactly one (the genre): %s", marks, body)
	}
}
