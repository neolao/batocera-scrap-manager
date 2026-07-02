package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

func TestGenerate_GamesGroupedBySystem_ListsNameDescriptionAndJaquette(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{
				Path:  "Sonic.zip",
				Name:  "Sonic the Hedgehog",
				Desc:  "A blue hedgehog runs fast.",
				Image: "images/sonic.png",
			}},
			{System: "mastersystem", Game: gamelist.Game{
				Path:  "Alex.zip",
				Name:  "Alex Kidd",
				Desc:  "A kid with miracle powers.",
				Image: "images/alex.png",
			}},
		},
	}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	for _, want := range []string{
		"megadrive", "mastersystem",
		"Sonic the Hedgehog", "A blue hedgehog runs fast.", "megadrive/images/sonic.png",
		"Alex Kidd", "A kid with miracle powers.", "mastersystem/images/alex.png",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("index.html does not contain %q", want)
		}
	}
}

func TestGenerate_GameWithoutJaquette_RendersWithoutBrokenImage(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{
				Path: "Streets.zip",
				Name: "Streets of Rage",
				Desc: "Beat 'em up.",
				// Image intentionally left empty.
			}},
		},
	}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	if !strings.Contains(html, "Streets of Rage") {
		t.Errorf("index.html does not contain the game name")
	}
	if strings.Contains(html, `src=""`) || strings.Contains(html, `<img src="megadrive/">`) {
		t.Errorf("index.html contains a broken image reference: %s", html)
	}
}

func TestGenerate_EmptyRegistry_ProducesValidSiteWithNoGamesMessage(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := readIndex(t, registryFolder)

	if !strings.Contains(html, "No games") {
		t.Errorf("index.html does not contain a no-games message, got: %s", html)
	}
}

func TestGenerate_WritesIndexHTMLDirectlyAtRegistryRoot(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{}

	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(registryFolder, "index.html")); err != nil {
		t.Errorf("index.html not found at the registry root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(registryFolder, "site", "index.html")); err == nil {
		t.Error("index.html was written under a site/ subfolder, want it directly at the registry root")
	}
}

func TestGenerate_LeftoverSiteSubfolder_IsLeftUntouched(t *testing.T) {
	registryFolder := t.TempDir()
	staleSiteFolder := filepath.Join(registryFolder, "site")
	if err := os.MkdirAll(staleSiteFolder, 0o755); err != nil {
		t.Fatalf("failed to set up test: %v", err)
	}
	stalePath := filepath.Join(staleSiteFolder, "index.html")
	if err := os.WriteFile(stalePath, []byte("stale content from a previous version"), 0o644); err != nil {
		t.Fatalf("failed to set up test: %v", err)
	}

	reg := &registry.Registry{}
	if err := Generate(reg, registryFolder); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := os.ReadFile(stalePath)
	if err != nil {
		t.Fatalf("leftover site/index.html was removed, want it left untouched: %v", err)
	}
	if string(data) != "stale content from a previous version" {
		t.Errorf("leftover site/index.html content changed, want it left untouched")
	}
}

func TestGenerate_IndexHTMLBlockedByExistingDirectory_ReturnsError(t *testing.T) {
	registryFolder := t.TempDir()
	// Create a directory where the index.html file needs to go, so writing fails.
	if err := os.MkdirAll(filepath.Join(registryFolder, "index.html"), 0o755); err != nil {
		t.Fatalf("failed to set up test: %v", err)
	}

	reg := &registry.Registry{}
	if err := Generate(reg, registryFolder); err == nil {
		t.Fatal("Generate() expected an error, got nil")
	}
}

func readIndex(t *testing.T, registryFolder string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(registryFolder, "index.html"))
	if err != nil {
		t.Fatalf("failed to read generated index.html: %v", err)
	}
	return string(data)
}
