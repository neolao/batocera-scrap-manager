package webui

import (
	"errors"
	"net/http"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/store"
)

// importPage is the import flow's page: the shared job page, plus the one
// thing no table can hold — the sentence saying how the last run turned out.
// A row of zeroes is the commonest outcome there is, and left to speak for
// itself it reads as a failure rather than as a registry already in step.
type importPage struct {
	jobPage
	Verdict string
	// ZeroSummary words an import that never happened, for the one state that
	// has counts to show without a run behind them: nothing configured to
	// import from.
	ZeroSummary string
}

// importVerdict words what the last import amounts to, before its counts
// detail it. The zero cases are told apart deliberately: a registry already
// holding everything is not the same news as folders holding nothing.
func importVerdict(report *jobReport) string {
	switch {
	case report.Failed:
		return "Last import stopped on an error"
	case report.Totals == [3]int{}:
		return "Last import finished — these folders held nothing to import"
	case report.Totals[0] == 0 && report.Totals[1] == 0:
		return "Last import finished — the registry was already up to date"
	default:
		return "Last import finished"
	}
}

// serveImport renders the import flow's page, in whichever of its states it
// holds. Opening it never writes anything: the submission is what does.
func (ui *webUI) serveImport(w http.ResponseWriter, r *http.Request) {
	page := importPage{
		jobPage:     ui.jobPage(jobImport, r),
		ZeroSummary: summarize(registry.ImportSummaryFormat, [3]int{}),
	}
	if page.Report != nil {
		page.Verdict = importVerdict(page.Report)
	}
	render(w, http.StatusOK, importTemplate, page)
}

// startImport starts the import of every configured ROMs folder and redirects
// straight back to the page, which then follows it along: a registry of
// several thousand games takes minutes to walk, far longer than a request may
// reasonably hold (decisions/025).
func (ui *webUI) startImport(w http.ResponseWriter, r *http.Request) {
	if run := ui.startJob(jobImport, w, r); run != nil {
		go ui.importRomsFolders(run)
	}
}

// importRomsFolders imports every configured ROMs folder into a clone of the
// served registry snapshot, persists that clone once every folder is through,
// and only then swaps it in — so a write that fails leaves the served pages
// and the disk agreeing.
//
// A folder that cannot be read stops the run and nothing is persisted at all,
// exactly as the update command saves nothing when a folder fails: the counts
// the earlier folders reached are kept as an account of how far it got, never
// as a claim that they were saved (decisions/028).
func (ui *webUI) importRomsFolders(run *jobHandle) {
	defer run.finish()
	defer recoverJob(run)

	ui.mu.RLock()
	reg, registryFolder := ui.reg, ui.registryFolder
	ui.mu.RUnlock()

	// The clone is what makes the whole run safe to do outside the lock: a
	// correction saved meanwhile applies to a clone of its own and swaps it in,
	// and this run's own writes touch nothing anybody is reading.
	candidate := reg.Clone()
	for _, folder := range ui.romsFolders {
		run.beginFolder(folder)

		added, updated, unchanged, err := registry.ImportFromRomsFolder(
			candidate, folder, registryFolder, run.progress)
		report := folderReport{Folder: folder, Counts: [3]int{added, updated, unchanged}}
		if err != nil {
			report.Problem = "This folder could not be imported."
		}
		run.recordFolder(report)

		if err != nil {
			run.fail("Nothing was written: the import stopped before the registry was saved.")
			return
		}
	}

	ui.commitImport(run, reg, candidate)
}

// commitImport persists what the run built and swaps it in — under the write
// lock for the whole of it, as every other change to the registry is (see
// save.go), so no correction can land between the write and the swap.
//
// captured is the snapshot the run worked off. A correction saved from the web
// UI while the run was going replaces that snapshot with one this candidate
// knows nothing about: writing the candidate out would erase, without a word,
// a value the user had just typed by hand. The import stands down instead —
// its work is a read of files still there, and running it again costs nothing
// but time.
func (ui *webUI) commitImport(run *jobHandle, captured, candidate *registry.Registry) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	if ui.reg != captured {
		run.fail("Nothing was written: the registry was corrected from these pages while the import was running, " +
			"and overwriting that correction is not something an import may do. Run it again to take both into account.")
		return
	}

	if err := store.Save(candidate, ui.registryFolder); err != nil {
		if !errors.Is(err, store.ErrSiteNotRegenerated) {
			run.fail("The registry could not be written.")
			return
		}
		// The registry is the source of truth and it was written: the games are
		// there, and only the site derived from them is behind. Saying the
		// import failed would have the user redo what already happened.
		run.note("The games are in the registry, but the consultation site could not be regenerated.")
	}
	ui.reg = candidate
}

// importTemplate renders the import flow: the run of the other job when one is
// holding the slot, then the report of the last import when there is one, then
// either the import going or the page offering to start one. That page is a
// briefing rather than a warning: unlike the completion, this reads the ROMs
// folders and writes only into the registry, which the tool can rebuild.
var importTemplate = newJobPage("import", `
{{define "title"}}Import from the ROMs folders - Registry{{end}}
{{define "body"}}
<nav class="crumbs" aria-label="Breadcrumb">
<a href="/">Registry</a><span class="crumbs__sep">/</span><span class="crumbs__current">Import from the ROMs folders</span>
</nav>
<main>
<article class="game">
{{if .Other}}
<h2 class="game__title">Import from the ROMs folders</h2>
{{template "elsewhere" .}}
{{else if .Running}}
<h2 class="game__title">Importing from the ROMs folders&hellip;</h2>
{{template "progress" .}}
{{else}}
<h2 class="game__title">Import from the ROMs folders?</h2>
{{if .Report}}
<div class="run" id="report" role="status" tabindex="-1">
<p class="run__lead">{{.Verdict}} &mdash; started at {{.Report.StartedAt}}, took {{.Report.Elapsed}}.</p>
{{template "folders" .Report}}
{{if .Report.Totalled}}<p class="run__total">{{.Report.Summary}}</p>{{end}}
{{if .Report.Problem}}<p class="run__problem">{{.Report.Problem}}</p>{{end}}
{{if .Report.Caveat}}<p class="run__caveat">{{.Report.Caveat}}</p>{{end}}
</div>
{{end}}
{{if not .Folders}}
<div class="confirm">
<p class="confirm__lead">No ROMs folder is configured, so there is nothing to import from.</p>
<p class="run__total">{{.ZeroSummary}}</p>
<p class="confirm__note">Configure one with <code>batocera-scrap-manager config add-roms-folder &lt;path&gt;</code>, then restart the server.</p>
<div class="form__actions">
<a class="button button--quiet" href="/">Back to the registry</a>
</div>
</div>
{{else}}
<div class="confirm">
<p class="confirm__lead">This brings into the registry what Batocera already knows. In each of these folders, every system's <code>gamelist.xml</code> is read and the media it references are copied into the registry:</p>
<ul class="confirm__files">
{{range .Folders}}<li><code>{{.}}</code></li>
{{end}}
</ul>
<p class="confirm__note">These folders are only read &mdash; they are left as they are. A game the registry already holds is refreshed from what the folder says, except for the values corrected by hand, which an import never overwrites. A game with neither a description nor a cover art has not really been scraped yet, and is not imported.</p>
<p class="confirm__note">The registry's game files are rewritten and its consultation site is regenerated, so the pages served here show the result straight away.</p>
<div class="form__actions">
<form method="post" action="{{.Action}}">
<button class="button" type="submit">Import these folders</button>
</form>
<a class="button button--quiet" href="/">Cancel</a>
</div>
</div>
{{end}}
{{end}}
</article>
</main>
{{end}}
`)
