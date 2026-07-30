package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setRemoveConfig(t *testing.T) string {
	t.Helper()
	withTempConfig(t)
	registryFolder := t.TempDir()

	var out bytes.Buffer
	Execute([]string{"config", "set-registry", registryFolder}, &out)
	return registryFolder
}

func TestExecute_Remove_ExistingGame_DeletesEntryAndConfirms(t *testing.T) {
	registryFolder := setRemoveConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer

	code := Execute([]string{"remove", "megadrive", "Sonic.zip"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "removed") {
		t.Errorf("output = %q, want a confirmation mentioning the removal", out.String())
	}
	if _, err := os.Stat(filepath.Join(registryFolder, "megadrive", "Sonic.json")); err == nil {
		t.Error("Sonic.json still exists, want it deleted")
	}
}

func TestExecute_Remove_MediumThatCannotBeDeleted_ConfirmsTheRemovalAndNamesWhatIsLeft(t *testing.T) {
	registryFolder := setRemoveConfig(t)
	systemDir := filepath.Join(registryFolder, "megadrive")
	if err := os.MkdirAll(systemDir, 0o755); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	entry := `{"path":"./Sonic.zip","name":"Sonic","image":"images/Sonic.png"}`
	if err := os.WriteFile(filepath.Join(systemDir, "Sonic.json"), []byte(entry), 0o644); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	// A non-empty directory where the cover art is expected: os.Remove refuses it.
	blocked := filepath.Join(systemDir, "images", "Sonic.png")
	if err := os.MkdirAll(blocked, 0o755); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(blocked, "keeps it non-empty"), []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	var out bytes.Buffer

	code := Execute([]string{"remove", "megadrive", "Sonic.zip"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0: the game itself was removed (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "removed") {
		t.Errorf("output = %q, want it to confirm the removal", out.String())
	}
	if !strings.Contains(out.String(), "Sonic.png") {
		t.Errorf("output = %q, want it to name the file left behind", out.String())
	}
	if _, err := os.Stat(filepath.Join(systemDir, "Sonic.json")); err == nil {
		t.Error("Sonic.json still exists, want it deleted")
	}
}

func TestExecute_Remove_ExistingGame_RegeneratesTheSiteWithoutIt(t *testing.T) {
	registryFolder := setRemoveConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic the Hedgehog", "A classic platformer.")
	writeRegistryEntry(t, registryFolder, "snes", "./Mario.zip", "Super Mario World", "Another classic.")
	var out bytes.Buffer

	code := Execute([]string{"remove", "megadrive", "Sonic.zip"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	index, err := os.ReadFile(filepath.Join(registryFolder, "index.html"))
	if err != nil {
		t.Fatalf("expected the consultation site to be regenerated: %v", err)
	}
	if strings.Contains(string(index), "Sonic the Hedgehog") {
		t.Error("index.html still lists the removed game, want it regenerated without it")
	}
	if !strings.Contains(string(index), "Super Mario World") {
		t.Error("index.html no longer lists the game that was kept, want the rest of the registry intact")
	}
}

func TestExecute_Remove_SiteCannotBeRegenerated_ConfirmsTheRemovalAndSaysTheSiteIsStale(t *testing.T) {
	registryFolder := setRemoveConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic the Hedgehog", "A classic platformer.")
	// A directory where index.html belongs: the registry still saves, the site
	// cannot be written.
	if err := os.Mkdir(filepath.Join(registryFolder, "index.html"), 0o755); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	var out bytes.Buffer

	code := Execute([]string{"remove", "megadrive", "Sonic.zip"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0: the game was removed (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "removed Sonic.zip") {
		t.Errorf("output = %q, want it to confirm what was removed", out.String())
	}
	if !strings.Contains(out.String(), "consultation site") {
		t.Errorf("output = %q, want it to warn that the consultation site is stale", out.String())
	}
	if !strings.Contains(out.String(), "update") {
		t.Errorf("output = %q, want it to name 'update' as the way to rebuild the site", out.String())
	}
	if _, err := os.Stat(filepath.Join(registryFolder, "megadrive", "Sonic.json")); err == nil {
		t.Error("Sonic.json still exists, want it deleted even though the site could not be regenerated")
	}
}

func TestExecute_Remove_RegistryCannotBeWritten_WarnsAboutTheRegistryNotAStaleSite(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: a read-only file would still be writable")
	}
	registryFolder := setRemoveConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic the Hedgehog", "A classic platformer.")
	writeRegistryEntry(t, registryFolder, "snes", "./Mario.zip", "Super Mario World", "Another classic.")
	// The entry that survives the removal is still readable, but its system
	// folder can no longer be written into: registry.Save writes each game
	// file through a temporary file renamed over it, which needs to create
	// and rename within the folder — read-only permission bits on the game
	// file itself would not stop a rename, only the folder's own can.
	keptFolder := filepath.Join(registryFolder, "snes")
	if err := os.Chmod(keptFolder, 0o555); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	t.Cleanup(func() { os.Chmod(keptFolder, 0o755) })
	var out bytes.Buffer

	code := Execute([]string{"remove", "megadrive", "Sonic.zip"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0: the game was removed (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "removed Sonic.zip") {
		t.Errorf("output = %q, want it to confirm what was removed", out.String())
	}
	if !strings.Contains(out.String(), "registry") {
		t.Errorf("output = %q, want it to warn that the registry could not be written", out.String())
	}
	if strings.Contains(out.String(), "consultation site") {
		t.Errorf("output = %q, want the registry failure worded apart from a merely stale site", out.String())
	}
	if _, err := os.Stat(filepath.Join(registryFolder, "megadrive", "Sonic.json")); err == nil {
		t.Error("Sonic.json still exists, want it deleted even though the registry could not be rewritten")
	}
}

func TestExecute_Remove_MediumLeftBehindAndStaleSite_ReportsBothWarnings(t *testing.T) {
	registryFolder := setRemoveConfig(t)
	systemDir := filepath.Join(registryFolder, "megadrive")
	if err := os.MkdirAll(systemDir, 0o755); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	entry := `{"path":"./Sonic.zip","name":"Sonic the Hedgehog","image":"images/Sonic.png"}`
	if err := os.WriteFile(filepath.Join(systemDir, "Sonic.json"), []byte(entry), 0o644); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	blocked := filepath.Join(systemDir, "images", "Sonic.png")
	if err := os.MkdirAll(blocked, 0o755); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(blocked, "keeps it non-empty"), []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	if err := os.Mkdir(filepath.Join(registryFolder, "index.html"), 0o755); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	var out bytes.Buffer

	code := Execute([]string{"remove", "megadrive", "Sonic.zip"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0: the game was removed (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "Sonic.png") {
		t.Errorf("output = %q, want it to name the file left behind", out.String())
	}
	if !strings.Contains(out.String(), "consultation site") {
		t.Errorf("output = %q, want it to also warn that the consultation site is stale", out.String())
	}
}

func TestExecute_Remove_GameNotFound_DoesNotRegenerateTheSite(t *testing.T) {
	registryFolder := setRemoveConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic the Hedgehog", "A classic platformer.")
	var out bytes.Buffer

	Execute([]string{"remove", "megadrive", "Does Not Exist.zip"}, &out)

	if _, err := os.Stat(filepath.Join(registryFolder, "index.html")); err == nil {
		t.Error("index.html was written, want a failed removal to leave the registry folder untouched")
	}
}

func TestExecute_Remove_GameNotFound_ReturnsErrorCode(t *testing.T) {
	registryFolder := setRemoveConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer

	code := Execute([]string{"remove", "megadrive", "Does Not Exist.zip"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "no game found for system") {
		t.Errorf("output = %q, want it to mention no game found for the system", out.String())
	}
	if _, err := os.Stat(filepath.Join(registryFolder, "megadrive", "Sonic.json")); err != nil {
		t.Errorf("Sonic.json should be untouched: %v", err)
	}
}

func TestExecute_Remove_GameInSubfolder_FoundByFilenameAlone(t *testing.T) {
	registryFolder := setRemoveConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./sub/Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer

	code := Execute([]string{"remove", "megadrive", "Sonic.zip"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if _, err := os.Stat(filepath.Join(registryFolder, "megadrive", "Sonic.json")); err == nil {
		t.Error("Sonic.json still exists, want it deleted even though the original ROM was in a subfolder")
	}
}

func TestExecute_Remove_RegistryNotConfigured_ReturnsErrorCode(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"remove", "megadrive", "Sonic.zip"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "registry") {
		t.Errorf("output = %q, want it to mention the registry is not configured", out.String())
	}
}

func TestExecute_Remove_Help_PrintsRemoveSpecificUsageAndReturnsSuccess(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"remove", "--help"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "<system>") || !strings.Contains(out.String(), "<rom-filename>") {
		t.Errorf("output = %q, want it to describe the <system> <rom-filename> arguments", out.String())
	}
}

func TestExecute_Remove_Help_DoesNotRemoveAnything(t *testing.T) {
	registryFolder := setRemoveConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer

	Execute([]string{"remove", "--help"}, &out)

	if _, err := os.Stat(filepath.Join(registryFolder, "megadrive", "Sonic.json")); err != nil {
		t.Errorf("Sonic.json should be untouched by --help: %v", err)
	}
}

func TestExecute_Remove_MissingArguments_ReturnsErrorCode(t *testing.T) {
	setRemoveConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"remove", "megadrive"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "Usage") && !strings.Contains(out.String(), "error") {
		t.Errorf("output = %q, want a usage or error message", out.String())
	}
}
