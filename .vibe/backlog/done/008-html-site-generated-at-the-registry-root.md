---
status: done
---
# HTML Site Generated At The Registry Root

## Description
The HTML site browsing the registry is currently generated inside a `site/` subfolder (`<registry>/site/index.html`). That `index.html` file must instead be generated directly at the root of the registry folder (`<registry>/index.html`), so it is easier to find and open.

## Acceptance Criteria
- [ ] After an `update`, the generated `index.html` file sits directly at the root of the registry folder, no longer inside a `site/` subfolder
- [ ] The cover art displayed in the page stays correctly reachable from that new location (relative media paths adjusted accordingly)
- [ ] An empty registry still generates a valid `index.html` at the root, with the message stating there are no games yet

## Notes
Changes the location chosen when the site was first implemented (item 007, see decision `.vibe/decisions/006`), which put the file inside a `site/` subfolder. To settle: what to do with the old `site/` subfolder if it still exists from a previous `update` (delete it, or leave it as is)?
