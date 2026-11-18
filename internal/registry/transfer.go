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
	// ErrMediaNotCopied matches a *MediaNotCopiedError: at least one medium
	// file of a transfer could not be copied.
	ErrMediaNotCopied = errors.New("registry: a medium could not be copied")
)

// The step a medium copy failed at.
const (
	StepRead         = "read"
	StepCreateFolder = "create-folder"
	StepWrite        = "write"
)

// MediaCopyError is one medium file that could not be copied: which step
// failed, and the operating system's own error, which names the file and
// the cause. Medium is set by a transfer, which knows which of the game's
// four media the file was.
type MediaCopyError struct {
	Medium Medium
	Step   string
	Err    error
}

func (e *MediaCopyError) Error() string {
	return fmt.Sprintf("registry: %s medium (step %s): %v", e.Medium, e.Step, e.Err)
}

func (e *MediaCopyError) Unwrap() error { return e.Err }

// MediaNotCopiedError is a transfer where at least one medium file could not
// be copied. Every value that could be written was: Copied names them, so
// the failures are all that is left to fix. A medium that failed is not
// referenced by its destination entry.
type MediaNotCopiedError struct {
	Copied   []string
	Failures []MediaCopyError
}

func (e *MediaNotCopiedError) Error() string {
	return fmt.Sprintf("registry: %d medium file(s) could not be copied: %v", len(e.Failures), e.Failures[0].Err)
}

// Is makes errors.Is(err, ErrMediaNotCopied) true.
func (e *MediaNotCopiedError) Is(target error) bool { return target == ErrMediaNotCopied }

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
		notCopied := &MediaNotCopiedError{}
		for _, gf := range gameFields {
			value := *gf.field(&src)
			if value == "" || !slices.Contains(fields, gf.id) {
				continue
			}
			if m, isMedium := LookupMedium(gf.id); isMedium {
				written, err := writeMediaFile(srcRoot, dstRoot, system, value)
				var copyErr *MediaCopyError
				if errors.As(err, &copyErr) {
					notCopied.Failures = append(notCopied.Failures, MediaCopyError{Medium: m, Step: copyErr.Step, Err: copyErr.Err})
					continue
				}
				changed = changed || written
			}
			if *gf.field(dst) != value {
				*gf.field(dst) = value
				changed = true
			}
			notCopied.Copied = append(notCopied.Copied, gf.id)
		}
		if len(notCopied.Failures) > 0 {
			return changed, notCopied
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
		return added, err
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
	write := pickFields(fields)
	var copyErr error
	var wrote bool
	_, added, _, err = sendGame(reg, romsFolder, registryFolder, system, rom, func(dst *gamelist.Game, src gamelist.Game, srcRoot, dstRoot, system string) (bool, error) {
		written, err := write(dst, src, srcRoot, dstRoot, system)
		wrote, copyErr = written, err
		return written, err
	}, nil)
	if err != nil {
		return false, false, err
	}
	return wrote, added, copyErr
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

// Differences lists, in TransferFields order, the values source holds that a
// transfer would change in destination: a metadata value destination lacks
// or holds differently, or a medium whose file source has and destination
// either references elsewhere or holds at a different size. A value source
// does not hold, or a medium source references without its file, is never
// one — there is nothing to copy. This is the one rule deciding both whether
// a transfer is offered and which of its values a review calls "Same".
//
// shortcut: two media files are compared by size, not by bytes — reading
// every file of a page of games at each display would be slow on a NAS; two
// images of the same size and different content are reported identical, and
// the copy itself still compares bytes before writing.
func Differences(source, destination gamelist.Game, srcRoot, dstRoot, system string) []string {
	var differ []string
	for _, gf := range gameFields {
		value := *gf.field(&source)
		if value == "" {
			continue
		}
		current := *gf.field(&destination)
		if _, isMedium := LookupMedium(gf.id); isMedium {
			if mediumDiffers(value, current, srcRoot, dstRoot, system) {
				differ = append(differ, gf.id)
			}
			continue
		}
		if value != current {
			differ = append(differ, gf.id)
		}
	}
	return differ
}

// mediumDiffers reports whether copying the medium source references from
// srcRoot would change what destination references in dstRoot.
func mediumDiffers(source, destination, srcRoot, dstRoot, system string) bool {
	srcPath, inside := mediumPath(filepath.Join(srcRoot, system), source)
	if !inside {
		return false
	}
	srcInfo, err := os.Stat(srcPath)
	if err != nil || srcInfo.IsDir() {
		return false
	}
	if filepath.Clean(source) != filepath.Clean(destination) {
		return true
	}
	dstPath, inside := mediumPath(filepath.Join(dstRoot, system), destination)
	if !inside {
		return true
	}
	dstInfo, err := os.Stat(dstPath)
	return err != nil || dstInfo.Size() != srcInfo.Size()
}
