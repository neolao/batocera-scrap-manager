package webui

import (
	"fmt"
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

func TestServeRomsFolderSystemStatus_NominalFixture_TellsFillableFromStuck(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)

	body := get(t, Handler(reg, registryFolder, []string{romsFolder}), romsFolderSystemStatusURLFor(romsFolder, "megadrive", 1)).Body.String()

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

func TestServeRomsFolderSystemStatus_GapKnownToTheRegistry_LinksToItsPage(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)

	body := get(t, Handler(reg, registryFolder, []string{romsFolder}), romsFolderSystemStatusURLFor(romsFolder, "megadrive", 1)).Body.String()

	want := gameURL("megadrive", "Golden Axe")
	if !strings.Contains(body, `href="`+want+`"`) {
		t.Errorf("the page does not link Golden Axe to its registry page (%s)\n--- page ---\n%s", want, body)
	}
}

func TestServeRomsFolderSystemStatus_GapUnknownToTheRegistry_HasNoRegistryLink(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)

	body := get(t, Handler(reg, registryFolder, []string{romsFolder}), romsFolderSystemStatusURLFor(romsFolder, "megadrive", 1)).Body.String()

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

func TestServeRomsFolderStatus_Summary_ListsEachSystemWithItsCountsAndLinksToItsPage(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)

	body := get(t, Handler(reg, registryFolder, []string{romsFolder}), romsFolderStatusURLFor(romsFolder)).Body.String()

	want := html.EscapeString(romsFolderSystemStatusURLFor(romsFolder, "megadrive", 1))
	if !strings.Contains(body, `href="`+want+`"`) {
		t.Errorf("the summary does not link to megadrive's own page (%s)\n--- page ---\n%s", want, body)
	}
	if strings.Contains(body, "Ghost.zip") || strings.Contains(body, "Golden Axe") {
		t.Errorf("the summary lists games; it must only list systems\n--- page ---\n%s", body)
	}
}

func TestServeRomsFolderSystemStatus_CompleteGame_IsListedAsComplete(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	completeGame := gamelist.Game{Path: "./Sonic.zip", Name: "Sonic", Desc: "d", Image: "./images/s.png",
		Video: "./v.mp4", Marquee: "./m.png", Thumbnail: "./t.png", Rating: "0.5", ReleaseDate: "19910101T000000",
		Developer: "Sega", Publisher: "Sega", Genre: "Platform", Players: "1"}
	writeSystemGames(t, romsFolder, "megadrive", []gamelist.Game{completeGame, {Path: "./Golden Axe.zip", Name: "Golden Axe"}})

	body := get(t, Handler(reg, registryFolder, []string{romsFolder}), romsFolderSystemStatusURLFor(romsFolder, "megadrive", 1)).Body.String()

	if !strings.Contains(body, "Sonic") || !strings.Contains(body, "Complete") {
		t.Errorf("the complete game Sonic is not listed as complete\n--- page ---\n%s", body)
	}
}

func TestServeRomsFolderSystemStatus_MoreThanOnePageOfGames_IsPaginated(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	games := make([]gamelist.Game, gamesPerPage+5)
	for i := range games {
		games[i] = gamelist.Game{Path: fmt.Sprintf("./Game %03d.zip", i), Name: fmt.Sprintf("Game %03d", i)}
	}
	writeSystemGames(t, romsFolder, "nes", games)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	first := get(t, handler, romsFolderSystemStatusURLFor(romsFolder, "nes", 1)).Body.String()
	second := get(t, handler, romsFolderSystemStatusURLFor(romsFolder, "nes", 2)).Body.String()

	if !strings.Contains(first, "Game 000") || strings.Contains(first, "Game 060") {
		t.Error("page 1 does not hold exactly the first sixty games")
	}
	if !strings.Contains(second, "Game 060") || !strings.Contains(second, "Game 064") || strings.Contains(second, "Game 059") {
		t.Error("page 2 does not hold the remaining five games")
	}
	next := html.EscapeString(romsFolderSystemStatusURLFor(romsFolder, "nes", 2))
	if !strings.Contains(first, `href="`+next+`"`) {
		t.Errorf("page 1 does not link to page 2 (%s)", next)
	}
}

func TestServeRomsFolderSystemStatus_PageOutOfBounds_AnswersNotFound(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	for _, page := range []string{"2", "0", "abc"} {
		rec := get(t, handler, romsFolderSystemStatusURLFor(romsFolder, "megadrive", 1)+"&"+pageParam+"="+page)
		if rec.Code != http.StatusNotFound {
			t.Errorf("page=%s: status = %d, want %d", page, rec.Code, http.StatusNotFound)
		}
	}
}

func TestServeRomsFolderSystemStatus_UnknownOrUnsafeSystem_AnswersNotFound(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	for _, system := range []string{"snes", "..", "../megadrive"} {
		rec := get(t, handler, romsFolderSystemStatusURLFor(romsFolder, system, 1))
		if rec.Code != http.StatusNotFound {
			t.Errorf("system %q: status = %d, want %d", system, rec.Code, http.StatusNotFound)
		}
	}
}

func TestServeRomsFolderSystemStatus_FolderNotConfigured_IsRefused(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)

	rec := get(t, Handler(reg, registryFolder, []string{romsFolder}), romsFolderSystemStatusURLFor("/not/configured", "megadrive", 1))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServeRomsFolderSystemStatus_ShowsLocalAndRegistryCoverArtSideBySide(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	reg.Entries[0].Game.Image = "./images/golden-reg.png"
	writeFile(t, filepath.Join(registryFolder, "megadrive", "images", "golden-reg.png"), "reg")
	writeFile(t, filepath.Join(romsFolder, "megadrive", "images", "golden-local.png"), "local")
	writeSystemGames(t, romsFolder, "megadrive", []gamelist.Game{{Path: "./Golden Axe.zip", Name: "Golden Axe", Image: "./images/golden-local.png"}})

	body := get(t, Handler(reg, registryFolder, []string{romsFolder}), romsFolderSystemStatusURLFor(romsFolder, "megadrive", 1)).Body.String()

	local := html.EscapeString(romsFolderMediaURL(romsFolder, "megadrive", "./images/golden-local.png"))
	if !strings.Contains(body, `src="`+local+`"`) {
		t.Errorf("the page does not show the ROMs folder's cover art (%s)\n--- page ---\n%s", local, body)
	}
	if !strings.Contains(body, `src="/media/megadrive/images/golden-reg.png"`) {
		t.Errorf("the page does not show the registry's cover art\n--- page ---\n%s", body)
	}
}

func TestServeRomsFolderSystemStatus_CoverArtFileMissing_ShowsAnEmptySlot(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	writeSystemGames(t, romsFolder, "megadrive", []gamelist.Game{{Path: "./Golden Axe.zip", Name: "Golden Axe", Image: "./images/gone.png"}})

	body := get(t, Handler(reg, registryFolder, []string{romsFolder}), romsFolderSystemStatusURLFor(romsFolder, "megadrive", 1)).Body.String()

	if strings.Contains(body, "gone.png") {
		t.Errorf("the page links a cover art file that does not exist\n--- page ---\n%s", body)
	}
	if !strings.Contains(body, "status__art--empty") {
		t.Errorf("the page shows no empty slot for the missing cover art\n--- page ---\n%s", body)
	}
}

func TestServeRomsFolderMedium_ExistingImage_IsServed(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	writeFile(t, filepath.Join(romsFolder, "megadrive", "images", "my cover.png"), "local-art")

	rec := get(t, Handler(reg, registryFolder, []string{romsFolder}), romsFolderMediaURL(romsFolder, "megadrive", "./images/my cover.png"))

	if rec.Code != http.StatusOK || rec.Body.String() != "local-art" {
		t.Errorf("status = %d, body = %q, want 200 and the file's content", rec.Code, rec.Body.String())
	}
}

func TestServeRomsFolderMedium_NonImageOrOutsideTheSystem_IsRefused(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	writeFile(t, filepath.Join(romsFolder, "secret.png"), "outside")
	handler := Handler(reg, registryFolder, []string{romsFolder})
	query := "?" + statusFolderParam + "=" + url.QueryEscape(romsFolder)

	targets := []string{
		romsFolderMediaURL(romsFolder, "megadrive", "./Golden Axe.zip"),
		romsFolderMediaURL(romsFolder, "megadrive", "./gamelist.xml"),
		romsFolderMediaPrefix + "megadrive/%2E%2E/secret.png" + query,
		romsFolderMediaPrefix + "%2E%2E/" + filepath.Base(romsFolder) + "/secret.png" + query,
		romsFolderMediaPrefix + "megadrive/images/missing.png" + query,
		romsFolderMediaPrefix + "megadrive/images/x.png?" + statusFolderParam + "=/not/configured",
	}
	for _, target := range targets {
		rec := get(t, handler, target)
		if rec.Code == http.StatusOK {
			t.Errorf("%s: status = 200 (body %q), want a refusal", target, rec.Body.String())
		}
	}
}

// writeSystemGames replaces system's gamelist.xml in romsFolder with games,
// creating each game's ROM file alongside.
func writeSystemGames(t *testing.T, romsFolder, system string, games []gamelist.Game) {
	t.Helper()
	folder := filepath.Join(romsFolder, system)
	for _, g := range games {
		writeFile(t, filepath.Join(folder, filepath.Base(g.Path)), "rom")
	}
	if err := gamelist.UpdateFile(filepath.Join(folder, "gamelist.xml"), games); err != nil {
		t.Fatalf("write gamelist: %v", err)
	}
}

// writeFile writes content to path, creating its parent folders.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
