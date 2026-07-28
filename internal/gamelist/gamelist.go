// Package gamelist parses and writes EmulationStation/Batocera gamelist.xml
// files.
package gamelist

import (
	"encoding/xml"
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

type gameListXML struct {
	XMLName xml.Name `xml:"gameList"`
	Games   []Game   `xml:"game"`
}

// Parse reads a gamelist.xml document from r and returns its game entries.
func Parse(r io.Reader) ([]Game, error) {
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

// Write encodes games as a gamelist.xml document to w.
func Write(w io.Writer, games []Game) error {
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

// defaultFileMode is what a gamelist.xml gets when it did not exist yet. A
// temporary file is created private to its owner, which would leave Batocera
// unable to read the list this tool just wrote.
const defaultFileMode = 0o644

// WriteFile writes games as a gamelist.xml document to the file at path,
// creating it if needed or replacing it if it already exists. The document is
// written to a temporary file in the same folder and only then swapped in, so
// an interrupted or failed write leaves the previous gamelist.xml untouched
// rather than truncated — it is the user's only copy, under no version
// control, and it holds fields this tool does not know about (see
// decisions/026). The file's existing permissions are kept.
func WriteFile(path string, games []Game) error {
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

	if err := Write(tmp, games); err != nil {
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
