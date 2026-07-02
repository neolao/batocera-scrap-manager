package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("BATOCERA_SCRAP_MANAGER_CONFIG", path)
	return path
}

func TestExecute_ConfigSetRegistry_PersistsPath(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"config", "set-registry", "/tmp/registry"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}

	var listOut bytes.Buffer
	Execute([]string{"config", "list"}, &listOut)
	if !strings.Contains(listOut.String(), "/tmp/registry") {
		t.Errorf("config list output = %q, want it to contain %q", listOut.String(), "/tmp/registry")
	}
}

func TestExecute_ConfigAddRomsFolder_PersistsAcrossInvocations(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"config", "add-roms-folder", "/tmp/roms1"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}

	var listOut bytes.Buffer
	Execute([]string{"config", "list"}, &listOut)
	if !strings.Contains(listOut.String(), "/tmp/roms1") {
		t.Errorf("config list output = %q, want it to contain %q", listOut.String(), "/tmp/roms1")
	}
}

func TestExecute_ConfigAddRomsFolder_DuplicateNotAddedTwice(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	Execute([]string{"config", "add-roms-folder", "/tmp/roms1"}, &out)
	Execute([]string{"config", "add-roms-folder", "/tmp/roms1"}, &out)

	var listOut bytes.Buffer
	Execute([]string{"config", "list"}, &listOut)
	if strings.Count(listOut.String(), "/tmp/roms1") != 1 {
		t.Errorf("config list output = %q, want /tmp/roms1 to appear exactly once", listOut.String())
	}
}

func TestExecute_ConfigList_NoConfigYet_PrintsNotSet(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"config", "list"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "not set") {
		t.Errorf("output = %q, want it to mention the registry is not set", out.String())
	}
}

func TestExecute_ConfigHelp_PrintsConfigSpecificUsageAndReturnsSuccess(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"config", "--help"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (output: %s)", code, out.String())
	}
	if !strings.Contains(out.String(), "set-registry") || !strings.Contains(out.String(), "add-roms-folder") || !strings.Contains(out.String(), "list") {
		t.Errorf("output = %q, want it to describe the config subcommands", out.String())
	}
}

func TestExecute_ConfigHelp_NoConfigFileYet_DoesNotCreateOne(t *testing.T) {
	configPath := withTempConfig(t)
	var out bytes.Buffer

	Execute([]string{"config", "--help"}, &out)

	if _, err := os.Stat(configPath); err == nil {
		t.Error("config file was created by --help, want no side effect")
	}
}

func TestExecute_ConfigNoSubcommand_ReturnsErrorCode(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"config"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestExecute_ConfigUnknownSubcommand_ReturnsErrorCode(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"config", "bogus"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "unknown") {
		t.Errorf("output = %q, want it to mention an unknown subcommand", out.String())
	}
}

func TestExecute_ConfigSetRegistryMissingPath_ReturnsErrorCode(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"config", "set-registry"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestExecute_ConfigAddRomsFolderMissingPath_ReturnsErrorCode(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	code := Execute([]string{"config", "add-roms-folder"}, &out)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}
