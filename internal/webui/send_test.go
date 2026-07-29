package webui

import (
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// registryForSending builds a registry holding three megadrive games and writes
// it to disk, so a test can assert the send left the registry folder alone.
// Sonic is what the folders are missing; Streets of Rage is what they already
// hold differently; Ghost is in no folder at all.
func registryForSending(t *testing.T) (*registry.Registry, string) {
	t.Helper()

	reg := &registry.Registry{Entries: []registry.Entry{
		{System: "megadrive", Game: gamelist.Game{
			Path:  "./Sonic.zip",
			Name:  "Sonic the Hedgehog",
			Desc:  "A blue hedgehog runs very fast.",
			Genre: "Platform",
		}},
		{System: "megadrive", Game: gamelist.Game{
			Path: "./Streets.zip",
			Name: "Streets of Rage",
			Desc: "The registry's own description.",
		}},
		{System: "megadrive", Game: gamelist.Game{
			Path: "./Ghost.zip",
			Name: "A game no folder holds",
			Desc: "Known to the registry alone.",
		}},
	}}

	registryFolder := t.TempDir()
	if err := registry.Save(registryFolder, reg); err != nil {
		t.Fatalf("failed to write the registry: %v", err)
	}
	return reg, registryFolder
}

// romsFolderForSending writes a ROMs folder whose megadrive gamelist holds two
// of the registry's games: Sonic with nothing but its name, and Streets of Rage
// with a description and a genre of its own. Sonic is what a fill has something
// to do about, Streets of Rage what only a replacement touches — and the genre
// the registry does not know is what neither may ever erase.
func romsFolderForSending(t *testing.T) string {
	t.Helper()

	romsFolder := t.TempDir()
	system := filepath.Join(romsFolder, "megadrive")
	if err := os.MkdirAll(system, 0o755); err != nil {
		t.Fatalf("failed to set up the ROMs folder: %v", err)
	}
	games := []gamelist.Game{
		{Path: "./Sonic.zip", Name: "Sonic the Hedgehog"},
		{Path: "./Streets.zip", Name: "Streets of Rage", Desc: "Three fighters clean up the city.", Genre: "Beat 'em up"},
	}
	if err := gamelist.UpdateFile(filepath.Join(system, "gamelist.xml"), games); err != nil {
		t.Fatalf("failed to write the gamelist: %v", err)
	}
	return romsFolder
}

// sendURLOf builds the URL of a game's send flow, for the given folder and rule.
func sendURLOf(id, folder, mode string) string {
	query := url.Values{sendFolderParam: {folder}, sendModeParam: {mode}}
	return gameURL("megadrive", id) + "/send?" + query.Encode()
}

// sendSubmission is what the confirmation page submits: the folder and the rule
// it was rendered for.
func sendSubmission(folder, mode string) url.Values {
	return url.Values{sendFolderParam: {folder}, sendModeParam: {mode}}
}

// bannerAfter follows the redirect a change answered with — dropping the fragment,
// as a browser does — and returns the text of the confirmation banner on the
// page it lands on. Asserting on the banner rather than on the whole page
// matters here: the folder a send names also appears in the page's own send
// control, so a test looking anywhere would pass without any confirmation at
// all.
func bannerAfter(t *testing.T, h http.Handler, rec *httptest.ResponseRecorder) string {
	t.Helper()

	location, _, _ := strings.Cut(rec.Header().Get("Location"), "#")
	if location == "" {
		t.Fatalf("the change answered no redirect (status %d)", rec.Code)
	}
	body := get(t, h, location).Body.String()

	_, afterID, found := strings.Cut(body, `id="saved"`)
	if !found {
		t.Fatalf("the game page carries no confirmation banner\n--- page ---\n%s", body)
	}
	banner, _, _ := strings.Cut(afterID, "</p>")
	_, text, _ := strings.Cut(banner, ">")
	return html.UnescapeString(text)
}

// localGameOf reads one game back out of a ROMs folder's own gamelist.xml.
func localGameOf(t *testing.T, romsFolder, name string) gamelist.Game {
	t.Helper()

	games, err := gamelist.ParseFile(filepath.Join(romsFolder, "megadrive", "gamelist.xml"))
	if err != nil {
		t.Fatalf("reading the folder's gamelist: %v", err)
	}
	for _, g := range games {
		if g.Name == name {
			return g
		}
	}
	t.Fatalf("no game named %q in the folder's gamelist", name)
	return gamelist.Game{}
}

// denyWritesTo makes a folder read-only, so the next write into it fails.
func denyWritesTo(t *testing.T, folder string) {
	t.Helper()

	if os.Geteuid() == 0 {
		t.Skip("root ignores folder permissions, so the write cannot be made to fail this way")
	}
	if err := os.Chmod(folder, 0o555); err != nil {
		t.Fatalf("failed to make %q read-only: %v", folder, err)
	}
	t.Cleanup(func() { _ = os.Chmod(folder, 0o755) })
}

func TestServeGame_RomsFoldersConfigured_OffersEveryOneOfThemAsAChoice(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	folders := []string{t.TempDir(), t.TempDir()}

	body := get(t, Handler(reg, registryFolder, folders), gameURL("megadrive", "Sonic")).Body.String()

	if !strings.Contains(body, `action="`+gameURL("megadrive", "Sonic")+`/send"`) {
		t.Fatalf("the game page offers no way to send this game to a ROMs folder\n--- page ---\n%s", body)
	}
	for _, folder := range folders {
		if !strings.Contains(body, folder) {
			t.Errorf("the game page does not offer the configured ROMs folder %q as a choice", folder)
		}
	}
}

func TestServeGame_RomsFoldersConfigured_OffersBothTheFillAndTheReplaceRule(t *testing.T) {
	reg, registryFolder := registryForSending(t)

	body := get(t, Handler(reg, registryFolder, []string{t.TempDir()}), gameURL("megadrive", "Sonic")).Body.String()

	for _, mode := range sendModes {
		if !strings.Contains(body, `value="`+mode.Value+`"`) {
			t.Errorf("the game page does not offer the %q rule", mode.Value)
		}
	}
}

func TestServeGame_NoRomsFolderConfigured_NamesTheCommandInsteadOfAnEmptyChoice(t *testing.T) {
	reg, registryFolder := registryForSending(t)

	rec := get(t, Handler(reg, registryFolder, nil), gameURL("megadrive", "Sonic"))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d — having nothing configured is not an error", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if strings.Contains(body, `/send"`) {
		t.Error("the game page offers to send the game with no ROMs folder to send it to")
	}
	if !strings.Contains(body, "config add-roms-folder") {
		t.Error("the game page does not name the command that configures a ROMs folder")
	}
}

func TestServeSendConfirmation_FillRule_NamesTheGameTheFolderAndWhatItCannotLose(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)

	rec := get(t, Handler(reg, registryFolder, []string{romsFolder}),
		sendURLOf("Sonic", romsFolder, sendModeFill))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Sonic the Hedgehog") {
		t.Error("the confirmation does not name the game it is about to send")
	}
	if !strings.Contains(body, romsFolder) {
		t.Error("the confirmation does not name the folder it is about to write to")
	}
	if !strings.Contains(body, "never overwritten") {
		t.Errorf("the confirmation does not say that filling the gaps overwrites nothing\n--- page ---\n%s", body)
	}
	if !strings.Contains(body, `method="post"`) {
		t.Error("the confirmation offers no form actually sending the game")
	}
	if !strings.Contains(body, `href="`+gameURL("megadrive", "Sonic")+`"`) {
		t.Error("the confirmation offers no way back that touches nothing")
	}
}

func TestServeSendConfirmation_ReplaceRule_WarnsThatLocalValuesAreOverwritten(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)

	body := get(t, Handler(reg, registryFolder, []string{romsFolder}),
		sendURLOf("Sonic", romsFolder, sendModeReplace)).Body.String()

	if strings.Contains(body, "never overwritten") {
		t.Error("the confirmation of a replacement claims nothing is overwritten")
	}
	if !strings.Contains(strings.ToLower(body), "overwrit") {
		t.Errorf("the confirmation of a replacement does not say values are overwritten\n--- page ---\n%s", body)
	}
	if !strings.Contains(strings.ToLower(body), "cannot be undone") {
		t.Error("the confirmation does not warn that the write cannot be undone")
	}
}

func TestServeSendConfirmation_FolderNotConfigured_IsRefused(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)

	rec := get(t, Handler(reg, registryFolder, []string{romsFolder}),
		sendURLOf("Sonic", "/somewhere/else", sendModeFill))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for a folder that is not configured", rec.Code, http.StatusBadRequest)
	}
}

func TestServeSendConfirmation_UnknownRule_IsRefused(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)

	rec := get(t, Handler(reg, registryFolder, []string{romsFolder}),
		sendURLOf("Sonic", romsFolder, "erase-everything"))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for a rule the flow does not know", rec.Code, http.StatusBadRequest)
	}
}

func TestServeSendConfirmation_UnknownGame_AnswersNotFound(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)

	rec := get(t, Handler(reg, registryFolder, []string{romsFolder}),
		sendURLOf("Nothing", romsFolder, sendModeFill))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSend_MethodNeitherGetNorPost_AnswersMethodNotAllowed(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	rec := httptest.NewRecorder()

	Handler(reg, registryFolder, []string{t.TempDir()}).ServeHTTP(rec,
		httptest.NewRequest(http.MethodPut, gameURL("megadrive", "Sonic")+"/send", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); allow != "GET, POST" {
		t.Errorf("Allow = %q, want %q", allow, "GET, POST")
	}
}

func TestSendGame_FillRule_FillsThatGameInThatFolderAlone(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	target, untouched := romsFolderForSending(t), romsFolderForSending(t)
	before := folderFingerprint(t, untouched)

	rec := post(t, Handler(reg, registryFolder, []string{target, untouched}),
		gameURL("megadrive", "Sonic")+"/send", sendSubmission(target, sendModeFill))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d\n--- page ---\n%s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if sonic := localGameOf(t, target, "Sonic the Hedgehog"); sonic.Desc != "A blue hedgehog runs very fast." {
		t.Errorf("the sent game's description = %q, want it filled from the registry", sonic.Desc)
	}
	if streets := localGameOf(t, target, "Streets of Rage"); streets.Desc != "Three fighters clean up the city." {
		t.Errorf("another game of the same folder was touched: its description is now %q", streets.Desc)
	}
	if after := folderFingerprint(t, untouched); after != before {
		t.Error("the other configured ROMs folder was written to, although only one was chosen")
	}
}

func TestSendGame_FillRule_NeverOverwritesAValueTheFolderAlreadyHolds(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)

	post(t, Handler(reg, registryFolder, []string{romsFolder}),
		gameURL("megadrive", "Streets")+"/send", sendSubmission(romsFolder, sendModeFill))

	if streets := localGameOf(t, romsFolder, "Streets of Rage"); streets.Desc != "Three fighters clean up the city." {
		t.Errorf("description = %q, want the folder's own value kept: filling the gaps overwrites nothing", streets.Desc)
	}
}

func TestSendGame_ReplaceRule_OverwritesTheValueTheFolderAlreadyHolds(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)

	post(t, Handler(reg, registryFolder, []string{romsFolder}),
		gameURL("megadrive", "Streets")+"/send", sendSubmission(romsFolder, sendModeReplace))

	streets := localGameOf(t, romsFolder, "Streets of Rage")
	if streets.Desc != "The registry's own description." {
		t.Errorf("description = %q, want the registry's value written over the folder's", streets.Desc)
	}
	if streets.Genre != "Beat 'em up" {
		t.Errorf("genre = %q, want %q kept: the registry holds no genre for this game", streets.Genre, "Beat 'em up")
	}
}

func TestSendGame_Sent_ConfirmsOnTheGamePageNamingTheFolder(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	rec := post(t, handler, gameURL("megadrive", "Sonic")+"/send", sendSubmission(romsFolder, sendModeFill))

	banner := bannerAfter(t, handler, rec)
	if !strings.Contains(banner, romsFolder) {
		t.Errorf("the confirmation %q does not name the folder the game was sent to", banner)
	}
	if !strings.Contains(banner, "filled") {
		t.Errorf("the confirmation %q does not say the folder's gaps were filled", banner)
	}
}

func TestSendGame_ReplaceRule_ConfirmsTheFolderNowHoldsWhatTheRegistryKnows(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	rec := post(t, handler, gameURL("megadrive", "Streets")+"/send", sendSubmission(romsFolder, sendModeReplace))

	banner := bannerAfter(t, handler, rec)
	if !strings.Contains(banner, romsFolder) {
		t.Errorf("the confirmation %q does not name the folder the game was sent to", banner)
	}
	if strings.Contains(banner, "filled") {
		t.Errorf("the confirmation %q describes a replacement as gaps being filled", banner)
	}
}

func TestSendGame_FolderAlreadyUpToDate_SaysSoRatherThanClaimingItWasSent(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	// Sent once, so the second send finds the folder already holding everything.
	post(t, handler, gameURL("megadrive", "Sonic")+"/send", sendSubmission(romsFolder, sendModeFill))
	rec := post(t, handler, gameURL("megadrive", "Sonic")+"/send", sendSubmission(romsFolder, sendModeFill))

	banner := bannerAfter(t, handler, rec)
	if !strings.Contains(banner, "already") {
		t.Errorf("the confirmation %q does not state the folder was already up to date", banner)
	}
	if strings.Contains(banner, "Sent to") {
		t.Errorf("the confirmation %q claims the game was sent although nothing was written", banner)
	}
}

func TestSendGame_GameAbsentFromTheFoldersGamelist_SaysSoRatherThanClaimingSuccess(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})

	rec := post(t, handler, gameURL("megadrive", "Ghost")+"/send", sendSubmission(romsFolder, sendModeFill))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d — the game exists, the folder simply does not hold it", rec.Code, http.StatusSeeOther)
	}
	banner := bannerAfter(t, handler, rec)
	if !strings.Contains(banner, "Not sent") {
		t.Errorf("the confirmation %q does not say the game was not sent", banner)
	}
	if !strings.Contains(banner, romsFolder) {
		t.Errorf("the confirmation %q does not name the folder that does not hold the game", banner)
	}
}

func TestSendGame_FolderNotConfigured_IsRefusedAndWritesNothing(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	configured, elsewhere := romsFolderForSending(t), romsFolderForSending(t)
	before := folderFingerprint(t, elsewhere)

	rec := post(t, Handler(reg, registryFolder, []string{configured}),
		gameURL("megadrive", "Sonic")+"/send", sendSubmission(elsewhere, sendModeFill))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for a folder that is not configured", rec.Code, http.StatusBadRequest)
	}
	if after := folderFingerprint(t, elsewhere); after != before {
		t.Error("a folder the configuration does not name was written to anyway")
	}
}

func TestSendGame_UnknownRule_IsRefusedAndWritesNothing(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)
	before := folderFingerprint(t, romsFolder)

	rec := post(t, Handler(reg, registryFolder, []string{romsFolder}),
		gameURL("megadrive", "Sonic")+"/send", sendSubmission(romsFolder, ""))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for a submission naming no rule", rec.Code, http.StatusBadRequest)
	}
	if after := folderFingerprint(t, romsFolder); after != before {
		t.Error("a refused submission wrote to the ROMs folder anyway")
	}
}

func TestSendGame_SubmissionFromAnotherSite_IsRefusedAndWritesNothing(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)
	before := folderFingerprint(t, romsFolder)
	form := sendSubmission(romsFolder, sendModeReplace)
	r := httptest.NewRequest(http.MethodPost, gameURL("megadrive", "Sonic")+"/send", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()

	Handler(reg, registryFolder, []string{romsFolder}).ServeHTTP(rec, r)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if after := folderFingerprint(t, romsFolder); after != before {
		t.Error("a refused submission wrote to the ROMs folder anyway")
	}
}

func TestSendGame_UnknownGame_AnswersNotFound(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)

	rec := post(t, Handler(reg, registryFolder, []string{romsFolder}),
		gameURL("megadrive", "Nothing")+"/send", sendSubmission(romsFolder, sendModeFill))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSendGame_FolderCannotBeWrittenTo_ReRendersTheConfirmationWithTheProblem(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)
	denyWritesTo(t, filepath.Join(romsFolder, "megadrive"))

	rec := post(t, Handler(reg, registryFolder, []string{romsFolder}),
		gameURL("megadrive", "Sonic")+"/send", sendSubmission(romsFolder, sendModeFill))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Not sent") {
		t.Errorf("the page does not say the game was not sent\n--- page ---\n%s", body)
	}
	if !strings.Contains(body, `method="post"`) {
		t.Error("the page does not offer to try the send again from where it failed")
	}
}

func TestSendGame_WhateverTheRule_LeavesTheRegistryFolderUntouched(t *testing.T) {
	reg, registryFolder := registryForSending(t)
	romsFolder := romsFolderForSending(t)
	handler := Handler(reg, registryFolder, []string{romsFolder})
	before := folderFingerprint(t, registryFolder)

	post(t, handler, gameURL("megadrive", "Sonic")+"/send", sendSubmission(romsFolder, sendModeFill))
	post(t, handler, gameURL("megadrive", "Streets")+"/send", sendSubmission(romsFolder, sendModeReplace))

	if after := folderFingerprint(t, registryFolder); after != before {
		t.Error("sending a game wrote into the registry folder — a send only ever reads the registry")
	}
}
