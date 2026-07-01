// Package registry stores the centralized index of scraped game metadata and
// media, populated by importing existing gamelist.xml files (and the media
// they reference) from Batocera ROMs folders without duplicating
// already-known entries.
package registry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
)

// indexFileName is the name of the JSON file holding the registry index,
// stored at the root of the registry folder alongside the per-system media
// folders mirroring the Batocera ROMs arborescence.
const indexFileName = "registry.json"

// Entry associates a parsed game with the Batocera system it belongs to.
type Entry struct {
	System string        `json:"system"`
	Game   gamelist.Game `json:"game"`
}

// Registry is the centralized index of games already known.
type Registry struct {
	Entries []Entry `json:"entries"`
}

// Load reads the registry index from the registry folder at path. If the
// folder or its index file does not exist, it returns an empty Registry
// with no error.
func Load(path string) (*Registry, error) {
	data, err := os.ReadFile(filepath.Join(path, indexFileName))
	if errors.Is(err, os.ErrNotExist) {
		return &Registry{}, nil
	}
	if err != nil {
		return nil, err
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	return &reg, nil
}

// Save writes reg as the registry index inside the registry folder at path,
// creating the folder as needed.
func Save(path string, reg *Registry) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, indexFileName), data, 0o644)
}

type importStatus int

const (
	statusUnchanged importStatus = iota
	statusAdded
	statusUpdated
)

// importGame merges g (belonging to system) into the registry, reporting
// whether it was newly added, replaced an existing entry with different
// metadata, or left unchanged.
func (r *Registry) importGame(system string, g gamelist.Game) importStatus {
	if i := r.indexOf(system, g.Path); i != -1 {
		if r.Entries[i].Game == g {
			return statusUnchanged
		}
		r.Entries[i].Game = g
		return statusUpdated
	}
	r.Entries = append(r.Entries, Entry{System: system, Game: g})
	return statusAdded
}

// Import merges games (belonging to system) into the registry. An entry is
// considered already known when an existing entry has the same system and
// the same game path; if its metadata differs from the imported game it is
// replaced and counted as updated, otherwise it is counted as unchanged.
func (r *Registry) Import(system string, games []gamelist.Game) (added, updated, unchanged int) {
	for _, g := range games {
		switch r.importGame(system, g) {
		case statusAdded:
			added++
		case statusUpdated:
			updated++
		default:
			unchanged++
		}
	}
	return added, updated, unchanged
}

func (r *Registry) indexOf(system, path string) int {
	for i, e := range r.Entries {
		if e.System == system && e.Game.Path == path {
			return i
		}
	}
	return -1
}

// ImportFromRomsFolder scans the immediate subdirectories of romsFolder (each
// one a Batocera system) for a gamelist.xml file, parses it, and imports its
// entries into reg. Subdirectories without a gamelist.xml are skipped
// silently, since not every system has been scraped yet. For every game that
// is newly added or whose metadata changed, its referenced media files
// (cover art, video, marquee, thumbnail) are copied from romsFolder into
// registryFolder, mirroring the Batocera per-system arborescence; unchanged
// games are not recopied.
func ImportFromRomsFolder(reg *Registry, romsFolder, registryFolder string) (added, updated, unchanged int, err error) {
	dirEntries, err := os.ReadDir(romsFolder)
	if err != nil {
		return 0, 0, 0, err
	}

	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}

		system := dirEntry.Name()
		gamelistPath := filepath.Join(romsFolder, system, "gamelist.xml")
		if _, statErr := os.Stat(gamelistPath); statErr != nil {
			continue
		}

		games, parseErr := gamelist.ParseFile(gamelistPath)
		if parseErr != nil {
			return added, updated, unchanged, parseErr
		}

		for _, g := range games {
			status := reg.importGame(system, g)
			switch status {
			case statusAdded:
				added++
			case statusUpdated:
				updated++
			default:
				unchanged++
				continue
			}

			if copyErr := copyGameMedia(romsFolder, registryFolder, system, g); copyErr != nil {
				return added, updated, unchanged, copyErr
			}
		}
	}

	return added, updated, unchanged, nil
}

// copyGameMedia copies every media file referenced by g (cover art, video,
// marquee, thumbnail) from its system folder under romsFolder into the same
// relative location under registryFolder.
func copyGameMedia(romsFolder, registryFolder, system string, g gamelist.Game) error {
	for _, relPath := range []string{g.Image, g.Video, g.Marquee, g.Thumbnail} {
		if err := copyMediaFile(romsFolder, registryFolder, system, relPath); err != nil {
			return err
		}
	}
	return nil
}

// copyMediaFile copies the media file at relPath (relative to the system
// folder, as referenced in gamelist.xml) from romsFolder to registryFolder.
// An empty relPath, or a referenced file missing on disk, is silently
// ignored.
func copyMediaFile(romsFolder, registryFolder, system, relPath string) error {
	if relPath == "" {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(romsFolder, system, relPath))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	dst := filepath.Join(registryFolder, system, relPath)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
