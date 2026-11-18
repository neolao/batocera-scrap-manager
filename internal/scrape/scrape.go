// Package scrape reads a game's metadata and media off a supported web page.
// It is the only place the tool reaches the network: an address is checked
// against the sites it knows how to read before any request is made, so a
// pasted URL can never make the server contact an arbitrary host.
package scrape

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// The ways a lookup or a download can fail, for a caller to tell the user
// which one happened.
var (
	// ErrUnsupportedSite is an address naming no site this package reads, or
	// not over plain https on its default port.
	ErrUnsupportedSite = errors.New("this address is not on a supported site")
	// ErrUnreachable is a site that could not be reached, answered an error,
	// or redirected elsewhere.
	ErrUnreachable = errors.New("the site could not be read")
	// ErrTooLarge is an answer larger than what a page or an image may weigh.
	ErrTooLarge = errors.New("the answer is too large")
	// ErrNotAnImage is a download whose content is not an image.
	ErrNotAnImage = errors.New("this is not an image")
)

// Size and time bounds of one request: a game page is a few kilobytes, a
// cover art a few hundred.
const (
	maxPageBytes   = 2 << 20
	maxImageBytes  = 20 << 20
	requestTimeout = 20 * time.Second
)

// Result is what a page says about a game, in the registry's own formats.
// A field the page does not state is left empty. ImageURL and ThumbnailURL
// are absolute addresses, to pass to Download.
type Result struct {
	Name         string
	Desc         string
	ReleaseDate  string
	Publisher    string
	Genre        string
	ImageURL     string
	ThumbnailURL string
}

// site is one web site this package knows how to read.
type site struct {
	host  string
	parse func(doc *html.Node, page *url.URL) Result
}

// sites lists every supported site. Adding one is adding its parser here.
var sites = []site{
	{host: retrogamingNeolaoHost, parse: parseRetrogamingNeolao},
}

// Fetcher reads supported pages and their images over HTTP.
type Fetcher struct {
	client *http.Client
}

// New returns a Fetcher sending its requests through transport, or through
// http.DefaultTransport when transport is nil. Redirects are followed only
// while they stay on the same supported site.
func New(transport http.RoundTripper) *Fetcher {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &Fetcher{client: &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			if _, err := supportedSite(req.URL.String()); err != nil {
				return err
			}
			if req.URL.Host != via[0].URL.Host {
				return errors.New("redirected to another site")
			}
			return nil
		},
	}}
}

// supportedSite parses raw and returns the site it belongs to, refusing
// anything but an https address on a known host and its default port.
func supportedSite(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return nil, ErrUnsupportedSite
	}
	for _, s := range sites {
		if u.Hostname() == s.host {
			return u, nil
		}
	}
	return nil, ErrUnsupportedSite
}

// parserOf returns the parser of u's site, which supportedSite already
// accepted.
func parserOf(u *url.URL) func(*html.Node, *url.URL) Result {
	for _, s := range sites {
		if u.Hostname() == s.host {
			return s.parse
		}
	}
	return nil
}

// Lookup reads the game page at raw and returns what it says.
func (f *Fetcher) Lookup(ctx context.Context, raw string) (Result, error) {
	u, err := supportedSite(raw)
	if err != nil {
		return Result{}, err
	}
	body, err := f.get(ctx, u, maxPageBytes)
	if err != nil {
		return Result{}, err
	}
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	return parserOf(u)(doc, u), nil
}

// Download fetches the image at raw, which must be on a supported site, and
// returns its content with the file name its address ends with.
func (f *Fetcher) Download(ctx context.Context, raw string) ([]byte, string, error) {
	u, err := supportedSite(raw)
	if err != nil {
		return nil, "", err
	}
	body, err := f.get(ctx, u, maxImageBytes)
	if err != nil {
		return nil, "", err
	}
	if !strings.HasPrefix(http.DetectContentType(body), "image/") {
		return nil, "", ErrNotAnImage
	}
	return body, path.Base(u.Path), nil
}

// get reads u's body, refusing an answer other than 200 or one larger than
// limit.
func (f *Fetcher) get(ctx context.Context, u *url.URL, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrUnreachable, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	if int64(len(body)) > limit {
		return nil, ErrTooLarge
	}
	return body, nil
}
