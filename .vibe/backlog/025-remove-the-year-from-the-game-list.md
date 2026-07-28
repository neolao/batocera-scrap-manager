---
status: todo
---
# Remove The Year From The Game List

## Description
Each card of the game list currently shows the release year under the title. That year adds nothing when browsing a system: the list is used to find a game by its name and its box art, and the year only crowds the card. The year must disappear from the cards of the game list, while remaining visible where a single game is detailed.

## Acceptance Criteria
- [ ] A game card in the list of a system shows no release year, whether the game has a release date or not
- [ ] The detailed view of a game still shows its year when a release date is known
- [ ] Editing the year of a game still works and the new value shows in the detailed view

## Notes
The year is rendered on the cards in two places: the static site template (`internal/site/site.go`, the `card__meta` span) and the web UI system page (`internal/webui/system.go`). Both must drop it. The detail rendering (`modal__meta` in `internal/site/site.go`) keeps it, so `Year` stays on the view model and `site.FormatYear` / `site.ReleaseDateFromYear` stay untouched. Check whether the `card__meta` rule in `internal/site/style.css` and `internal/webui/page.css` becomes unused once the span disappears — if so, remove it too.
