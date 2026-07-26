# Module: webui
**Role:** Serves the registry over HTTP — a page listing every game grouped by system, one page per game with its full metadata and media, a themed not-found page, and the media files themselves.
**Files:** `internal/webui/webui.go`, `internal/webui/page.css`
**Exports:**
- `webui.Handler(reg *registry.Registry, registryFolder string) http.Handler` — the whole served site as a single `http.ServeMux`. `reg` is a snapshot: it is rendered as given and never reloaded from disk, so the registry is read once, when [`modules/cli.md`](cli.md)'s `serve` command starts.

**Depends on:** [`modules/registry.md`](registry.md), [`modules/site.md`](site.md)

## Routes
- `GET /` (exact, via the `/{$}` pattern) — the game list: systems sorted by name with an anchored section each (`id="<system>"`), a sticky system navigation bar, and one card per game linking to its own page. Cards carry `loading="lazy"` cover art and never embed a `<video>`, so a registry of hundreds of games does not make the browser fetch every medium on load. An empty registry renders the same page with a "No games in the registry yet." message rather than a blank body.
- `GET /game/{system}/{id}` — one game's page: breadcrumb (`/`, then `/#<system>`), cover art or a labelled "No cover art" placeholder, description or "No description available.", a `<dl>` of the six metadata labels (Rating, Year, Developer, Publisher, Genre, Players) whose rows are **always** rendered — an empty value shows a dash, so the label set stays identical from one game to the next — and a media section listing the video, marquee and thumbnail actually present on disk. `{id}` is a `registry.GameID`, resolved through `(*Registry).FindByID`; an unknown system, an unknown game, or a game requested under the wrong system all fall through to the not-found page.
- `GET /media/` — the registry folder's media files, through `http.StripPrefix` + `http.FileServer` over `fileOnlyFS`. `http.Dir` confines every lookup to the registry folder (neutralizing path traversal), and `fileOnlyFS` additionally refuses directories, so the registry's arborescence is never listed.
- `/` (catch-all) — anything else, including a malformed game URL (`/game/`, `/game/<system>`, `/game/<system>/`, or an extra path segment), renders the themed 404 page naming what was not found, with a link back to the list.

## Rendering
Pages share one `layout` (theme, marquee header) that each template fills with its own `title`/`body` blocks; the stylesheet is `site.StyleSheet` (shared with the static site) plus this module's own `page.css` (breadcrumb, game page, metadata grid, not-found page). `render` executes a template into a buffer *before* writing anything, so a template failure surfaces as a clean 500 instead of a truncated page, and no `Cache-Control: no-store` is ever sent — that directive would defeat the browser's scroll restoration when going back to the list.

Game URLs are built by `gameURL` (each of system and id percent-encoded), media URLs by `mediaURL`, which prefixes `/media/` and **cleans** the path: `gamelist.xml` writes media references as `./images/foo.png`, and served as-is that non-canonical URL makes `http.FileServer` answer a redirect whose `Location` re-escapes the already-escaped path, sending the browser to a file that does not exist. The static site links the same media relatively, where the extra `./` is harmless — hence the cleaning lives here rather than in [`modules/site.md`](site.md).

**Architecture note:** the served UI navigates to real per-game pages instead of reproducing the static site's CSS-only `:target` overlay, and deliberately ships neither — see [`decisions/015`](../decisions/015-real-per-game-pages-in-the-served-web-ui.md); backlog items 015 (edit), 016 (delete) and 017 (media management) build their forms on those URLs. Tests drive the handler through `net/http/httptest` without opening a port.
