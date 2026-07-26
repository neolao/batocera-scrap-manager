package site

import (
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

func TestGroupBySystem_SeveralSystems_SortedBySystemThenGameName(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic"}},
			{System: "mastersystem", Game: gamelist.Game{Path: "Alex.zip", Name: "Alex Kidd"}},
			{System: "megadrive", Game: gamelist.Game{Path: "Altered.zip", Name: "Altered Beast"}},
		},
	}

	systems := GroupBySystem(reg.Entries, registryFolder)

	if len(systems) != 2 {
		t.Fatalf("len(systems) = %d, want 2", len(systems))
	}
	if systems[0].Name != "mastersystem" || systems[1].Name != "megadrive" {
		t.Errorf("systems = %q, %q; want mastersystem, megadrive", systems[0].Name, systems[1].Name)
	}
	if len(systems[1].Games) != 2 {
		t.Fatalf("len(megadrive games) = %d, want 2", len(systems[1].Games))
	}
	if systems[1].Games[0].Name != "Altered Beast" || systems[1].Games[1].Name != "Sonic" {
		t.Errorf("megadrive games = %q, %q; want Altered Beast, Sonic",
			systems[1].Games[0].Name, systems[1].Games[1].Name)
	}
}

func TestGroupBySystem_GameWithEveryMediaOnDisk_ExposesIDSystemAndEscapedPaths(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{
				Path:      "./roms/Sonic & Knuckles (USA).zip",
				Name:      "Sonic & Knuckles",
				Image:     "images/Sonic & Knuckles [!].png",
				Video:     "videos/sonic.mp4",
				Marquee:   "marquees/sonic.png",
				Thumbnail: "thumbs/sonic.png",
				Rating:    "0.8",
			}},
		},
	}
	for _, relPath := range []string{
		"images/Sonic & Knuckles [!].png",
		"videos/sonic.mp4",
		"marquees/sonic.png",
		"thumbs/sonic.png",
	} {
		writeMediaFile(t, registryFolder, "megadrive", relPath)
	}

	game := GroupBySystem(reg.Entries, registryFolder)[0].Games[0]

	if game.ID != "Sonic & Knuckles (USA)" {
		t.Errorf("game.ID = %q, want %q", game.ID, "Sonic & Knuckles (USA)")
	}
	if game.System != "megadrive" {
		t.Errorf("game.System = %q, want %q", game.System, "megadrive")
	}
	if game.ImagePath != "megadrive/images/Sonic%20&%20Knuckles%20%5B%21%5D.png" {
		t.Errorf("game.ImagePath = %q, want the percent-encoded media path", game.ImagePath)
	}
	if game.VideoPath != "megadrive/videos/sonic.mp4" {
		t.Errorf("game.VideoPath = %q, want %q", game.VideoPath, "megadrive/videos/sonic.mp4")
	}
	if game.MarqueePath != "megadrive/marquees/sonic.png" {
		t.Errorf("game.MarqueePath = %q, want %q", game.MarqueePath, "megadrive/marquees/sonic.png")
	}
	if game.ThumbnailPath != "megadrive/thumbs/sonic.png" {
		t.Errorf("game.ThumbnailPath = %q, want %q", game.ThumbnailPath, "megadrive/thumbs/sonic.png")
	}
	if game.Stars != "★★★★☆" {
		t.Errorf("game.Stars = %q, want %q", game.Stars, "★★★★☆")
	}
	if game.RatingLabel != "4/5" {
		t.Errorf("game.RatingLabel = %q, want %q, so the rating is not conveyed by glyphs alone", game.RatingLabel, "4/5")
	}
}

func TestGroupBySystem_MediaReferencedButMissingOnDisk_LeavesItsPathEmpty(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{
				Path:      "Sonic.zip",
				Name:      "Sonic",
				Image:     "images/sonic.png",
				Video:     "videos/sonic.mp4",
				Marquee:   "marquees/sonic.png",
				Thumbnail: "thumbs/sonic.png",
			}},
		},
	}
	writeMediaFile(t, registryFolder, "megadrive", "images/sonic.png")

	game := GroupBySystem(reg.Entries, registryFolder)[0].Games[0]

	if game.ImagePath == "" {
		t.Error("game.ImagePath is empty although the jaquette exists on disk")
	}
	for label, path := range map[string]string{
		"VideoPath":     game.VideoPath,
		"MarqueePath":   game.MarqueePath,
		"ThumbnailPath": game.ThumbnailPath,
	} {
		if path != "" {
			t.Errorf("game.%s = %q, want empty since the file is missing from the registry folder", label, path)
		}
	}
}

func TestGroupBySystem_GameWithNoRating_HasNoStarsAndNoRatingLabel(t *testing.T) {
	registryFolder := t.TempDir()
	reg := &registry.Registry{
		Entries: []registry.Entry{
			{System: "megadrive", Game: gamelist.Game{Path: "Sonic.zip", Name: "Sonic"}},
		},
	}

	game := GroupBySystem(reg.Entries, registryFolder)[0].Games[0]

	if game.Stars != "" {
		t.Errorf("game.Stars = %q, want empty for a game with no rating", game.Stars)
	}
	if game.RatingLabel != "" {
		t.Errorf("game.RatingLabel = %q, want empty for a game with no rating", game.RatingLabel)
	}
}

func TestGroupBySystem_EmptyRegistry_ReturnsNoSystem(t *testing.T) {
	if systems := GroupBySystem(nil, t.TempDir()); len(systems) != 0 {
		t.Errorf("len(systems) = %d, want 0 for an empty registry", len(systems))
	}
}

func TestFormatStars_RatingBetweenZeroAndOne_RendersRoundedFiveStarScale(t *testing.T) {
	cases := map[string]string{
		"0":            "☆☆☆☆☆",
		"0.5":          "★★★☆☆",
		"1":            "★★★★★",
		"1.5":          "★★★★★",
		"-0.2":         "☆☆☆☆☆",
		"not-a-number": "",
		"":             "",
	}
	for rating, want := range cases {
		if got := FormatStars(rating); got != want {
			t.Errorf("FormatStars(%q) = %q, want %q", rating, got, want)
		}
	}
}

func TestFormatYear_ReleaseDate_KeepsOnlyAFourDigitYear(t *testing.T) {
	cases := map[string]string{
		"19910623T000000": "1991",
		"1991":            "1991",
		"199":             "",
		"abcd1234":        "",
		"":                "",
	}
	for releaseDate, want := range cases {
		if got := FormatYear(releaseDate); got != want {
			t.Errorf("FormatYear(%q) = %q, want %q", releaseDate, got, want)
		}
	}
}

func TestEscapeMediaPath_ReservedCharacters_ArePercentEncodedPerSegment(t *testing.T) {
	got := EscapeMediaPath("megadrive", "images/Sonic [USA] (rev 1).png")

	want := "megadrive/images/Sonic%20%5BUSA%5D%20%28rev%201%29.png"
	if got != want {
		t.Errorf("EscapeMediaPath() = %q, want %q", got, want)
	}
}

func TestStyleSheet_IsSharedAndCarriesThemeAndResponsiveRules(t *testing.T) {
	css := string(StyleSheet)

	for _, want := range []string{"--bg:", "--cyan:", ".card", ".console", "@media"} {
		if !strings.Contains(css, want) {
			t.Errorf("shared stylesheet does not contain %q", want)
		}
	}
	if strings.Contains(css, ".modal__panel") {
		t.Error("shared stylesheet contains the static site's modal rules, which the served pages do not use")
	}
}
