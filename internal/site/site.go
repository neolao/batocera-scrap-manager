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
<title>Registry</title>
</head>
<body>
<h1>Registry</h1>
{{if not .}}
<p>No games in the registry yet.</p>
{{else}}
{{range .}}
{{$sys := .Name}}
<h2>{{$sys}}</h2>
<ul>
{{range .Games}}
<li>
<h3>{{.Name}}</h3>
<p>{{.Desc}}</p>
{{if .Image}}<img src="{{printf "%s/%s" $sys .Image}}" alt="{{.Name}}">{{end}}
</li>
{{end}}
</ul>
{{end}}
{{end}}
</body>
</html>
`))
