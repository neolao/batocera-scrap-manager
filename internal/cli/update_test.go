package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeUpdateFixtureRomsFolder(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	megadrive := filepath.Join(root, "megadrive")
	if err := os.MkdirAll(megadrive, 0o755); err != nil {
		t.Fatalf("mkdir megadrive: %v", err)
	}
	xml := `<?xml version="1.0"?>
<gameList>
  <game><path>./Sonic.zip</path><name>Sonic</name><desc>A blue hedgehog runs fast.</desc></game>
  <game><path>./Golden Axe.zip</path><name>Golden Axe</name><desc>A classic beat 'em up.</desc></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(megadrive, "gamelist.xml"), []byte(xml), 0o644); err != nil {
		t.Fatalf("write megadrive gamelist: %v", err)
	}

	return root
}

func setUpdateConfig(t *testing.T, romsFolder string) string {
	t.Helper()
	withTempConfig(t)
	registryPath := filepath.Join(t.TempDir(), "registry.json")

	var out bytes.Buffer
	Execute([]string{"config", "set-registry", registryPath}, &out)
	if romsFolder != "" {
		Execute([]string{"config", "add-roms-folder", romsFolder}, &out)
	}
	return registryPath
}

func TestExecute_Update_NominalFixture_AddsEntriesAndPrintsSummary(t *testing.T) {
	romsFolder := writeUpdateFixtureRomsFolder(t)
	registryPath := setUpdateConfig(t, romsFolder)
	var out bytes.Buffer

	code := Execute([]string{"update"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "2 added") {
		t.Errorf("output = %q, want it to mention 2 added", out.String())
	}
	if !strings.Contains(out.String(), "0 updated") {
		t.Errorf("output = %q, want it to mention 0 updated", out.String())
	}
	if !strings.Contains(out.String(), "0 unchanged") {
		t.Errorf("output = %q, want it to mention 0 unchanged", out.String())
	}
	if _, err := os.Stat(registryPath); err != nil {
		t.Errorf("registry file not created: %v", err)
	}
}

func TestExecute_Update_GameWithNoDescriptionNorImage_NotCountedAsAdded(t *testing.T) {
	romsFolder := writeUpdateFixtureRomsFolder(t)
	xml := `<?xml version="1.0"?>
<gameList>
  <game><path>./Sonic.zip</path><name>Sonic</name><desc>A blue hedgehog runs fast.</desc></game>
  <game><path>./Golden Axe.zip</path><name>Golden Axe</name><desc>A classic beat 'em up.</desc></game>
  <game><path>./Unknown.zip</path><name>Unknown</name></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"), []byte(xml), 0o644); err != nil {
		t.Fatalf("rewrite megadrive gamelist: %v", err)
	}
	registryPath := setUpdateConfig(t, romsFolder)
	var out bytes.Buffer

	code := Execute([]string{"update"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "2 added") {
		t.Errorf("output = %q, want it to mention 2 added (Unknown, with no description or image, should not be counted)", out.String())
	}
	if strings.Contains(out.String(), "Unknown") {
		t.Errorf("output = %q, want no mention of the skipped Unknown game", out.String())
	}
	if _, err := os.Stat(filepath.Join(registryPath, "megadrive", "Unknown.json")); err == nil {
		t.Error("Unknown.json was written to the registry, want it skipped for having no description and no image")
	}
}

func TestExecute_Update_RerunWithoutChanges_ReportsUnchanged(t *testing.T) {
	romsFolder := writeUpdateFixtureRomsFolder(t)
	setUpdateConfig(t, romsFolder)
	var firstOut bytes.Buffer
	Execute([]string{"update"}, &firstOut)

	var out bytes.Buffer
	code := Execute([]string{"update"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "0 added") {
		t.Errorf("output = %q, want it to mention 0 added", out.String())
	}
	if !strings.Contains(out.String(), "2 unchanged") {
		t.Errorf("output = %q, want it to mention 2 unchanged", out.String())
	}
}

func TestExecute_Update_ChangedGamelistMetadata_ReportsUpdated(t *testing.T) {
	romsFolder := writeUpdateFixtureRomsFolder(t)
	setUpdateConfig(t, romsFolder)
	var firstOut bytes.Buffer
	Execute([]string{"update"}, &firstOut)

	changedXML := `<?xml version="1.0"?>
<gameList>
  <game><path>./Sonic.zip</path><name>Sonic</name><desc>Updated description</desc></game>
  <game><path>./Golden Axe.zip</path><name>Golden Axe</name><desc>A classic beat 'em up.</desc></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"), []byte(changedXML), 0o644); err != nil {
		t.Fatalf("rewrite megadrive gamelist: %v", err)
	}

	var out bytes.Buffer
	code := Execute([]string{"update"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "1 updated") {
		t.Errorf("output = %q, want it to mention 1 updated", out.String())
	}
	if !strings.Contains(out.String(), "1 unchanged") {
		t.Errorf("output = %q, want it to mention 1 unchanged", out.String())
	}
}

func TestExecute_Update_NominalFixture_PrintsProgressPerSystemAndGame(t *testing.T) {
	romsFolder := writeUpdateFixtureRomsFolder(t)
	setUpdateConfig(t, romsFolder)
	var out bytes.Buffer

	code := Execute([]string{"update"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	output := out.String()
	if !strings.Contains(output, "megadrive") {
		t.Errorf("output = %q, want it to mention the system being processed", output)
	}
	if !strings.Contains(output, "[1/2]") || !strings.Contains(output, "[2/2]") {
		t.Errorf("output = %q, want per-game progress counters", output)
	}
	if !strings.Contains(output, "Sonic") || !strings.Contains(output, "Golden Axe") {
		t.Errorf("output = %q, want game names in the progress output", output)
	}

	summaryIndex := strings.Index(output, "2 added")
	progressIndex := strings.Index(output, "[1/2]")
	if summaryIndex == -1 || progressIndex == -1 || progressIndex > summaryIndex {
		t.Errorf("output = %q, want progress lines to appear before the final summary", output)
	}
}

func TestExecute_Update_NoRomsFoldersConfigured_PrintsNoProgressLines(t *testing.T) {
	setUpdateConfig(t, "")
	var out bytes.Buffer

	code := Execute([]string{"update"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if strings.Contains(out.String(), "[") {
		t.Errorf("output = %q, want no per-game progress when no ROMs folder is configured", out.String())
	}
}

func TestExecute_Update_NoRomsFoldersConfigured_PrintsZeroSummary(t *testing.T) {
	setUpdateConfig(t, "")
	var out bytes.Buffer

	code := Execute([]string{"update"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "0 added") || !strings.Contains(out.String(), "0 updated") || !strings.Contains(out.String(), "0 unchanged") {
		t.Errorf("output = %q, want a zero summary", out.String())
	}
}

func TestExecute_Update_NominalFixture_GeneratesSiteWithGames(t *testing.T) {
	romsFolder := writeUpdateFixtureRomsFolder(t)
	registryPath := setUpdateConfig(t, romsFolder)
	var out bytes.Buffer

	code := Execute([]string{"update"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	html, err := os.ReadFile(filepath.Join(registryPath, "index.html"))
	if err != nil {
		t.Fatalf("site not generated: %v", err)
	}
	if !strings.Contains(string(html), "Sonic") || !strings.Contains(string(html), "megadrive") {
		t.Errorf("generated site = %q, want it to list the imported games", string(html))
	}
}

func TestExecute_Update_NoRomsFoldersConfigured_StillGeneratesSite(t *testing.T) {
	registryPath := setUpdateConfig(t, "")
	var out bytes.Buffer

	code := Execute([]string{"update"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	html, err := os.ReadFile(filepath.Join(registryPath, "index.html"))
	if err != nil {
		t.Fatalf("site not generated: %v", err)
	}
	if !strings.Contains(string(html), "No games") {
		t.Errorf("generated site = %q, want a no-games message", string(html))
	}
}

func TestExecute_Update_TargetedPath_AddsOnlyThatGame(t *testing.T) {
	romsFolder := writeUpdateFixtureRomsFolder(t)
	registryPath := setUpdateConfig(t, romsFolder)
	gamePath := filepath.Join(romsFolder, "megadrive", "Sonic.zip")
	var out bytes.Buffer

	code := Execute([]string{"update", gamePath}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "1 added") || !strings.Contains(out.String(), "0 updated") || !strings.Contains(out.String(), "0 unchanged") {
		t.Errorf("output = %q, want a summary mentioning 1 added, 0 updated, 0 unchanged", out.String())
	}
	if _, err := os.Stat(filepath.Join(registryPath, "megadrive", "Sonic.json")); err != nil {
		t.Errorf("Sonic.json not written to the registry: %v", err)
	}
	if _, err := os.Stat(filepath.Join(registryPath, "megadrive", "Golden Axe.json")); err == nil {
		t.Error("Golden Axe.json was written to the registry, want it left untouched (not the targeted game)")
	}
}

func TestExecute_Update_TargetedPath_AlreadyKnownUnchanged_PrintsUnchangedSummary(t *testing.T) {
	romsFolder := writeUpdateFixtureRomsFolder(t)
	setUpdateConfig(t, romsFolder)
	var firstOut bytes.Buffer
	Execute([]string{"update"}, &firstOut)

	gamePath := filepath.Join(romsFolder, "megadrive", "Sonic.zip")
	var out bytes.Buffer

	code := Execute([]string{"update", gamePath}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "0 added") || !strings.Contains(out.String(), "0 updated") || !strings.Contains(out.String(), "1 unchanged") {
		t.Errorf("output = %q, want a summary mentioning 0 added, 0 updated, 1 unchanged", out.String())
	}
}

func TestExecute_Update_TargetedPath_NoScrapedData_NotAddedNoError(t *testing.T) {
	romsFolder := writeUpdateFixtureRomsFolder(t)
	xml := `<?xml version="1.0"?>
<gameList>
  <game><path>./Sonic.zip</path><name>Sonic</name><desc>A blue hedgehog runs fast.</desc></game>
  <game><path>./Golden Axe.zip</path><name>Golden Axe</name><desc>A classic beat 'em up.</desc></game>
  <game><path>./Unknown.zip</path><name>Unknown</name></game>
</gameList>`
	if err := os.WriteFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"), []byte(xml), 0o644); err != nil {
		t.Fatalf("rewrite megadrive gamelist: %v", err)
	}
	registryPath := setUpdateConfig(t, romsFolder)
	gamePath := filepath.Join(romsFolder, "megadrive", "Unknown.zip")
	var out bytes.Buffer

	code := Execute([]string{"update", gamePath}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "0 added") || !strings.Contains(out.String(), "0 updated") || !strings.Contains(out.String(), "0 unchanged") {
		t.Errorf("output = %q, want a zero summary (no scraped data, not an error)", out.String())
	}
	if _, err := os.Stat(filepath.Join(registryPath, "megadrive", "Unknown.json")); err == nil {
		t.Error("Unknown.json was written to the registry, want it skipped for having no description and no image")
	}
}

func TestExecute_Update_TargetedPath_OutsideConfiguredRomsFolders_ReturnsErrorCode(t *testing.T) {
	romsFolder := writeUpdateFixtureRomsFolder(t)
	setUpdateConfig(t, romsFolder)
	outsidePath := filepath.Join(t.TempDir(), "megadrive", "Sonic.zip")
	var out bytes.Buffer

	code := Execute([]string{"update", outsidePath}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "error") {
		t.Errorf("output = %q, want it to mention an error", out.String())
	}
}

func TestExecute_Update_TargetedPath_NotInLocalGamelist_ReturnsErrorCode(t *testing.T) {
	romsFolder := writeUpdateFixtureRomsFolder(t)
	setUpdateConfig(t, romsFolder)
	gamePath := filepath.Join(romsFolder, "megadrive", "Ghost.zip")
	var out bytes.Buffer

	code := Execute([]string{"update", gamePath}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "error") {
		t.Errorf("output = %q, want it to mention an error", out.String())
	}
}

func TestExecute_Update_RegistryNotConfigured_ReturnsErrorCode(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"update"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "registry") {
		t.Errorf("output = %q, want it to mention the registry is not configured", out.String())
	}
}

func TestExecute_Update_RomsFolderMissingOnDisk_ReturnsErrorCode(t *testing.T) {
	missingFolder := filepath.Join(t.TempDir(), "does-not-exist")
	setUpdateConfig(t, missingFolder)
	var out bytes.Buffer

	code := Execute([]string{"update"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "error") {
		t.Errorf("output = %q, want it to mention an error", out.String())
	}
}
