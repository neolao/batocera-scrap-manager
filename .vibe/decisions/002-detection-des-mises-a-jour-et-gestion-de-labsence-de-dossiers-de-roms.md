# 002 — Update detection and handling missing ROMs folders (item 002)

date: 2026-07-01
status: accepted

**Context:** Implementation of the `update` command (backlog item 002). The item requires a summary distinguishing "added / updated / unchanged", whereas the existing import mechanism (001) only distinguished "added / unchanged" (an already-known entry was never compared against its up-to-date content).
**Decision:** `Registry.Import` is extended to compare the full game content (all metadata) against the already-known entry (system + ROM path key); on a difference, the entry is replaced and counted as "updated" rather than "unchanged". Additionally, the absence of a configured ROMs folder is not treated as an error by the `update` command — it simply displays a zero summary.
**Reason:** Faithfully satisfy the 3-category summary required by the acceptance criteria; the absence of a configured ROMs folder is a valid state (e.g. right after item 001, before any `config add-roms-folder`), not a blocking error — unlike a configured folder that is missing from disk, which remains an error.
**Rejected alternatives:** Keeping "unchanged" for already-known entries without content comparison — rejected because it does not satisfy the 3-category summary acceptance criterion; failing `update` when no ROMs folder is configured — rejected to stay consistent with the tolerant behavior of `config list` (which shows "(none)" instead of failing).
