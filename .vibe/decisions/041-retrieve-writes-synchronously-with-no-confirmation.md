---
date: 2026-09-19
status: accepted
---
# Retrieve writes synchronously with no confirmation, gated to Listed gaps

**Context:** The ROMs folder status page needed a way to pull one gap game's already-scraped local data into the registry, directly from its row in the gap list.

**Decision:** Retrieve is a single POST that imports the one game, writes the registry and redirects back to the status page with a banner — no separate confirmation page — and is offered only on a gap whose local `gamelist.xml` already has an entry (`Listed`); a gap with none has nothing to pull.

**Reason:** Retrieve only ever writes the registry, which the tool can always rebuild from the ROMs folders — the same reasoning that already lets a correction or a protection change apply on one submit, unlike Completion or Replacement, which write the user's own Batocera files and always confirm first. A `Listed=false` gap's local `gamelist.xml` carries no entry at all, so there is nothing on disk for an import to read; offering the control there would only ever fail.

**Rejected alternatives:** A confirmation page mirroring Send's — rejected as unearned caution for a write this codebase already treats as low-stakes everywhere else it happens in one request. Offering Retrieve on every gap regardless of `Listed` — rejected because it would routinely present a control that cannot do anything.
