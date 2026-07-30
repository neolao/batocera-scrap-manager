package registry

import (
	"errors"
	"path"
	"path/filepath"
	"strings"
)

// maxEntryFileName is the longest name most filesystems accept for a single
// file. A game ID longer than that, once suffixed with ".json", could not be
// written at all, so it is refused while the registry can still say why —
// rather than at Save time, where the failure would read as "the registry
// folder could not be written to".
const maxEntryFileName = 255

// The reasons a submitted ROM path is refused. They are told apart because a
// caller shows one message per reason: a single "invalid path" would leave the
// user guessing which rule was broken.
var (
	// ErrEmptyPath marks a path holding nothing but blanks. A game has to be
	// identified by something.
	ErrEmptyPath = errors.New("registry: the ROM path is empty")

	// ErrAbsolutePath marks a path anchored at the filesystem root. Paths are
	// stored relative to the system's folder, as gamelist.xml writes them.
	ErrAbsolutePath = errors.New("registry: the ROM path is absolute")

	// ErrEscapingPath marks a path leaving the system's folder through "..".
	// The registry only ever names files it owns.
	ErrEscapingPath = errors.New("registry: the ROM path escapes the system folder")

	// ErrUnusablePath marks a path that breaks no rule above yet names no file
	// the registry could store: a folder, an extension with nothing before it,
	// a Windows separator, a control character, or a name too long to write.
	ErrUnusablePath = errors.New("registry: the ROM path names no usable file")

	// ErrDuplicateGameID marks a path whose identifier (see GameID) already
	// belongs to another game of the same system. Two entries sharing one
	// identifier would collide on the same JSON file, and Save would silently
	// keep only the last one written.
	ErrDuplicateGameID = errors.New("registry: another game of this system already uses this identifier")
)

// ValidatePath reports whether romPath is storable as a game's ROM path,
// naming the rule it breaks otherwise. It is exported so a caller can refuse a
// submission — with its own wording — before building the candidate registry
// ChangePath would apply it to.
func ValidatePath(romPath string) error {
	if strings.TrimSpace(romPath) == "" {
		return ErrEmptyPath
	}
	if strings.HasPrefix(romPath, "/") {
		return ErrAbsolutePath
	}

	// A backslash is a separator on Windows and an ordinary character
	// everywhere else, so a path carrying one would name a different file
	// depending on where the tool runs. Batocera writes "/" — that is the one
	// form stored.
	if strings.ContainsRune(romPath, '\\') {
		return ErrUnusablePath
	}
	for _, r := range romPath {
		if r < 0x20 || r == 0x7f {
			return ErrUnusablePath
		}
	}

	cleaned := path.Clean(romPath)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return ErrEscapingPath
	}
	// The last segment is read before cleaning: "disc1/", "disc1/." and
	// "disc1/.." all name a folder, but Clean turns each of them back into a
	// perfectly ordinary "disc1".
	segments := strings.Split(romPath, "/")
	switch segments[len(segments)-1] {
	case "", ".", "..":
		return ErrUnusablePath
	}

	// What is left resolves inside the system folder but may still name no
	// file: ".zip" yields an identifier that could name no JSON file, nor be
	// routed back to as a URL.
	id := GameID(cleaned)
	if id == "" || id == "." || id == ".." {
		return ErrUnusablePath
	}
	if len(EntryFileName(id)) > maxEntryFileName {
		return ErrUnusablePath
	}
	return nil
}

// ChangePath stores newPath as the ROM path of the entry of system whose game
// ID is id — which moves the entry under the identifier newPath derives (see
// GameID), and with it the JSON file naming it on disk and the URL addressing
// it. The path is stored exactly as given: it is what gamelist.xml holds, and
// cleaning it here would rewrite a value the user did not change.
//
// It changes nothing in the registry folder: the caller applies it to a
// Clone(), writes that clone, and only then removes the file left under the
// old identifier (see RemoveGameFile and decisions/024). Nothing is marked as
// hand-edited — the path identifies the entry rather than describing it, so
// the protection that shields metadata from later imports does not apply.
//
// It returns ErrGameNotFound if no entry matches, ErrDuplicateGameID if the
// identifier already belongs to another game of the same system, or whatever
// ValidatePath refuses — in every case without modifying anything.
func ChangePath(reg *Registry, system, id, newPath string) error {
	i := reg.indexOfID(system, id)
	if i == -1 {
		return ErrGameNotFound
	}
	if err := ValidatePath(newPath); err != nil {
		return err
	}

	// The comparison is by position, not by identifier: an entry whose path
	// changes without changing its identifier — a new subfolder, another file
	// extension — holds that identifier itself, and would otherwise be refused
	// as a duplicate of itself.
	if other := reg.indexOfID(system, GameID(newPath)); other != -1 && other != i {
		return ErrDuplicateGameID
	}

	reg.Entries[i].Game.Path = newPath
	// The rename changes the entry's identity (its GameID) but not its
	// position: rekey it in place rather than dropping the whole index, since
	// reg.index is already guaranteed built and accurate here (indexOfID ran
	// twice just above, for i and for the duplicate check).
	delete(reg.index, entryKey{system: system, id: id})
	reg.index[entryKey{system: system, id: GameID(newPath)}] = i
	return nil
}

// RemoveGameFile deletes the metadata file that the entry of system was stored
// under while its game ID was id, and nothing else — no media, unlike
// RemoveByID: a rename never moves the files a game's media references name,
// which are stored in their own right rather than derived from its ROM path.
//
// It is what keeps a rename from leaving a duplicate behind: Save only ever
// writes files, so the file named after the old identifier would otherwise
// survive and load as a second game. A file already absent is the wanted state,
// not an error. An id that is not a plain base name is refused rather than
// followed — os.Remove has no undo.
func RemoveGameFile(registryFolder, system, id string) error {
	if id == "" || id == "." || id == ".." || strings.ContainsRune(id, '/') || strings.ContainsRune(id, filepath.Separator) {
		return ErrUnusablePath
	}
	return removeIfExists(filepath.Join(registryFolder, system, EntryFileName(id)))
}

// EntryFileName names the JSON file an entry of the given game ID is stored
// under. It is the one place that rule lives, so gameFileName (which derives
// the id from a game's path) and a caller already holding an id cannot drift
// apart — the drift decisions/014 was written about.
func EntryFileName(id string) string {
	return id + ".json"
}
