---
status: todo
---
# Send One Game To A Chosen ROMs Folder

## Description
Completing the ROMs folders from the registry is today an all-or-nothing operation: it walks every configured ROMs folder and every system in it. When a single game has just been corrected, there is no way to push that one game to the one ROMs folder that needs it — the whole run has to be launched, or the ROM file's path has to be typed on the command line. From a game's page, the user must be able to pick one of the configured ROMs folders and send that game's data to it alone.

## Acceptance Criteria
- [ ] A game's page offers to send that game to a ROMs folder, listing every configured ROMs folder as a choice; with no ROMs folder configured, it says so instead of offering an empty choice
- [ ] Sending the game writes only that game's missing fields and media into the chosen ROMs folder, leaving every other game of that folder and every other configured ROMs folder untouched
- [ ] The result is reported on the page, naming the folder and stating whether the game was completed, was already up to date, or could not be found in that folder's gamelist
- [ ] A value already present in the target gamelist is never overwritten, exactly as a full completion run behaves

## Notes
The domain operation already exists: `registry.CompleteGame` (`internal/registry/registry.go`) completes one game in one ROMs folder for one system, and the CLI already exposes it through `scrape <path>`, which derives the folder and system from a real disk path (see `decisions/013`). What is missing is the web UI entry point, where the folder is *chosen* rather than derived, so the ROM file's location need not be known.

Open question to settle when implementing: `CompleteGame` returns `ErrGameNotFound` when the game is absent from the target folder's `gamelist.xml` — completion fills gaps, it never adds a game to a folder that does not hold the ROM. The page must state that outcome plainly rather than claiming a success; whether sending should instead *add* the game is deliberately out of scope here.

Unlike the whole-folder import and completion, this touches one game and runs in milliseconds, so it needs neither the background job slot nor the shared job page (see `decisions/025` and `decisions/027`) — a plain form post on the game's own page is enough. It writes outside the registry into files under no version control, so it confirms before writing, like the other completion entry points.
