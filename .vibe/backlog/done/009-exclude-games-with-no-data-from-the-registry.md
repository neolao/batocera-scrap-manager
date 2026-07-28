---
status: done
---
# Exclude Games With No Data From The Registry

## Description
The `update` command currently imports into the registry every game found in the `gamelist.xml` files, including those with neither a description nor an image (hence no usable scraping data at all). Those games must no longer be added to the registry: only games holding at least a description or an image may enter it.

## Acceptance Criteria
- [ ] During an `update`, a game with no description and no image is not added to the registry
- [ ] A game holding at least a description or an image keeps being imported normally
- [ ] The displayed summary (added/updated/unchanged) does not count those skipped games among the added ones

## Notes
To settle: what to do with a game already present in the registry if a later update of its local `gamelist.xml` no longer holds a description nor an image (remove it from the registry, or keep the existing entry untouched)?
