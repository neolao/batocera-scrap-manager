package webui

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/store"
)

// mediaRegistry writes a registry holding one game of megadrive whose cover art
// and video are really on disk, a second game of that system with no medium at
// all, and a game of another system — so a test can watch one medium change and
// see everything else stay as it was.
func mediaRegistry(t *testing.T) (*registry.Registry, string) {
	t.Helper()

	folder := t.TempDir()
	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{
			Path: "./Sonic.zip", Name: "Sonic the Hedgehog", Desc: "Fast.",
			Image: "images/Sonic-image.png", Video: "videos/Sonic-video.mp4",
		}},
		{System: "megadrive", Game: gamelist.Game{
			Path: "./Ecco.zip", Name: "Ecco the Dolphin",
		}},
		{System: "mastersystem", Game: gamelist.Game{
			Path: "./Alex Kidd.zip", Name: "Alex Kidd in Miracle World", Image: "images/alex.png",
		}},
	}}
	writeMediaFile(t, folder, "megadrive", "images/Sonic-image.png")
	writeMediaFile(t, folder, "megadrive", "videos/Sonic-video.mp4")
	writeMediaFile(t, folder, "mastersystem", "images/alex.png")
	if err := store.Save(reg, folder); err != nil {
		t.Fatalf("failed to write the test registry: %v", err)
	}
	return reg, folder
}

// uploadURLOf builds the URL a medium of a megadrive game is uploaded to.
func uploadURLOf(id string, m registry.Medium) string {
	return gameURL("megadrive", id) + "/media/" + string(m)
}

// upload submits one file for one medium, the way the game page's own form
// does: a multipart body carrying a single file part, from the same origin.
func upload(t *testing.T, h http.Handler, target, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if filename != "" {
		part, err := writer.CreateFormFile(mediaFileParam, filename)
		if err != nil {
			t.Fatalf("building the upload: %v", err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatalf("building the upload: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("building the upload: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, target, &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

// filler is a body of endless bytes, so an upload larger than the cap can be
// sent without ever holding it in memory.
type filler struct{}

func (filler) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

// uploadOfSize sends an upload of exactly size bytes under a name the medium
// accepts, with no Content-Length — the shape the cap has to stop while the
// body is being read rather than before it starts.
func uploadOfSize(t *testing.T, h http.Handler, target string, size int64) *httptest.ResponseRecorder {
	t.Helper()

	const boundary = "an-upload-larger-than-the-cap"
	head := "--" + boundary + "\r\n" +
		"Content-Disposition: form-data; name=\"" + mediaFileParam + "\"; filename=\"huge.png\"\r\n" +
		"Content-Type: image/png\r\n\r\n"
	tail := "\r\n--" + boundary + "--\r\n"

	body := io.MultiReader(strings.NewReader(head), io.LimitReader(filler{}, size), strings.NewReader(tail))
	r := httptest.NewRequest(http.MethodPost, target, body)
	r.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

// mediaFilesOf lists, sorted, every file present under a system's folder of the
// registry, as paths relative to that folder — what a test compares before and
// after to prove nothing but the medium at stake moved.
func mediaFilesOf(t *testing.T, registryFolder, system string) []string {
	t.Helper()

	root := filepath.Join(registryFolder, system)
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	sort.Strings(files)
	return files
}

// pageProblem returns the text of the refusal block a page carries, or an empty
// string when it carries none.
func pageProblem(body string) string {
	_, after, found := strings.Cut(body, `id="errors"`)
	if !found {
		return ""
	}
	block, _, _ := strings.Cut(after, "</div>")
	return block
}

func TestServeGame_EveryMedium_OffersAnUploadControlNamingWhatItAccepts(t *testing.T) {
	reg, folder := mediaRegistry(t)

	body := get(t, Handler(reg, folder, nil), gameURL("megadrive", "Sonic")).Body.String()

	for _, m := range registry.Media() {
		if !strings.Contains(body, `action="`+uploadURLOf("Sonic", m)+`"`) {
			t.Errorf("the game page offers no way to upload the %q medium", m)
		}
		for _, ext := range m.Extensions() {
			if !strings.Contains(body, ext) {
				t.Errorf("the game page does not name %q among the file types the %q medium accepts", ext, m)
			}
		}
	}
	if !strings.Contains(body, `enctype="multipart/form-data"`) {
		t.Error("the upload forms are not encoded as multipart, so no file could ever reach the server")
	}
}

func TestServeGame_MediumTheGameDoesNotHold_SaysSoRatherThanShowingNothing(t *testing.T) {
	reg, folder := mediaRegistry(t)

	body := get(t, Handler(reg, folder, nil), gameURL("megadrive", "Ecco")).Body.String()

	if !strings.Contains(body, "None yet") {
		t.Errorf("the game page does not say which media are missing\n--- page ---\n%s", body)
	}
}

func TestUploadMedium_EachMedium_StoresItAndPointsTheGameAtTheFileOnDisk(t *testing.T) {
	cases := []struct {
		medium   registry.Medium
		filename string
		want     string
	}{
		{registry.MediumImage, "my cover.PNG", "images/Ecco-image.png"},
		{registry.MediumVideo, "trailer.mp4", "videos/Ecco-video.mp4"},
		{registry.MediumMarquee, "logo.jpg", "images/Ecco-marquee.jpg"},
		{registry.MediumThumbnail, "thumb.webp", "images/Ecco-thumb.webp"},
	}

	for _, c := range cases {
		t.Run(string(c.medium), func(t *testing.T) {
			reg, folder := mediaRegistry(t)
			handler := Handler(reg, folder, nil)

			rec := upload(t, handler, uploadURLOf("Ecco", c.medium), c.filename, []byte("the bytes"))

			if rec.Code != http.StatusSeeOther {
				t.Fatalf("status = %d, want %d\n--- page ---\n%s", rec.Code, http.StatusSeeOther, rec.Body.String())
			}
			stored := filepath.Join(folder, "megadrive", filepath.FromSlash(c.want))
			data, err := os.ReadFile(stored)
			if err != nil {
				t.Fatalf("the medium is not on disk where its reference names it: %v", err)
			}
			if string(data) != "the bytes" {
				t.Errorf("stored content = %q, want the uploaded bytes", data)
			}

			body := get(t, handler, gameURL("megadrive", "Ecco")).Body.String()
			if !strings.Contains(body, mediaURLPrefix+"megadrive/"+c.want) {
				t.Errorf("the game page does not show the uploaded %q medium\n--- page ---\n%s", c.medium, body)
			}
		})
	}
}

func TestUploadMedium_Succeeded_ConfirmsWithABannerNamingTheMedium(t *testing.T) {
	reg, folder := mediaRegistry(t)
	handler := Handler(reg, folder, nil)

	rec := upload(t, handler, uploadURLOf("Ecco", registry.MediumMarquee), "logo.png", []byte("pixels"))

	banner := bannerAfter(t, handler, rec)
	if !strings.Contains(strings.ToLower(banner), "marquee") {
		t.Errorf("banner = %q, want it naming the medium that was saved", banner)
	}
}

func TestUploadMedium_Succeeded_RegeneratesTheConsultationSite(t *testing.T) {
	reg, folder := mediaRegistry(t)

	upload(t, Handler(reg, folder, nil), uploadURLOf("Ecco", registry.MediumImage), "cover.png", []byte("pixels"))

	site, err := os.ReadFile(filepath.Join(folder, "index.html"))
	if err != nil {
		t.Fatalf("reading the consultation site: %v", err)
	}
	if !strings.Contains(string(site), "Ecco-image.png") {
		t.Errorf("the consultation site does not show the medium that was just added\n--- site ---\n%s", site)
	}
}

func TestUploadMedium_MediumAlreadyPresentUnderAnotherExtension_LeavesNoOrphanBehind(t *testing.T) {
	reg, folder := mediaRegistry(t)
	handler := Handler(reg, folder, nil)

	rec := upload(t, handler, uploadURLOf("Sonic", registry.MediumImage), "better.jpg", []byte("a better cover"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	files := mediaFilesOf(t, folder, "megadrive")
	for _, unwanted := range []string{"images/Sonic-image.png"} {
		for _, file := range files {
			if file == unwanted {
				t.Errorf("%s is still there, want the replaced file erased once the registry held the new one", unwanted)
			}
		}
	}
	data, err := os.ReadFile(filepath.Join(folder, "megadrive", "images", "Sonic-image.jpg"))
	if err != nil || string(data) != "a better cover" {
		t.Errorf("the new cover = %q (err %v), want the uploaded bytes", data, err)
	}
}

func TestUploadMedium_MediumAlreadyPresentUnderTheSameExtension_OverwritesItInPlace(t *testing.T) {
	reg, folder := mediaRegistry(t)
	before := mediaFilesOf(t, folder, "megadrive")

	rec := upload(t, Handler(reg, folder, nil), uploadURLOf("Sonic", registry.MediumImage),
		"better.png", []byte("a better cover"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if after := mediaFilesOf(t, folder, "megadrive"); len(after) != len(before) {
		t.Errorf("the system folder holds %v, want the same files as before %v", after, before)
	}
	data, _ := os.ReadFile(filepath.Join(folder, "megadrive", "images", "Sonic-image.png"))
	if string(data) != "a better cover" {
		t.Errorf("stored content = %q, want the uploaded bytes", data)
	}
}

func TestUploadMedium_FileTypeTheMediumDoesNotAccept_IsRefusedWithoutChangingAnything(t *testing.T) {
	reg, folder := mediaRegistry(t)
	handler := Handler(reg, folder, nil)
	before := mediaFilesOf(t, folder, "megadrive")

	rec := upload(t, handler, uploadURLOf("Sonic", registry.MediumImage), "payload.php", []byte("<?php ?>"))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	problem := pageProblem(rec.Body.String())
	if problem == "" {
		t.Fatalf("the refusal says nothing\n--- page ---\n%s", rec.Body.String())
	}
	if !strings.Contains(problem, ".png") {
		t.Errorf("the refusal = %q, want it naming the file types this medium accepts", problem)
	}
	if after := mediaFilesOf(t, folder, "megadrive"); !equalFileLists(after, before) {
		t.Errorf("the system folder holds %v, want it left as %v — a refusal changes nothing", after, before)
	}
	if reg.Entries[0].Game.Image != "images/Sonic-image.png" {
		t.Errorf("Image = %q, want the previous cover art still referenced", reg.Entries[0].Game.Image)
	}
}

func TestUploadMedium_FileLargerThanTheCap_IsRefusedWithoutChangingAnything(t *testing.T) {
	reg, folder := mediaRegistry(t)
	handler := Handler(reg, folder, nil)
	before := mediaFilesOf(t, folder, "megadrive")

	rec := uploadOfSize(t, handler, uploadURLOf("Sonic", registry.MediumImage), maxUploadBytes+1024)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d\n--- page ---\n%s", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
	if pageProblem(rec.Body.String()) == "" {
		t.Fatalf("the refusal says nothing\n--- page ---\n%s", rec.Body.String())
	}
	if after := mediaFilesOf(t, folder, "megadrive"); !equalFileLists(after, before) {
		t.Errorf("the system folder holds %v, want it left as %v — a refusal changes nothing", after, before)
	}
}

func TestUploadMedium_NoFileChosen_IsRefusedWithAnExplicitMessage(t *testing.T) {
	reg, folder := mediaRegistry(t)

	rec := upload(t, Handler(reg, folder, nil), uploadURLOf("Sonic", registry.MediumMarquee), "", nil)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if pageProblem(rec.Body.String()) == "" {
		t.Errorf("the refusal says nothing\n--- page ---\n%s", rec.Body.String())
	}
}

func TestUploadMedium_EmptyFile_IsRefusedRatherThanStoredAsAMedium(t *testing.T) {
	reg, folder := mediaRegistry(t)

	rec := upload(t, Handler(reg, folder, nil), uploadURLOf("Sonic", registry.MediumMarquee), "empty.png", nil)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d — an empty file is a broken image everywhere", rec.Code, http.StatusBadRequest)
	}
	if reg.Entries[0].Game.Marquee != "" {
		t.Errorf("Marquee = %q, want nothing stored", reg.Entries[0].Game.Marquee)
	}
}

func TestUploadMedium_SubmissionFromAnotherSite_IsRefused(t *testing.T) {
	reg, folder := mediaRegistry(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile(mediaFileParam, "cover.png")
	_, _ = part.Write([]byte("pixels"))
	_ = writer.Close()

	r := httptest.NewRequest(http.MethodPost, uploadURLOf("Sonic", registry.MediumImage), &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()
	Handler(reg, folder, nil).ServeHTTP(rec, r)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestUploadMedium_UnknownGame_AnswersNotFound(t *testing.T) {
	reg, folder := mediaRegistry(t)

	rec := upload(t, Handler(reg, folder, nil), uploadURLOf("Mario", registry.MediumImage),
		"cover.png", []byte("pixels"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUploadMedium_MediumTheRegistryDoesNotHold_AnswersNotFound(t *testing.T) {
	reg, folder := mediaRegistry(t)

	rec := upload(t, Handler(reg, folder, nil), gameURL("megadrive", "Sonic")+"/media/screenshot",
		"shot.png", []byte("pixels"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUploadMedium_MethodTheURLDoesNotAccept_AnswersMethodNotAllowed(t *testing.T) {
	reg, folder := mediaRegistry(t)

	rec := get(t, Handler(reg, folder, nil), uploadURLOf("Sonic", registry.MediumImage))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodPost {
		t.Errorf("Allow = %q, want %q", allow, http.MethodPost)
	}
}

// removeURLOf builds the URL one medium of a megadrive game is removed at.
func removeURLOf(id string, m registry.Medium) string {
	return uploadURLOf(id, m) + "/delete"
}

func TestServeGame_MediumTheGameHolds_OffersToRemoveIt(t *testing.T) {
	reg, folder := mediaRegistry(t)

	body := get(t, Handler(reg, folder, nil), gameURL("megadrive", "Sonic")).Body.String()

	if !strings.Contains(body, `href="`+removeURLOf("Sonic", registry.MediumImage)+`"`) {
		t.Errorf("the game page offers no way to remove the cover art it holds\n--- page ---\n%s", body)
	}
	if strings.Contains(body, `href="`+removeURLOf("Sonic", registry.MediumMarquee)+`"`) {
		t.Error("the game page offers to remove a marquee it does not hold")
	}
}

func TestServeMediumDeleteConfirmation_NamesTheFileAndErasesNothing(t *testing.T) {
	reg, folder := mediaRegistry(t)
	before := mediaFilesOf(t, folder, "megadrive")

	rec := get(t, Handler(reg, folder, nil), removeURLOf("Sonic", registry.MediumImage))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "megadrive/images/Sonic-image.png") {
		t.Errorf("the confirmation does not name the file about to be erased\n--- page ---\n%s", body)
	}
	if !strings.Contains(body, "Cover art") {
		t.Errorf("the confirmation does not name the medium\n--- page ---\n%s", body)
	}
	if after := mediaFilesOf(t, folder, "megadrive"); !equalFileLists(after, before) {
		t.Errorf("the system folder holds %v, want %v — opening the confirmation erases nothing", after, before)
	}
}

func TestServeMediumDeleteConfirmation_IsRevalidatedRatherThanServedFromTheCache(t *testing.T) {
	// Going back to this page after the deletion must not offer to erase the
	// medium again.
	reg, folder := mediaRegistry(t)

	rec := get(t, Handler(reg, folder, nil), removeURLOf("Sonic", registry.MediumVideo))

	if cache := rec.Header().Get("Cache-Control"); cache != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", cache, "no-cache")
	}
}

func TestRemoveMedium_Confirmed_ErasesTheFileAndEmptiesTheReference(t *testing.T) {
	reg, folder := mediaRegistry(t)
	handler := Handler(reg, folder, nil)

	rec := post(t, handler, removeURLOf("Sonic", registry.MediumVideo), nil)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d\n--- page ---\n%s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	erased := filepath.Join(folder, "megadrive", "videos", "Sonic-video.mp4")
	if _, err := os.Stat(erased); !os.IsNotExist(err) {
		t.Errorf("Stat(%s) error = %v, want the file gone", erased, err)
	}
	if gameFile(t, folder) == "" {
		t.Fatal("the game file is gone, want it rewritten without the video")
	}
	if strings.Contains(gameFile(t, folder), "Sonic-video.mp4") {
		t.Errorf("the stored game still refers to the video:\n%s", gameFile(t, folder))
	}

	body := get(t, handler, gameURL("megadrive", "Sonic")).Body.String()
	if strings.Contains(body, "Sonic-video.mp4") {
		t.Errorf("the game page still shows the removed video\n--- page ---\n%s", body)
	}
	if !strings.Contains(body, "None yet") {
		t.Error("the game page does not say the video slot is now empty")
	}
}

func TestRemoveMedium_Confirmed_ConfirmsWithABannerNamingTheMedium(t *testing.T) {
	reg, folder := mediaRegistry(t)
	handler := Handler(reg, folder, nil)

	rec := post(t, handler, removeURLOf("Sonic", registry.MediumImage), nil)

	banner := bannerAfter(t, handler, rec)
	if !strings.Contains(banner, "Cover art") || !strings.Contains(banner, "removed") {
		t.Errorf("banner = %q, want it naming the medium that was removed", banner)
	}
}

func TestRemoveMedium_Confirmed_RegeneratesTheConsultationSiteWithoutIt(t *testing.T) {
	reg, folder := mediaRegistry(t)

	post(t, Handler(reg, folder, nil), removeURLOf("Sonic", registry.MediumImage), nil)

	site, err := os.ReadFile(filepath.Join(folder, "index.html"))
	if err != nil {
		t.Fatalf("reading the consultation site: %v", err)
	}
	if strings.Contains(string(site), "Sonic-image.png") {
		t.Errorf("the consultation site still shows the medium that was removed\n--- site ---\n%s", site)
	}
}

func TestRemoveMedium_LeavesTheGamesOtherMediaAlone(t *testing.T) {
	reg, folder := mediaRegistry(t)

	post(t, Handler(reg, folder, nil), removeURLOf("Sonic", registry.MediumImage), nil)

	if files := mediaFilesOf(t, folder, "megadrive"); !slices.Contains(files, "videos/Sonic-video.mp4") {
		t.Errorf("the system folder holds %v, want the video left alone", files)
	}
	if files := mediaFilesOf(t, folder, "mastersystem"); !slices.Contains(files, "images/alex.png") {
		t.Errorf("another system's folder holds %v, want it untouched", files)
	}
}

func TestRemoveMedium_MediumAlreadyGone_SaysSoRatherThanFailing(t *testing.T) {
	// A page rendered before the medium was removed elsewhere submits exactly
	// this: a harmless repeat, still truthfully confirmed.
	reg, folder := mediaRegistry(t)
	handler := Handler(reg, folder, nil)
	post(t, handler, removeURLOf("Sonic", registry.MediumImage), nil)

	rec := post(t, handler, removeURLOf("Sonic", registry.MediumImage), nil)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if banner := bannerAfter(t, handler, rec); !strings.Contains(banner, "Cover art") {
		t.Errorf("banner = %q, want the resulting state stated all the same", banner)
	}
}

func TestServeMediumDeleteConfirmation_MediumAlreadyGone_PromisesNoFile(t *testing.T) {
	reg, folder := mediaRegistry(t)

	body := get(t, Handler(reg, folder, nil), removeURLOf("Sonic", registry.MediumMarquee)).Body.String()

	if !strings.Contains(body, "nothing left to erase") {
		t.Errorf("the confirmation does not say the medium is already gone\n--- page ---\n%s", body)
	}
}

func TestRemoveMedium_SubmissionFromAnotherSite_IsRefused(t *testing.T) {
	reg, folder := mediaRegistry(t)

	r := httptest.NewRequest(http.MethodPost, removeURLOf("Sonic", registry.MediumImage), nil)
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()
	Handler(reg, folder, nil).ServeHTTP(rec, r)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if _, err := os.Stat(filepath.Join(folder, "megadrive", "images", "Sonic-image.png")); err != nil {
		t.Errorf("the cover art is gone: %v — a refused request erases nothing", err)
	}
}

func TestRemoveMedium_UnknownGame_AnswersNotFound(t *testing.T) {
	reg, folder := mediaRegistry(t)

	rec := post(t, Handler(reg, folder, nil), removeURLOf("Mario", registry.MediumImage), nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRemoveMedium_MediumTheRegistryDoesNotHold_AnswersNotFound(t *testing.T) {
	reg, folder := mediaRegistry(t)

	rec := post(t, Handler(reg, folder, nil), gameURL("megadrive", "Sonic")+"/media/screenshot/delete", nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRemoveMedium_MethodTheURLDoesNotAccept_AnswersMethodNotAllowed(t *testing.T) {
	reg, folder := mediaRegistry(t)

	r := httptest.NewRequest(http.MethodDelete, removeURLOf("Sonic", registry.MediumImage), nil)
	rec := httptest.NewRecorder()
	Handler(reg, folder, nil).ServeHTTP(rec, r)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); allow != readAndSubmit {
		t.Errorf("Allow = %q, want %q", allow, readAndSubmit)
	}
}

func TestRemoveMedium_FileThatCannotBeErased_ShowsTheConfirmationAgainRatherThanADeadEnd(t *testing.T) {
	reg, folder := mediaRegistry(t)
	denyWritesTo(t, filepath.Join(folder, "megadrive", "images"))

	rec := post(t, Handler(reg, folder, nil), removeURLOf("Sonic", registry.MediumImage), nil)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if pageProblem(rec.Body.String()) == "" {
		t.Errorf("the failure says nothing\n--- page ---\n%s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Remove this Cover art") {
		t.Error("the confirmation is not shown again, so the removal cannot be retried from where it failed")
	}
	if reg.Entries[0].Game.Image != "images/Sonic-image.png" {
		t.Errorf("Image = %q, want the reference kept: the file is still there", reg.Entries[0].Game.Image)
	}
}

// equalFileLists compares two sorted file listings.
func equalFileLists(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
