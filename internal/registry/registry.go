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
	"strings"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
)

// Entry associates a parsed game with the Batocera system it belongs to.
type Entry struct {
	System string
	Game   gamelist.Game
}

// Registry is the centralized index of games already known.
type Registry struct {
	Entries []Entry
}

// Load reconstructs the registry from the registry folder at path, by
// scanning its per-system subfolders for the game JSON files written there
// by Save. If the folder does not exist, it returns an empty Registry with
// no error.
func Load(path string) (*Registry, error) {
	systemDirs, err := os.ReadDir(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Registry{}, nil
	}
	if err != nil {
		return nil, err
	}

	reg := &Registry{}
	for _, systemDir := range systemDirs {
		if !systemDir.IsDir() {
			continue
		}
		system := systemDir.Name()

		gameFiles, err := os.ReadDir(filepath.Join(path, system))
		if err != nil {
			return nil, err
		}
		for _, gameFile := range gameFiles {
			if gameFile.IsDir() || filepath.Ext(gameFile.Name()) != ".json" {
				continue
			}

			data, err := os.ReadFile(filepath.Join(path, system, gameFile.Name()))
			if err != nil {
				return nil, err
			}
			var g gamelist.Game
			if err := json.Unmarshal(data, &g); err != nil {
				return nil, err
			}
			reg.Entries = append(reg.Entries, Entry{System: system, Game: g})
		}
	}
	return reg, nil
}

// Save writes reg to the registry folder at path, as one JSON file per game
// inside its system's subfolder (named after the ROM's base name), creating
// folders as needed.
func Save(path string, reg *Registry) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}

	for _, e := range reg.Entries {
		systemDir := filepath.Join(path, e.System)
		if err := os.MkdirAll(systemDir, 0o755); err != nil {
			return err
		}

		data, err := json.MarshalIndent(e.Game, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(systemDir, gameFileName(e.Game)), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// gameFileName derives the name of the JSON file storing g's metadata, from
// the base name of its ROM path.
func gameFileName(g gamelist.Game) string {
	base := filepath.Base(g.Path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext) + ".json"
}

// ErrGameNotFound is returned by Remove when no entry matches the given
// system and ROM filename.
var ErrGameNotFound = errors.New("registry: game not found")

// Remove deletes, from the registry folder at registryFolder, the metadata
// file and every media file (cover art, video, marquee, thumbnail)
// belonging to the entry matching system and romFilename (e.g. "Sonic.zip"
// — any directory prefix is ignored, since the registry does not reproduce
// the ROM's original subfolder), then removes that entry from reg. It
// returns ErrGameNotFound, without modifying anything, if no entry matches.
func Remove(reg *Registry, registryFolder, system, romFilename string) error {
	i := reg.indexOf(system, romFilename)
	if i == -1 {
		return ErrGameNotFound
	}
	g := reg.Entries[i].Game

	if err := removeIfExists(filepath.Join(registryFolder, system, gameFileName(g))); err != nil {
		return err
	}
	for _, relPath := range []string{g.Image, g.Video, g.Marquee, g.Thumbnail} {
		if relPath == "" {
			continue
		}
		if err := removeIfExists(filepath.Join(registryFolder, system, relPath)); err != nil {
			return err
		}
	}

	reg.Entries = append(reg.Entries[:i], reg.Entries[i+1:]...)
	return nil
}

// removeIfExists deletes the file at path, silently ignoring the case where
// it does not exist.
func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
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
// the same ROM filename (ignoring any directory prefix, since the registry
// stores entries flat per system); if its metadata differs from the
// imported game it is replaced and counted as updated, otherwise it is
// counted as unchanged.
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

// indexOf finds the entry matching system and the ROM's filename (the
// base name of path, ignoring any directory prefix) — the registry is
// stored flat per system on disk, so two ROMs sharing a filename in
// different subfolders of the same system are the same entry.
func (r *Registry) indexOf(system, path string) int {
	name := filepath.Base(path)
	for i, e := range r.Entries {
		if e.System == system && filepath.Base(e.Game.Path) == name {
			return i
		}
	}
	return -1
}

// ProgressEvent describes one game being processed by ImportFromRomsFolder,
// for callers that want to report progress to the user as the import runs.
type ProgressEvent struct {
	System    string
	GameIndex int // 1-based index of this game within System's game list
	GameCount int // total number of games found for System
	GameName  string
}

// ImportFromRomsFolder scans the immediate subdirectories of romsFolder (each
// one a Batocera system) for a gamelist.xml file, parses it, and imports its
// entries into reg. Subdirectories without a gamelist.xml are skipped
// silently, since not every system has been scraped yet. For every game that
// is newly added or whose metadata changed, its referenced media files
// (cover art, video, marquee, thumbnail) are copied from romsFolder into
// registryFolder, mirroring the Batocera per-system arborescence; unchanged
// games are not recopied. If onProgress is non-nil, it is called once per
// game as it is processed.
func ImportFromRomsFolder(reg *Registry, romsFolder, registryFolder string, onProgress func(ProgressEvent)) (added, updated, unchanged int, err error) {
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

		for i, g := range games {
			if !hasScrapedData(g) {
				continue
			}

			if onProgress != nil {
				onProgress(ProgressEvent{System: system, GameIndex: i + 1, GameCount: len(games), GameName: g.Name})
			}

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

// hasScrapedData reports whether g carries any information worth keeping in
// the registry (a description or a jaquette). A game with neither is
// considered unscraped and is not added to the registry.
func hasScrapedData(g gamelist.Game) bool {
	return g.Desc != "" || g.Image != ""
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
// folder, as referenced in gamelist.xml) from srcRoot to dstRoot. An empty
// relPath, or a referenced file missing on disk, is silently ignored.
func copyMediaFile(srcRoot, dstRoot, system, relPath string) error {
	if relPath == "" {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(srcRoot, system, relPath))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	dst := filepath.Join(dstRoot, system, relPath)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// ImportGame imports a single local game entry, identified by system and
// romFilename (matched by ROM base name, ignoring any directory prefix, like
// the rest of the registry — see decisions/005), from romsFolder's local
// gamelist.xml into reg, then copies its referenced media into
// registryFolder if it was newly added or changed. It returns
// ErrGameNotFound if the game has no entry in the system's local
// gamelist.xml. A game with neither a description nor an image is silently
// skipped, exactly like ImportFromRomsFolder — not imported, not counted, no
// progress event, no error (see decisions/007). If onProgress is non-nil, it
// is called once, only if the game was actually imported (added, updated,
// or unchanged).
func ImportGame(reg *Registry, romsFolder, registryFolder, system, romFilename string, onProgress func(ProgressEvent)) (added, updated, unchanged int, err error) {
	gamelistPath := filepath.Join(romsFolder, system, "gamelist.xml")
	games, parseErr := gamelist.ParseFile(gamelistPath)
	if parseErr != nil {
		return 0, 0, 0, ErrGameNotFound
	}

	name := filepath.Base(romFilename)
	i := -1
	for idx, g := range games {
		if filepath.Base(g.Path) == name {
			i = idx
			break
		}
	}
	if i == -1 {
		return 0, 0, 0, ErrGameNotFound
	}

	g := games[i]
	if !hasScrapedData(g) {
		return 0, 0, 0, nil
	}

	if onProgress != nil {
		onProgress(ProgressEvent{System: system, GameIndex: i + 1, GameCount: len(games), GameName: g.Name})
	}

	switch reg.importGame(system, g) {
	case statusAdded:
		added = 1
	case statusUpdated:
		updated = 1
	default:
		return 0, 0, 1, nil
	}

	if copyErr := copyGameMedia(romsFolder, registryFolder, system, g); copyErr != nil {
		return added, updated, unchanged, copyErr
	}
	return added, updated, unchanged, nil
}

// CompletionEvent describes one game being examined by CompleteRomsFolder,
// for callers that want to report progress to the user as it runs.
type CompletionEvent struct {
	System    string
	GameIndex int // 1-based index of this game within System's local game list
	GameCount int // total number of games found for System
	GameName  string
}

// CompleteRomsFolder scans the immediate subdirectories of romsFolder (each
// one a Batocera system) for a gamelist.xml file, and for every local game
// entry missing metadata fields (name, description, image, ...), fills the
// gaps from the matching entry already known in reg (matched by system and
// ROM path), then copies any newly referenced media file from
// registryFolder into romsFolder, mirroring the Batocera per-system
// arborescence. Games with no matching registry entry, or already complete,
// are left untouched. Systems without a local gamelist.xml are silently
// skipped, since the registry only ever completes games already known
// locally. If onProgress is non-nil, it is called once per game that
// actually had a field filled from the registry (whether or not copying its
// media then succeeded) — games left untouched because they were already
// complete, or unknown to the registry, produce no event.
func CompleteRomsFolder(reg *Registry, romsFolder, registryFolder string, onProgress func(CompletionEvent)) (processed, completed, failed int, err error) {
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
			return processed, completed, failed, parseErr
		}

		dirty := false
		for i := range games {
			processed++

			j := reg.indexOf(system, games[i].Path)
			if j == -1 {
				continue
			}

			before := games[i]
			if !mergeGame(&games[i], reg.Entries[j].Game) {
				continue
			}
			dirty = true

			if onProgress != nil {
				onProgress(CompletionEvent{System: system, GameIndex: i + 1, GameCount: len(games), GameName: games[i].Name})
			}

			if copyErr := copyFilledMedia(before, games[i], registryFolder, romsFolder, system); copyErr != nil {
				failed++
				continue
			}
			completed++
		}

		if dirty {
			if writeErr := gamelist.WriteFile(gamelistPath, games); writeErr != nil {
				return processed, completed, failed, writeErr
			}
		}
	}

	return processed, completed, failed, nil
}

// mergeGame fills any empty field of dst with the corresponding non-empty
// field of src, reporting whether anything was filled.
func mergeGame(dst *gamelist.Game, src gamelist.Game) bool {
	changed := false
	fill := func(d *string, s string) {
		if *d == "" && s != "" {
			*d = s
			changed = true
		}
	}
	fill(&dst.Name, src.Name)
	fill(&dst.Desc, src.Desc)
	fill(&dst.Image, src.Image)
	fill(&dst.Video, src.Video)
	fill(&dst.Marquee, src.Marquee)
	fill(&dst.Thumbnail, src.Thumbnail)
	fill(&dst.Rating, src.Rating)
	fill(&dst.ReleaseDate, src.ReleaseDate)
	fill(&dst.Developer, src.Developer)
	fill(&dst.Publisher, src.Publisher)
	fill(&dst.Genre, src.Genre)
	fill(&dst.Players, src.Players)
	return changed
}

// CompleteGame fills the gaps of a single local game entry, identified by
// system and romFilename (matched by ROM base name, ignoring any directory
// prefix, like the rest of the registry — see decisions/005), from the
// matching entry already known in reg, then copies any newly referenced
// media file from registryFolder into romsFolder. It returns
// ErrGameNotFound if the game has no entry in the system's local
// gamelist.xml, or no matching entry exists in reg. completed reports
// whether any field was actually filled; failed reports whether copying the
// newly filled media then failed — the local gamelist.xml is still
// rewritten with the filled fields in that case, mirroring
// CompleteRomsFolder's per-game failure handling. If onProgress is
// non-nil, it is called once, only when a field was actually filled.
func CompleteGame(reg *Registry, romsFolder, registryFolder, system, romFilename string, onProgress func(CompletionEvent)) (completed, failed bool, err error) {
	gamelistPath := filepath.Join(romsFolder, system, "gamelist.xml")
	games, parseErr := gamelist.ParseFile(gamelistPath)
	if parseErr != nil {
		return false, false, ErrGameNotFound
	}

	name := filepath.Base(romFilename)
	i := -1
	for idx, g := range games {
		if filepath.Base(g.Path) == name {
			i = idx
			break
		}
	}
	if i == -1 {
		return false, false, ErrGameNotFound
	}

	j := reg.indexOf(system, games[i].Path)
	if j == -1 {
		return false, false, ErrGameNotFound
	}

	before := games[i]
	if !mergeGame(&games[i], reg.Entries[j].Game) {
		return false, false, nil
	}

	if onProgress != nil {
		onProgress(CompletionEvent{System: system, GameIndex: i + 1, GameCount: len(games), GameName: games[i].Name})
	}

	copyErr := copyFilledMedia(before, games[i], registryFolder, romsFolder, system)

	if writeErr := gamelist.WriteFile(gamelistPath, games); writeErr != nil {
		return false, false, writeErr
	}

	if copyErr != nil {
		return false, true, nil
	}
	return true, false, nil
}

// copyFilledMedia copies, from srcRoot into dstRoot, every media file whose
// reference was newly filled between before and after (i.e. empty in
// before, non-empty in after).
func copyFilledMedia(before, after gamelist.Game, srcRoot, dstRoot, system string) error {
	for _, pair := range [][2]string{
		{before.Image, after.Image},
		{before.Video, after.Video},
		{before.Marquee, after.Marquee},
		{before.Thumbnail, after.Thumbnail},
	} {
		if pair[0] == "" && pair[1] != "" {
			if err := copyMediaFile(srcRoot, dstRoot, system, pair[1]); err != nil {
				return err
			}
		}
	}
	return nil
}
