package registry

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
)

func TestSave_EntryWithHandEditedFields_ReloadsWithThemStillMarked(t *testing.T) {
	path := t.TempDir()
	reg := &Registry{Entries: []Entry{{
		System:       "megadrive",
		Game:         gamelist.Game{Path: "./Sonic.zip", Name: "Sonic the Hedgehog", Genre: "Platform"},
		ManualFields: []string{"name", "genre"},
	}}}

	if err := Save(path, reg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("Entries = %v, want exactly 1", got.Entries)
	}
	if strings.Join(got.Entries[0].ManualFields, ",") != "name,genre" {
		t.Errorf("ManualFields = %v, want [name genre]", got.Entries[0].ManualFields)
	}
	if got.Entries[0].Game.Name != "Sonic the Hedgehog" {
		t.Errorf("Game.Name = %q, want the game's own fields to stay at the root of its file", got.Entries[0].Game.Name)
	}
}

func TestLoad_GameFileWrittenBeforeHandEditing_LoadsWithNothingMarked(t *testing.T) {
	// A registry written by an earlier version has no manual_fields key: it
	// must keep loading, with no field protected.
	path := t.TempDir()
	megadrive := filepath.Join(path, "megadrive")
	if err := os.MkdirAll(megadrive, 0o755); err != nil {
		t.Fatalf("mkdir megadrive: %v", err)
	}
	legacy := `{"path":"./Sonic.zip","name":"Sonic","desc":"Fast."}`
	if err := os.WriteFile(filepath.Join(megadrive, "Sonic.json"), []byte(legacy), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	reg, err := Load(path)

	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if len(reg.Entries) != 1 {
		t.Fatalf("Entries = %v, want exactly 1", reg.Entries)
	}
	if reg.Entries[0].Game.Name != "Sonic" || reg.Entries[0].Game.Desc != "Fast." {
		t.Errorf("Game = %+v, want the legacy fields to be read as before", reg.Entries[0].Game)
	}
	if len(reg.Entries[0].ManualFields) != 0 {
		t.Errorf("ManualFields = %v, want none", reg.Entries[0].ManualFields)
	}
}

func TestSave_EntryWithNoHandEditedField_LeavesTheKeyOutOfTheGameFile(t *testing.T) {
	path := t.TempDir()
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic"}}}}

	if err := Save(path, reg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	data, err := os.ReadFile(filepath.Join(path, "megadrive", "Sonic.json"))
	if err != nil {
		t.Fatalf("failed to read the written game file: %v", err)
	}
	if strings.Contains(string(data), "manual_fields") {
		t.Errorf("game file = %s, want no manual_fields key for a game nobody edited", data)
	}
}

func TestImport_ChangedGameWithAHandEditedField_KeepsItAndRefreshesTheRest(t *testing.T) {
	reg := &Registry{Entries: []Entry{{
		System:       "megadrive",
		Game:         gamelist.Game{Path: "./Sonic.zip", Name: "Sonic the Hedgehog", Genre: "Platform"},
		ManualFields: []string{"name"},
	}}}

	added, updated, unchanged := reg.Import("megadrive", []gamelist.Game{
		{Path: "./Sonic.zip", Name: "Sonic the Hedghog", Genre: "Platformer", Desc: "Freshly scraped."},
	})

	if added != 0 || updated != 1 || unchanged != 0 {
		t.Errorf("added=%d updated=%d unchanged=%d, want 0,1,0", added, updated, unchanged)
	}
	got := reg.Entries[0].Game
	if got.Name != "Sonic the Hedgehog" {
		t.Errorf("Name = %q, want the hand-edited name to survive the import", got.Name)
	}
	if got.Genre != "Platformer" {
		t.Errorf("Genre = %q, want the unprotected field to be refreshed from the ROMs folder", got.Genre)
	}
	if got.Desc != "Freshly scraped." {
		t.Errorf("Desc = %q, want the newly scraped description to be imported", got.Desc)
	}
	if strings.Join(reg.Entries[0].ManualFields, ",") != "name" {
		t.Errorf("ManualFields = %v, want the mark itself to survive the import", reg.Entries[0].ManualFields)
	}
}

func TestImport_GameDifferingOnlyByAHandEditedField_CountsAsUnchanged(t *testing.T) {
	// The protected field is put back, so nothing is left to update: reporting
	// the game as "updated" on every single run would be pure noise.
	reg := &Registry{Entries: []Entry{{
		System:       "megadrive",
		Game:         gamelist.Game{Path: "./Sonic.zip", Name: "Sonic the Hedgehog"},
		ManualFields: []string{"name"},
	}}}

	added, updated, unchanged := reg.Import("megadrive", []gamelist.Game{
		{Path: "./Sonic.zip", Name: "Sonic the Hedghog"},
	})

	if added != 0 || updated != 0 || unchanged != 1 {
		t.Errorf("added=%d updated=%d unchanged=%d, want 0,0,1", added, updated, unchanged)
	}
	if reg.Entries[0].Game.Name != "Sonic the Hedgehog" {
		t.Errorf("Name = %q, want the hand-edited name untouched", reg.Entries[0].Game.Name)
	}
}

func TestImport_MarkNamingSomethingUneditable_IsIgnored(t *testing.T) {
	// A mark can only ever protect an editable metadata field: a ROM path or a
	// name no version of the tool knows about must not freeze anything, since
	// the path is what identifies the entry in the first place.
	reg := &Registry{Entries: []Entry{{
		System:       "megadrive",
		Game:         gamelist.Game{Path: "./Sonic.zip", Name: "Sonic", Image: "images/old.png"},
		ManualFields: []string{"path", "image", "bogus"},
	}}}

	added, updated, unchanged := reg.Import("megadrive", []gamelist.Game{
		{Path: "./Sonic.zip", Name: "Sonic", Image: "images/new.png"},
	})

	if added != 0 || updated != 1 || unchanged != 0 {
		t.Errorf("added=%d updated=%d unchanged=%d, want 0,1,0", added, updated, unchanged)
	}
	if reg.Entries[0].Game.Image != "images/new.png" {
		t.Errorf("Image = %q, want the import to refresh a field that is not editable metadata", reg.Entries[0].Game.Image)
	}
}

func TestImport_HandEditedFieldOnAGameOfAnotherSystem_DoesNotProtectThisOne(t *testing.T) {
	reg := &Registry{Entries: []Entry{
		{
			System:       "mastersystem",
			Game:         gamelist.Game{Path: "./Sonic.zip", Name: "Sonic (hand-fixed)"},
			ManualFields: []string{"name"},
		},
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic"}},
	}}

	reg.Import("megadrive", []gamelist.Game{{Path: "./Sonic.zip", Name: "Sonic the Hedgehog"}})

	if reg.Entries[1].Game.Name != "Sonic the Hedgehog" {
		t.Errorf("megadrive Name = %q, want it refreshed: the mark belongs to the mastersystem entry", reg.Entries[1].Game.Name)
	}
	if reg.Entries[0].Game.Name != "Sonic (hand-fixed)" {
		t.Errorf("mastersystem Name = %q, want it untouched", reg.Entries[0].Game.Name)
	}
}

// sonicEntry is the registry used by the UpdateMetadata tests: one fully
// scraped game, with its media referenced, and nothing marked yet.
func sonicEntry() *Registry {
	return &Registry{Entries: []Entry{{
		System: "megadrive",
		Game: gamelist.Game{
			Path: "./Sonic.zip", Name: "Sonic the Hedghog", Desc: "Fast.",
			Image: "images/sonic.png", Video: "videos/sonic.mp4",
			Marquee: "images/marquee.png", Thumbnail: "images/thumb.png",
			Rating: "0.85", ReleaseDate: "19910623T000000",
			Developer: "Sonic Team", Publisher: "Sega", Genre: "Platform", Players: "1",
		},
	}}}
}

// storedMetadata reads back the editable metadata of the registry's first
// entry, so a test can state what it expects field by field.
func storedMetadata(reg *Registry) Metadata {
	g := reg.Entries[0].Game
	return Metadata{
		Name: g.Name, Desc: g.Desc, Rating: g.Rating, ReleaseDate: g.ReleaseDate,
		Developer: g.Developer, Publisher: g.Publisher, Genre: g.Genre, Players: g.Players,
	}
}

func TestUpdateMetadata_CorrectedFields_AppliesThemAndMarksOnlyThem(t *testing.T) {
	reg := sonicEntry()
	m := storedMetadata(reg)
	m.Name = "Sonic the Hedgehog"
	m.Genre = "Platformer"

	if err := UpdateMetadata(reg, "megadrive", "Sonic", m, nil); err != nil {
		t.Fatalf("UpdateMetadata() error = %v, want nil", err)
	}

	got := reg.Entries[0]
	if got.Game.Name != "Sonic the Hedgehog" || got.Game.Genre != "Platformer" {
		t.Errorf("Game = %+v, want the corrected name and genre applied", got.Game)
	}
	if got.Game.Desc != "Fast." || got.Game.Rating != "0.85" {
		t.Errorf("Game = %+v, want the untouched fields left as they were", got.Game)
	}
	if strings.Join(got.ManualFields, ",") != "name,genre" {
		t.Errorf("ManualFields = %v, want exactly the two corrected fields", got.ManualFields)
	}
}

func TestUpdateMetadata_ValuesIdenticalToTheStoredOnes_MarksNothing(t *testing.T) {
	// Opening the form and saving without touching anything is not a
	// correction: marking every field would freeze the whole game against
	// future scrapes behind the user's back.
	reg := sonicEntry()

	if err := UpdateMetadata(reg, "megadrive", "Sonic", storedMetadata(reg), nil); err != nil {
		t.Fatalf("UpdateMetadata() error = %v, want nil", err)
	}

	if len(reg.Entries[0].ManualFields) != 0 {
		t.Errorf("ManualFields = %v, want none", reg.Entries[0].ManualFields)
	}
}

func TestUpdateMetadata_AlreadyMarkedFieldCorrectedAgain_StaysMarkedOnce(t *testing.T) {
	reg := sonicEntry()
	reg.Entries[0].ManualFields = []string{"name"}
	m := storedMetadata(reg)
	m.Name = "Sonic"

	if err := UpdateMetadata(reg, "megadrive", "Sonic", m, nil); err != nil {
		t.Fatalf("UpdateMetadata() error = %v, want nil", err)
	}

	if strings.Join(reg.Entries[0].ManualFields, ",") != "name" {
		t.Errorf("ManualFields = %v, want name listed exactly once", reg.Entries[0].ManualFields)
	}
}

func TestUpdateMetadata_FieldHandedBackToTheScraper_LosesItsMarkButKeepsItsValue(t *testing.T) {
	reg := sonicEntry()
	reg.Entries[0].ManualFields = []string{"name", "genre"}
	m := storedMetadata(reg)
	m.Name = "Sonic (fixed once more)"

	if err := UpdateMetadata(reg, "megadrive", "Sonic", m, []string{"name"}); err != nil {
		t.Fatalf("UpdateMetadata() error = %v, want nil", err)
	}

	got := reg.Entries[0]
	if got.Game.Name != "Sonic (fixed once more)" {
		t.Errorf("Name = %q, want the submitted value stored anyway", got.Game.Name)
	}
	if strings.Join(got.ManualFields, ",") != "genre" {
		t.Errorf("ManualFields = %v, want only genre: name was handed back to the scraper", got.ManualFields)
	}
}

func TestUpdateMetadata_EmptiedField_ClearsItAndMarksIt(t *testing.T) {
	// Emptying a wrong value is a correction like any other: the next import
	// must not put the wrong value back.
	reg := sonicEntry()
	m := storedMetadata(reg)
	m.Desc = ""

	if err := UpdateMetadata(reg, "megadrive", "Sonic", m, nil); err != nil {
		t.Fatalf("UpdateMetadata() error = %v, want nil", err)
	}

	if reg.Entries[0].Game.Desc != "" {
		t.Errorf("Desc = %q, want it cleared", reg.Entries[0].Game.Desc)
	}
	if strings.Join(reg.Entries[0].ManualFields, ",") != "desc" {
		t.Errorf("ManualFields = %v, want desc marked", reg.Entries[0].ManualFields)
	}
}

func TestUpdateMetadata_AnyCorrection_LeavesTheRomPathAndTheMediaUntouched(t *testing.T) {
	reg := sonicEntry()
	before := reg.Entries[0].Game
	m := storedMetadata(reg)
	m.Name = "Anything else"

	if err := UpdateMetadata(reg, "megadrive", "Sonic", m, nil); err != nil {
		t.Fatalf("UpdateMetadata() error = %v, want nil", err)
	}

	got := reg.Entries[0].Game
	if got.Path != before.Path {
		t.Errorf("Path = %q, want %q: the ROM path identifies the entry and is never edited", got.Path, before.Path)
	}
	if got.Image != before.Image || got.Video != before.Video || got.Marquee != before.Marquee || got.Thumbnail != before.Thumbnail {
		t.Errorf("media = %q/%q/%q/%q, want them untouched", got.Image, got.Video, got.Marquee, got.Thumbnail)
	}
}

func TestUpdateMetadata_UnknownGame_ReturnsErrGameNotFoundWithoutChangingAnything(t *testing.T) {
	reg := sonicEntry()
	before := reg.Entries[0]

	err := UpdateMetadata(reg, "megadrive", "Golden Axe", Metadata{Name: "Golden Axe"}, nil)

	if !errors.Is(err, ErrGameNotFound) {
		t.Errorf("error = %v, want ErrGameNotFound", err)
	}
	if reg.Entries[0].Game != before.Game || len(reg.Entries[0].ManualFields) != 0 {
		t.Errorf("entry = %+v, want it untouched", reg.Entries[0])
	}
}

func TestUpdateMetadata_UnknownSystem_ReturnsErrGameNotFound(t *testing.T) {
	reg := sonicEntry()

	err := UpdateMetadata(reg, "mastersystem", "Sonic", Metadata{Name: "Sonic"}, nil)

	if !errors.Is(err, ErrGameNotFound) {
		t.Errorf("error = %v, want ErrGameNotFound", err)
	}
}

func TestUpdateMetadata_GameIDContainingADot_IsFoundByItsFullID(t *testing.T) {
	// "Micro Machines v3.0" must not be truncated to "Micro Machines v3" on
	// the way in, the way a second GameID pass would.
	reg := &Registry{Entries: []Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Micro Machines v3.0.zip", Name: "Micro Machines"}},
	}}

	if err := UpdateMetadata(reg, "megadrive", "Micro Machines v3.0", Metadata{Name: "Micro Machines V3"}, nil); err != nil {
		t.Fatalf("UpdateMetadata() error = %v, want nil", err)
	}

	if reg.Entries[0].Game.Name != "Micro Machines V3" {
		t.Errorf("Name = %q, want the correction applied", reg.Entries[0].Game.Name)
	}
}

// fullyScrapedGame is a game with every editable field set, so a test can
// assert that protecting or lifting a protection changed none of them.
func fullyScrapedGame() gamelist.Game {
	return gamelist.Game{
		Path:        "./Sonic.zip",
		Name:        "Sonic the Hedgehog",
		Desc:        "Fast blue mascot.",
		Image:       "./images/Sonic.png",
		Video:       "./videos/Sonic.mp4",
		Rating:      "0.8",
		ReleaseDate: "19910623T000000",
		Developer:   "Sonic Team",
		Publisher:   "Sega",
		Genre:       "Platform",
		Players:     "1",
	}
}

func TestProtect_ScrapedGame_MarksEveryEditableFieldWithoutChangingAValue(t *testing.T) {
	game := fullyScrapedGame()
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: game}}}

	if err := Protect(reg, "megadrive", "Sonic"); err != nil {
		t.Fatalf("Protect() error = %v, want nil", err)
	}

	got := reg.Entries[0]
	if got.Game != game {
		t.Errorf("Game = %+v, want it byte-identical to %+v: protecting states values are right, it does not change them", got.Game, game)
	}
	want := []string{"name", "desc", "rating", "release_date", "developer", "publisher", "genre", "players"}
	for _, field := range want {
		if !slices.Contains(got.ManualFields, field) {
			t.Errorf("ManualFields = %v, want it to contain %q", got.ManualFields, field)
		}
	}
	if len(got.ManualFields) != len(want) {
		t.Errorf("ManualFields = %v, want exactly the %d editable fields", got.ManualFields, len(want))
	}
}

func TestProtect_ThenImportingADifferentlyScrapedGame_OverwritesNothing(t *testing.T) {
	// The whole point of the feature: an update run against a ROMs folder that
	// still holds the badly scraped values must leave the game alone.
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: fullyScrapedGame()}}}
	if err := Protect(reg, "megadrive", "Sonic"); err != nil {
		t.Fatalf("Protect() error = %v, want nil", err)
	}

	added, updated, unchanged := reg.Import("megadrive", []gamelist.Game{{
		Path:        "./Sonic.zip",
		Name:        "Sonic teh Hedghog",
		Desc:        "Badly scraped.",
		Image:       "./images/Sonic.png",
		Video:       "./videos/Sonic.mp4",
		Rating:      "0.2",
		ReleaseDate: "19950101T000000",
		Developer:   "Unknown",
		Publisher:   "Unknown",
		Genre:       "Action",
		Players:     "4",
	}})

	if added != 0 || updated != 0 || unchanged != 1 {
		t.Errorf("added=%d updated=%d unchanged=%d, want 0,0,1", added, updated, unchanged)
	}
	if got := reg.Entries[0].Game; got != fullyScrapedGame() {
		t.Errorf("Game = %+v, want every field to survive the import unchanged", got)
	}
}

func TestProtect_ThenImportingAGameWhoseMediaMoved_StillRefreshesTheMedia(t *testing.T) {
	// Protection covers the eight editable metadata fields and nothing else:
	// media references are managed by their own flow, so a protected game still
	// follows its artwork when the ROMs folder points somewhere else. Were it
	// otherwise, a protected entry would keep pointing at a file that moved.
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: fullyScrapedGame()}}}
	if err := Protect(reg, "megadrive", "Sonic"); err != nil {
		t.Fatalf("Protect() error = %v, want nil", err)
	}

	reg.Import("megadrive", []gamelist.Game{{
		Path:  "./Sonic.zip",
		Name:  "Sonic teh Hedghog",
		Image: "./images/Sonic-boxart.png",
		Video: "./videos/Sonic.mp4",
	}})

	got := reg.Entries[0].Game
	if got.Image != "./images/Sonic-boxart.png" {
		t.Errorf("Image = %q, want the media reference to follow the ROMs folder", got.Image)
	}
	if got.Name != "Sonic the Hedgehog" {
		t.Errorf("Name = %q, want the protected metadata untouched", got.Name)
	}
}

func TestProtect_GameAlreadyProtected_StaysMarkedOnceEach(t *testing.T) {
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: fullyScrapedGame()}}}

	for range 3 {
		if err := Protect(reg, "megadrive", "Sonic"); err != nil {
			t.Fatalf("Protect() error = %v, want nil", err)
		}
	}

	if got := len(reg.Entries[0].ManualFields); got != 8 {
		t.Errorf("len(ManualFields) = %d, want 8: repeating the command must not grow the stored list", got)
	}
}

func TestProtect_GameWithOneFieldAlreadyHandEdited_KeepsThatFieldMarkedOnce(t *testing.T) {
	reg := &Registry{Entries: []Entry{{
		System:       "megadrive",
		Game:         fullyScrapedGame(),
		ManualFields: []string{"name"},
	}}}

	if err := Protect(reg, "megadrive", "Sonic"); err != nil {
		t.Fatalf("Protect() error = %v, want nil", err)
	}

	got := reg.Entries[0].ManualFields
	if len(got) != 8 {
		t.Errorf("ManualFields = %v, want the 8 editable fields with no duplicate", got)
	}
}

func TestProtect_UnknownGame_ReturnsErrGameNotFoundWithoutChangingAnything(t *testing.T) {
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: fullyScrapedGame()}}}

	err := Protect(reg, "megadrive", "Streets of Rage")

	if !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("Protect() error = %v, want ErrGameNotFound", err)
	}
	if len(reg.Entries[0].ManualFields) != 0 {
		t.Errorf("ManualFields = %v, want the known game left untouched", reg.Entries[0].ManualFields)
	}
}

func TestProtect_UnknownSystem_ReturnsErrGameNotFound(t *testing.T) {
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: fullyScrapedGame()}}}

	if err := Protect(reg, "nes", "Sonic"); !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("Protect() error = %v, want ErrGameNotFound", err)
	}
}

func TestProtect_GameIDContainingADot_IsFoundByItsFullID(t *testing.T) {
	// Same trap as UpdateMetadata's: running the id through GameID again would
	// truncate it at the dot and lose the entry.
	reg := &Registry{Entries: []Entry{{
		System: "psx",
		Game:   gamelist.Game{Path: "./Micro Machines v3.0.chd", Name: "Micro Machines V3"},
	}}}

	if err := Protect(reg, "psx", "Micro Machines v3.0"); err != nil {
		t.Fatalf("Protect() error = %v, want nil", err)
	}
	if len(reg.Entries[0].ManualFields) != 8 {
		t.Errorf("ManualFields = %v, want the entry to have been found and protected", reg.Entries[0].ManualFields)
	}
}

func TestProtect_AppliedToAClone_DoesNotWriteThroughToTheOriginal(t *testing.T) {
	// The web UI protects a clone, persists it, and only then swaps it in: a
	// failed write must leave the served snapshot exactly as it was.
	original := &Registry{Entries: []Entry{{
		System:       "megadrive",
		Game:         fullyScrapedGame(),
		ManualFields: []string{"name"},
	}}}
	clone := original.Clone()

	if err := Protect(clone, "megadrive", "Sonic"); err != nil {
		t.Fatalf("Protect() error = %v, want nil", err)
	}

	if got := original.Entries[0].ManualFields; strings.Join(got, ",") != "name" {
		t.Errorf("original ManualFields = %v, want [name] untouched", got)
	}
}

func TestUnprotect_ProtectedGame_ClearsEveryMarkWithoutChangingAValue(t *testing.T) {
	game := fullyScrapedGame()
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: game}}}
	if err := Protect(reg, "megadrive", "Sonic"); err != nil {
		t.Fatalf("Protect() error = %v, want nil", err)
	}

	if err := Unprotect(reg, "megadrive", "Sonic"); err != nil {
		t.Fatalf("Unprotect() error = %v, want nil", err)
	}

	got := reg.Entries[0]
	if len(got.ManualFields) != 0 {
		t.Errorf("ManualFields = %v, want none left", got.ManualFields)
	}
	if got.Game != game {
		t.Errorf("Game = %+v, want it unchanged: lifting a protection stores no new value", got.Game)
	}
}

func TestUnprotect_ThenImporting_RefreshesTheGameAgain(t *testing.T) {
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: fullyScrapedGame()}}}
	if err := Protect(reg, "megadrive", "Sonic"); err != nil {
		t.Fatalf("Protect() error = %v, want nil", err)
	}
	if err := Unprotect(reg, "megadrive", "Sonic"); err != nil {
		t.Fatalf("Unprotect() error = %v, want nil", err)
	}

	_, updated, _ := reg.Import("megadrive", []gamelist.Game{{Path: "./Sonic.zip", Name: "Sonic", Genre: "Action"}})

	if updated != 1 {
		t.Errorf("updated = %d, want 1: an unprotected game is refreshable again", updated)
	}
	if got := reg.Entries[0].Game.Genre; got != "Action" {
		t.Errorf("Genre = %q, want %q from the ROMs folder", got, "Action")
	}
}

func TestUnprotect_GameWithOnlySomeFieldsHandEdited_ClearsThoseToo(t *testing.T) {
	// Lifting is a whole-game statement: it hands everything back, including
	// the marks left by earlier per-field corrections (see decisions/021).
	reg := &Registry{Entries: []Entry{{
		System:       "megadrive",
		Game:         fullyScrapedGame(),
		ManualFields: []string{"name", "genre"},
	}}}

	if err := Unprotect(reg, "megadrive", "Sonic"); err != nil {
		t.Fatalf("Unprotect() error = %v, want nil", err)
	}

	if got := reg.Entries[0].ManualFields; len(got) != 0 {
		t.Errorf("ManualFields = %v, want none left", got)
	}
}

func TestUnprotect_GameThatWasNotProtected_SucceedsAndChangesNothing(t *testing.T) {
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: fullyScrapedGame()}}}

	if err := Unprotect(reg, "megadrive", "Sonic"); err != nil {
		t.Fatalf("Unprotect() error = %v, want nil: lifting an absent protection is not a failure", err)
	}
	if len(reg.Entries[0].ManualFields) != 0 {
		t.Errorf("ManualFields = %v, want none", reg.Entries[0].ManualFields)
	}
}

func TestUnprotect_UnknownGame_ReturnsErrGameNotFoundWithoutChangingAnything(t *testing.T) {
	reg := &Registry{Entries: []Entry{{
		System:       "megadrive",
		Game:         fullyScrapedGame(),
		ManualFields: []string{"name"},
	}}}

	err := Unprotect(reg, "megadrive", "Streets of Rage")

	if !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("Unprotect() error = %v, want ErrGameNotFound", err)
	}
	if strings.Join(reg.Entries[0].ManualFields, ",") != "name" {
		t.Errorf("ManualFields = %v, want the known game left untouched", reg.Entries[0].ManualFields)
	}
}

func TestUnprotect_AppliedToAClone_DoesNotWriteThroughToTheOriginal(t *testing.T) {
	original := &Registry{Entries: []Entry{{
		System:       "megadrive",
		Game:         fullyScrapedGame(),
		ManualFields: []string{"name", "genre"},
	}}}
	clone := original.Clone()

	if err := Unprotect(clone, "megadrive", "Sonic"); err != nil {
		t.Fatalf("Unprotect() error = %v, want nil", err)
	}

	if got := original.Entries[0].ManualFields; strings.Join(got, ",") != "name,genre" {
		t.Errorf("original ManualFields = %v, want [name genre] untouched", got)
	}
}

func TestFullyProtected_EveryEditableFieldMarked_ReportsTrue(t *testing.T) {
	entry := Entry{ManualFields: []string{
		"players", "genre", "publisher", "developer", "release_date", "rating", "desc", "name",
	}}

	if !entry.FullyProtected() {
		t.Errorf("FullyProtected() = false, want true: the marks are the whole set, order is irrelevant")
	}
}

func TestFullyProtected_OneEditableFieldMissing_ReportsFalse(t *testing.T) {
	// desc is not among the rows a game's page shows, so counting only the
	// displayed ones would wrongly call this game protected.
	entry := Entry{ManualFields: []string{
		"name", "rating", "release_date", "developer", "publisher", "genre", "players",
	}}

	if entry.FullyProtected() {
		t.Errorf("FullyProtected() = true, want false: %q is still refreshable", "desc")
	}
}

func TestFullyProtected_NothingMarked_ReportsFalse(t *testing.T) {
	if (Entry{}).FullyProtected() {
		t.Errorf("FullyProtected() = true, want false for a game with no mark at all")
	}
}

func TestFullyProtected_EveryFieldMarkedPlusAnUnknownName_StillReportsTrue(t *testing.T) {
	// A mark written by a later version of the tool must not make the game read
	// as partly protected.
	entry := Entry{ManualFields: []string{
		"name", "desc", "rating", "release_date", "developer", "publisher", "genre", "players", "region",
	}}

	if !entry.FullyProtected() {
		t.Errorf("FullyProtected() = false, want true: an unknown extra mark protects nothing but breaks nothing")
	}
}
