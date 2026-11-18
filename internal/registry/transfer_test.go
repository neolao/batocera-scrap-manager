package registry

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
)

// writeTransferFixture builds a megadrive ROMs folder whose gamelist.xml
// lists Golden Axe with a local name, genre and cover art file, and a
// registry folder holding Golden Axe with another name, a description and
// its own cover art file.
func writeTransferFixture(t *testing.T) (reg *Registry, romsFolder, registryFolder string) {
	t.Helper()
	romsFolder, registryFolder = t.TempDir(), t.TempDir()
	write := func(path, content string) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	write(filepath.Join(romsFolder, "megadrive", "Golden Axe.zip"), "rom")
	write(filepath.Join(romsFolder, "megadrive", "Lonely.zip"), "rom")
	write(filepath.Join(romsFolder, "megadrive", "images", "local.png"), "local-art")
	write(filepath.Join(registryFolder, "megadrive", "images", "reg.png"), "registry-art")
	local := []gamelist.Game{
		{Path: "./Golden Axe.zip", Name: "Golden Axe (local)", Genre: "Beat 'em up", Image: "./images/local.png"},
		{Path: "./Lonely.zip", Name: "Lonely", Desc: "Only known here."},
	}
	if err := gamelist.UpdateFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"), local); err != nil {
		t.Fatalf("write gamelist: %v", err)
	}
	reg = &Registry{Entries: []Entry{{System: "megadrive", ManualFields: []string{FieldName}, Game: gamelist.Game{
		Path: "./Golden Axe.zip", Name: "Golden Axe", Desc: "Two warriors.", Image: "./images/reg.png",
	}}}}
	return reg, romsFolder, registryFolder
}

func TestLocalGame_ListedRom_ReturnsItsEntry(t *testing.T) {
	_, romsFolder, _ := writeTransferFixture(t)

	g, err := LocalGame(romsFolder, "megadrive", "Golden Axe.zip")

	if err != nil || g.Name != "Golden Axe (local)" {
		t.Errorf("LocalGame() = %+v, %v; want the local entry", g, err)
	}
}

func TestLocalGame_UnlistedRomOrUnsafeSystem_ReturnsErrGameNotFound(t *testing.T) {
	_, romsFolder, _ := writeTransferFixture(t)

	for _, c := range [][2]string{{"megadrive", "Nowhere.zip"}, {"snes", "Golden Axe.zip"}, {"..", "Golden Axe.zip"}} {
		if _, err := LocalGame(romsFolder, c[0], c[1]); !errors.Is(err, ErrGameNotFound) {
			t.Errorf("LocalGame(%q, %q) error = %v, want ErrGameNotFound", c[0], c[1], err)
		}
	}
}

func TestRetrieveFields_TickedValuesOnly_OverwriteTheRegistryAndCopyMedia(t *testing.T) {
	reg, romsFolder, registryFolder := writeTransferFixture(t)

	added, err := RetrieveFields(reg, romsFolder, registryFolder, "megadrive", "Golden Axe.zip", []string{FieldGenre, string(MediumImage)})

	if err != nil || added {
		t.Fatalf("RetrieveFields() added=%v err=%v, want an update without error", added, err)
	}
	g := reg.Entries[0].Game
	if g.Name != "Golden Axe" || g.Desc != "Two warriors." {
		t.Errorf("unticked values changed: %+v", g)
	}
	if g.Genre != "Beat 'em up" || g.Image != "./images/local.png" {
		t.Errorf("ticked values not taken: genre %q, image %q", g.Genre, g.Image)
	}
	data, err := os.ReadFile(filepath.Join(registryFolder, "megadrive", "images", "local.png"))
	if err != nil || string(data) != "local-art" {
		t.Errorf("the local cover art was not copied into the registry (err %v)", err)
	}
	if len(reg.Entries[0].ManualFields) != 1 {
		t.Errorf("ManualFields = %v, want the existing mark kept as is", reg.Entries[0].ManualFields)
	}
}

func TestRetrieveFields_GameUnknownToRegistry_CreatesItsEntry(t *testing.T) {
	reg, romsFolder, registryFolder := writeTransferFixture(t)

	added, err := RetrieveFields(reg, romsFolder, registryFolder, "megadrive", "Lonely.zip", []string{FieldName, FieldDesc})

	if err != nil || !added {
		t.Fatalf("RetrieveFields() added=%v err=%v, want the game added", added, err)
	}
	entry, found := reg.FindByID("megadrive", "Lonely")
	if !found || entry.Game.Path != "./Lonely.zip" || entry.Game.Desc != "Only known here." {
		t.Errorf("entry = %+v (found %v), want Lonely with its local description", entry, found)
	}
}

func TestRetrieveFields_NothingTickedOrNoLocalEntry_ChangesNothing(t *testing.T) {
	reg, romsFolder, registryFolder := writeTransferFixture(t)
	before := reg.Entries[0].Game

	if _, err := RetrieveFields(reg, romsFolder, registryFolder, "megadrive", "Golden Axe.zip", nil); !errors.Is(err, ErrNothingToTransfer) {
		t.Errorf("nothing ticked: error = %v, want ErrNothingToTransfer", err)
	}
	if _, err := RetrieveFields(reg, romsFolder, registryFolder, "megadrive", "Nowhere.zip", []string{FieldName}); !errors.Is(err, ErrGameNotFound) {
		t.Errorf("no local entry: error = %v, want ErrGameNotFound", err)
	}
	if reg.Entries[0].Game != before || len(reg.Entries) != 1 {
		t.Error("a refused transfer changed the registry")
	}
}

func TestRetrieveFields_MediumOutsideTheSystemFolder_IsRefused(t *testing.T) {
	reg, romsFolder, registryFolder := writeTransferFixture(t)
	local := []gamelist.Game{{Path: "./Golden Axe.zip", Name: "Golden Axe", Image: "../../escape.png"}}
	if err := gamelist.UpdateFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"), local); err != nil {
		t.Fatalf("write gamelist: %v", err)
	}

	_, err := RetrieveFields(reg, romsFolder, registryFolder, "megadrive", "Golden Axe.zip", []string{string(MediumImage)})

	if !errors.Is(err, ErrUnsafeMediaPath) {
		t.Errorf("error = %v, want ErrUnsafeMediaPath", err)
	}
}

func TestSendFields_TickedValuesOnly_OverwriteTheFolderAndCopyMedia(t *testing.T) {
	reg, romsFolder, registryFolder := writeTransferFixture(t)

	changed, added, err := SendFields(reg, romsFolder, registryFolder, "megadrive", "Golden Axe.zip", []string{FieldName, string(MediumImage)})

	if err != nil || !changed || added {
		t.Fatalf("SendFields() changed=%v added=%v err=%v", changed, added, err)
	}
	g, _ := LocalGame(romsFolder, "megadrive", "Golden Axe.zip")
	if g.Name != "Golden Axe" || g.Genre != "Beat 'em up" || g.Desc != "" || g.Image != "./images/reg.png" {
		t.Errorf("local entry = %+v, want name and image from the registry, the rest untouched", g)
	}
	data, err := os.ReadFile(filepath.Join(romsFolder, "megadrive", "images", "reg.png"))
	if err != nil || string(data) != "registry-art" {
		t.Errorf("the registry cover art was not copied into the folder (err %v)", err)
	}
}

func TestSendFields_RomWithNoLocalEntry_CreatesIt(t *testing.T) {
	reg, romsFolder, registryFolder := writeTransferFixture(t)
	reg.Entries = append(reg.Entries, Entry{System: "megadrive", Game: gamelist.Game{Path: "./Fresh.zip", Name: "Fresh", Desc: "New."}})
	reg.index = nil
	if err := os.WriteFile(filepath.Join(romsFolder, "megadrive", "Fresh.zip"), []byte("rom"), 0o644); err != nil {
		t.Fatalf("write rom: %v", err)
	}

	_, added, err := SendFields(reg, romsFolder, registryFolder, "megadrive", "Fresh.zip", []string{FieldName, FieldDesc})

	if err != nil || !added {
		t.Fatalf("SendFields() added=%v err=%v, want the entry created", added, err)
	}
	if g, err := LocalGame(romsFolder, "megadrive", "Fresh.zip"); err != nil || g.Desc != "New." {
		t.Errorf("local entry = %+v, %v", g, err)
	}
}

func TestSendFields_NothingTickedOrUnknownToRegistry_WritesNothing(t *testing.T) {
	reg, romsFolder, registryFolder := writeTransferFixture(t)
	gamelistPath := filepath.Join(romsFolder, "megadrive", "gamelist.xml")
	before, _ := os.ReadFile(gamelistPath)

	if _, _, err := SendFields(reg, romsFolder, registryFolder, "megadrive", "Golden Axe.zip", nil); !errors.Is(err, ErrNothingToTransfer) {
		t.Errorf("nothing ticked: error = %v, want ErrNothingToTransfer", err)
	}
	if _, _, err := SendFields(reg, romsFolder, registryFolder, "megadrive", "Lonely.zip", []string{FieldName}); !errors.Is(err, ErrGameNotFound) {
		t.Errorf("unknown to registry: error = %v, want ErrGameNotFound", err)
	}
	after, _ := os.ReadFile(gamelistPath)
	if string(after) != string(before) {
		t.Error("a refused send rewrote the gamelist")
	}
}

func TestTransferFields_ListsTheTwelveValuesReadableByFieldValue(t *testing.T) {
	g := gamelist.Game{Name: "n", Desc: "d", Image: "i", Video: "v", Marquee: "m", Thumbnail: "t",
		Rating: "r", ReleaseDate: "rd", Developer: "dev", Publisher: "p", Genre: "g", Players: "pl"}

	ids := TransferFields()

	if len(ids) != 12 {
		t.Fatalf("TransferFields() = %v, want 12 identifiers", ids)
	}
	seen := map[string]bool{}
	for _, id := range ids {
		value := FieldValue(g, id)
		if value == "" || seen[value] {
			t.Errorf("FieldValue(%q) = %q, want a distinct non-empty value", id, value)
		}
		seen[value] = true
	}
	if FieldValue(g, "unknown") != "" {
		t.Error("FieldValue of an unknown identifier is not empty")
	}
}
