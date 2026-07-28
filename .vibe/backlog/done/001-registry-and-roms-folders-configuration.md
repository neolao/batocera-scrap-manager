---
status: done
---
# Registry And ROMs Folders Configuration

## Description
The project must allow configuring a "registry" folder centralizing the scraping data (gamelist and media) already collected, along with one or more Batocera ROMs folders to watch. The goal is to populate that registry from the `gamelist.xml` files and media files already scraped inside the ROMs folders, without having to run a full scrape again. This feature is the foundation of every future scraping management operation.

## Acceptance Criteria
- [ ] The user can configure the path of a "registry" folder where the scraped data will be centralized
- [ ] The user can configure one or more Batocera ROMs folders to associate with the registry
- [ ] The system detects and reads the existing `gamelist.xml` files in every configured ROMs folder
- [ ] The system imports into the registry the gamelist entries and media files already present, without duplicating the entries already imported

## Notes
The `gamelist.xml` format and the media folder layout (images, videos, manuals) follow the EmulationStation/Batocera convention. The registry's exact structure (storage format: flat files, database, etc.) is left to be defined at implementation time.
