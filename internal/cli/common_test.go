package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/config"
	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

func TestLoadConfigAndRegistry_RegistryNotConfigured_ReturnsNotOk(t *testing.T) {
	withTempConfig(t)
	var out bytes.Buffer

	_, _, ok := loadConfigAndRegistry(&out)

	if ok {
		t.Error("ok = true, want false when the registry folder is not configured")
	}
	if !strings.Contains(out.String(), "registry not configured") {
		t.Errorf("output = %q, want it to mention the registry is not configured", out.String())
	}
}

func TestLoadConfigAndRegistry_MalformedConfigFile_ReturnsNotOk(t *testing.T) {
	path := withTempConfig(t)
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}
	var out bytes.Buffer

	_, _, ok := loadConfigAndRegistry(&out)

	if ok {
		t.Error("ok = true, want false when the config file is malformed")
	}
	if !strings.Contains(out.String(), "error") {
		t.Errorf("output = %q, want it to mention an error", out.String())
	}
}

func TestLoadConfigAndRegistry_ValidConfig_ReturnsConfigAndRegistry(t *testing.T) {
	withTempConfig(t)
	registryFolder := t.TempDir()
	var setupOut bytes.Buffer
	Execute([]string{"config", "set-registry", registryFolder}, &setupOut)
	var out bytes.Buffer

	cfg, reg, ok := loadConfigAndRegistry(&out)

	if !ok {
		t.Fatalf("ok = false, want true (output: %s)", out.String())
	}
	if cfg.RegistryFolder != registryFolder {
		t.Errorf("cfg.RegistryFolder = %q, want %q", cfg.RegistryFolder, registryFolder)
	}
	if reg == nil {
		t.Error("reg = nil, want a loaded (possibly empty) registry")
	}
}

func TestSaveAndGenerateSite_SystemFolderBlockedByFile_ReturnsFalse(t *testing.T) {
	registryFolder := t.TempDir()
	if err := os.WriteFile(filepath.Join(registryFolder, "megadrive"), []byte("blocker"), 0o644); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}
	cfg := config.Config{RegistryFolder: registryFolder}
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic"}},
	}}
	var out bytes.Buffer

	ok := saveAndGenerateSite(cfg, reg, &out)

	if ok {
		t.Error("ok = true, want false when registry.Save fails")
	}
	if !strings.Contains(out.String(), "error") {
		t.Errorf("output = %q, want it to mention an error", out.String())
	}
}

func TestSaveAndGenerateSite_IndexHTMLBlockedByDirectory_ReturnsFalse(t *testing.T) {
	registryFolder := t.TempDir()
	if err := os.MkdirAll(filepath.Join(registryFolder, "index.html"), 0o755); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}
	cfg := config.Config{RegistryFolder: registryFolder}
	reg := &registry.Registry{}
	var out bytes.Buffer

	ok := saveAndGenerateSite(cfg, reg, &out)

	if ok {
		t.Error("ok = true, want false when site.Generate fails")
	}
	if !strings.Contains(out.String(), "error") {
		t.Errorf("output = %q, want it to mention an error", out.String())
	}
}

func TestSaveAndGenerateSite_Success_WritesRegistryAndSite(t *testing.T) {
	registryFolder := t.TempDir()
	cfg := config.Config{RegistryFolder: registryFolder}
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic"}},
	}}
	var out bytes.Buffer

	ok := saveAndGenerateSite(cfg, reg, &out)

	if !ok {
		t.Fatalf("ok = false, want true (output: %s)", out.String())
	}
	if _, err := os.Stat(filepath.Join(registryFolder, "megadrive", "Sonic.json")); err != nil {
		t.Errorf("registry JSON not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(registryFolder, "index.html")); err != nil {
		t.Errorf("site index.html not written: %v", err)
	}
}
