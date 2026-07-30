package webui

import (
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// importURL and completeURL are where each of the two long operations lives —
// one URL apiece, holding every state that operation can be in, so "run it
// again" is the very control that started it (see decisions/025).
const (
	importURL   = "/import"
	completeURL = "/complete"
)

// refreshSeconds is how long a page following a run waits before reloading
// itself. The UI ships no JavaScript, so a meta refresh is the only way a run
// can ever show its own end.
const refreshSeconds = "5"

// quietParam turns that reloading off, the page staying otherwise identical.
// A page that reloads itself with no way out resets whoever is reading it
// every few seconds, which makes the folder list unreadable to a screen reader
// and to anyone slow to scan it.
const quietParam = "quiet"

// jobKind names one of the two long operations the server detaches from its
// request. They flow in opposite directions over the same files, which is why
// there is a name for them at all: a page has to be able to say which one is
// holding the slot (see decisions/027).
type jobKind string

const (
	jobImport     jobKind = "import"
	jobCompletion jobKind = "completion"
)

// jobDescription is everything about a job that its own page and the other
// one's need to know: where it lives, how a bystander page names it, and how
// its three counts are worded. The wording of the counts belongs to the domain
// package owning the values, never to a literal here.
type jobDescription struct {
	URL     string
	Running string
	Format  string
}

var jobDescriptions = map[jobKind]jobDescription{
	jobImport: {
		URL:     importURL,
		Running: "An import of the ROMs folders is in progress.",
		Format:  registry.ImportSummaryFormat,
	},
	jobCompletion: {
		URL:     completeURL,
		Running: "A completion of the ROMs folders is in progress.",
		Format:  registry.CompletionSummaryFormat,
	},
}

// jobs is the one background operation the server runs at a time, whichever
// direction it goes, plus what the last run of each kind had to say. It holds
// its own mutex rather than sharing the registry's — a run lasts minutes, and
// keeping the snapshot lock that long would queue every served page behind the
// first correction submitted meanwhile (see decisions/025).
type jobs struct {
	mu     sync.Mutex
	active jobKind
	runs   map[jobKind]*jobRun
}

// jobRun is what one run has to say for itself: where it is while it goes, and
// what it did once it is over.
type jobRun struct {
	startedAt time.Time
	endedAt   time.Time
	running   bool
	folder    string
	system    string
	game      string
	folders   []folderReport
	// problem is a failure of the run itself, beyond any one folder — a
	// registry that could not be written. caveat is the opposite: a reservation
	// on a run that did apply, which must never read as a failure.
	problem string
	caveat  string
}

// folderReport is one ROMs folder's outcome. Problem is empty when the folder
// was gone through to the end: a folder that stopped the run keeps the counts
// it reached before failing, so the report says what was really done rather
// than dropping it. Counts are held positionally because the two jobs count
// different things under the same shape — each one's wording comes from its
// own jobDescription.
type folderReport struct {
	Folder  string
	Counts  [3]int
	Problem string
}

// jobHandle is the running side of a job: the one thing able to write into a
// run, handed out by start and given up by finish. A run that does not hold
// one cannot exist, which is what keeps the slot and the report in step.
type jobHandle struct {
	jobs *jobs
	run  *jobRun
}

// start takes the slot for a new run of kind, returning nil when it is already
// taken — by that same kind or by the other one. Two runs at once on the same
// gamelist.xml is what this guards against, and the two directions are as
// exclusive with each other as each is with itself (decisions/027).
func (j *jobs) start(kind jobKind) *jobHandle {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.active != "" {
		return nil
	}
	if j.runs == nil {
		j.runs = map[jobKind]*jobRun{}
	}

	run := &jobRun{startedAt: time.Now(), running: true}
	j.active, j.runs[kind] = kind, run
	return &jobHandle{jobs: j, run: run}
}

// beginFolder records which ROMs folder the run moved on to. It is what proves
// the run is advancing when nothing else does: a job only reports a game when
// it actually does something to it, so a folder already in step would leave
// the game line frozen for minutes.
func (h *jobHandle) beginFolder(folder string) {
	h.jobs.mu.Lock()
	defer h.jobs.mu.Unlock()

	h.run.folder = folder
	h.run.system, h.run.game = "", ""
}

// progress records the game the run is on right now. Its signature is what the
// domain packages take as their progress callback, so it can be passed as is.
func (h *jobHandle) progress(event registry.ProgressEvent) {
	h.jobs.mu.Lock()
	defer h.jobs.mu.Unlock()

	h.run.system, h.run.game = event.System, event.GameName
}

// recordFolder adds one folder's outcome to the report being built.
func (h *jobHandle) recordFolder(report folderReport) {
	h.jobs.mu.Lock()
	defer h.jobs.mu.Unlock()

	h.run.folders = append(h.run.folders, report)
}

// fail records a failure of the run as a whole, told apart from a folder's own
// problem because it is not about any one folder.
func (h *jobHandle) fail(problem string) {
	h.jobs.mu.Lock()
	defer h.jobs.mu.Unlock()

	h.run.problem = problem
}

// note records a reservation on a run that otherwise applied. It is not a
// failure and is never counted as one: telling the user a change failed when
// it did not would have them redo what already happened.
func (h *jobHandle) note(caveat string) {
	h.jobs.mu.Lock()
	defer h.jobs.mu.Unlock()

	h.run.caveat = caveat
}

// recoverJob turns a panic during a run's own work into a failed run instead
// of letting it escape the goroutine started for it — which would otherwise
// crash the whole process and take down every other request being served
// with it. Deferred ahead of finish (defers run last-in-first-out), so the
// failure it records is already in place by the time the slot is freed.
func recoverJob(h *jobHandle) {
	if p := recover(); p != nil {
		h.fail(fmt.Sprintf("This run stopped on an internal error: %v", p))
	}
}

// finish frees the slot and closes the report, which stays readable until the
// next run of the same kind replaces it — a run writes where nothing can undo
// it, so its account must outlive the browser tab that started it.
func (h *jobHandle) finish() {
	h.jobs.mu.Lock()
	defer h.jobs.mu.Unlock()

	h.run.running = false
	h.run.endedAt = time.Now()
	h.run.folder, h.run.system, h.run.game = "", "", ""
	h.jobs.active = ""
}

// jobView is what one job's page sees: its own last (or current) run, and the
// other kind when that one is holding the slot. Both come from a single lock,
// or a run ending between two reads would leave the page contradicting itself.
type jobView struct {
	Report *jobReport
	Other  jobKind
}

func (j *jobs) view(kind jobKind) jobView {
	j.mu.Lock()
	defer j.mu.Unlock()

	other := j.active
	if other == kind {
		other = ""
	}
	return jobView{Report: reportOf(j.runs[kind], jobDescriptions[kind].Format), Other: other}
}

// jobReport is a run as a page shows it — a copy taken under the lock, so a
// template never reads a field the run is writing, and everything a template
// would otherwise have to compute already worded.
type jobReport struct {
	Running   bool
	StartedAt string
	Elapsed   string
	Current   string
	Folders   []folderLine
	// Totals carries the counts a page needs to word its own verdict; Summary
	// is those same counts already worded.
	Totals   [3]int
	Summary  string
	Totalled bool
	Failed   bool
	Problem  string
	Caveat   string
}

// folderLine is one folder's outcome, its counts already worded.
type folderLine struct {
	Folder  string
	Summary string
	Problem string
}

// reportOf turns a run into what a page renders, or nil when no run of that
// kind ever happened.
func reportOf(run *jobRun, format string) *jobReport {
	if run == nil {
		return nil
	}

	var totals [3]int
	failed := run.problem != ""
	lines := make([]folderLine, len(run.folders))
	for i, folder := range run.folders {
		for count := range totals {
			totals[count] += folder.Counts[count]
		}
		lines[i] = folderLine{
			Folder:  folder.Folder,
			Summary: summarize(format, folder.Counts),
			Problem: folder.Problem,
		}
		failed = failed || folder.Problem != ""
	}

	until := run.endedAt
	if run.running {
		until = time.Now()
	}
	return &jobReport{
		Running:   run.running,
		StartedAt: run.startedAt.Format("15:04:05"),
		Elapsed:   until.Sub(run.startedAt).Round(time.Second).String(),
		Current:   currentStep(run),
		Folders:   lines,
		Totals:    totals,
		Summary:   summarize(format, totals),
		// With a single folder the totals are the same sentence twice, which
		// reads as a stutter rather than as a total.
		Totalled: len(lines) > 1,
		Failed:   failed,
		Problem:  run.problem,
		Caveat:   run.caveat,
	}
}

// summarize words three counts the way the job they belong to words them.
func summarize(format string, counts [3]int) string {
	return fmt.Sprintf(format, counts[0], counts[1], counts[2])
}

// currentStep words where a run has got to, down to the game it is on when one
// is known.
func currentStep(run *jobRun) string {
	if run.folder == "" {
		return ""
	}
	if run.game == "" {
		return run.folder
	}
	return run.folder + " — " + run.system + ": " + run.game
}

// jobPage is one job's page, in whichever of its states it happens to be.
// Folders is what the job goes through, named in full. Report is nil until a
// first run has something to say, and Other is set only while the opposite
// job holds the slot.
type jobPage struct {
	Folders  []string
	Action   string
	Report   *jobReport
	Other    *otherJob
	Refresh  bool
	QuietURL string
	LoudURL  string
}

// otherJob is the run holding the slot, as this page names it.
type otherJob struct {
	Sentence string
	URL      string
}

// Running reports whether this job's own run is going, which is what decides
// between the two things its page can offer: watching, or starting one.
func (p jobPage) Running() bool {
	return p.Report != nil && p.Report.Running
}

// jobPage builds the page of one job from the request asking for it.
func (ui *webUI) jobPage(kind jobKind, r *http.Request) jobPage {
	view := ui.jobs.view(kind)
	description := jobDescriptions[kind]

	page := jobPage{
		Folders:  ui.romsFolders,
		Action:   description.URL,
		Report:   view.Report,
		LoudURL:  description.URL,
		QuietURL: description.URL + "?" + quietParam,
	}
	if view.Other != "" {
		page.Other = &otherJob{
			Sentence: jobDescriptions[view.Other].Running,
			URL:      jobDescriptions[view.Other].URL,
		}
	}
	page.Refresh = page.Running() && !r.URL.Query().Has(quietParam)
	return page
}

// startJob is what a job's POST handler does before starting anything: refuse
// what is not this site's own, refuse a body too big to be a submission of
// ours, and take the slot. It reports the handle to run with, or nil when
// there is nothing to start — a run already going, or nowhere to run it. A
// nil handle is not a failure to report: the page redirected to already words
// every one of those states.
func (ui *webUI) startJob(kind jobKind, w http.ResponseWriter, r *http.Request) *jobHandle {
	if crossSite(r) {
		renderProblem(w, http.StatusForbidden, "Refused",
			"This request did not come from the registry's own pages.")
		return nil
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		renderProblem(w, http.StatusBadRequest, "Bad request", "This request could not be read.")
		return nil
	}

	var run *jobHandle
	// The slot is taken here rather than inside the goroutine: a second
	// submission arriving before the first one was scheduled would otherwise
	// find it free and set two runs on the same files. Nothing configured
	// starts nothing — the page says why.
	if len(ui.romsFolders) > 0 {
		run = ui.jobs.start(kind)
	}
	http.Redirect(w, r, jobDescriptions[kind].URL, http.StatusSeeOther)
	return run
}

// jobPartials are the parts of a job's page that are the same for both jobs,
// word for word: the auto-reload, the list of folders with their counts, the
// run in progress, and the page a run of the other kind leaves. Everything
// that differs — every verdict, every confirmation — stays in each job's own
// template rather than being parameterized here.
const jobPartials = `
{{define "head"}}{{if .Refresh}}<meta http-equiv="refresh" content="` + refreshSeconds + `">{{end}}{{end}}

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

{{define "progress"}}
<div class="run">
<p class="run__lead" role="status">Started at {{.Report.StartedAt}}, running for {{.Report.Elapsed}}.</p>
{{if .Report.Current}}<p class="run__current"><code>{{.Report.Current}}</code></p>{{end}}
{{template "folders" .Report}}
<p class="run__note">{{if .Refresh}}This page reloads by itself every ` + refreshSeconds + ` seconds until the run is over.{{else}}This page does not reload by itself.{{end}} Leaving it does not stop anything &mdash; come back to this address to read the report.</p>
<div class="form__actions">
<a class="button button--quiet" href="{{.LoudURL}}">Reload now</a>
{{if .Refresh}}<a class="button button--quiet" href="{{.QuietURL}}">Stop reloading</a>{{end}}
</div>
</div>
{{end}}

{{define "elsewhere"}}
<div class="run">
<p class="run__lead" role="status">{{.Other.Sentence}}</p>
<p class="run__note">Only one of the two runs at a time: the registry and the ROMs folders would otherwise each be read while the other is being written.</p>
<div class="form__actions">
<a class="button" href="{{.Other.URL}}">Follow it</a>
<a class="button button--quiet" href="/">Back to the registry</a>
</div>
</div>
{{end}}
`

// newJobPage parses one job's page on top of the layout and the parts the two
// jobs share.
func newJobPage(name, body string) *template.Template {
	return newPage(name, jobPartials+body)
}
