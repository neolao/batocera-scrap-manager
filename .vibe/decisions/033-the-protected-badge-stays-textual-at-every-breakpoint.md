---
date: 2026-07-30
status: accepted
---
# The protected badge stays textual at every breakpoint
**Context:** Marking a fully-protected game on the per-system game list (backlog item 031). The list renders as a card grid on desktop and collapses to a dense 3.5rem row on phones, where the cover thumbnail shrinks and the description is hidden entirely to fit the name on one line.
**Decision:** The "Protected" badge keeps the same textual pill (reusing the `meta__manual` visual language) at every breakpoint, sitting next to the game name. On the dense mobile row, the name truncates with an ellipsis to make room for the badge rather than the badge shrinking into a color-only glyph.
**Reason:** A single markup/style pair is simpler to keep in sync across the two responsive layouts than maintaining a separate compact glyph variant for mobile, and a real text label trivially satisfies "not conveyed by color alone" without leaning on a `title`/`aria-label` that touch devices cannot reveal on hover.
**Rejected alternatives:** A compact color/glyph-only indicator for the dense mobile row (as one consulted expert first suggested) — rejected because it reintroduces a second visual vocabulary for the same concept and risks failing the accessible-label requirement on touch devices where `title` never surfaces.
