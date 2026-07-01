// Package gamelist parses EmulationStation/Batocera gamelist.xml files.
package gamelist

import (
	"encoding/xml"
	"io"
	"os"
)

// Game is a single entry of a gamelist.xml file.
type Game struct {
	Path        string `xml:"path" json:"path"`
	Name        string `xml:"name" json:"name"`
	Desc        string `xml:"desc" json:"desc"`
	Image       string `xml:"image" json:"image"`
	Video       string `xml:"video" json:"video"`
	Marquee     string `xml:"marquee" json:"marquee"`
	Thumbnail   string `xml:"thumbnail" json:"thumbnail"`
	Rating      string `xml:"rating" json:"rating"`
	ReleaseDate string `xml:"releasedate" json:"release_date"`
	Developer   string `xml:"developer" json:"developer"`
	Publisher   string `xml:"publisher" json:"publisher"`
	Genre       string `xml:"genre" json:"genre"`
	Players     string `xml:"players" json:"players"`
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
