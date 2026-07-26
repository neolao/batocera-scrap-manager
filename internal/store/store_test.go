package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

func sonicRegistry() *registry.Registry {
	return &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic the Hedgehog"}},
	}}
}

func TestSave_ValidRegistry_WritesTheGameFilesAndTheConsultationSite(t *testing.T) {
	folder := t.TempDir()

	if err := Save(sonicRegistry(), folder); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	if _, err := os.Stat(filepath.Join(folder, "megadrive", "Sonic.json")); err != nil {
		t.Errorf("expected the game file to be written: %v", err)
	}
	index, err := os.ReadFile(filepath.Join(folder, "index.html"))
	if err != nil {
		t.Fatalf("expected the consultation site to be regenerated: %v", err)
	}
	if !strings.Contains(string(index), "Sonic the Hedgehog") {
		t.Error("index.html does not mention the game, want the site regenerated from the saved registry")
	}
}

func TestSave_RegistryCannotBeWritten_ReturnsAPlainFailure(t *testing.T) {
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "megadrive"), []byte("blocker"), 0o644); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}

	err := Save(sonicRegistry(), folder)

	if err == nil {
		t.Fatal("Save() error = nil, want an error when the registry cannot be written")
	}
	if errors.Is(err, ErrSiteNotRegenerated) {
		t.Error("error is ErrSiteNotRegenerated, want a plain failure: nothing was saved at all")
	}
}

func TestSave_SiteCannotBeGenerated_ReportsItWithoutDenyingTheRegistryWasSaved(t *testing.T) {
	folder := t.TempDir()
	// A directory where index.html belongs: the registry still saves, the site
	// cannot be written.
	if err := os.Mkdir(filepath.Join(folder, "index.html"), 0o755); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}

	err := Save(sonicRegistry(), folder)

	if !errors.Is(err, ErrSiteNotRegenerated) {
		t.Fatalf("error = %v, want ErrSiteNotRegenerated", err)
	}
	if _, statErr := os.Stat(filepath.Join(folder, "megadrive", "Sonic.json")); statErr != nil {
		t.Errorf("expected the registry itself to have been saved: %v", statErr)
	}
}
