package registry

import (
	"errors"
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
	gamelistPath := filepath.Join(romsFolder, "megadrive", "gamelist.xml")
	if err := os.Chmod(gamelistPath, 0o444); err != nil {
		t.Fatalf("failed to make gamelist.xml read-only: %v", err)
	}
	t.Cleanup(func() { os.Chmod(gamelistPath, 0o644) })

	_, _, _, err := CompleteRomsFolder(reg, romsFolder, registryFolder, nil)

	if err == nil {
		t.Fatal("CompleteRomsFolder() error = nil, want error when the local gamelist.xml cannot be rewritten")
	}
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
	gamelistPath := filepath.Join(romsFolder, "megadrive", "gamelist.xml")
	if err := os.Chmod(gamelistPath, 0o444); err != nil {
		t.Fatalf("failed to make gamelist.xml read-only: %v", err)
	}
	t.Cleanup(func() { os.Chmod(gamelistPath, 0o644) })

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
