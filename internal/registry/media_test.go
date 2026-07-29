package registry

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
)

// oneScrapedGame builds a registry holding a single game, the shape every
// medium test starts from: one entry to write a medium onto, and one of
// another system so the lookups are proven to be per-system.
func oneScrapedGame() *Registry {
	return &Registry{Entries: []Entry{
		{System: "megadrive", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic the Hedgehog"}},
		{System: "snes", Game: gamelist.Game{Path: "./Sonic.zip", Name: "Sonic on the SNES"}},
	}}
}

// writeMediumFile puts content at relPath inside a system's folder of the
// registry, standing for a medium a previous scrape had copied there.
func writeMediumFile(t *testing.T, registryFolder, system, relPath, content string) string {
	t.Helper()
	full := filepath.Join(registryFolder, system, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", full, err)
	}
	return full
}

// mediumOfEntry reads back the reference an entry holds for m, so a test can
// assert on the very field the operation was asked to write.
func mediumOfEntry(t *testing.T, e Entry, m Medium) string {
	t.Helper()
	switch m {
	case MediumImage:
		return e.Game.Image
	case MediumVideo:
		return e.Game.Video
	case MediumMarquee:
		return e.Game.Marquee
	case MediumThumbnail:
		return e.Game.Thumbnail
	}
	t.Fatalf("mediumOfEntry: %q is not a medium", m)
	return ""
}

func TestMedia_ListsTheFourMediaAGameHolds(t *testing.T) {
	got := Media()

	want := []Medium{MediumImage, MediumVideo, MediumMarquee, MediumThumbnail}
	if len(got) != len(want) {
		t.Fatalf("Media() = %v, want the four media %v", got, want)
	}
	for i, m := range want {
		if got[i] != m {
			t.Errorf("Media()[%d] = %q, want %q", i, got[i], m)
		}
	}
}

func TestMediumExtensions_EveryMedium_OffersDottedLowercaseExtensions(t *testing.T) {
	for _, m := range Media() {
		exts := m.Extensions()
		if len(exts) == 0 {
			t.Errorf("%q.Extensions() is empty, want the file types it accepts", m)
		}
		for _, ext := range exts {
			if !strings.HasPrefix(ext, ".") || ext != strings.ToLower(ext) {
				t.Errorf("%q.Extensions() holds %q, want a dotted lowercase extension", m, ext)
			}
		}
	}
}

func TestLookupMedium_NameOfferedByTheRegistry_IsResolved(t *testing.T) {
	m, ok := LookupMedium("marquee")

	if !ok || m != MediumMarquee {
		t.Errorf("LookupMedium(\"marquee\") = %q, %v, want %q, true", m, ok, MediumMarquee)
	}
}

func TestLookupMedium_NameTheRegistryDoesNotHold_IsRefused(t *testing.T) {
	if m, ok := LookupMedium("screenshot"); ok {
		t.Errorf("LookupMedium(\"screenshot\") = %q, true, want it refused", m)
	}
}

func TestWriteMedium_EachMedium_StoresTheFileUnderANameDerivedFromTheGameID(t *testing.T) {
	cases := []struct {
		medium   Medium
		filename string
		want     string
	}{
		{MediumImage, "whatever the user called it.PNG", "images/Sonic-image.png"},
		{MediumVideo, "trailer.mp4", "videos/Sonic-video.mp4"},
		{MediumMarquee, "logo.png", "images/Sonic-marquee.png"},
		{MediumThumbnail, "thumb.jpg", "images/Sonic-thumb.jpg"},
	}

	for _, c := range cases {
		t.Run(string(c.medium), func(t *testing.T) {
			reg, folder := oneScrapedGame(), t.TempDir()

			replaced, err := WriteMedium(reg, folder, "megadrive", "Sonic", c.medium,
				c.filename, strings.NewReader("the bytes of "+string(c.medium)))
			if err != nil {
				t.Fatalf("WriteMedium() error = %v, want nil", err)
			}
			if replaced != "" {
				t.Errorf("replaced = %q, want nothing replaced: the game held no such medium", replaced)
			}

			if got := mediumOfEntry(t, reg.Entries[0], c.medium); got != c.want {
				t.Errorf("stored reference = %q, want %q — derived from the game ID, never from the submitted filename", got, c.want)
			}
			data, err := os.ReadFile(filepath.Join(folder, "megadrive", filepath.FromSlash(c.want)))
			if err != nil {
				t.Fatalf("the medium was not written where the reference names it: %v", err)
			}
			if string(data) != "the bytes of "+string(c.medium) {
				t.Errorf("stored content = %q, want the submitted bytes", data)
			}
		})
	}
}

func TestWriteMedium_ExtensionOutsideTheMediumsList_IsRefusedWithoutWritingAnything(t *testing.T) {
	reg, folder := oneScrapedGame(), t.TempDir()

	_, err := WriteMedium(reg, folder, "megadrive", "Sonic", MediumImage,
		"payload.php", strings.NewReader("<?php ?>"))

	if !errors.Is(err, ErrUnsupportedMediaFile) {
		t.Fatalf("WriteMedium() error = %v, want ErrUnsupportedMediaFile", err)
	}
	if reg.Entries[0].Game.Image != "" {
		t.Errorf("Image = %q, want it left empty: a refusal changes nothing", reg.Entries[0].Game.Image)
	}
	if entries, _ := os.ReadDir(filepath.Join(folder, "megadrive")); len(entries) != 0 {
		t.Errorf("the system folder holds %d entries, want nothing written", len(entries))
	}
}

func TestWriteMedium_VideoOfferedAsCoverArt_IsRefused(t *testing.T) {
	// The allow-list is per medium, not one list shared by the four: a video
	// stored as a cover art would render as a broken image everywhere.
	reg, folder := oneScrapedGame(), t.TempDir()

	_, err := WriteMedium(reg, folder, "megadrive", "Sonic", MediumImage,
		"trailer.mp4", strings.NewReader("moving pictures"))

	if !errors.Is(err, ErrUnsupportedMediaFile) {
		t.Fatalf("WriteMedium() error = %v, want ErrUnsupportedMediaFile", err)
	}
}

func TestWriteMedium_FilenameNamingAnotherFolder_WritesOnlyInsideTheRegistry(t *testing.T) {
	reg, folder := oneScrapedGame(), t.TempDir()
	outside := filepath.Join(folder, "outside.png")

	replaced, err := WriteMedium(reg, folder, "megadrive", "Sonic", MediumImage,
		"../../outside.png", strings.NewReader("hostile"))
	if err != nil {
		t.Fatalf("WriteMedium() error = %v, want the filename read for its extension alone", err)
	}
	if replaced != "" {
		t.Errorf("replaced = %q, want nothing replaced", replaced)
	}

	if reg.Entries[0].Game.Image != "images/Sonic-image.png" {
		t.Errorf("Image = %q, want the reference derived from the game ID", reg.Entries[0].Game.Image)
	}
	if _, err := os.Stat(outside); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s exists, want nothing written outside the system folder", outside)
	}
}

func TestWriteMedium_UnknownMedium_IsRefusedWithoutWritingAnything(t *testing.T) {
	reg, folder := oneScrapedGame(), t.TempDir()

	_, err := WriteMedium(reg, folder, "megadrive", "Sonic", Medium("screenshot"),
		"shot.png", strings.NewReader("pixels"))

	if !errors.Is(err, ErrUnknownMedium) {
		t.Fatalf("WriteMedium() error = %v, want ErrUnknownMedium", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(folder, "megadrive")); len(entries) != 0 {
		t.Errorf("the system folder holds %d entries, want nothing written", len(entries))
	}
}

func TestWriteMedium_UnknownGame_IsRefusedWithoutWritingAnything(t *testing.T) {
	reg, folder := oneScrapedGame(), t.TempDir()

	_, err := WriteMedium(reg, folder, "megadrive", "Mario", MediumImage,
		"cover.png", strings.NewReader("pixels"))

	if !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("WriteMedium() error = %v, want ErrGameNotFound", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(folder, "megadrive")); len(entries) != 0 {
		t.Errorf("the system folder holds %d entries, want nothing written", len(entries))
	}
}

func TestWriteMedium_ReplacingWithTheSameExtension_OverwritesTheSameFileAndLeavesNothingBehind(t *testing.T) {
	reg, folder := oneScrapedGame(), t.TempDir()
	reg.Entries[0].Game.Image = "images/Sonic-image.png"
	writeMediumFile(t, folder, "megadrive", "images/Sonic-image.png", "the old cover")

	replaced, err := WriteMedium(reg, folder, "megadrive", "Sonic", MediumImage,
		"new.png", strings.NewReader("the new cover"))
	if err != nil {
		t.Fatalf("WriteMedium() error = %v, want nil", err)
	}

	if replaced != "" {
		t.Errorf("replaced = %q, want nothing to erase: the new file took the old one's place", replaced)
	}
	data, err := os.ReadFile(filepath.Join(folder, "megadrive", "images", "Sonic-image.png"))
	if err != nil || string(data) != "the new cover" {
		t.Errorf("stored content = %q (err %v), want \"the new cover\"", data, err)
	}
}

func TestWriteMedium_ReplacingWithAnotherExtension_ReportsThePreviousReferenceToErase(t *testing.T) {
	reg, folder := oneScrapedGame(), t.TempDir()
	reg.Entries[0].Game.Image = "images/Sonic-image.jpg"
	writeMediumFile(t, folder, "megadrive", "images/Sonic-image.jpg", "the old cover")

	replaced, err := WriteMedium(reg, folder, "megadrive", "Sonic", MediumImage,
		"new.png", strings.NewReader("the new cover"))
	if err != nil {
		t.Fatalf("WriteMedium() error = %v, want nil", err)
	}

	if replaced != "images/Sonic-image.jpg" {
		t.Errorf("replaced = %q, want the previous reference, which now names an unreferenced file", replaced)
	}
	if reg.Entries[0].Game.Image != "images/Sonic-image.png" {
		t.Errorf("Image = %q, want it pointing at the new file", reg.Entries[0].Game.Image)
	}
	// The erasure is the caller's, after the registry was written: the previous
	// file is still there at this point.
	if _, err := os.Stat(filepath.Join(folder, "megadrive", "images", "Sonic-image.jpg")); err != nil {
		t.Errorf("the previous file is already gone: %v — erasing it is the caller's, once the registry holds the new reference", err)
	}
}

func TestWriteMedium_PreviousReferenceWrittenRelatively_IsNotMistakenForAnotherFile(t *testing.T) {
	// gamelist.xml writes references as "./images/foo.png". That names the very
	// file the new upload lands on, and reporting it as replaced would have the
	// caller erase what was just written.
	reg, folder := oneScrapedGame(), t.TempDir()
	reg.Entries[0].Game.Image = "./images/Sonic-image.png"
	writeMediumFile(t, folder, "megadrive", "images/Sonic-image.png", "the old cover")

	replaced, err := WriteMedium(reg, folder, "megadrive", "Sonic", MediumImage,
		"new.png", strings.NewReader("the new cover"))
	if err != nil {
		t.Fatalf("WriteMedium() error = %v, want nil", err)
	}

	if replaced != "" {
		t.Errorf("replaced = %q, want nothing: it names the same file as the new reference", replaced)
	}
	data, _ := os.ReadFile(filepath.Join(folder, "megadrive", "images", "Sonic-image.png"))
	if string(data) != "the new cover" {
		t.Errorf("stored content = %q, want the new cover", data)
	}
}

func TestWriteMedium_PreviousReferenceEscapingTheSystemFolder_IsNotReportedForErasure(t *testing.T) {
	reg, folder := oneScrapedGame(), t.TempDir()
	reg.Entries[0].Game.Image = "../../elsewhere.png"

	replaced, err := WriteMedium(reg, folder, "megadrive", "Sonic", MediumImage,
		"new.png", strings.NewReader("the new cover"))
	if err != nil {
		t.Fatalf("WriteMedium() error = %v, want nil", err)
	}

	if replaced != "" {
		t.Errorf("replaced = %q, want nothing: the registry does not own that file", replaced)
	}
}

func TestWriteMedium_SourceFailingMidway_LeavesThePreviousMediumIntact(t *testing.T) {
	// This is what an upload cut off by the size cap looks like from here: the
	// bytes stop coming halfway through.
	reg, folder := oneScrapedGame(), t.TempDir()
	reg.Entries[0].Game.Image = "images/Sonic-image.png"
	writeMediumFile(t, folder, "megadrive", "images/Sonic-image.png", "the old cover")

	src := io.MultiReader(strings.NewReader("half of "), errorReader{})
	_, err := WriteMedium(reg, folder, "megadrive", "Sonic", MediumImage, "new.png", src)

	if err == nil {
		t.Fatal("WriteMedium() error = nil, want the read failure reported")
	}
	data, readErr := os.ReadFile(filepath.Join(folder, "megadrive", "images", "Sonic-image.png"))
	if readErr != nil || string(data) != "the old cover" {
		t.Errorf("stored content = %q (err %v), want the previous medium untouched", data, readErr)
	}
	if reg.Entries[0].Game.Image != "images/Sonic-image.png" {
		t.Errorf("Image = %q, want the reference left as it was", reg.Entries[0].Game.Image)
	}
	entries, _ := os.ReadDir(filepath.Join(folder, "megadrive", "images"))
	if len(entries) != 1 {
		t.Errorf("the images folder holds %d files, want only the previous medium — no half-written leftover", len(entries))
	}
}

// errorReader stands for a request body that stops delivering bytes partway
// through, which is what the web UI's size cap does to an oversized upload.
type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("the upload was cut off") }

func TestClearMedium_ExistingMedium_ErasesTheFileAndEmptiesTheReference(t *testing.T) {
	reg, folder := oneScrapedGame(), t.TempDir()
	reg.Entries[0].Game.Video = "./videos/Sonic-video.mp4"
	full := writeMediumFile(t, folder, "megadrive", "videos/Sonic-video.mp4", "moving pictures")

	cleared, err := ClearMedium(reg, folder, "megadrive", "Sonic", MediumVideo)
	if err != nil {
		t.Fatalf("ClearMedium() error = %v, want nil", err)
	}

	if cleared != "./videos/Sonic-video.mp4" {
		t.Errorf("cleared = %q, want the reference the entry held", cleared)
	}
	if reg.Entries[0].Game.Video != "" {
		t.Errorf("Video = %q, want it emptied", reg.Entries[0].Game.Video)
	}
	if _, err := os.Stat(full); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Stat(%s) error = %v, want the file gone", full, err)
	}
}

func TestClearMedium_MediumTheGameDoesNotHold_ChangesNothingAndIsNotAnError(t *testing.T) {
	// A page rendered before the medium was cleared elsewhere submits this: a
	// harmless repeat, not a failure.
	reg, folder := oneScrapedGame(), t.TempDir()

	cleared, err := ClearMedium(reg, folder, "megadrive", "Sonic", MediumMarquee)

	if err != nil {
		t.Fatalf("ClearMedium() error = %v, want nil", err)
	}
	if cleared != "" {
		t.Errorf("cleared = %q, want nothing: the game held no marquee", cleared)
	}
}

func TestClearMedium_ReferencedFileAlreadyGone_StillEmptiesTheReference(t *testing.T) {
	reg, folder := oneScrapedGame(), t.TempDir()
	reg.Entries[0].Game.Marquee = "images/Sonic-marquee.png"

	cleared, err := ClearMedium(reg, folder, "megadrive", "Sonic", MediumMarquee)

	if err != nil {
		t.Fatalf("ClearMedium() error = %v, want a file already absent to be the wanted state", err)
	}
	if cleared != "images/Sonic-marquee.png" || reg.Entries[0].Game.Marquee != "" {
		t.Errorf("cleared = %q, Marquee = %q, want the reference reported and emptied", cleared, reg.Entries[0].Game.Marquee)
	}
}

func TestClearMedium_ReferenceEscapingTheSystemFolder_EmptiesItAndReportsTheFileLeftBehind(t *testing.T) {
	// Media references come from scraped data, so one of them reaching outside
	// must not make a deletion erase a file the registry does not own.
	reg, folder := oneScrapedGame(), t.TempDir()
	reg.Entries[0].Game.Thumbnail = "../../elsewhere.png"
	outside := filepath.Join(folder, "elsewhere.png")
	if err := os.WriteFile(outside, []byte("not ours"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cleared, err := ClearMedium(reg, folder, "megadrive", "Sonic", MediumThumbnail)

	if !errors.Is(err, ErrMediaLeftBehind) {
		t.Fatalf("ClearMedium() error = %v, want ErrMediaLeftBehind", err)
	}
	if cleared != "../../elsewhere.png" || reg.Entries[0].Game.Thumbnail != "" {
		t.Errorf("cleared = %q, Thumbnail = %q, want the reference reported and emptied all the same",
			cleared, reg.Entries[0].Game.Thumbnail)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("Stat(%s) error = %v, want the file left alone", outside, err)
	}
}

func TestClearMedium_UnknownGame_IsRefused(t *testing.T) {
	reg, folder := oneScrapedGame(), t.TempDir()

	if _, err := ClearMedium(reg, folder, "megadrive", "Mario", MediumImage); !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("ClearMedium() error = %v, want ErrGameNotFound", err)
	}
}

func TestClearMedium_UnknownMedium_IsRefused(t *testing.T) {
	reg, folder := oneScrapedGame(), t.TempDir()

	if _, err := ClearMedium(reg, folder, "megadrive", "Sonic", Medium("screenshot")); !errors.Is(err, ErrUnknownMedium) {
		t.Fatalf("ClearMedium() error = %v, want ErrUnknownMedium", err)
	}
}

func TestClearMedium_AGameOfAnotherSystemSharingTheIdentifier_IsLeftAlone(t *testing.T) {
	reg, folder := oneScrapedGame(), t.TempDir()
	reg.Entries[0].Game.Image = "images/Sonic-image.png"
	reg.Entries[1].Game.Image = "images/Sonic-image.png"
	writeMediumFile(t, folder, "megadrive", "images/Sonic-image.png", "megadrive cover")
	snes := writeMediumFile(t, folder, "snes", "images/Sonic-image.png", "snes cover")

	if _, err := ClearMedium(reg, folder, "megadrive", "Sonic", MediumImage); err != nil {
		t.Fatalf("ClearMedium() error = %v, want nil", err)
	}

	if reg.Entries[1].Game.Image != "images/Sonic-image.png" {
		t.Errorf("the SNES game's Image = %q, want it untouched", reg.Entries[1].Game.Image)
	}
	if data, err := os.ReadFile(snes); err != nil || string(data) != "snes cover" {
		t.Errorf("the SNES cover = %q (err %v), want it left on disk", data, err)
	}
}

func TestRemoveMediumFile_ReferenceInsideTheSystemFolder_ErasesIt(t *testing.T) {
	folder := t.TempDir()
	full := writeMediumFile(t, folder, "megadrive", "images/Sonic-image.jpg", "the old cover")

	if err := RemoveMediumFile(folder, "megadrive", "./images/Sonic-image.jpg"); err != nil {
		t.Fatalf("RemoveMediumFile() error = %v, want nil", err)
	}
	if _, err := os.Stat(full); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Stat(%s) error = %v, want the file gone", full, err)
	}
}

func TestRemoveMediumFile_EmptyReference_ErasesNothingAndIsNotAnError(t *testing.T) {
	if err := RemoveMediumFile(t.TempDir(), "megadrive", ""); err != nil {
		t.Fatalf("RemoveMediumFile() error = %v, want nil", err)
	}
}

func TestRemoveMediumFile_ReferenceEscapingTheSystemFolder_IsRefusedWithoutErasingAnything(t *testing.T) {
	folder := t.TempDir()
	outside := filepath.Join(folder, "elsewhere.png")
	if err := os.WriteFile(outside, []byte("not ours"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := RemoveMediumFile(folder, "megadrive", "../../elsewhere.png")

	if !errors.Is(err, ErrUnusablePath) {
		t.Fatalf("RemoveMediumFile() error = %v, want ErrUnusablePath", err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("Stat(%s) error = %v, want the file left alone", outside, err)
	}
}
