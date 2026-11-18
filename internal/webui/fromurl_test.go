package webui

import (
	"context"
	"html"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
	"github.com/neolao/batocera-scrap-manager/internal/scrape"
)

const blueyPage = "https://retrogaming.neolao.com/switch/games/Bluey%E2%80%99s%20Happy%20Snaps/index.html"

// pngBytes is the start of a real PNG file: enough for content sniffing.
var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

// fakeRetrogaming serves the real Bluey's Happy Snaps page fixture for any
// .html address and a PNG for any other, whatever host was asked for, and
// counts the requests it received.
func fakeRetrogaming(t *testing.T) (*scrape.Fetcher, *int) {
	t.Helper()
	page, err := os.ReadFile("../scrape/testdata/retrogaming-neolao-game.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	requests := 0
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if strings.HasSuffix(r.URL.Path, ".html") {
			w.Write(page)
			return
		}
		w.Write(pngBytes)
	}))
	t.Cleanup(srv.Close)
	return fetcherTo(srv), &requests
}

// unreachableRetrogaming answers every request with an error.
func unreachableRetrogaming(t *testing.T) *scrape.Fetcher {
	t.Helper()
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	return fetcherTo(srv)
}

// fetcherTo returns a Fetcher sending every request to srv.
func fetcherTo(srv *httptest.Server) *scrape.Fetcher {
	transport := srv.Client().Transport.(*http.Transport).Clone()
	transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	transport.TLSClientConfig.ServerName = "example.com"
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, srv.Listener.Addr().String())
	}
	return scrape.New(transport)
}

// importChoices is what the review form submits: the page address and the
// values ticked.
func importChoices(page string, take ...string) url.Values {
	return url.Values{fromURLParam: {page}, takeParam: take}
}

func TestServeEditForm_OffersToImportFromAURL(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	fetcher, _ := fakeRetrogaming(t)

	body := get(t, newHandler(reg, registryFolder, []string{romsFolder}, fetcher), gameURL("megadrive", "Golden Axe")+"/edit").Body.String()

	if !strings.Contains(body, `action="`+html.EscapeString(gameFromURL("megadrive", "Golden Axe"))+`"`) ||
		!strings.Contains(body, `name="`+fromURLParam+`"`) {
		t.Errorf("the edit form offers no lookup by URL\n--- page ---\n%s", body)
	}
}

func TestServeFillForm_OffersToImportFromAURLForThatRom(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	fetcher, _ := fakeRetrogaming(t)

	body := get(t, newHandler(reg, registryFolder, []string{romsFolder}, fetcher), fillURLFor(romsFolder, "megadrive", "Ghost.zip")).Body.String()

	if !strings.Contains(body, `action="`+fillFromURL+`"`) ||
		!strings.Contains(body, `name="`+retrieveROMParam+`" value="Ghost.zip"`) {
		t.Errorf("the fill-in form offers no lookup by URL for Ghost.zip\n--- page ---\n%s", body)
	}
}

func TestServeGameFromURL_ExistingGame_TicksOnlyEmptyFieldsAndShowsBothValues(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	reg.Entries[0].Game.Genre = "Casse-tête, Aventure, Éducation"
	fetcher, _ := fakeRetrogaming(t)

	rec := get(t, newHandler(reg, registryFolder, []string{romsFolder}, fetcher),
		gameFromURL("megadrive", "Golden Axe")+"?"+fromURLParam+"="+url.QueryEscape(blueyPage))

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\n%s", rec.Code, body)
	}
	if !strings.Contains(body, `name="`+takeParam+`" value="publisher" checked`) {
		t.Error("the empty publisher is not ticked")
	}
	if !strings.Contains(body, `name="`+takeParam+`" value="name">`) {
		t.Error("the name, which already holds a different value, is not offered unticked")
	}
	if strings.Contains(body, `value="genre"`) {
		t.Error("the genre, already identical, is offered as a choice")
	}
	if !strings.Contains(body, "Golden Axe") || !strings.Contains(body, "Bluey&#39;s Happy Snaps") {
		t.Error("the review does not show both the current and the proposed name")
	}
	if !strings.Contains(body, `name="`+takeParam+`" value="image" checked`) || !strings.Contains(body, `name="`+takeParam+`" value="thumbnail" checked`) {
		t.Error("the cover art and the screenshot, absent from the registry, are not ticked")
	}
}

func TestServeGameFromURL_UnsupportedAddress_ExplainsAndReachesNothing(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	fetcher, requests := fakeRetrogaming(t)

	rec := get(t, newHandler(reg, registryFolder, []string{romsFolder}, fetcher),
		gameFromURL("megadrive", "Golden Axe")+"?"+fromURLParam+"="+url.QueryEscape("http://192.168.1.1/admin"))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	if !strings.Contains(rec.Body.String(), "retrogaming.neolao.com") {
		t.Error("the refusal does not name the supported sites")
	}
	if *requests != 0 {
		t.Errorf("%d requests reached the network for an unsupported address", *requests)
	}
}

func TestServeGameFromURL_UnknownGame_AnswersNotFound(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	fetcher, _ := fakeRetrogaming(t)

	rec := get(t, newHandler(reg, registryFolder, []string{romsFolder}, fetcher),
		gameFromURL("megadrive", "Nobody")+"?"+fromURLParam+"="+url.QueryEscape(blueyPage))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestImportGameFromURL_TickedValuesOnly_AreWrittenAndMarkedHandEdited(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	fetcher, _ := fakeRetrogaming(t)
	handler := newHandler(reg, registryFolder, []string{romsFolder}, fetcher)

	rec := post(t, handler, gameFromURL("megadrive", "Golden Axe"), importChoices(blueyPage, "publisher", "release_date", "image"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	reloaded, err := registry.Load(registryFolder)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	entry, _ := reloaded.FindByID("megadrive", "Golden Axe")
	if entry.Game.Name != "Golden Axe" || entry.Game.Desc != "Two warriors against an evil sorcerer." {
		t.Errorf("unticked values changed: %+v", entry.Game)
	}
	if entry.Game.Publisher != "Gameloft" || entry.Game.ReleaseDate != "20261015T000000" {
		t.Errorf("ticked values not written: publisher %q, release date %q", entry.Game.Publisher, entry.Game.ReleaseDate)
	}
	if !slices.Contains(entry.ManualFields, registry.FieldPublisher) || slices.Contains(entry.ManualFields, registry.FieldName) {
		t.Errorf("ManualFields = %v, want publisher marked and name not", entry.ManualFields)
	}
	if entry.Game.Image == "" || entry.Game.Thumbnail != "" {
		t.Fatalf("image %q / thumbnail %q, want only the ticked cover art stored", entry.Game.Image, entry.Game.Thumbnail)
	}
	stored, err := os.ReadFile(filepath.Join(registryFolder, "megadrive", filepath.FromSlash(entry.Game.Image)))
	if err != nil || string(stored) != string(pngBytes) {
		t.Errorf("the cover art file was not stored (err %v)", err)
	}
	if !strings.Contains(rec.Header().Get("Location"), savedParam+"="+savedImported) {
		t.Errorf("Location = %q, want the import confirmation", rec.Header().Get("Location"))
	}
}

func TestImportGameFromURL_NothingTicked_IsRefused(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	fetcher, _ := fakeRetrogaming(t)

	rec := post(t, newHandler(reg, registryFolder, []string{romsFolder}, fetcher), gameFromURL("megadrive", "Golden Axe"), importChoices(blueyPage))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestImportGameFromURL_SiteUnreachable_WritesNothing(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)

	rec := post(t, newHandler(reg, registryFolder, []string{romsFolder}, unreachableRetrogaming(t)),
		gameFromURL("megadrive", "Golden Axe"), importChoices(blueyPage, "publisher"))

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
	reloaded, _ := registry.Load(registryFolder)
	if entry, _ := reloaded.FindByID("megadrive", "Golden Axe"); entry.Game.Publisher != "" {
		t.Errorf("publisher = %q, want nothing written", entry.Game.Publisher)
	}
}

func TestImportGameFromURL_CrossSiteSubmission_IsRefused(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	fetcher, requests := fakeRetrogaming(t)
	form := importChoices(blueyPage, "publisher")
	req := httptest.NewRequest(http.MethodPost, gameFromURL("megadrive", "Golden Axe"), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rec := httptest.NewRecorder()

	newHandler(reg, registryFolder, []string{romsFolder}, fetcher).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden || *requests != 0 {
		t.Errorf("status = %d after %d requests, want %d and none", rec.Code, *requests, http.StatusForbidden)
	}
}

func TestFillFromURL_Review_TicksEveryValueTheSiteHas(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	fetcher, _ := fakeRetrogaming(t)
	query := url.Values{statusFolderParam: {romsFolder}, retrieveSystemParam: {"megadrive"}, retrieveROMParam: {"Ghost.zip"}, fromURLParam: {blueyPage}}

	rec := get(t, newHandler(reg, registryFolder, []string{romsFolder}, fetcher), fillFromURL+"?"+query.Encode())

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\n%s", rec.Code, body)
	}
	for _, key := range []string{"name", "desc", "release_date", "publisher", "genre", "image", "thumbnail"} {
		if !strings.Contains(body, `name="`+takeParam+`" value="`+key+`" checked`) {
			t.Errorf("%s is not ticked for a game with nothing yet", key)
		}
	}
}

func TestFillFromURL_Apply_CreatesTheGameWithItsMediaAndOffersToSendIt(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	fetcher, _ := fakeRetrogaming(t)
	handler := newHandler(reg, registryFolder, []string{romsFolder}, fetcher)
	form := importChoices(blueyPage, "name", "desc", "release_date", "publisher", "genre", "image", "thumbnail")
	form.Set(statusFolderParam, romsFolder)
	form.Set(retrieveSystemParam, "megadrive")
	form.Set(retrieveROMParam, "Ghost.zip")

	rec := post(t, handler, fillFromURL, form)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	reloaded, _ := registry.Load(registryFolder)
	entry, found := reloaded.FindByID("megadrive", "Ghost")
	if !found {
		t.Fatal("Ghost.zip was not created")
	}
	if entry.Game.Path != "./Ghost.zip" || entry.Game.Name != "Bluey's Happy Snaps" || entry.Game.Genre != "Casse-tête, Aventure, Éducation" {
		t.Errorf("entry = %+v", entry.Game)
	}
	if entry.Game.Image == "" || entry.Game.Thumbnail == "" {
		t.Errorf("image %q / thumbnail %q, want both stored", entry.Game.Image, entry.Game.Thumbnail)
	}
	location, _, _ := strings.Cut(rec.Header().Get("Location"), "#")
	page := get(t, handler, location).Body.String()
	if !strings.Contains(html.UnescapeString(page), sendURLFor("megadrive", "Ghost", romsFolder)) {
		t.Error("the system page does not offer to send the created game")
	}
}

func TestFillFromURL_Apply_NameNotTicked_IsRefusedAndCreatesNothing(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	fetcher, _ := fakeRetrogaming(t)
	form := importChoices(blueyPage, "publisher")
	form.Set(statusFolderParam, romsFolder)
	form.Set(retrieveSystemParam, "megadrive")
	form.Set(retrieveROMParam, "Ghost.zip")

	rec := post(t, newHandler(reg, registryFolder, []string{romsFolder}, fetcher), fillFromURL, form)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	reloaded, _ := registry.Load(registryFolder)
	if _, found := reloaded.FindByID("megadrive", "Ghost"); found {
		t.Error("a game was created without a name")
	}
}

func TestFillFromURL_GameAlreadyInRegistry_IsRefused(t *testing.T) {
	reg, registryFolder, romsFolder := registryAndRomsFolderForStatus(t)
	fetcher, requests := fakeRetrogaming(t)
	query := url.Values{statusFolderParam: {romsFolder}, retrieveSystemParam: {"megadrive"}, retrieveROMParam: {"Golden Axe.zip"}, fromURLParam: {blueyPage}}

	rec := get(t, newHandler(reg, registryFolder, []string{romsFolder}, fetcher), fillFromURL+"?"+query.Encode())

	if rec.Code != http.StatusBadRequest || *requests != 0 {
		t.Errorf("status = %d after %d requests, want %d and none", rec.Code, *requests, http.StatusBadRequest)
	}
}
