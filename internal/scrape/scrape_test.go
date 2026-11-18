package scrape

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// pngBytes is the start of a real PNG file: enough for content sniffing to
// recognize an image.
var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

const gamePage = "https://retrogaming.neolao.com/switch/games/Bluey%E2%80%99s%20Happy%20Snaps/index.html"

// fakeSite starts a TLS server answering for every supported host, whatever
// name the request was addressed to, and returns a Fetcher wired to it — the
// tests never reach the real network.
func fakeSite(t *testing.T, handler http.Handler) *Fetcher {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)

	transport := srv.Client().Transport.(*http.Transport).Clone()
	transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	transport.TLSClientConfig.ServerName = "example.com"
	transport.DialTLSContext = nil
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, srv.Listener.Addr().String())
	}
	return New(transport)
}

// fixture serves the named testdata file as an HTML page.
func fixture(t *testing.T, name string) http.HandlerFunc {
	t.Helper()
	page, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	}
}

func TestLookup_UnsupportedAddress_IsRefusedBeforeAnyRequest(t *testing.T) {
	requested := false
	fetcher := fakeSite(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requested = true }))

	for _, raw := range []string{
		"https://example.com/game.html",
		"http://retrogaming.neolao.com/switch/games/x/index.html",
		"https://retrogaming.neolao.com:8443/switch/games/x/index.html",
		"https://user@retrogaming.neolao.com/switch/games/x/index.html",
		"https://192.168.1.1/admin",
		"retrogaming.neolao.com/switch/games/x/index.html",
		"::not a url",
		"",
	} {
		if _, err := fetcher.Lookup(context.Background(), raw); !errors.Is(err, ErrUnsupportedSite) {
			t.Errorf("Lookup(%q) error = %v, want ErrUnsupportedSite", raw, err)
		}
	}
	if requested {
		t.Error("an unsupported address still reached the network")
	}
}

func TestLookup_SiteAnswersAnError_ReturnsErrUnreachable(t *testing.T) {
	fetcher := fakeSite(t, http.NotFoundHandler())

	if _, err := fetcher.Lookup(context.Background(), gamePage); !errors.Is(err, ErrUnreachable) {
		t.Errorf("error = %v, want ErrUnreachable", err)
	}
}

func TestLookup_RedirectToAnotherSite_IsRefused(t *testing.T) {
	fetcher := fakeSite(t, http.RedirectHandler("https://example.com/elsewhere", http.StatusFound))

	if _, err := fetcher.Lookup(context.Background(), gamePage); !errors.Is(err, ErrUnreachable) {
		t.Errorf("error = %v, want ErrUnreachable", err)
	}
}

func TestLookup_PageTooLarge_ReturnsErrTooLarge(t *testing.T) {
	fetcher := fakeSite(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(bytes.Repeat([]byte("a"), maxPageBytes+1))
	}))

	if _, err := fetcher.Lookup(context.Background(), gamePage); !errors.Is(err, ErrTooLarge) {
		t.Errorf("error = %v, want ErrTooLarge", err)
	}
}

func TestDownload_Image_ReturnsItsBytesAndFilename(t *testing.T) {
	fetcher := fakeSite(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(pngBytes)
	}))

	data, filename, err := fetcher.Download(context.Background(), "https://retrogaming.neolao.com/switch/images/Bluey%E2%80%99s%20Happy%20Snaps.png")

	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if !bytes.Equal(data, pngBytes) || filename != "Bluey’s Happy Snaps.png" {
		t.Errorf("Download() = %d bytes, %q; want the PNG and its decoded filename", len(data), filename)
	}
}

func TestDownload_NotAnImage_ReturnsErrNotAnImage(t *testing.T) {
	fetcher := fakeSite(t, fixture(t, "retrogaming-neolao-game.html"))

	if _, _, err := fetcher.Download(context.Background(), "https://retrogaming.neolao.com/fake.png"); !errors.Is(err, ErrNotAnImage) {
		t.Errorf("error = %v, want ErrNotAnImage", err)
	}
}

func TestDownload_UnsupportedSite_IsRefused(t *testing.T) {
	fetcher := fakeSite(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(pngBytes) }))

	if _, _, err := fetcher.Download(context.Background(), "https://example.com/cover.png"); !errors.Is(err, ErrUnsupportedSite) {
		t.Errorf("error = %v, want ErrUnsupportedSite", err)
	}
}

func TestDownload_ImageTooLarge_ReturnsErrTooLarge(t *testing.T) {
	fetcher := fakeSite(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(pngBytes)
		w.Write(make([]byte, maxImageBytes))
	}))

	if _, _, err := fetcher.Download(context.Background(), "https://retrogaming.neolao.com/big.png"); !errors.Is(err, ErrTooLarge) {
		t.Errorf("error = %v, want ErrTooLarge", err)
	}
}
