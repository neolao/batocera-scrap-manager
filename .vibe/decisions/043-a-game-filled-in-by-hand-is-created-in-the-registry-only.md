---
date: 2026-11-18
status: accepted
---
# A game filled in by hand is created in the registry only

**Context:** Some games of a ROMs folder have no scrape anywhere, neither in the folder's `gamelist.xml` nor in the registry. The status page had no way to give them metadata.

**Decision:** A "Fill in by hand" form, offered only for a game the registry does not know, creates the entry in the registry. It records each typed value as a hand-edited field, through the same `UpdateMetadata` path as a correction, and then sends the user back to the system's status page. The ROMs folder is written only by the existing send confirmation. Media are added afterwards through the game page's existing upload.

**Reason:** The registry stays the source of truth (decisions/004), and send stays the only confirmed way this tool writes into a ROMs folder (decisions/040). Marking the values as hand-edited keeps a later `update` from wiping them with the folder's still-empty entry.

**Rejected alternatives:** Writing straight into the folder's `gamelist.xml` skips the write confirmation. Online scraping needs network access and an account the tool does not have. An upload field in the form would duplicate the existing media upload and its validation.
