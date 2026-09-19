---
date: 2026-09-19
status: accepted
---
# Send from the ROMs folder status page links to the existing confirmation

**Context:** The ROMs folder status page lists per-game gaps and, for a game the registry already knows, needed a more direct way to send that game to this exact folder without first opening the game's own page.

**Decision:** A gap row known to the registry gets a plain link to the existing per-game send confirmation page, with this folder and the "fill gaps only" rule already set in the query — no new send endpoint, control or confirmation template.

**Reason:** Send already has a complete, tested confirm-then-write flow (see the Send entry of the glossary and decisions/037); a link built from data the status page already holds reaches it directly, so the two-step, ROM-files-are-precious confirmation stays the only way this tool ever writes into a ROMs folder.

**Rejected alternatives:** A second, inline send control on the status page itself — rejected as a near-duplicate of the existing one, doubling the surface that must stay consistent (folder validation, the fill/replace wording, the confirmation copy) for no behavioural gain.
