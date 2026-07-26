package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setProtectConfig points the CLI at a fresh registry folder and returns it.
func setProtectConfig(t *testing.T) string {
	t.Helper()
	withTempConfig(t)
	registryFolder := t.TempDir()

	var out bytes.Buffer
	Execute([]string{"config", "set-registry", registryFolder}, &out)
	return registryFolder
}

// storedMarks reads back the marks a game's file records, so a test asserts
// what was persisted rather than what stayed in memory.
func storedMarks(t *testing.T, registryFolder, system, gameFile string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(registryFolder, system, gameFile))
	if err != nil {
		t.Fatalf("read stored game: %v", err)
	}

	var stored struct {
		Name         string   `json:"name"`
		Desc         string   `json:"desc"`
		ManualFields []string `json:"manual_fields"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("parse stored game: %v", err)
	}
	if stored.Name == "" {
		t.Errorf("stored name is empty, want the game's own values left in place")
	}
	return stored.ManualFields
}

func TestExecute_Protect_ExistingGame_MarksEveryFieldAndConfirms(t *testing.T) {
	registryFolder := setProtectConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer

	code := Execute([]string{"protect", "megadrive", "Sonic.zip"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if got := out.String(); got != "protected Sonic.zip (system: megadrive)\n" {
		t.Errorf("output = %q, want the confirmation naming the game and its system", got)
	}
	if marks := storedMarks(t, registryFolder, "megadrive", "Sonic.json"); len(marks) != 8 {
		t.Errorf("manual_fields = %v, want the 8 editable fields", marks)
	}
}

func TestExecute_Protect_ExistingGame_ChangesNoStoredValue(t *testing.T) {
	registryFolder := setProtectConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")

	var out bytes.Buffer
	Execute([]string{"protect", "megadrive", "Sonic.zip"}, &out)

	data, err := os.ReadFile(filepath.Join(registryFolder, "megadrive", "Sonic.json"))
	if err != nil {
		t.Fatalf("read stored game: %v", err)
	}
	var stored map[string]any
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("parse stored game: %v", err)
	}
	if stored["name"] != "Sonic" {
		t.Errorf("name = %v, want %q: protecting states the values are right, it does not change them", stored["name"], "Sonic")
	}
	if stored["desc"] != "A classic platformer." {
		t.Errorf("desc = %v, want it untouched", stored["desc"])
	}
	if stored["path"] != "./Sonic.zip" {
		t.Errorf("path = %v, want it untouched", stored["path"])
	}
}

func TestExecute_Protect_GameInSubfolder_FoundByFilenameAlone(t *testing.T) {
	registryFolder := setProtectConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./sub/Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer

	code := Execute([]string{"protect", "megadrive", "Sonic.zip"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if marks := storedMarks(t, registryFolder, "megadrive", "Sonic.json"); len(marks) != 8 {
		t.Errorf("manual_fields = %v, want the game found by its filename alone", marks)
	}
}

func TestExecute_Protect_RunTwice_StaysProtectedOnceAndSucceeds(t *testing.T) {
	registryFolder := setProtectConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")

	var out bytes.Buffer
	Execute([]string{"protect", "megadrive", "Sonic.zip"}, &out)
	out.Reset()
	code := Execute([]string{"protect", "megadrive", "Sonic.zip"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 on a repeat (output: %s)", code, out.String())
	}
	if marks := storedMarks(t, registryFolder, "megadrive", "Sonic.json"); len(marks) != 8 {
		t.Errorf("manual_fields = %v, want 8 with no duplicate after two runs", marks)
	}
}

func TestExecute_Unprotect_ProtectedGame_ClearsTheMarksAndConfirms(t *testing.T) {
	registryFolder := setProtectConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer
	Execute([]string{"protect", "megadrive", "Sonic.zip"}, &out)
	out.Reset()

	code := Execute([]string{"unprotect", "megadrive", "Sonic.zip"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if got := out.String(); got != "unprotected Sonic.zip (system: megadrive)\n" {
		t.Errorf("output = %q, want the confirmation naming the game and its system", got)
	}
	if marks := storedMarks(t, registryFolder, "megadrive", "Sonic.json"); len(marks) != 0 {
		t.Errorf("manual_fields = %v, want none left", marks)
	}
}

func TestExecute_Unprotect_GameThatWasNotProtected_SucceedsAndChangesNothing(t *testing.T) {
	registryFolder := setProtectConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer

	code := Execute([]string{"unprotect", "megadrive", "Sonic.zip"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if marks := storedMarks(t, registryFolder, "megadrive", "Sonic.json"); len(marks) != 0 {
		t.Errorf("manual_fields = %v, want none", marks)
	}
}

func TestExecute_Protect_GameNotFound_ReturnsErrorCodeAndTouchesNothing(t *testing.T) {
	registryFolder := setProtectConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	before, err := os.ReadFile(filepath.Join(registryFolder, "megadrive", "Sonic.json"))
	if err != nil {
		t.Fatalf("read stored game: %v", err)
	}
	var out bytes.Buffer

	code := Execute([]string{"protect", "megadrive", "Does Not Exist.zip"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "no game found for system") {
		t.Errorf("output = %q, want it to mention no game found for the system", out.String())
	}
	after, err := os.ReadFile(filepath.Join(registryFolder, "megadrive", "Sonic.json"))
	if err != nil {
		t.Fatalf("read stored game: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("stored game changed from %s to %s, want it byte-identical", before, after)
	}
}

func TestExecute_Unprotect_GameNotFound_ReturnsErrorCode(t *testing.T) {
	registryFolder := setProtectConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer

	code := Execute([]string{"unprotect", "megadrive", "Does Not Exist.zip"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "no game found for system") {
		t.Errorf("output = %q, want it to mention no game found for the system", out.String())
	}
}

func TestExecute_Protect_UnknownSystem_ReturnsErrorCode(t *testing.T) {
	registryFolder := setProtectConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer

	code := Execute([]string{"protect", "nes", "Sonic.zip"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if marks := storedMarks(t, registryFolder, "megadrive", "Sonic.json"); len(marks) != 0 {
		t.Errorf("manual_fields = %v, want the megadrive game left alone", marks)
	}
}

func TestExecute_Protect_MissingArgument_PrintsErrorThenUsage(t *testing.T) {
	setProtectConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"protect", "megadrive"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	got := out.String()
	if !strings.HasPrefix(got, "error:") {
		t.Errorf("output = %q, want it to start with the error line", got)
	}
	if !strings.Contains(got, "Usage") {
		t.Errorf("output = %q, want the usage after the error", got)
	}
}

func TestExecute_Protect_ExtraArgument_IsRefusedRatherThanIgnored(t *testing.T) {
	registryFolder := setProtectConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer

	code := Execute([]string{"protect", "megadrive", "Sonic.zip", "extra"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1: a typo must not be silently swallowed", code)
	}
	if marks := storedMarks(t, registryFolder, "megadrive", "Sonic.json"); len(marks) != 0 {
		t.Errorf("manual_fields = %v, want nothing applied for a refused command", marks)
	}
}

func TestExecute_Protect_UnknownOption_IsNamedInTheError(t *testing.T) {
	setProtectConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"protect", "--off", "megadrive", "Sonic.zip"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "--off") {
		t.Errorf("output = %q, want the refused option named", out.String())
	}
}

func TestExecute_Protect_Help_PrintsUsageWithoutNeedingAConfiguredRegistry(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"protect", "--help"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	got := out.String()
	if !strings.Contains(got, "<system>") || !strings.Contains(got, "<rom-filename>") {
		t.Errorf("output = %q, want it to describe the <system> <rom-filename> arguments", got)
	}
	if strings.Contains(got, "error") {
		t.Errorf("output = %q, want no error: --help must not load the registry", got)
	}
}

func TestExecute_Unprotect_Help_WarnsItAlsoClearsHandMadeCorrections(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"unprotect", "--help"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "web UI") {
		t.Errorf("output = %q, want it to warn that corrections made in the web UI lose their mark too", out.String())
	}
}

func TestExecute_Protect_Help_ProtectsNothing(t *testing.T) {
	registryFolder := setProtectConfig(t)
	writeRegistryEntry(t, registryFolder, "megadrive", "./Sonic.zip", "Sonic", "A classic platformer.")
	var out bytes.Buffer

	Execute([]string{"protect", "--help"}, &out)

	if marks := storedMarks(t, registryFolder, "megadrive", "Sonic.json"); len(marks) != 0 {
		t.Errorf("manual_fields = %v, want --help to apply nothing", marks)
	}
}

func TestExecute_Protect_RegistryNotConfigured_ReturnsErrorCode(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"protect", "megadrive", "Sonic.zip"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "registry") {
		t.Errorf("output = %q, want it to mention the registry is not configured", out.String())
	}
}

func TestExecute_Help_ListsProtectAndUnprotect(t *testing.T) {
	var out bytes.Buffer

	Execute([]string{"--help"}, &out)

	got := out.String()
	if !strings.Contains(got, "protect") {
		t.Errorf("usage = %q, want it to list the protect command", got)
	}
	if !strings.Contains(got, "unprotect") {
		t.Errorf("usage = %q, want it to list the unprotect command", got)
	}
}
