package scrape

import (
	"strings"

	"golang.org/x/net/html"
)

// find returns the first element under n, n included, matching match.
func find(n *html.Node, match func(*html.Node) bool) *html.Node {
	if n.Type == html.ElementNode && match(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := find(c, match); found != nil {
			return found
		}
	}
	return nil
}

// findAll returns every element under n, in document order, matching match.
func findAll(n *html.Node, match func(*html.Node) bool) []*html.Node {
	var found []*html.Node
	if n.Type == html.ElementNode && match(n) {
		found = append(found, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		found = append(found, findAll(c, match)...)
	}
	return found
}

// hasClass matches an element whose class attribute lists class.
func hasClass(class string) func(*html.Node) bool {
	return func(n *html.Node) bool {
		return strings.Contains(" "+attr(n, "class")+" ", " "+class+" ")
	}
}

// attr returns n's attribute key, or nothing.
func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// text returns n's text content with its whitespace collapsed.
func text(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}
