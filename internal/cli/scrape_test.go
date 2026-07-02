package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeScrapeFixtureRomsFolder(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	megadrive := filepath.Join(root, "megadrive")
	if err := os.MkdirAll(megadrive, 0o755); err != nil {
		t.Fatalf("mkdir megadrive: %v", err)
	}
	xml := `<?xml version="1.0"?>
<gameList>
  <game><path>./Sonic.zip</path><name>Sonic</name></game>
  <game><path>./Golden Axe.zip</path><name>Golden Axe</name><desc>Already complete</desc></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(megadrive, "gamelist.xml"), []byte(xml), 0o644); err != nil {
		t.Fatalf("write megadrive gamelist: %v", err)
	}

	return root
}

func setScrapeConfig(t *testing.T, romsFolder string) string {
	t.Helper()
	withTempConfig(t)
	registryFolder := t.TempDir()

	var out bytes.Buffer
	Execute([]string{"config", "set-registry", registryFolder}, &out)
	if romsFolder != "" {
		Execute([]string{"config", "add-roms-folder", romsFolder}, &out)
	}
	return registryFolder
}

func writeRegistryEntry(t *testing.T, registryFolder, system, romPath, name, desc string) {
	t.Helper()
	systemDir := filepath.Join(registryFolder, system)
	if err := os.MkdirAll(systemDir, 0o755); err != nil {
		t.Fatalf("mkdir registry system dir: %v", err)
	}
	base := strings.TrimSuffix(filepath.Base(romPath), filepath.Ext(romPath))
	json := `{"path":"` + romPath + `","name":"` + name + `","desc":"` + desc + `"}`
	if err := os.WriteFile(filepath.Join(systemDir, base+".json"), []byte(json), 0o644); err != nil {
		t.Fatalf("write registry entry: %v", err)
	}
}

func TestExecute_Scrape_NominalFixture_CompletesGameAndPrintsSummary(t *testing.T) {
	romsFolder := writeScrapeFixtureRomsFolder(t)
	registryFolder := setScrapeConfig(t, romsFolder)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer

	code := Execute([]string{"scrape"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "2 processed") {
		t.Errorf("output = %q, want it to mention 2 processed", out.String())
	}
	if !strings.Contains(out.String(), "1 completed") {
		t.Errorf("output = %q, want it to mention 1 completed", out.String())
	}
	if !strings.Contains(out.String(), "0 failed") {
		t.Errorf("output = %q, want it to mention 0 failed", out.String())
	}

	games, err := os.ReadFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"))
	if err != nil {
		t.Fatalf("read gamelist.xml: %v", err)
	}
	if !strings.Contains(string(games), "A classic platformer.") {
		t.Errorf("gamelist.xml = %q, want Sonic's description filled in", games)
	}
}

func TestExecute_Scrape_NominalFixture_LiveOutputOnlyMentionsChangedGame(t *testing.T) {
	romsFolder := writeScrapeFixtureRomsFolder(t)
	registryFolder := setScrapeConfig(t, romsFolder)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	writeRegistryEntry(t, registryFolder, "megadrive", "./Golden Axe.zip", "Golden Axe", "A different desc, should not be shown")
	var out bytes.Buffer

	code := Execute([]string{"scrape"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "Sonic") {
		t.Errorf("output = %q, want it to mention Sonic (actually changed)", out.String())
	}
	if strings.Contains(out.String(), "Golden Axe") {
		t.Errorf("output = %q, want it to NOT mention Golden Axe (already complete, no change)", out.String())
	}
}

func TestExecute_Scrape_AlreadyCompleteEntry_NotOverwrittenAndNotCounted(t *testing.T) {
	romsFolder := writeScrapeFixtureRomsFolder(t)
	registryFolder := setScrapeConfig(t, romsFolder)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Golden Axe.zip", "Golden Axe", "A different desc")
	var out bytes.Buffer

	code := Execute([]string{"scrape"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "0 completed") {
		t.Errorf("output = %q, want it to mention 0 completed", out.String())
	}

	games, err := os.ReadFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"))
	if err != nil {
		t.Fatalf("read gamelist.xml: %v", err)
	}
	if !strings.Contains(string(games), "Already complete") {
		t.Errorf("gamelist.xml = %q, want Golden Axe's local description preserved", games)
	}
}

func TestExecute_Scrape_SingleRomsFolder_LiveOutputMentionsItsPath(t *testing.T) {
	romsFolder := writeScrapeFixtureRomsFolder(t)
	registryFolder := setScrapeConfig(t, romsFolder)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer

	code := Execute([]string{"scrape"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), romsFolder) {
		t.Errorf("output = %q, want it to mention the roms folder %q", out.String(), romsFolder)
	}
}

func TestExecute_Scrape_MultipleRomsFolders_EachGameLineIdentifiesItsOwnFolder(t *testing.T) {
	romsFolder1 := writeScrapeFixtureRomsFolder(t)
	romsFolder2 := writeScrapeFixtureRomsFolder(t)
	registryFolder := setScrapeConfig(t, romsFolder1)
	var setup bytes.Buffer
	if code := Execute([]string{"config", "add-roms-folder", romsFolder2}, &setup); code != 0 {
		t.Fatalf("failed to add second roms folder: %s", setup.String())
	}
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer

	code := Execute([]string{"scrape"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}

	sawFolder1, sawFolder2 := false, false
	for _, line := range strings.Split(out.String(), "\n") {
		if !strings.Contains(line, "Sonic") {
			continue
		}
		if strings.Contains(line, romsFolder1) {
			sawFolder1 = true
		}
		if strings.Contains(line, romsFolder2) {
			sawFolder2 = true
		}
	}
	if !sawFolder1 || !sawFolder2 {
		t.Errorf("output = %q, want a Sonic line naming each roms folder (%q and %q)", out.String(), romsFolder1, romsFolder2)
	}
}

func TestExecute_Scrape_NoRomsFoldersConfigured_PrintsNoFolderLine(t *testing.T) {
	setScrapeConfig(t, "")
	var out bytes.Buffer

	code := Execute([]string{"scrape"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if strings.TrimSpace(out.String()) != "0 processed, 0 completed, 0 failed" {
		t.Errorf("output = %q, want only the zero summary, no folder line", out.String())
	}
}

func TestExecute_Scrape_NoRomsFoldersConfigured_PrintsZeroSummary(t *testing.T) {
	setScrapeConfig(t, "")
	var out bytes.Buffer

	code := Execute([]string{"scrape"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "0 processed") || !strings.Contains(out.String(), "0 completed") || !strings.Contains(out.String(), "0 failed") {
		t.Errorf("output = %q, want a zero summary", out.String())
	}
}

func TestExecute_Scrape_RegistryNotConfigured_ReturnsErrorCode(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"scrape"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "registry") {
		t.Errorf("output = %q, want it to mention the registry is not configured", out.String())
	}
}

func TestExecute_Scrape_TargetedPath_CompletesOnlyThatGame(t *testing.T) {
	romsFolder := writeScrapeFixtureRomsFolder(t)
	registryFolder := setScrapeConfig(t, romsFolder)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	gamePath := filepath.Join(romsFolder, "megadrive", "Sonic.zip")
	var out bytes.Buffer

	code := Execute([]string{"scrape", gamePath}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if strings.TrimSpace(out.String()) == "" || !strings.Contains(out.String(), "1 processed") || !strings.Contains(out.String(), "1 completed") {
		t.Errorf("output = %q, want a summary mentioning 1 processed, 1 completed", out.String())
	}

	games, err := os.ReadFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"))
	if err != nil {
		t.Fatalf("read gamelist.xml: %v", err)
	}
	if !strings.Contains(string(games), "A classic platformer.") {
		t.Errorf("gamelist.xml = %q, want Sonic's description filled in", games)
	}
}

func TestExecute_Scrape_TargetedPath_AlreadyComplete_PrintsZeroCompletedSummary(t *testing.T) {
	romsFolder := writeScrapeFixtureRomsFolder(t)
	registryFolder := setScrapeConfig(t, romsFolder)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Golden Axe.zip", "Golden Axe", "A different desc")
	gamePath := filepath.Join(romsFolder, "megadrive", "Golden Axe.zip")
	var out bytes.Buffer

	code := Execute([]string{"scrape", gamePath}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "1 processed") || !strings.Contains(out.String(), "0 completed") {
		t.Errorf("output = %q, want a summary mentioning 1 processed, 0 completed", out.String())
	}
}

func TestExecute_Scrape_TargetedPath_OutsideConfiguredRomsFolders_ReturnsErrorCode(t *testing.T) {
	romsFolder := writeScrapeFixtureRomsFolder(t)
	setScrapeConfig(t, romsFolder)
	outsidePath := filepath.Join(t.TempDir(), "megadrive", "Sonic.zip")
	var out bytes.Buffer

	code := Execute([]string{"scrape", outsidePath}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "error") {
		t.Errorf("output = %q, want it to mention an error", out.String())
	}
}

func TestExecute_Scrape_TargetedPath_NoMatchingRegistryEntry_ReturnsErrorCode(t *testing.T) {
	romsFolder := writeScrapeFixtureRomsFolder(t)
	setScrapeConfig(t, romsFolder)
	gamePath := filepath.Join(romsFolder, "megadrive", "Sonic.zip")
	var out bytes.Buffer

	code := Execute([]string{"scrape", gamePath}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "error") {
		t.Errorf("output = %q, want it to mention an error", out.String())
	}
}

func TestExecute_Scrape_RomsFolderMissingOnDisk_ReturnsErrorCode(t *testing.T) {
	missingFolder := filepath.Join(t.TempDir(), "does-not-exist")
	setScrapeConfig(t, missingFolder)
	var out bytes.Buffer

	code := Execute([]string{"scrape"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "error") {
		t.Errorf("output = %q, want it to mention an error", out.String())
	}
}
