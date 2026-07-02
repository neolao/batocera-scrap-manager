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

func TestExecute_Remove_GameNotFound_ReturnsErrorCode(t *testing.T) {
	registryFolder := setRemoveConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer

	code := Execute([]string{"remove", "megadrive", "Does Not Exist.zip"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "error") {
		t.Errorf("output = %q, want it to mention an error", out.String())
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
