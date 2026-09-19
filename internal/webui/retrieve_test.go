package webui

import (
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

// registryAndRomsFolderForRetrieve builds a registry and a ROMs folder with
// four megadrive games covering every case a retrieve must tell apart:
//   - "Streets.zip" has a local entry with real scraped data (a description
//     and a cover art file actually on disk) the registry does not know at
//     all yet — what a retrieve is for.
//   - "Golden Axe.zip" is already identical between the folder and the
//     registry — nothing for a retrieve to change.
//   - "Ghost.zip" has a local entry but neither a description nor a cover
//     art — nothing scraped worth keeping.
//   - "Unscraped.zip" has no local entry at all — not a candidate for
//     retrieval, only for real scraping.
func registryAndRomsFolderForRetrieve(t *testing.T) (*registry.Registry, string, string) {
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

// retrieveSubmission is what a gap row's form submits: the folder, the
// system and the ROM filename it targets.
func retrieveSubmission(folder, system, rom string) url.Values {
	return url.Values{
		statusFolderParam:   {folder},
		retrieveSystemParam: {system},
		retrieveROMParam:    {rom},
	}
}

// statusBannerAfter follows a retrieve's redirect back to the status page and
// returns the text of its confirmation banner.
func statusBannerAfter(t *testing.T, h http.Handler, rec *httptest.ResponseRecorder) string {
	t.Helper()

	location, _, _ := strings.Cut(rec.Header().Get("Location"), "#")
	if location == "" {
		t.Fatalf("the change answered no redirect (status %d)", rec.Code)
	}
	body := get(t, h, location).Body.String()

	_, afterID, found := strings.Cut(body, `id="retrieved"`)
	if !found {
		t.Fatalf("the status page carries no confirmation banner\n--- page ---\n%s", body)
	}
	banner, _, _ := strings.Cut(afterID, "</p>")
	_, text, _ := strings.Cut(banner, ">")
	return text
}

func TestRetrieveGame_ListedGapUnknownToRegistry_AddsItToTheRegistry(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForRetrieve(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	rec := post(t, handler, retrieveURL, retrieveSubmission(romsFolder, "megadrive", "Streets.zip"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	reloaded, err := registry.Load(registryFolder)
	if err != nil {
		t.Fatalf("failed to reload the registry: %v", err)
	}
	entry, found := reloaded.FindByID("megadrive", "Streets")
	if !found {
		t.Fatal("Streets of Rage was not added to the registry")
	}
	if entry.Game.Desc != "Three fighters clean up the city." {
		t.Errorf("Desc = %q, want the locally scraped description", entry.Game.Desc)
	}
	banner := statusBannerAfter(t, handler, rec)
	if !strings.Contains(banner, "Streets of Rage") {
		t.Errorf("the confirmation %q does not name the game that was added", banner)
	}
	if !strings.Contains(banner, "added") {
		t.Errorf("the confirmation %q does not say the game was added", banner)
	}
}

func TestRetrieveGame_ListedGapAlreadyMatchingTheRegistry_ReportsUnchanged(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForRetrieve(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	rec := post(t, handler, retrieveURL, retrieveSubmission(romsFolder, "megadrive", "Golden Axe.zip"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	banner := statusBannerAfter(t, handler, rec)
	if !strings.Contains(banner, "already matched") {
		t.Errorf("the confirmation %q does not say nothing changed", banner)
	}
}

func TestRetrieveGame_ListedGapWithNoScrapedDataLocally_ReportsNothingToRetrieveAndWritesNothing(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForRetrieve(t)
	before, err := os.ReadDir(registryFolder)
	if err != nil {
		t.Fatalf("failed to read the registry folder: %v", err)
	}
	handler := Handler(reg, registryFolder, []string{romsFolder})

	rec := post(t, handler, retrieveURL, retrieveSubmission(romsFolder, "megadrive", "Ghost.zip"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	banner := statusBannerAfter(t, handler, rec)
	if !strings.Contains(banner, "nothing to retrieve") {
		t.Errorf("the confirmation %q does not say there was nothing to retrieve", banner)
	}
	if _, found := reg.FindByID("megadrive", "Ghost"); found {
		t.Error("a game with no scraped data was added to the registry")
	}
	after, err := os.ReadDir(registryFolder)
	if err != nil {
		t.Fatalf("failed to read the registry folder: %v", err)
	}
	if len(after) != len(before) {
		t.Errorf("the registry folder gained files (%d -> %d) for a game with nothing to retrieve", len(before), len(after))
	}
}

func TestRetrieveGame_NoLocalEntry_OffersNoActionAndIsRefused(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForRetrieve(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	rec := post(t, handler, retrieveURL, retrieveSubmission(romsFolder, "megadrive", "Unscraped.zip"))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for a ROM with no local entry to retrieve from", rec.Code, http.StatusBadRequest)
	}
	if _, found := reg.FindByID("megadrive", "Unscraped"); found {
		t.Error("a game with no local entry was added to the registry")
	}
}

func TestRetrieveGame_MediaCannotBeWritten_ReportsFailureAndChangesNothing(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForRetrieve(t)
	// A file where the images folder should go, so copying the cover art fails.
	if err := os.MkdirAll(filepath.Join(registryFolder, "megadrive"), 0o755); err != nil {
		t.Fatalf("mkdir megadrive: %v", err)
	}
	if err := os.WriteFile(filepath.Join(registryFolder, "megadrive", "images"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
	handler := Handler(reg, registryFolder, []string{romsFolder})

	rec := post(t, handler, retrieveURL, retrieveSubmission(romsFolder, "megadrive", "Streets.zip"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	banner := statusBannerAfter(t, handler, rec)
	if !strings.Contains(banner, "could not be retrieved") {
		t.Errorf("the confirmation %q does not say the game could not be retrieved", banner)
	}
	if _, found := reg.FindByID("megadrive", "Streets"); found {
		t.Error("a failed retrieve still added the game to the registry")
	}
}

func TestRetrieveGame_DoubleSubmission_SecondReportsUnchanged(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForRetrieve(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	post(t, handler, retrieveURL, retrieveSubmission(romsFolder, "megadrive", "Streets.zip"))
	rec := post(t, handler, retrieveURL, retrieveSubmission(romsFolder, "megadrive", "Streets.zip"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	banner := statusBannerAfter(t, handler, rec)
	if !strings.Contains(banner, "already matched") {
		t.Errorf("a resubmitted retrieve should report no change, got %q", banner)
	}
}

func TestRetrieveGame_FolderNotConfigured_IsRefusedAndWritesNothing(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForRetrieve(t)

	rec := post(t, Handler(reg, registryFolder, nil), retrieveURL,
		retrieveSubmission(romsFolder, "megadrive", "Streets.zip"))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for a folder that is not configured", rec.Code, http.StatusBadRequest)
	}
	if _, found := reg.FindByID("megadrive", "Streets"); found {
		t.Error("retrieving from an unconfigured folder still wrote to the registry")
	}
}

func TestRetrieveGame_CrossSiteSubmission_IsRefusedAndWritesNothing(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForRetrieve(t)
	form := retrieveSubmission(romsFolder, "megadrive", "Streets.zip")
	r := httptest.NewRequest(http.MethodPost, retrieveURL, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()

	Handler(reg, registryFolder, []string{romsFolder}).ServeHTTP(rec, r)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if _, found := reg.FindByID("megadrive", "Streets"); found {
		t.Error("a cross-site submission wrote to the registry")
	}
}

func TestRetrieveGame_WrongMethod_IsRefusedNamingTheAllowedOne(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForRetrieve(t)

	rec := get(t, Handler(reg, registryFolder, []string{romsFolder}), retrieveURL)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodPost {
		t.Errorf("Allow = %q, want %q: this URL only ever accepts a submission", allow, http.MethodPost)
	}
}
