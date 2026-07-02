// Package gamelist parses and writes EmulationStation/Batocera gamelist.xml
// files.
package gamelist

import (
	"encoding/xml"
	"io"
	"os"
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

// WriteFile writes games as a gamelist.xml document to the file at path,
// creating it if needed or truncating it if it already exists.
func WriteFile(path string, games []Game) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return Write(f, games)
}
