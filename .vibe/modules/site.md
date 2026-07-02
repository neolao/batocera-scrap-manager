# Module: site
**Role:** Generates a static HTML site to browse the registry's content (games grouped by system, with name, description, and jaquette) in a web browser.
**Files:** `internal/site/site.go`
**Exports:**
- `site.Generate(reg *registry.Registry, registryFolder string) error` — groups `reg`'s entries by system (sorted alphabetically, games sorted by name within each system for deterministic output) and writes `<registryFolder>/index.html`, directly at the registry's root. A game without a jaquette is rendered without an `<img>` tag (no broken image). An empty registry still produces a valid page, with a "No games in the registry yet." message. Media (jaquette) references are resolved relative to the generated page as `<system>/<Image>`, mirroring where `registry.ImportFromRomsFolder` copies media under the registry folder.

**Depends on:** [`modules/registry.md`](registry.md), [`modules/gamelist.md`](gamelist.md)

**Architecture note:** Added for backlog item 007. Rendering uses Go's `html/template` (not `text/template`) so game names/descriptions are auto-escaped, since they originate from user-editable `gamelist.xml` files. Regeneration is not exposed as its own CLI command — see [`modules/cli.md`](cli.md) and [`decisions/006`](../decisions/006-auto-regenerate-html-site-on-update.md): it is instead triggered automatically at the end of the `update` command, the only command that actually mutates the registry. The generated page moved from a `site/` subfolder to the registry root for backlog item 008 — see [`decisions/008`](../decisions/008-move-consultation-site-to-registry-root.md); a `site/` folder left over from a previous version is never auto-deleted, by explicit product decision.
