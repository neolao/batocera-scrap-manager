package registry

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestSave_SystemDirectoryBlockedByFile_ReturnsError(t *testing.T) {
	path := t.TempDir()
	if err := os.WriteFile(filepath.Join(path, "megadrive"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic"}}}}

	if err := Save(path, reg); err == nil {
		t.Fatal("Save() error = nil, want error when a system subfolder is blocked by a file")
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

func TestImport_SameBaseNameDifferentExtension_SecondGameUpdatesFirstEntry(t *testing.T) {
	// Both ROMs would be stored under the same "Sonic.json" file (gameFileName
	// strips the extension), so they must be deduplicated as the same entry
	// by Import/indexOf too, or the second Save() would silently overwrite
	// the first game's file without either being reported as lost.
	reg := &Registry{}
	games := []gamelist.Game{
		{Path: "./Sonic.zip", Name: "Sonic (Cart)"},
		{Path: "./Sonic.iso", Name: "Sonic (Disc)"},
	}

	added, updated, unchanged := reg.Import("megadrive", games)

	if added != 1 || updated != 1 || unchanged != 0 {
		t.Errorf("added=%d updated=%d unchanged=%d, want 1,1,0", added, updated, unchanged)
	}
	if len(reg.Entries) != 1 {
		t.Fatalf("Entries = %v, want exactly 1 entry (avoiding a silent Save() file collision)", reg.Entries)
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

func TestImport_SameFilenameDifferentSubfolder_TreatedAsSameEntry(t *testing.T) {
	reg := &Registry{}
	reg.Import("megadrive", []gamelist.Game{{Path: "./sub1/Sonic.zip", Name: "Sonic"}})

	added, updated, unchanged := reg.Import("megadrive", []gamelist.Game{{Path: "./sub2/Sonic.zip", Name: "Sonic"}})

	if added != 0 {
		t.Errorf("added = %d, want 0 (same filename in a different subfolder matches the existing entry)", added)
	}
	if updated != 1 {
		t.Errorf("updated = %d, want 1 (the stored path changed)", updated)
	}
	if unchanged != 0 {
		t.Errorf("unchanged = %d, want 0", unchanged)
	}
	if len(reg.Entries) != 1 {
		t.Fatalf("Entries = %v, want 1 (deduplicated by filename, not full path)", reg.Entries)
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
  <game><path>./Sonic.zip</path><name>Sonic</name><desc>A blue hedgehog runs fast.</desc></game>
  <game><path>./Golden Axe.zip</path><name>Golden Axe</name><desc>A classic beat 'em up.</desc></game>
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
  <game><path>./Alex Kidd.zip</path><name>Alex Kidd</name><desc>A kid with miracle powers.</desc></game>
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

	added, updated, unchanged, err := ImportFromRomsFolder(reg, romsFolder, registryFolder, nil)

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
	ImportFromRomsFolder(reg, romsFolder, registryFolder, nil)

	added, updated, unchanged, err := ImportFromRomsFolder(reg, romsFolder, registryFolder, nil)

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
	ImportFromRomsFolder(reg, romsFolder, registryFolder, nil)

	changedXML := `<?xml version="1.0"?>
<gameList>
  <game><path>./Sonic.zip</path><name>Sonic</name><desc>Updated description</desc></game>
  <game><path>./Golden Axe.zip</path><name>Golden Axe</name><desc>A classic beat 'em up.</desc></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"), []byte(changedXML), 0o644); err != nil {
		t.Fatalf("rewrite megadrive gamelist: %v", err)
	}

	added, updated, unchanged, err := ImportFromRomsFolder(reg, romsFolder, registryFolder, nil)

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

func TestImportFromRomsFolder_NominalFixture_ReportsProgressPerGame(t *testing.T) {
	romsFolder := writeFixtureRomsFolder(t)
	registryFolder := t.TempDir()
	reg := &Registry{}

	var events []ProgressEvent
	_, _, _, err := ImportFromRomsFolder(reg, romsFolder, registryFolder, func(e ProgressEvent) {
		events = append(events, e)
	})

	if err != nil {
		t.Fatalf("ImportFromRomsFolder() error = %v, want nil", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d progress events, want 3 (one per game)", len(events))
	}

	var sonicEvent, alexKiddEvent *ProgressEvent
	for i := range events {
		if events[i].GameName == "Sonic" {
			sonicEvent = &events[i]
		}
		if events[i].GameName == "Alex Kidd" {
			alexKiddEvent = &events[i]
		}
	}
	if sonicEvent == nil {
		t.Fatal("no progress event reported for Sonic")
	}
	if sonicEvent.System != "megadrive" || sonicEvent.GameIndex != 1 || sonicEvent.GameCount != 2 {
		t.Errorf("Sonic event = %+v, want System=megadrive GameIndex=1 GameCount=2", *sonicEvent)
	}
	if alexKiddEvent == nil {
		t.Fatal("no progress event reported for Alex Kidd")
	}
	if alexKiddEvent.System != "mastersystem" || alexKiddEvent.GameIndex != 1 || alexKiddEvent.GameCount != 1 {
		t.Errorf("Alex Kidd event = %+v, want System=mastersystem GameIndex=1 GameCount=1", *alexKiddEvent)
	}
}

func TestImportFromRomsFolder_NilProgressCallback_DoesNotPanic(t *testing.T) {
	romsFolder := writeFixtureRomsFolder(t)
	registryFolder := t.TempDir()
	reg := &Registry{}

	_, _, _, err := ImportFromRomsFolder(reg, romsFolder, registryFolder, nil)

	if err != nil {
		t.Fatalf("ImportFromRomsFolder() error = %v, want nil", err)
	}
}

func TestImportFromRomsFolder_RomsFolderDoesNotExist_ReturnsError(t *testing.T) {
	reg := &Registry{}

	_, _, _, err := ImportFromRomsFolder(reg, filepath.Join(t.TempDir(), "does-not-exist"), t.TempDir(), nil)

	if err == nil {
		t.Fatal("ImportFromRomsFolder() error = nil, want error for missing ROMs folder")
	}
}

func TestImportFromRomsFolder_MalformedGamelistXML_ReturnsError(t *testing.T) {
	romsFolder := t.TempDir()
	megadrive := filepath.Join(romsFolder, "megadrive")
	if err := os.MkdirAll(megadrive, 0o755); err != nil {
		t.Fatalf("mkdir megadrive: %v", err)
	}
	if err := os.WriteFile(filepath.Join(megadrive, "gamelist.xml"), []byte("<gameList><game><name>oops</game></gameList>"), 0o644); err != nil {
		t.Fatalf("write malformed gamelist: %v", err)
	}
	reg := &Registry{}

	_, _, _, err := ImportFromRomsFolder(reg, romsFolder, t.TempDir(), nil)

	if err == nil {
		t.Fatal("ImportFromRomsFolder() error = nil, want error for malformed gamelist.xml")
	}
}

func writeMixedDataRomsFolder(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	megadrive := filepath.Join(root, "megadrive")
	images := filepath.Join(megadrive, "images")
	if err := os.MkdirAll(images, 0o755); err != nil {
		t.Fatalf("mkdir images: %v", err)
	}
	if err := os.WriteFile(filepath.Join(images, "GoldenAxe.png"), []byte("fake-cover-art"), 0o644); err != nil {
		t.Fatalf("write cover art: %v", err)
	}

	xml := `<?xml version="1.0"?>
<gameList>
  <game><path>./Sonic.zip</path><name>Sonic</name><desc>A blue hedgehog runs fast.</desc></game>
  <game><path>./Golden Axe.zip</path><name>Golden Axe</name><image>./images/GoldenAxe.png</image></game>
  <game><path>./Unknown.zip</path><name>Unknown</name></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(megadrive, "gamelist.xml"), []byte(xml), 0o644); err != nil {
		t.Fatalf("write megadrive gamelist: %v", err)
	}

	return root
}

func TestImportFromRomsFolder_GameWithNoDescriptionNorImage_NotAddedToRegistry(t *testing.T) {
	romsFolder := writeMixedDataRomsFolder(t)
	registryFolder := t.TempDir()
	reg := &Registry{}

	added, updated, unchanged, err := ImportFromRomsFolder(reg, romsFolder, registryFolder, nil)

	if err != nil {
		t.Fatalf("ImportFromRomsFolder() error = %v, want nil", err)
	}
	if added != 2 {
		t.Errorf("added = %d, want 2 (the game with neither description nor image is skipped)", added)
	}
	if updated != 0 || unchanged != 0 {
		t.Errorf("updated=%d unchanged=%d, want 0/0", updated, unchanged)
	}
	if len(reg.Entries) != 2 {
		t.Fatalf("Entries = %v, want 2 (Unknown should not be in the registry)", reg.Entries)
	}
	for _, e := range reg.Entries {
		if e.Game.Name == "Unknown" {
			t.Errorf("Unknown game was added to the registry, want it skipped for having no description and no image")
		}
	}
}

func TestImportFromRomsFolder_GameWithOnlyImageOrOnlyDescription_StillAdded(t *testing.T) {
	romsFolder := writeMixedDataRomsFolder(t)
	registryFolder := t.TempDir()
	reg := &Registry{}

	ImportFromRomsFolder(reg, romsFolder, registryFolder, nil)

	var sonic, goldenAxe *Entry
	for i := range reg.Entries {
		e := &reg.Entries[i]
		if e.Game.Name == "Sonic" {
			sonic = e
		}
		if e.Game.Name == "Golden Axe" {
			goldenAxe = e
		}
	}
	if sonic == nil {
		t.Error("Sonic (description only) was not added to the registry")
	}
	if goldenAxe == nil {
		t.Error("Golden Axe (image only) was not added to the registry")
	}
}

func TestImportFromRomsFolder_GameWithOnlyVideo_StillSkipped(t *testing.T) {
	// hasScrapedData deliberately only checks Desc and Image — a game with
	// only a video reference (no description, no image) carries no data
	// worth keeping in the registry either, and must still be skipped.
	romsFolder := t.TempDir()
	megadrive := filepath.Join(romsFolder, "megadrive")
	if err := os.MkdirAll(megadrive, 0o755); err != nil {
		t.Fatalf("mkdir megadrive: %v", err)
	}
	xml := `<?xml version="1.0"?>
<gameList>
  <game><path>./Sonic.zip</path><name>Sonic</name><video>./videos/Sonic.mp4</video></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(megadrive, "gamelist.xml"), []byte(xml), 0o644); err != nil {
		t.Fatalf("write megadrive gamelist: %v", err)
	}
	reg := &Registry{}

	added, updated, unchanged, err := ImportFromRomsFolder(reg, romsFolder, t.TempDir(), nil)

	if err != nil {
		t.Fatalf("ImportFromRomsFolder() error = %v, want nil", err)
	}
	if added != 0 || updated != 0 || unchanged != 0 {
		t.Errorf("added=%d updated=%d unchanged=%d, want 0/0/0 (video-only game should be skipped)", added, updated, unchanged)
	}
	if len(reg.Entries) != 0 {
		t.Errorf("Entries = %v, want empty", reg.Entries)
	}
}

func TestImportFromRomsFolder_GameWithNoDescriptionNorImage_ProducesNoProgressEventAndNoMediaFolder(t *testing.T) {
	romsFolder := writeMixedDataRomsFolder(t)
	registryFolder := t.TempDir()
	reg := &Registry{}

	var events []ProgressEvent
	_, _, _, err := ImportFromRomsFolder(reg, romsFolder, registryFolder, func(e ProgressEvent) {
		events = append(events, e)
	})

	if err != nil {
		t.Fatalf("ImportFromRomsFolder() error = %v, want nil", err)
	}
	for _, e := range events {
		if e.GameName == "Unknown" {
			t.Errorf("got a progress event for the skipped Unknown game: %+v", e)
		}
	}
}

func TestImportFromRomsFolder_OnlyGamesWithoutData_ReportsZeroSummary(t *testing.T) {
	root := t.TempDir()
	megadrive := filepath.Join(root, "megadrive")
	if err := os.MkdirAll(megadrive, 0o755); err != nil {
		t.Fatalf("mkdir megadrive: %v", err)
	}
	xml := `<?xml version="1.0"?>
<gameList>
  <game><path>./Unknown1.zip</path><name>Unknown1</name></game>
  <game><path>./Unknown2.zip</path><name>Unknown2</name></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(megadrive, "gamelist.xml"), []byte(xml), 0o644); err != nil {
		t.Fatalf("write megadrive gamelist: %v", err)
	}
	registryFolder := t.TempDir()
	reg := &Registry{}

	added, updated, unchanged, err := ImportFromRomsFolder(reg, root, registryFolder, nil)

	if err != nil {
		t.Fatalf("ImportFromRomsFolder() error = %v, want nil", err)
	}
	if added != 0 || updated != 0 || unchanged != 0 {
		t.Errorf("added=%d updated=%d unchanged=%d, want 0/0/0", added, updated, unchanged)
	}
	if len(reg.Entries) != 0 {
		t.Errorf("Entries = %v, want empty", reg.Entries)
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

	_, _, _, err := ImportFromRomsFolder(reg, romsFolder, registryFolder, nil)

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
	ImportFromRomsFolder(reg, romsFolder, registryFolder, nil)

	copiedImage := filepath.Join(registryFolder, "megadrive", "images", "Sonic.png")
	if err := os.Remove(copiedImage); err != nil {
		t.Fatalf("failed to remove copied image fixture: %v", err)
	}

	added, updated, unchanged, err := ImportFromRomsFolder(reg, romsFolder, registryFolder, nil)

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

func writeIncompleteRomsFolder(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	megadrive := filepath.Join(root, "megadrive")
	if err := os.MkdirAll(megadrive, 0o755); err != nil {
		t.Fatalf("mkdir megadrive: %v", err)
	}
	xml := `<?xml version="1.0"?>
<gameList>
  <game><path>./Sonic.zip</path><name>Sonic</name></game>
  <game><path>./Golden Axe.zip</path><name>Golden Axe</name><desc>Already complete</desc><genre>Beat 'em up</genre></game>
  <game><path>./Unknown.zip</path><name>Unknown</name></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(megadrive, "gamelist.xml"), []byte(xml), 0o644); err != nil {
		t.Fatalf("write megadrive gamelist: %v", err)
	}

	return root
}

func registryWithSonicAndGoldenAxe(t *testing.T, registryFolder string) *Registry {
	t.Helper()
	images := filepath.Join(registryFolder, "megadrive", "images")
	if err := os.MkdirAll(images, 0o755); err != nil {
		t.Fatalf("mkdir registry images: %v", err)
	}
	if err := os.WriteFile(filepath.Join(images, "Sonic.png"), []byte("fake-cover-art"), 0o644); err != nil {
		t.Fatalf("write registry cover art: %v", err)
	}

	return &Registry{Entries: []Entry{
		{System: "megadrive", Game: gamelist.Game{
			Path: "./Sonic.zip", Name: "Sonic", Desc: "A classic platformer.",
			Image: "./images/Sonic.png", Genre: "Platform",
		}},
		{System: "megadrive", Game: gamelist.Game{
			Path: "./Golden Axe.zip", Name: "Golden Axe", Desc: "A different desc, should not overwrite.",
		}},
	}}
}

func TestCompleteRomsFolder_IncompleteLocalEntry_FillsFromRegistryAndCopiesMedia(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	processed, completed, failed, err := CompleteRomsFolder(reg, romsFolder, registryFolder, nil)

	if err != nil {
		t.Fatalf("CompleteRomsFolder() error = %v, want nil", err)
	}
	if processed != 3 {
		t.Errorf("processed = %d, want 3", processed)
	}
	if completed != 1 {
		t.Errorf("completed = %d, want 1 (only Sonic had gaps filled)", completed)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}

	games, err := gamelist.ParseFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"))
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}
	var sonic gamelist.Game
	for _, g := range games {
		if g.Name == "Sonic" {
			sonic = g
		}
	}
	if sonic.Desc != "A classic platformer." {
		t.Errorf("Sonic.Desc = %q, want filled from registry", sonic.Desc)
	}
	if sonic.Image != "./images/Sonic.png" {
		t.Errorf("Sonic.Image = %q, want filled from registry", sonic.Image)
	}

	copiedImage := filepath.Join(romsFolder, "megadrive", "images", "Sonic.png")
	data, err := os.ReadFile(copiedImage)
	if err != nil {
		t.Fatalf("cover art not copied to %s: %v", copiedImage, err)
	}
	if string(data) != "fake-cover-art" {
		t.Errorf("copied cover art content = %q, want %q", data, "fake-cover-art")
	}
}

func TestCompleteRomsFolder_AlreadyCompleteLocalField_NotOverwrittenByRegistry(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	_, completed, _, err := CompleteRomsFolder(reg, romsFolder, registryFolder, nil)

	if err != nil {
		t.Fatalf("CompleteRomsFolder() error = %v, want nil", err)
	}
	if completed != 1 {
		t.Fatalf("completed = %d, want 1 (Golden Axe already complete, should not count)", completed)
	}

	games, err := gamelist.ParseFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"))
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}
	for _, g := range games {
		if g.Name == "Golden Axe" && g.Desc != "Already complete" {
			t.Errorf("Golden Axe.Desc = %q, want local value preserved (not overwritten by registry)", g.Desc)
		}
	}
}

func TestCompleteRomsFolder_NoMatchingRegistryEntry_LeavesGameUnchanged(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	_, _, _, err := CompleteRomsFolder(reg, romsFolder, registryFolder, nil)

	if err != nil {
		t.Fatalf("CompleteRomsFolder() error = %v, want nil", err)
	}

	games, err := gamelist.ParseFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"))
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}
	for _, g := range games {
		if g.Name == "Unknown" && g.Desc != "" {
			t.Errorf("Unknown.Desc = %q, want left empty (no registry match)", g.Desc)
		}
	}
}

func TestCompleteRomsFolder_ProgressCallback_OnlyReportsGamesActuallyChanged(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	var events []CompletionEvent
	_, _, _, err := CompleteRomsFolder(reg, romsFolder, registryFolder, func(e CompletionEvent) {
		events = append(events, e)
	})

	if err != nil {
		t.Fatalf("CompleteRomsFolder() error = %v, want nil", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d progress events, want 1 (only Sonic had metadata filled; Golden Axe already complete and Unknown has no registry match are not events)", len(events))
	}
	if events[0].System != "megadrive" || events[0].GameName != "Sonic" {
		t.Errorf("events[0] = %+v, want System=megadrive GameName=Sonic", events[0])
	}
}

func TestCompleteRomsFolder_MediaCopyFails_StillReportsProgressForTheAttemptedChange(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)
	megadrive := filepath.Join(romsFolder, "megadrive")
	if err := os.WriteFile(filepath.Join(megadrive, "images"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}

	var events []CompletionEvent
	_, _, _, err := CompleteRomsFolder(reg, romsFolder, registryFolder, func(e CompletionEvent) {
		events = append(events, e)
	})

	if err != nil {
		t.Fatalf("CompleteRomsFolder() error = %v, want nil", err)
	}
	if len(events) != 1 || events[0].GameName != "Sonic" {
		t.Fatalf("events = %+v, want 1 event for Sonic even though its media copy failed", events)
	}
}

func TestCompleteRomsFolder_RomsFolderDoesNotExist_ReturnsError(t *testing.T) {
	reg := &Registry{}

	_, _, _, err := CompleteRomsFolder(reg, filepath.Join(t.TempDir(), "does-not-exist"), t.TempDir(), nil)

	if err == nil {
		t.Fatal("CompleteRomsFolder() error = nil, want error for missing ROMs folder")
	}
}

func TestCompleteRomsFolder_MalformedGamelistXML_ReturnsError(t *testing.T) {
	romsFolder := t.TempDir()
	megadrive := filepath.Join(romsFolder, "megadrive")
	if err := os.MkdirAll(megadrive, 0o755); err != nil {
		t.Fatalf("mkdir megadrive: %v", err)
	}
	if err := os.WriteFile(filepath.Join(megadrive, "gamelist.xml"), []byte("<gameList><game><name>oops</game></gameList>"), 0o644); err != nil {
		t.Fatalf("write malformed gamelist: %v", err)
	}
	reg := &Registry{}

	_, _, _, err := CompleteRomsFolder(reg, romsFolder, t.TempDir(), nil)

	if err == nil {
		t.Fatal("CompleteRomsFolder() error = nil, want error for malformed gamelist.xml")
	}
}

func TestCompleteRomsFolder_LocalGamelistWriteFails_ReturnsError(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)
	makeSystemFolderReadOnly(t, filepath.Join(romsFolder, "megadrive"))

	_, _, _, err := CompleteRomsFolder(reg, romsFolder, registryFolder, nil)

	if err == nil {
		t.Fatal("CompleteRomsFolder() error = nil, want error when the local gamelist.xml cannot be rewritten")
	}
}

// makeSystemFolderReadOnly blocks the rewrite of a system's gamelist.xml. The
// folder is what has to refuse, not the file: the gamelist is written beside
// the old one and swapped in (see decisions/026), so a read-only gamelist.xml
// alone no longer stands in the way. Skipped as root, which ignores folder
// permissions.
func makeSystemFolderReadOnly(t *testing.T, folder string) {
	t.Helper()

	if os.Geteuid() == 0 {
		t.Skip("root ignores folder permissions, so the write cannot be made to fail this way")
	}
	if err := os.Chmod(folder, 0o555); err != nil {
		t.Fatalf("failed to make %q read-only: %v", folder, err)
	}
	t.Cleanup(func() { _ = os.Chmod(folder, 0o755) })
}

func TestCompleteRomsFolder_MediaDestinationBlockedByFile_CountsGameAsFailedAndContinues(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	// Block the destination "images" folder in the ROMs folder with a plain
	// file, so copying Sonic's cover art there fails.
	megadrive := filepath.Join(romsFolder, "megadrive")
	if err := os.WriteFile(filepath.Join(megadrive, "images"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}

	processed, completed, failed, err := CompleteRomsFolder(reg, romsFolder, registryFolder, nil)

	if err != nil {
		t.Fatalf("CompleteRomsFolder() error = %v, want nil (per-game failure, not fatal)", err)
	}
	if processed != 3 {
		t.Errorf("processed = %d, want 3", processed)
	}
	if failed != 1 {
		t.Errorf("failed = %d, want 1 (Sonic's media copy blocked)", failed)
	}
	if completed != 0 {
		t.Errorf("completed = %d, want 0", completed)
	}
}

// writeRegistryMedium creates a media file inside a system's folder of the
// registry, at the relative path a game's media reference points to.
func writeRegistryMedium(t *testing.T, registryFolder, system, relPath, content string) {
	t.Helper()
	fullPath := filepath.Join(registryFolder, system, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
}

func writeRegistryWithSonicAndMedia(t *testing.T) (registryFolder string, reg *Registry) {
	t.Helper()
	registryFolder = t.TempDir()
	images := filepath.Join(registryFolder, "megadrive", "images")
	if err := os.MkdirAll(images, 0o755); err != nil {
		t.Fatalf("mkdir registry images: %v", err)
	}
	if err := os.WriteFile(filepath.Join(images, "Sonic.png"), []byte("fake-cover-art"), 0o644); err != nil {
		t.Fatalf("write registry cover art: %v", err)
	}

	sonic := gamelist.Game{Path: "./Sonic.zip", Name: "Sonic", Desc: "A classic platformer.", Image: "./images/Sonic.png"}
	goldenAxe := gamelist.Game{Path: "./Golden Axe.zip", Name: "Golden Axe"}
	reg = &Registry{Entries: []Entry{
		{System: "megadrive", Game: sonic},
		{System: "megadrive", Game: goldenAxe},
	}}
	if err := Save(registryFolder, reg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}
	return registryFolder, reg
}

func TestRemove_ExistingGameWithMedia_DeletesJSONAndMediaAndEntry(t *testing.T) {
	registryFolder, reg := writeRegistryWithSonicAndMedia(t)

	err := Remove(reg, registryFolder, "megadrive", "Sonic.zip")

	if err != nil {
		t.Fatalf("Remove() error = %v, want nil", err)
	}
	if len(reg.Entries) != 1 || reg.Entries[0].Game.Name != "Golden Axe" {
		t.Errorf("Entries = %v, want only Golden Axe left", reg.Entries)
	}
	if _, statErr := os.Stat(filepath.Join(registryFolder, "megadrive", "Sonic.json")); statErr == nil {
		t.Error("Sonic.json still exists, want it deleted")
	}
	if _, statErr := os.Stat(filepath.Join(registryFolder, "megadrive", "images", "Sonic.png")); statErr == nil {
		t.Error("Sonic.png still exists, want it deleted")
	}
	if _, statErr := os.Stat(filepath.Join(registryFolder, "megadrive", "Golden Axe.json")); statErr != nil {
		t.Errorf("Golden Axe.json should still exist: %v", statErr)
	}
}

func TestRemove_GameWithoutMedia_DeletesJSONWithoutError(t *testing.T) {
	registryFolder, reg := writeRegistryWithSonicAndMedia(t)

	err := Remove(reg, registryFolder, "megadrive", "Golden Axe.zip")

	if err != nil {
		t.Fatalf("Remove() error = %v, want nil", err)
	}
	if len(reg.Entries) != 1 || reg.Entries[0].Game.Name != "Sonic" {
		t.Errorf("Entries = %v, want only Sonic left", reg.Entries)
	}
}

func TestRemove_GameNotFound_ReturnsErrGameNotFoundWithoutModifyingRegistry(t *testing.T) {
	registryFolder, reg := writeRegistryWithSonicAndMedia(t)

	err := Remove(reg, registryFolder, "megadrive", "Does Not Exist.zip")

	if !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("Remove() error = %v, want ErrGameNotFound", err)
	}
	if len(reg.Entries) != 2 {
		t.Errorf("Entries = %v, want unchanged (still 2)", reg.Entries)
	}
	if _, statErr := os.Stat(filepath.Join(registryFolder, "megadrive", "Sonic.json")); statErr != nil {
		t.Errorf("Sonic.json should be untouched: %v", statErr)
	}
}

func TestRemove_SameRomPathDifferentSystem_OnlyRemovesMatchingSystem(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &Registry{Entries: []Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./a.zip", Name: "A"}},
		{System: "mastersystem", Game: gamelist.Game{Path: "./a.zip", Name: "A"}},
	}}
	if err := Save(registryFolder, reg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	err := Remove(reg, registryFolder, "megadrive", "a.zip")

	if err != nil {
		t.Fatalf("Remove() error = %v, want nil", err)
	}
	if len(reg.Entries) != 1 || reg.Entries[0].System != "mastersystem" {
		t.Errorf("Entries = %v, want only the mastersystem entry left", reg.Entries)
	}
	if _, statErr := os.Stat(filepath.Join(registryFolder, "mastersystem", "a.json")); statErr != nil {
		t.Errorf("mastersystem/a.json should still exist: %v", statErr)
	}
}

func TestRemove_GameInSubfolder_FoundByFilenameAlone(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &Registry{Entries: []Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./sub/Sonic.zip", Name: "Sonic"}},
	}}
	if err := Save(registryFolder, reg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	err := Remove(reg, registryFolder, "megadrive", "Sonic.zip")

	if err != nil {
		t.Fatalf("Remove() error = %v, want nil (should be found by filename alone, regardless of its original subfolder)", err)
	}
	if len(reg.Entries) != 0 {
		t.Errorf("Entries = %v, want empty", reg.Entries)
	}
}

// writeRegistryWithDottedGameID builds a registry holding a game whose base
// name contains a dot — the case that tells apart addressing by ROM filename
// (which strips what looks like an extension) from addressing by game ID.
func writeRegistryWithDottedGameID(t *testing.T) (registryFolder string, reg *Registry) {
	t.Helper()
	registryFolder = t.TempDir()
	reg = &Registry{Entries: []Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Micro Machines v3.0.zip", Name: "Micro Machines v3.0"}},
		{System: "megadrive", Game: gamelist.Game{Path: "./Micro Machines v3.zip", Name: "Micro Machines v3"}},
	}}
	if err := Save(registryFolder, reg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}
	return registryFolder, reg
}

func TestRemoveByID_GameIDContainingADot_RemovesThatEntryAndNoOther(t *testing.T) {
	registryFolder, reg := writeRegistryWithDottedGameID(t)

	err := RemoveByID(reg, registryFolder, "megadrive", "Micro Machines v3.0")

	if err != nil {
		t.Fatalf("RemoveByID() error = %v, want nil", err)
	}
	if len(reg.Entries) != 1 || reg.Entries[0].Game.Name != "Micro Machines v3" {
		t.Errorf("Entries = %v, want only \"Micro Machines v3\" left", reg.Entries)
	}
	if _, statErr := os.Stat(filepath.Join(registryFolder, "megadrive", "Micro Machines v3.0.json")); statErr == nil {
		t.Error("Micro Machines v3.0.json still exists, want it deleted")
	}
	if _, statErr := os.Stat(filepath.Join(registryFolder, "megadrive", "Micro Machines v3.json")); statErr != nil {
		t.Errorf("Micro Machines v3.json should be untouched: %v", statErr)
	}
}

func TestRemoveByID_ExistingGame_DeletesEveryMediumAndTheEntry(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: gamelist.Game{
		Path:  "./Sonic.zip",
		Name:  "Sonic",
		Image: "images/Sonic.png", Video: "videos/Sonic.mp4",
		Marquee: "images/Sonic-marquee.png", Thumbnail: "images/Sonic-thumb.png",
	}}}}
	media := []string{"images/Sonic.png", "videos/Sonic.mp4", "images/Sonic-marquee.png", "images/Sonic-thumb.png"}
	for _, relPath := range media {
		writeRegistryMedium(t, registryFolder, "megadrive", relPath, "fake medium")
	}
	if err := Save(registryFolder, reg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	err := RemoveByID(reg, registryFolder, "megadrive", "Sonic")

	if err != nil {
		t.Fatalf("RemoveByID() error = %v, want nil", err)
	}
	if len(reg.Entries) != 0 {
		t.Errorf("Entries = %v, want empty", reg.Entries)
	}
	for _, relPath := range media {
		if _, statErr := os.Stat(filepath.Join(registryFolder, "megadrive", filepath.FromSlash(relPath))); statErr == nil {
			t.Errorf("%s still exists, want it deleted", relPath)
		}
	}
}

func TestRemoveByID_UnknownID_ReturnsErrGameNotFoundWithoutModifyingAnything(t *testing.T) {
	registryFolder, reg := writeRegistryWithSonicAndMedia(t)

	err := RemoveByID(reg, registryFolder, "megadrive", "Does Not Exist")

	if !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("RemoveByID() error = %v, want ErrGameNotFound", err)
	}
	if len(reg.Entries) != 2 {
		t.Errorf("Entries = %v, want unchanged (still 2)", reg.Entries)
	}
	if _, statErr := os.Stat(filepath.Join(registryFolder, "megadrive", "Sonic.json")); statErr != nil {
		t.Errorf("Sonic.json should be untouched: %v", statErr)
	}
}

func TestRemoveByID_KnownIDOfAnotherSystem_ReturnsErrGameNotFound(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &Registry{Entries: []Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic"}},
	}}
	if err := Save(registryFolder, reg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	err := RemoveByID(reg, registryFolder, "mastersystem", "Sonic")

	if !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("RemoveByID() error = %v, want ErrGameNotFound", err)
	}
	if _, statErr := os.Stat(filepath.Join(registryFolder, "megadrive", "Sonic.json")); statErr != nil {
		t.Errorf("the megadrive game should be untouched: %v", statErr)
	}
}

func TestRemove_RomFilenameWhoseBaseNameContainsADot_StillFindsTheEntry(t *testing.T) {
	registryFolder, reg := writeRegistryWithDottedGameID(t)

	err := Remove(reg, registryFolder, "megadrive", "Micro Machines v3.0.zip")

	if err != nil {
		t.Fatalf("Remove() error = %v, want nil", err)
	}
	if len(reg.Entries) != 1 || reg.Entries[0].Game.Name != "Micro Machines v3" {
		t.Errorf("Entries = %v, want only \"Micro Machines v3\" left", reg.Entries)
	}
}

func TestRemoveByID_MediumPointingOutsideTheRegistryFolder_LeavesThatFileAlone(t *testing.T) {
	base := t.TempDir()
	registryFolder := filepath.Join(base, "registry")
	outside := filepath.Join(base, "not-ours.png")
	if err := os.WriteFile(outside, []byte("someone else's file"), 0o644); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: gamelist.Game{
		Path: "./Sonic.zip", Name: "Sonic", Image: "../../not-ours.png",
	}}}}
	if err := Save(registryFolder, reg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	err := RemoveByID(reg, registryFolder, "megadrive", "Sonic")

	if _, statErr := os.Stat(outside); statErr != nil {
		t.Fatalf("a file outside the registry folder was deleted: %v", statErr)
	}
	if !errors.Is(err, ErrMediaLeftBehind) {
		t.Errorf("RemoveByID() error = %v, want it to report the medium as left behind", err)
	}
	if len(reg.Entries) != 0 {
		t.Errorf("Entries = %v, want the entry gone: its game file was deleted", reg.Entries)
	}
	if _, statErr := os.Stat(filepath.Join(registryFolder, "megadrive", "Sonic.json")); statErr == nil {
		t.Error("Sonic.json still exists, want it deleted")
	}
}

func TestRemoveByID_MediumThatCannotBeDeleted_StillDeletesTheOthersAndReportsIt(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: gamelist.Game{
		Path: "./Sonic.zip", Name: "Sonic",
		Image: "images/Sonic.png", Video: "videos/Sonic.mp4",
	}}}}
	// A non-empty directory where a medium is expected: os.Remove refuses it,
	// without depending on file permissions to stage the failure.
	blocked := filepath.Join(registryFolder, "megadrive", "images", "Sonic.png")
	if err := os.MkdirAll(blocked, 0o755); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(blocked, "keeps it non-empty"), []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	writeRegistryMedium(t, registryFolder, "megadrive", "videos/Sonic.mp4", "fake video")
	if err := Save(registryFolder, reg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	err := RemoveByID(reg, registryFolder, "megadrive", "Sonic")

	if !errors.Is(err, ErrMediaLeftBehind) {
		t.Fatalf("RemoveByID() error = %v, want it to report a medium left behind", err)
	}
	if !strings.Contains(err.Error(), "Sonic.png") {
		t.Errorf("error = %q, want it to name the file left behind", err)
	}
	if _, statErr := os.Stat(filepath.Join(registryFolder, "megadrive", "videos", "Sonic.mp4")); statErr == nil {
		t.Error("the video still exists: removal stopped at the first failure instead of going on")
	}
	if _, statErr := os.Stat(filepath.Join(registryFolder, "megadrive", "Sonic.json")); statErr == nil {
		t.Error("Sonic.json still exists, want the deletion to hold")
	}
	if len(reg.Entries) != 0 {
		t.Errorf("Entries = %v, want the entry gone: its game file was deleted", reg.Entries)
	}
}

func TestRemoveByID_GameFileThatCannotBeDeleted_ChangesNothing(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &Registry{Entries: []Entry{{System: "megadrive", Game: gamelist.Game{
		Path: "./Sonic.zip", Name: "Sonic", Image: "images/Sonic.png",
	}}}}
	writeRegistryMedium(t, registryFolder, "megadrive", "images/Sonic.png", "fake cover art")
	// A non-empty directory in place of the game file: os.Remove refuses it.
	blocked := filepath.Join(registryFolder, "megadrive", "Sonic.json")
	if err := os.MkdirAll(blocked, 0o755); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(blocked, "keeps it non-empty"), []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}

	err := RemoveByID(reg, registryFolder, "megadrive", "Sonic")

	if err == nil {
		t.Fatal("RemoveByID() error = nil, want the failure to delete the game file reported")
	}
	if errors.Is(err, ErrMediaLeftBehind) {
		t.Errorf("error = %v, want it reported as a failed deletion, not as media left behind", err)
	}
	if len(reg.Entries) != 1 {
		t.Errorf("Entries = %v, want the entry kept: nothing was deleted", reg.Entries)
	}
	if _, statErr := os.Stat(filepath.Join(registryFolder, "megadrive", "images", "Sonic.png")); statErr != nil {
		t.Errorf("the cover art was deleted although the game file survived: %v", statErr)
	}
}

func TestCompleteGame_IncompleteLocalEntry_FillsFromRegistryAndCopiesMedia(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	completed, failed, err := CompleteGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", nil)

	if err != nil {
		t.Fatalf("CompleteGame() error = %v, want nil", err)
	}
	if !completed {
		t.Error("completed = false, want true (Sonic had gaps filled)")
	}
	if failed {
		t.Error("failed = true, want false")
	}

	games, err := gamelist.ParseFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"))
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}
	var sonic, goldenAxe gamelist.Game
	for _, g := range games {
		switch g.Name {
		case "Sonic":
			sonic = g
		case "Golden Axe":
			goldenAxe = g
		}
	}
	if sonic.Desc != "A classic platformer." {
		t.Errorf("Sonic.Desc = %q, want filled from registry", sonic.Desc)
	}
	if goldenAxe.Desc != "Already complete" {
		t.Errorf("Golden Axe.Desc = %q, want left untouched (not the targeted game)", goldenAxe.Desc)
	}

	copiedImage := filepath.Join(romsFolder, "megadrive", "images", "Sonic.png")
	if _, err := os.Stat(copiedImage); err != nil {
		t.Errorf("cover art not copied to %s: %v", copiedImage, err)
	}
}

func TestCompleteGame_AlreadyCompleteLocalEntry_ReturnsNotCompletedNoError(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	completed, failed, err := CompleteGame(reg, romsFolder, registryFolder, "megadrive", "Golden Axe.zip", nil)

	if err != nil {
		t.Fatalf("CompleteGame() error = %v, want nil", err)
	}
	if completed {
		t.Error("completed = true, want false (Golden Axe already complete)")
	}
	if failed {
		t.Error("failed = true, want false")
	}

	games, err := gamelist.ParseFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"))
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}
	for _, g := range games {
		if g.Name == "Golden Axe" && g.Desc != "Already complete" {
			t.Errorf("Golden Axe.Desc = %q, want local value preserved", g.Desc)
		}
	}
}

func TestCompleteGame_NoMatchingRegistryEntry_ReturnsErrGameNotFound(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	_, _, err := CompleteGame(reg, romsFolder, registryFolder, "megadrive", "Unknown.zip", nil)

	if !errors.Is(err, ErrGameNotFound) {
		t.Errorf("CompleteGame() error = %v, want ErrGameNotFound (Unknown.zip has no registry entry)", err)
	}
}

func TestCompleteGame_RomNotInLocalGamelist_ReturnsErrGameNotFound(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	_, _, err := CompleteGame(reg, romsFolder, registryFolder, "megadrive", "Ghost.zip", nil)

	if !errors.Is(err, ErrGameNotFound) {
		t.Errorf("CompleteGame() error = %v, want ErrGameNotFound (Ghost.zip is not in the local gamelist.xml)", err)
	}
}

func TestCompleteGame_SystemHasNoLocalGamelist_ReturnsErrGameNotFound(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	_, _, err := CompleteGame(reg, romsFolder, registryFolder, "mastersystem", "Alex Kidd.zip", nil)

	if !errors.Is(err, ErrGameNotFound) {
		t.Errorf("CompleteGame() error = %v, want ErrGameNotFound (mastersystem has no local gamelist.xml)", err)
	}
}

func TestCompleteGame_ProgressCallback_FiresWithTheGamesLocalPosition(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	var events []CompletionEvent
	_, _, err := CompleteGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", func(e CompletionEvent) {
		events = append(events, e)
	})

	if err != nil {
		t.Fatalf("CompleteGame() error = %v, want nil", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d progress events, want 1", len(events))
	}
	if events[0].System != "megadrive" || events[0].GameName != "Sonic" || events[0].GameIndex != 1 || events[0].GameCount != 3 {
		t.Errorf("events[0] = %+v, want System=megadrive GameName=Sonic GameIndex=1 GameCount=3", events[0])
	}
}

func TestCompleteGame_MediaCopyFails_ReturnsFailedButStillFillsGamelist(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)
	megadrive := filepath.Join(romsFolder, "megadrive")
	if err := os.WriteFile(filepath.Join(megadrive, "images"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}

	completed, failed, err := CompleteGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", nil)

	if err != nil {
		t.Fatalf("CompleteGame() error = %v, want nil (per-game failure, not fatal)", err)
	}
	if completed {
		t.Error("completed = true, want false (media copy failed)")
	}
	if !failed {
		t.Error("failed = false, want true")
	}

	games, err := gamelist.ParseFile(filepath.Join(megadrive, "gamelist.xml"))
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}
	for _, g := range games {
		if g.Name == "Sonic" && g.Desc != "A classic platformer." {
			t.Errorf("Sonic.Desc = %q, want filled from registry despite the media copy failure", g.Desc)
		}
	}
}

func TestCompleteGame_LocalGamelistWriteFails_ReturnsError(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)
	makeSystemFolderReadOnly(t, filepath.Join(romsFolder, "megadrive"))

	_, _, err := CompleteGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", nil)

	if err == nil {
		t.Fatal("CompleteGame() error = nil, want error when the local gamelist.xml cannot be rewritten")
	}
}

func TestImportGame_NewGame_AddsEntryAndCopiesMedia(t *testing.T) {
	romsFolder := writeFixtureRomsFolderWithMedia(t)
	registryFolder := t.TempDir()
	reg := &Registry{}

	added, updated, unchanged, err := ImportGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", nil)

	if err != nil {
		t.Fatalf("ImportGame() error = %v, want nil", err)
	}
	if added != 1 || updated != 0 || unchanged != 0 {
		t.Fatalf("added=%d updated=%d unchanged=%d, want 1/0/0", added, updated, unchanged)
	}
	if len(reg.Entries) != 1 || reg.Entries[0].Game.Name != "Sonic" {
		t.Errorf("Entries = %v, want 1 entry named Sonic", reg.Entries)
	}

	copiedImage := filepath.Join(registryFolder, "megadrive", "images", "Sonic.png")
	if _, err := os.Stat(copiedImage); err != nil {
		t.Errorf("cover art not copied to %s: %v", copiedImage, err)
	}
}

func TestImportGame_OtherGameInSameSystem_NotImported(t *testing.T) {
	romsFolder := writeFixtureRomsFolder(t)
	registryFolder := t.TempDir()
	reg := &Registry{}

	_, _, _, err := ImportGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", nil)

	if err != nil {
		t.Fatalf("ImportGame() error = %v, want nil", err)
	}
	for _, e := range reg.Entries {
		if e.Game.Name == "Golden Axe" {
			t.Errorf("Entries = %v, want Golden Axe left untouched (not the targeted game)", reg.Entries)
		}
	}
}

func TestImportGame_ReimportUnchangedGame_ReturnsUnchanged(t *testing.T) {
	romsFolder := writeFixtureRomsFolder(t)
	registryFolder := t.TempDir()
	reg := &Registry{}
	ImportGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", nil)

	added, updated, unchanged, err := ImportGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", nil)

	if err != nil {
		t.Fatalf("ImportGame() error = %v, want nil", err)
	}
	if added != 0 || updated != 0 || unchanged != 1 {
		t.Fatalf("added=%d updated=%d unchanged=%d, want 0/0/1", added, updated, unchanged)
	}
	if len(reg.Entries) != 1 {
		t.Errorf("Entries = %v, want still 1 (no duplicate)", reg.Entries)
	}
}

func TestImportGame_ChangedLocalMetadata_ReturnsUpdated(t *testing.T) {
	romsFolder := writeFixtureRomsFolder(t)
	registryFolder := t.TempDir()
	reg := &Registry{}
	ImportGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", nil)

	changedXML := `<?xml version="1.0"?>
<gameList>
  <game><path>./Sonic.zip</path><name>Sonic</name><desc>Updated description</desc></game>
  <game><path>./Golden Axe.zip</path><name>Golden Axe</name><desc>A classic beat 'em up.</desc></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"), []byte(changedXML), 0o644); err != nil {
		t.Fatalf("rewrite megadrive gamelist: %v", err)
	}

	added, updated, unchanged, err := ImportGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", nil)

	if err != nil {
		t.Fatalf("ImportGame() error = %v, want nil", err)
	}
	if added != 0 || updated != 1 || unchanged != 0 {
		t.Fatalf("added=%d updated=%d unchanged=%d, want 0/1/0", added, updated, unchanged)
	}
}

func TestImportGame_NoScrapedData_SkippedNotAddedNoError(t *testing.T) {
	romsFolder := writeMixedDataRomsFolder(t)
	registryFolder := t.TempDir()
	reg := &Registry{}

	added, updated, unchanged, err := ImportGame(reg, romsFolder, registryFolder, "megadrive", "Unknown.zip", nil)

	if err != nil {
		t.Fatalf("ImportGame() error = %v, want nil", err)
	}
	if added != 0 || updated != 0 || unchanged != 0 {
		t.Fatalf("added=%d updated=%d unchanged=%d, want 0/0/0 (no scraped data, skipped)", added, updated, unchanged)
	}
	if len(reg.Entries) != 0 {
		t.Errorf("Entries = %v, want empty (Unknown has no scraped data)", reg.Entries)
	}
}

func TestImportGame_RomNotInLocalGamelist_ReturnsErrGameNotFound(t *testing.T) {
	romsFolder := writeFixtureRomsFolder(t)
	registryFolder := t.TempDir()
	reg := &Registry{}

	_, _, _, err := ImportGame(reg, romsFolder, registryFolder, "megadrive", "Ghost.zip", nil)

	if !errors.Is(err, ErrGameNotFound) {
		t.Errorf("ImportGame() error = %v, want ErrGameNotFound (Ghost.zip is not in the local gamelist.xml)", err)
	}
}

func TestImportGame_SystemHasNoLocalGamelist_ReturnsErrGameNotFound(t *testing.T) {
	romsFolder := writeFixtureRomsFolder(t)
	registryFolder := t.TempDir()
	reg := &Registry{}

	_, _, _, err := ImportGame(reg, romsFolder, registryFolder, "nes", "Anything.zip", nil)

	if !errors.Is(err, ErrGameNotFound) {
		t.Errorf("ImportGame() error = %v, want ErrGameNotFound (nes has no local gamelist.xml)", err)
	}
}

func TestImportGame_ProgressCallback_FiresWithTheGamesLocalPosition(t *testing.T) {
	romsFolder := writeFixtureRomsFolder(t)
	registryFolder := t.TempDir()
	reg := &Registry{}

	var events []ProgressEvent
	_, _, _, err := ImportGame(reg, romsFolder, registryFolder, "megadrive", "Golden Axe.zip", func(e ProgressEvent) {
		events = append(events, e)
	})

	if err != nil {
		t.Fatalf("ImportGame() error = %v, want nil", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d progress events, want 1", len(events))
	}
	if events[0].System != "megadrive" || events[0].GameName != "Golden Axe" || events[0].GameIndex != 2 || events[0].GameCount != 2 {
		t.Errorf("events[0] = %+v, want System=megadrive GameName=%q GameIndex=2 GameCount=2", events[0], "Golden Axe")
	}
}

func TestImportGame_MediaCopyFails_ReturnsError(t *testing.T) {
	romsFolder := writeFixtureRomsFolderWithMedia(t)
	registryFolder := t.TempDir()
	reg := &Registry{}
	if err := os.MkdirAll(filepath.Join(registryFolder, "megadrive"), 0o755); err != nil {
		t.Fatalf("mkdir registry system dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(registryFolder, "megadrive", "images"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}

	_, _, _, err := ImportGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", nil)

	if err == nil {
		t.Fatal("ImportGame() error = nil, want error when the media copy fails")
	}
}

func TestGameID_PathWithFolderAndExtension_ReturnsBareFileName(t *testing.T) {
	cases := map[string]string{
		"./roms/Sonic.zip":        "Sonic",
		"Sonic.zip":               "Sonic",
		"sub/folder/Sonic.zip":    "Sonic",
		"Sonic 3 & Knuckles.md":   "Sonic 3 & Knuckles",
		"Sonic":                   "Sonic",
		"Micro Machines v3.0.zip": "Micro Machines v3.0",
	}
	for path, want := range cases {
		if got := GameID(path); got != want {
			t.Errorf("GameID(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestFindByID_KnownGame_ReturnsItsEntry(t *testing.T) {
	reg := &Registry{Entries: []Entry{
		{System: "mastersystem", Game: gamelist.Game{Path: "./Alex.zip", Name: "Alex Kidd"}},
		{System: "megadrive", Game: gamelist.Game{Path: "./roms/Sonic.zip", Name: "Sonic"}},
	}}

	entry, found := reg.FindByID("megadrive", "Sonic")

	if !found {
		t.Fatal("FindByID() found = false, want true")
	}
	if entry.Game.Name != "Sonic" || entry.System != "megadrive" {
		t.Errorf("FindByID() entry = %+v, want the megadrive Sonic entry", entry)
	}
}

func TestFindByID_IDContainingADot_IsNotTruncatedLikeAPath(t *testing.T) {
	// A game ID is already extension-free: re-deriving an ID from it would
	// turn "Micro Machines v3.0" into "Micro Machines v3" and lose the game.
	reg := &Registry{Entries: []Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "Micro Machines v3.0.zip", Name: "Micro Machines"}},
	}}

	if _, found := reg.FindByID("megadrive", "Micro Machines v3.0"); !found {
		t.Error("FindByID() found = false, want true for an ID whose name contains a dot")
	}
	if _, found := reg.FindByID("megadrive", "Micro Machines v3"); found {
		t.Error("FindByID() found = true for a truncated ID, want false")
	}
}

func TestFindByID_UnknownGame_ReturnsNotFound(t *testing.T) {
	reg := &Registry{Entries: []Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic"}},
	}}

	if _, found := reg.FindByID("megadrive", "Does Not Exist"); found {
		t.Error("FindByID() found = true for an unknown game, want false")
	}
}

func TestFindByID_KnownGameUnderAnotherSystem_ReturnsNotFound(t *testing.T) {
	reg := &Registry{Entries: []Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic"}},
	}}

	if _, found := reg.FindByID("mastersystem", "Sonic"); found {
		t.Error("FindByID() found = true for a game belonging to another system, want false")
	}
}

func TestFindByID_EmptyRegistry_ReturnsNotFound(t *testing.T) {
	reg := &Registry{}

	if _, found := reg.FindByID("megadrive", "Sonic"); found {
		t.Error("FindByID() found = true on an empty registry, want false")
	}
}

func TestClone_EditingTheCopy_LeavesTheOriginalUntouched(t *testing.T) {
	// A caller that must not commit anything until the write to disk succeeded
	// edits a clone and only then swaps it in.
	reg := &Registry{Entries: []Entry{{
		System:       "megadrive",
		Game:         gamelist.Game{Path: "./Sonic.zip", Name: "Sonic"},
		ManualFields: []string{"genre"},
	}}}

	clone := reg.Clone()
	if err := UpdateMetadata(clone, "megadrive", "Sonic", Metadata{Name: "Sonic the Hedgehog"}, nil); err != nil {
		t.Fatalf("UpdateMetadata() error = %v, want nil", err)
	}

	if reg.Entries[0].Game.Name != "Sonic" {
		t.Errorf("original Name = %q, want it untouched by the edit on the clone", reg.Entries[0].Game.Name)
	}
	if strings.Join(reg.Entries[0].ManualFields, ",") != "genre" {
		t.Errorf("original ManualFields = %v, want them untouched", reg.Entries[0].ManualFields)
	}
	if clone.Entries[0].Game.Name != "Sonic the Hedgehog" {
		t.Errorf("clone Name = %q, want the edit applied there", clone.Entries[0].Game.Name)
	}
}

func TestClone_EmptyRegistry_ReturnsAnEmptyRegistry(t *testing.T) {
	clone := (&Registry{}).Clone()

	if clone == nil || len(clone.Entries) != 0 {
		t.Errorf("Clone() = %v, want an empty registry", clone)
	}
}

// writeRomsFolderWithUserData writes a ROMs folder whose game sheet holds, on
// top of the scraped metadata, what the user's own play sessions left there —
// the data no completion has any business erasing. desc is the description
// already present locally: empty leaves a gap for a completion to fill, while
// a non-empty one makes the game worth importing (a game with neither
// description nor cover art is deliberately skipped on import).
func writeRomsFolderWithUserData(t *testing.T, desc string) string {
	t.Helper()

	localDesc := ""
	if desc != "" {
		localDesc = "\n    <desc>" + desc + "</desc>"
	}

	root := t.TempDir()
	megadrive := filepath.Join(root, "megadrive")
	if err := os.MkdirAll(megadrive, 0o755); err != nil {
		t.Fatalf("mkdir megadrive: %v", err)
	}
	doc := `<?xml version="1.0"?>
<gameList>
  <game>
    <path>./Sonic.zip</path>
    <name>Sonic</name>` + localDesc + `
    <favorite>true</favorite>
    <playcount>17</playcount>
    <lastplayed>20260101T120000</lastplayed>
  </game>
</gameList>`
	if err := os.WriteFile(filepath.Join(megadrive, "gamelist.xml"), []byte(doc), 0o644); err != nil {
		t.Fatalf("write megadrive gamelist: %v", err)
	}
	return root
}

// registryKnowingSonic returns a registry holding a description for Sonic and
// nothing else, so completing a ROMs folder from it rewrites the game sheet
// without involving any media.
func registryKnowingSonic() *Registry {
	return &Registry{Entries: []Entry{
		{System: "megadrive", Game: gamelist.Game{
			Path: "./Sonic.zip",
			Name: "Sonic",
			Desc: "A blue hedgehog runs very fast.",
		}},
	}}
}

func TestCompleteRomsFolder_GameSheetHoldsTheUsersOwnData_LeavesItInPlace(t *testing.T) {
	romsFolder := writeRomsFolderWithUserData(t, "")
	gamelistPath := filepath.Join(romsFolder, "megadrive", "gamelist.xml")

	_, completed, _, err := CompleteRomsFolder(registryKnowingSonic(), romsFolder, t.TempDir(), nil)

	if err != nil {
		t.Fatalf("CompleteRomsFolder() error = %v, want nil", err)
	}
	if completed != 1 {
		t.Fatalf("completed = %d, want 1 — the description should have been filled", completed)
	}
	content, readErr := os.ReadFile(gamelistPath)
	if readErr != nil {
		t.Fatalf("reading the completed gamelist: %v", readErr)
	}
	got := string(content)
	if !strings.Contains(got, "<desc>A blue hedgehog runs very fast.</desc>") {
		t.Errorf("the completion did not fill the description\n--- file ---\n%s", got)
	}
	for _, want := range []string{"<favorite>true</favorite>", "<playcount>17</playcount>", "<lastplayed>20260101T120000</lastplayed>"} {
		if !strings.Contains(got, want) {
			t.Errorf("completing the ROMs folder erased %s — the user's play history is not ours to drop\n--- file ---\n%s", want, got)
		}
	}
}

func TestImportFromRomsFolder_GameSheetHoldsTheUsersOwnData_KeepsItOutOfTheRegistry(t *testing.T) {
	romsFolder := writeRomsFolderWithUserData(t, "Already scraped, so the import keeps it.")
	registryFolder := t.TempDir()
	reg := &Registry{}

	if _, _, _, err := ImportFromRomsFolder(reg, romsFolder, registryFolder, nil); err != nil {
		t.Fatalf("ImportFromRomsFolder() error = %v, want nil", err)
	}
	if err := Save(registryFolder, reg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	content, err := os.ReadFile(filepath.Join(registryFolder, "megadrive", EntryFileName(GameID("./Sonic.zip"))))
	if err != nil {
		t.Fatalf("reading the stored game: %v", err)
	}
	// Those values belong to the ROMs folder. The registry indexes scraped
	// metadata; it has no use for a play count and must not start carrying one.
	for _, unwanted := range []string{"favorite", "playcount", "lastplayed"} {
		if strings.Contains(string(content), unwanted) {
			t.Errorf("the stored game holds %q — the registry must keep ignoring it\n--- file ---\n%s", unwanted, string(content))
		}
	}
}

func TestReplaceGame_LocalValuePresent_OverwritesItWithTheRegistrys(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	replaced, failed, err := ReplaceGame(reg, romsFolder, registryFolder, "megadrive", "Golden Axe.zip", nil)

	if err != nil {
		t.Fatalf("ReplaceGame() error = %v, want nil", err)
	}
	if !replaced {
		t.Error("replaced = false, want true (the local description differs from the registry's)")
	}
	if failed {
		t.Error("failed = true, want false")
	}

	goldenAxe := localGame(t, romsFolder, "megadrive", "Golden Axe")
	if goldenAxe.Desc != "A different desc, should not overwrite." {
		t.Errorf("Golden Axe.Desc = %q, want the registry's value written over the local one", goldenAxe.Desc)
	}
}

func TestReplaceGame_RegistryFieldEmpty_LeavesTheLocalValueInPlace(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	if _, _, err := ReplaceGame(reg, romsFolder, registryFolder, "megadrive", "Golden Axe.zip", nil); err != nil {
		t.Fatalf("ReplaceGame() error = %v, want nil", err)
	}

	// The registry's Golden Axe entry holds no genre at all: not knowing a value
	// is no reason to make the user lose the one their own folder holds.
	goldenAxe := localGame(t, romsFolder, "megadrive", "Golden Axe")
	if goldenAxe.Genre != "Beat 'em up" {
		t.Errorf("Golden Axe.Genre = %q, want %q kept: the registry holds no genre for it", goldenAxe.Genre, "Beat 'em up")
	}
}

func TestReplaceGame_SameMediaReferenceOtherFile_WritesTheRegistrysFileOver(t *testing.T) {
	romsFolder := writeRomsFolderWithStaleCoverArt(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	replaced, failed, err := ReplaceGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", nil)

	if err != nil {
		t.Fatalf("ReplaceGame() error = %v, want nil", err)
	}
	if failed {
		t.Error("failed = true, want false")
	}
	// Every metadata field already matches the registry's: the cover art file is
	// the only thing that differs, and it is what makes this a real change.
	if !replaced {
		t.Error("replaced = false, want true (the folder's cover art file differs from the registry's)")
	}

	cover, err := os.ReadFile(filepath.Join(romsFolder, "megadrive", "images", "Sonic.png"))
	if err != nil {
		t.Fatalf("read the folder's cover art: %v", err)
	}
	if string(cover) != "fake-cover-art" {
		t.Errorf("the folder's cover art = %q, want the registry's %q written over it", string(cover), "fake-cover-art")
	}
}

func TestReplaceGame_SecondIdenticalRun_ReportsNothingChangedAndWritesNothing(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	if _, _, err := ReplaceGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", nil); err != nil {
		t.Fatalf("first ReplaceGame() error = %v, want nil", err)
	}

	// A read-only system folder is what proves nothing is written: rewriting the
	// gamelist.xml or the cover art would fail outright.
	makeSystemFolderReadOnly(t, filepath.Join(romsFolder, "megadrive"))

	replaced, failed, err := ReplaceGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", nil)

	if err != nil {
		t.Fatalf("second ReplaceGame() error = %v, want nil: nothing differs, so nothing should be written", err)
	}
	if replaced {
		t.Error("replaced = true, want false (the folder already holds exactly what the registry knows)")
	}
	if failed {
		t.Error("failed = true, want false")
	}
}

func TestReplaceGame_OtherGamesOfTheFolder_AreLeftUntouched(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	if _, _, err := ReplaceGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", nil); err != nil {
		t.Fatalf("ReplaceGame() error = %v, want nil", err)
	}

	sonic := localGame(t, romsFolder, "megadrive", "Sonic")
	if sonic.Desc != "A classic platformer." {
		t.Errorf("Sonic.Desc = %q, want the registry's value", sonic.Desc)
	}
	goldenAxe := localGame(t, romsFolder, "megadrive", "Golden Axe")
	if goldenAxe.Desc != "Already complete" {
		t.Errorf("Golden Axe.Desc = %q, want left untouched: it is not the targeted game", goldenAxe.Desc)
	}
	unknown := localGame(t, romsFolder, "megadrive", "Unknown")
	if unknown.Desc != "" {
		t.Errorf("Unknown.Desc = %q, want left untouched: it is not the targeted game", unknown.Desc)
	}
}

func TestReplaceGame_RomNotInLocalGamelist_ReturnsErrGameNotFound(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	_, _, err := ReplaceGame(reg, romsFolder, registryFolder, "megadrive", "Ghost.zip", nil)

	if !errors.Is(err, ErrGameNotFound) {
		t.Errorf("ReplaceGame() error = %v, want ErrGameNotFound (Ghost.zip is not in the local gamelist.xml)", err)
	}
}

func TestReplaceGame_NoMatchingRegistryEntry_ReturnsErrGameNotFound(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	_, _, err := ReplaceGame(reg, romsFolder, registryFolder, "megadrive", "Unknown.zip", nil)

	if !errors.Is(err, ErrGameNotFound) {
		t.Errorf("ReplaceGame() error = %v, want ErrGameNotFound (Unknown.zip has no registry entry)", err)
	}
}

func TestReplaceGame_SystemHasNoLocalGamelist_ReturnsErrGameNotFound(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	_, _, err := ReplaceGame(reg, romsFolder, registryFolder, "mastersystem", "Alex Kidd.zip", nil)

	if !errors.Is(err, ErrGameNotFound) {
		t.Errorf("ReplaceGame() error = %v, want ErrGameNotFound (mastersystem has no local gamelist.xml)", err)
	}
}

func TestReplaceGame_MediaCopyFails_ReturnsFailedButStillWritesTheGamelist(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)
	megadrive := filepath.Join(romsFolder, "megadrive")
	if err := os.WriteFile(filepath.Join(megadrive, "images"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}

	replaced, failed, err := ReplaceGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", nil)

	if err != nil {
		t.Fatalf("ReplaceGame() error = %v, want nil (a media failure is per-game, not fatal)", err)
	}
	if replaced {
		t.Error("replaced = true, want false (the cover art could not be written)")
	}
	if !failed {
		t.Error("failed = false, want true")
	}

	sonic := localGame(t, romsFolder, "megadrive", "Sonic")
	if sonic.Desc != "A classic platformer." {
		t.Errorf("Sonic.Desc = %q, want written from the registry despite the media failure", sonic.Desc)
	}
}

func TestReplaceGame_LocalGamelistWriteFails_ReturnsError(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)
	// The registry's Golden Axe entry holds no media, so nothing is copied and
	// rewriting the gamelist.xml is the only write left to fail.
	makeSystemFolderReadOnly(t, filepath.Join(romsFolder, "megadrive"))

	_, _, err := ReplaceGame(reg, romsFolder, registryFolder, "megadrive", "Golden Axe.zip", nil)

	if err == nil {
		t.Fatal("ReplaceGame() error = nil, want an error when the local gamelist.xml cannot be rewritten")
	}
}

func TestReplaceGame_ProgressCallback_FiresOnceWithTheGamesLocalPosition(t *testing.T) {
	romsFolder := writeIncompleteRomsFolder(t)
	registryFolder := t.TempDir()
	reg := registryWithSonicAndGoldenAxe(t, registryFolder)

	var events []CompletionEvent
	_, _, err := ReplaceGame(reg, romsFolder, registryFolder, "megadrive", "Sonic.zip", func(e CompletionEvent) {
		events = append(events, e)
	})

	if err != nil {
		t.Fatalf("ReplaceGame() error = %v, want nil", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d progress events, want 1", len(events))
	}
	if events[0].System != "megadrive" || events[0].GameName != "Sonic" || events[0].GameIndex != 1 || events[0].GameCount != 3 {
		t.Errorf("events[0] = %+v, want System=megadrive GameName=Sonic GameIndex=1 GameCount=3", events[0])
	}
}

// writeRomsFolderWithStaleCoverArt writes a ROMs folder whose Sonic entry
// already holds every value the registry knows — including the very same cover
// art reference — but whose cover art file on disk is another image. It is what
// tells a replacement that follows the reference from one that follows the
// bytes.
func writeRomsFolderWithStaleCoverArt(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	images := filepath.Join(root, "megadrive", "images")
	if err := os.MkdirAll(images, 0o755); err != nil {
		t.Fatalf("mkdir megadrive images: %v", err)
	}
	xml := `<?xml version="1.0"?>
<gameList>
  <game><path>./Sonic.zip</path><name>Sonic</name><desc>A classic platformer.</desc><image>./images/Sonic.png</image><genre>Platform</genre></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(root, "megadrive", "gamelist.xml"), []byte(xml), 0o644); err != nil {
		t.Fatalf("write megadrive gamelist: %v", err)
	}
	if err := os.WriteFile(filepath.Join(images, "Sonic.png"), []byte("a blurry cover the registry replaces"), 0o644); err != nil {
		t.Fatalf("write stale cover art: %v", err)
	}

	return root
}

// localGame reads one game out of a ROMs folder's own gamelist.xml, by name.
func localGame(t *testing.T, romsFolder, system, name string) gamelist.Game {
	t.Helper()

	games, err := gamelist.ParseFile(filepath.Join(romsFolder, system, "gamelist.xml"))
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}
	for _, g := range games {
		if g.Name == name {
			return g
		}
	}
	t.Fatalf("no game named %q in the %s gamelist.xml", name, system)
	return gamelist.Game{}
}
