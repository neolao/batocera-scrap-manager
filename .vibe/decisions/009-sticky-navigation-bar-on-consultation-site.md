---
date: 2026-07-02
status: accepted
---
# Sticky navigation bar on the consultation site
**Context:** The consultation site (backlog item 007/008) needed a way to jump between systems and to return to the top of the page from any section (backlog item 010).
**Decision:** The navigation bar listing all systems stays fixed/sticky at the top of the viewport while scrolling, rather than being a one-time summary block at the top of the page.
**Reason:** A sticky bar is reachable from any scroll position, which also satisfies the "return to top/summary from any section" acceptance criterion without needing a separate per-section "back to top" affordance. Confirmed with the user over a simple, non-sticky summary alternative.
**Rejected alternatives:** A plain summary list of systems shown once at the top of the page (not sticky) — would require scrolling back up manually or a separate "back to top" link in every section.
