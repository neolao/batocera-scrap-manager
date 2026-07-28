package webui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/store"
)

const (
	sonicGameURL = "/game/megadrive/Sonic"
	sonicSaveURL = "/game/megadrive/Sonic/edit"
)

// savedRegistry builds a registry holding one fully scraped game and writes
// it to disk (game files and consultation site), the way it would be after an
// update — so a test can compare what a save changed on disk.
func savedRegistry(t *testing.T) (*registry.Registry, string) {
	t.Helper()
	folder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{{
		System: "megadrive",
		Game: gamelist.Game{
			Path: "./Sonic.zip", Name: "Sonic the Hedghog", Desc: "Fast.",
			Image: "images/sonic.png", Video: "videos/sonic.mp4",
			Marquee: "images/marquee.png", Thumbnail: "images/thumb.png",
			Rating: "0.85", ReleaseDate: "19910623T000000",
			Developer: "Sonic Team", Publisher: "Sega", Genre: "Platform", Players: "1",
		},
	}}}
	if err := store.Save(reg, folder); err != nil {
		t.Fatalf("failed to write the test registry: %v", err)
	}
	return reg, folder
}

// storedValues is what the form carries when nothing was corrected: the
// stored game exactly as the form displays it.
func storedValues() url.Values {
	return url.Values{
		"path":      {"./Sonic.zip"},
		"name":      {"Sonic the Hedghog"},
		"desc":      {"Fast."},
		"rating":    {"4"},
		"year":      {"1991"},
		"developer": {"Sonic Team"},
		"publisher": {"Sega"},
		"genre":     {"Platform"},
		"players":   {"1"},
	}
}

// post submits form to target, as a browser on the same origin would.
func post(t *testing.T, h http.Handler, target string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

// gameFile reads the game's stored JSON file.
func gameFile(t *testing.T, folder string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(folder, "megadrive", "Sonic.json"))
	if err != nil {
		t.Fatalf("failed to read the game file: %v", err)
	}
	return string(data)
}

// indexFile reads the generated consultation site.
func indexFile(t *testing.T, folder string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(folder, "index.html"))
	if err != nil {
		t.Fatalf("failed to read the consultation site: %v", err)
	}
	return string(data)
}

func TestHandler_Save_Correction_RedirectsBackToTheGamePage(t *testing.T) {
	reg, folder := savedRegistry(t)
	form := storedValues()
	form.Set("name", "Sonic the Hedgehog")

	rec := post(t, Handler(reg, folder, nil), sonicSaveURL, form)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 (body: %s)", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, sonicGameURL) {
		t.Errorf("Location = %q, want it to lead back to %q", location, sonicGameURL)
	}
	if !strings.Contains(location, "saved") {
		t.Errorf("Location = %q, want it to carry the saved confirmation", location)
	}
}

func TestHandler_Save_Correction_IsVisibleOnThePageAndOnDisk(t *testing.T) {
	reg, folder := savedRegistry(t)
	h := Handler(reg, folder, nil)
	form := storedValues()
	form.Set("name", "Sonic the Hedgehog")
	form.Set("genre", "Platformer")

	post(t, h, sonicSaveURL, form)

	body := get(t, h, sonicGameURL).Body.String()
	if !strings.Contains(body, "Sonic the Hedgehog") || !strings.Contains(body, "Platformer") {
		t.Errorf("the reloaded game page does not show the corrections, got: %s", body)
	}
	stored := gameFile(t, folder)
	if !strings.Contains(stored, "Sonic the Hedgehog") || !strings.Contains(stored, "Platformer") {
		t.Errorf("the game file does not hold the corrections, got: %s", stored)
	}
	if !strings.Contains(indexFile(t, folder), "Sonic the Hedgehog") {
		t.Error("the consultation site was not regenerated with the correction")
	}
}

func TestHandler_Save_MarksOnlyTheCorrectedFieldsAsHandEdited(t *testing.T) {
	reg, folder := savedRegistry(t)
	form := storedValues()
	form.Set("genre", "Platformer")

	post(t, Handler(reg, folder, nil), sonicSaveURL, form)

	stored := gameFile(t, folder)
	if !strings.Contains(stored, `"genre"`) || !strings.Contains(stored, "manual_fields") {
		t.Fatalf("the game file does not mark anything as hand-edited, got: %s", stored)
	}
	for _, untouched := range []string{`"name"`, `"desc"`, `"players"`} {
		if strings.Contains(marksOf(t, stored), untouched) {
			t.Errorf("manual_fields = %s, want only the genre", marksOf(t, stored))
		}
	}
}

// marksOf extracts the manual_fields block of a stored game file.
func marksOf(t *testing.T, stored string) string {
	t.Helper()
	i := strings.Index(stored, "manual_fields")
	if i == -1 {
		return ""
	}
	return stored[i:]
}

func TestHandler_Save_NothingCorrected_LeavesTheGameFileByteIdentical(t *testing.T) {
	// Opening the form and saving must not degrade the stored rating to the
	// star count it displays as, nor invent a month and a day for the release
	// date, nor mark anything.
	reg, folder := savedRegistry(t)
	before := gameFile(t, folder)

	post(t, Handler(reg, folder, nil), sonicSaveURL, storedValues())

	if after := gameFile(t, folder); after != before {
		t.Errorf("the game file changed:\nbefore: %s\nafter:  %s", before, after)
	}
}

func TestHandler_Save_Correction_LeavesTheRomPathAndTheMediaUntouched(t *testing.T) {
	reg, folder := savedRegistry(t)
	form := storedValues()
	form.Set("name", "Anything else")

	post(t, Handler(reg, folder, nil), sonicSaveURL, form)

	stored := gameFile(t, folder)
	for _, want := range []string{
		`"path": "./Sonic.zip"`,
		`"image": "images/sonic.png"`,
		`"video": "videos/sonic.mp4"`,
		`"marquee": "images/marquee.png"`,
		`"thumbnail": "images/thumb.png"`,
	} {
		if !strings.Contains(stored, want) {
			t.Errorf("the game file no longer holds %s, got: %s", want, stored)
		}
	}
}

func TestHandler_Save_ClearedFields_ShowTheirPlaceholderOnReload(t *testing.T) {
	reg, folder := savedRegistry(t)
	h := Handler(reg, folder, nil)
	form := storedValues()
	form.Set("desc", "")
	form.Set("rating", "")

	post(t, h, sonicSaveURL, form)

	body := get(t, h, sonicGameURL).Body.String()
	if !strings.Contains(body, "No description available.") {
		t.Errorf("the game page still shows a description, want the placeholder, got: %s", body)
	}
	if strings.Contains(body, "★") {
		t.Error("the game page still shows stars, want the rating cleared")
	}
}

func TestHandler_Save_FieldHandedBackToTheScraper_LosesItsMark(t *testing.T) {
	folder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{{
		System:       "megadrive",
		Game:         gamelist.Game{Path: "./Sonic.zip", Name: "Sonic", Genre: "Platformer"},
		ManualFields: []string{"genre"},
	}}}
	if err := store.Save(reg, folder); err != nil {
		t.Fatalf("failed to write the test registry: %v", err)
	}
	form := url.Values{"path": {"./Sonic.zip"}, "name": {"Sonic"}, "genre": {"Platformer"}, handBackParam: {"genre"}}

	post(t, Handler(reg, folder, nil), sonicSaveURL, form)

	if stored := gameFile(t, folder); strings.Contains(stored, "manual_fields") {
		t.Errorf("the game file still marks a field as hand-edited, got: %s", stored)
	}
}

func TestHandler_Save_EmptyName_RefusesAndKeepsWhatWasTyped(t *testing.T) {
	reg, folder := savedRegistry(t)
	before := gameFile(t, folder)
	form := storedValues()
	form.Set("name", "   ")
	form.Set("genre", "Typed but not saved")

	rec := post(t, Handler(reg, folder, nil), sonicSaveURL, form)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	body := rec.Body.String()
	if got, _ := inputValue(t, body, "genre"); got != "Typed but not saved" {
		t.Errorf("genre = %q, want the submitted value kept in the form", got)
	}
	if !strings.Contains(body, "errors") {
		t.Errorf("the refused form shows no error summary, got: %s", body)
	}
	if after := gameFile(t, folder); after != before {
		t.Error("the game file changed although the submission was refused")
	}
}

func TestHandler_Save_YearOutOfRange_Refuses(t *testing.T) {
	reg, folder := savedRegistry(t)
	before := gameFile(t, folder)
	form := storedValues()
	form.Set("year", "1291")

	rec := post(t, Handler(reg, folder, nil), sonicSaveURL, form)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if got, _ := inputValue(t, rec.Body.String(), "year"); got != "1291" {
		t.Errorf("year = %q, want the refused value kept in the form", got)
	}
	if after := gameFile(t, folder); after != before {
		t.Error("the game file changed although the submission was refused")
	}
}

func TestHandler_Save_RatingOutsideTheOfferedChoices_Refuses(t *testing.T) {
	reg, folder := savedRegistry(t)
	before := gameFile(t, folder)
	form := storedValues()
	form.Set("rating", "0.9")

	rec := post(t, Handler(reg, folder, nil), sonicSaveURL, form)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if after := gameFile(t, folder); after != before {
		t.Error("the game file changed although the submission was refused")
	}
}

func TestHandler_Save_UnknownGame_Returns404AndChangesNothing(t *testing.T) {
	reg, folder := savedRegistry(t)
	before := gameFile(t, folder)

	rec := post(t, Handler(reg, folder, nil), "/game/megadrive/Golden%20Axe/edit", storedValues())

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if after := gameFile(t, folder); after != before {
		t.Error("the game file changed although the game does not exist")
	}
}

func TestHandler_Save_GetOnTheSameURL_ChangesNothing(t *testing.T) {
	reg, folder := savedRegistry(t)
	before := gameFile(t, folder)

	get(t, Handler(reg, folder, nil), sonicSaveURL+"?name=Hacked")

	if after := gameFile(t, folder); after != before {
		t.Error("a GET on the edit URL modified the registry, want it read-only")
	}
}

func TestHandler_Save_SubmissionLargerThanAllowed_Returns400AndChangesNothing(t *testing.T) {
	reg, folder := savedRegistry(t)
	before := gameFile(t, folder)
	form := storedValues()
	form.Set("desc", strings.Repeat("a", maxFormBytes+1))

	rec := post(t, Handler(reg, folder, nil), sonicSaveURL, form)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if after := gameFile(t, folder); after != before {
		t.Error("the game file changed although the submission was too large")
	}
}

func TestHandler_Save_SubmissionFromAnotherSite_IsRefused(t *testing.T) {
	reg, folder := savedRegistry(t)
	before := gameFile(t, folder)
	form := storedValues()
	form.Set("name", "Hacked")

	r := httptest.NewRequest(http.MethodPost, sonicSaveURL, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	r.Header.Set("Origin", "http://evil.example")
	rec := httptest.NewRecorder()
	Handler(reg, folder, nil).ServeHTTP(rec, r)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if after := gameFile(t, folder); after != before {
		t.Error("a cross-site submission modified the registry")
	}
}

func TestHandler_Save_RegistryCannotBeWritten_SaysNothingWasSavedAndKeepsTheOldValue(t *testing.T) {
	reg, folder := savedRegistry(t)
	if err := os.RemoveAll(filepath.Join(folder, "megadrive")); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(folder, "megadrive"), []byte("blocker"), 0o644); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	h := Handler(reg, folder, nil)
	form := storedValues()
	form.Set("name", "Sonic the Hedgehog")

	rec := post(t, h, sonicSaveURL, form)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "not saved") {
		t.Errorf("the page does not say the correction was not saved, got: %s", rec.Body.String())
	}
	if body := get(t, h, sonicGameURL).Body.String(); !strings.Contains(body, "Sonic the Hedghog") {
		t.Error("the served page shows the correction although it could not be written to disk")
	}
}

func TestHandler_Save_SiteCannotBeRegenerated_StillConfirmsTheSaveAndSaysSo(t *testing.T) {
	reg, folder := savedRegistry(t)
	if err := os.Remove(filepath.Join(folder, "index.html")); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	if err := os.Mkdir(filepath.Join(folder, "index.html"), 0o755); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	h := Handler(reg, folder, nil)
	form := storedValues()
	form.Set("name", "Sonic the Hedgehog")

	rec := post(t, h, sonicSaveURL, form)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303: the registry itself was saved", rec.Code)
	}
	if !strings.Contains(gameFile(t, folder), "Sonic the Hedgehog") {
		t.Fatal("the game file does not hold the correction")
	}
	// A browser drops the fragment before requesting the redirect target.
	target, _, _ := strings.Cut(rec.Header().Get("Location"), "#")
	body := get(t, h, target).Body.String()
	if !strings.Contains(body, "consultation site") {
		t.Errorf("the game page does not warn that the consultation site is stale, got: %s", body)
	}
}

func TestHandler_GamePage_AfterASave_ShowsTheConfirmation(t *testing.T) {
	reg, folder := savedRegistry(t)
	h := Handler(reg, folder, nil)

	saved := get(t, h, sonicGameURL+"?saved=1").Body.String()
	plain := get(t, h, sonicGameURL).Body.String()

	if !strings.Contains(saved, "saved") {
		t.Errorf("the game page shows no confirmation after a save, got: %s", saved)
	}
	if strings.Contains(plain, `role="status"`) {
		t.Error("the game page shows a confirmation banner although nothing was just saved")
	}
}

func TestHandler_Save_ConcurrentWithReads_KeepsTheRegistryConsistent(t *testing.T) {
	// Run with -race: the served snapshot is shared by every request.
	reg, folder := savedRegistry(t)
	h := Handler(reg, folder, nil)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			get(t, h, "/")
			get(t, h, "/system/megadrive")
			get(t, h, sonicGameURL)
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		form := storedValues()
		form.Set("name", "Sonic the Hedgehog")
		post(t, h, sonicSaveURL, form)
	}()
	wg.Wait()

	if !strings.Contains(gameFile(t, folder), "Sonic the Hedgehog") {
		t.Error("the correction was lost")
	}
}
