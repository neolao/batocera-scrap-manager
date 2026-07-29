package webui

import (
	"errors"
	"net/http"
	"net/url"
	"slices"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// The two things a send needs to be fully specified: which of the configured
// ROMs folders it writes to, and which rule it follows once there. Both travel
// in the query of the confirmation page and in the submission that follows it,
// so a confirmation is always about one folder and one rule — never about an
// action still to be decided.
const (
	sendFolderParam = "folder"
	sendModeParam   = "mode"

	sendModeFill    = "fill"
	sendModeReplace = "replace"
)

// sendMode is one of the two rules a send may follow. Value is what the form
// submits, Label what the game page and the confirmation call it, and Note the
// one sentence saying what it does to values the folder already holds — the
// only difference that matters, and the one a user must not have to infer from
// the verb.
type sendMode struct {
	Value string
	Label string
	Note  string
}

// sendModes lists the rules in the order the game page offers them: filling the
// gaps comes first because it cannot lose anything, and is what a user reaching
// for this control usually means.
var sendModes = []sendMode{
	{
		Value: sendModeFill,
		Label: "Fill the gaps only",
		Note:  "A value the folder already holds is never overwritten.",
	},
	{
		Value: sendModeReplace,
		Label: "Replace with what the registry knows",
		Note:  "Every value and every medium the registry holds is overwritten in the folder.",
	},
}

// The outcome a send redirects to the game page with. Every one of them names
// the folder, which is why they are composed rather than looked up in the
// savedConfirmations table.
const (
	sentWritten   = "sent"
	sentNothing   = "sent-nothing"
	sentMediaLeft = "sent-media-left"
	sentMissing   = "sent-missing"
)

// sendControl is the game page's send section: where the choice is submitted,
// every folder it may name, and the rules it may follow. Folders is empty when
// none is configured, which the page states in words rather than rendering an
// empty list — a control offering nothing is worse than a sentence saying why.
type sendControl struct {
	Action  string
	Folders []string
	Modes   []sendMode
}

// sendConfirmation is the page asking whether to really send one game to one
// folder under one rule: what is written where, what that rule does to what is
// already there, and the two ways out.
type sendConfirmation struct {
	Name      string
	System    string
	SystemURL string
	GameURL   string
	Action    string
	Folder    string
	Mode      sendMode
	Replaces  bool
	// Problem states why the game was not sent, when the page is shown again
	// after a failed attempt rather than for the first time.
	Problem string
}

// sendControlOf describes how a game's page offers to send it.
func sendControlOf(system, id string, romsFolders []string) sendControl {
	return sendControl{
		Action:  gameURL(system, id) + "/send",
		Folders: romsFolders,
		Modes:   sendModes,
	}
}

// serveSendConfirmation renders the page confirming one send. Opening it never
// writes anything: the submission is what does. Both the folder and the rule
// come from the query the game page's control built, and both are checked here
// — a URL naming a folder the configuration does not hold is refused before it
// can name a place on the disk to write to.
func (ui *webUI) serveSendConfirmation(w http.ResponseWriter, r *http.Request) {
	system, id := r.PathValue("system"), r.PathValue("id")

	ui.mu.RLock()
	entry, found := ui.reg.FindByID(system, id)
	ui.mu.RUnlock()
	if !found {
		renderGameNotFound(w, system, id)
		return
	}

	query := r.URL.Query()
	folder, mode, ok := ui.requestedSend(query.Get(sendFolderParam), query.Get(sendModeParam))
	if !ok {
		renderBadSend(w)
		return
	}

	render(w, http.StatusOK, sendTemplate, sendForm(entry, system, id, folder, mode, ""))
}

// sendGame writes one game into one chosen ROMs folder, then redirects back to
// the game's page, which confirms what happened there.
//
// It is the one change of the web UI that writes nothing to the registry: the
// registry is the source it reads from. So there is no clone to persist, no
// snapshot to swap in and no site to regenerate — only the read lock long
// enough to capture what the run needs, exactly as the whole-folder completion
// does.
func (ui *webUI) sendGame(w http.ResponseWriter, r *http.Request) {
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

	system, id := r.PathValue("system"), r.PathValue("id")

	ui.mu.RLock()
	entry, found := ui.reg.FindByID(system, id)
	reg, registryFolder := ui.reg, ui.registryFolder
	ui.mu.RUnlock()
	if !found {
		renderGameNotFound(w, system, id)
		return
	}

	folder, mode, ok := ui.requestedSend(r.PostFormValue(sendFolderParam), r.PostFormValue(sendModeParam))
	if !ok {
		renderBadSend(w)
		return
	}

	send := registry.CompleteGame
	if mode.Value == sendModeReplace {
		send = registry.ReplaceGame
	}
	written, failed, err := send(reg, folder, registryFolder, system, entry.Game.Path, nil)

	outcome := sentNothing
	switch {
	case errors.Is(err, registry.ErrGameNotFound):
		outcome = sentMissing
	case err != nil:
		page := sendForm(entry, system, id, folder, mode,
			"This ROMs folder could not be written to, so the game was not sent: "+err.Error())
		render(w, http.StatusInternalServerError, sendTemplate, page)
		return
	case failed:
		outcome = sentMediaLeft
	case written:
		outcome = sentWritten
	}

	http.Redirect(w, r, sentURL(system, id, folder, mode.Value, outcome), http.StatusSeeOther)
}

// serveWrongSendMethod refuses a method the send URL does not support. It needs
// its own handler because serveWrongMethod is worded for the edit URL.
func (ui *webUI) serveWrongSendMethod(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
	renderProblem(w, http.StatusMethodNotAllowed, "Not allowed",
		r.Method+" is not something this page accepts.")
}

// requestedSend reads the folder and the rule a request named, reporting
// whether both are ones this server offers. A folder is accepted only when the
// configuration holds it word for word: this is the flow that writes outside the
// registry, and a path taken from a request is otherwise a path to anywhere.
func (ui *webUI) requestedSend(folder, mode string) (string, sendMode, bool) {
	if !slices.Contains(ui.romsFolders, folder) {
		return "", sendMode{}, false
	}
	for _, known := range sendModes {
		if known.Value == mode {
			return folder, known, true
		}
	}
	return "", sendMode{}, false
}

// renderBadSend refuses a send naming a folder that is not configured, or a
// rule this flow does not know. Both are refused the same way: the game page's
// control cannot produce either, so whatever did is not worth telling apart.
func renderBadSend(w http.ResponseWriter) {
	renderProblem(w, http.StatusBadRequest, "Bad request",
		"This request named a ROMs folder or a rule this server does not offer.")
}

// sendForm builds the confirmation's view, for the first attempt as for one
// shown again after a failure.
func sendForm(entry registry.Entry, system, id, folder string, mode sendMode, problem string) sendConfirmation {
	return sendConfirmation{
		Name:      entry.Game.Name,
		System:    system,
		SystemURL: systemURL(system),
		GameURL:   gameURL(system, id),
		Action:    gameURL(system, id) + "/send",
		Folder:    folder,
		Mode:      mode,
		Replaces:  mode.Value == sendModeReplace,
		Problem:   problem,
	}
}

// sentURL builds the game page URL confirming a send: the outcome, plus the
// folder and the rule it composes its sentence from.
func sentURL(system, id, folder, mode, outcome string) string {
	query := url.Values{
		savedParam:      {outcome},
		sendFolderParam: {folder},
		sendModeParam:   {mode},
	}
	return gameURL(system, id) + "?" + query.Encode() + savedAnchor
}

// sentConfirmation turns what a send redirected with into the sentence the game
// page confirms it with, or nothing at all when the query is not a send's. The
// folder is always named: the whole point of this flow is that the user picked
// one among several, and a confirmation that does not say which one confirms
// nothing.
func sentConfirmation(query url.Values) string {
	folder := query.Get(sendFolderParam)
	if folder == "" {
		return ""
	}
	replaced := query.Get(sendModeParam) == sendModeReplace

	switch query.Get(savedParam) {
	case sentWritten:
		return "Sent to " + folder + " — " + whatWasWritten(replaced)
	case sentMediaLeft:
		return "Sent to " + folder + " — " + whatWasWritten(replaced) +
			" Some of its media files could not be copied there."
	case sentNothing:
		return "Nothing to send to " + folder + " — it already holds everything the registry knows about this game."
	case sentMissing:
		return "Not sent: " + folder + " has no such game in its gamelist.xml. Sending a game fills in what a folder already lists; it never adds a game to a folder that does not hold its ROM."
	}
	return ""
}

// whatWasWritten says what the chosen rule actually did, since "sent" alone
// does not tell a folder that gained what it lacked from one whose own values
// were overwritten.
func whatWasWritten(replaced bool) string {
	if replaced {
		return "its gamelist.xml now holds what the registry knows."
	}
	return "the gaps in its gamelist.xml were filled from the registry."
}

// sendTemplate renders the confirmation of one send: the game, the folder, what
// the chosen rule does to what is already there, and the two ways out. A
// replacement is warned about in its own terms — it is the only thing the web
// UI does that destroys a value the user may have nowhere else.
var sendTemplate = newPage("send", `
{{define "title"}}Send {{.Name}} to a ROMs folder - Registry{{end}}
{{define "body"}}
<nav class="crumbs" aria-label="Breadcrumb">
<a href="/">Registry</a><span class="crumbs__sep">/</span><a href="{{.SystemURL}}">{{.System}}</a><span class="crumbs__sep">/</span><a href="{{.GameURL}}">{{.Name}}</a><span class="crumbs__sep">/</span><span class="crumbs__current">Send</span>
</nav>
<main>
<article class="game">
<h2 class="game__title">Send {{.Name}} to this ROMs folder?</h2>
{{if .Problem}}
<div class="errors" role="alert" tabindex="-1" id="errors">
<p class="errors__title">Not sent</p>
<p>{{.Problem}}</p>
</div>
{{end}}
<div class="confirm">
<p class="confirm__lead">This writes what the registry knows about {{.Name}} into the <code>{{.System}}</code> folder of:</p>
<ul class="confirm__files">
<li><code>{{.Folder}}</code></li>
</ul>
<p class="confirm__lead">{{.Mode.Label}} &mdash; {{.Mode.Note}}</p>
{{if .Replaces}}
<p class="confirm__note">Only this game is touched, in this folder alone. A value the registry does not know is left as the folder holds it, and no file is removed.</p>
<p class="confirm__warning">The overwritten values are your own Batocera files: this cannot be undone.</p>
{{else}}
<p class="confirm__note">Only this game is touched, in this folder alone. The registry folder itself is left unchanged &mdash; its game files and its consultation site are not written to.</p>
<p class="confirm__warning">The rewritten file is your own Batocera file: this cannot be undone.</p>
{{end}}
<div class="form__actions">
<form method="post" action="{{.Action}}">
<input type="hidden" name="`+sendFolderParam+`" value="{{.Folder}}">
<input type="hidden" name="`+sendModeParam+`" value="{{.Mode.Value}}">
<button class="button{{if .Replaces}} button--danger{{end}}" type="submit">Send this game</button>
</form>
<a class="button button--quiet" href="{{.GameURL}}">Cancel</a>
</div>
</div>
</article>
</main>
{{end}}
`)
