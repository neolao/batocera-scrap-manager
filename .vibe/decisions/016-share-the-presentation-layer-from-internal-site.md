---
date: 2026-07-26
status: accepted
---
# Share the presentation layer from internal/site

**Context:** The served web UI (backlog item 014) must look like the existing static site: same theme, same card grid, same sticky system navigation, same rating stars, same percent-encoded media paths, same "missing media file" handling. All of that lives in `internal/site` — the grouping/formatting helpers (`groupBySystem`, `formatStars`, `formatYear`, `escapeMediaPath`, `mediaFileExists`) and a ~300-line inline `<style>` block, all unexported.

**Decision:** `internal/site` becomes the shared presentation layer of the registry: its view types, grouping/formatting helpers and its stylesheet are exported, and `internal/webui` imports them. No presentation code is copied into `internal/webui`, and no new intermediate package is introduced.

**Reason:** Two renderers of the same registry data is the predictable failure mode of this feature — a theme or formatting fix applied to one and not the other. `internal/site` is already the only module whose role is "turn the registry into HTML", so widening it from "generates a static file" to "renders the registry as HTML, for a file or for a response" is a smaller conceptual move than inventing a third package. `site_test.go` exercises `Generate` only (no unexported symbol), so exporting them does not touch the existing test suite, which stays as the regression guard proving the static site's output is unchanged.

**Rejected alternatives:**
- A new `internal/view` package holding the shared helpers and CSS: one more package and one more import hop, for a boundary that carries no extra meaning — `internal/site` would be reduced to a thin file writer.
- Duplicating the helpers and the stylesheet into `internal/webui`: guarantees visual drift between the static site and the served UI as soon as either is touched.
- Having `internal/webui` serve HTML rendered by `site.Generate`: the static site is a single page with modals, which item 014 deliberately does not reproduce — see [`decisions/015`](015-real-per-game-pages-in-the-served-web-ui.md).
