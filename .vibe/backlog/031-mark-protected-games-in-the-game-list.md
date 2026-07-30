---
status: in_progress
---
# Mark Protected Games In The Game List

## Description
A game can be fully protected against `update` overwrites (decision from item 018, exposed as `registry.Entry.FullyProtected()`), but the per-system game list — the card grid served by `serveSystem` in `internal/webui/system.go` — currently renders every card the same way regardless of protection state. Users browsing a system's games have no way to tell which ones are protected without opening each game's page.

## Acceptance Criteria
- [ ] A game whose `FullyProtected()` is true is visually distinguished from unprotected games in the per-system game list (e.g. a badge or icon on its card)
- [ ] The distinction is visible without opening the game's own page
- [ ] Unprotected games, and games with only some fields hand-edited (not fully protected), do not show the fully-protected marker
- [ ] The marker has an accessible label (not conveyed by color alone), consistent with the existing `hand-edited` marker on the game page

## Notes
`gameCards` in `internal/webui/system.go` builds the `card` view models consumed by the `grid`/`card` template block (around line 176-186 of that file); add a protected flag there sourced from `registry.Entry.FullyProtected()`. Reuse the visual language already used for the "hand-edited" per-field marker on the game detail page (`meta__manual` in `internal/webui/webui.go`) rather than inventing a new style.
