---
status: todo
depends_on: [014]
---
# Manage Media From The Web UI

## Description
A missing or low-quality cover art cannot be corrected today: the registry only receives the media copied from the ROMs folders during an `update`. From a game's page, the web UI must allow uploading a file to replace a medium (cover art, video, marquee, thumbnail) and deleting an existing medium, with the registry updated accordingly.

## Acceptance Criteria
- [ ] A game's page allows uploading a file for each of the four media (cover art, video, marquee, thumbnail); once uploaded, the medium shows on the page and the game's matching field points at a file really present in the registry folder
- [ ] The page allows deleting an existing medium: the file disappears from the registry folder and the game's matching field is emptied
- [ ] A file whose extension is not allowed for the targeted medium is rejected with an explicit error message, without modifying anything in the registry
- [ ] An upload exceeding the maximum allowed size is rejected with an explicit message, without modifying anything in the registry
- [ ] The static `index.html` site reflects the addition or the deletion of a medium once the operation is done

## Notes
The name of the file stored in the registry must be derived from the game's identifier (`gameID`) and the validated extension, never from the filename supplied by the client, so no path traversal nor overwrite of another game is possible. Cap the upload size (`http.MaxBytesReader`) and allow only a list of extensions per medium type. On the domain side, add to `internal/registry` the writing and the deletion of an entry's medium (reusing `removeIfExists` and the `mediaFields` table), then persist through the same `registry.Save` + `site.Generate` sequence as the other changes.
