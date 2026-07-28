package webui

import (
	"errors"
	"net/url"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// pathKey is the form field, the error anchor and the element id prefix of the
// ROM path control. It is deliberately not one of editableFields' keys: the
// path identifies the entry rather than describing it, so it gets neither a
// hand-edited mark nor the checkbox handing a value back to the scraper.
const pathKey = "path"

// The query parameters composing the confirmation of a path change. Unlike
// every other outcome, that sentence cannot be a table entry: it names the
// path the user just gave, and what it says next depends on whether the game's
// address changed and on whether the file it left behind could be erased.
const (
	pathParam       = "path"
	movedParam      = "moved"
	leftBehindParam = "left"

	// savedPath is the outcome a path change redirects with.
	savedPath = "path"
)

// pathLabel and pathHint word the control. The hint states the one rule that
// makes the rest of this page's behaviour predictable — that the file name
// alone identifies the game — since a subfolder-only edit otherwise looks like
// it did nothing, and two visibly different paths otherwise collide for no
// visible reason.
const (
	pathLabel = "ROM file"
	pathHint  = "Path relative to the system folder, subfolder included — e.g. disc1/Sonic.zip. " +
		"The file name without its extension identifies the game, so changing it changes this page's address."
)

// pathMessages words each rule ValidatePath enforces, one message per reason:
// a single "invalid path" would leave the user guessing which one was broken.
var pathMessages = map[error]string{
	registry.ErrEmptyPath:    "Give the ROM file a path: it is how this game is identified.",
	registry.ErrAbsolutePath: "Use a path relative to the system folder, without a leading slash.",
	registry.ErrEscapingPath: "Keep the path inside the system folder: \"..\" leaves it.",
	registry.ErrUnusablePath: "Name a ROM file, not a folder: separate folders with \"/\" and end with the file's own name.",
}

// pathError refuses a submitted path, in words, or returns an empty string
// when it is storable. The duplicate check needs the registry, so it names the
// game already holding the identifier rather than only the identifier: the
// user is comparing two paths that visibly differ, and would otherwise read
// the refusal as a bug.
func pathError(reg *registry.Registry, system, id, romPath string) string {
	if err := registry.ValidatePath(romPath); err != nil {
		return pathRefusal(err)
	}

	newID := registry.GameID(romPath)
	if newID == id {
		return ""
	}
	other, found := reg.FindByID(system, newID)
	if !found {
		return ""
	}

	// The game is named only when its name adds something: a file called after
	// the game it holds would otherwise be introduced twice in one sentence.
	held := ""
	if other.Game.Name != newID {
		held = " — the one named " + other.Game.Name
	}
	return "Another game of this system already uses the file name \"" + newID + "\"" + held +
		". File names must be unique within a system, whatever their subfolder."
}

// pathRefusal words whichever rule err names. An error the table does not know
// still gets the catch-all rather than an empty message: a refused submission
// showing no reason is worse than an imprecise one.
func pathRefusal(err error) string {
	for reason, message := range pathMessages {
		if errors.Is(err, reason) {
			return message
		}
	}
	if errors.Is(err, registry.ErrDuplicateGameID) {
		return "Another game of this system already uses that file name. File names must be unique within a system, whatever their subfolder."
	}
	return pathMessages[registry.ErrUnusablePath]
}

// pathControl builds the ROM file control. It reuses the same shape as the
// metadata controls, so the error summary, the anchors and the styling stay
// one pattern rather than two.
func pathControl(value, message string) formControl {
	return formControl{
		Key:      pathKey,
		Label:    pathLabel,
		Kind:     kindText,
		Hint:     pathHint,
		Required: true,
		Value:    value,
		Error:    message,
	}
}

// pathConfirmation composes the sentence confirming a path change. It states
// the resulting path rather than the action, like every other confirmation, so
// it stays true when a stale page repeats a change that already holds.
func pathConfirmation(query url.Values) string {
	romPath := query.Get(pathParam)
	if romPath == "" {
		return ""
	}

	message := "ROM file saved: this game is now stored as " + romPath + "."
	if query.Get(movedParam) != "" {
		message += " Its page has moved to this address."
	}
	if left := query.Get(leftBehindParam); left != "" {
		message += " The file it was stored under, " + left +
			", could not be deleted — remove it by hand, or the game comes back twice when the server restarts."
	}
	return message
}

// renameQuery builds what the redirect carries so the game's page can confirm
// the change: the new path, whether the page moved, and the file left behind
// if one was. moved is stated rather than recomputed, because the page landed
// on cannot know which address the user came from.
func renameQuery(romPath string, moved bool, leftBehind string) url.Values {
	query := url.Values{}
	query.Set(pathParam, romPath)
	if moved {
		query.Set(movedParam, "1")
	}
	if leftBehind != "" {
		query.Set(leftBehindParam, leftBehind)
	}
	return query
}
