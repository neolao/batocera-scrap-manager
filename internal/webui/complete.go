package webui

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// completeURL is where the whole completion flow lives — the confirmation
// asking to run it, the run in progress, and the report of the last one. One
// URL rather than one per state: "run it again" is then the very control that
// started it (see decisions/025).
const completeURL = "/complete"

// refreshSeconds is how long the running page waits before reloading itself.
// The UI ships no JavaScript, so a meta refresh is the only way a run can ever
// show its own end; it is emitted while the run goes and never after, or the
// report would keep stealing the focus from whoever is reading it.
const refreshSeconds = "5"

// completionState is the one completion the server tracks: whether one is
// running, and what the last one did. It holds its own mutex rather than
// sharing the registry's — a run lasts minutes, and keeping the snapshot lock
// that long would queue every served page behind the first correction
// submitted meanwhile (see decisions/025).
type completionState struct {
	mu      sync.Mutex
	running bool
	run     *completionRun
}

// completionRun is what one completion has to say for itself: where it is
// while it goes, and what it did once it is over.
type completionRun struct {
	startedAt time.Time
	endedAt   time.Time
	folder    string
	system    string
	game      string
	folders   []folderReport
}

// folderReport is one ROMs folder's outcome. Problem is empty when the folder
// was completed to the end: a folder that stopped the run keeps the counts it
// reached before failing, so the report says what was really done rather than
// dropping it.
type folderReport struct {
	Folder    string
	Processed int
	Completed int
	Failed    int
	Problem   string
}

// Summary words one folder's counts exactly as the totals are worded, and as
// the command line words its own — one sentence pattern for the whole
// operation, never a second copy drifting from the first.
func (r folderReport) Summary() string {
	return fmt.Sprintf(registry.CompletionSummaryFormat, r.Processed, r.Completed, r.Failed)
}

// completionReport is a completion as a page shows it — a copy taken under the
// lock, so a template never reads a field the run is writing.
type completionReport struct {
	Running   bool
	StartedAt string
	Elapsed   string
	Current   string
	Folders   []folderReport
	Summary   string
	Failed    bool
}

// Totalled reports whether the totals say anything the per-folder lines do not.
// With a single ROMs folder they are the same sentence twice, which reads as a
// stutter rather than as a total.
func (r completionReport) Totalled() bool {
	return len(r.Folders) > 1
}

// start takes the completion slot for a new run, reporting whether it was
// free. A run already going keeps it: two completions writing the same
// gamelist.xml at once is the one thing this guards against.
func (c *completionState) start() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return false
	}
	c.running = true
	c.run = &completionRun{startedAt: time.Now()}
	return true
}

// beginFolder records which ROMs folder the run moved on to. It is what proves
// the run is advancing when nothing else does: the domain only reports a game
// when it actually fills one, so a registry already in step would leave the
// game line frozen for minutes.
func (c *completionState) beginFolder(folder string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.run.folder = folder
	c.run.system, c.run.game = "", ""
}

// progress records the game the run is filling right now.
func (c *completionState) progress(event registry.CompletionEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.run.system, c.run.game = event.System, event.GameName
}

// recordFolder adds one folder's outcome to the report being built.
func (c *completionState) recordFolder(report folderReport) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.run.folders = append(c.run.folders, report)
}

// finish frees the slot and closes the report, which stays readable until the
// next run replaces it — the operation writes outside the registry with no way
// back, so its account must outlive the browser tab that started it.
func (c *completionState) finish() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.running = false
	c.run.endedAt = time.Now()
	c.run.folder, c.run.system, c.run.game = "", "", ""
}

// snapshot copies the current state of the completion for one page to render,
// or nil when none ever ran.
func (c *completionState) snapshot() *completionReport {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.run == nil {
		return nil
	}

	var processed, completed, failed int
	anyProblem := false
	for _, folder := range c.run.folders {
		processed += folder.Processed
		completed += folder.Completed
		failed += folder.Failed
		anyProblem = anyProblem || folder.Problem != ""
	}

	until := c.run.endedAt
	if c.running {
		until = time.Now()
	}
	return &completionReport{
		Running:   c.running,
		StartedAt: c.run.startedAt.Format("15:04:05"),
		Elapsed:   until.Sub(c.run.startedAt).Round(time.Second).String(),
		Current:   currentStep(c.run),
		Folders:   append([]folderReport(nil), c.run.folders...),
		Summary:   fmt.Sprintf(registry.CompletionSummaryFormat, processed, completed, failed),
		Failed:    anyProblem,
	}
}

// currentStep words where a run has got to, down to the game it is filling
// when one is known.
func currentStep(run *completionRun) string {
	if run.folder == "" {
		return ""
	}
	if run.game == "" {
		return run.folder
	}
	return run.folder + " — " + run.system + ": " + run.game
}

// completePage is the completion flow's page, in whichever of its states it
// happens to be. Folders is what will be written to, named in full: this is
// the one flow reaching outside the registry, into files no version control
// would bring back. Report is nil until a first run has something to say.
type completePage struct {
	Folders []string
	Action  string
	Report  *completionReport
}

// Running reports whether a run is going, which is what decides between the
// two things this page can offer: watching, or starting one.
func (p completePage) Running() bool {
	return p.Report != nil && p.Report.Running
}

// serveComplete renders the completion flow's page. Opening it never writes
// anything: the submission is what does.
func (ui *webUI) serveComplete(w http.ResponseWriter, r *http.Request) {
	render(w, http.StatusOK, completeTemplate, completePage{
		Folders: ui.romsFolders,
		Action:  completeURL,
		Report:  ui.completion.snapshot(),
	})
}

// startCompletion starts the completion of every configured ROMs folder and
// redirects straight back to the page, which then follows it along: the work
// outlives the request on purpose, so a dropped connection cannot cost the
// user the account of what was written to their own folders (decisions/025).
func (ui *webUI) startCompletion(w http.ResponseWriter, r *http.Request) {
	if crossSite(r) {
		renderProblem(w, http.StatusForbidden, "Refused",
			"This request did not come from the registry's own pages.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		renderProblem(w, http.StatusBadRequest, "Bad request", "This request could not be read.")
		return
	}

	// The slot is taken here rather than inside the goroutine: a second
	// submission arriving before the first one was scheduled would otherwise
	// find it free and set two runs on the same gamelist.xml. Nothing
	// configured starts nothing — the page says why.
	if len(ui.romsFolders) > 0 && ui.completion.start() {
		go ui.completeRomsFolders()
	}
	http.Redirect(w, r, completeURL, http.StatusSeeOther)
}

// completeRomsFolders completes every configured ROMs folder from the registry
// snapshot as it stands when the run begins. That snapshot is captured under
// the read lock and then let go of: a correction saved meanwhile applies to a
// clone and swaps it in, leaving the pointer this run holds untouched.
//
// A folder that cannot be read stops the run: the ones already done keep their
// counts and the one that failed is named, exactly as the CLI's scrape stops.
func (ui *webUI) completeRomsFolders() {
	defer ui.completion.finish()

	ui.mu.RLock()
	reg, registryFolder := ui.reg, ui.registryFolder
	ui.mu.RUnlock()

	for _, folder := range ui.romsFolders {
		ui.completion.beginFolder(folder)

		processed, completed, failed, err := registry.CompleteRomsFolder(
			reg, folder, registryFolder, ui.completion.progress)
		report := folderReport{Folder: folder, Processed: processed, Completed: completed, Failed: failed}
		if err != nil {
			report.Problem = "This folder could not be completed: " + err.Error()
		}
		ui.completion.recordFolder(report)

		if err != nil {
			return
		}
	}
}

// serveWrongCompleteMethod refuses a method the completion URL does not
// support, naming the ones it does. It needs its own handler because the
// catch-all route matches every method, so the mux's own method-not-allowed
// handling would never fire for this URL.
func (ui *webUI) serveWrongCompleteMethod(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
	renderProblem(w, http.StatusMethodNotAllowed, "Not allowed",
		r.Method+" is not something this page accepts.")
}

// completeTemplate renders the completion flow: the report of the last run
// when there is one, then either the run going or the confirmation offering to
// start one. The confirmation says what is rewritten rather than only what is
// gained — "completing" sounds additive, and the gamelist.xml files it
// replaces are the user's only copy.
var completeTemplate = newPage("complete", `
{{define "head"}}{{if .Running}}<meta http-equiv="refresh" content="`+refreshSeconds+`">{{end}}{{end}}
{{define "title"}}Complete the ROMs folders - Registry{{end}}
{{define "body"}}
<nav class="crumbs" aria-label="Breadcrumb">
<a href="/">Registry</a><span class="crumbs__sep">/</span><span class="crumbs__current">Complete the ROMs folders</span>
</nav>
<main>
<article class="game">
{{if .Running}}
<h2 class="game__title">Completing the ROMs folders&hellip;</h2>
<div class="run">
<p class="run__lead" role="status">Started at {{.Report.StartedAt}}, running for {{.Report.Elapsed}}.</p>
{{if .Report.Current}}<p class="run__current"><code>{{.Report.Current}}</code></p>{{end}}
{{template "folders" .Report}}
<p class="run__note">This page reloads on its own until the completion is over. Leaving it does not stop anything — come back to this address to read the report.</p>
<div class="form__actions">
<a class="button button--quiet" href="`+completeURL+`">Reload now</a>
</div>
</div>
{{else}}
<h2 class="game__title">Complete the ROMs folders?</h2>
{{if .Report}}
<div class="run" id="report" role="status" tabindex="-1">
<p class="run__lead">{{if .Report.Failed}}Last completion stopped on an error{{else}}Last completion finished{{end}} — started at {{.Report.StartedAt}}, took {{.Report.Elapsed}}.</p>
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

{{define "folders"}}
{{if .Folders}}
<ul class="run__folders">
{{range .Folders}}
<li>
<code>{{.Folder}}</code>
<span class="run__counts">{{.Summary}}</span>
{{if .Problem}}<span class="run__problem">{{.Problem}}</span>{{end}}
</li>
{{end}}
</ul>
{{end}}
{{end}}
`)
