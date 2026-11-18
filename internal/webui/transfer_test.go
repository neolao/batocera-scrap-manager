package webui

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// registryAndRomsFolderForTransfer builds a registry and a ROMs folder with
// four megadrive games covering every case a transfer must tell apart:
//   - "Streets.zip" has a local entry with real scraped data (a description
//     and a cover art file actually on disk) the registry does not know at
//     all yet — what copying into the registry is for.
//   - "Golden Axe.zip" is already identical between the folder and the
//     registry — nothing to copy either way.
//   - "Ghost.zip" has a local entry but neither a description nor a cover
//     art — nothing scraped worth keeping.
//   - "Unscraped.zip" has no local entry at all — not a candidate for
//     copying into the registry, only for real scraping.
func registryAndRomsFolderForTransfer(t *testing.T) (*registry.Registry, string, string) {
	t.Helper()

	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{
			Path: "./Golden Axe.zip", Name: "Golden Axe",
			Desc: "Two warriors against an evil sorcerer.",
		}},
	}}
	registryFolder := t.TempDir()
	if err := registry.Save(registryFolder, reg); err != nil {
		t.Fatalf("failed to write the registry: %v", err)
	}

	romsFolder := t.TempDir()
	megadrive := filepath.Join(romsFolder, "megadrive")
	if err := os.MkdirAll(filepath.Join(megadrive, "images"), 0o755); err != nil {
		t.Fatalf("failed to set up the ROMs folder: %v", err)
	}
	for _, name := range []string{"Streets.zip", "Golden Axe.zip", "Ghost.zip", "Unscraped.zip"} {
		if err := os.WriteFile(filepath.Join(megadrive, name), []byte("rom"), 0o644); err != nil {
			t.Fatalf("write rom %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(megadrive, "images", "streets.png"), []byte("cover"), 0o644); err != nil {
		t.Fatalf("write cover art: %v", err)
	}
	games := []gamelist.Game{
		{Path: "./Streets.zip", Name: "Streets of Rage", Desc: "Three fighters clean up the city.", Image: "./images/streets.png"},
		{Path: "./Golden Axe.zip", Name: "Golden Axe", Desc: "Two warriors against an evil sorcerer."},
		{Path: "./Ghost.zip", Name: "Ghost"},
	}
	if err := gamelist.UpdateFile(filepath.Join(megadrive, "gamelist.xml"), games); err != nil {
		t.Fatalf("write gamelist: %v", err)
	}

	return reg, registryFolder, romsFolder
}

// statusBannerAfter follows a change's redirect back to the status page and
// returns the text of its confirmation banner.
func statusBannerAfter(t *testing.T, h http.Handler, rec *httptest.ResponseRecorder) string {
	t.Helper()

	location, _, _ := strings.Cut(rec.Header().Get("Location"), "#")
	if location == "" {
		t.Fatalf("the change answered no redirect (status %d)", rec.Code)
	}
	body := get(t, h, location).Body.String()

	_, afterID, found := strings.Cut(body, `id="changed"`)
	if !found {
		t.Fatalf("the status page carries no confirmation banner\n--- page ---\n%s", body)
	}
	banner, _, _ := strings.Cut(afterID, "</p>")
	_, text, _ := strings.Cut(banner, ">")
	return text
}

// transferChoices is what the review form submits.
func transferChoices(folder, rom, direction string, take ...string) url.Values {
	return url.Values{
		statusFolderParam: {folder},
		gameSystemParam:   {"megadrive"},
		gameROMParam:      {rom},
		directionParam:    {direction},
		takeParam:         take,
	}
}

func TestServeRomsFolderSystemStatus_OffersEachDirectionOnlyWhenItsSourceExists(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForTransfer(t)

	body := html.UnescapeString(get(t, Handler(reg, registryFolder, []string{romsFolder}), romsFolderSystemStatusURLFor(romsFolder, "megadrive", 1)).Body.String())

	cases := []struct {
		rom       string
		direction string
		offered   bool
	}{
		{"Streets.zip", directionToRegistry, true},
		{"Streets.zip", directionToFolder, false},
		{"Golden Axe.zip", directionToRegistry, true},
		{"Golden Axe.zip", directionToFolder, true},
		{"Unscraped.zip", directionToRegistry, false},
	}
	for _, c := range cases {
		link := transferURLFor(romsFolder, "megadrive", c.rom, c.direction)
		if strings.Contains(body, link) != c.offered {
			t.Errorf("%s %s: offered = %v, want %v", c.rom, c.direction, !c.offered, c.offered)
		}
	}
	if strings.Contains(body, "Retrieve into the registry") {
		t.Error("the page still offers the former, unclear retrieve button")
	}
}

func TestServeTransfer_ToRegistry_ComparesEachValueAndTicksOnlyGaps(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForTransfer(t)
	reg.Entries[0].Game.Developer = "Sega"
	local := []gamelist.Game{{Path: "./Golden Axe.zip", Name: "Golden Axe", Desc: "Another story.", Genre: "Beat 'em up", Developer: "Sega"}}
	writeSystemGames(t, romsFolder, "megadrive", local)

	rec := get(t, Handler(reg, registryFolder, []string{romsFolder}), transferURLFor(romsFolder, "megadrive", "Golden Axe.zip", directionToRegistry))

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\n%s", rec.Code, body)
	}
	if !strings.Contains(body, `name="`+takeParam+`" value="genre" checked`) {
		t.Error("the genre, missing from the registry, is not ticked")
	}
	if !strings.Contains(body, `name="`+takeParam+`" value="desc">`) {
		t.Error("the description, which would overwrite the registry's, is not offered unticked")
	}
	if strings.Contains(body, `value="name"`) || strings.Contains(body, `value="developer"`) {
		t.Error("an identical value is offered as a choice")
	}
	if !strings.Contains(body, "Two warriors against an evil sorcerer.") || !strings.Contains(body, "Another story.") {
		t.Error("the review does not show both descriptions side by side")
	}
}

func TestTransfer_ToRegistry_CopiesTickedValuesAndMediaAndCreatesTheGame(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForTransfer(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	rec := post(t, handler, transferURL, transferChoices(romsFolder, "Streets.zip", directionToRegistry, "name", "desc", "image"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	reloaded, _ := registry.Load(registryFolder)
	entry, found := reloaded.FindByID("megadrive", "Streets")
	if !found || entry.Game.Desc != "Three fighters clean up the city." || entry.Game.Image == "" {
		t.Fatalf("entry = %+v (found %v), want Streets of Rage with its description and cover art", entry, found)
	}
	data, err := os.ReadFile(filepath.Join(registryFolder, "megadrive", filepath.FromSlash(entry.Game.Image)))
	if err != nil || string(data) != "cover" {
		t.Errorf("the cover art was not copied into the registry (err %v)", err)
	}
	banner := statusBannerAfter(t, handler, rec)
	if !strings.Contains(banner, "Streets of Rage") || !strings.Contains(banner, "registry") {
		t.Errorf("the confirmation %q does not say what was copied where", banner)
	}
}

func TestTransfer_ToFolder_WritesOnlyTickedValuesIntoTheGamelist(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForTransfer(t)
	reg.Entries[0].Game.Publisher = "Sega"
	reg.Entries[0].Game.Genre = "Beat 'em up"
	handler := Handler(reg, registryFolder, []string{romsFolder})

	rec := post(t, handler, transferURL, transferChoices(romsFolder, "Golden Axe.zip", directionToFolder, "publisher"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	local, err := registry.LocalGame(romsFolder, "megadrive", "Golden Axe.zip")
	if err != nil || local.Publisher != "Sega" || local.Genre != "" {
		t.Errorf("local entry = %+v (%v), want the publisher only", local, err)
	}
	if banner := statusBannerAfter(t, handler, rec); !strings.Contains(banner, "folder") {
		t.Errorf("the confirmation %q does not say the folder was written", banner)
	}
}

func TestServeTransfer_SourceMissingOrBadRequest_IsRefused(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForTransfer(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	for _, target := range []string{
		transferURLFor(romsFolder, "megadrive", "Streets.zip", directionToFolder),
		transferURLFor(romsFolder, "megadrive", "Unscraped.zip", directionToRegistry),
		transferURLFor(romsFolder, "megadrive", "Nowhere.zip", directionToRegistry),
		transferURLFor(romsFolder, "megadrive", "Golden Axe.zip", "sideways"),
		transferURLFor("/not/configured", "megadrive", "Golden Axe.zip", directionToRegistry),
	} {
		if rec := get(t, handler, target); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", target, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestTransfer_NothingTicked_IsRefusedAndWritesNothing(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForTransfer(t)

	rec := post(t, Handler(reg, registryFolder, []string{romsFolder}), transferURL, transferChoices(romsFolder, "Streets.zip", directionToRegistry))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	if _, found := reg.FindByID("megadrive", "Streets"); found {
		t.Error("an empty submission still created the game")
	}
}

func TestTransfer_CrossSiteSubmission_IsRefused(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForTransfer(t)
	form := transferChoices(romsFolder, "Streets.zip", directionToRegistry, "name")
	req := httptest.NewRequest(http.MethodPost, transferURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()

	Handler(reg, registryFolder, []string{romsFolder}).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestTransfer_ToRegistry_RegistryCannotBeWritten_KeepsTheServedRegistry(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForTransfer(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})
	systemFolder := filepath.Join(registryFolder, "megadrive")
	for _, folder := range []string{registryFolder, systemFolder} {
		if err := os.MkdirAll(folder, 0o700); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.Chmod(folder, 0o500); err != nil {
			t.Fatalf("chmod: %v", err)
		}
	}
	t.Cleanup(func() {
		os.Chmod(registryFolder, 0o700)
		os.Chmod(systemFolder, 0o700)
	})

	rec := post(t, handler, transferURL, transferChoices(romsFolder, "Streets.zip", directionToRegistry, "name", "desc"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	page := html.UnescapeString(get(t, handler, romsFolderSystemStatusURLFor(romsFolder, "megadrive", 1)).Body.String())
	if strings.Contains(page, transferURLFor(romsFolder, "megadrive", "Streets.zip", directionToFolder)) {
		t.Error("the served registry holds a game that was never written to disk")
	}
}
