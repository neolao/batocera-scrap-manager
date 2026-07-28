package webui

import (
	"net/http"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// serveComplete renders the completion flow's page, in whichever of its states
// it holds. Opening it never writes anything: the submission is what does.
func (ui *webUI) serveComplete(w http.ResponseWriter, r *http.Request) {
	render(w, http.StatusOK, completeTemplate, ui.jobPage(jobCompletion, r))
}

// startCompletion starts the completion of every configured ROMs folder and
// redirects straight back to the page, which then follows it along: the work
// outlives the request on purpose, so a dropped connection cannot cost the
// user the account of what was written to their own folders (decisions/025).
func (ui *webUI) startCompletion(w http.ResponseWriter, r *http.Request) {
	if run := ui.startJob(jobCompletion, w, r); run != nil {
		go ui.completeRomsFolders(run)
	}
}

// completeRomsFolders completes every configured ROMs folder from the registry
// snapshot as it stands when the run begins. That snapshot is captured under
// the read lock and then let go of: a correction saved meanwhile applies to a
// clone and swaps it in, leaving the pointer this run holds untouched.
//
// A folder that cannot be read stops the run: the ones already done keep their
// counts and the one that failed is named, exactly as the CLI's scrape stops.
func (ui *webUI) completeRomsFolders(run *jobHandle) {
	defer run.finish()

	ui.mu.RLock()
	reg, registryFolder := ui.reg, ui.registryFolder
	ui.mu.RUnlock()

	for _, folder := range ui.romsFolders {
		run.beginFolder(folder)

		processed, completed, failed, err := registry.CompleteRomsFolder(
			reg, folder, registryFolder, run.progress)
		report := folderReport{Folder: folder, Counts: [3]int{processed, completed, failed}}
		if err != nil {
			report.Problem = "This folder could not be completed: " + err.Error()
		}
		run.recordFolder(report)

		if err != nil {
			return
		}
	}
}

// completeTemplate renders the completion flow: the run of the other job when
// one is holding the slot, then the report of the last completion when there
// is one, then either the completion going or the confirmation offering to
// start one. The confirmation says what is rewritten rather than only what is
// gained — "completing" sounds additive, and the gamelist.xml files it
// replaces are the user's only copy.
var completeTemplate = newJobPage("complete", `
{{define "title"}}Complete the ROMs folders - Registry{{end}}
{{define "body"}}
<nav class="crumbs" aria-label="Breadcrumb">
<a href="/">Registry</a><span class="crumbs__sep">/</span><span class="crumbs__current">Complete the ROMs folders</span>
</nav>
<main>
<article class="game">
{{if .Other}}
<h2 class="game__title">Complete the ROMs folders</h2>
{{template "elsewhere" .}}
{{else if .Running}}
<h2 class="game__title">Completing the ROMs folders&hellip;</h2>
{{template "progress" .}}
{{else}}
<h2 class="game__title">Complete the ROMs folders?</h2>
{{if .Report}}
<div class="run" id="report" role="status" tabindex="-1">
<p class="run__lead">{{if .Report.Failed}}Last completion stopped on an error{{else}}Last completion finished{{end}} &mdash; started at {{.Report.StartedAt}}, took {{.Report.Elapsed}}.</p>
{{template "folders" .Report}}
{{if .Report.Totalled}}<p class="run__total">{{.Report.Summary}}</p>{{end}}
</div>
{{end}}
{{if not .Folders}}
<div class="confirm">
<p class="confirm__lead">No ROMs folder is configured, so there is nothing to write to.</p>
<p class="confirm__note">Configure one with <code>batocera-scrap-manager config add-roms-folder &lt;path&gt;</code>, then restart the server.</p>
<div class="form__actions">
<a class="button button--quiet" href="/">Back to the registry</a>
</div>
</div>
{{else}}
<div class="confirm">
<p class="confirm__lead">This sends what the registry knows back to Batocera. In each of these folders, every system's <code>gamelist.xml</code> is rewritten and the media the registry holds are copied next to the ROMs:</p>
<ul class="confirm__files">
{{range .Folders}}<li><code>{{.}}</code></li>
{{end}}
</ul>
<p class="confirm__note">Only gaps are filled: a value already present in a <code>gamelist.xml</code> is never overwritten. The registry folder itself is left unchanged — its game files and its consultation site are not touched.</p>
<p class="confirm__warning">The rewritten files are your own Batocera files: this cannot be undone.</p>
<div class="form__actions">
<form method="post" action="{{.Action}}">
<button class="button" type="submit">Complete these folders</button>
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
