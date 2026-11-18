package scrape

import (
	"context"
	"strings"
	"testing"
)

func TestLookup_RetrogamingNeolaoGamePage_ReadsEveryField(t *testing.T) {
	fetcher := fakeSite(t, fixture(t, "retrogaming-neolao-game.html"))

	got, err := fetcher.Lookup(context.Background(), gamePage)

	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if got.Name != "Bluey's Happy Snaps" {
		t.Errorf("Name = %q", got.Name)
	}
	if got.ReleaseDate != "20261015T000000" {
		t.Errorf("ReleaseDate = %q, want 20261015T000000", got.ReleaseDate)
	}
	if got.Publisher != "Gameloft" {
		t.Errorf("Publisher = %q, want Gameloft", got.Publisher)
	}
	if got.Genre != "Casse-tête, Aventure, Éducation" {
		t.Errorf("Genre = %q, want the three categories", got.Genre)
	}
	if !strings.HasPrefix(got.Desc, "Jouez et immortalisez chaque instant") ||
		!strings.Contains(got.Desc, "\n\nBluey et Bingo ont trouvé") ||
		!strings.HasSuffix(got.Desc, "aucune fonctionnalité de jeu en ligne.") {
		t.Errorf("Desc = %q, want every paragraph separated by a blank line", got.Desc)
	}
	if got.ImageURL != "https://retrogaming.neolao.com/switch/images/Bluey%E2%80%99s%20Happy%20Snaps.jpg" {
		t.Errorf("ImageURL = %q", got.ImageURL)
	}
	if got.ThumbnailURL != "https://retrogaming.neolao.com/switch/games/Bluey%E2%80%99s%20Happy%20Snaps/screenshots/IMG_20261020_130056_339.jpg" {
		t.Errorf("ThumbnailURL = %q, want the first screenshot", got.ThumbnailURL)
	}
}

func TestLookup_PageMissingMostData_LeavesThoseFieldsEmpty(t *testing.T) {
	fetcher := fakeSite(t, fixture(t, "retrogaming-neolao-sparse.html"))

	got, err := fetcher.Lookup(context.Background(), gamePage)

	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	want := Result{Name: "Lonely Game"}
	if got != want {
		t.Errorf("Lookup() = %+v, want %+v", got, want)
	}
}

func TestReleaseDate_NotADate_IsLeftEmpty(t *testing.T) {
	for _, value := range []string{"", "2026", "2026-1-15", "15/10/2026", "abcd-ef-gh"} {
		if got := releaseDate(value); got != "" {
			t.Errorf("releaseDate(%q) = %q, want empty", value, got)
		}
	}
}
