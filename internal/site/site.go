// Package site generates a static HTML site to browse the registry's
// content (games grouped by system, with name, description, and jaquette)
// in a web browser.
package site

import (
	"bytes"
	_ "embed"
	"html/template"
	"os"
	"path/filepath"

	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

//go:embed modal.css
var modalCSS string

// Generate writes a static HTML site directly at registryFolder/index.html
// listing every entry of reg, grouped by system, with each game's name,
// description, and jaquette (when available). An empty registry still
// produces a valid site, with a message indicating there is nothing to show
// yet.
func Generate(reg *registry.Registry, registryFolder string) error {
	if err := os.MkdirAll(registryFolder, 0o755); err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := indexTemplate.Execute(&buf, GroupBySystem(reg.Entries, registryFolder)); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(registryFolder, "index.html"), buf.Bytes(), 0o644)
}

// indexTemplate renders the whole static site as a single page. Its
// stylesheet is the shared StyleSheet plus the modal rules, which only this
// page uses — the served web UI navigates to real per-game pages instead
// (see decisions/015).
var indexTemplate = template.Must(template.New("index").Funcs(template.FuncMap{
	"stylesheet": func() template.CSS { return StyleSheet + template.CSS(modalCSS) },
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Registry</title>
<style>{{stylesheet}}</style>
</head>
<body id="top">
<header class="marquee">
<h1>Registry</h1>
</header>
{{if not .}}
<p class="empty-state">No games in the registry yet.</p>
{{else}}
<nav class="console">
<a class="console__brand" href="#top">Registry</a>
<div class="console__systems">
{{range .}}<a href="#{{.Name}}">{{.Name}}</a>
{{end}}
</div>
</nav>
<main>
{{range .}}
{{$sys := .Name}}
<section id="{{$sys}}" class="system">
<h2 class="system__title">{{$sys}}</h2>
<div class="grid">
{{range $i, $g := .Games}}
<a class="card" href="#modal-{{$sys}}-{{$i}}">
<div class="card__art{{if not $g.ImagePath}} card__art--empty{{end}}">
{{if $g.ImagePath}}<img src="{{$g.ImagePath}}" alt="{{$g.Name}}" loading="lazy">{{end}}
</div>
<div class="card__body">
<h3 class="card__name">{{$g.Name}}</h3>
{{if $g.Year}}<span class="card__meta">{{$g.Year}}</span>{{end}}
<p class="card__desc">{{$g.Desc}}</p>
</div>
</a>
{{end}}
</div>
{{range $i, $g := .Games}}
<div class="modal" id="modal-{{$sys}}-{{$i}}" role="dialog" aria-modal="true">
<a class="modal__backdrop" href="#_modal-close" aria-label="Close"></a>
<div class="modal__panel">
<a class="modal__close" href="#_modal-close" aria-label="Close">&times;</a>
{{if $g.ImagePath}}<div class="modal__art"><img src="{{$g.ImagePath}}" alt="{{$g.Name}}"></div>{{end}}
{{if $g.VideoPath}}<video class="modal__video" src="{{$g.VideoPath}}" controls muted loop playsinline preload="none"></video>{{end}}
<h3 class="modal__name">{{$g.Name}}</h3>
<p class="modal__desc">{{$g.Desc}}</p>
<ul class="modal__meta">
{{if $g.Stars}}<li><span class="modal__meta-label">Rating</span><span class="modal__meta-value">{{$g.Stars}}</span></li>{{end}}
{{if $g.Year}}<li><span class="modal__meta-label">Year</span><span class="modal__meta-value">{{$g.Year}}</span></li>{{end}}
{{if $g.Developer}}<li><span class="modal__meta-label">Developer</span><span class="modal__meta-value">{{$g.Developer}}</span></li>{{end}}
{{if $g.Publisher}}<li><span class="modal__meta-label">Publisher</span><span class="modal__meta-value">{{$g.Publisher}}</span></li>{{end}}
{{if $g.Genre}}<li><span class="modal__meta-label">Genre</span><span class="modal__meta-value">{{$g.Genre}}</span></li>{{end}}
{{if $g.Players}}<li><span class="modal__meta-label">Players</span><span class="modal__meta-value">{{$g.Players}}</span></li>{{end}}
</ul>
</div>
</div>
{{end}}
<a class="back-to-top" href="#top">&#9650; Back to top</a>
</section>
{{end}}
</main>
{{end}}
</body>
</html>
`))
