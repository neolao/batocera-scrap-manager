package registry

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
)

// writeStatusRomsFolder builds a ROMs folder exercising every combination
// RomsFolderStatus tells apart:
//   - Sonic.zip: a local entry with every field already filled (complete)
//   - Golden Axe.zip: a local entry missing its description and cover art,
//     both of which the registry already holds (incomplete, fillable)
//   - Unknown.zip: a local entry missing its description, which the registry
//     does not hold either (incomplete, not fillable)
//   - Orphan.zip: a ROM file with no local entry at all, known to the
//     registry (not listed, fillable)
//   - Ghost.zip: a ROM file with no local entry and no registry match either
//     (not listed, not fillable)
//   - images/ and gamelist.xml themselves: never counted as ROM files
//
// mastersystem is an empty folder: no ROM files, no gamelist.xml, nothing to
// report — it must not appear in the result at all.
func writeStatusRomsFolder(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	megadrive := filepath.Join(root, "megadrive")
	if err := os.MkdirAll(filepath.Join(megadrive, "images"), 0o755); err != nil {
		t.Fatalf("mkdir megadrive/images: %v", err)
	}
	for _, name := range []string{"Sonic.zip", "Golden Axe.zip", "Unknown.zip", "Orphan.zip", "Ghost.zip"} {
		if err := os.WriteFile(filepath.Join(megadrive, name), []byte("rom"), 0o644); err != nil {
			t.Fatalf("write rom %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(megadrive, "images", "not-a-rom.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write media file: %v", err)
	}
	xml := `<?xml version="1.0"?>
<gameList>
  <game><path>./Sonic.zip</path><name>Sonic</name><desc>A blue hedgehog runs fast.</desc>
    <image>./images/Sonic.png</image><video>./videos/Sonic.mp4</video>
    <marquee>./images/Sonic-marquee.png</marquee><thumbnail>./images/Sonic-thumb.png</thumbnail>
    <rating>0.8</rating><releasedate>19910101T000000</releasedate>
    <developer>Sega</developer><publisher>Sega</publisher><genre>Platform</genre><players>1</players></game>
  <game><path>./Golden Axe.zip</path><name>Golden Axe</name><genre>Beat 'em up</genre></game>
  <game><path>./Unknown.zip</path><name>Unknown</name></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(megadrive, "gamelist.xml"), []byte(xml), 0o644); err != nil {
		t.Fatalf("write gamelist: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(root, "mastersystem"), 0o755); err != nil {
		t.Fatalf("mkdir mastersystem: %v", err)
	}

	return root
}

func statusRegistry() *Registry {
	return &Registry{Entries: []Entry{
		{System: "megadrive", Game: gamelist.Game{
			Path: "./Golden Axe.zip", Name: "Golden Axe",
			Desc: "Two warriors against an evil sorcerer.", Image: "./images/Golden Axe.png",
		}},
		{System: "megadrive", Game: gamelist.Game{
			Path: "./Orphan.zip", Name: "Orphan", Desc: "Never scraped locally, but known here.",
		}},
	}}
}

func TestRomsFolderStatus_NominalFixture_ReportsEveryKindOfGap(t *testing.T) {
	romsFolder := writeStatusRomsFolder(t)
	reg := statusRegistry()

	status, err := RomsFolderStatus(reg, romsFolder)

	if err != nil {
		t.Fatalf("RomsFolderStatus() error = %v, want nil", err)
	}
	if len(status.Systems) != 1 {
		t.Fatalf("Systems = %v, want exactly megadrive (mastersystem is empty)", status.Systems)
	}
	sys := status.Systems[0]
	if sys.System != "megadrive" {
		t.Fatalf("System = %q, want megadrive", sys.System)
	}
	if sys.ROMCount != 5 {
		t.Errorf("ROMCount = %d, want 5 (gamelist.xml and images/ excluded)", sys.ROMCount)
	}
	if sys.CompleteCount != 1 {
		t.Errorf("CompleteCount = %d, want 1 (Sonic)", sys.CompleteCount)
	}
	if sys.IncompleteCount != 2 {
		t.Errorf("IncompleteCount = %d, want 2 (Golden Axe, Unknown)", sys.IncompleteCount)
	}
	if sys.NotListedCount != 2 {
		t.Errorf("NotListedCount = %d, want 2 (Orphan, Ghost)", sys.NotListedCount)
	}
	if sys.FillableCount != 2 {
		t.Errorf("FillableCount = %d, want 2 (Golden Axe and Orphan, from the registry)", sys.FillableCount)
	}
	if len(sys.Gaps) != 4 {
		t.Fatalf("Gaps = %v, want 4 entries (Sonic is complete and excluded)", sys.Gaps)
	}

	byFile := map[string]GameGap{}
	for _, g := range sys.Gaps {
		byFile[g.ROMFilename] = g
	}

	goldenAxe, ok := byFile["Golden Axe.zip"]
	if !ok {
		t.Fatal("Golden Axe.zip missing from Gaps")
	}
	if !goldenAxe.Listed {
		t.Error("Golden Axe.zip Listed = false, want true (it has a local entry)")
	}
	if !slices.Contains(goldenAxe.Missing, FieldDesc) || !slices.Contains(goldenAxe.Missing, string(MediumImage)) {
		t.Errorf("Golden Axe.zip Missing = %v, want desc and image", goldenAxe.Missing)
	}
	if !slices.Contains(goldenAxe.Fillable, FieldDesc) || !slices.Contains(goldenAxe.Fillable, string(MediumImage)) {
		t.Errorf("Golden Axe.zip Fillable = %v, want desc and image (the registry holds both)", goldenAxe.Fillable)
	}

	unknown, ok := byFile["Unknown.zip"]
	if !ok {
		t.Fatal("Unknown.zip missing from Gaps")
	}
	if !unknown.Listed {
		t.Error("Unknown.zip Listed = false, want true")
	}
	if len(unknown.Fillable) != 0 {
		t.Errorf("Unknown.zip Fillable = %v, want none (no matching registry entry)", unknown.Fillable)
	}

	orphan, ok := byFile["Orphan.zip"]
	if !ok {
		t.Fatal("Orphan.zip missing from Gaps")
	}
	if orphan.Listed {
		t.Error("Orphan.zip Listed = true, want false (no local gamelist entry)")
	}
	if orphan.Name != "Orphan" {
		t.Errorf("Orphan.zip Name = %q, want the registry's name since there is no local one", orphan.Name)
	}
	if !slices.Contains(orphan.Fillable, FieldDesc) {
		t.Errorf("Orphan.zip Fillable = %v, want desc (the registry holds it)", orphan.Fillable)
	}

	ghost, ok := byFile["Ghost.zip"]
	if !ok {
		t.Fatal("Ghost.zip missing from Gaps")
	}
	if ghost.Listed {
		t.Error("Ghost.zip Listed = true, want false")
	}
	if len(ghost.Fillable) != 0 {
		t.Errorf("Ghost.zip Fillable = %v, want none (unknown to the registry too)", ghost.Fillable)
	}
}

func TestRomsFolderStatus_SystemHasNoLocalGamelist_EveryROMIsNotListed(t *testing.T) {
	root := t.TempDir()
	nes := filepath.Join(root, "nes")
	if err := os.MkdirAll(nes, 0o755); err != nil {
		t.Fatalf("mkdir nes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nes, "Zelda.zip"), []byte("rom"), 0o644); err != nil {
		t.Fatalf("write rom: %v", err)
	}
	reg := &Registry{}

	status, err := RomsFolderStatus(reg, root)

	if err != nil {
		t.Fatalf("RomsFolderStatus() error = %v, want nil", err)
	}
	if len(status.Systems) != 1 {
		t.Fatalf("Systems = %v, want exactly nes", status.Systems)
	}
	sys := status.Systems[0]
	if sys.NotListedCount != 1 || len(sys.Gaps) != 1 {
		t.Fatalf("NotListedCount/Gaps = %d/%v, want 1/[Zelda.zip]", sys.NotListedCount, sys.Gaps)
	}
	if sys.Gaps[0].ROMFilename != "Zelda.zip" {
		t.Errorf("Gaps[0].ROMFilename = %q, want Zelda.zip", sys.Gaps[0].ROMFilename)
	}
}

func TestRomsFolderStatus_SystemFolderEmpty_IsExcludedFromTheReport(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatalf("mkdir empty: %v", err)
	}
	reg := &Registry{}

	status, err := RomsFolderStatus(reg, root)

	if err != nil {
		t.Fatalf("RomsFolderStatus() error = %v, want nil", err)
	}
	if len(status.Systems) != 0 {
		t.Errorf("Systems = %v, want none (nothing to report for an empty system folder)", status.Systems)
	}
}

func TestRomsFolderStatus_SystemGamelistIsMalformed_NamesTheProblemAndKeepsOtherSystems(t *testing.T) {
	root := t.TempDir()
	broken := filepath.Join(root, "broken")
	if err := os.MkdirAll(broken, 0o755); err != nil {
		t.Fatalf("mkdir broken: %v", err)
	}
	if err := os.WriteFile(filepath.Join(broken, "gamelist.xml"), []byte("<not valid xml"), 0o644); err != nil {
		t.Fatalf("write malformed gamelist: %v", err)
	}
	nes := filepath.Join(root, "nes")
	if err := os.MkdirAll(nes, 0o755); err != nil {
		t.Fatalf("mkdir nes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nes, "Zelda.zip"), []byte("rom"), 0o644); err != nil {
		t.Fatalf("write rom: %v", err)
	}
	reg := &Registry{}

	status, err := RomsFolderStatus(reg, root)

	if err != nil {
		t.Fatalf("RomsFolderStatus() error = %v, want nil (one bad system must not fail the whole report)", err)
	}
	var broke, other *SystemStatus
	for i := range status.Systems {
		switch status.Systems[i].System {
		case "broken":
			broke = &status.Systems[i]
		case "nes":
			other = &status.Systems[i]
		}
	}
	if broke == nil || broke.Problem == "" {
		t.Fatalf("broken system = %v, want a non-empty Problem", broke)
	}
	if other == nil || len(other.Gaps) != 1 {
		t.Fatalf("nes system = %v, want its own gap still reported", other)
	}
}

func TestRomsFolderStatus_FolderDoesNotExist_ReturnsError(t *testing.T) {
	reg := &Registry{}

	_, err := RomsFolderStatus(reg, filepath.Join(t.TempDir(), "does-not-exist"))

	if err == nil {
		t.Fatal("RomsFolderStatus() error = nil, want an error for a missing folder")
	}
}

func TestRomsFolderStatus_MostProblematicSystemFirst(t *testing.T) {
	root := t.TempDir()
	// "quiet" has one gap, "noisy" has two: noisy must sort first.
	quiet := filepath.Join(root, "quiet")
	noisy := filepath.Join(root, "noisy")
	if err := os.MkdirAll(quiet, 0o755); err != nil {
		t.Fatalf("mkdir quiet: %v", err)
	}
	if err := os.MkdirAll(noisy, 0o755); err != nil {
		t.Fatalf("mkdir noisy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(quiet, "A.zip"), []byte("rom"), 0o644); err != nil {
		t.Fatalf("write rom: %v", err)
	}
	if err := os.WriteFile(filepath.Join(noisy, "B.zip"), []byte("rom"), 0o644); err != nil {
		t.Fatalf("write rom: %v", err)
	}
	if err := os.WriteFile(filepath.Join(noisy, "C.zip"), []byte("rom"), 0o644); err != nil {
		t.Fatalf("write rom: %v", err)
	}
	reg := &Registry{}

	status, err := RomsFolderStatus(reg, root)

	if err != nil {
		t.Fatalf("RomsFolderStatus() error = %v, want nil", err)
	}
	if len(status.Systems) != 2 || status.Systems[0].System != "noisy" {
		t.Fatalf("Systems = %v, want noisy first (2 gaps against quiet's 1)", status.Systems)
	}
}
