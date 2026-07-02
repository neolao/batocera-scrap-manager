package gamelist

import (
	"os"
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

func TestWrite_ThenParse_RoundTripsAllFields(t *testing.T) {
	games := []Game{
		{
			Path: "./Sonic.zip", Name: "Sonic the Hedgehog", Desc: "A classic platformer.",
			Image: "./media/images/Sonic.png", Video: "./media/videos/Sonic.mp4",
			Marquee: "./media/marquees/Sonic.png", Thumbnail: "./media/thumbnails/Sonic.png",
			Rating: "0.8", ReleaseDate: "19910101T000000", Developer: "Sonic Team",
			Publisher: "Sega", Genre: "Platform", Players: "1",
		},
		{Path: "./Streets of Rage.zip", Name: "Streets of Rage", Genre: "Beat 'em up"},
	}
	var buf strings.Builder

	err := Write(&buf, games)
	if err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	got, err := Parse(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("Parse(Write() output) error = %v, want nil", err)
	}
	if len(got) != len(games) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(games))
	}
	if got[0] != games[0] {
		t.Errorf("got[0] = %+v, want %+v", got[0], games[0])
	}
	if got[1] != games[1] {
		t.Errorf("got[1] = %+v, want %+v", got[1], games[1])
	}
}

func TestWrite_EmptyGameList_ProducesParsableEmptyList(t *testing.T) {
	var buf strings.Builder

	err := Write(&buf, nil)
	if err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	got, err := Parse(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("Parse(Write() output) error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Errorf("len(got) = %d, want 0", len(got))
	}
}

func TestWrite_NameWithSpecialXMLCharacters_EscapesAndRoundTrips(t *testing.T) {
	games := []Game{{Path: "./game.zip", Name: `Tom & Jerry: "Cat" <Mouse>`}}
	var buf strings.Builder

	if err := Write(&buf, games); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	got, err := Parse(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("Parse(Write() output) error = %v, want nil", err)
	}
	if len(got) != 1 || got[0].Name != games[0].Name {
		t.Fatalf("got = %+v, want Name %q preserved", got, games[0].Name)
	}
}

func TestWriteFile_ThenParseFile_RoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gamelist.xml")
	games := []Game{{Path: "./Sonic.zip", Name: "Sonic"}}

	if err := WriteFile(path, games); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	got, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}
	if len(got) != 1 || got[0] != games[0] {
		t.Errorf("got = %+v, want %+v", got, games)
	}
}

func TestWriteFile_DirectoryDoesNotExist_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-dir", "gamelist.xml")

	err := WriteFile(path, []Game{{Path: "./Sonic.zip"}})

	if err == nil {
		t.Fatal("WriteFile() error = nil, want error when parent directory does not exist")
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Errorf("file %q should not have been created", path)
	}
}
