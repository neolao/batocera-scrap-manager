package webui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/store"
)

const sonicDeleteURL = "/game/megadrive/Sonic/delete"

// sonicMedia lists the media files the deletable registry's Sonic entry
// references, as paths relative to its system folder.
var sonicMedia = []string{"images/sonic.png", "videos/sonic.mp4", "images/marquee.png", "images/thumb.png"}

// deletableRegistry writes a registry holding one game of megadrive with all
// four of its media actually present on disk, a second game of that system so
// it outlives the deletion, plus a game of another system — so a test can
// watch a deletion erase exactly one game's files and leave the rest of the
// registry alone.
func deletableRegistry(t *testing.T) (*registry.Registry, string) {
	t.Helper()
	folder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{
			Path: "./Sonic.zip", Name: "Sonic the Hedgehog", Desc: "Fast.",
			Image: "images/sonic.png", Video: "videos/sonic.mp4",
			Marquee: "images/marquee.png", Thumbnail: "images/thumb.png",
			Rating: "0.85", ReleaseDate: "19910623T000000",
			Developer: "Sonic Team", Publisher: "Sega", Genre: "Platform", Players: "1",
		}},
		{System: "megadrive", Game: gamelist.Game{
			Path: "./Ecco.zip", Name: "Ecco the Dolphin",
		}},
		{System: "mastersystem", Game: gamelist.Game{
			Path: "./Alex Kidd.zip", Name: "Alex Kidd in Miracle World", Image: "images/alex.png",
		}},
	}}
	for _, medium := range []struct{ system, relPath string }{
		{"megadrive", "images/sonic.png"},
		{"megadrive", "videos/sonic.mp4"},
		{"megadrive", "images/marquee.png"},
		{"megadrive", "images/thumb.png"},
		{"mastersystem", "images/alex.png"},
	} {
		writeMediaFile(t, folder, medium.system, medium.relPath)
	}
	if err := store.Save(reg, folder); err != nil {
		t.Fatalf("failed to write the test registry: %v", err)
	}
	return reg, folder
}

// registryFiles lists what a game of the registry still has on disk: its
// metadata file when it is there, plus each of the given media paths present.
func registryFiles(t *testing.T, folder, system, gameFile string, media []string) []string {
	t.Helper()
	var present []string
	if _, err := os.Stat(filepath.Join(folder, system, gameFile)); gameFile != "" && err == nil {
		present = append(present, gameFile)
	}
	for _, relPath := range media {
		if _, err := os.Stat(filepath.Join(folder, system, filepath.FromSlash(relPath))); err == nil {
			present = append(present, relPath)
		}
	}
	return present
}

// blockDeletionOf replaces path with a non-empty directory, which os.Remove
// refuses to delete — staging a deletion failure without touching permissions.
func blockDeletionOf(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(path, "keeps it non-empty"), []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
}

func TestHandler_DeletePage_ExistingGame_AsksForConfirmationWithoutDeletingAnything(t *testing.T) {
	reg, folder := deletableRegistry(t)
	h := Handler(reg, folder)

	rec := get(t, h, sonicDeleteURL)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Sonic the Hedgehog") {
		t.Errorf("the confirmation does not name the game, got: %s", body)
	}
	present := registryFiles(t, folder, "megadrive", "Sonic.json", sonicMedia)
	if len(present) != 5 {
		t.Errorf("files still on disk = %v, want all 5: asking is not deleting", present)
	}
	if !strings.Contains(get(t, h, "/system/megadrive").Body.String(), "Sonic the Hedgehog") {
		t.Error("the game left the list although nothing was confirmed")
	}
}

func TestHandler_DeletePage_ListsTheGameFileAndEveryMediumItWillDelete(t *testing.T) {
	reg, folder := deletableRegistry(t)

	body := get(t, Handler(reg, folder), sonicDeleteURL).Body.String()

	// Every path is listed relative to the registry folder, the metadata file
	// included: a list mixing two origins reads as two different places.
	want := []string{"megadrive/Sonic.json"}
	for _, relPath := range sonicMedia {
		want = append(want, "megadrive/"+relPath)
	}
	for _, expected := range want {
		if !strings.Contains(body, "<code>"+expected+"</code>") {
			t.Errorf("the confirmation does not list %q, got: %s", expected, body)
		}
	}
}

func TestHandler_DeletePage_MediumMissingFromDisk_IsNotAnnouncedAsAboutToBeDeleted(t *testing.T) {
	reg, folder := deletableRegistry(t)
	if err := os.Remove(filepath.Join(folder, "megadrive", "videos", "sonic.mp4")); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}

	body := get(t, Handler(reg, folder), sonicDeleteURL).Body.String()

	if strings.Contains(body, "videos/sonic.mp4") {
		t.Errorf("the confirmation promises to delete a file that is not there, got: %s", body)
	}
	if !strings.Contains(body, "images/sonic.png") {
		t.Errorf("the confirmation dropped the media that are there, got: %s", body)
	}
}

func TestHandler_DeletePage_OffersToCancelBackToTheGamePage(t *testing.T) {
	reg, folder := deletableRegistry(t)

	body := get(t, Handler(reg, folder), sonicDeleteURL).Body.String()

	var found bool
	for _, link := range cardLinks(body) {
		if link == sonicGameURL {
			found = true
		}
	}
	if !found {
		t.Errorf("the confirmation offers no way back to the game page, got: %s", body)
	}
}

func TestHandler_DeletePage_ProtectedGame_SaysProtectionDoesNotPreventDeletion(t *testing.T) {
	reg, folder := deletableRegistry(t)
	h := Handler(reg, folder)
	post(t, h, sonicProtectURL, protectForm("on"))

	body := get(t, h, sonicDeleteURL).Body.String()

	if !strings.Contains(body, "protected") {
		t.Errorf("the confirmation does not warn that this game is protected, got: %s", body)
	}
}

func TestHandler_DeletePage_UnknownGame_RendersTheNotFoundPage(t *testing.T) {
	reg, folder := deletableRegistry(t)

	rec := get(t, Handler(reg, folder), "/game/megadrive/Nope/delete")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestHandler_Delete_Confirmed_RemovesTheGameFileAndEveryMedium(t *testing.T) {
	reg, folder := deletableRegistry(t)

	rec := post(t, Handler(reg, folder), sonicDeleteURL, nil)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 (body: %s)", rec.Code, rec.Body.String())
	}
	if present := registryFiles(t, folder, "megadrive", "Sonic.json", sonicMedia); len(present) != 0 {
		t.Errorf("files still on disk = %v, want none left", present)
	}
}

func TestHandler_Delete_Confirmed_LeadsBackToTheSystemListWithoutTheGame(t *testing.T) {
	reg, folder := deletableRegistry(t)
	h := Handler(reg, folder)

	rec := post(t, h, sonicDeleteURL, nil)

	location := rec.Header().Get("Location")
	target, _, _ := strings.Cut(location, "#")
	if !strings.HasPrefix(location, "/system/megadrive?") {
		t.Fatalf("Location = %q, want it to lead back to the system's list", location)
	}
	// The banner names the deleted game on purpose, so what must be gone is its
	// card — the link to a page that no longer exists — not its name.
	body := get(t, h, target).Body.String()
	if links := cardLinks(body); slices.Contains(links, sonicGameURL) {
		t.Errorf("the list still links to the deleted game: %v", links)
	}
	if !strings.Contains(body, "Ecco the Dolphin") {
		t.Errorf("the list lost a game that was not deleted, got: %s", body)
	}
}

func TestHandler_Delete_LastGameOfItsSystem_LeadsBackToTheHomePage(t *testing.T) {
	// Deleting the last game of a system makes the system itself disappear:
	// landing on its page would answer a 404 to a successful deletion.
	folder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic the Hedgehog"}},
	}}
	if err := store.Save(reg, folder); err != nil {
		t.Fatalf("failed to write the test registry: %v", err)
	}
	h := Handler(reg, folder)

	rec := post(t, h, sonicDeleteURL, nil)

	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "/?") {
		t.Fatalf("Location = %q, want it to lead back to the home page", location)
	}
	target, _, _ := strings.Cut(location, "#")
	landing := get(t, h, target)
	if landing.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", target, landing.Code)
	}
	if !strings.Contains(landing.Body.String(), "Sonic the Hedgehog") {
		t.Errorf("the home page does not confirm which game was deleted, got: %s", landing.Body.String())
	}
}

func TestHandler_Delete_Confirmed_ConfirmationNamesTheDeletedGame(t *testing.T) {
	reg, folder := deletableRegistry(t)
	h := Handler(reg, folder)

	rec := post(t, h, sonicDeleteURL, nil)

	target, _, _ := strings.Cut(rec.Header().Get("Location"), "#")
	body := get(t, h, target).Body.String()
	if !strings.Contains(body, "Sonic the Hedgehog") {
		t.Errorf("the confirmation does not name the deleted game, got: %s", body)
	}
	if !strings.Contains(body, `id="deleted"`) {
		t.Errorf("the confirmation is not the announced banner, got: %s", body)
	}
}

func TestHandler_HomePage_BrowsedTo_ShowsNoDeletionBanner(t *testing.T) {
	reg, folder := deletableRegistry(t)

	body := get(t, Handler(reg, folder), "/").Body.String()

	if strings.Contains(body, `id="deleted"`) {
		t.Errorf("the list announces a deletion nobody made, got: %s", body)
	}
}

func TestHandler_Delete_Confirmed_TheGamePageIsGoneToo(t *testing.T) {
	reg, folder := deletableRegistry(t)
	h := Handler(reg, folder)

	post(t, h, sonicDeleteURL, nil)

	if code := get(t, h, sonicGameURL).Code; code != http.StatusNotFound {
		t.Errorf("status = %d, want 404: the game page should be gone", code)
	}
}

func TestHandler_Delete_Confirmed_RegeneratesTheConsultationSiteWithoutTheGame(t *testing.T) {
	reg, folder := deletableRegistry(t)

	post(t, Handler(reg, folder), sonicDeleteURL, nil)

	site := indexFile(t, folder)
	if strings.Contains(site, "Sonic the Hedgehog") {
		t.Errorf("the consultation site still shows the deleted game, got: %s", site)
	}
	if !strings.Contains(site, "Alex Kidd in Miracle World") {
		t.Error("the consultation site lost a game that was not deleted")
	}
}

func TestHandler_Delete_Confirmed_LeavesTheOtherSystemsFilesAlone(t *testing.T) {
	reg, folder := deletableRegistry(t)

	post(t, Handler(reg, folder), sonicDeleteURL, nil)

	present := registryFiles(t, folder, "mastersystem", "Alex Kidd.json", []string{"images/alex.png"})
	if len(present) != 2 {
		t.Errorf("files still on disk for the other system = %v, want both kept", present)
	}
}

func TestHandler_Delete_Replayed_SaysTheGameIsNoLongerThereAndChangesNothing(t *testing.T) {
	reg, folder := deletableRegistry(t)
	h := Handler(reg, folder)
	post(t, h, sonicDeleteURL, nil)

	rec := post(t, h, sonicDeleteURL, nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body: %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "already") {
		t.Errorf("the page does not suggest the game was already deleted, got: %s", rec.Body.String())
	}
	present := registryFiles(t, folder, "mastersystem", "Alex Kidd.json", []string{"images/alex.png"})
	if len(present) != 2 {
		t.Errorf("files of the other system = %v, want both kept", present)
	}
}

func TestHandler_Delete_UnknownGame_Returns404AndChangesNothing(t *testing.T) {
	reg, folder := deletableRegistry(t)

	rec := post(t, Handler(reg, folder), "/game/megadrive/Nope/delete", nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if present := registryFiles(t, folder, "megadrive", "Sonic.json", sonicMedia); len(present) != 5 {
		t.Errorf("files on disk = %v, want all 5 kept", present)
	}
}

func TestHandler_Delete_GameRequestedUnderTheWrongSystem_IsNotFound(t *testing.T) {
	reg, folder := deletableRegistry(t)

	rec := post(t, Handler(reg, folder), "/game/mastersystem/Sonic/delete", nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if present := registryFiles(t, folder, "megadrive", "Sonic.json", sonicMedia); len(present) != 5 {
		t.Errorf("files on disk = %v, want all 5 kept", present)
	}
}

func TestHandler_Delete_GameIDContainingADot_DeletesThatGameAndNoOther(t *testing.T) {
	folder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Micro Machines v3.0.zip", Name: "Micro Machines v3.0"}},
		{System: "megadrive", Game: gamelist.Game{Path: "./Micro Machines v3.zip", Name: "Micro Machines v3"}},
	}}
	if err := store.Save(reg, folder); err != nil {
		t.Fatalf("failed to write the test registry: %v", err)
	}

	rec := post(t, Handler(reg, folder), "/game/megadrive/"+url.PathEscape("Micro Machines v3.0")+"/delete", nil)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 (body: %s)", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(folder, "megadrive", "Micro Machines v3.0.json")); err == nil {
		t.Error("Micro Machines v3.0.json still exists, want it deleted")
	}
	if _, err := os.Stat(filepath.Join(folder, "megadrive", "Micro Machines v3.json")); err != nil {
		t.Errorf("the other game was deleted instead: %v", err)
	}
}

func TestHandler_Delete_CrossSiteSubmission_IsRefusedAndChangesNothing(t *testing.T) {
	reg, folder := deletableRegistry(t)
	h := Handler(reg, folder)
	r := httptest.NewRequest(http.MethodPost, sonicDeleteURL, strings.NewReader(""))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body: %s)", rec.Code, rec.Body.String())
	}
	if present := registryFiles(t, folder, "megadrive", "Sonic.json", sonicMedia); len(present) != 5 {
		t.Errorf("files on disk = %v, want all 5 kept", present)
	}
}

func TestHandler_Delete_WrongMethod_IsRefusedNamingTheAllowedOnes(t *testing.T) {
	reg, folder := deletableRegistry(t)
	rec := httptest.NewRecorder()

	Handler(reg, folder).ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, sonicDeleteURL, nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405 (body: %s)", rec.Code, rec.Body.String())
	}
	if allow := rec.Header().Get("Allow"); allow != "GET, POST" {
		t.Errorf("Allow = %q, want \"GET, POST\"", allow)
	}
	if present := registryFiles(t, folder, "megadrive", "Sonic.json", sonicMedia); len(present) != 5 {
		t.Errorf("files on disk = %v, want all 5 kept", present)
	}
}

func TestHandler_Delete_GameFileCannotBeDeleted_SaysSoAndKeepsTheGame(t *testing.T) {
	reg, folder := deletableRegistry(t)
	blockDeletionOf(t, filepath.Join(folder, "megadrive", "Sonic.json"))
	h := Handler(reg, folder)

	rec := post(t, h, sonicDeleteURL, nil)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body: %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "not deleted") {
		t.Errorf("the page does not say the game was not deleted, got: %s", rec.Body.String())
	}
	if !strings.Contains(get(t, h, "/system/megadrive").Body.String(), "Sonic the Hedgehog") {
		t.Error("the served list dropped a game that is still on disk")
	}
	if present := registryFiles(t, folder, "megadrive", "", sonicMedia); len(present) != 4 {
		t.Errorf("media still on disk = %v, want all 4 kept: nothing was deleted", present)
	}
}

func TestHandler_Delete_MediumCannotBeDeleted_StillDeletesTheGameAndSaysWhatIsLeft(t *testing.T) {
	reg, folder := deletableRegistry(t)
	blockDeletionOf(t, filepath.Join(folder, "megadrive", "images", "sonic.png"))
	h := Handler(reg, folder)

	rec := post(t, h, sonicDeleteURL, nil)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303: the game itself was deleted (body: %s)", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(folder, "megadrive", "Sonic.json")); err == nil {
		t.Error("Sonic.json still exists, want the deletion to hold")
	}
	target, _, _ := strings.Cut(rec.Header().Get("Location"), "#")
	body := get(t, h, target).Body.String()
	if !strings.Contains(body, "media") {
		t.Errorf("the confirmation does not mention the media left behind, got: %s", body)
	}
	if strings.Contains(body, "Sonic the Hedgehog</h3>") {
		t.Error("the list still shows the deleted game")
	}
}

func TestHandler_Delete_SiteCannotBeRegenerated_StillDeletesAndSaysSo(t *testing.T) {
	reg, folder := deletableRegistry(t)
	blockDeletionOf(t, filepath.Join(folder, "index.html"))
	h := Handler(reg, folder)

	rec := post(t, h, sonicDeleteURL, nil)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303: the game itself was deleted (body: %s)", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(folder, "megadrive", "Sonic.json")); err == nil {
		t.Error("Sonic.json still exists, want the deletion to hold")
	}
	target, _, _ := strings.Cut(rec.Header().Get("Location"), "#")
	body := get(t, h, target).Body.String()
	if !strings.Contains(body, "consultation site") {
		t.Errorf("the confirmation does not warn that the consultation site is stale, got: %s", body)
	}
}

func TestHandler_Delete_AfterADeletion_AnotherGameSaveDoesNotBringItBack(t *testing.T) {
	reg, folder := deletableRegistry(t)
	h := Handler(reg, folder)
	post(t, h, sonicDeleteURL, nil)

	// Any later write rewrites every entry of the served snapshot: a snapshot
	// still holding the deleted game would recreate its file (see decisions/022).
	post(t, h, "/game/mastersystem/"+url.PathEscape("Alex Kidd")+"/protect", protectForm("on"))

	if _, err := os.Stat(filepath.Join(folder, "megadrive", "Sonic.json")); err == nil {
		t.Error("Sonic.json came back after another game was written")
	}
}

func TestHandler_GamePage_LinksToItsDeletePage(t *testing.T) {
	reg, folder := deletableRegistry(t)

	body := get(t, Handler(reg, folder), sonicGameURL).Body.String()

	var found bool
	for _, link := range cardLinks(body) {
		if link == sonicDeleteURL {
			found = true
		}
	}
	if !found {
		t.Errorf("the game page offers no way to delete the game, got: %s", body)
	}
}
