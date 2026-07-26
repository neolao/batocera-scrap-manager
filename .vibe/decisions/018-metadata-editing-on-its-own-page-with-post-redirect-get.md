---
date: 2026-07-26
status: accepted
---
# Metadata editing on its own page, with POST/redirect/GET

**Context:** Backlog item 015 adds a pre-filled form to correct a game's text metadata from the
served web UI. Two consulted experts disagreed on where it belongs: on the game page itself
(writing via `POST /game/{system}/{id}`, the resource being edited), or on a page of its own.

**Decision:** The form lives at `GET /game/{system}/{id}/edit` and is saved by
`POST` to that same URL; the game page keeps its read-only metadata list and gains an
"Edit metadata" link right under it. A successful save answers `303 See Other` towards
`/game/{system}/{id}?saved=1#saved`, where the game page renders a `role="status"` banner. A refused
submission is not a redirect: the edit page is re-rendered with the values as submitted, an error
summary and per-field messages.

**Reason:** A refused submission must come back with the user's own values and field-level errors,
which needs a URL that can render the form with a body of its own — on the game page the same
response would have to be both the read-only view and the failed form. POST/redirect/GET keeps a
browser reload from re-submitting the save, and the fragment moves focus to the confirmation banner
without a line of JavaScript. Per-game URLs as the surface to build on were already established by
[`015`](015-real-per-game-pages-in-the-served-web-ui.md).

**Rejected alternatives:**
- *Form inline on the game page, `POST` on the game URL* — the more RESTful shape, but it forces the
  failed-validation response to be the game page in a half-form state.
- *`<details>` disclosure on the game page* — snaps shut on every round-trip and cannot be reopened
  server-side after a refused save.
