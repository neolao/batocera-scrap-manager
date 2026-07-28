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
	"github.com/neolao/batocera-scrap-manager/internal/store"
)

// renamedGameURL is where Sonic lands once its ROM path names another file.
const renamedGameURL = "/game/megadrive/Sonic%202"

// entryFiles lists the JSON files a system folder holds, so a test can state
// which entry file exists rather than only which entry the registry reports.
func entryFiles(t *testing.T, folder, system string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(folder, system, "*.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	names := make([]string, len(matches))
	for i, match := range matches {
		names[i] = filepath.Base(match)
	}
	return names
}

// follow requests the page a redirect points at, the way a browser does: the
// fragment never leaves it, and httptest would otherwise fold it into the last
// query value.
func follow(t *testing.T, h http.Handler, rec *httptest.ResponseRecorder) *httptest.ResponseRecorder {
	t.Helper()
	location, _, _ := strings.Cut(rec.Header().Get("Location"), "#")
	if location == "" {
		t.Fatalf("no Location to follow (status %d)", rec.Code)
	}
	return get(t, h, location)
}

// fieldError returns the message rendered under the control named name.
func fieldError(t *testing.T, body, name string) string {
	t.Helper()
	block := regexp.MustCompile(`(?s)<p class="field__error" id="error-` + regexp.QuoteMeta(name) + `">(.*?)</p>`)
	match := block.FindStringSubmatch(body)
	if match == nil {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func TestHandler_GamePage_ShowsTheStoredRomPath(t *testing.T) {
	// The path is what identifies the game in gamelist.xml, so it has to be
	// readable — subfolder included, exactly as stored.
	reg, folder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, folder, nil), "/game/megadrive/Sonic%20the%20Hedgehog")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "./Sonic the Hedgehog.zip") {
		t.Errorf("the game page does not show the stored ROM path, want %q in:\n%s", "./Sonic the Hedgehog.zip", body)
	}
	if !strings.Contains(body, "ROM file") {
		t.Error("the ROM path is shown without a label naming it")
	}
}

func TestHandler_EditForm_PreFillsTheRomPathWithTheStoredValue(t *testing.T) {
	reg, folder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, folder, nil), sonicEditURL)

	value, found := inputValue(t, rec.Body.String(), "path")
	if !found {
		t.Fatalf("no control named %q on the edit form:\n%s", "path", rec.Body.String())
	}
	if value != "./Sonic the Hedgehog.zip" {
		t.Errorf("path control = %q, want the stored path %q", value, "./Sonic the Hedgehog.zip")
	}
}

func TestHandler_EditForm_ProtectedGame_OffersNoHandBackForTheRomPath(t *testing.T) {
	// A path is the entry's identity, not one of its values: the mechanism
	// shielding hand-corrected metadata from later imports does not apply to
	// it, so it must offer neither the mark nor the hand-back checkbox.
	reg, folder := fullyScrapedRegistry(t)
	if err := registry.Protect(reg, "megadrive", "Sonic the Hedgehog"); err != nil {
		t.Fatalf("Protect() error = %v", err)
	}

	rec := get(t, Handler(reg, folder, nil), sonicEditURL)

	body := rec.Body.String()
	if strings.Contains(body, `value="path"`) {
		t.Error("the ROM path offers a hand-back checkbox, want none: it is not a metadata field")
	}
	if strings.Contains(body, `id="hand-back-path"`) {
		t.Error("the ROM path carries a hand-back control, want none")
	}
}

func TestHandler_Save_NewRomPath_MovesTheEntryFileAndRedirectsToTheNewURL(t *testing.T) {
	reg, folder := savedRegistry(t)
	form := storedValues()
	form.Set("path", "disc1/Sonic 2.zip")

	rec := post(t, Handler(reg, folder, nil), sonicSaveURL, form)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 (body: %s)", rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); !strings.HasPrefix(location, renamedGameURL) {
		t.Errorf("Location = %q, want it to lead to the game's new address %q", location, renamedGameURL)
	}
	if got := entryFiles(t, folder, "megadrive"); len(got) != 1 || got[0] != "Sonic 2.json" {
		t.Errorf("entry files = %v, want exactly [Sonic 2.json]: the old one must not survive", got)
	}
	if stored := storedPath(t, folder, "Sonic 2"); stored != "disc1/Sonic 2.zip" {
		t.Errorf("stored path = %q, want %q", stored, "disc1/Sonic 2.zip")
	}
}

func TestHandler_Save_NewRomPath_ServesTheGameAtItsNewURLAndNotTheOldOne(t *testing.T) {
	reg, folder := savedRegistry(t)
	handler := Handler(reg, folder, nil)
	form := storedValues()
	form.Set("path", "disc1/Sonic 2.zip")
	post(t, handler, sonicSaveURL, form)

	if rec := get(t, handler, renamedGameURL); rec.Code != http.StatusOK {
		t.Errorf("GET %s = %d, want 200: the game must be reachable at its new address", renamedGameURL, rec.Code)
	}
	if rec := get(t, handler, sonicGameURL); rec.Code != http.StatusNotFound {
		t.Errorf("GET %s = %d, want 404: nothing answers under the old identifier", sonicGameURL, rec.Code)
	}
}

func TestHandler_Save_NewRomPath_ConfirmsTheNewPathAndThatThePageMoved(t *testing.T) {
	reg, folder := savedRegistry(t)
	handler := Handler(reg, folder, nil)
	form := storedValues()
	form.Set("path", "disc1/Sonic 2.zip")

	rec := post(t, handler, sonicSaveURL, form)

	banner := confirmationBanner(t, follow(t, handler, rec).Body.String())
	if !strings.Contains(banner, "disc1/Sonic 2.zip") {
		t.Errorf("banner = %q, want it to name the new ROM path", banner)
	}
	if !strings.Contains(strings.ToLower(banner), "moved") {
		t.Errorf("banner = %q, want it to say the page moved: its address changed under the user", banner)
	}
}

func TestHandler_Save_RomPathKeepingTheSameIdentifier_StaysWhereItIs(t *testing.T) {
	// disc1/Sonic.iso and ./Sonic.zip share a base name: the entry file does
	// not move and the page does not change address, so the confirmation must
	// not claim otherwise — and the freshly written file must not be deleted
	// as if it were the old one.
	reg, folder := savedRegistry(t)
	handler := Handler(reg, folder, nil)
	form := storedValues()
	form.Set("path", "disc1/Sonic.iso")

	rec := post(t, handler, sonicSaveURL, form)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 (body: %s)", rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); !strings.HasPrefix(location, sonicGameURL) {
		t.Errorf("Location = %q, want the unchanged address %q", location, sonicGameURL)
	}
	if got := entryFiles(t, folder, "megadrive"); len(got) != 1 || got[0] != "Sonic.json" {
		t.Fatalf("entry files = %v, want exactly [Sonic.json] still present", got)
	}
	if stored := storedPath(t, folder, "Sonic"); stored != "disc1/Sonic.iso" {
		t.Errorf("stored path = %q, want %q", stored, "disc1/Sonic.iso")
	}

	banner := confirmationBanner(t, follow(t, handler, rec).Body.String())
	if !strings.Contains(banner, "disc1/Sonic.iso") {
		t.Errorf("banner = %q, want it to name the new ROM path", banner)
	}
	if strings.Contains(strings.ToLower(banner), "moved") {
		t.Errorf("banner = %q, want no claim that the page moved: it did not", banner)
	}
}

func TestHandler_Save_RefusedRomPath_RerendersTheFormAndChangesNothing(t *testing.T) {
	for _, tc := range []struct {
		name        string
		path        string
		wantMessage string
	}{
		{"empty", "", "identified"},
		{"absolute", "/roms/megadrive/Sonic.zip", "relative"},
		{"escaping", "../mastersystem/Sonic.zip", "inside"},
		{"naming a folder", "disc1/", "file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reg, folder := savedRegistry(t)
			before := gameFile(t, folder)
			form := storedValues()
			form.Set("path", tc.path)

			rec := post(t, Handler(reg, folder, nil), sonicSaveURL, form)

			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422", rec.Code)
			}
			body := rec.Body.String()
			if message := fieldError(t, body, "path"); !strings.Contains(message, tc.wantMessage) {
				t.Errorf("message = %q, want it to mention %q", message, tc.wantMessage)
			}
			if value, _ := inputValue(t, body, "path"); value != tc.path {
				t.Errorf("path control = %q, want the submitted value %q back", value, tc.path)
			}
			if got := entryFiles(t, folder, "megadrive"); len(got) != 1 || got[0] != "Sonic.json" {
				t.Errorf("entry files = %v, want [Sonic.json] untouched", got)
			}
			if after := gameFile(t, folder); after != before {
				t.Error("the stored game changed, want the registry strictly unchanged")
			}
		})
	}
}

func TestHandler_Save_RomPathTakenByAnotherGame_IsRefusedWithoutOverwritingIt(t *testing.T) {
	folder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic the Hedghog", Rating: "0.85", ReleaseDate: "19910623T000000",
			Developer: "Sonic Team", Publisher: "Sega", Genre: "Platform", Players: "1", Desc: "Fast."}},
		// Named apart from its file on purpose: the message has to name the
		// clashing file name and the game holding it, which a game called
		// after its own file would not tell apart.
		{System: "megadrive", Game: gamelist.Game{Path: "./sor.zip", Name: "Streets of Rage"}},
	}}
	if err := store.Save(reg, folder); err != nil {
		t.Fatalf("failed to write the test registry: %v", err)
	}
	form := storedValues()
	form.Set("path", "disc2/sor.zip")

	rec := post(t, Handler(reg, folder, nil), sonicSaveURL, form)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	message := fieldError(t, rec.Body.String(), "path")
	if !strings.Contains(message, "Streets of Rage") {
		t.Errorf("message = %q, want it to name the game already holding that file name", message)
	}
	if !strings.Contains(message, "sor") {
		t.Errorf("message = %q, want it to name the clashing file name, not only the game", message)
	}
	if stored := storedPath(t, folder, "sor"); stored != "./sor.zip" {
		t.Errorf("the other game's stored path = %q, want it untouched", stored)
	}
	if got := entryFiles(t, folder, "megadrive"); len(got) != 2 {
		t.Errorf("entry files = %v, want both games still stored", got)
	}
}

func TestHandler_Save_RomPathTakenByAGameNamedAfterIt_DoesNotSayTheNameTwice(t *testing.T) {
	// "already uses the file name \"Streets of Rage\" — the one named Streets
	// of Rage" reads as a bug. The game is named only when that adds something.
	folder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic the Hedghog", Rating: "0.85", ReleaseDate: "19910623T000000",
			Developer: "Sonic Team", Publisher: "Sega", Genre: "Platform", Players: "1", Desc: "Fast."}},
		{System: "megadrive", Game: gamelist.Game{Path: "./Streets of Rage.zip", Name: "Streets of Rage"}},
	}}
	if err := store.Save(reg, folder); err != nil {
		t.Fatalf("failed to write the test registry: %v", err)
	}
	form := storedValues()
	form.Set("path", "disc2/Streets of Rage.zip")

	rec := post(t, Handler(reg, folder, nil), sonicSaveURL, form)

	message := fieldError(t, rec.Body.String(), "path")
	if strings.Count(message, "Streets of Rage") != 1 {
		t.Errorf("message = %q, want the name stated exactly once", message)
	}
}

func TestHandler_Save_RefusedRomPath_KeepsTheOtherCorrectionsInTheForm(t *testing.T) {
	// The path is submitted alongside the eight metadata controls: re-reading
	// the stored values on refusal would silently discard what the user typed.
	reg, folder := savedRegistry(t)
	form := storedValues()
	form.Set("path", "/absolute/Sonic.zip")
	form.Set("name", "Sonic the Hedgehog")
	form.Set("genre", "Run and jump")

	rec := post(t, Handler(reg, folder, nil), sonicSaveURL, form)

	body := rec.Body.String()
	if value, _ := inputValue(t, body, "name"); value != "Sonic the Hedgehog" {
		t.Errorf("name control = %q, want the submitted correction back", value)
	}
	if value, _ := inputValue(t, body, "genre"); value != "Run and jump" {
		t.Errorf("genre control = %q, want the submitted correction back", value)
	}
}

func TestHandler_Save_RefusedRomPath_AssociatesBothItsHintAndItsErrorWithTheControl(t *testing.T) {
	// The ROM path is the first control that always carries a hint and can be
	// refused: announcing only one of the two would hide the reason it was
	// refused from a screen reader.
	reg, folder := savedRegistry(t)
	form := storedValues()
	form.Set("path", "")

	rec := post(t, Handler(reg, folder, nil), sonicSaveURL, form)

	body := rec.Body.String()
	control := regexp.MustCompile(`<input[^>]*name="path"[^>]*>`).FindString(body)
	if control == "" {
		t.Fatalf("no control named path in:\n%s", body)
	}
	described := regexp.MustCompile(`aria-describedby="([^"]*)"`).FindStringSubmatch(control)
	if described == nil {
		t.Fatalf("control = %q, want an aria-describedby", control)
	}
	for _, id := range []string{"hint-path", "error-path"} {
		if !strings.Contains(described[1], id) {
			t.Errorf("aria-describedby = %q, want it to name %q", described[1], id)
		}
	}
	if !strings.Contains(control, `aria-invalid="true"`) {
		t.Errorf("control = %q, want aria-invalid=\"true\"", control)
	}
	if !strings.Contains(body, `href="#field-path"`) {
		t.Error("the error summary does not link to the ROM path control")
	}
}

func TestHandler_Save_RegistryFolderUnwritable_LeavesTheEntryFileWhereItWas(t *testing.T) {
	reg, folder := savedRegistry(t)
	system := filepath.Join(folder, "megadrive")
	if err := os.Chmod(system, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(system, 0o755) })
	form := storedValues()
	form.Set("path", "disc1/Sonic 2.zip")

	rec := post(t, Handler(reg, folder, nil), sonicSaveURL, form)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: nothing was written", rec.Code)
	}
	if got := entryFiles(t, folder, "megadrive"); len(got) != 1 || got[0] != "Sonic.json" {
		t.Errorf("entry files = %v, want [Sonic.json] untouched", got)
	}
	if stored := storedPath(t, folder, "Sonic"); stored != "./Sonic.zip" {
		t.Errorf("stored path = %q, want it unchanged on disk", stored)
	}
}

func TestHandler_Save_OldEntryFileCannotBeErased_StillSavesAndWarnsAboutIt(t *testing.T) {
	// The rename is committed by the write, not by the erasure (decisions/024):
	// a leftover file is a caveat on a successful save, never a failure — but
	// it must be said, since it resurrects the game as a duplicate on restart.
	reg, folder := savedRegistry(t)
	stubborn := filepath.Join(folder, "megadrive", "Sonic.json")
	if err := os.Remove(stubborn); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(stubborn, "held"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	handler := Handler(reg, folder, nil)
	form := storedValues()
	form.Set("path", "disc1/Sonic 2.zip")

	rec := post(t, handler, sonicSaveURL, form)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303: the rename did go through (body: %s)", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, renamedGameURL) {
		t.Fatalf("Location = %q, want the new address %q", location, renamedGameURL)
	}
	if stored := storedPath(t, folder, "Sonic 2"); stored != "disc1/Sonic 2.zip" {
		t.Errorf("stored path = %q, want the rename written", stored)
	}
	banner := confirmationBanner(t, follow(t, handler, rec).Body.String())
	if !strings.Contains(strings.ToLower(banner), "sonic.json") {
		t.Errorf("banner = %q, want it to name the file left behind", banner)
	}
}

func TestHandler_Save_UnchangedRomPath_LeavesTheEntryFileByteIdentical(t *testing.T) {
	// Opening the form and saving it back must not rewrite the identity: the
	// path travels through the form and comes back exactly as it left.
	reg, folder := savedRegistry(t)
	before := gameFile(t, folder)

	rec := post(t, Handler(reg, folder, nil), sonicSaveURL, storedValues())

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 (body: %s)", rec.Code, rec.Body.String())
	}
	if after := gameFile(t, folder); after != before {
		t.Errorf("the stored game changed:\nbefore: %s\nafter:  %s", before, after)
	}
}

// storedPath reads the ROM path out of the JSON file storing the game of the
// given identifier.
func storedPath(t *testing.T, folder, id string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(folder, "megadrive", id+".json"))
	if err != nil {
		t.Fatalf("failed to read the entry file of %q: %v", id, err)
	}
	match := regexp.MustCompile(`"path":\s*"([^"]*)"`).FindStringSubmatch(string(data))
	if match == nil {
		t.Fatalf("no path in the entry file of %q: %s", id, data)
	}
	return strings.ReplaceAll(match[1], `\/`, "/")
}

// confirmationBanner returns the text of the banner a game page confirms a
// save with.
func confirmationBanner(t *testing.T, body string) string {
	t.Helper()
	match := regexp.MustCompile(`(?s)<p class="banner" id="saved"[^>]*>(.*?)</p>`).FindStringSubmatch(body)
	if match == nil {
		t.Fatalf("no confirmation banner in:\n%s", body)
	}
	return html.UnescapeString(strings.TrimSpace(match[1]))
}
