// Package gamelist parses and writes EmulationStation/Batocera gamelist.xml
// files.
package gamelist

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Game is a single entry of a gamelist.xml file.
type Game struct {
	Path        string `xml:"path" json:"path"`
	Name        string `xml:"name,omitempty" json:"name"`
	Desc        string `xml:"desc,omitempty" json:"desc"`
	Image       string `xml:"image,omitempty" json:"image"`
	Video       string `xml:"video,omitempty" json:"video"`
	Marquee     string `xml:"marquee,omitempty" json:"marquee"`
	Thumbnail   string `xml:"thumbnail,omitempty" json:"thumbnail"`
	Rating      string `xml:"rating,omitempty" json:"rating"`
	ReleaseDate string `xml:"releasedate,omitempty" json:"release_date"`
	Developer   string `xml:"developer,omitempty" json:"developer"`
	Publisher   string `xml:"publisher,omitempty" json:"publisher"`
	Genre       string `xml:"genre,omitempty" json:"genre"`
	Players     string `xml:"players,omitempty" json:"players"`
}

// unmodelledElement is one child of a `<game>` this package does not know
// about — a favourite mark, a play count, a last-played date, or whatever else
// Batocera and its scrapers write there. It is captured whole (name,
// attributes, raw inner content) so a rewrite can put it back exactly as it
// was found, markup included.
type unmodelledElement struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Inner   string     `xml:",innerxml"`
}

// documentGame is a `<game>` as this package round-trips it: the fields Game
// models, plus everything else the element carried. It stays unexported on
// purpose — Game itself must remain a plain comparable struct the registry can
// test with `==`, and must never start carrying values that belong to the
// user's ROMs folder rather than to the registry (see backlog item 024).
type documentGame struct {
	Game
	Attrs      []xml.Attr          `xml:",any,attr"`
	Unmodelled []unmodelledElement `xml:",any"`
}

type gameListXML struct {
	XMLName xml.Name       `xml:"gameList"`
	Games   []documentGame `xml:"game"`
}

// Parse reads a gamelist.xml document from r and returns its game entries.
// What the document holds beyond those fields is dropped here: only a rewrite
// through UpdateFile has any use for it.
func Parse(r io.Reader) ([]Game, error) {
	parsed, err := parseDocument(r)
	if err != nil {
		return nil, err
	}

	games := make([]Game, len(parsed))
	for i, game := range parsed {
		games[i] = game.Game
	}
	return games, nil
}

// parseDocument reads a gamelist.xml document from r, keeping what this
// package does not model alongside what it does.
func parseDocument(r io.Reader) ([]documentGame, error) {
	var gl gameListXML
	if err := xml.NewDecoder(r).Decode(&gl); err != nil {
		return nil, err
	}
	return gl.Games, nil
}

// ParseFile reads and parses the gamelist.xml file at path.
func ParseFile(path string) ([]Game, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return Parse(f)
}

// Write encodes games as a gamelist.xml document to w. It writes exactly what
// Game models — use UpdateFile to rewrite an existing document without
// discarding the rest of it.
func Write(w io.Writer, games []Game) error {
	return writeDocument(w, documentGames(games, nil))
}

// writeDocument encodes games — modelled fields and preserved remainder alike
// — as a gamelist.xml document to w.
func writeDocument(w io.Writer, games []documentGame) error {
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(gameListXML{Games: games}); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

// documentGames pairs each game with whatever preserved was holding for its
// ROM path, so a rewrite puts it back on the game it came from. A nil
// preserved — or a game the document did not hold — simply carries nothing.
func documentGames(games []Game, preserved map[string]documentGame) []documentGame {
	document := make([]documentGame, len(games))
	for i, game := range games {
		document[i] = documentGame{
			Game:       game,
			Attrs:      preserved[game.Path].Attrs,
			Unmodelled: preserved[game.Path].Unmodelled,
		}
	}
	return document
}

// preservedOf indexes, by ROM path, what each game of the document at path
// carries that this package does not model. A file that is not there yet
// yields nothing to preserve and no error — the caller is then simply
// creating it. A ROM path listed twice keeps the last entry's remainder, a
// case a game sheet should not present in the first place.
func preservedOf(path string) (map[string]documentGame, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	games, err := parseDocument(f)
	if err != nil {
		return nil, err
	}

	preserved := make(map[string]documentGame, len(games))
	for _, game := range games {
		if len(game.Attrs) > 0 || len(game.Unmodelled) > 0 {
			preserved[game.Path] = game
		}
	}
	return preserved, nil
}

// defaultFileMode is what a gamelist.xml gets when it did not exist yet. A
// temporary file is created private to its owner, which would leave Batocera
// unable to read the list this tool just wrote.
const defaultFileMode = 0o644

// UpdateFile rewrites the gamelist.xml at path with games, creating it if it
// does not exist yet.
//
// It reads the document already in place first, and puts back on each game
// everything that document held which this package does not model — the
// favourite mark, the play count, the last-played date, the attributes the
// scraper left on the element. Rebuilding the file from Game alone would erase
// all of it, and none of it is this tool's to erase: it is the user's, and
// their game sheet is its only copy (see backlog item 024). Games are matched
// by ROM path, so one dropped from the list takes its own remainder with it.
//
// The document is written to a temporary file in the same folder and only then
// swapped in, so an interrupted or failed write leaves the previous
// gamelist.xml untouched rather than truncated (see decisions/026). The file's
// existing permissions are kept.
func UpdateFile(path string, games []Game) error {
	preserved, err := preservedOf(path)
	if err != nil {
		return err
	}

	mode := os.FileMode(defaultFileMode)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}

	// Created in the target's own folder: a rename is only atomic within one
	// filesystem, and the system folder may well be a mount of its own.
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpPath)
	}()

	if err := writeDocument(tmp, documentGames(games, preserved)); err != nil {
		return err
	}
	// Flushed before the swap, or a crash right after the rename could leave
	// the new name pointing at a file whose content never reached the disk.
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
