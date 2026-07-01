// Package registry stores the centralized index of scraped game metadata,
// populated by importing existing gamelist.xml files from Batocera ROMs
// folders without duplicating already-known entries.
package registry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
)

// Entry associates a parsed game with the Batocera system it belongs to.
type Entry struct {
	System string        `json:"system"`
	Game   gamelist.Game `json:"game"`
}

// Registry is the centralized index of games already known.
type Registry struct {
	Entries []Entry `json:"entries"`
}

// Load reads the registry from path. If the file does not exist, it returns
// an empty Registry with no error.
func Load(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
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

// Save writes reg to path as JSON, creating parent directories as needed.
func Save(path string, reg *Registry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Import merges games (belonging to system) into the registry. An entry is
// considered already known when an existing entry has the same system and
// the same game path; if its metadata differs from the imported game it is
// replaced and counted as updated, otherwise it is counted as unchanged.
func (r *Registry) Import(system string, games []gamelist.Game) (added, updated, unchanged int) {
	for _, g := range games {
		if i := r.indexOf(system, g.Path); i != -1 {
			if r.Entries[i].Game == g {
				unchanged++
			} else {
				r.Entries[i].Game = g
				updated++
			}
			continue
		}
		r.Entries = append(r.Entries, Entry{System: system, Game: g})
		added++
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
// silently, since not every system has been scraped yet.
func ImportFromRomsFolder(reg *Registry, romsFolder string) (added, updated, unchanged int, err error) {
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

		a, u, unc := reg.Import(system, games)
		added += a
		updated += u
		unchanged += unc
	}

	return added, updated, unchanged, nil
}
