---
status: in_progress
depends_on: [014]
---
# Complete ROMs Folders From The Web UI

## Description
Once metadata has been corrected in the web UI, sending it back to Batocera means leaving the interface and running `scrape` on the command line. The web UI must be able to trigger that same completion of the configured ROMs folders — rewriting the `gamelist.xml` files and copying the media back from the registry — and display its outcome. This is the registry → ROMs counterpart of item 022, which covers the opposite direction.

## Acceptance Criteria
- [ ] Before anything is written, a confirmation page names the ROMs folders that will be modified and offers a way out that touches nothing
- [ ] After confirmation, the `gamelist.xml` files and media of the configured folders are completed from the registry, and a page displays the same summary as the CLI
- [ ] A configured ROMs folder that cannot be found produces an error message naming that folder, and the displayed report faithfully reflects the folders already processed before the failure
- [ ] The operation modifies no file of the registry folder: the games' JSON files and the static site are unchanged once it has run

## Notes
Unlike every existing mutating route, this one writes **outside** the registry, into the user's Batocera folders, rewriting `gamelist.xml` files that are under no version control — hence the upfront confirmation page, modelled on the deletion flow (`internal/webui/delete.go`, [`decisions/018`](../decisions/018-metadata-editing-on-its-own-page-with-post-redirect-get.md)). On the domain side, `registry.CompleteRomsFolder` already exists and reads the registry without ever writing it: the read lock is enough, no `Clone()` nor `internal/store` is involved, and nothing must swap the served snapshot. As in item 022, `webui.Handler` will need the configured ROMs folders, and the operation's duration raises the same open question (synchronous with a result page, or background with a status page) — to be settled the same way as 022 so both commands look alike. Same `crossSite` check and same `POST` + `303` shape as the other mutating routes.
