package webui

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// twoRomsFolders returns two paths standing for configured ROMs folders. They
// need not exist for the confirmation page: it names what is configured, and
// only the run itself goes to the disk.
func twoRomsFolders(t *testing.T) []string {
	t.Helper()
	return []string{t.TempDir(), t.TempDir()}
}

func TestServeComplete_RomsFoldersConfigured_NamesEveryOneOfThem(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	romsFolders := twoRomsFolders(t)

	body := get(t, Handler(reg, registryFolder, romsFolders), completeURL).Body.String()

	for _, folder := range romsFolders {
		if !strings.Contains(body, folder) {
			t.Errorf("confirmation page does not name the ROMs folder %q — the user cannot check what is about to be written to", folder)
		}
	}
}

func TestServeComplete_RomsFoldersConfigured_SaysWhatIsRewrittenAndWhatIsNot(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder, twoRomsFolders(t)), completeURL).Body.String()

	if !strings.Contains(body, "gamelist.xml") {
		t.Error("confirmation page does not name gamelist.xml — rewriting it in place is the whole stake")
	}
	if !strings.Contains(strings.ToLower(body), "cannot be undone") {
		t.Error("confirmation page does not warn that the write cannot be undone")
	}
	if !strings.Contains(body, "registry") || !strings.Contains(strings.ToLower(body), "unchanged") {
		t.Error("confirmation page does not state that the registry itself is left unchanged")
	}
}

func TestServeComplete_RomsFoldersConfigured_OffersBothAWayInAndAWayOut(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder, twoRomsFolders(t)), completeURL).Body.String()

	if !strings.Contains(body, `method="post"`) || !strings.Contains(body, `action="`+completeURL+`"`) {
		t.Error("confirmation page offers no form submitting the completion")
	}
	if !strings.Contains(body, `href="/"`) {
		t.Error("confirmation page offers no way back that touches nothing")
	}
}

func TestServeComplete_NoRomsFolderConfigured_NamesTheCommandInsteadOfOfferingToRun(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	rec := get(t, Handler(reg, registryFolder, nil), completeURL)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d — having nothing configured is not an error", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if strings.Contains(body, "<button") {
		t.Error("confirmation page offers a button with no ROMs folder to write to")
	}
	if !strings.Contains(body, "config add-roms-folder") {
		t.Error("confirmation page does not name the command that configures a ROMs folder")
	}
}

func TestServeHome_LinksToTheCompletionPage(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)

	body := get(t, Handler(reg, registryFolder, twoRomsFolders(t)), "/").Body.String()

	if !strings.Contains(body, `href="`+completeURL+`"`) {
		t.Errorf("home page does not link to %s — the flow has no entry point", completeURL)
	}
}

func TestComplete_MethodNeitherGetNorPost_AnswersMethodNotAllowed(t *testing.T) {
	reg, registryFolder := fullyScrapedRegistry(t)
	rec := httptest.NewRecorder()

	Handler(reg, registryFolder, twoRomsFolders(t)).
		ServeHTTP(rec, httptest.NewRequest(http.MethodPut, completeURL, nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); allow != "GET, POST" {
		t.Errorf("Allow = %q, want %q", allow, "GET, POST")
	}
}

// romsFolderNeedingCompletion writes a ROMs folder whose gamelist.xml is
// missing the description the registry holds for one of its two games, and
// returns the registry that can fill it. Nothing references a medium, so the
// completion is a pure gamelist rewrite — which is what these tests are about.
func romsFolderNeedingCompletion(t *testing.T) (*registry.Registry, string, string) {
	t.Helper()

	romsFolder := t.TempDir()
	system := filepath.Join(romsFolder, "megadrive")
	if err := os.MkdirAll(system, 0o755); err != nil {
		t.Fatalf("failed to set up the ROMs folder: %v", err)
	}
	localGames := []gamelist.Game{
		{Path: "./Sonic.zip", Name: "Sonic the Hedgehog"},
		{Path: "./Streets.zip", Name: "Streets of Rage", Desc: "Three fighters clean up the city."},
	}
	if err := gamelist.WriteFile(filepath.Join(system, "gamelist.xml"), localGames); err != nil {
		t.Fatalf("failed to write the gamelist: %v", err)
	}

	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{
			Path:  "./Sonic.zip",
			Name:  "Sonic the Hedgehog",
			Desc:  "A blue hedgehog runs very fast.",
			Genre: "Platform",
		}},
	}}
	return reg, t.TempDir(), romsFolder
}

// completedGamelist reads back the gamelist.xml the completion rewrote.
func completedGamelist(t *testing.T, romsFolder string) []gamelist.Game {
	t.Helper()

	games, err := gamelist.ParseFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"))
	if err != nil {
		t.Fatalf("reading the completed gamelist: %v", err)
	}
	return games
}

// folderFingerprint hashes every file of folder with its path, so a test can
// assert nothing under it changed — neither content nor which files exist.
func folderFingerprint(t *testing.T, folder string) string {
	t.Helper()

	sum := sha256.New()
	err := filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		fmt.Fprintf(sum, "%s:%x\n", path, sha256.Sum256(content))
		return nil
	})
	if err != nil {
		t.Fatalf("fingerprinting %q: %v", folder, err)
	}
	return fmt.Sprintf("%x", sum.Sum(nil))
}

// waitForCompletion polls the completion page until the run it reports is
// over — the page stops carrying its auto-reload exactly then — and returns
// the final page, so a test asserts on a report rather than on a moving one.
func waitForCompletion(t *testing.T, h http.Handler) string {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		body := get(t, h, completeURL).Body.String()
		if !strings.Contains(body, `http-equiv="refresh"`) {
			return body
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("the completion was still running after 10s")
	return ""
}

func TestStartCompletion_Submitted_RedirectsToTheCompletionPage(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	rec := post(t, handler, completeURL, nil)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if location := rec.Header().Get("Location"); location != completeURL {
		t.Errorf("Location = %q, want %q", location, completeURL)
	}
	waitForCompletion(t, handler)
}

func TestStartCompletion_SubmissionFromAnotherSite_IsRefused(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)
	before := folderFingerprint(t, romsFolder)
	r := httptest.NewRequest(http.MethodPost, completeURL, nil)
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()

	Handler(reg, registryFolder, []string{romsFolder}).ServeHTTP(rec, r)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if after := folderFingerprint(t, romsFolder); after != before {
		t.Error("a refused submission wrote to the ROMs folder anyway")
	}
}

func TestStartCompletion_RunFinished_ReportsTheSameSummaryAsTheCommandLine(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	post(t, handler, completeURL, nil)
	body := waitForCompletion(t, handler)

	// Two games looked at, the one the registry could fill actually filled.
	want := "2 processed, 1 completed, 0 failed"
	if !strings.Contains(body, want) {
		t.Errorf("report does not carry %q\n--- page ---\n%s", want, body)
	}
}

func TestStartCompletion_RunFinished_FillsTheGamelistFromTheRegistry(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	post(t, handler, completeURL, nil)
	waitForCompletion(t, handler)

	games := completedGamelist(t, romsFolder)
	if len(games) != 2 {
		t.Fatalf("len(games) = %d, want 2", len(games))
	}
	if games[0].Desc != "A blue hedgehog runs very fast." {
		t.Errorf("Sonic.Desc = %q, want it filled from the registry", games[0].Desc)
	}
	if games[0].Genre != "Platform" {
		t.Errorf("Sonic.Genre = %q, want %q", games[0].Genre, "Platform")
	}
	if games[1].Desc != "Three fighters clean up the city." {
		t.Errorf("Streets.Desc = %q, want the local value left alone", games[1].Desc)
	}
}

func TestStartCompletion_RunFinished_LeavesTheRegistryFolderUntouched(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)
	writeMediaFile(t, registryFolder, "megadrive", "images/sonic.png")
	handler := Handler(reg, registryFolder, []string{romsFolder})
	before := folderFingerprint(t, registryFolder)

	post(t, handler, completeURL, nil)
	waitForCompletion(t, handler)

	if after := folderFingerprint(t, registryFolder); after != before {
		t.Error("the completion wrote to the registry folder — it must only ever read it")
	}
}

func TestStartCompletion_OneFolderMissing_NamesItAndKeepsWhatTheOthersDid(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)
	missing := filepath.Join(t.TempDir(), "unplugged-sd-card")
	handler := Handler(reg, registryFolder, []string{romsFolder, missing})

	post(t, handler, completeURL, nil)
	body := waitForCompletion(t, handler)

	if !strings.Contains(body, missing) {
		t.Errorf("report does not name the folder that could not be read (%q)\n--- page ---\n%s", missing, body)
	}
	if !strings.Contains(body, "2 processed, 1 completed, 0 failed") {
		t.Errorf("report drops what the folder processed before the failure\n--- page ---\n%s", body)
	}
	if !strings.Contains(body, romsFolder) {
		t.Error("report does not name the folder that was processed")
	}
}

func TestServeComplete_NothingRanYet_ShowsTheConfirmationAndNoReport(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)

	body := get(t, Handler(reg, registryFolder, []string{romsFolder}), completeURL).Body.String()

	if strings.Contains(body, "processed") {
		t.Error("page reports a run that never happened")
	}
	if !strings.Contains(body, "Complete these folders") {
		t.Error("page does not offer to run the completion")
	}
}

func TestServeComplete_RunInProgress_ReloadsItselfAndOffersNoSecondRun(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)
	ui := &webUI{reg: reg, registryFolder: registryFolder, romsFolders: []string{romsFolder}}
	if !ui.completion.start() {
		t.Fatal("the completion slot should have been free")
	}
	rec := httptest.NewRecorder()

	ui.serveComplete(rec, httptest.NewRequest(http.MethodGet, completeURL, nil))

	body := rec.Body.String()
	if !strings.Contains(body, `http-equiv="refresh"`) {
		t.Error("a running completion does not reload its page, so it never shows its own end")
	}
	if strings.Contains(body, "<button") {
		t.Error("a running completion still offers to start another one")
	}
}

func TestServeComplete_RunOver_StopsReloadingAndOffersToRunAgain(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	post(t, handler, completeURL, nil)
	body := waitForCompletion(t, handler)

	if strings.Contains(body, `http-equiv="refresh"`) {
		t.Error("the report reloads itself forever, which makes it unreadable")
	}
	if !strings.Contains(body, "Complete these folders") {
		t.Error("the report does not offer to run the completion again")
	}
}

func TestCompletionSlot_AlreadyTaken_RefusesASecondRunUntilTheFirstIsOver(t *testing.T) {
	var state completionState

	if !state.start() {
		t.Fatal("the first run could not take the free slot")
	}
	if state.start() {
		t.Error("a second run started while the first was still going — the same gamelist.xml would be written twice at once")
	}

	state.finish()

	if !state.start() {
		t.Error("the slot stayed taken after the run ended, so no further completion is possible")
	}
}

func TestCompletionSlot_RunOver_KeepsItsReportReadable(t *testing.T) {
	var state completionState
	state.start()
	state.recordFolder(folderReport{Folder: "/roms", Processed: 7, Completed: 3})
	state.finish()

	report := state.snapshot()

	if report == nil {
		t.Fatal("the report is gone once the run ended — closing the tab would lose it for good")
	}
	if report.Running {
		t.Error("the report still claims the run is going")
	}
	if len(report.Folders) != 1 || report.Folders[0].Processed != 7 {
		t.Errorf("report.Folders = %+v, want the folder's own counts kept", report.Folders)
	}
	if report.Summary != "7 processed, 3 completed, 0 failed" {
		t.Errorf("report.Summary = %q, want the totals worded as the command line words them", report.Summary)
	}
}

func TestStartCompletion_WhileTheRegistryIsBeingCorrected_NeitherBlocksTheOther(t *testing.T) {
	// Run with -race. The completion reads the served snapshot for its whole
	// duration while corrections replace it: it must take the read lock only
	// long enough to capture the pointer, never for the run (see decisions/025).
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		post(t, handler, completeURL, nil)
	}()
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			get(t, handler, "/")
			get(t, handler, "/system/megadrive")
			get(t, handler, completeURL)
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		form := url.Values{}
		form.Set("path", "./Sonic.zip")
		form.Set("name", "Sonic the Hedgehog")
		form.Set("desc", "Corrected while the completion was running.")
		form.Set("rating", "")
		post(t, handler, "/game/megadrive/Sonic/edit", form)
	}()
	wg.Wait()

	waitForCompletion(t, handler)

	// The run worked off the snapshot it captured, so the gamelist holds what
	// the registry said when it started — not a half-applied correction.
	games := completedGamelist(t, romsFolder)
	if games[0].Desc == "" {
		t.Error("the completion filled nothing while a correction was being saved")
	}
}

func TestServeComplete_SingleFolderRun_DoesNotRepeatItsCountsAsATotal(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	post(t, handler, completeURL, nil)
	body := waitForCompletion(t, handler)

	if count := strings.Count(body, "2 processed, 1 completed, 0 failed"); count != 1 {
		t.Errorf("the run's counts appear %d times, want 1 — with one folder the total is the folder's own line", count)
	}
}

func TestServeComplete_SeveralFoldersRun_AddsTheirCountsUp(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)
	_, _, otherFolder := romsFolderNeedingCompletion(t)
	handler := Handler(reg, registryFolder, []string{romsFolder, otherFolder})

	post(t, handler, completeURL, nil)
	body := waitForCompletion(t, handler)

	// The registry only knows the first folder's Sonic by name, and both folders
	// hold the same two games — so four looked at, one filled in each.
	if !strings.Contains(body, "4 processed, 2 completed, 0 failed") {
		t.Errorf("the report does not total the folders up\n--- page ---\n%s", body)
	}
}
