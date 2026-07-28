---
status: done
depends_on: [014]
---
# Edit Metadata From The Web UI

## Description
When a game has been badly scraped (approximate name, truncated description, missing genre), the only way to correct the registry today is to edit the source `gamelist.xml` then run `update` again. The game page served by the web UI must offer a pre-filled form correcting its text metadata and saving it directly into the registry. Once saved, the static `index.html` site is regenerated to stay consistent with the registry.

## Acceptance Criteria
- [ ] A game's page displays a form pre-filled with its editable text fields: name, description, rating, release date, developer, publisher, genre, number of players
- [ ] Once saved, the modified value shows on reloading the page and is present in the game's JSON file inside the registry folder
- [ ] The game's ROM path and media fields are never modified by that form
- [ ] The static `index.html` site is regenerated after a save, and reflects the new value
- [ ] The save is only accepted as a `POST` (a `GET` request modifies nothing) and saving onto an unknown game answers 404

## Notes
On the domain side, add to `internal/registry` a function locating the entry through `indexOf` and applying the editable text fields only, returning `ErrGameNotFound` otherwise — the same error convention as `Remove`. Persistence reuses the `registry.Save` + `site.Generate` sequence already applied by `saveAndGenerateSite` (`internal/cli/common.go`), to be made callable from `internal/webui` without depending on the `cli` package. After the write, redirect with a `303` to the game's page to avoid resubmitting the form on refresh.
