package registry

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
)

// Medium names one of the four media a registry entry may hold. It is a name
// rather than an index because it travels in URLs and in submissions, where a
// position would silently mean something else the day the list changes.
type Medium string

const (
	MediumImage     Medium = "image"
	MediumVideo     Medium = "video"
	MediumMarquee   Medium = "marquee"
	MediumThumbnail Medium = "thumbnail"
)

// The reasons a medium cannot be written. They are told apart because a caller
// words one message per reason — a file of the wrong kind is the user's to fix,
// an unknown medium is a URL that names nothing.
var (
	// ErrUnknownMedium marks a name that is not one of the four media.
	ErrUnknownMedium = errors.New("registry: unknown medium")

	// ErrUnsupportedMediaFile marks a file whose extension is not one the
	// targeted medium accepts.
	ErrUnsupportedMediaFile = errors.New("registry: this file type is not accepted for this medium")
)

// imageExtensions and videoExtensions are what a medium of each kind accepts.
// The list is deliberately short: it is what Batocera renders, and an extension
// outside it buys nothing but a broken image on every page showing the game.
var (
	imageExtensions = []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp"}
	videoExtensions = []string{".mp4", ".mkv", ".webm", ".avi", ".mpg", ".mpeg"}
)

// mediumKind describes one of the four media: how to reach its reference on a
// game, where a file stored for it sits inside the system folder, the suffix
// telling it apart from the game's other media, and the file types it accepts.
type mediumKind struct {
	name       Medium
	field      func(*gamelist.Game) *string
	folder     string
	suffix     string
	extensions []string
}

// mediaKinds is the single table of a game's four media. Every function of this
// package that has to walk them goes through it — RemoveByID, copyGameMedia,
// copyFilledMedia, copyEveryMedium and the media management below — rather than
// each re-listing the same four fields, so a medium added later cannot be
// honoured by one and forgotten by another (the drift decisions/014 was written
// about).
//
// The suffixes follow what Batocera's own scrapers write beside a ROM, so a
// medium uploaded here and one scraped there are named alike.
var mediaKinds = []mediumKind{
	{
		name:       MediumImage,
		field:      func(g *gamelist.Game) *string { return &g.Image },
		folder:     "images",
		suffix:     "-image",
		extensions: imageExtensions,
	},
	{
		name:       MediumVideo,
		field:      func(g *gamelist.Game) *string { return &g.Video },
		folder:     "videos",
		suffix:     "-video",
		extensions: videoExtensions,
	},
	{
		name:       MediumMarquee,
		field:      func(g *gamelist.Game) *string { return &g.Marquee },
		folder:     "images",
		suffix:     "-marquee",
		extensions: imageExtensions,
	},
	{
		name:       MediumThumbnail,
		field:      func(g *gamelist.Game) *string { return &g.Thumbnail },
		folder:     "images",
		suffix:     "-thumb",
		extensions: imageExtensions,
	},
}

// Media lists the four media a game holds, in the order the registry describes
// them — which is the order a page offering them renders.
func Media() []Medium {
	media := make([]Medium, len(mediaKinds))
	for i, kind := range mediaKinds {
		media[i] = kind.name
	}
	return media
}

// LookupMedium resolves a submitted medium name, reporting whether it is one of
// the four. It is what keeps a name coming from a URL or a form from being
// carried any further than this check.
func LookupMedium(name string) (Medium, bool) {
	for _, kind := range mediaKinds {
		if string(kind.name) == name {
			return kind.name, true
		}
	}
	return "", false
}

// Extensions lists the file extensions m accepts, dotted and lowercase, so a
// caller can offer them and word its own refusal from the same list this
// package refuses on.
func (m Medium) Extensions() []string {
	kind, ok := kindOf(m)
	if !ok {
		return nil
	}
	return kind.extensions
}

// kindOf resolves a medium to its description.
func kindOf(m Medium) (mediumKind, bool) {
	for _, kind := range mediaKinds {
		if kind.name == m {
			return kind, true
		}
	}
	return mediumKind{}, false
}

// WriteMedium stores the bytes read from src as the medium m of the entry of
// system whose game ID is id, and points that entry at it. The stored file is
// named from the game ID, the medium's own suffix and the extension of
// filename — filename itself never reaches the disk, so a name coming from a
// request can name no file but the one this registry would have named anyway
// (see decisions/031). It returns ErrUnsupportedMediaFile if that extension is
// not one m accepts, ErrUnknownMedium if m is not a medium, and ErrGameNotFound
// if no entry matches — in every case without writing anything.
//
// The bytes go to a temporary file in the destination's own folder, renamed
// over it only once they are all there: an upload cut off partway — which is
// what the caller's size cap does to an oversized one — leaves the medium that
// was already there intact.
//
// replaced is the reference the entry held, and only when it names a different
// file than the one just written: re-uploading a medium under the same
// extension lands on the very same file, and the "./images/foo.png" form
// gamelist.xml writes names it too. It is the caller's to erase, after the
// registry has been written — the write is what fulfills the intent here, so it
// comes first (decisions/024), and a leftover file is a caveat rather than a
// failure.
func WriteMedium(reg *Registry, registryFolder, system, id string, m Medium, filename string, src io.Reader) (replaced string, err error) {
	i := reg.indexOfID(system, id)
	if i == -1 {
		return "", ErrGameNotFound
	}
	kind, ok := kindOf(m)
	if !ok {
		return "", ErrUnknownMedium
	}
	if err := usableAsFileName(id); err != nil {
		return "", err
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !acceptsExtension(kind, ext) {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedMediaFile, ext)
	}

	relPath := kind.folder + "/" + id + kind.suffix + ext
	systemFolder := filepath.Join(registryFolder, system)
	dst, inside := mediumPath(systemFolder, relPath)
	if !inside {
		return "", ErrUnusablePath
	}
	if err := writeFileAtomically(dst, src); err != nil {
		return "", err
	}

	reference := kind.field(&reg.Entries[i].Game)
	previous := *reference
	*reference = relPath

	if previous == "" {
		return "", nil
	}
	previousPath, previousInside := mediumPath(systemFolder, previous)
	if !previousInside || previousPath == dst {
		// A file outside the system folder is not the registry's to erase, and
		// one resolving to the destination has just been overwritten.
		return "", nil
	}
	return previous, nil
}

// ClearMedium erases the file the entry's medium m refers to and empties that
// reference, in that order: the erasure is what a deletion means, so it is the
// commit point (decisions/022). A medium the entry does not hold is a harmless
// no-op — a page rendered before it was cleared elsewhere asks for exactly
// that. A file already absent is the wanted state.
//
// It returns the reference the entry held, and ErrGameNotFound or
// ErrUnknownMedium — changing nothing — when the entry or the medium does not
// exist. A reference resolving outside the system folder is left on disk rather
// than followed, since scraped data is what fills those references and
// os.Remove has no undo: the reference is emptied all the same and
// ErrMediaLeftBehind reports the file that stayed. A file that resisted erasure
// is a plain failure instead, leaving the reference in place: the deletion did
// not happen, and the page must keep offering it.
func ClearMedium(reg *Registry, registryFolder, system, id string, m Medium) (cleared string, err error) {
	i := reg.indexOfID(system, id)
	if i == -1 {
		return "", ErrGameNotFound
	}
	kind, ok := kindOf(m)
	if !ok {
		return "", ErrUnknownMedium
	}

	reference := kind.field(&reg.Entries[i].Game)
	relPath := *reference
	if relPath == "" {
		return "", nil
	}

	path, inside := mediumPath(filepath.Join(registryFolder, system), relPath)
	if !inside {
		*reference = ""
		return relPath, fmt.Errorf("%w: %s", ErrMediaLeftBehind, relPath)
	}
	if err := removeIfExists(path); err != nil {
		return "", err
	}
	*reference = ""
	return relPath, nil
}

// RemoveMediumFile erases the file a media reference names inside a system's
// folder of the registry, and nothing else. It is what a caller erases a
// replaced medium with, once the registry holds the new reference. An empty
// reference erases nothing, a file already absent is the wanted state, and a
// reference resolving outside the system folder is refused with ErrUnusablePath
// rather than followed.
func RemoveMediumFile(registryFolder, system, relPath string) error {
	if relPath == "" {
		return nil
	}
	path, inside := mediumPath(filepath.Join(registryFolder, system), relPath)
	if !inside {
		return ErrUnusablePath
	}
	return removeIfExists(path)
}

// acceptsExtension reports whether kind stores files of that extension.
func acceptsExtension(kind mediumKind, ext string) bool {
	for _, accepted := range kind.extensions {
		if accepted == ext {
			return true
		}
	}
	return false
}

// usableAsFileName refuses a game ID that could not be part of a file name.
// An ID is a ROM path's base name, so it carries no separator in practice —
// but it is derived from scraped data, and a medium is written from a request.
func usableAsFileName(id string) error {
	if id == "" || id == "." || id == ".." ||
		strings.ContainsRune(id, '/') || strings.ContainsRune(id, filepath.Separator) {
		return ErrUnusablePath
	}
	return nil
}

// writeFileAtomically writes everything src yields to path, through a temporary
// file in path's own folder that is renamed over it once the copy succeeded. A
// copy that fails partway leaves neither a truncated destination nor the
// temporary file behind — which is what makes a refused upload change nothing.
func writeFileAtomically(path string, src io.Reader) error {
	folder := filepath.Dir(path)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(folder, "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	if _, err := io.Copy(tmp, src); err != nil {
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
