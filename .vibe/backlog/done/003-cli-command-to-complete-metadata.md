---
status: done
depends_on: [001, 002]
---
# CLI Command To Complete Metadata

## Description
The project must expose a CLI command completing the missing metadata of the ROMs referenced in the registry (name, description, year, genre, rating, cover art, etc.). That command identifies the ROMs for which the registry holds no complete gamelist entry or media yet, then fetches the missing information through an external scrape. It fills the gaps left by the synchronization of the already scraped data, for the games never scraped before.

## Acceptance Criteria
- [ ] The user can run a dedicated CLI command (e.g. `batocera-scrap-manager scrape`) to complete the missing metadata of the registry's ROMs
- [ ] The system automatically identifies, from the registry, the ROMs whose metadata (gamelist) or media are missing or incomplete
- [ ] The system fetches the missing metadata from a scraping source and updates the registry accordingly
- [ ] The system displays a summary (number of ROMs processed, completed successfully, failed) when the run ends

## Notes
Depends on item 001 (registry and ROMs configuration) and item 002 (updating the registry from the ROMs folders), which must be implemented first so the registry reflects the current state of the ROMs before looking for missing metadata. The external scraping source to use (API, service, exchange format) is left to be defined.
