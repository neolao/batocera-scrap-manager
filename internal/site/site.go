// Package site generates a static HTML site to browse the registry's
// content (games grouped by system, with name, description, and jaquette)
// in a web browser.
package site

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"sort"

	"github.com/neolao/batocera-scrap-manager/internal/gamelist"
	"github.com/neolao/batocera-scrap-manager/internal/registry"
)

// systemView groups a system's games for rendering by indexTemplate.
type systemView struct {
	Name  string
	Games []gamelist.Game
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
	if err := indexTemplate.Execute(&buf, groupBySystem(reg.Entries)); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(registryFolder, "index.html"), buf.Bytes(), 0o644)
}

// groupBySystem groups entries by system, sorted by system name and then by
// game name within each system, for deterministic output.
func groupBySystem(entries []registry.Entry) []systemView {
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
		systems = append(systems, systemView{Name: name, Games: games})
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
  flex-wrap: wrap;
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
}
.console__systems {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
.console__systems a {
  font-family: var(--font-display);
  font-size: 0.75rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  text-decoration: none;
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
}
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
{{range .Games}}
<article class="card">
<div class="card__art{{if not .Image}} card__art--empty{{end}}">
{{if .Image}}<img src="{{printf "%s/%s" $sys .Image}}" alt="{{.Name}}">{{end}}
</div>
<div class="card__body">
<h3 class="card__name">{{.Name}}</h3>
<p class="card__desc">{{.Desc}}</p>
</div>
</article>
{{end}}
</div>
<a class="back-to-top" href="#top">&#9650; Back to top</a>
</section>
{{end}}
</main>
{{end}}
</body>
</html>
`))
