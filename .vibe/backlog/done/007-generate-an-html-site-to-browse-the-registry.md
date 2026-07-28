---
status: done
---
# Generate An HTML Site To Browse The Registry

## Description
Beyond the metadata and media files it already stores, the registry must also generate a small static HTML site making its content easy to browse from a web browser, without opening the JSON files one by one. That site presents the games grouped by system, with their main information (name, description, cover art, etc.), giving a quick and readable overview of the registry.

## Acceptance Criteria
- [ ] After the registry has been updated or completed, an HTML site is generated (or regenerated) inside the registry folder
- [ ] The user can open that site in a browser and see the list of games grouped by system
- [ ] For every game, the site displays at least its name, its description and its cover art (when available)
- [ ] A system with no game, or a game with no cover art, renders correctly with no error and no broken image

## Notes
Left to be defined: at which point(s) the site is (re)generated (on every `update`/`scrape` command, or through a dedicated command), and the level of detail displayed per game (main fields only, or every available metadata field).
