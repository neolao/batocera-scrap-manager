---
date: 2026-07-26
status: accepted
---
# Real per-game pages in the served web UI

**Context:** Backlog item 014 adds a `serve` command exposing the registry over HTTP. The existing static site (`internal/site`) already renders every game's full details, but as a CSS-only `:target` modal inside a single `index.html` page (see [`decisions/010`](010-css-only-detail-modal-via-target.md)), so a game has no URL of its own.

**Decision:** The served web UI exposes one real page per game at `/game/<system>/<gameID>`, where `gameID` is the same key the registry already uses on disk (`registry.GameID`), and does not ship the `:target` modal markup at all. The static site keeps its modal, unchanged.

**Reason:** Backlog items 015 (edit metadata), 016 (delete with confirmation) and 017 (media upload/delete) all need forms posting to a stable, per-game URL, and 016 explicitly requires a confirmation step — impossible to build correctly in a JavaScript-free UI without a distinct URL to navigate to. A game URL that a `404` can be returned for is also what acceptance criterion 4 of item 014 demands. Serving both a modal and a detail route would give the same content two ways to open, which is a UX trap and doubles the templates to maintain.

**Rejected alternatives:**
- Serving the generated `index.html` as-is from the HTTP server: zero interaction possible, no per-game URL, no `404` semantics — leaves items 015–017 with nothing to build on.
- Keeping the `:target` modal in the served UI in addition to the detail pages: duplicate rendering of the same data, and back-navigation semantics differ between the two.
