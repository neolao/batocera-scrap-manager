// Package site generates a static HTML site to browse the registry's
// content (games grouped by system, with name, description, and jaquette)
// in a web browser.
package site

import (
	"bytes"
	"html/template"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// systemView groups a system's games for rendering by indexTemplate.
type systemView struct {
	Name  string
	Games []gameView
}

// gameView wraps a gamelist.Game with fields precomputed for rendering:
// properly percent-encoded media paths and human-readable rating/release
// year, so the template stays free of formatting logic.
type gameView struct {
	gamelist.Game
	ImagePath string
	VideoPath string
	Stars     string
	Year      string
}

// escapeMediaPath builds a relative URL from system and a media path (as
// found in gamelist.xml), percent-encoding each path segment. Go's
// html/template contextual auto-escaping deliberately leaves reserved
// characters such as '[' and ']' untouched in URL attributes, which some
// HTTP servers mishandle; encoding them explicitly here avoids that.
func escapeMediaPath(system, relPath string) string {
	segments := strings.Split(system+"/"+relPath, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}

// formatStars renders a gamelist rating (a decimal string between 0 and 1)
// as a 5-star string, or an empty string if rating is missing or invalid.
func formatStars(rating string) string {
	r, err := strconv.ParseFloat(rating, 64)
	if err != nil {
		return ""
	}
	if r < 0 {
		r = 0
	}
	if r > 1 {
		r = 1
	}
	filled := int(math.Round(r * 5))
	return strings.Repeat("★", filled) + strings.Repeat("☆", 5-filled)
}

// formatYear extracts the year from a gamelist release date
// (EmulationStation's "YYYYMMDDTHHMMSS" format), or an empty string if
// releaseDate is missing or does not start with a 4-digit year.
func formatYear(releaseDate string) string {
	if len(releaseDate) < 4 {
		return ""
	}
	year := releaseDate[:4]
	for _, c := range year {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return year
}

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
	if err := indexTemplate.Execute(&buf, groupBySystem(reg.Entries, registryFolder)); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(registryFolder, "index.html"), buf.Bytes(), 0o644)
}

// mediaFileExists reports whether the media file referenced by a game's
// Image/Video field actually exists at <registryFolder>/<system>/<relPath>.
// A game's metadata can reference a file that is missing from the registry
// folder (partial scrape, manual cleanup, etc.); linking to it anyway would
// produce a broken image/video in the browser instead of the placeholder.
func mediaFileExists(registryFolder, system, relPath string) bool {
	info, err := os.Stat(filepath.Join(registryFolder, system, filepath.FromSlash(relPath)))
	return err == nil && !info.IsDir()
}

// groupBySystem groups entries by system, sorted by system name and then by
// game name within each system, for deterministic output.
func groupBySystem(entries []registry.Entry, registryFolder string) []systemView {
	bySystem := map[string][]gamelist.Game{}
	for _, e := range entries {
		bySystem[e.System] = append(bySystem[e.System], e.Game)
	}

	names := make([]string, 0, len(bySystem))
	for name := range bySystem {
		names = append(names, name)
	}
	sort.Strings(names)

	systems := make([]systemView, 0, len(names))
	for _, name := range names {
		games := bySystem[name]
		sort.Slice(games, func(i, j int) bool { return games[i].Name < games[j].Name })

		views := make([]gameView, len(games))
		for i, g := range games {
			view := gameView{Game: g, Stars: formatStars(g.Rating), Year: formatYear(g.ReleaseDate)}
			if g.Image != "" && mediaFileExists(registryFolder, name, g.Image) {
				view.ImagePath = escapeMediaPath(name, g.Image)
			}
			if g.Video != "" && mediaFileExists(registryFolder, name, g.Video) {
				view.VideoPath = escapeMediaPath(name, g.Video)
			}
			views[i] = view
		}
		systems = append(systems, systemView{Name: name, Games: views})
	}
	return systems
}

var indexTemplate = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Registry</title>
<style>
:root {
  --bg: #0a0d14;
  --bg-card: #141a24;
  --ink: #e8f1f2;
  --ink-dim: #94a3b0;
  --cyan: #4df0ff;
  --magenta: #ff3ec8;
  --line: rgba(77, 240, 255, 0.25);
  --radius: 10px;
  --font-display: ui-monospace, "Cascadia Code", "SFMono-Regular", Menlo, Consolas, "Liberation Mono", monospace;
  --font-body: Georgia, "Iowan Old Style", "Palatino Linotype", "Book Antiqua", serif;
}
* , *::before, *::after { box-sizing: border-box; }
html { scroll-behavior: smooth; }
body {
  margin: 0;
  background: var(--bg);
  color: var(--ink);
  font-family: var(--font-body);
  overflow-x: hidden;
  position: relative;
}
body::before {
  content: "";
  position: fixed;
  inset: 0;
  pointer-events: none;
  background: repeating-linear-gradient(rgba(255, 255, 255, 0.025) 0px, transparent 1px, transparent 2px),
    radial-gradient(ellipse at center, transparent 55%, rgba(0, 0, 0, 0.6) 100%);
  z-index: 50;
}
img { max-width: 100%; display: block; }
.marquee {
  text-align: center;
  padding: 2.5rem 1rem 1.5rem;
}
.marquee h1 {
  margin: 0;
  font-family: var(--font-display);
  font-size: clamp(1.5rem, 5vw, 2.5rem);
  letter-spacing: 0.35em;
  text-transform: uppercase;
  color: var(--ink);
  text-shadow: 0 0 6px var(--cyan), 0 0 18px var(--cyan), 0 0 2px var(--magenta);
}
.console {
  position: sticky;
  top: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  gap: 0.75rem 1rem;
  padding: 0.75rem 1rem;
  background: rgba(10, 13, 20, 0.85);
  backdrop-filter: blur(10px);
  border-bottom: 2px solid var(--cyan);
  box-shadow: 0 0 12px rgba(77, 240, 255, 0.35);
}
.console__brand {
  font-family: var(--font-display);
  font-size: 0.8rem;
  letter-spacing: 0.2em;
  color: var(--cyan);
  text-decoration: none;
  white-space: nowrap;
  flex: 0 0 auto;
}
.console__systems {
  display: flex;
  flex: 1 1 auto;
  min-width: 0;
  gap: 0.5rem;
  overflow-x: auto;
  scrollbar-width: thin;
  -webkit-overflow-scrolling: touch;
  padding-bottom: 0.2rem;
}
.console__systems a {
  flex: 0 0 auto;
  font-family: var(--font-display);
  font-size: 0.75rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  text-decoration: none;
  white-space: nowrap;
  color: var(--ink-dim);
  padding: 0.35rem 0.75rem;
  border: 1px solid var(--line);
  border-radius: 999px;
  transition: color 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease;
}
.console__systems a:hover,
.console__systems a:focus-visible {
  color: var(--bg);
  background: var(--cyan);
  border-color: var(--cyan);
  box-shadow: 0 0 10px var(--cyan);
}
main { max-width: 72rem; margin: 0 auto; padding: 0 1.25rem 2rem; }
.empty-state {
  text-align: center;
  color: var(--ink-dim);
  font-family: var(--font-display);
  padding: 3rem 1rem;
}
.system { padding-top: 2rem; min-width: 0; }
.system__title {
  font-family: var(--font-display);
  text-transform: uppercase;
  letter-spacing: 0.15em;
  color: var(--cyan);
  border-bottom: 1px solid var(--line);
  padding-bottom: 0.5rem;
  margin: 0 0 1.25rem;
}
.system__title::before { content: "\25b8 "; color: var(--magenta); }
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 1.25rem;
}
.card {
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border: 1px solid var(--line);
  border-radius: var(--radius);
  overflow: hidden;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}
.card:hover { transform: translateY(-4px); box-shadow: 0 8px 24px rgba(77, 240, 255, 0.2); }
.card { color: inherit; text-decoration: none; cursor: pointer; }
.card__art {
  aspect-ratio: 4 / 3;
  background: var(--bg-card);
}
.card__art--empty {
  background: repeating-linear-gradient(45deg, rgba(77, 240, 255, 0.08), rgba(77, 240, 255, 0.08) 10px, rgba(255, 62, 200, 0.08) 10px, rgba(255, 62, 200, 0.08) 20px);
}
.card__art img { width: 100%; height: 100%; object-fit: cover; }
.card__body { padding: 0.85rem 1rem 1.1rem; }
.card__name {
  margin: 0 0 0.4rem;
  font-family: var(--font-display);
  font-size: 0.85rem;
  letter-spacing: 0.03em;
  color: var(--ink);
}
.card__desc {
  margin: 0;
  font-size: 0.9rem;
  line-height: 1.4;
  color: var(--ink-dim);
  overflow-wrap: anywhere;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
html:has(.modal:target) { overflow: hidden; }
.modal {
  position: fixed;
  inset: 0;
  z-index: 100;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.15s ease;
}
.modal:target { opacity: 1; pointer-events: auto; }
.modal__backdrop {
  position: absolute;
  inset: 0;
  background: rgba(5, 6, 10, 0.85);
  backdrop-filter: blur(2px);
}
.modal__panel {
  position: relative;
  max-width: 32rem;
  margin: 6vh auto;
  max-height: 88vh;
  overflow-y: auto;
  background: var(--bg-card);
  border: 1px solid var(--cyan);
  border-radius: var(--radius);
  padding: 1.5rem;
  box-shadow: 0 0 30px rgba(77, 240, 255, 0.35);
}
.modal__close {
  position: absolute;
  top: 0.4rem;
  right: 0.75rem;
  font-family: var(--font-display);
  font-size: 1.5rem;
  line-height: 1;
  color: var(--ink-dim);
  text-decoration: none;
}
.modal__close:hover, .modal__close:focus-visible { color: var(--magenta); }
.modal__art {
  margin: 0 0 1rem;
  aspect-ratio: 4 / 3;
  border-radius: calc(var(--radius) - 4px);
  overflow: hidden;
}
.modal__art img { width: 100%; height: 100%; object-fit: cover; }
.modal__name {
  margin: 0 0 0.75rem;
  font-family: var(--font-display);
  letter-spacing: 0.05em;
  color: var(--cyan);
}
.modal__desc { margin: 0 0 1rem; line-height: 1.5; color: var(--ink); }
.modal__video { width: 100%; aspect-ratio: 4 / 3; border-radius: calc(var(--radius) - 4px); margin: 0 0 1rem; background: #000; }
.modal__meta {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.5rem 1rem;
  margin: 0;
  padding: 1rem 0 0;
  border-top: 1px solid var(--line);
  list-style: none;
}
.modal__meta-label {
  display: block;
  font-family: var(--font-display);
  font-size: 0.65rem;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--ink-dim);
}
.modal__meta-value { color: var(--ink); font-size: 0.9rem; }
.back-to-top {
  display: inline-block;
  margin-top: 1.25rem;
  font-family: var(--font-display);
  font-size: 0.7rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  text-decoration: none;
  color: var(--ink-dim);
}
.back-to-top:hover, .back-to-top:focus-visible { color: var(--magenta); }
@media (max-width: 480px) {
  .marquee { padding: 1.75rem 1rem 1rem; }
  .marquee h1 { letter-spacing: 0.2em; }
  .console { padding: 0.6rem 0.75rem; }
  main { padding: 0 0.85rem 1.5rem; }
  .grid { grid-template-columns: 1fr; gap: 1rem; }
  .modal__panel { margin: 0; max-height: 100vh; width: 100%; border-radius: 0; }
}
</style>
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
