# 001 — Registry storage format and import scope

date: 2026-07-01
status: accepted

**Context:** Implementation of backlog item 001 (registry and ROMs configuration) — the exact registry structure was explicitly left open by the item.
**Decision:** The registry is a single JSON file (`registry.json`) at the configured path, indexing game metadata (deduplicated by system + ROM path key) without copying media files. The import function (`registry.ImportFromRomsFolder`) stays internal and is not exposed via a CLI command in this item — it will be wired up by item 002 (the `update` command).
**Reason:** Avoid the complexity of copying/synchronizing media files this early; clearly separate the configuration/import building block (001) from the dedicated CLI command with summary and exit code (002), so this item ships something testable via the CLI without anticipating item 002's command.
**Rejected alternatives:** Storing the registry in a database (SQLite) — deemed premature for a simple CLI project; copying media files into the registry — deferred to a future iteration if the need is confirmed.
