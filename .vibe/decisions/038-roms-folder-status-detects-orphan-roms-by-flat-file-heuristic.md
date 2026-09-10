---
date: 2026-09-10
status: accepted
---
# ROMs folder status detects orphan ROMs by a flat-file heuristic

**Context:** The web UI gained a read-only status page for one ROMs folder, reporting what is still missing compared to what a Completion could already fill from the registry — including ROM files that have no `gamelist.xml` entry at all yet.

**Decision:** A ROM file with no local entry is detected by listing the files sitting directly inside a system's folder (excluding `gamelist.xml` and any subfolder such as `images/`/`videos/`) and treating every one of them as a candidate ROM, with no check on file extension.

**Reason:** The tool holds no per-system list of valid ROM extensions today, and Batocera's own set of extensions is large and system-specific. A flat-file scan matches the folder layout every other flow already assumes (media and `gamelist.xml` live in known places, ROMs sit flat in the system folder) and needs no new configuration.

**Rejected alternatives:** An extension allowlist per system — more precise, but a second config surface to maintain and keep in sync with Batocera's own list, for a page whose purpose is a quick read, not an authoritative inventory.
