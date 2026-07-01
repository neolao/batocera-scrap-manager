package gamelist

import (
	"path/filepath"
	"strings"
	"testing"
)

const twoGamesXML = `<?xml version="1.0"?>
<gameList>
  <game>
    <path>./Sonic.zip</path>
    <name>Sonic the Hedgehog</name>
    <desc>A classic platformer.</desc>
    <image>./media/images/Sonic.png</image>
    <video>./media/videos/Sonic.mp4</video>
    <marquee>./media/marquees/Sonic.png</marquee>
    <thumbnail>./media/thumbnails/Sonic.png</thumbnail>
    <rating>0.8</rating>
    <releasedate>19910101T000000</releasedate>
    <developer>Sonic Team</developer>
    <publisher>Sega</publisher>
    <genre>Platform</genre>
    <players>1</players>
  </game>
  <game>
    <path>./Streets of Rage.zip</path>
    <name>Streets of Rage</name>
    <genre>Beat 'em up</genre>
  </game>
</gameList>
`

const emptyGameListXML = `<?xml version="1.0"?>
<gameList>
</gameList>
`

func TestParse_MultipleGames_ReturnsAllFieldsPopulated(t *testing.T) {
	games, err := Parse(strings.NewReader(twoGamesXML))

	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(games) != 2 {
		t.Fatalf("len(games) = %d, want 2", len(games))
	}

	sonic := games[0]
	if sonic.Path != "./Sonic.zip" {
		t.Errorf("Path = %q, want %q", sonic.Path, "./Sonic.zip")
	}
	if sonic.Name != "Sonic the Hedgehog" {
		t.Errorf("Name = %q, want %q", sonic.Name, "Sonic the Hedgehog")
	}
	if sonic.Desc != "A classic platformer." {
		t.Errorf("Desc = %q, want %q", sonic.Desc, "A classic platformer.")
	}
	if sonic.Image != "./media/images/Sonic.png" {
		t.Errorf("Image = %q, want %q", sonic.Image, "./media/images/Sonic.png")
	}
	if sonic.Developer != "Sonic Team" {
		t.Errorf("Developer = %q, want %q", sonic.Developer, "Sonic Team")
	}
	if sonic.Genre != "Platform" {
		t.Errorf("Genre = %q, want %q", sonic.Genre, "Platform")
	}
}

func TestParse_EmptyGameList_ReturnsEmptySlice(t *testing.T) {
	games, err := Parse(strings.NewReader(emptyGameListXML))

	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(games) != 0 {
		t.Errorf("len(games) = %d, want 0", len(games))
	}
}

func TestParse_GameMissingOptionalFields_ReturnsZeroValues(t *testing.T) {
	games, err := Parse(strings.NewReader(twoGamesXML))

	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}

	streetsOfRage := games[1]
	if streetsOfRage.Video != "" {
		t.Errorf("Video = %q, want empty", streetsOfRage.Video)
	}
	if streetsOfRage.Rating != "" {
		t.Errorf("Rating = %q, want empty", streetsOfRage.Rating)
	}
	if streetsOfRage.Name != "Streets of Rage" {
		t.Errorf("Name = %q, want %q", streetsOfRage.Name, "Streets of Rage")
	}
}

func TestParse_MalformedXML_ReturnsError(t *testing.T) {
	_, err := Parse(strings.NewReader("<gameList><game><name>oops</game></gameList>"))

	if err == nil {
		t.Fatal("Parse() error = nil, want error for malformed XML")
	}
}

func TestParseFile_FileDoesNotExist_ReturnsError(t *testing.T) {
	_, err := ParseFile(filepath.Join(t.TempDir(), "missing-gamelist.xml"))

	if err == nil {
		t.Fatal("ParseFile() error = nil, want error for missing file")
	}
}
