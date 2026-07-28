---
status: done
depends_on: [014]
---
# Mobile Navigation Of The Web UI

## Description
The web UI served by `serve` is not usable on a phone. The home page serializes every game of every system into a single page, and the theme's only responsive rule turns the grid into one column below 480px: every game becomes a full-width card with a 4:3 cover art and three lines of description, i.e. about one game per screen. The only navigation offered is a `#system` anchor inside that single page. Games must take up markedly less room individually, and routes are needed to reach a system directly without loading the whole registry.

## Acceptance Criteria
- [x] On a small screen, a game is presented as a compact row (thumbnail, name, year) rather than a full-height card; at least ten games fit on a phone screen
- [x] The web UI's home page is a summary of the systems stating each one's game count, and renders no game individually
- [x] Every system has its own `/system/{name}` route listing its games, paginated, with previous page / next page links
- [x] A page number that does not exist, or an unknown system, answers 404 rather than an empty list
- [x] The links back to a system (breadcrumb, "Back to …", the landing page of a deletion) point at the system's page rather than at an anchor of the home page
- [x] The static `index.html` site stays a single file and inherits the same compact rows
- [x] No page overflows horizontally on a 390px screen, and the controls stay reachable with a thumb

## Notes
Mobile refinement of the feature delivered by item 014. The compact rows live in the shared `internal/site/style.css` stylesheet, so they apply to both renderings; the routes and the pagination only concern `internal/webui`, the static site staying a self-contained artifact (see [`decisions/008`](../../decisions/008-move-consultation-site-to-registry-root.md)). See [`decisions/023`](../../decisions/023-system-pages-with-pagination-replace-the-single-game-list.md).
