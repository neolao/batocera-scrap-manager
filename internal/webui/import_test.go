package webui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// romsFolderToImport writes a ROMs folder holding one system whose gamelist
// carries two scraped games, and returns the empty registry to import them
// into with its own folder. Nothing references a medium, so the import is a
// pure metadata read — which is what these tests are about.
func romsFolderToImport(t *testing.T) (*registry.Registry, string, string) {
	t.Helper()

	romsFolder := t.TempDir()
	system := filepath.Join(romsFolder, "megadrive")
	if err := os.MkdirAll(system, 0o755); err != nil {
		t.Fatalf("failed to set up the ROMs folder: %v", err)
	}
	games := []gamelist.Game{
		{Path: "./Sonic.zip", Name: "Sonic the Hedgehog", Desc: "A blue hedgehog runs very fast.", Genre: "Platform"},
		{Path: "./Streets.zip", Name: "Streets of Rage", Desc: "Three fighters clean up the city."},
	}
	if err := gamelist.UpdateFile(filepath.Join(system, "gamelist.xml"), games); err != nil {
		t.Fatalf("failed to write the gamelist: %v", err)
	}

	return &registry.Registry{}, t.TempDir(), romsFolder
}

// waitForImport polls the import page until the run it reports is over — the
// page stops carrying its auto-reload exactly then — and returns the final
// page, so a test asserts on a report rather than on a moving one.
func waitForImport(t *testing.T, h http.Handler) string {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		body := get(t, h, importURL).Body.String()
		if !strings.Contains(body, `http-equiv="refresh"`) {
			return body
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("the import was still running after 10s")
	return ""
}

func TestServeImport_RomsFoldersConfigured_NamesEveryOneOfThem(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	romsFolders := twoRomsFolders(t)

	body := get(t, Handler(reg, registryFolder, romsFolders), importURL).Body.String()

	for _, folder := range romsFolders {
		if !strings.Contains(body, folder) {
			t.Errorf("the page does not name the ROMs folder %q — the user cannot check what is about to be read", folder)
		}
	}
}

func TestServeImport_RomsFoldersConfigured_SaysWhichWayItGoesAndWhatItLeavesAlone(t *testing.T) {
	// The two maintenance flows are opposites, and clicking the wrong one is
	// the mistake worth designing against: this page must say that it reads the
	// ROMs folders and writes the registry, never the other way round.
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder, twoRomsFolders(t)), importURL).Body.String()

	if !strings.Contains(body, "gamelist.xml") {
		t.Error("the page does not name what it reads")
	}
	if !strings.Contains(strings.ToLower(body), "left as they are") {
		t.Errorf("the page does not say the ROMs folders are left alone\n--- page ---\n%s", body)
	}
	if !strings.Contains(strings.ToLower(body), "hand") {
		t.Error("the page does not say what happens to values corrected by hand")
	}
	if !strings.Contains(body, `method="post"`) || !strings.Contains(body, `action="`+importURL+`"`) {
		t.Error("the page offers no form submitting the import")
	}
	if !strings.Contains(body, `href="/"`) {
		t.Error("the page offers no way back that touches nothing")
	}
}

func TestServeImport_NoRomsFolderConfigured_ShowsAZeroSummaryAndNamesTheCommand(t *testing.T) {
	// Nothing configured is a valid state, not an error: the page has to say
	// so with a summary of its own rather than with a refusal.
	reg, registryFolder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, registryFolder, nil), importURL)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d — having nothing configured is not an error", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "0 added, 0 updated, 0 unchanged") {
		t.Errorf("the page shows no zero summary\n--- page ---\n%s", body)
	}
	if strings.Contains(body, "<button") {
		t.Error("the page offers a button with no ROMs folder to read")
	}
	if !strings.Contains(body, "config add-roms-folder") {
		t.Error("the page does not name the command that configures a ROMs folder")
	}
}

func TestStartImport_Submitted_RedirectsToTheImportPage(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderToImport(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	rec := post(t, handler, importURL, nil)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if location := rec.Header().Get("Location"); location != importURL {
		t.Errorf("Location = %q, want %q", location, importURL)
	}
	waitForImport(t, handler)
}

func TestStartImport_SubmissionFromAnotherSite_IsRefused(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderToImport(t)
	before := folderFingerprint(t, registryFolder)
	r := httptest.NewRequest(http.MethodPost, importURL, nil)
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()

	Handler(reg, registryFolder, []string{romsFolder}).ServeHTTP(rec, r)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if after := folderFingerprint(t, registryFolder); after != before {
		t.Error("a refused submission wrote to the registry anyway")
	}
}

func TestStartImport_RunFinished_ReportsTheSameSummaryAsTheCommandLine(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderToImport(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	post(t, handler, importURL, nil)
	body := waitForImport(t, handler)

	want := "2 added, 0 updated, 0 unchanged"
	if !strings.Contains(body, want) {
		t.Errorf("report does not carry %q\n--- page ---\n%s", want, body)
	}
}

func TestStartImport_RunFinished_ServesTheImportedGamesWithoutARestart(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderToImport(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	post(t, handler, importURL, nil)
	waitForImport(t, handler)

	home := get(t, handler, "/").Body.String()
	if !strings.Contains(home, "megadrive") {
		t.Errorf("the home page does not list the imported system\n--- page ---\n%s", home)
	}
	system := get(t, handler, "/system/megadrive")
	if system.Code != http.StatusOK || !strings.Contains(system.Body.String(), "Sonic the Hedgehog") {
		t.Errorf("the system page does not list the imported game (status %d)", system.Code)
	}
	game := get(t, handler, "/game/megadrive/Sonic")
	if game.Code != http.StatusOK {
		t.Fatalf("the imported game has no page of its own: status = %d, want %d", game.Code, http.StatusOK)
	}
	if !strings.Contains(game.Body.String(), "A blue hedgehog runs very fast.") {
		t.Error("the imported game's page does not show what was imported")
	}
}

func TestStartImport_RunFinished_WritesTheRegistryAndRegeneratesTheSite(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderToImport(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	post(t, handler, importURL, nil)
	waitForImport(t, handler)

	if _, err := os.Stat(filepath.Join(registryFolder, "megadrive", "Sonic.json")); err != nil {
		t.Errorf("the imported game was not written to the registry folder: %v", err)
	}
	site, err := os.ReadFile(filepath.Join(registryFolder, "index.html"))
	if err != nil {
		t.Fatalf("the consultation site was not regenerated: %v", err)
	}
	if !strings.Contains(string(site), "Sonic the Hedgehog") {
		t.Error("the regenerated consultation site does not list the imported game")
	}
}

func TestStartImport_NothingNew_ReadsAsASuccessRatherThanAnError(t *testing.T) {
	// The common case is the repeat import: a report worded as a bare row of
	// zeroes reads as a failure to whoever just clicked the button.
	reg, registryFolder, romsFolder := romsFolderToImport(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	post(t, handler, importURL, nil)
	waitForImport(t, handler)
	post(t, handler, importURL, nil)
	body := waitForImport(t, handler)

	if !strings.Contains(body, "0 added, 0 updated, 2 unchanged") {
		t.Errorf("report does not carry the counts of the second import\n--- page ---\n%s", body)
	}
	if !strings.Contains(body, "already up to date") {
		t.Errorf("report does not say the registry was already up to date\n--- page ---\n%s", body)
	}
	// The stylesheet names that class whatever the page says, so what is
	// looked for here is the markup actually rendered with it.
	if strings.Contains(body, `class="run__problem"`) {
		t.Error("a repeat import with nothing to do is presented as a problem")
	}
}

func TestStartImport_EmptyFolder_SaysThereWasNothingToImport(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	handler := Handler(reg, registryFolder, []string{t.TempDir()})

	post(t, handler, importURL, nil)
	body := waitForImport(t, handler)

	if !strings.Contains(body, "nothing to import") {
		t.Errorf("report does not say the folders held nothing\n--- page ---\n%s", body)
	}
	if strings.Contains(body, "already up to date") {
		t.Error("a folder holding no game at all is reported as a registry already in step")
	}
}

func TestStartImport_OneFolderMissing_NamesItAndWritesNothing(t *testing.T) {
	// A folder that cannot be read stops the run and nothing is persisted, as
	// the update command does not persist either (decisions/028).
	reg, registryFolder, romsFolder := romsFolderToImport(t)
	missing := filepath.Join(t.TempDir(), "unplugged-sd-card")
	handler := Handler(reg, registryFolder, []string{romsFolder, missing})
	before := folderFingerprint(t, registryFolder)

	post(t, handler, importURL, nil)
	body := waitForImport(t, handler)

	if !strings.Contains(body, missing) {
		t.Errorf("report does not name the folder that could not be read (%q)\n--- page ---\n%s", missing, body)
	}
	if !strings.Contains(body, "2 added, 0 updated, 0 unchanged") {
		t.Errorf("report drops what the folder read before the failure\n--- page ---\n%s", body)
	}
	if !strings.Contains(strings.ToLower(body), "nothing was written") {
		t.Errorf("report lets the counts read as saved when nothing was\n--- page ---\n%s", body)
	}
	if after := folderFingerprint(t, registryFolder); after != before {
		t.Error("an interrupted import wrote to the registry folder anyway")
	}
	if home := get(t, handler, "/").Body.String(); strings.Contains(home, "megadrive") {
		t.Error("the served pages list games the registry on disk does not hold")
	}
}

func TestStartImport_RegistryCouldNotBeWritten_ReportsItAndServesWhatIsOnDisk(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderToImport(t)
	if err := os.Chmod(registryFolder, 0o500); err != nil {
		t.Fatalf("failed to make the registry folder read-only: %v", err)
	}
	t.Cleanup(func() { os.Chmod(registryFolder, 0o700) })
	handler := Handler(reg, registryFolder, []string{romsFolder})

	post(t, handler, importURL, nil)
	body := waitForImport(t, handler)

	if !strings.Contains(strings.ToLower(body), "could not be written") {
		t.Errorf("report does not say the registry could not be written\n--- page ---\n%s", body)
	}
	// This page is reachable by anyone on the local network (decisions/032):
	// the underlying OS error is for the server's own log, never for a page
	// served with no account behind it, since it can carry the server's
	// folder layout.
	if strings.Contains(strings.ToLower(body), "permission denied") || strings.Contains(body, registryFolder) {
		t.Errorf("report leaks the underlying OS error to the page\n--- page ---\n%s", body)
	}
	if home := get(t, handler, "/").Body.String(); strings.Contains(home, "megadrive") {
		t.Error("the served pages claim games the registry on disk does not hold")
	}
}

func TestStartImport_SiteCouldNotBeRegenerated_IsACaveatNotAFailedImport(t *testing.T) {
	// The registry is the source of truth: a stale consultation site is a
	// reservation on an import that did apply, never a reason to redo it.
	reg, registryFolder, romsFolder := romsFolderToImport(t)
	if err := os.Mkdir(filepath.Join(registryFolder, "index.html"), 0o755); err != nil {
		t.Fatalf("failed to block the site generation: %v", err)
	}
	handler := Handler(reg, registryFolder, []string{romsFolder})

	post(t, handler, importURL, nil)
	body := waitForImport(t, handler)

	if !strings.Contains(body, "2 added, 0 updated, 0 unchanged") {
		t.Errorf("the import is reported as having done nothing\n--- page ---\n%s", body)
	}
	if !strings.Contains(strings.ToLower(body), "consultation site") {
		t.Errorf("report does not mention the site that could not be regenerated\n--- page ---\n%s", body)
	}
	if game := get(t, handler, "/game/megadrive/Sonic"); game.Code != http.StatusOK {
		t.Errorf("the imported game is not served although the registry was written: status = %d", game.Code)
	}
}

func TestCommitImport_RegistryCorrectedWhileItRan_KeepsTheCorrectionAndWritesNothing(t *testing.T) {
	// An import works off the snapshot it captured, for minutes. A correction
	// saved meanwhile is not in that snapshot: writing it out would silently
	// erase a value the user had just typed by hand.
	_, registryFolder, _ := romsFolderToImport(t)
	ui := &webUI{reg: &registry.Registry{}, registryFolder: registryFolder}
	captured := ui.reg
	corrected := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Corrected by hand"}},
	}}
	ui.reg = corrected
	candidate := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic the Hedgehog"}},
	}}
	var state jobs
	run := state.start(jobImport)

	ui.commitImport(run, captured, candidate)
	run.finish()

	problem := strings.ToLower(state.view(jobImport).Report.Problem)
	if !strings.Contains(problem, "nothing was written") || !strings.Contains(problem, "corrected") {
		t.Errorf("report.Problem = %q, want it saying nothing was written because of a correction", problem)
	}
	if ui.reg != corrected {
		t.Error("the import swapped its own snapshot over a correction saved while it ran")
	}
	if entries, err := os.ReadDir(registryFolder); err != nil || len(entries) != 0 {
		t.Errorf("the registry folder holds %d entries, want none written", len(entries))
	}
}

func TestStartImport_NoRomsFolderConfigured_StartsNothing(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	before := folderFingerprint(t, registryFolder)
	handler := Handler(reg, registryFolder, nil)

	rec := post(t, handler, importURL, nil)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d — the page it lands on is what words the state", rec.Code, http.StatusSeeOther)
	}
	if after := folderFingerprint(t, registryFolder); after != before {
		t.Error("an import with nothing configured wrote to the registry")
	}
}

func TestServeImport_CompletionRunning_SaysSoAndOffersNoRunOfItsOwn(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderToImport(t)
	ui := &webUI{reg: reg, registryFolder: registryFolder, romsFolders: []string{romsFolder}}
	ui.jobs.start(jobCompletion)
	rec := httptest.NewRecorder()

	ui.serveImport(rec, httptest.NewRequest(http.MethodGet, importURL, nil))

	body := rec.Body.String()
	if strings.Contains(body, "<button") {
		t.Error("the import offers to start while a completion holds the slot")
	}
	if !strings.Contains(body, completeURL) {
		t.Errorf("the page does not link to the operation holding the slot\n--- page ---\n%s", body)
	}
}

func TestImport_MethodNeitherGetNorPost_AnswersMethodNotAllowed(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	rec := httptest.NewRecorder()

	Handler(reg, registryFolder, twoRomsFolders(t)).
		ServeHTTP(rec, httptest.NewRequest(http.MethodPut, importURL, nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); allow != "GET, POST" {
		t.Errorf("Allow = %q, want %q", allow, "GET, POST")
	}
}

func TestServeHome_NamesBothMaintenanceActionsAndTheirDirection(t *testing.T) {
	// Two neighbouring buttons whose verbs are equally opaque: the one that
	// writes into the user's own Batocera files must not be the one clicked by
	// someone who meant the other.
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder, twoRomsFolders(t)), "/").Body.String()

	if !strings.Contains(body, `href="`+importURL+`"`) {
		t.Errorf("the home page does not link to %s — the flow has no entry point", importURL)
	}
	if !strings.Contains(body, `href="`+completeURL+`"`) {
		t.Errorf("the home page no longer links to %s", completeURL)
	}
	if !strings.Contains(body, "into the registry") || !strings.Contains(body, "back into") {
		t.Errorf("the home page does not say which way each action goes\n--- page ---\n%s", body)
	}
}

func TestServeHome_EmptyRegistry_StillOffersTheImport(t *testing.T) {
	// An empty registry is exactly when importing matters most, and it is the
	// one state where the home page used to hide its actions.
	handler := Handler(&registry.Registry{}, t.TempDir(), twoRomsFolders(t))

	body := get(t, handler, "/").Body.String()

	if !strings.Contains(body, "No games in the registry yet.") {
		t.Error("the empty home page no longer says the registry is empty")
	}
	if !strings.Contains(body, `href="`+importURL+`"`) {
		t.Errorf("the empty home page does not offer the import\n--- page ---\n%s", body)
	}
}

func TestServeHome_EmptyRegistryAndNothingConfigured_OffersNoDeadEnd(t *testing.T) {
	handler := Handler(&registry.Registry{}, t.TempDir(), nil)

	body := get(t, handler, "/").Body.String()

	if strings.Contains(body, `href="`+importURL+`"`) {
		t.Error("the home page offers an import with no ROMs folder to import from")
	}
	if !strings.Contains(body, "config add-roms-folder") {
		t.Errorf("the home page does not name the command that gets the user started\n--- page ---\n%s", body)
	}
}
