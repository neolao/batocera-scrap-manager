package registry

import (
	"slices"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
)

// editableFields lists the metadata fields a user may correct by hand, each
// paired with the name identifying it in an entry's ManualFields and in the
// game's JSON file. Media references and the ROM path are deliberately absent:
// the path identifies the entry itself, and media are managed by their own
// flows rather than typed in.
var editableFields = []struct {
	Name string
	get  func(*gamelist.Game) *string
}{
	{"name", func(g *gamelist.Game) *string { return &g.Name }},
	{"desc", func(g *gamelist.Game) *string { return &g.Desc }},
	{"rating", func(g *gamelist.Game) *string { return &g.Rating }},
	{"release_date", func(g *gamelist.Game) *string { return &g.ReleaseDate }},
	{"developer", func(g *gamelist.Game) *string { return &g.Developer }},
	{"publisher", func(g *gamelist.Game) *string { return &g.Publisher }},
	{"genre", func(g *gamelist.Game) *string { return &g.Genre }},
	{"players", func(g *gamelist.Game) *string { return &g.Players }},
}

// editableField returns the accessor of the editable field called name,
// reporting whether such a field exists. A name that does not match any (a
// mark written by a later version of the tool, or one naming something that
// is not editable metadata at all) is simply not editable.
func editableField(name string) (func(*gamelist.Game) *string, bool) {
	for _, field := range editableFields {
		if field.Name == name {
			return field.get, true
		}
	}
	return nil, false
}

// keepHandEditedFields restores, into incoming, the value each of manual's
// fields holds in stored — so importing a ROMs folder that still carries the
// badly scraped value cannot undo a correction made by hand (see
// decisions/017).
func keepHandEditedFields(incoming *gamelist.Game, stored gamelist.Game, manual []string) {
	for _, name := range manual {
		get, ok := editableField(name)
		if !ok {
			continue
		}
		*get(incoming) = *get(&stored)
	}
}

// Metadata carries the metadata of a game as a user corrects it: every
// editable field at once, an empty one meaning the value is cleared. The ROM
// path and the media references are deliberately absent — they identify the
// entry and are managed elsewhere, so no correction can reach them.
type Metadata struct {
	Name        string
	Desc        string
	Rating      string
	ReleaseDate string
	Developer   string
	Publisher   string
	Genre       string
	Players     string
}

// set applies m to g, field by field.
func (m Metadata) set(g *gamelist.Game) {
	g.Name = m.Name
	g.Desc = m.Desc
	g.Rating = m.Rating
	g.ReleaseDate = m.ReleaseDate
	g.Developer = m.Developer
	g.Publisher = m.Publisher
	g.Genre = m.Genre
	g.Players = m.Players
}

// UpdateMetadata replaces the editable metadata of the entry of system whose
// game ID is id, and marks as hand-edited every field the correction actually
// changes, so later imports stop overwriting it. A field named in handBack is
// unmarked instead — its submitted value is still stored, but the next import
// is free to refresh it again. It returns ErrGameNotFound, without modifying
// anything, if no entry matches.
func UpdateMetadata(reg *Registry, system, id string, m Metadata, handBack []string) error {
	i := reg.indexOfID(system, id)
	if i == -1 {
		return ErrGameNotFound
	}

	before := reg.Entries[i].Game
	m.set(&reg.Entries[i].Game)
	reg.Entries[i].ManualFields = markedFields(reg.Entries[i], before, handBack)
	return nil
}

// Protect marks every editable field of the entry of system whose game ID is
// id, so no import refreshes any of them — the whole-game statement "these
// values are the right ones", as opposed to UpdateMetadata's per-field marking
// of what a correction changed. No value is written: protecting states that the
// stored ones are correct. It returns ErrGameNotFound, without modifying
// anything, if no entry matches.
func Protect(reg *Registry, system, id string) error {
	i := reg.indexOfID(system, id)
	if i == -1 {
		return ErrGameNotFound
	}

	reg.Entries[i].ManualFields = editableFieldNames()
	return nil
}

// Unprotect hands every field of that same entry back to the scraper,
// including the marks left by earlier per-field corrections — lifting is a
// whole-game statement too (see decisions/021). Values are left as they are:
// the next import is simply free to refresh them again. It returns
// ErrGameNotFound, without modifying anything, if no entry matches.
func Unprotect(reg *Registry, system, id string) error {
	i := reg.indexOfID(system, id)
	if i == -1 {
		return ErrGameNotFound
	}

	reg.Entries[i].ManualFields = nil
	return nil
}

// FullyProtected reports whether every editable field of the entry is marked,
// which is what Protect leaves behind. It is derived from the same table the
// marking uses rather than stored, so no caller can hold its own idea of what
// a protected game is. A mark naming something uneditable is neither required
// nor in the way.
func (e Entry) FullyProtected() bool {
	for _, field := range editableFields {
		if !slices.Contains(e.ManualFields, field.Name) {
			return false
		}
	}
	return true
}

// editableFieldNames lists the name every editable field is marked under. The
// slice is built fresh on each call, so an entry cloned from another one never
// writes through to it.
func editableFieldNames() []string {
	names := make([]string, len(editableFields))
	for i, field := range editableFields {
		names[i] = field.Name
	}
	return names
}

// markedFields computes the hand-edited fields of entry now that its game
// changed from before: the fields already marked, plus those this correction
// changed, minus those handed back to the scraper. The result is a fresh
// slice, so an entry cloned from another one never writes through to it.
func markedFields(entry Entry, before gamelist.Game, handBack []string) []string {
	var marked []string
	for _, field := range editableFields {
		if slices.Contains(handBack, field.Name) {
			continue
		}
		changed := *field.get(&before) != *field.get(&entry.Game)
		if changed || slices.Contains(entry.ManualFields, field.Name) {
			marked = append(marked, field.Name)
		}
	}
	return marked
}
