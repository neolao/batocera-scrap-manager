package scrape

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// retrogamingNeolaoHost serves one static page per game: a title, a grid of
// labelled values (in French), a cover art, screenshots and a description.
const retrogamingNeolaoHost = "retrogaming.neolao.com"

// parseRetrogamingNeolao reads one game page of retrogamingNeolaoHost.
func parseRetrogamingNeolao(doc *html.Node, page *url.URL) Result {
	var r Result
	if h1 := find(doc, func(n *html.Node) bool { return n.Data == "h1" }); h1 != nil {
		r.Name = text(h1)
	}

	for _, row := range findAll(doc, hasClass("meta-row")) {
		key := find(row, hasClass("meta-key"))
		val := find(row, hasClass("meta-val"))
		if key == nil || val == nil {
			continue
		}
		switch text(key) {
		case "Publication":
			r.ReleaseDate = releaseDate(text(val))
		case "Éditeur":
			r.Publisher = text(val)
		case "Catégories":
			var genres []string
			for _, tag := range findAll(val, hasClass("tag")) {
				genres = append(genres, text(tag))
			}
			r.Genre = strings.Join(genres, ", ")
		}
	}

	if body := find(doc, hasClass("desc-body")); body != nil {
		var paragraphs []string
		for _, p := range findAll(body, func(n *html.Node) bool { return n.Data == "p" }) {
			if t := text(p); t != "" {
				paragraphs = append(paragraphs, t)
			}
		}
		r.Desc = strings.Join(paragraphs, "\n\n")
	}

	r.ImageURL = imageIn(doc, "hero-cover", page)
	r.ThumbnailURL = imageIn(doc, "gallery", page)
	return r
}

// releaseDate turns the page's YYYY-MM-DD into the registry's
// YYYYMMDDT000000, or nothing when the value is not such a date.
func releaseDate(value string) string {
	parts := strings.Split(value, "-")
	if len(parts) != 3 || len(parts[0]) != 4 || len(parts[1]) != 2 || len(parts[2]) != 2 {
		return ""
	}
	date := parts[0] + parts[1] + parts[2]
	for _, c := range date {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return date + "T000000"
}

// imageIn returns the absolute address of the first image inside the first
// element of class, resolved against page.
func imageIn(doc *html.Node, class string, page *url.URL) string {
	container := find(doc, hasClass(class))
	if container == nil {
		return ""
	}
	img := find(container, func(n *html.Node) bool { return n.Data == "img" })
	if img == nil {
		return ""
	}
	src := attr(img, "src")
	if src == "" {
		return ""
	}
	ref, err := url.Parse(src)
	if err != nil {
		return ""
	}
	return page.ResolveReference(ref).String()
}
