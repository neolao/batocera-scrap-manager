package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FileDoesNotExist_ReturnsEmptyConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	cfg, err := Load(path)

	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.RegistryPath != "" {
		t.Errorf("RegistryPath = %q, want empty", cfg.RegistryPath)
	}
	if len(cfg.RomsFolders) != 0 {
		t.Errorf("RomsFolders = %v, want empty", cfg.RomsFolders)
	}
}

func TestLoad_MalformedJSON_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}

	_, err := Load(path)

	if err == nil {
		t.Fatal("Load() error = nil, want error for malformed JSON")
	}
}

func TestSave_WritesConfigThatCanBeReloaded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := Config{RegistryPath: "/registry", RomsFolders: []string{"/roms1", "/roms2"}}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if got.RegistryPath != cfg.RegistryPath {
		t.Errorf("RegistryPath = %q, want %q", got.RegistryPath, cfg.RegistryPath)
	}
	if len(got.RomsFolders) != 2 || got.RomsFolders[0] != "/roms1" || got.RomsFolders[1] != "/roms2" {
		t.Errorf("RomsFolders = %v, want [/roms1 /roms2]", got.RomsFolders)
	}
}

func TestSave_CreatesParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "config.json")

	if err := Save(path, Config{RegistryPath: "/registry"}); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	if _, err := Load(path); err != nil {
		t.Fatalf("Load() after Save() error = %v, want nil", err)
	}
}

func TestSetRegistryPath_RelativePath_StoresAbsolutePath(t *testing.T) {
	var cfg Config

	if err := cfg.SetRegistryPath("relative/registry"); err != nil {
		t.Fatalf("SetRegistryPath() error = %v, want nil", err)
	}

	if !filepath.IsAbs(cfg.RegistryPath) {
		t.Errorf("RegistryPath = %q, want absolute path", cfg.RegistryPath)
	}
}

func TestAddRomsFolder_NewPath_AddsIt(t *testing.T) {
	var cfg Config

	added, err := cfg.AddRomsFolder("/roms1")

	if err != nil {
		t.Fatalf("AddRomsFolder() error = %v, want nil", err)
	}
	if !added {
		t.Error("added = false, want true for a new folder")
	}
	if len(cfg.RomsFolders) != 1 {
		t.Fatalf("RomsFolders = %v, want 1 entry", cfg.RomsFolders)
	}
}

func TestAddRomsFolder_DuplicatePath_DoesNotAddTwice(t *testing.T) {
	var cfg Config
	if _, err := cfg.AddRomsFolder("/roms1"); err != nil {
		t.Fatalf("first AddRomsFolder() error = %v", err)
	}

	added, err := cfg.AddRomsFolder("/roms1")

	if err != nil {
		t.Fatalf("second AddRomsFolder() error = %v, want nil", err)
	}
	if added {
		t.Error("added = true, want false for a duplicate folder")
	}
	if len(cfg.RomsFolders) != 1 {
		t.Errorf("RomsFolders = %v, want 1 entry, no duplicate", cfg.RomsFolders)
	}
}

func TestAddRomsFolder_DuplicateRelativePath_DetectedAsSameAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	var cfg Config
	if _, err := cfg.AddRomsFolder(dir); err != nil {
		t.Fatalf("first AddRomsFolder() error = %v", err)
	}

	added, err := cfg.AddRomsFolder(dir + "/.")
	if err != nil {
		t.Fatalf("second AddRomsFolder() error = %v, want nil", err)
	}
	if added {
		t.Error("added = true, want false: relative path resolves to the same absolute folder")
	}
}
