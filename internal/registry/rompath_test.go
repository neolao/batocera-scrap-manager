package registry

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
)

// twoGames builds a registry holding Sonic and Mario in megadrive, plus a
// Sonic of another system — the shape most of these tests need to prove the
// duplicate check is per-system.
func twoGames() *Registry {
	return &Registry{Entries: []Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic the Hedgehog", Image: "./images/Sonic-image.png"}},
		{System: "megadrive", Game: gamelist.Game{Path: "./Mario.zip", Name: "Mario"}},
		{System: "snes", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic on the SNES"}},
	}}
}

func TestChangePath_NewPath_StoresItAndMovesTheEntryUnderTheNewIdentifier(t *testing.T) {
	reg := twoGames()

	if err := ChangePath(reg, "megadrive", "Sonic", "disc1/Sonic Adventure.zip"); err != nil {
		t.Fatalf("ChangePath() error = %v, want nil", err)
	}

	if reg.Entries[0].Game.Path != "disc1/Sonic Adventure.zip" {
		t.Errorf("Game.Path = %q, want the submitted path stored as it was given", reg.Entries[0].Game.Path)
	}
	if _, found := reg.FindByID("megadrive", "Sonic Adventure"); !found {
		t.Error("FindByID(megadrive, \"Sonic Adventure\") found nothing, want the renamed game")
	}
	if _, found := reg.FindByID("megadrive", "Sonic"); found {
		t.Error("FindByID(megadrive, \"Sonic\") still finds a game, want the old identifier to be free")
	}
	if len(reg.Entries) != 3 {
		t.Errorf("Entries = %d, want 3: a rename adds nothing", len(reg.Entries))
	}
}

func TestChangePath_PathDifferingOnlyBySubfolderAndExtension_KeepsTheSameIdentifier(t *testing.T) {
	// disc1/Sonic.iso and ./Sonic.zip share a base name, so they are the same
	// game: the entry moves nowhere, but the stored path must still change.
	reg := twoGames()

	if err := ChangePath(reg, "megadrive", "Sonic", "disc1/Sonic.iso"); err != nil {
		t.Fatalf("ChangePath() error = %v, want nil", err)
	}

	if reg.Entries[0].Game.Path != "disc1/Sonic.iso" {
		t.Errorf("Game.Path = %q, want \"disc1/Sonic.iso\"", reg.Entries[0].Game.Path)
	}
	if _, found := reg.FindByID("megadrive", "Sonic"); !found {
		t.Error("FindByID(megadrive, Sonic) found nothing, want the game still under its unchanged identifier")
	}
}

func TestChangePath_PathIdenticalToTheStoredOne_IsNotRefusedAsADuplicateOfItself(t *testing.T) {
	// The entry itself holds the identifier being asked for: comparing by
	// identifier alone rather than by position would make a game refuse its
	// own path.
	reg := twoGames()

	if err := ChangePath(reg, "megadrive", "Sonic", "./Sonic.zip"); err != nil {
		t.Fatalf("ChangePath() error = %v, want nil", err)
	}
	if reg.Entries[0].Game.Path != "./Sonic.zip" {
		t.Errorf("Game.Path = %q, want it left as it was", reg.Entries[0].Game.Path)
	}
}

func TestChangePath_IdentifierAlreadyUsedByAnotherGame_IsRefusedWithoutOverwritingIt(t *testing.T) {
	reg := twoGames()

	err := ChangePath(reg, "megadrive", "Mario", "disc2/Sonic.zip")

	if !errors.Is(err, ErrDuplicateGameID) {
		t.Fatalf("ChangePath() error = %v, want ErrDuplicateGameID", err)
	}
	if reg.Entries[1].Game.Path != "./Mario.zip" {
		t.Errorf("Mario's Game.Path = %q, want it untouched", reg.Entries[1].Game.Path)
	}
	if reg.Entries[0].Game.Name != "Sonic the Hedgehog" {
		t.Errorf("the game holding the identifier is now %q, want it untouched", reg.Entries[0].Game.Name)
	}
	if len(reg.Entries) != 3 {
		t.Errorf("Entries = %d, want 3", len(reg.Entries))
	}
}

func TestChangePath_IdentifierFreedByAnEarlierRemoval_Succeeds(t *testing.T) {
	// A lookup index invalidated by RemoveByID is rebuilt lazily on the next
	// lookup — this pins that a rename right after such a removal does not
	// resolve a stale mapping still pointing at the now-gone entry's old
	// slot, refusing the identifier as a duplicate of a game that left.
	reg := twoGames()
	registryFolder := t.TempDir()
	if err := RemoveByID(reg, registryFolder, "megadrive", "Mario"); err != nil {
		t.Fatalf("RemoveByID() error = %v, want nil", err)
	}

	if err := ChangePath(reg, "megadrive", "Sonic", "./Mario.zip"); err != nil {
		t.Fatalf("ChangePath() error = %v, want nil", err)
	}

	if _, found := reg.FindByID("megadrive", "Mario"); !found {
		t.Error("FindByID(megadrive, Mario) found nothing, want the renamed Sonic entry")
	}
	if _, found := reg.FindByID("megadrive", "Sonic"); found {
		t.Error("FindByID(megadrive, Sonic) still finds a game, want the old identifier free")
	}
	if len(reg.Entries) != 2 {
		t.Errorf("Entries = %d, want 2 (Mario removed, Sonic renamed)", len(reg.Entries))
	}
}

func TestChangePath_IdentifierUsedByAGameOfAnotherSystem_IsAccepted(t *testing.T) {
	// snes already holds a "Sonic": identifiers are unique per system, not
	// registry-wide, since each system has its own folder.
	reg := twoGames()

	if err := ChangePath(reg, "megadrive", "Mario", "./Sonic on the SNES.zip"); err != nil {
		t.Fatalf("ChangePath() error = %v, want nil", err)
	}
	if reg.Entries[1].Game.Path != "./Sonic on the SNES.zip" {
		t.Errorf("Game.Path = %q, want the submitted path", reg.Entries[1].Game.Path)
	}
}

func TestChangePath_UnknownGame_ReturnsErrGameNotFoundWithoutChangingAnything(t *testing.T) {
	reg := twoGames()

	err := ChangePath(reg, "megadrive", "Ecco", "./Ecco the Dolphin.zip")

	if !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("ChangePath() error = %v, want ErrGameNotFound", err)
	}
	if reg.Entries[0].Game.Path != "./Sonic.zip" || reg.Entries[1].Game.Path != "./Mario.zip" {
		t.Error("an unknown game changed a path, want the registry left alone")
	}
}

func TestChangePath_UnknownSystem_ReturnsErrGameNotFound(t *testing.T) {
	reg := twoGames()

	if err := ChangePath(reg, "gamegear", "Sonic", "./Sonic 2.zip"); !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("ChangePath() error = %v, want ErrGameNotFound", err)
	}
}

func TestChangePath_RefusedPath_LeavesTheStoredPathUntouched(t *testing.T) {
	for _, tc := range []struct {
		name    string
		path    string
		wantErr error
	}{
		{"empty", "", ErrEmptyPath},
		{"blank", "   ", ErrEmptyPath},
		{"absolute", "/roms/megadrive/Sonic.zip", ErrAbsolutePath},
		{"escaping through ..", "../snes/Sonic.zip", ErrEscapingPath},
		{"nothing but ..", "..", ErrEscapingPath},
		{"escaping deeper in", "disc1/../../Sonic.zip", ErrEscapingPath},
		{"backslash separator", `disc1\Sonic.zip`, ErrUnusablePath},
		{"trailing separator", "disc1/", ErrUnusablePath},
		{"naming a folder", "disc1/.", ErrUnusablePath},
		{"extension only", ".zip", ErrUnusablePath},
		{"extension only in a subfolder", "disc1/.iso", ErrUnusablePath},
		{"control character", "Sonic\n.zip", ErrUnusablePath},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reg := twoGames()

			err := ChangePath(reg, "megadrive", "Sonic", tc.path)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ChangePath(%q) error = %v, want %v", tc.path, err, tc.wantErr)
			}
			if reg.Entries[0].Game.Path != "./Sonic.zip" {
				t.Errorf("Game.Path = %q, want it strictly unchanged", reg.Entries[0].Game.Path)
			}
		})
	}
}

func TestChangePath_IdentifierTooLongForAFilename_IsRefused(t *testing.T) {
	reg := twoGames()

	err := ChangePath(reg, "megadrive", "Sonic", strings.Repeat("a", 252)+".zip")

	if !errors.Is(err, ErrUnusablePath) {
		t.Fatalf("ChangePath() error = %v, want ErrUnusablePath: %q.json exceeds 255 bytes", err, strings.Repeat("a", 252))
	}
	if reg.Entries[0].Game.Path != "./Sonic.zip" {
		t.Errorf("Game.Path = %q, want it unchanged", reg.Entries[0].Game.Path)
	}
}

func TestChangePath_PathWalkingBackIntoItsOwnFolder_IsAccepted(t *testing.T) {
	// disc1/../Sonic 2.zip resolves to Sonic 2.zip, which never leaves the
	// system folder: only a path that really escapes is refused.
	reg := twoGames()

	if err := ChangePath(reg, "megadrive", "Sonic", "disc1/../Sonic 2.zip"); err != nil {
		t.Fatalf("ChangePath() error = %v, want nil", err)
	}
	if _, found := reg.FindByID("megadrive", "Sonic 2"); !found {
		t.Error("FindByID(megadrive, \"Sonic 2\") found nothing, want the renamed game")
	}
}

func TestChangePath_AnyRename_LeavesTheHandEditedMarksAndTheMediaAlone(t *testing.T) {
	// The path identifies the entry, it is not one of its values: renaming it
	// must neither mark anything as hand-edited nor touch a media reference.
	reg := &Registry{Entries: []Entry{{
		System:       "megadrive",
		Game:         gamelist.Game{Path: "./Sonic.zip", Name: "Sonic", Image: "./images/Sonic-image.png", Video: "./videos/Sonic.mp4"},
		ManualFields: []string{"name"},
	}}}

	if err := ChangePath(reg, "megadrive", "Sonic", "./Sonic 2.zip"); err != nil {
		t.Fatalf("ChangePath() error = %v, want nil", err)
	}

	if strings.Join(reg.Entries[0].ManualFields, ",") != "name" {
		t.Errorf("ManualFields = %v, want only [name]: a path is never a hand-edited field", reg.Entries[0].ManualFields)
	}
	if reg.Entries[0].Game.Image != "./images/Sonic-image.png" {
		t.Errorf("Game.Image = %q, want it untouched", reg.Entries[0].Game.Image)
	}
	if reg.Entries[0].Game.Video != "./videos/Sonic.mp4" {
		t.Errorf("Game.Video = %q, want it untouched", reg.Entries[0].Game.Video)
	}
	if reg.Entries[0].Game.Name != "Sonic" {
		t.Errorf("Game.Name = %q, want it untouched", reg.Entries[0].Game.Name)
	}
}

func TestChangePath_AppliedToAClone_LeavesTheOriginalUntouched(t *testing.T) {
	// The web UI renames on a Clone and only swaps it in once the write
	// succeeded: writing through would defeat that.
	reg := twoGames()
	clone := reg.Clone()

	if err := ChangePath(clone, "megadrive", "Sonic", "./Sonic 2.zip"); err != nil {
		t.Fatalf("ChangePath() error = %v, want nil", err)
	}

	if reg.Entries[0].Game.Path != "./Sonic.zip" {
		t.Errorf("the original's Game.Path = %q, want \"./Sonic.zip\"", reg.Entries[0].Game.Path)
	}
	if clone.Entries[0].Game.Path != "./Sonic 2.zip" {
		t.Errorf("the clone's Game.Path = %q, want \"./Sonic 2.zip\"", clone.Entries[0].Game.Path)
	}
}

func TestValidatePath_UsablePath_ReturnsNil(t *testing.T) {
	for _, romPath := range []string{
		"Sonic.zip",
		"./Sonic.zip",
		"disc1/Sonic.zip",
		"disc1/disc2/Sonic Adventure (USA).zip",
		"Sonic",
		"Micro Machines v3.0.zip",
	} {
		t.Run(romPath, func(t *testing.T) {
			if err := ValidatePath(romPath); err != nil {
				t.Errorf("ValidatePath(%q) error = %v, want nil", romPath, err)
			}
		})
	}
}

func TestValidatePath_RefusedPath_NamesWhichRuleItBreaks(t *testing.T) {
	// Each reason is told apart because each maps to its own message in the
	// form: a single "invalid path" would leave the user guessing.
	for _, tc := range []struct {
		path    string
		wantErr error
	}{
		{"", ErrEmptyPath},
		{"\t", ErrEmptyPath},
		{"/Sonic.zip", ErrAbsolutePath},
		{"/", ErrAbsolutePath},
		{"../Sonic.zip", ErrEscapingPath},
		{"..", ErrEscapingPath},
		{`a\b.zip`, ErrUnusablePath},
		{"disc1/", ErrUnusablePath},
		{".", ErrUnusablePath},
		{".zip", ErrUnusablePath},
	} {
		t.Run(tc.path, func(t *testing.T) {
			if err := ValidatePath(tc.path); !errors.Is(err, tc.wantErr) {
				t.Errorf("ValidatePath(%q) error = %v, want %v", tc.path, err, tc.wantErr)
			}
		})
	}
}

func TestRemoveGameFile_EntryFileOfTheOldIdentifier_IsDeletedAndTheMediaLeftAlone(t *testing.T) {
	// Renaming moves the metadata file only: media are referenced by their own
	// paths, which a rename never derives from the ROM path.
	folder := t.TempDir()
	system := filepath.Join(folder, "megadrive")
	images := filepath.Join(system, "images")
	if err := os.MkdirAll(images, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	entryFile := filepath.Join(system, "Sonic.json")
	medium := filepath.Join(images, "Sonic-image.png")
	for _, path := range []string{entryFile, medium} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	if err := RemoveGameFile(folder, "megadrive", "Sonic"); err != nil {
		t.Fatalf("RemoveGameFile() error = %v, want nil", err)
	}

	if _, err := os.Stat(entryFile); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Stat(Sonic.json) err = %v, want the file to be gone", err)
	}
	if _, err := os.Stat(medium); err != nil {
		t.Errorf("Stat(Sonic-image.png) err = %v, want the medium left in place", err)
	}
}

func TestRemoveGameFile_FileAlreadyGone_IsNotAnError(t *testing.T) {
	folder := t.TempDir()

	if err := RemoveGameFile(folder, "megadrive", "Sonic"); err != nil {
		t.Errorf("RemoveGameFile() error = %v, want nil: nothing to delete is the wanted state", err)
	}
}

func TestRemoveGameFile_IdentifierReachingOutsideTheSystemFolder_DeletesNothing(t *testing.T) {
	// An identifier is a base name by construction, but os.Remove has no undo:
	// one that carries a separator is refused rather than followed.
	folder := t.TempDir()
	outside := filepath.Join(folder, "keepme.json")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	for _, id := range []string{"../keepme", "..", ".", "", "disc1/Sonic"} {
		t.Run(id, func(t *testing.T) {
			if err := RemoveGameFile(folder, "megadrive", id); err == nil {
				t.Errorf("RemoveGameFile(%q) error = nil, want a refusal", id)
			}
		})
	}

	if _, err := os.Stat(outside); err != nil {
		t.Errorf("Stat(keepme.json) err = %v, want the file left untouched", err)
	}
}

func TestRemoveGameFile_ThenLoad_LeavesOnlyTheRenamedEntry(t *testing.T) {
	// The whole point of the deletion: Save never removes, so without it the
	// old file would resurrect the game as a duplicate on the next Load.
	folder := t.TempDir()
	reg := &Registry{Entries: []Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic"}},
	}}
	if err := Save(folder, reg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}
	if err := ChangePath(reg, "megadrive", "Sonic", "./Sonic 2.zip"); err != nil {
		t.Fatalf("ChangePath() error = %v, want nil", err)
	}
	if err := Save(folder, reg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	if err := RemoveGameFile(folder, "megadrive", "Sonic"); err != nil {
		t.Fatalf("RemoveGameFile() error = %v, want nil", err)
	}

	loaded, err := Load(folder)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("Entries = %d, want exactly 1: the old file must not resurrect the game", len(loaded.Entries))
	}
	if loaded.Entries[0].Game.Path != "./Sonic 2.zip" {
		t.Errorf("Game.Path = %q, want \"./Sonic 2.zip\"", loaded.Entries[0].Game.Path)
	}
}
