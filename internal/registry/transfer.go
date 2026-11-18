package registry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
)

// The ways a field-by-field transfer is refused before it writes anything.
var (
	// ErrNothingToTransfer is a transfer naming no field at all.
	ErrNothingToTransfer = errors.New("registry: no field chosen to transfer")
	// ErrUnsafeMediaPath is a medium reference pointing outside its system
	// folder, which no transfer follows.
	ErrUnsafeMediaPath = errors.New("registry: medium reference outside its system folder")
	// ErrMediaNotCopied is a send whose values were written but one of whose
	// media files could not be copied.
	ErrMediaNotCopied = errors.New("registry: a medium could not be copied")
)

// validSystemName reports whether system names a folder directly inside a
// ROMs folder, never a path reaching elsewhere.
func validSystemName(system string) bool {
	return system != "" && system != "." && system != ".." && !strings.ContainsAny(system, `/\`)
}

// LocalGame returns the entry romsFolder's own gamelist.xml holds for rom of
// system, or ErrGameNotFound when there is none — including when the file is
// missing or unreadable, or system is not a plain folder name.
func LocalGame(romsFolder, system, rom string) (gamelist.Game, error) {
	if !validSystemName(system) {
		return gamelist.Game{}, ErrGameNotFound
	}
	games, err := gamelist.ParseFile(filepath.Join(romsFolder, system, "gamelist.xml"))
	if err != nil {
		return gamelist.Game{}, ErrGameNotFound
	}
	i := findGameByFilename(games, rom)
	if i == -1 {
		return gamelist.Game{}, ErrGameNotFound
	}
	return games[i], nil
}

// pickFields is the write rule of a transfer the user chose value by value:
// every field named in fields that src holds overwrites dst's, and every such
// medium is copied from srcRoot into dstRoot. Fields src leaves empty are
// skipped: a transfer never erases. Every medium reference is checked to stay
// inside its system folder, on both sides, before anything changes.
func pickFields(fields []string) writeRule {
	return func(dst *gamelist.Game, src gamelist.Game, srcRoot, dstRoot, system string) (bool, error) {
		for _, gf := range gameFields {
			value := *gf.field(&src)
			if value == "" || !slices.Contains(fields, gf.id) {
				continue
			}
			if _, isMedium := LookupMedium(gf.id); !isMedium {
				continue
			}
			_, srcInside := mediumPath(filepath.Join(srcRoot, system), value)
			_, dstInside := mediumPath(filepath.Join(dstRoot, system), value)
			if !srcInside || !dstInside {
				return false, ErrUnsafeMediaPath
			}
		}

		changed := false
		for _, gf := range gameFields {
			value := *gf.field(&src)
			if value == "" || !slices.Contains(fields, gf.id) {
				continue
			}
			if _, isMedium := LookupMedium(gf.id); isMedium {
				written, err := writeMediaFile(srcRoot, dstRoot, system, value)
				if err != nil {
					return changed, err
				}
				changed = changed || written
			}
			if *gf.field(dst) != value {
				*gf.field(dst) = value
				changed = true
			}
		}
		return changed, nil
	}
}

// RetrieveFields copies the values named in fields from romsFolder's local
// entry for rom into reg's entry, creating that entry when the registry does
// not know the game yet (added). Hand-edited marks are left as they are: the
// user chose each value. Media files are copied into registryFolder, keeping
// their relative path.
func RetrieveFields(reg *Registry, romsFolder, registryFolder, system, rom string, fields []string) (added bool, err error) {
	if len(fields) == 0 {
		return false, ErrNothingToTransfer
	}
	local, err := LocalGame(romsFolder, system, rom)
	if err != nil {
		return false, err
	}

	j := reg.indexOf(system, rom)
	if j == -1 {
		reg.mergeGameEntry(system, gamelist.Game{Path: local.Path})
		j = reg.indexOf(system, rom)
		added = true
	}
	if _, err := pickFields(fields)(&reg.Entries[j].Game, local, romsFolder, registryFolder, system); err != nil {
		return added, fmt.Errorf("retrieve %s: %w", rom, err)
	}
	return added, nil
}

// SendFields writes the values named in fields from reg's entry for rom into
// romsFolder's gamelist.xml, creating the local entry when the ROM file is
// present but not listed yet (added, see decisions/037). Media files are
// copied into romsFolder, keeping their relative path.
func SendFields(reg *Registry, romsFolder, registryFolder, system, rom string, fields []string) (changed, added bool, err error) {
	if len(fields) == 0 {
		return false, false, ErrNothingToTransfer
	}
	if !validSystemName(system) {
		return false, false, ErrGameNotFound
	}
	if _, statErr := os.Stat(filepath.Join(romsFolder, system)); statErr != nil {
		return false, false, ErrGameNotFound
	}
	changed, added, failed, err := sendGame(reg, romsFolder, registryFolder, system, rom, pickFields(fields), nil)
	if err != nil {
		return false, false, err
	}
	if failed {
		return changed, added, ErrMediaNotCopied
	}
	return changed, added, nil
}

// TransferFields lists the identifiers of every value a transfer may copy,
// metadata and media alike, in the order a review shows them.
func TransferFields() []string {
	ids := make([]string, len(gameFields))
	for i, gf := range gameFields {
		ids[i] = gf.id
	}
	return ids
}

// FieldValue returns g's value for the field or medium id, or nothing for an
// unknown id.
func FieldValue(g gamelist.Game, id string) string {
	for _, gf := range gameFields {
		if gf.id == id {
			return *gf.field(&g)
		}
	}
	return ""
}
