---
status: done
depends_on: [001]
---
# CLI Command To Update The Registry

## Description
The project must expose a CLI command updating the registry from the configured Batocera ROMs folders. That command walks every ROMs folder, detects the changes in the `gamelist.xml` and media files, and synchronizes that information into the registry. It is the main entry point for keeping the registry up to date after a new scrape or after ROMs have been added.

## Acceptance Criteria
- [ ] The user can run a dedicated CLI command (e.g. `batocera-scrap-manager update`) to update the registry from the configured ROMs folders
- [ ] The system walks every configured ROMs folder and updates the registry with the newly detected gamelist and media entries
- [ ] The system displays a summary (number of entries added, updated, unchanged) when the run ends
- [ ] The system returns a non-zero exit code on failure (ROMs folder not found, registry not configured, etc.)

## Notes
Depends on item 001 (registry and ROMs folders configuration), which must be implemented first to provide the paths this command needs.
