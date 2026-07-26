package webui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

const sonicProtectURL = "/game/megadrive/Sonic/protect"

// protectForm is what the game page's control submits: the state the user
// asked for, never "flip whatever it is now".
func protectForm(state string) url.Values {
	return url.Values{protectedParam: {state}}
}

// allEditableMarks names every field a fully protected game carries a mark
// for, as the registry stores them.
var allEditableMarks = []string{
	"name", "desc", "rating", "release_date", "developer", "publisher", "genre", "players",
}

func TestHandler_Protect_Game_RedirectsBackAndProtectsEveryField(t *testing.T) {
	reg, folder := savedRegistry(t)

	rec := post(t, Handler(reg, folder), sonicProtectURL, protectForm("on"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 (body: %s)", rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); !strings.HasPrefix(location, sonicGameURL) {
		t.Errorf("Location = %q, want it to lead back to %q", location, sonicGameURL)
	}
	stored := gameFile(t, folder)
	for _, mark := range allEditableMarks {
		if !strings.Contains(stored, `"`+mark+`"`) {
			t.Errorf("game file does not mark %q as protected, got: %s", mark, stored)
		}
	}
}

func TestHandler_Protect_Game_ChangesNoStoredValue(t *testing.T) {
	reg, folder := savedRegistry(t)
	before := gameFile(t, folder)

	post(t, Handler(reg, folder), sonicProtectURL, protectForm("on"))

	after := gameFile(t, folder)
	for _, value := range []string{"Sonic the Hedghog", "Fast.", "0.85", "19910623T000000", "Sonic Team", "Sega", "Platform"} {
		if !strings.Contains(after, value) {
			t.Errorf("game file lost %q, want protecting to change no value.\nbefore: %s\nafter: %s", value, before, after)
		}
	}
}

func TestHandler_GamePage_ProtectedGame_SaysSoAndOffersToLiftIt(t *testing.T) {
	reg, folder := savedRegistry(t)
	h := Handler(reg, folder)
	post(t, h, sonicProtectURL, protectForm("on"))

	body := get(t, h, sonicGameURL).Body.String()

	if !strings.Contains(body, "Protected — updates leave every field alone.") {
		t.Errorf("the game page does not state it is protected, got: %s", body)
	}
	if !strings.Contains(body, "Let updates overwrite this game") {
		t.Errorf("the game page does not offer to lift the protection, got: %s", body)
	}
	if strings.Contains(body, "Protect this game") {
		t.Errorf("the game page still offers to protect an already protected game, got: %s", body)
	}
}

func TestHandler_GamePage_ProtectedGame_DropsThePerFieldMarks(t *testing.T) {
	// Six lit badges under a "Protected" line read as noise, or as a
	// contradiction: the game-level sentence already says it.
	reg, folder := savedRegistry(t)
	h := Handler(reg, folder)
	post(t, h, sonicProtectURL, protectForm("on"))

	body := get(t, h, sonicGameURL).Body.String()

	if strings.Contains(body, "hand-edited") {
		t.Errorf("the game page still shows per-field marks on a fully protected game, got: %s", body)
	}
}

func TestHandler_GamePage_UnprotectedGame_SaysSoAndOffersToProtectIt(t *testing.T) {
	reg, folder := savedRegistry(t)

	body := get(t, Handler(reg, folder), sonicGameURL).Body.String()

	if !strings.Contains(body, "Not protected — updates can overwrite every field.") {
		t.Errorf("the game page does not state it is unprotected, got: %s", body)
	}
	if !strings.Contains(body, "Protect this game") {
		t.Errorf("the game page does not offer to protect it, got: %s", body)
	}
}

func TestHandler_GamePage_PartlyProtectedGame_SaysSoAndKeepsThePerFieldMarks(t *testing.T) {
	reg, folder := savedRegistry(t)
	reg.Entries[0].ManualFields = []string{"genre"}

	body := get(t, Handler(reg, folder), sonicGameURL).Body.String()

	if !strings.Contains(body, "Partly protected — updates leave the hand-edited fields alone.") {
		t.Errorf("the game page does not state it is partly protected, got: %s", body)
	}
	if !strings.Contains(body, "hand-edited") {
		t.Errorf("the game page dropped the per-field mark of a partly protected game, got: %s", body)
	}
	if strings.Contains(body, "Let updates overwrite this game") {
		t.Errorf("the game page offers a bulk lift from the partial state, which would discard which fields were corrected, got: %s", body)
	}
}

func TestHandler_Unprotect_ProtectedGame_ClearsEveryMark(t *testing.T) {
	reg, folder := savedRegistry(t)
	h := Handler(reg, folder)
	post(t, h, sonicProtectURL, protectForm("on"))

	rec := post(t, h, sonicProtectURL, protectForm("off"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 (body: %s)", rec.Code, rec.Body.String())
	}
	if stored := gameFile(t, folder); strings.Contains(stored, "manual_fields") {
		t.Errorf("game file still carries marks, got: %s", stored)
	}
	body := get(t, h, sonicGameURL).Body.String()
	if !strings.Contains(body, "Not protected — updates can overwrite every field.") {
		t.Errorf("the game page does not state the protection was lifted, got: %s", body)
	}
}

func TestHandler_Protect_AlreadyProtectedGameFromAStalePage_StaysProtected(t *testing.T) {
	// A page opened before the game was protected elsewhere submits "on", which
	// must be a no-op — never an accidental lift.
	reg, folder := savedRegistry(t)
	h := Handler(reg, folder)
	post(t, h, sonicProtectURL, protectForm("on"))

	rec := post(t, h, sonicProtectURL, protectForm("on"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 (body: %s)", rec.Code, rec.Body.String())
	}
	body := get(t, h, sonicGameURL).Body.String()
	if !strings.Contains(body, "Protected — updates leave every field alone.") {
		t.Errorf("the game page no longer states the game is protected, got: %s", body)
	}
}

func TestHandler_Unprotect_GameThatWasNotProtected_SucceedsAndChangesNothing(t *testing.T) {
	reg, folder := savedRegistry(t)

	rec := post(t, Handler(reg, folder), sonicProtectURL, protectForm("off"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 (body: %s)", rec.Code, rec.Body.String())
	}
	if stored := gameFile(t, folder); strings.Contains(stored, "manual_fields") {
		t.Errorf("game file carries marks, got: %s", stored)
	}
}

func TestHandler_Protect_Confirmation_ReportsTheResultingState(t *testing.T) {
	reg, folder := savedRegistry(t)
	h := Handler(reg, folder)

	rec := post(t, h, sonicProtectURL, protectForm("on"))

	// A browser drops the fragment before requesting the redirect target.
	target, _, _ := strings.Cut(rec.Header().Get("Location"), "#")
	body := get(t, h, target).Body.String()
	if !strings.Contains(body, "This game is protected — updates will leave its metadata alone.") {
		t.Errorf("the game page does not confirm the protection, got: %s", body)
	}
}

func TestHandler_Unprotect_Confirmation_ReportsTheResultingState(t *testing.T) {
	reg, folder := savedRegistry(t)
	h := Handler(reg, folder)
	post(t, h, sonicProtectURL, protectForm("on"))

	rec := post(t, h, sonicProtectURL, protectForm("off"))

	target, _, _ := strings.Cut(rec.Header().Get("Location"), "#")
	body := get(t, h, target).Body.String()
	if !strings.Contains(body, "This game is no longer protected — updates can overwrite its metadata again.") {
		t.Errorf("the game page does not confirm the lift, got: %s", body)
	}
}

func TestHandler_Protect_MissingState_IsRefusedWithoutChangingAnything(t *testing.T) {
	reg, folder := savedRegistry(t)
	before := gameFile(t, folder)

	rec := post(t, Handler(reg, folder), sonicProtectURL, url.Values{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
	if after := gameFile(t, folder); after != before {
		t.Error("a request that said nothing modified the registry")
	}
}

func TestHandler_Protect_UnrecognizedState_IsRefusedRatherThanDefaulted(t *testing.T) {
	reg, folder := savedRegistry(t)
	before := gameFile(t, folder)

	rec := post(t, Handler(reg, folder), sonicProtectURL, protectForm("maybe"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
	if after := gameFile(t, folder); after != before {
		t.Error("an unrecognized state modified the registry")
	}
}

func TestHandler_Protect_CrossSiteSubmission_IsRefused(t *testing.T) {
	reg, folder := savedRegistry(t)
	before := gameFile(t, folder)
	r := httptest.NewRequest(http.MethodPost, sonicProtectURL, strings.NewReader(protectForm("on").Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()

	Handler(reg, folder).ServeHTTP(rec, r)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if after := gameFile(t, folder); after != before {
		t.Error("a cross-site submission modified the registry")
	}
}

func TestHandler_Protect_UnknownGame_IsNotFoundAndChangesNothing(t *testing.T) {
	reg, folder := savedRegistry(t)
	before := gameFile(t, folder)

	rec := post(t, Handler(reg, folder), "/game/megadrive/Streets/protect", protectForm("on"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body: %s)", rec.Code, rec.Body.String())
	}
	if after := gameFile(t, folder); after != before {
		t.Error("protecting an unknown game modified the registry")
	}
}

func TestHandler_Protect_GameRequestedUnderTheWrongSystem_IsNotFound(t *testing.T) {
	reg, folder := savedRegistry(t)

	rec := post(t, Handler(reg, folder), "/game/nes/Sonic/protect", protectForm("on"))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestHandler_Protect_WrongMethod_IsRefusedNamingTheAllowedOne(t *testing.T) {
	reg, folder := savedRegistry(t)

	rec := get(t, Handler(reg, folder), sonicProtectURL)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405 (body: %s)", rec.Code, rec.Body.String())
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodPost {
		t.Errorf("Allow = %q, want %q: this URL only ever accepts a submission", allow, http.MethodPost)
	}
}

func TestHandler_Protect_RegistryCannotBeWritten_SaysSoAndKeepsTheOldState(t *testing.T) {
	reg, folder := savedRegistry(t)
	if err := os.RemoveAll(filepath.Join(folder, "megadrive")); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(folder, "megadrive"), []byte("blocker"), 0o644); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	h := Handler(reg, folder)

	rec := post(t, h, sonicProtectURL, protectForm("on"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body: %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "not changed") {
		t.Errorf("the page does not say the protection was not changed, got: %s", rec.Body.String())
	}
	if body := get(t, h, sonicGameURL).Body.String(); !strings.Contains(body, "Not protected") {
		t.Error("the served page claims a protection that could not be written to disk")
	}
}

func TestHandler_Protect_SiteCannotBeRegenerated_StillAppliesAndSaysSo(t *testing.T) {
	reg, folder := savedRegistry(t)
	if err := os.Remove(filepath.Join(folder, "index.html")); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	if err := os.Mkdir(filepath.Join(folder, "index.html"), 0o755); err != nil {
		t.Fatalf("failed to prepare the test fixture: %v", err)
	}
	h := Handler(reg, folder)

	rec := post(t, h, sonicProtectURL, protectForm("on"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303: the registry itself was written", rec.Code)
	}
	if !strings.Contains(gameFile(t, folder), "manual_fields") {
		t.Fatal("the game file does not hold the protection")
	}
	target, _, _ := strings.Cut(rec.Header().Get("Location"), "#")
	body := get(t, h, target).Body.String()
	if !strings.Contains(body, "This game is protected") {
		t.Errorf("the game page does not confirm the protection, got: %s", body)
	}
	if !strings.Contains(body, "consultation site") {
		t.Errorf("the game page does not warn that the consultation site is stale, got: %s", body)
	}
}

func TestHandler_Protect_ThenTheEditForm_OffersToHandBackEveryField(t *testing.T) {
	// Selective lifting stays reachable after a bulk protection: that is what
	// makes the missing bulk-lift control in the partial state not a dead end.
	reg, folder := savedRegistry(t)
	h := Handler(reg, folder)
	post(t, h, sonicProtectURL, protectForm("on"))

	body := get(t, h, sonicSaveURL).Body.String()

	for _, key := range []string{"name", "desc", "year", "genre"} {
		if !strings.Contains(body, `id="hand-back-`+key+`"`) {
			t.Errorf("the edit form does not offer to hand %q back, got: %s", key, body)
		}
	}
}

func TestHandler_Protect_ControlSubmitsAnAbsoluteStateNotTheOppositeOfTheCurrentOne(t *testing.T) {
	// The rendered control must name the state it asks for, so a page left open
	// while the state changed elsewhere cannot produce the opposite effect.
	reg, folder := savedRegistry(t)

	body := get(t, Handler(reg, folder), sonicGameURL).Body.String()

	if !strings.Contains(body, `name="`+protectedParam+`" value="on"`) {
		t.Errorf("the control does not carry the state it asks for, got: %s", body)
	}
	if !strings.Contains(body, `action="`+sonicProtectURL+`"`) {
		t.Errorf("the control does not submit to the game's protect URL, got: %s", body)
	}
	if !strings.Contains(body, `method="post"`) {
		t.Errorf("the control is not a real submission, got: %s", body)
	}
}

func TestHandler_Protect_WhatWasWrittenReadsBackAsFullyProtected(t *testing.T) {
	// The state a page reports and the marks a protection writes must come from
	// the same table, or the two would disagree the day a field is added. The
	// check goes through disk on purpose: the handler protects a clone and only
	// swaps it in, so the registry it was handed is deliberately left alone.
	reg, folder := savedRegistry(t)
	post(t, Handler(reg, folder), sonicProtectURL, protectForm("on"))

	reloaded, err := registry.Load(folder)
	if err != nil {
		t.Fatalf("failed to reload the registry: %v", err)
	}
	entry, found := reloaded.FindByID("megadrive", "Sonic")
	if !found {
		t.Fatal("the game is gone from the stored registry")
	}
	if !entry.FullyProtected() {
		t.Errorf("ManualFields = %v, want the stored entry to read as fully protected", entry.ManualFields)
	}
	if original, _ := reg.FindByID("megadrive", "Sonic"); original.FullyProtected() {
		t.Error("the registry handed to the handler was modified in place, want the change applied to a clone")
	}
}
