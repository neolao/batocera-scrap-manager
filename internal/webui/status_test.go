package webui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// registryAndRomsFolderForStatus builds a registry holding one game the local
// gamelist.xml is missing a description for, plus a ROM file with no local
// entry and no registry match either — one fillable gap, one that needs real
// scraping.
func registryAndRomsFolderForStatus(t *testing.T) (*registry.Registry, string, string) {
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
	if err := os.MkdirAll(megadrive, 0o755); err != nil {
		t.Fatalf("failed to set up the ROMs folder: %v", err)
	}
	for _, name := range []string{"Golden Axe.zip", "Ghost.zip"} {
		if err := os.WriteFile(filepath.Join(megadrive, name), []byte("rom"), 0o644); err != nil {
			t.Fatalf("write rom %s: %v", name, err)
		}
	}
	games := []gamelist.Game{{Path: "./Golden Axe.zip", Name: "Golden Axe"}}
	if err := gamelist.UpdateFile(filepath.Join(megadrive, "gamelist.xml"), games); err != nil {
		t.Fatalf("write gamelist: %v", err)
	}

	return reg, registryFolder, romsFolder
}

func TestServeRomsFolderStatus_NominalFixture_ShowsCountsAndTellsFillableFromStuck(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)

	body := get(t, Handler(reg, registryFolder, []string{romsFolder}), romsFolderStatusURL+"?"+statusFolderParam+"="+romsFolder).Body.String()

	if !strings.Contains(body, "megadrive") {
		t.Error("the page does not name the system")
	}
	if !strings.Contains(body, "Golden Axe") {
		t.Error("the page does not name the incomplete game")
	}
	if !strings.Contains(body, "A Completion would already fill") {
		t.Errorf("the page does not say Golden Axe is fillable from the registry\n--- page ---\n%s", body)
	}
	if !strings.Contains(body, "Ghost.zip") {
		t.Error("the page does not name the ROM with no local entry")
	}
	if !strings.Contains(body, "Needs real scraping") {
		t.Errorf("the page does not say Ghost.zip needs real scraping\n--- page ---\n%s", body)
	}
}

func TestServeRomsFolderStatus_GapKnownToTheRegistry_LinksToItsPage(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)

	body := get(t, Handler(reg, registryFolder, []string{romsFolder}), romsFolderStatusURL+"?"+statusFolderParam+"="+romsFolder).Body.String()

	want := gameURL("megadrive", "Golden Axe")
	if !strings.Contains(body, `href="`+want+`"`) {
		t.Errorf("the page does not link Golden Axe to its registry page (%s)\n--- page ---\n%s", want, body)
	}
}

func TestServeRomsFolderStatus_GapUnknownToTheRegistry_HasNoRegistryLink(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)

	body := get(t, Handler(reg, registryFolder, []string{romsFolder}), romsFolderStatusURL+"?"+statusFolderParam+"="+romsFolder).Body.String()

	unwanted := gameURL("megadrive", "Ghost")
	if strings.Contains(body, `href="`+unwanted+`"`) {
		t.Errorf("the page links Ghost.zip to a registry page it has none of (%s)\n--- page ---\n%s", unwanted, body)
	}
}

func TestServeRomsFolderStatus_FolderNotConfigured_IsRefused(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, registryFolder, twoRomsFolders(t)),
		romsFolderStatusURL+"?"+statusFolderParam+"=/not/configured")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for an unconfigured folder", rec.Code, http.StatusBadRequest)
	}
}

func TestServeRomsFolderStatus_NoFolderParam_IsRefused(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, registryFolder, twoRomsFolders(t)), romsFolderStatusURL)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for a missing folder parameter", rec.Code, http.StatusBadRequest)
	}
}

func TestServeRomsFolderStatus_ConfiguredFolderGoneFromDisk_RendersAClearError(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	missing := filepath.Join(t.TempDir(), "was-here")

	rec := get(t, Handler(reg, registryFolder, []string{missing}),
		romsFolderStatusURL+"?"+statusFolderParam+"="+missing)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d for a configured folder that cannot be read", rec.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(rec.Body.String(), "could not be read") {
		t.Error("the page does not say the folder could not be read")
	}
}

func TestRomsFolderStatus_MethodNeitherGet_AnswersMethodNotAllowed(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	handler := Handler(reg, registryFolder, twoRomsFolders(t))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, romsFolderStatusURL, nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestServeHome_RomsFoldersConfigured_LinksToEachOnesStatus(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	romsFolders := twoRomsFolders(t)

	body := get(t, Handler(reg, registryFolder, romsFolders), "/").Body.String()

	for _, folder := range romsFolders {
		if !strings.Contains(body, folder) {
			t.Errorf("the home page does not name the ROMs folder %q", folder)
		}
		if !strings.Contains(body, romsFolderStatusURL+"?"+statusFolderParam+"=") {
			t.Error("the home page does not link to a ROMs folder's status page")
		}
	}
}

func TestServeHome_NoRomsFolderConfigured_ListsNoStatusLink(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder, nil), "/").Body.String()

	if strings.Contains(body, romsFolderStatusURL) {
		t.Error("the home page links to a ROMs folder status page with none configured")
	}
}
