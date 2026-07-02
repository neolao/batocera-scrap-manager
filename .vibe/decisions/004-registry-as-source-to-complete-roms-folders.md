---
date: 2026-07-02
status: accepted
---
# Registry as source to complete ROMs folders' gamelist metadata

**Context:** Backlog item 003 originally described a `scrape` command that would complete missing registry metadata via an external scraping service, with the service itself left undefined. The Product Owner clarified that the actual need is the reverse: the registry (already populated from other ROMs folders) should act as the source of truth used to fill gaps in the gamelist.xml/media of ROMs folders that have incomplete local entries — no external scraping service is involved.

**Decision:** The `scrape` command reads the registry read-only and writes completions (missing gamelist fields and their referenced media files) directly into the ROMs folders' own `gamelist.xml` and system subfolders. It only completes games that already have a local gamelist entry but with missing fields; ROMs with no local entry at all are out of scope for this iteration. A `gamelist.WriteFile` capability was added to persist the merged entries back to disk (the package previously only supported parsing).

**Reason:** This matches the real use case of propagating already-scraped data across multiple ROMs folders/devices sharing the same registry, without depending on an undefined external service or unbounded ROM-file-discovery logic.

**Rejected alternatives:** Calling an external scraping API (e.g. ScreenScraper) to complete the registry itself — rejected because no service/credentials were defined and it solved the wrong direction of the problem. Also rejected: also handling ROMs with zero local gamelist entry in this same iteration — deferred to keep the scope focused and avoid ROM-file-vs-non-ROM-file detection heuristics.
