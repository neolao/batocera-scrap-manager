---
status: todo
---
# Fill Unscraped Games From ROMs Folder

## Description
Some games have no scrape at all: the ROMs folder has no entry for them, and the registry does not know them either. Today the ROMs folder status page can only say they "need real scraping". The user must be able to fill in such a game's metadata and media straight from that page.

## Acceptance Criteria
- [ ] On a ROMs folder's system page, a game with no scrape locally or in the registry offers a "Fill in" action.
- [ ] Filling it in adds the game to the registry under the right system, using that ROM's filename, with the metadata and media the user entered.
- [ ] After saving, the system page shows the game as known to the registry, and "Send to this folder" is offered for it.
- [ ] Invalid input, such as an unknown folder, an unknown system or a ROM missing from disk, is refused and nothing is written.

## Notes
Depends on the per-system ROMs folder pages being built (work in progress). Open question: should the game be created in the registry and then sent, or written directly into the ROMs folder's gamelist.xml? The first option keeps the registry as the source of truth (see decisions/004).
