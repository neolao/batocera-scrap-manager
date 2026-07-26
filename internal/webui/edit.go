package webui

import (
	"net/http"
	"slices"
	"strconv"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/site"
)

// handBackParam is the form field naming the fields the user hands back to
// the scraper, letting later updates overwrite them again.
const handBackParam = "hand_back"

// fieldKind tells the template which control renders a field.
type fieldKind string

const (
	kindText     fieldKind = "text"
	kindTextarea fieldKind = "textarea"
	kindStars    fieldKind = "stars"
	kindYear     fieldKind = "year"
)

// yearRange bounds the year control. Video games do not predate it, and a
// browser refusing an absurd year spares a round-trip — the server checks the
// same bounds anyway, since nothing stops a client from ignoring them.
const (
	minYear = 1950
	maxYear = 2100
)

// editableField describes one control of the edit form: which value it
// carries, how it is rendered, and which registry field it corrects. Manual
// is that registry field's name — the form's own key is the friendlier one
// the user sees (a "year" control corrects a "release_date").
type editableField struct {
	Key      string
	Manual   string
	Label    string
	Kind     fieldKind
	Hint     string
	Required bool
	value    func(editValues) string
}

// editableFields is the form, in the order the read-only page lists the same
// values, so the eye lands on the field it came to fix.
var editableFields = []editableField{
	{Key: "name", Manual: "name", Label: "Name", Kind: kindText, Required: true,
		value: func(v editValues) string { return v.Name }},
	{Key: "desc", Manual: "desc", Label: "Description", Kind: kindTextarea,
		value: func(v editValues) string { return v.Desc }},
	{Key: "rating", Manual: "rating", Label: "Rating", Kind: kindStars,
		value: func(v editValues) string { return v.Rating }},
	{Key: "year", Manual: "release_date", Label: "Year", Kind: kindYear, Hint: "e.g. 1991",
		value: func(v editValues) string { return v.Year }},
	{Key: "developer", Manual: "developer", Label: "Developer", Kind: kindText,
		value: func(v editValues) string { return v.Developer }},
	{Key: "publisher", Manual: "publisher", Label: "Publisher", Kind: kindText,
		value: func(v editValues) string { return v.Publisher }},
	{Key: "genre", Manual: "genre", Label: "Genre", Kind: kindText,
		value: func(v editValues) string { return v.Genre }},
	{Key: "players", Manual: "players", Label: "Players", Kind: kindText, Hint: "e.g. 1-2",
		value: func(v editValues) string { return v.Players }},
}

// editValues holds the eight editable values as the form shows them: a star
// count out of five and a year, not the decimal rating and full timestamp the
// registry stores (see decisions/019).
type editValues struct {
	Name      string
	Desc      string
	Rating    string
	Year      string
	Developer string
	Publisher string
	Genre     string
	Players   string
}

// displayedValues reads a stored game as the form shows it.
func displayedValues(g gamelist.Game) editValues {
	rating := ""
	if filled, ok := site.FilledStars(g.Rating); ok {
		rating = starCount(filled)
	}
	return editValues{
		Name:      g.Name,
		Desc:      g.Desc,
		Rating:    rating,
		Year:      site.FormatYear(g.ReleaseDate),
		Developer: g.Developer,
		Publisher: g.Publisher,
		Genre:     g.Genre,
		Players:   g.Players,
	}
}

// editPage is the edit form's view: the game it corrects, one control per
// editable field, and the errors a refused submission must show.
type editPage struct {
	Name      string
	System    string
	SystemURL string
	GameURL   string
	Action    string
	Fields    []formControl
	Errors    []formError
	// Problem states what went wrong when the correction itself was fine but
	// could not be stored — a field-level message would blame the wrong thing.
	Problem string
}

// formControl is one rendered control of the form.
type formControl struct {
	Key        string
	Manual     string
	Label      string
	Kind       fieldKind
	Hint       string
	Required   bool
	Value      string
	Error      string
	HandEdited bool
	Stars      []starOption
	MinYear    int
	MaxYear    int
}

// starOption is one choice of the rating control: "Not rated" plus the six
// star counts, which stay distinct — a game rated zero out of five is not a
// game nobody rated.
type starOption struct {
	Value    string
	Label    string
	Selected bool
}

// formError is one entry of the error summary, linking to the control it
// concerns.
type formError struct {
	Key     string
	Label   string
	Message string
}

// serveEditForm renders the pre-filled form correcting one game's metadata,
// or the not-found page when the URL names a game the registry does not know.
func (ui *webUI) serveEditForm(w http.ResponseWriter, r *http.Request) {
	system, id := r.PathValue("system"), r.PathValue("id")

	ui.mu.RLock()
	entry, found := ui.reg.FindByID(system, id)
	ui.mu.RUnlock()
	if !found {
		renderGameNotFound(w, system, id)
		return
	}

	render(w, http.StatusOK, editTemplate, editForm(entry, displayedValues(entry.Game), nil))
}

// serveWrongMethod refuses a method the game's edit URL does not support,
// naming the ones it does.
func (ui *webUI) serveWrongMethod(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
	renderProblem(w, http.StatusMethodNotAllowed, "Not allowed",
		r.Method+" is not something this page accepts.")
}

// editForm builds the form's view for entry, filled with values — the stored
// ones when the form is first opened, the submitted ones when a refused
// submission is shown again — and the messages of the fields errs refused.
func editForm(entry registry.Entry, values editValues, errs map[string]string) editPage {
	id := registry.GameID(entry.Game.Path)
	page := editPage{
		Name:      entry.Game.Name,
		System:    entry.System,
		SystemURL: "/#" + entry.System,
		GameURL:   gameURL(entry.System, id),
		Action:    gameURL(entry.System, id) + "/edit",
	}

	for _, field := range editableFields {
		control := formControl{
			Key:        field.Key,
			Manual:     field.Manual,
			Label:      field.Label,
			Kind:       field.Kind,
			Hint:       field.Hint,
			Required:   field.Required,
			Value:      field.value(values),
			Error:      errs[field.Key],
			HandEdited: slices.Contains(entry.ManualFields, field.Manual),
			MinYear:    minYear,
			MaxYear:    maxYear,
		}
		if field.Kind == kindStars {
			control.Stars = starOptions(control.Value)
		}
		page.Fields = append(page.Fields, control)

		if control.Error != "" {
			page.Errors = append(page.Errors, formError{Key: field.Key, Label: field.Label, Message: control.Error})
		}
	}
	return page
}

// starOptions builds the rating control's choices, marking the one matching
// selected.
func starOptions(selected string) []starOption {
	options := []starOption{{Value: "", Label: "Not rated", Selected: selected == ""}}
	for filled := 0; filled <= 5; filled++ {
		value := starCount(filled)
		options = append(options, starOption{
			Value:    value,
			Label:    value + "/5",
			Selected: value == selected,
		})
	}
	return options
}

// starCount renders a star count as the value the form carries it under.
func starCount(filled int) string {
	return strconv.Itoa(filled)
}

// editTemplate renders the form correcting one game's metadata: one labelled
// control per editable field, the marks telling which values were corrected
// by hand along with the checkbox handing them back to the scraper, and the
// summary of a refused submission.
var editTemplate = newPage("edit", `
{{define "title"}}Edit {{.Name}} - Registry{{end}}
{{define "body"}}
<nav class="crumbs" aria-label="Breadcrumb">
<a href="/">Registry</a><span class="crumbs__sep">/</span><a href="{{.SystemURL}}">{{.System}}</a><span class="crumbs__sep">/</span><a href="{{.GameURL}}">{{.Name}}</a><span class="crumbs__sep">/</span><span class="crumbs__current">Edit</span>
</nav>
<main>
<article class="game">
<h2 class="game__title">Edit {{.Name}}</h2>
{{if .Problem}}
<div class="errors" role="alert" tabindex="-1" id="errors">
<p class="errors__title">Not saved</p>
<p>{{.Problem}}</p>
</div>
{{end}}
{{if .Errors}}
<div class="errors" role="alert" tabindex="-1" id="errors">
<p class="errors__title">This was not saved:</p>
<ul>
{{range .Errors}}<li><a href="#field-{{.Key}}">{{.Label}}: {{.Message}}</a></li>
{{end}}
</ul>
</div>
{{end}}
<form class="form" method="post" action="{{.Action}}">
<fieldset class="form__fields">
<legend>Metadata</legend>
{{range .Fields}}
<div class="field{{if .Error}} field--invalid{{end}}">
<label class="field__label" for="field-{{.Key}}">{{.Label}}{{if .Required}} <span class="field__required">(required)</span>{{end}}</label>
{{if .Error}}<p class="field__error" id="error-{{.Key}}">{{.Error}}</p>{{end}}
{{if .Hint}}<p class="field__hint" id="hint-{{.Key}}">{{.Hint}}</p>{{end}}
{{if eq .Kind "textarea"}}
<textarea class="field__control" id="field-{{.Key}}" name="{{.Key}}" rows="10"{{if .Error}} aria-invalid="true" aria-describedby="error-{{.Key}}"{{end}}>{{.Value}}</textarea>
{{else if eq .Kind "stars"}}
<select class="field__control" id="field-{{.Key}}" name="{{.Key}}"{{if .Error}} aria-invalid="true" aria-describedby="error-{{.Key}}"{{end}}>
{{range .Stars}}<option value="{{.Value}}"{{if .Selected}} selected{{end}}>{{.Label}}</option>
{{end}}
</select>
{{else if eq .Kind "year"}}
<input class="field__control" type="number" id="field-{{.Key}}" name="{{.Key}}" value="{{.Value}}" min="{{.MinYear}}" max="{{.MaxYear}}" step="1" inputmode="numeric"{{if .Hint}} aria-describedby="hint-{{.Key}}"{{end}}{{if .Error}} aria-invalid="true"{{end}}>
{{else}}
<input class="field__control" type="text" id="field-{{.Key}}" name="{{.Key}}" value="{{.Value}}"{{if .Required}} required{{end}}{{if .Hint}} aria-describedby="hint-{{.Key}}"{{end}}{{if .Error}} aria-invalid="true"{{end}}>
{{end}}
{{if .HandEdited}}
<p class="field__manual">
<input type="checkbox" id="hand-back-{{.Key}}" name="`+handBackParam+`" value="{{.Manual}}">
<label for="hand-back-{{.Key}}">Corrected by hand &mdash; let updates overwrite it again</label>
</p>
{{end}}
</div>
{{end}}
</fieldset>
<div class="form__actions">
<button class="button" type="submit">Save changes</button>
<a class="button button--quiet" href="{{.GameURL}}">Cancel</a>
</div>
</form>
</article>
</main>
{{end}}
`)
