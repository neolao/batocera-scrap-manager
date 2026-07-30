// Package registry stores the centralized index of scraped game metadata and
// media, populated by importing existing gamelist.xml files (and the media
// they reference) from Batocera ROMs folders without duplicating
// already-known entries.
package registry

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
)

// Entry associates a parsed game with the Batocera system it belongs to.
// ManualFields names the metadata fields of Game that were corrected by hand
// (see editableFields): an import puts those values back instead of letting
// the ROMs folder's own, still badly scraped, value win — see decisions/017.
type Entry struct {
	System       string
	Game         gamelist.Game
	ManualFields []string
}

// storedGame is the on-disk shape of an entry: the game's own fields, flat at
// the root of its JSON file exactly as before, plus the registry's own
// bookkeeping. The key is omitted for a game nobody edited, so files written
// by an earlier version and files written now are byte-identical for
// untouched games, and reading one back yields an entry with nothing
// protected.
type storedGame struct {
	gamelist.Game
	ManualFields []string `json:"manual_fields,omitempty"`
}

// Registry is the centralized index of games already known.
type Registry struct {
	Entries []Entry

	// saved holds the entries as they were last known to match the registry
	// folder on disk — set by Load, carried across by Clone, and refreshed by
	// Save once it has written what changed. It is what lets Save tell which
	// entries actually differ from disk instead of rewriting all of them
	// (backlog item 027). A Registry built directly, without going through
	// Load or Clone, has no baseline and Save falls back to writing every
	// entry — the same behaviour as before this field existed.
	saved []Entry
}

// Clone returns a copy of the registry that can be modified without touching
// the original — the way a caller applies a change, tries to write it to
// disk, and only swaps it in once the write succeeded, so a failed save never
// leaves memory claiming something the disk does not hold. The clone starts
// with the same notion of what is saved on disk as the original, so a Save
// call on the clone still rewrites only what the clone's own changes touched.
func (r *Registry) Clone() *Registry {
	clone := &Registry{Entries: make([]Entry, len(r.Entries)), saved: make([]Entry, len(r.saved))}
	copy(clone.Entries, r.Entries)
	copy(clone.saved, r.saved)
	return clone
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

		entries, err := loadSystemEntries(path, system)
		if err != nil {
			return nil, err
		}
		reg.Entries = append(reg.Entries, entries...)
	}
	reg.saved = make([]Entry, len(reg.Entries))
	copy(reg.saved, reg.Entries)
	return reg, nil
}

// loadSystemEntries reads every game JSON file in the registry folder's
// subfolder for system, returning one Entry per file.
func loadSystemEntries(path, system string) ([]Entry, error) {
	gameFiles, err := os.ReadDir(filepath.Join(path, system))
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, gameFile := range gameFiles {
		if gameFile.IsDir() || filepath.Ext(gameFile.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(path, system, gameFile.Name()))
		if err != nil {
			return nil, err
		}
		var stored storedGame
		if err := json.Unmarshal(data, &stored); err != nil {
			return nil, err
		}
		entries = append(entries, Entry{System: system, Game: stored.Game, ManualFields: stored.ManualFields})
	}
	return entries, nil
}

// Save writes reg to the registry folder at path, as one JSON file per game
// inside its system's subfolder (named after the ROM's base name), creating
// folders as needed. Each file goes through writeFileAtomically, the same
// temporary-file-then-rename helper media copies already use, so a crash or a
// full disk partway through one game's write leaves that game's previous file
// exactly as it was rather than truncated.
//
// Only an entry that is new or differs from reg's saved baseline (see the
// Registry.saved field) is actually written — a single-game correction no
// longer costs one write per unrelated entry in the registry (backlog item
// 027). Once every changed entry has been written, the baseline is refreshed
// to match reg.Entries, so a later Save on the same *Registry diffs against
// what is now really on disk.
func Save(path string, reg *Registry) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}

	baseline := make(map[entryKey]Entry, len(reg.saved))
	for _, e := range reg.saved {
		baseline[keyOf(e)] = e
	}

	for _, e := range reg.Entries {
		if base, found := baseline[keyOf(e)]; found && base.Game == e.Game && slices.Equal(base.ManualFields, e.ManualFields) {
			continue
		}

		systemDir := filepath.Join(path, e.System)
		if err := os.MkdirAll(systemDir, 0o755); err != nil {
			return err
		}

		data, err := json.MarshalIndent(storedGame{Game: e.Game, ManualFields: e.ManualFields}, "", "  ")
		if err != nil {
			return err
		}
		if err := writeFileAtomically(filepath.Join(systemDir, gameFileName(e.Game)), bytes.NewReader(data)); err != nil {
			return err
		}
	}

	reg.saved = make([]Entry, len(reg.Entries))
	copy(reg.saved, reg.Entries)
	return nil
}

// entryKey identifies an entry the same way the registry deduplicates and
// stores it on disk: by system and game ID (see GameID). Save uses it to
// match a current entry against reg's saved baseline regardless of Entries
// order, which an import's appends or a removal can change.
type entryKey struct {
	system string
	id     string
}

// keyOf returns the entryKey identifying e.
func keyOf(e Entry) entryKey {
	return entryKey{system: e.System, id: GameID(e.Game.Path)}
}

// GameID returns the identifier used to deduplicate registry entries (see
// indexOf), name their JSON file on disk (see gameFileName), and address
// them from outside the package (see FindByID): a ROM path's base name,
// without its directory prefix or file extension. Two ROMs that would
// collide on that on-disk file — e.g. differing only by extension, or
// nested in different subfolders of the same system — must therefore also
// be treated as the same registry entry, or a later Save() would silently
// overwrite one with the other (see decisions/005, which established the
// same principle for the subfolder case).
func GameID(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// gameFileName derives the name of the JSON file storing g's metadata, from
// the base name of its ROM path.
func gameFileName(g gamelist.Game) string {
	return EntryFileName(GameID(g.Path))
}

// gameFields lists accessors for every value of a local game entry the
// registry can write into. Both write rules — filling the gaps and replacing
// what is there — walk this one table rather than each re-listing the twelve
// fields, so a field added to Game is never honoured by one and forgotten by
// the other. The ROM path is deliberately absent: it identifies the entry
// rather than describing it, and no send ever moves a game.
var gameFields = []func(*gamelist.Game) *string{
	func(g *gamelist.Game) *string { return &g.Name },
	func(g *gamelist.Game) *string { return &g.Desc },
	func(g *gamelist.Game) *string { return &g.Image },
	func(g *gamelist.Game) *string { return &g.Video },
	func(g *gamelist.Game) *string { return &g.Marquee },
	func(g *gamelist.Game) *string { return &g.Thumbnail },
	func(g *gamelist.Game) *string { return &g.Rating },
	func(g *gamelist.Game) *string { return &g.ReleaseDate },
	func(g *gamelist.Game) *string { return &g.Developer },
	func(g *gamelist.Game) *string { return &g.Publisher },
	func(g *gamelist.Game) *string { return &g.Genre },
	func(g *gamelist.Game) *string { return &g.Players },
}

// ErrGameNotFound is returned by Remove and RemoveByID when no entry matches
// the given system and ROM filename or game ID.
var ErrGameNotFound = errors.New("registry: game not found")

// ErrMediaLeftBehind marks a deletion that went through — the game's metadata
// file is gone from the registry folder, and its entry from the registry —
// while at least one of its media files could not be deleted. It is worth
// telling apart from a failed deletion: the game is really gone, and only
// unreferenced files remain, which no caller should report as "nothing was
// deleted" (see decisions/022).
var ErrMediaLeftBehind = errors.New("registry: media files left behind")

// Remove deletes the entry of system whose ROM filename is romFilename (e.g.
// "Sonic.zip" — any directory prefix is ignored, since the registry does not
// reproduce the ROM's original subfolder), as RemoveByID does for the game ID
// derived from it.
func Remove(reg *Registry, registryFolder, system, romFilename string) error {
	return RemoveByID(reg, registryFolder, system, GameID(romFilename))
}

// RemoveByID deletes, from the registry folder at registryFolder, the metadata
// file and every media file (cover art, video, marquee, thumbnail) belonging to
// the entry of system whose game ID (see GameID) is id, then removes that entry
// from reg. It is the deletion callers that already hold an identifier use:
// unlike Remove it does not run id through GameID, which would truncate an id
// whose game name contains a dot and either miss the entry or match another one.
//
// The metadata file is deleted first, and its deletion is what commits the
// whole operation: once it is gone, the entry leaves reg even if a medium
// resists, and the remaining media are still attempted rather than abandoned at
// the first failure — reported together as ErrMediaLeftBehind. The reverse
// order would leave a listed game with broken images. It returns
// ErrGameNotFound, without deleting anything, if no entry matches.
func RemoveByID(reg *Registry, registryFolder, system, id string) error {
	i := reg.indexOfID(system, id)
	if i == -1 {
		return ErrGameNotFound
	}
	g := reg.Entries[i].Game
	systemFolder := filepath.Join(registryFolder, system)

	if err := removeIfExists(filepath.Join(systemFolder, gameFileName(g))); err != nil {
		return err
	}
	reg.Entries = append(reg.Entries[:i], reg.Entries[i+1:]...)

	var leftBehind []string
	for _, kind := range mediaKinds {
		relPath := *kind.field(&g)
		if relPath == "" {
			continue
		}
		path, inside := mediumPath(systemFolder, relPath)
		if !inside {
			leftBehind = append(leftBehind, relPath)
			continue
		}
		if err := removeIfExists(path); err != nil {
			leftBehind = append(leftBehind, relPath)
		}
	}
	if len(leftBehind) > 0 {
		return fmt.Errorf("%w: %s", ErrMediaLeftBehind, strings.Join(leftBehind, ", "))
	}
	return nil
}

// mediumPath resolves a game's media reference against its system's folder in
// the registry, reporting whether it stays inside it. Media references come
// from scraped gamelist.xml files, so one of them reaching outside — with a
// "../.." prefix — must not make a deletion erase a file the registry does not
// own: os.Remove has no undo.
func mediumPath(systemFolder, relPath string) (path string, inside bool) {
	path = filepath.Join(systemFolder, filepath.FromSlash(relPath))
	if !strings.HasPrefix(path, systemFolder+string(filepath.Separator)) {
		return "", false
	}
	return path, true
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

// mergeGameEntry merges g (belonging to system) into the registry, reporting
// whether it was newly added, replaced an existing entry with different
// metadata, or left unchanged. Fields the existing entry marks as
// hand-edited keep their stored value, so a game still badly scraped in the
// ROMs folder cannot undo a correction; when they are the only difference,
// nothing is left to merge and the game counts as unchanged.
func (r *Registry) mergeGameEntry(system string, g gamelist.Game) importStatus {
	if i := r.indexOf(system, g.Path); i != -1 {
		keepHandEditedFields(&g, r.Entries[i].Game, r.Entries[i].ManualFields)
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
		switch r.mergeGameEntry(system, g) {
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

// indexOf finds the entry matching system and path's GameID — the registry
// is stored flat per system on disk, one JSON file per game ID, so two ROMs
// that share a game ID (e.g. differing only by subfolder or extension) are
// the same entry.
func (r *Registry) indexOf(system, path string) int {
	return r.indexOfID(system, GameID(path))
}

// indexOfID finds the entry matching system and the already-derived game
// ID id. Unlike indexOf it does not run id through GameID, which would
// truncate an id whose game name contains a dot (e.g. "Micro Machines
// v3.0") and lose the entry.
func (r *Registry) indexOfID(system, id string) int {
	for i, e := range r.Entries {
		if e.System == system && GameID(e.Game.Path) == id {
			return i
		}
	}
	return -1
}

// FindByID returns the entry of system whose game ID (see GameID) is id,
// reporting whether it was found. It is the lookup used by callers that
// address a game by its identifier rather than by a ROM path, such as the
// web UI's per-game URLs.
func (r *Registry) FindByID(system, id string) (Entry, bool) {
	i := r.indexOfID(system, id)
	if i == -1 {
		return Entry{}, false
	}
	return r.Entries[i], true
}

// ProgressEvent describes one game being processed by ImportFromRomsFolder,
// for callers that want to report progress to the user as the import runs.
type ProgressEvent struct {
	System    string
	GameIndex int // 1-based index of this game within System's game list
	GameCount int // total number of games found for System
	GameName  string
}

// ImportSummaryFormat words the outcome of an import: how many games the
// registry gained, how many it refreshed, and how many it already held as
// they are. Like CompletionSummaryFormat, it is a constant rather than each
// caller's own literal, so the update command and the web UI report the very
// same sentence for the very same operation.
const ImportSummaryFormat = "%d added, %d updated, %d unchanged"

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

		systemAdded, systemUpdated, systemUnchanged, systemErr := importSystemGames(reg, games, romsFolder, registryFolder, system, onProgress)
		added += systemAdded
		updated += systemUpdated
		unchanged += systemUnchanged
		if systemErr != nil {
			return added, updated, unchanged, systemErr
		}
	}

	return added, updated, unchanged, nil
}

// importSystemGames imports every game of one system's already-parsed local
// gamelist into reg, copying media for each newly added or changed game.
func importSystemGames(reg *Registry, games []gamelist.Game, romsFolder, registryFolder, system string, onProgress func(ProgressEvent)) (added, updated, unchanged int, err error) {
	for i, g := range games {
		if !hasScrapedData(g) {
			continue
		}

		if onProgress != nil {
			onProgress(ProgressEvent{System: system, GameIndex: i + 1, GameCount: len(games), GameName: g.Name})
		}

		status, copyErr := importOneGame(reg, romsFolder, registryFolder, system, g)
		switch status {
		case statusAdded:
			added++
		case statusUpdated:
			updated++
		default:
			unchanged++
			continue
		}
		if copyErr != nil {
			return added, updated, unchanged, copyErr
		}
	}
	return added, updated, unchanged, nil
}

// importOneGame merges g into reg and, if it was newly added or its metadata
// changed, copies its referenced media from romsFolder into registryFolder.
func importOneGame(reg *Registry, romsFolder, registryFolder, system string, g gamelist.Game) (status importStatus, copyErr error) {
	status = reg.mergeGameEntry(system, g)
	if status == statusUnchanged {
		return status, nil
	}
	return status, copyGameMedia(romsFolder, registryFolder, system, g)
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
func copyGameMedia(srcRoot, dstRoot, system string, g gamelist.Game) error {
	for _, kind := range mediaKinds {
		if err := copyMediaFile(srcRoot, dstRoot, system, *kind.field(&g)); err != nil {
			return err
		}
	}
	return nil
}

// copyMediaFile copies the media file at relPath (relative to the system
// folder, as referenced in gamelist.xml) from srcRoot to dstRoot. An empty
// relPath, or a referenced file missing on disk, is silently ignored.
func copyMediaFile(srcRoot, dstRoot, system, relPath string) error {
	_, err := writeMediaFile(srcRoot, dstRoot, system, relPath)
	return err
}

// writeMediaFile does the copying, and reports whether it actually wrote — what
// callers replacing a folder's media need in order to tell a run that changed
// something from one that found everything already right.
//
// A destination already holding the same bytes is left alone. The folders
// written here are the user's own, outside any version control, and a rewrite
// that changes nothing still costs them a file whose timestamp moved for no
// reason.
func writeMediaFile(srcRoot, dstRoot, system, relPath string) (written bool, err error) {
	if relPath == "" {
		return false, nil
	}

	data, err := os.ReadFile(filepath.Join(srcRoot, system, relPath))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	dst := filepath.Join(dstRoot, system, relPath)
	if current, readErr := os.ReadFile(dst); readErr == nil && bytes.Equal(current, data) {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return false, err
	}
	return true, nil
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

	i := findGameByFilename(games, romFilename)
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

	status, copyErr := importOneGame(reg, romsFolder, registryFolder, system, g)
	switch status {
	case statusAdded:
		return 1, 0, 0, copyErr
	case statusUpdated:
		return 0, 1, 0, copyErr
	default:
		return 0, 0, 1, nil
	}
}

// findGameByFilename returns the index in games of the entry whose ROM path
// has the given base filename (ignoring any directory prefix), or -1 if none
// matches.
func findGameByFilename(games []gamelist.Game, romFilename string) int {
	name := filepath.Base(romFilename)
	for i, g := range games {
		if filepath.Base(g.Path) == name {
			return i
		}
	}
	return -1
}

// CompletionSummaryFormat words the outcome of a completion: how many games
// were looked at, how many the registry could fill, and how many could not be
// written back. It is a constant rather than each caller's own literal so the
// CLI and the web UI report the very same sentence — the wording is part of
// what the two entry points share, like the operation itself.
const CompletionSummaryFormat = "%d processed, %d completed, %d failed"

// CompletionEvent describes one game being examined by CompleteRomsFolder,
// for callers that want to report progress to the user as it runs. It has
// the same shape as ProgressEvent (both describe a game's position within
// its system's local game list) but is kept as a distinct name to match the
// vocabulary of the completion flow it belongs to.
type CompletionEvent = ProgressEvent

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

		systemProcessed, systemCompleted, systemFailed, needsRewrite := completeSystemGames(reg, games, romsFolder, registryFolder, system, onProgress)
		processed += systemProcessed
		completed += systemCompleted
		failed += systemFailed

		if needsRewrite {
			if writeErr := gamelist.UpdateFile(gamelistPath, games); writeErr != nil {
				return processed, completed, failed, writeErr
			}
		}
	}

	return processed, completed, failed, nil
}

// completeSystemGames fills the gaps of one system's already-parsed local
// games from reg, copying any newly referenced media for each game changed.
// needsRewrite reports whether any game was changed, so the caller knows
// whether games must be written back to the local gamelist.xml.
func completeSystemGames(reg *Registry, games []gamelist.Game, romsFolder, registryFolder, system string, onProgress func(CompletionEvent)) (processed, completed, failed int, needsRewrite bool) {
	for i := range games {
		processed++

		found, changed, before := fillGameFromRegistry(reg, games, i, system)
		if !found || !changed {
			continue
		}
		needsRewrite = true

		if onProgress != nil {
			onProgress(CompletionEvent{System: system, GameIndex: i + 1, GameCount: len(games), GameName: games[i].Name})
		}

		if copyErr := copyFilledMedia(before, games[i], registryFolder, romsFolder, system); copyErr != nil {
			failed++
			continue
		}
		completed++
	}
	return processed, completed, failed, needsRewrite
}

// mergeGame fills any empty field of dst with the corresponding non-empty
// field of src, reporting whether anything was filled.
func mergeGame(dst *gamelist.Game, src gamelist.Game) bool {
	changed := false
	for _, field := range gameFields {
		d, s := field(dst), *field(&src)
		if *d == "" && s != "" {
			*d = s
			changed = true
		}
	}
	return changed
}

// overwriteGame writes every value src holds over dst's, reporting whether any
// of them actually changed. A field src holds empty leaves dst's alone: the
// registry not knowing something is no reason to make the user lose what their
// own folder holds (see decisions/030). It is the exact opposite of mergeGame,
// and is what a Replacement rests on rather than a Completion.
func overwriteGame(dst *gamelist.Game, src gamelist.Game) bool {
	changed := false
	for _, field := range gameFields {
		d, s := field(dst), *field(&src)
		if s != "" && *d != s {
			*d = s
			changed = true
		}
	}
	return changed
}

// fillGameFromRegistry merges the registry's known metadata for games[i] (matched
// by system and ROM path) into it, filling only empty fields. It reports
// whether a matching registry entry was found and whether anything was
// filled; before holds games[i]'s value prior to merging, for callers that
// need to detect which media files were newly referenced.
func fillGameFromRegistry(reg *Registry, games []gamelist.Game, i int, system string) (found, changed bool, before gamelist.Game) {
	before = games[i]
	j := reg.indexOf(system, games[i].Path)
	if j == -1 {
		return false, false, before
	}
	return true, mergeGame(&games[i], reg.Entries[j].Game), before
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
	return sendGame(reg, romsFolder, registryFolder, system, romFilename, fillGaps, onProgress)
}

// ReplaceGame writes what reg knows about a single local game entry over what
// the folder holds, instead of only filling its gaps: it is the deliberate
// opposite of CompleteGame, and it is a separate operation rather than a mode
// of it because the two mean opposite things about the user's own files (see
// decisions/030). The game is identified the same way, and the same results
// are reported — replaced tells whether anything actually changed, failed
// whether writing a medium then failed with the gamelist.xml still rewritten,
// and ErrGameNotFound whether the folder or the registry does not hold it.
//
// Two rules bound what it may destroy: a field reg holds empty never blanks
// the folder's value, and a medium is written only when the file already there
// is missing or holds different bytes — so a repeat replacement rewrites
// nothing and honestly reports that nothing changed.
func ReplaceGame(reg *Registry, romsFolder, registryFolder, system, romFilename string, onProgress func(CompletionEvent)) (replaced, failed bool, err error) {
	return sendGame(reg, romsFolder, registryFolder, system, romFilename, overwrite, onProgress)
}

// writeRule is how one registry entry is written into a local game entry: it
// mutates dst, copies from srcRoot into dstRoot whatever media doing so brought
// in, and reports whether anything changed at all — in dst or on disk. It is
// the single parameter telling CompleteGame's flow from ReplaceGame's, which
// are otherwise identical step for step.
type writeRule func(dst *gamelist.Game, src gamelist.Game, srcRoot, dstRoot, system string) (changed bool, copyErr error)

// fillGaps is a Completion's rule: only the empty fields of dst are filled, and
// only the media those newly filled references brought in are copied.
func fillGaps(dst *gamelist.Game, src gamelist.Game, srcRoot, dstRoot, system string) (bool, error) {
	before := *dst
	if !mergeGame(dst, src) {
		return false, nil
	}
	return true, copyFilledMedia(before, *dst, srcRoot, dstRoot, system)
}

// overwrite is a Replacement's rule: every value src holds wins over dst's, and
// every medium dst then refers to is written into dstRoot unless the file
// already there is identical. A folder whose values all already match can still
// be changed by this — its cover art file may differ while its reference does
// not — which is why the copy's own verdict counts as a change.
func overwrite(dst *gamelist.Game, src gamelist.Game, srcRoot, dstRoot, system string) (bool, error) {
	changed := overwriteGame(dst, src)
	copied, err := copyEveryMedium(*dst, srcRoot, dstRoot, system)
	return changed || copied, err
}

// sendGame is what CompleteGame and ReplaceGame both do: find the game in the
// folder's own gamelist.xml, write reg's knowledge of it into that entry under
// the rule given, and rewrite the file. The media are copied before the
// rewrite, so a medium that could not be written still leaves the metadata
// filled — a per-game failure, never a fatal one.
func sendGame(reg *Registry, romsFolder, registryFolder, system, romFilename string, write writeRule, onProgress func(CompletionEvent)) (changed, failed bool, err error) {
	gamelistPath := filepath.Join(romsFolder, system, "gamelist.xml")
	games, parseErr := gamelist.ParseFile(gamelistPath)
	if parseErr != nil {
		return false, false, ErrGameNotFound
	}

	i := findGameByFilename(games, romFilename)
	if i == -1 {
		return false, false, ErrGameNotFound
	}

	j := reg.indexOf(system, games[i].Path)
	if j == -1 {
		return false, false, ErrGameNotFound
	}

	written, copyErr := write(&games[i], reg.Entries[j].Game, registryFolder, romsFolder, system)
	if !written {
		// Nothing to rewrite the file for. A media failure is still reported: it
		// is the reason nothing changed, not a sign that nothing needed to.
		return false, copyErr != nil, nil
	}

	if onProgress != nil {
		onProgress(CompletionEvent{System: system, GameIndex: i + 1, GameCount: len(games), GameName: games[i].Name})
	}

	if writeErr := gamelist.UpdateFile(gamelistPath, games); writeErr != nil {
		return false, false, writeErr
	}

	if copyErr != nil {
		return false, true, nil
	}
	return true, false, nil
}

// copyEveryMedium writes, from srcRoot into dstRoot, every medium g refers to,
// over whatever sits there, and reports whether any file was actually written.
// It is a Replacement's counterpart to copyFilledMedia: the reference alone
// cannot tell whether the folder holds the right file, since the same relative
// path names a different image in each of the two folders.
//
// A reference srcRoot holds no file for copies nothing, which is not a failure
// — a gamelist.xml may well name media that were never scraped.
func copyEveryMedium(g gamelist.Game, srcRoot, dstRoot, system string) (copied bool, err error) {
	for _, kind := range mediaKinds {
		written, err := writeMediaFile(srcRoot, dstRoot, system, *kind.field(&g))
		if err != nil {
			return copied, err
		}
		copied = copied || written
	}
	return copied, nil
}

// copyFilledMedia copies, from srcRoot into dstRoot, every media file whose
// reference was newly filled between before and after (i.e. empty in
// before, non-empty in after).
func copyFilledMedia(before, after gamelist.Game, srcRoot, dstRoot, system string) error {
	for _, kind := range mediaKinds {
		b, a := *kind.field(&before), *kind.field(&after)
		if b == "" && a != "" {
			if err := copyMediaFile(srcRoot, dstRoot, system, a); err != nil {
				return err
			}
		}
	}
	return nil
}
