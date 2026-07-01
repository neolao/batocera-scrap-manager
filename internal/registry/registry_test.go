package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
)

func TestLoad_FolderDoesNotExist_ReturnsEmptyRegistry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry")

	reg, err := Load(path)

	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if len(reg.Entries) != 0 {
		t.Errorf("Entries = %v, want empty", reg.Entries)
	}
}

func TestLoad_MalformedGameJSON_ReturnsError(t *testing.T) {
	path := t.TempDir()
	megadrive := filepath.Join(path, "megadrive")
	if err := os.MkdirAll(megadrive, 0o755); err != nil {
		t.Fatalf("mkdir megadrive: %v", err)
	}
	if err := os.WriteFile(filepath.Join(megadrive, "Sonic.json"), []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	_, err := Load(path)

	if err == nil {
		t.Fatal("Load() error = nil, want error for malformed JSON")
	}
}

func TestSave_WritesRegistryThatCanBeReloaded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "registry")
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic"}}}}

	if err := Save(path, reg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if len(got.Entries) != 1 || got.Entries[0].Game.Name != "Sonic" {
		t.Errorf("Entries = %v, want 1 entry named Sonic", got.Entries)
	}
}

func TestSave_WritesOneJSONFilePerGameInsideSystemFolder(t *testing.T) {
	path := t.TempDir()
	reg := &Registry{Entries: []Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic"}},
		{System: "megadrive", Game: gamelist.Game{Path: "./Golden Axe.zip", Name: "Golden Axe"}},
		{System: "mastersystem", Game: gamelist.Game{Path: "./Alex Kidd.zip", Name: "Alex Kidd"}},
	}}

	if err := Save(path, reg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	for _, want := range []string{
		filepath.Join(path, "megadrive", "Sonic.json"),
		filepath.Join(path, "megadrive", "Golden Axe.json"),
		filepath.Join(path, "mastersystem", "Alex Kidd.json"),
	} {
		if _, err := os.Stat(want); err != nil {
			t.Errorf("expected game file %s to exist: %v", want, err)
		}
	}

	if _, err := os.Stat(filepath.Join(path, "registry.json")); err == nil {
		t.Error("a single registry.json should not be created, want one JSON file per game instead")
	}
}

func TestImport_NewGames_AddsAllAndReturnsCount(t *testing.T) {
	reg := &Registry{}
	games := []gamelist.Game{{Path: "./a.zip", Name: "A"}, {Path: "./b.zip", Name: "B"}}

	added, updated, unchanged := reg.Import("megadrive", games)

	if added != 2 {
		t.Errorf("added = %d, want 2", added)
	}
	if updated != 0 {
		t.Errorf("updated = %d, want 0", updated)
	}
	if unchanged != 0 {
		t.Errorf("unchanged = %d, want 0", unchanged)
	}
	if len(reg.Entries) != 2 {
		t.Fatalf("Entries = %v, want 2", reg.Entries)
	}
}

func TestImport_SameGameReimported_NotDuplicated(t *testing.T) {
	reg := &Registry{}
	games := []gamelist.Game{{Path: "./a.zip", Name: "A"}}
	reg.Import("megadrive", games)

	added, updated, unchanged := reg.Import("megadrive", games)

	if added != 0 {
		t.Errorf("added = %d, want 0", added)
	}
	if updated != 0 {
		t.Errorf("updated = %d, want 0", updated)
	}
	if unchanged != 1 {
		t.Errorf("unchanged = %d, want 1", unchanged)
	}
	if len(reg.Entries) != 1 {
		t.Errorf("Entries = %v, want still 1 (no duplicate)", reg.Entries)
	}
}

func TestImport_SamePathDifferentSystem_TreatedAsDistinctEntries(t *testing.T) {
	reg := &Registry{}
	games := []gamelist.Game{{Path: "./a.zip", Name: "A"}}
	reg.Import("megadrive", games)

	added, updated, unchanged := reg.Import("mastersystem", games)

	if added != 1 {
		t.Errorf("added = %d, want 1 (same path but different system)", added)
	}
	if updated != 0 {
		t.Errorf("updated = %d, want 0", updated)
	}
	if unchanged != 0 {
		t.Errorf("unchanged = %d, want 0", unchanged)
	}
	if len(reg.Entries) != 2 {
		t.Errorf("Entries = %v, want 2 distinct entries", reg.Entries)
	}
}

func TestImport_ExistingGameWithChangedMetadata_UpdatesEntryAndReturnsCount(t *testing.T) {
	reg := &Registry{}
	reg.Import("megadrive", []gamelist.Game{{Path: "./a.zip", Name: "A", Desc: "old desc"}})

	added, updated, unchanged := reg.Import("megadrive", []gamelist.Game{{Path: "./a.zip", Name: "A", Desc: "new desc"}})

	if added != 0 {
		t.Errorf("added = %d, want 0", added)
	}
	if updated != 1 {
		t.Errorf("updated = %d, want 1", updated)
	}
	if unchanged != 0 {
		t.Errorf("unchanged = %d, want 0", unchanged)
	}
	if len(reg.Entries) != 1 {
		t.Fatalf("Entries = %v, want still 1 (no duplicate)", reg.Entries)
	}
	if reg.Entries[0].Game.Desc != "new desc" {
		t.Errorf("Entries[0].Game.Desc = %q, want %q", reg.Entries[0].Game.Desc, "new desc")
	}
}

func writeFixtureRomsFolder(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	megadrive := filepath.Join(root, "megadrive")
	if err := os.MkdirAll(megadrive, 0o755); err != nil {
		t.Fatalf("mkdir megadrive: %v", err)
	}
	megadriveXML := `<?xml version="1.0"?>
<gameList>
  <game><path>./Sonic.zip</path><name>Sonic</name></game>
  <game><path>./Golden Axe.zip</path><name>Golden Axe</name></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(megadrive, "gamelist.xml"), []byte(megadriveXML), 0o644); err != nil {
		t.Fatalf("write megadrive gamelist: %v", err)
	}

	mastersystem := filepath.Join(root, "mastersystem")
	if err := os.MkdirAll(mastersystem, 0o755); err != nil {
		t.Fatalf("mkdir mastersystem: %v", err)
	}
	mastersystemXML := `<?xml version="1.0"?>
<gameList>
  <game><path>./Alex Kidd.zip</path><name>Alex Kidd</name></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(mastersystem, "gamelist.xml"), []byte(mastersystemXML), 0o644); err != nil {
		t.Fatalf("write mastersystem gamelist: %v", err)
	}

	// A system folder with ROMs but no gamelist.xml yet — should be skipped silently.
	nes := filepath.Join(root, "nes")
	if err := os.MkdirAll(nes, 0o755); err != nil {
		t.Fatalf("mkdir nes: %v", err)
	}

	return root
}

func TestImportFromRomsFolder_NominalFixture_ImportsGamesGroupedBySystem(t *testing.T) {
	romsFolder := writeFixtureRomsFolder(t)
	registryFolder := t.TempDir()
	reg := &Registry{}

	added, updated, unchanged, err := ImportFromRomsFolder(reg, romsFolder, registryFolder)

	if err != nil {
		t.Fatalf("ImportFromRomsFolder() error = %v, want nil", err)
	}
	if added != 3 {
		t.Errorf("added = %d, want 3", added)
	}
	if updated != 0 {
		t.Errorf("updated = %d, want 0", updated)
	}
	if unchanged != 0 {
		t.Errorf("unchanged = %d, want 0", unchanged)
	}

	var sonic, alexKidd *Entry
	for i := range reg.Entries {
		e := &reg.Entries[i]
		if e.Game.Name == "Sonic" {
			sonic = e
		}
		if e.Game.Name == "Alex Kidd" {
			alexKidd = e
		}
	}
	if sonic == nil || sonic.System != "megadrive" {
		t.Errorf("Sonic entry = %v, want System = megadrive", sonic)
	}
	if alexKidd == nil || alexKidd.System != "mastersystem" {
		t.Errorf("Alex Kidd entry = %v, want System = mastersystem", alexKidd)
	}
}

func TestImportFromRomsFolder_ReimportSameFolder_NoDuplicates(t *testing.T) {
	romsFolder := writeFixtureRomsFolder(t)
	registryFolder := t.TempDir()
	reg := &Registry{}
	ImportFromRomsFolder(reg, romsFolder, registryFolder)

	added, updated, unchanged, err := ImportFromRomsFolder(reg, romsFolder, registryFolder)

	if err != nil {
		t.Fatalf("second ImportFromRomsFolder() error = %v, want nil", err)
	}
	if added != 0 {
		t.Errorf("added = %d, want 0 on reimport", added)
	}
	if updated != 0 {
		t.Errorf("updated = %d, want 0 on reimport", updated)
	}
	if unchanged != 3 {
		t.Errorf("unchanged = %d, want 3 on reimport", unchanged)
	}
	if len(reg.Entries) != 3 {
		t.Errorf("Entries = %v, want still 3 (no duplicates)", reg.Entries)
	}
}

func TestImportFromRomsFolder_ChangedGamelistMetadata_UpdatesEntry(t *testing.T) {
	romsFolder := writeFixtureRomsFolder(t)
	registryFolder := t.TempDir()
	reg := &Registry{}
	ImportFromRomsFolder(reg, romsFolder, registryFolder)

	changedXML := `<?xml version="1.0"?>
<gameList>
  <game><path>./Sonic.zip</path><name>Sonic</name><desc>Updated description</desc></game>
  <game><path>./Golden Axe.zip</path><name>Golden Axe</name></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"), []byte(changedXML), 0o644); err != nil {
		t.Fatalf("rewrite megadrive gamelist: %v", err)
	}

	added, updated, unchanged, err := ImportFromRomsFolder(reg, romsFolder, registryFolder)

	if err != nil {
		t.Fatalf("ImportFromRomsFolder() error = %v, want nil", err)
	}
	if added != 0 {
		t.Errorf("added = %d, want 0", added)
	}
	if updated != 1 {
		t.Errorf("updated = %d, want 1", updated)
	}
	if unchanged != 2 {
		t.Errorf("unchanged = %d, want 2", unchanged)
	}
}

func TestImportFromRomsFolder_RomsFolderDoesNotExist_ReturnsError(t *testing.T) {
	reg := &Registry{}

	_, _, _, err := ImportFromRomsFolder(reg, filepath.Join(t.TempDir(), "does-not-exist"), t.TempDir())

	if err == nil {
		t.Fatal("ImportFromRomsFolder() error = nil, want error for missing ROMs folder")
	}
}

func writeFixtureRomsFolderWithMedia(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	megadrive := filepath.Join(root, "megadrive")
	images := filepath.Join(megadrive, "images")
	videos := filepath.Join(megadrive, "videos")
	if err := os.MkdirAll(images, 0o755); err != nil {
		t.Fatalf("mkdir images: %v", err)
	}
	if err := os.MkdirAll(videos, 0o755); err != nil {
		t.Fatalf("mkdir videos: %v", err)
	}
	if err := os.WriteFile(filepath.Join(images, "Sonic.png"), []byte("fake-cover-art"), 0o644); err != nil {
		t.Fatalf("write cover art: %v", err)
	}
	if err := os.WriteFile(filepath.Join(videos, "Sonic.mp4"), []byte("fake-video"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	xml := `<?xml version="1.0"?>
<gameList>
  <game><path>./Sonic.zip</path><name>Sonic</name><image>./images/Sonic.png</image><video>./videos/Sonic.mp4</video></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(megadrive, "gamelist.xml"), []byte(xml), 0o644); err != nil {
		t.Fatalf("write megadrive gamelist: %v", err)
	}

	return root
}

func TestImportFromRomsFolder_GameWithMedia_CopiesMediaMirroringBatoceraLayout(t *testing.T) {
	romsFolder := writeFixtureRomsFolderWithMedia(t)
	registryFolder := t.TempDir()
	reg := &Registry{}

	_, _, _, err := ImportFromRomsFolder(reg, romsFolder, registryFolder)

	if err != nil {
		t.Fatalf("ImportFromRomsFolder() error = %v, want nil", err)
	}

	copiedImage := filepath.Join(registryFolder, "megadrive", "images", "Sonic.png")
	data, err := os.ReadFile(copiedImage)
	if err != nil {
		t.Fatalf("cover art not copied to %s: %v", copiedImage, err)
	}
	if string(data) != "fake-cover-art" {
		t.Errorf("copied cover art content = %q, want %q", data, "fake-cover-art")
	}

	copiedVideo := filepath.Join(registryFolder, "megadrive", "videos", "Sonic.mp4")
	if _, err := os.Stat(copiedVideo); err != nil {
		t.Errorf("video not copied to %s: %v", copiedVideo, err)
	}
}

func TestImportFromRomsFolder_ReimportUnchangedGame_DoesNotRecopyMedia(t *testing.T) {
	romsFolder := writeFixtureRomsFolderWithMedia(t)
	registryFolder := t.TempDir()
	reg := &Registry{}
	ImportFromRomsFolder(reg, romsFolder, registryFolder)

	copiedImage := filepath.Join(registryFolder, "megadrive", "images", "Sonic.png")
	if err := os.Remove(copiedImage); err != nil {
		t.Fatalf("failed to remove copied image fixture: %v", err)
	}

	added, updated, unchanged, err := ImportFromRomsFolder(reg, romsFolder, registryFolder)

	if err != nil {
		t.Fatalf("ImportFromRomsFolder() error = %v, want nil", err)
	}
	if added != 0 || updated != 0 || unchanged != 1 {
		t.Fatalf("added=%d updated=%d unchanged=%d, want 0/0/1", added, updated, unchanged)
	}
	if _, err := os.Stat(copiedImage); err == nil {
		t.Error("copied image was recreated for an unchanged game, want no recopy")
	}
}
