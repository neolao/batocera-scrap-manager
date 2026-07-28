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

func TestUpdateFile_ThenParseFile_RoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gamelist.xml")
	games := []Game{{Path: "./Sonic.zip", Name: "Sonic"}}

	if err := UpdateFile(path, games); err != nil {
		t.Fatalf("UpdateFile() error = %v, want nil", err)
	}

	got, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}
	if len(got) != 1 || got[0] != games[0] {
		t.Errorf("got = %+v, want %+v", got, games)
	}
}

func TestUpdateFile_DirectoryDoesNotExist_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-dir", "gamelist.xml")

	err := UpdateFile(path, []Game{{Path: "./Sonic.zip"}})

	if err == nil {
		t.Fatal("UpdateFile() error = nil, want error when parent directory does not exist")
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Errorf("file %q should not have been created", path)
	}
}

func TestUpdateFile_TargetAlreadyExists_ReplacesItLeavingNoOtherFileBehind(t *testing.T) {
	folder := t.TempDir()
	path := filepath.Join(folder, "gamelist.xml")
	if err := UpdateFile(path, []Game{{Path: "./Sonic.zip", Name: "Sonic"}}); err != nil {
		t.Fatalf("UpdateFile() error = %v, want nil", err)
	}

	if err := UpdateFile(path, []Game{{Path: "./Streets.zip", Name: "Streets of Rage"}}); err != nil {
		t.Fatalf("UpdateFile() error = %v, want nil", err)
	}

	got, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}
	if len(got) != 1 || got[0].Name != "Streets of Rage" {
		t.Errorf("got = %+v, want the single game %q", got, "Streets of Rage")
	}
	// The document is written beside the old one before being swapped in: that
	// intermediate file must not survive, or every run would litter the user's
	// system folder with copies of their gamelist.
	if names := fileNames(t, folder); len(names) != 1 || names[0] != "gamelist.xml" {
		t.Errorf("folder holds %v, want only [gamelist.xml]", names)
	}
}

func TestUpdateFile_FolderRefusesNewFiles_LeavesTheExistingGamelistUntouched(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions, so the write cannot be made to fail this way")
	}

	folder := t.TempDir()
	path := filepath.Join(folder, "gamelist.xml")
	if err := os.WriteFile(path, []byte(twoGamesXML), 0o644); err != nil {
		t.Fatalf("preparing the gamelist: %v", err)
	}
	if err := os.Chmod(folder, 0o555); err != nil {
		t.Fatalf("making the folder read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(folder, 0o755) })

	err := UpdateFile(path, []Game{{Path: "./Sonic.zip", Name: "Sonic"}})

	if err == nil {
		t.Fatal("UpdateFile() error = nil, want error when the folder refuses new files")
	}
	// The stake of the whole feature: the gamelist is the user's only copy and
	// holds fields the registry never sees. A refused write must leave it as it
	// was, byte for byte — not emptied, not half-rewritten.
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("reading the gamelist back: %v", readErr)
	}
	if string(content) != twoGamesXML {
		t.Errorf("gamelist content = %q, want it unchanged at %q", string(content), twoGamesXML)
	}
	if names := fileNames(t, folder); len(names) != 1 || names[0] != "gamelist.xml" {
		t.Errorf("folder holds %v, want only [gamelist.xml]", names)
	}
}

func TestUpdateFile_TargetAlreadyExists_KeepsItsPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gamelist.xml")
	if err := os.WriteFile(path, []byte(emptyGameListXML), 0o640); err != nil {
		t.Fatalf("preparing the gamelist: %v", err)
	}

	if err := UpdateFile(path, []Game{{Path: "./Sonic.zip", Name: "Sonic"}}); err != nil {
		t.Fatalf("UpdateFile() error = %v, want nil", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v, want nil", err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %o, want 0640 — swapping the file in must not change who may read it", info.Mode().Perm())
	}
}

func TestUpdateFile_TargetIsNew_IsReadableByEveryone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gamelist.xml")

	if err := UpdateFile(path, []Game{{Path: "./Sonic.zip", Name: "Sonic"}}); err != nil {
		t.Fatalf("UpdateFile() error = %v, want nil", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v, want nil", err)
	}
	// A temporary file is created with 0600: left as is, Batocera itself could
	// no longer read the gamelist this tool just wrote.
	if info.Mode().Perm() != 0o644 {
		t.Errorf("mode = %o, want 0644", info.Mode().Perm())
	}
}

// fileNames lists what folder holds, so a test can assert no intermediate file
// was left behind next to the gamelist.
func fileNames(t *testing.T, folder string) []string {
	t.Helper()

	dirEntries, err := os.ReadDir(folder)
	if err != nil {
		t.Fatalf("reading %q: %v", folder, err)
	}
	names := make([]string, len(dirEntries))
	for i, dirEntry := range dirEntries {
		names[i] = dirEntry.Name()
	}
	return names
}

// gamelistWithUserData is a system's game sheet as Batocera really leaves it:
// besides the scraped metadata, it holds what the user's own play sessions
// wrote there, an element carrying markup of its own, and attributes on the
// <game> element itself. None of it is anything this package models.
const gamelistWithUserData = `<?xml version="1.0"?>
<gameList>
  <game id="1234" source="ScreenScraper.fr">
    <path>./Sonic.zip</path>
    <name>Sonic the Hedgehog</name>
    <favorite>true</favorite>
    <playcount>17</playcount>
    <lastplayed>20260101T120000</lastplayed>
  </game>
  <game>
    <path>./Streets.zip</path>
    <name>Streets of Rage</name>
    <desc>Three fighters clean up the city.</desc>
    <hidden>true</hidden>
    <scrap><source name="ScreenScraper"/>done</scrap>
  </game>
</gameList>
`

// updatedFrom rewrites gamelistWithUserData through UpdateFile after applying
// change to the games it holds, and returns the resulting document as text —
// the preserved elements are deliberately absent from Game, so the file itself
// is the only place they can be observed.
func updatedFrom(t *testing.T, change func(games []Game)) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "gamelist.xml")
	if err := os.WriteFile(path, []byte(gamelistWithUserData), 0o644); err != nil {
		t.Fatalf("preparing the gamelist: %v", err)
	}

	games, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}
	change(games)

	if err := UpdateFile(path, games); err != nil {
		t.Fatalf("UpdateFile() error = %v, want nil", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the rewritten gamelist: %v", err)
	}
	return string(content)
}

func TestUpdateFile_CompletedGameHeldTheUsersOwnData_KeepsItWithItsValues(t *testing.T) {
	got := updatedFrom(t, func(games []Game) {
		games[0].Desc = "A blue hedgehog runs very fast."
	})

	if !strings.Contains(got, "<desc>A blue hedgehog runs very fast.</desc>") {
		t.Errorf("the completion did not take: %s", got)
	}
	for _, want := range []string{
		"<favorite>true</favorite>",
		"<playcount>17</playcount>",
		"<lastplayed>20260101T120000</lastplayed>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rewritten gamelist lost %s — that is the user's own play history\n--- file ---\n%s", want, got)
		}
	}
}

func TestUpdateFile_UntouchedGameHeldAnUnmodelledElement_KeepsItToo(t *testing.T) {
	got := updatedFrom(t, func(games []Game) {
		games[0].Desc = "Only the first game is completed."
	})

	if !strings.Contains(got, "<hidden>true</hidden>") {
		t.Errorf("rewritten gamelist lost the second game's <hidden>, which nothing even asked to change\n--- file ---\n%s", got)
	}
}

func TestUpdateFile_UnmodelledElementCarriesMarkup_KeepsItVerbatim(t *testing.T) {
	got := updatedFrom(t, func(games []Game) {
		games[0].Desc = "A blue hedgehog runs very fast."
	})

	if !strings.Contains(got, `<scrap><source name="ScreenScraper"/>done</scrap>`) {
		t.Errorf("an unmodelled element holding markup of its own was not written back as it was\n--- file ---\n%s", got)
	}
}

func TestUpdateFile_GameElementCarriesAttributes_KeepsThem(t *testing.T) {
	got := updatedFrom(t, func(games []Game) {
		games[0].Desc = "A blue hedgehog runs very fast."
	})

	if !strings.Contains(got, `id="1234"`) || !strings.Contains(got, `source="ScreenScraper.fr"`) {
		t.Errorf("the <game> element lost the attributes it carried\n--- file ---\n%s", got)
	}
}

func TestUpdateFile_GameDroppedFromTheList_TakesItsUnmodelledElementsWithIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gamelist.xml")
	if err := os.WriteFile(path, []byte(gamelistWithUserData), 0o644); err != nil {
		t.Fatalf("preparing the gamelist: %v", err)
	}
	games, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}

	// Only the first game is written back — the second leaves the document.
	if err := UpdateFile(path, games[:1]); err != nil {
		t.Fatalf("UpdateFile() error = %v, want nil", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the rewritten gamelist: %v", err)
	}
	got := string(content)
	if strings.Contains(got, "<hidden>true</hidden>") {
		t.Errorf("the dropped game's own elements survived it, re-attached to nothing\n--- file ---\n%s", got)
	}
	if !strings.Contains(got, "<playcount>17</playcount>") {
		t.Errorf("the game that stayed lost what it carried\n--- file ---\n%s", got)
	}
}

func TestUpdateFile_RewrittenTwice_DoesNotDuplicateWhatItPreserves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gamelist.xml")
	if err := os.WriteFile(path, []byte(gamelistWithUserData), 0o644); err != nil {
		t.Fatalf("preparing the gamelist: %v", err)
	}
	games, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}

	for range 2 {
		if err := UpdateFile(path, games); err != nil {
			t.Fatalf("UpdateFile() error = %v, want nil", err)
		}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the rewritten gamelist: %v", err)
	}
	if count := strings.Count(string(content), "<playcount>17</playcount>"); count != 1 {
		t.Errorf("<playcount> appears %d times after two rewrites, want 1\n--- file ---\n%s", count, string(content))
	}
}
