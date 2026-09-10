package registry

import (
	"errors"
	"os"
	"path/filepath"
	"sort"

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
	Gaps    []GameGap
}

// GameGap is one ROM file that is not, locally, everything it could be: what
// names it, whether it has a local gamelist.xml entry at all, which of the
// twelve fields are still missing, and which of those the registry already
// holds — the distinction the whole report exists for.
type GameGap struct {
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
}

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
		if de.IsDir() || de.Name() == "gamelist.xml" {
			continue
		}
		romFiles = append(romFiles, de.Name())
	}

	games, parseErr := gamelist.ParseFile(filepath.Join(systemFolder, "gamelist.xml"))
	if parseErr != nil {
		if !errors.Is(parseErr, os.ErrNotExist) {
			return SystemStatus{System: system, Problem: "This system's gamelist.xml could not be read."}
		}
		games = nil
	}

	st := SystemStatus{System: system, ROMCount: len(romFiles)}

	listed := make(map[string]bool, len(games))
	for _, g := range games {
		name := filepath.Base(g.Path)
		listed[name] = true
		st.addGap(name, g, g.Name, true, registryGameOf(reg, system, g.Path))
	}

	for _, file := range romFiles {
		if listed[file] {
			continue
		}
		st.addGap(file, gamelist.Game{}, "", false, registryGameOf(reg, system, file))
	}

	sort.Slice(st.Gaps, func(i, j int) bool {
		return gapSortKey(st.Gaps[i]) < gapSortKey(st.Gaps[j])
	})

	return st
}

// addGap classifies one ROM (local, its local entry — a zero value when it
// has none) against the matching registry entry, if any, updating the
// system's counts and, when it is not already complete, appending its gap.
func (st *SystemStatus) addGap(romFilename string, local gamelist.Game, localName string, listedLocally bool, registryGame *gamelist.Game) {
	missing, fillable := gapFields(local, registryGame)
	if len(missing) == 0 {
		st.CompleteCount++
		return
	}

	if listedLocally {
		st.IncompleteCount++
	} else {
		st.NotListedCount++
	}
	if len(fillable) > 0 {
		st.FillableCount++
	}

	name := localName
	if name == "" && registryGame != nil {
		name = registryGame.Name
	}
	st.Gaps = append(st.Gaps, GameGap{
		ROMFilename: romFilename,
		Name:        name,
		Listed:      listedLocally,
		Missing:     missing,
		Fillable:    fillable,
	})
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
func gapSortKey(g GameGap) string {
	if g.Name != "" {
		return g.Name
	}
	return g.ROMFilename
}
