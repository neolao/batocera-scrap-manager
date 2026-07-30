package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

func TestJobSlot_TakenByOneKind_RefusesTheOtherKindToo(t *testing.T) {
	// The two long operations flow in opposite directions over the same files:
	// one slot for both, so no page has to explain which run won (decisions/027).
	var state jobs

	first := state.start(jobCompletion)
	if first == nil {
		t.Fatal("the first run could not take the free slot")
	}
	if state.start(jobImport) != nil {
		t.Error("an import started while a completion was going — each would read what the other is mid-writing")
	}
	if state.start(jobCompletion) != nil {
		t.Error("a second completion started while the first was still going")
	}

	first.finish()

	if state.start(jobImport) == nil {
		t.Error("the slot stayed taken once the run ended, so no further job could ever run")
	}
}

func TestJobsView_OtherKindRunning_NamesItAndLinksToIt(t *testing.T) {
	var state jobs
	state.start(jobImport)

	view := state.view(jobCompletion)

	if view.Other != jobImport {
		t.Errorf("view.Other = %q, want %q — the page cannot say what is holding the slot", view.Other, jobImport)
	}
	if view.Report != nil {
		t.Error("the completion is shown a report belonging to the import")
	}
}

func TestJobsView_OwnKindRunning_LeavesTheSlotUnclaimedForThatPage(t *testing.T) {
	var state jobs
	state.start(jobImport)

	view := state.view(jobImport)

	if view.Other != "" {
		t.Errorf("view.Other = %q, want empty — a job does not report itself as somebody else's", view.Other)
	}
	if view.Report == nil || !view.Report.Running {
		t.Fatal("a job's own page cannot see its own run")
	}
}

func TestJobsView_RunOver_KeepsItsReportReadable(t *testing.T) {
	var state jobs
	run := state.start(jobCompletion)
	run.recordFolder(folderReport{Folder: "/roms", Counts: [3]int{7, 3, 0}})
	run.finish()

	report := state.view(jobCompletion).Report

	if report == nil {
		t.Fatal("the report is gone once the run ended — closing the tab would lose it for good")
	}
	if report.Running {
		t.Error("the report still claims the run is going")
	}
	if len(report.Folders) != 1 || report.Folders[0].Summary != "7 processed, 3 completed, 0 failed" {
		t.Errorf("report.Folders = %+v, want the folder's counts worded as the command line words them", report.Folders)
	}
	if report.Totals != [3]int{7, 3, 0} {
		t.Errorf("report.Totals = %v, want the folder's own counts", report.Totals)
	}
}

func TestJobsView_ImportRun_WordsItsCountsAsTheUpdateCommandDoes(t *testing.T) {
	// The import's three counts are not the completion's: one machine, two
	// vocabularies, each owned by the package holding the values.
	var state jobs
	run := state.start(jobImport)
	run.recordFolder(folderReport{Folder: "/roms", Counts: [3]int{4, 1, 12}})
	run.finish()

	report := state.view(jobImport).Report

	want := "4 added, 1 updated, 12 unchanged"
	if len(report.Folders) != 1 || report.Folders[0].Summary != want {
		t.Errorf("report.Folders = %+v, want the counts worded %q", report.Folders, want)
	}
	if report.Summary != want {
		t.Errorf("report.Summary = %q, want %q", report.Summary, want)
	}
}

func TestJobsView_SingleFolder_DoesNotRepeatItsCountsAsATotal(t *testing.T) {
	var state jobs
	run := state.start(jobImport)
	run.recordFolder(folderReport{Folder: "/roms", Counts: [3]int{1, 0, 0}})
	run.finish()

	if state.view(jobImport).Report.Totalled {
		t.Error("a run over a single folder totals its own only line, which reads as a stutter")
	}
}

func TestJobsView_SeveralFolders_AddsTheirCountsUp(t *testing.T) {
	var state jobs
	run := state.start(jobImport)
	run.recordFolder(folderReport{Folder: "/roms", Counts: [3]int{1, 2, 3}})
	run.recordFolder(folderReport{Folder: "/other", Counts: [3]int{10, 20, 30}})
	run.finish()

	report := state.view(jobImport).Report

	if !report.Totalled {
		t.Error("a run over several folders says nothing about the whole")
	}
	if report.Totals != [3]int{11, 22, 33} {
		t.Errorf("report.Totals = %v, want %v", report.Totals, [3]int{11, 22, 33})
	}
}

func TestJobRun_Failed_ReportsTheFailureApartFromTheFolderCounts(t *testing.T) {
	var state jobs
	run := state.start(jobImport)
	run.recordFolder(folderReport{Folder: "/roms", Counts: [3]int{2, 0, 0}})
	run.fail("The registry could not be written: disk full")
	run.finish()

	report := state.view(jobImport).Report

	if report.Problem != "The registry could not be written: disk full" {
		t.Errorf("report.Problem = %q, want the failure of the run itself", report.Problem)
	}
	if !report.Failed {
		t.Error("report.Failed is false although the run failed")
	}
}

func TestJobRun_FolderProblem_MarksTheReportFailedAndNamesTheFolder(t *testing.T) {
	var state jobs
	run := state.start(jobCompletion)
	run.recordFolder(folderReport{Folder: "/unplugged", Problem: "This folder could not be read"})
	run.finish()

	report := state.view(jobCompletion).Report

	if !report.Failed {
		t.Error("a folder that stopped the run leaves the report reading as a success")
	}
	if report.Folders[0].Problem == "" || report.Folders[0].Folder != "/unplugged" {
		t.Errorf("report.Folders[0] = %+v, want the failing folder named with its problem", report.Folders[0])
	}
}

func TestJobRun_Caveat_IsNotAFailure(t *testing.T) {
	// A stale consultation site is a reservation on a change that did apply —
	// telling the user it failed would have them redo what already happened.
	var state jobs
	run := state.start(jobImport)
	run.note("The consultation site could not be regenerated.")
	run.finish()

	report := state.view(jobImport).Report

	if report.Failed {
		t.Error("a caveat is reported as a failure")
	}
	if report.Caveat != "The consultation site could not be regenerated." {
		t.Errorf("report.Caveat = %q, want the reservation kept", report.Caveat)
	}
}

func TestJobRun_InProgress_SaysWhichFolderAndGameItIsOn(t *testing.T) {
	var state jobs
	run := state.start(jobCompletion)
	run.beginFolder("/roms")
	run.progress(registry.ProgressEvent{System: "megadrive", GameName: "Sonic the Hedgehog"})

	current := state.view(jobCompletion).Report.Current

	if !strings.Contains(current, "/roms") || !strings.Contains(current, "Sonic the Hedgehog") {
		t.Errorf("Current = %q, want the folder and the game being processed", current)
	}
}

func TestJobRun_NewFolder_DropsThePreviousFoldersGame(t *testing.T) {
	var state jobs
	run := state.start(jobCompletion)
	run.beginFolder("/roms")
	run.progress(registry.ProgressEvent{System: "megadrive", GameName: "Sonic the Hedgehog"})
	run.beginFolder("/other")

	current := state.view(jobCompletion).Report.Current

	if strings.Contains(current, "Sonic the Hedgehog") {
		t.Errorf("Current = %q, want the previous folder's game dropped — it is not being processed any more", current)
	}
	if !strings.Contains(current, "/other") {
		t.Errorf("Current = %q, want the folder the run moved on to", current)
	}
}

func TestServeComplete_RunInProgress_StatesHowOftenItReloadsAndOffersToStopIt(t *testing.T) {
	// A page reloading itself with no way out resets whoever is reading it every
	// five seconds — a screen reader never reaches the end of the folder list.
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)
	ui := &webUI{reg: reg, registryFolder: registryFolder, romsFolders: []string{romsFolder}}
	ui.jobs.start(jobCompletion)
	rec := httptest.NewRecorder()

	ui.serveComplete(rec, httptest.NewRequest(http.MethodGet, completeURL, nil))

	body := rec.Body.String()
	if !strings.Contains(body, `http-equiv="refresh"`) {
		t.Error("a running completion does not reload its page, so it never shows its own end")
	}
	if !strings.Contains(body, "every "+refreshSeconds+" seconds") {
		t.Errorf("the page does not say how often it reloads\n--- page ---\n%s", body)
	}
	if !strings.Contains(body, `href="`+completeURL+`?`+quietParam+`"`) {
		t.Errorf("the page offers no way to stop the reloading\n--- page ---\n%s", body)
	}
}

func TestServeComplete_ReloadingTurnedOff_ShowsTheSameRunWithoutReloading(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)
	ui := &webUI{reg: reg, registryFolder: registryFolder, romsFolders: []string{romsFolder}}
	run := ui.jobs.start(jobCompletion)
	run.beginFolder(romsFolder)
	rec := httptest.NewRecorder()

	ui.serveComplete(rec, httptest.NewRequest(http.MethodGet, completeURL+"?"+quietParam, nil))

	body := rec.Body.String()
	if strings.Contains(body, `http-equiv="refresh"`) {
		t.Error("the page reloads itself although the reloading was turned off")
	}
	if !strings.Contains(body, romsFolder) {
		t.Error("the quiet page dropped the run it is meant to show")
	}
	if !strings.Contains(body, `href="`+completeURL+`"`) {
		t.Error("the quiet page offers no way back to the reloading one")
	}
}

func TestServeComplete_ImportRunning_SaysSoAndOffersNoRunOfItsOwn(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)
	ui := &webUI{reg: reg, registryFolder: registryFolder, romsFolders: []string{romsFolder}}
	ui.jobs.start(jobImport)
	rec := httptest.NewRecorder()

	ui.serveComplete(rec, httptest.NewRequest(http.MethodGet, completeURL, nil))

	body := rec.Body.String()
	if strings.Contains(body, "<button") {
		t.Error("the completion offers to start while an import holds the slot")
	}
	if !strings.Contains(body, importURL) {
		t.Errorf("the page does not link to the operation that is holding the slot\n--- page ---\n%s", body)
	}
	if !strings.Contains(strings.ToLower(body), "import") {
		t.Error("the page does not name the operation in progress, leaving the missing button unexplained")
	}
}

func TestStartCompletion_ImportRunning_StartsNothing(t *testing.T) {
	reg, registryFolder, romsFolder := romsFolderNeedingCompletion(t)
	ui := &webUI{reg: reg, registryFolder: registryFolder, romsFolders: []string{romsFolder}}
	ui.jobs.start(jobImport)
	before := folderFingerprint(t, romsFolder)
	rec := httptest.NewRecorder()

	ui.startCompletion(rec, httptest.NewRequest(http.MethodPost, completeURL, nil))

	if rec.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d — the page it lands on is what words the refusal", rec.Code, http.StatusSeeOther)
	}
	if after := folderFingerprint(t, romsFolder); after != before {
		t.Error("a completion ran while an import held the slot")
	}
}

func TestRecoverJob_PanicDuringWork_FailsTheRunAndFreesTheSlot(t *testing.T) {
	// A registry or a ROMs folder large enough to panic somewhere deep inside
	// an import or a completion must not take the whole server down with it —
	// every other request being served depends on that same goroutine dying
	// quietly instead.
	var state jobs
	run := state.start(jobImport)

	func() {
		defer run.finish()
		defer recoverJob(run)
		panic("something went badly wrong")
	}()

	if state.active != "" {
		t.Errorf("active = %q, want the slot freed after a panic", state.active)
	}
	problem := state.runs[jobImport].problem
	if !strings.Contains(strings.ToLower(problem), "internal error") {
		t.Errorf("problem = %q, want it naming an internal error", problem)
	}
	if !strings.Contains(problem, "something went badly wrong") {
		t.Errorf("problem = %q, want it carrying the panic's own message", problem)
	}
}
