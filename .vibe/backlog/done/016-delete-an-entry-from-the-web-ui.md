---
status: done
depends_on: [014]
---
# Delete An Entry From The Web UI

## Description
Deleting an entry from the registry is only possible on the command line (`remove <system> <rom-filename>`, item 005), which forces the user to know the exact ROM filename. From a game's page in the web UI, it must be possible to delete its registry entry — metadata and media — with an explicit confirmation before the action, then be sent back to the list.

## Acceptance Criteria
- [ ] A game's page offers a deletion asking for an explicit confirmation before acting
- [ ] After confirmation, the game disappears from the home page and from the static `index.html` site, and neither its JSON file nor its media remain in the registry folder
- [ ] A `GET` request on the deletion URL deletes nothing
- [ ] Deleting an already deleted (or unknown) game answers 404 without modifying the registry

## Notes
Reuse `registry.Remove`, which already deletes the JSON file and the four media through the `mediaFields` table, then regenerate the static site the way the `remove` command does. Redirect with a `303` to the home page after the deletion.
