package webui

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// fillSubmission is what the fill-in form submits for rom of megadrive in
// folder, with name and year as entered.
func fillSubmission(folder, rom, name, year string) url.Values {
	return url.Values{
		statusFolderParam:   {folder},
		retrieveSystemParam: {"megadrive"},
		retrieveROMParam:    {rom},
		"name":              {name},
		"desc":              {"Typed by hand."},
		"rating":            {"4"},
		"year":              {year},
		"developer":         {"Someone"},
	}
}

func TestServeRomsFolderSystemStatus_GameUnknownToRegistry_OffersFillIn(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)

	body := get(t, Handler(reg, registryFolder, []string{romsFolder}), romsFolderSystemStatusURLFor(romsFolder, "megadrive", 1)).Body.String()

	want := html.EscapeString(fillURLFor(romsFolder, "megadrive", "Ghost.zip"))
	if !strings.Contains(body, `href="`+want+`"`) {
		t.Errorf("the page offers no fill-in link for Ghost.zip (%s)\n--- page ---\n%s", want, body)
	}
	unwanted := html.EscapeString(fillURLFor(romsFolder, "megadrive", "Golden Axe.zip"))
	if strings.Contains(body, unwanted) {
		t.Error("the page offers a fill-in link for Golden Axe, which the registry already knows")
	}
}

func TestServeFillForm_GameUnknownToRegistry_RendersAnEmptyFormNamingTheRom(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)

	rec := get(t, Handler(reg, registryFolder, []string{romsFolder}), fillURLFor(romsFolder, "megadrive", "Ghost.zip"))

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\n%s", rec.Code, body)
	}
	if !strings.Contains(body, "Ghost.zip") || !strings.Contains(body, `name="name"`) || !strings.Contains(body, `name="year"`) {
		t.Errorf("the form does not name the ROM or lacks the metadata controls\n--- page ---\n%s", body)
	}
}

func TestServeFillForm_GameAlreadyInRegistryOrUnknownRom_IsRefused(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	for _, target := range []string{
		fillURLFor(romsFolder, "megadrive", "Golden Axe.zip"),
		fillURLFor(romsFolder, "megadrive", "Nowhere.zip"),
		fillURLFor(romsFolder, "snes", "Ghost.zip"),
		fillURLFor("/not/configured", "megadrive", "Ghost.zip"),
	} {
		if rec := get(t, handler, target); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", target, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestFillGame_ValidSubmission_CreatesTheGameMarkedHandEditedAndConfirms(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	rec := post(t, handler, fillURL, fillSubmission(romsFolder, "Ghost.zip", "Ghost Story", "1992"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	reloaded, err := registry.Load(registryFolder)
	if err != nil {
		t.Fatalf("failed to reload the registry: %v", err)
	}
	entry, found := reloaded.FindByID("megadrive", "Ghost")
	if !found {
		t.Fatal("Ghost.zip was not created in the registry")
	}
	if entry.Game.Path != "./Ghost.zip" || entry.Game.Name != "Ghost Story" || entry.Game.Desc != "Typed by hand." {
		t.Errorf("entry = %+v, want path ./Ghost.zip, the typed name and description", entry.Game)
	}
	if entry.Game.ReleaseDate != "19920101T000000" || entry.Game.Rating != "0.8" {
		t.Errorf("release date / rating = %q / %q, want 19920101T000000 / 0.8", entry.Game.ReleaseDate, entry.Game.Rating)
	}
	if !slices.Contains(entry.ManualFields, registry.FieldName) || !slices.Contains(entry.ManualFields, registry.FieldDesc) {
		t.Errorf("ManualFields = %v, want the typed fields marked hand-edited", entry.ManualFields)
	}
	if slices.Contains(entry.ManualFields, registry.FieldPublisher) {
		t.Errorf("ManualFields = %v, want the publisher left empty and unmarked", entry.ManualFields)
	}

	banner := statusBannerAfter(t, handler, rec)
	if !strings.Contains(banner, "Ghost Story") || !strings.Contains(banner, "created") {
		t.Errorf("the confirmation %q does not say Ghost Story was created", banner)
	}
	location, _, _ := strings.Cut(rec.Header().Get("Location"), "#")
	page := get(t, handler, location).Body.String()
	if !strings.Contains(html.UnescapeString(page), sendURLFor("megadrive", "Ghost", romsFolder)) {
		t.Error("after filling in, the system page does not offer to send the game to this folder")
	}
}

func TestFillGame_InvalidValues_ShowsTheFormAgainAndWritesNothing(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	for _, submission := range []url.Values{
		fillSubmission(romsFolder, "Ghost.zip", "", "1992"),
		fillSubmission(romsFolder, "Ghost.zip", "Ghost Story", "1800"),
	} {
		rec := post(t, handler, fillURL, submission)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
		}
		if !strings.Contains(rec.Body.String(), "Typed by hand.") {
			t.Error("the refused form does not keep what was typed")
		}
	}
	reloaded, _ := registry.Load(registryFolder)
	if _, found := reloaded.FindByID("megadrive", "Ghost"); found {
		t.Error("a refused submission created the game anyway")
	}
}

func TestFillGame_GameAlreadyInRegistryOrForeignOrigin_IsRefusedAndWritesNothing(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	rec := post(t, handler, fillURL, fillSubmission(romsFolder, "Golden Axe.zip", "Overwritten", "1992"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("game already known: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	rec = post(t, handler, fillURL, fillSubmission("/not/configured", "Ghost.zip", "Ghost", "1992"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("unconfigured folder: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	form := fillSubmission(romsFolder, "Ghost.zip", "Ghost", "1992")
	req := httptest.NewRequest(http.MethodPost, fillURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	crossRec := httptest.NewRecorder()
	handler.ServeHTTP(crossRec, req)
	if code := crossRec.Code; code != http.StatusForbidden {
		t.Errorf("cross-site submission: status = %d, want %d", code, http.StatusForbidden)
	}

	reloaded, _ := registry.Load(registryFolder)
	if entry, _ := reloaded.FindByID("megadrive", "Golden Axe"); entry.Game.Name != "Golden Axe" {
		t.Errorf("Golden Axe was renamed to %q", entry.Game.Name)
	}
	if _, found := reloaded.FindByID("megadrive", "Ghost"); found {
		t.Error("a refused submission created Ghost anyway")
	}
}

func TestFillGame_RegistryCannotBeWritten_ShowsTheProblemAndKeepsTheServedRegistry(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
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

	rec := post(t, handler, fillURL, fillSubmission(romsFolder, "Ghost.zip", "Ghost Story", "1992"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	page := get(t, handler, romsFolderSystemStatusURLFor(romsFolder, "megadrive", 1)).Body.String()
	if strings.Contains(html.UnescapeString(page), sendURLFor("megadrive", "Ghost", romsFolder)) {
		t.Error("the served registry holds a game that was never written to disk")
	}
}
