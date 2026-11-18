package registry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
)

// FolderStatus is a read-only report of one ROMs folder: for every system it
// holds, how much of it is still missing compared to what a Completion could
// already fill from reg. It writes nothing, neither to the ROMs folder nor to
// the registry — the counterpart of CompleteRomsFolder and
// ImportFromRomsFolder that only ever reads.
type FolderStatus struct {
	RomsFolder string
	Systems    []SystemStatus
}

// SystemStatus is one system's share of a FolderStatus: how many ROM files it
// holds, how many are fully scraped locally, and the list of the ones that
// are not. Problem is set instead, leaving every count at zero, when the
// system's own gamelist.xml exists but could not be read — a system this bad
// still deserves to be named rather than silently dropped from the report.
type SystemStatus struct {
	System string
	// ROMCount is every file found sitting directly in the system's folder,
	// gamelist.xml and any subfolder (images, videos, ...) excluded — see
	// decisions/038 for why no extension is checked.
	ROMCount int
	// CompleteCount is how many local gamelist.xml entries already hold every
	// one of the twelve fields gameFields walks.
	CompleteCount int
	// IncompleteCount is how many local entries are missing at least one of
	// them.
	IncompleteCount int
	// NotListedCount is how many ROM files have no local entry at all.
	NotListedCount int
	// FillableCount is how many of the incomplete or not-listed games the
	// registry already holds enough to fill, in whole or in part, were a
	// Completion run against this folder right now.
	FillableCount int
	// Problem names why this system's status could not be determined at all
	// (its gamelist.xml exists but could not be parsed, or its folder could
	// not be read), leaving every count above at zero.
	Problem string
	// Games is every visible game of the system, complete ones included,
	// sorted by name.
	Games []GameStatus
}

// GameStatus is one ROM file of a system: what names it, whether it has a
// local gamelist.xml entry at all, which of the twelve fields are still
// missing (none for a complete game), and which of those the registry already
// holds — the distinction the whole report exists for.
type GameStatus struct {
	// ROMFilename is the file's base name, as found on disk or as its local
	// entry's path names it.
	ROMFilename string
	// Name is the best name known for it: the local entry's if it has one,
	// otherwise the registry's, otherwise empty.
	Name string
	// Listed reports whether the local gamelist.xml has an entry for this ROM
	// at all. false means every one of the twelve fields is missing.
	Listed bool
	// Missing names the fields (registry.Field* and registry.Medium values)
	// still empty locally.
	Missing []string
	// Fillable is the subset of Missing the registry already holds a value
	// for — what a Completion run would fill right now, as opposed to a gap
	// the registry does not know how to close either.
	Fillable []string
	// RegistryID is the identifier of the matching registry entry, when one
	// exists — what a caller builds a link to that entry's own page from, so
	// the scrape can be finished by hand right there. Empty when the registry
	// does not know this game at all.
	RegistryID string
	// LocalImage is the cover art reference of the local entry, relative to
	// the system's folder as gamelist.xml writes it; empty when none.
	LocalImage string
	// RegistryImage is the cover art reference of the matching registry
	// entry, relative to its system's folder in the registry; empty when none.
	RegistryImage string
}

// Complete reports whether nothing is missing locally.
func (g GameStatus) Complete() bool {
	return len(g.Missing) == 0
}

// infoTxtFileName is a note some Batocera setups leave directly in a system's
// ROM folder — never a ROM, so the flat-file scan behind ROMCount and the
// not-listed detection must not count it as one.
const infoTxtFileName = "_info.txt"

// RomsFolderStatus reports, for every system subfolder of romsFolder, what
// its local gamelist.xml is still missing compared to what CompleteRomsFolder
// could already fill from reg. A system holding neither a ROM file nor a
// local entry has nothing to report and is left out of Systems entirely.
func RomsFolderStatus(reg *Registry, romsFolder string) (FolderStatus, error) {
	dirEntries, err := os.ReadDir(romsFolder)
	if err != nil {
		return FolderStatus{}, err
	}

	var systems []SystemStatus
	for _, de := range dirEntries {
		if !de.IsDir() {
			continue
		}
		st := systemStatus(reg, romsFolder, de.Name())
		if st.Problem == "" && st.ROMCount == 0 &&
			st.CompleteCount == 0 && st.IncompleteCount == 0 && st.NotListedCount == 0 {
			continue
		}
		systems = append(systems, st)
	}

	// The system with the most gaps first, so a folder spanning many systems
	// says where the problems are instead of leaving that to an alphabetical
	// scan.
	sort.Slice(systems, func(i, j int) bool {
		pi := systems[i].IncompleteCount + systems[i].NotListedCount
		pj := systems[j].IncompleteCount + systems[j].NotListedCount
		if pi != pj {
			return pi > pj
		}
		return systems[i].System < systems[j].System
	})

	return FolderStatus{RomsFolder: romsFolder, Systems: systems}, nil
}

// systemStatus reports one system's share of a FolderStatus. A folder this
// system's own subfolder could not be read from, or a gamelist.xml that
// exists but could not be parsed, is named in Problem rather than aborting
// the whole report — one bad system must not hide the rest of the folder.
func systemStatus(reg *Registry, romsFolder, system string) SystemStatus {
	systemFolder := filepath.Join(romsFolder, system)

	dirEntries, err := os.ReadDir(systemFolder)
	if err != nil {
		return SystemStatus{System: system, Problem: "This system's folder could not be read."}
	}

	var romFiles []string
	for _, de := range dirEntries {
		if de.IsDir() || de.Name() == "gamelist.xml" || de.Name() == infoTxtFileName {
			continue
		}
		romFiles = append(romFiles, de.Name())
	}

	games, hidden, parseErr := gamelist.ParseWithVisibility(filepath.Join(systemFolder, "gamelist.xml"))
	if parseErr != nil {
		if !errors.Is(parseErr, os.ErrNotExist) {
			return SystemStatus{System: system, Problem: "This system's gamelist.xml could not be read."}
		}
		games, hidden = nil, nil
	}

	// A game EmulationStation itself marks <hidden> is left out of the report
	// entirely — not just out of the gap list, but out of ROMCount too, exactly
	// as it is left out of Batocera's own list. Its ROM file is dropped from
	// romFiles here, before ROMCount is ever read off its length.
	hiddenNames := make(map[string]bool)
	listed := make(map[string]bool, len(games))
	for i, g := range games {
		name := filepath.Base(g.Path)
		if hidden[i] {
			hiddenNames[name] = true
			continue
		}
		listed[name] = true
	}
	var visible []string
	for _, file := range romFiles {
		if !hiddenNames[file] {
			visible = append(visible, file)
		}
	}
	romFiles = visible

	st := SystemStatus{System: system, ROMCount: len(romFiles)}

	for i, g := range games {
		if hidden[i] {
			continue
		}
		st.addGap(filepath.Base(g.Path), g, g.Name, true, registryGameOf(reg, system, g.Path))
	}

	for _, file := range romFiles {
		if listed[file] {
			continue
		}
		st.addGap(file, gamelist.Game{}, "", false, registryGameOf(reg, system, file))
	}

	sort.Slice(st.Games, func(i, j int) bool {
		return gapSortKey(st.Games[i]) < gapSortKey(st.Games[j])
	})

	return st
}

// addGap classifies one ROM (local, its local entry — a zero value when it
// has none) against the matching registry entry, if any, updating the
// system's counts and appending its status.
func (st *SystemStatus) addGap(romFilename string, local gamelist.Game, localName string, listedLocally bool, registryGame *gamelist.Game) {
	missing, fillable := gapFields(local, registryGame)
	switch {
	case len(missing) == 0:
		st.CompleteCount++
	case listedLocally:
		st.IncompleteCount++
	default:
		st.NotListedCount++
	}
	if len(fillable) > 0 {
		st.FillableCount++
	}

	name := localName
	registryID := ""
	registryImage := ""
	if registryGame != nil {
		if name == "" {
			name = registryGame.Name
		}
		registryID = GameID(registryGame.Path)
		registryImage = registryGame.Image
	}
	st.Games = append(st.Games, GameStatus{
		ROMFilename:   romFilename,
		Name:          name,
		Listed:        listedLocally,
		Missing:       missing,
		Fillable:      fillable,
		RegistryID:    registryID,
		LocalImage:    local.Image,
		RegistryImage: registryImage,
	})
}

// RomsFolderSystemStatus reports one system of romsFolder alone, without
// reading any other system. A system that is not a plain folder name directly
// inside romsFolder answers an error wrapping os.ErrNotExist — a request
// naming "../elsewhere" is as unknown as one naming a missing system.
func RomsFolderSystemStatus(reg *Registry, romsFolder, system string) (SystemStatus, error) {
	if system == "" || system == "." || system == ".." || strings.ContainsAny(system, `/\`) {
		return SystemStatus{}, fmt.Errorf("system %q: %w", system, os.ErrNotExist)
	}
	info, err := os.Stat(filepath.Join(romsFolder, system))
	if err != nil {
		return SystemStatus{}, err
	}
	if !info.IsDir() {
		return SystemStatus{}, fmt.Errorf("system %q: %w", system, os.ErrNotExist)
	}
	return systemStatus(reg, romsFolder, system), nil
}

// RomsFolderSystems lists, sorted, the system folders of romsFolder holding
// at least one file — a cheap listing, reading no gamelist.xml, for
// navigating between systems without computing every system's status.
func RomsFolderSystems(romsFolder string) ([]string, error) {
	dirEntries, err := os.ReadDir(romsFolder)
	if err != nil {
		return nil, err
	}
	var systems []string
	for _, de := range dirEntries {
		if !de.IsDir() {
			continue
		}
		if holdsAFile(filepath.Join(romsFolder, de.Name())) {
			systems = append(systems, de.Name())
		}
	}
	return systems, nil
}

// holdsAFile reports whether folder directly holds at least one file other
// than _info.txt; an unreadable folder holds nothing worth navigating to.
func holdsAFile(folder string) bool {
	dirEntries, err := os.ReadDir(folder)
	if err != nil {
		return false
	}
	for _, de := range dirEntries {
		if !de.IsDir() && de.Name() != infoTxtFileName {
			return true
		}
	}
	return false
}

// registryGameOf looks up the registry entry matching system and romFilename
// (through the same rule every other registry lookup uses, see GameID),
// returning nil rather than a zero value when there is none — gapFields tells
// "the registry has nothing" apart from "the registry's own field is empty"
// exactly the way CompleteGame's fillGaps rule already does.
func registryGameOf(reg *Registry, system, romFilename string) *gamelist.Game {
	if i := reg.indexOf(system, romFilename); i != -1 {
		return &reg.Entries[i].Game
	}
	return nil
}

// gapFields reports, against local (a zero value standing for "no local
// entry at all") and the matching registry entry when one exists, which of
// the twelve fields gameFields walks are still empty locally, and which of
// those the registry already holds a value for — the same test mergeGame
// applies when filling gaps for real, computed here without writing
// anything.
func gapFields(local gamelist.Game, registryGame *gamelist.Game) (missing, fillable []string) {
	for _, gf := range gameFields {
		if *gf.field(&local) != "" {
			continue
		}
		missing = append(missing, gf.id)
		if registryGame != nil && *gf.field(registryGame) != "" {
			fillable = append(fillable, gf.id)
		}
	}
	return missing, fillable
}

// gapSortKey orders a system's gaps by the best name known for them, falling
// back to the ROM filename for the rare case neither the local entry nor the
// registry names it.
func gapSortKey(g GameStatus) string {
	if g.Name != "" {
		return g.Name
	}
	return g.ROMFilename
}
