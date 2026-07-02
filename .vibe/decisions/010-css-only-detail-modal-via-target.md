---
date: 2026-07-02
status: accepted
---
# CSS-only detail modal via :target, no JavaScript
**Context:** Game cards on the consultation site needed a way to reveal a game's full description without stretching every card to fit its longest text (the card description is now clamped to a few lines).
**Decision:** Each card is a link to an anchor (`#modal-<system>-<index>`); a matching hidden overlay element becomes visible via the CSS `:target` pseudo-class when that anchor is active, and a close link inside it points back to the system's anchor to hide it again. No JavaScript is added to the generated page.
**Reason:** The consultation site is a single, dependency-free static file (see [`decisions/008`](008-move-consultation-site-to-registry-root.md)); a CSS-only technique keeps that property instead of introducing inline JavaScript for what is otherwise a simple show/hide interaction.
**Rejected alternatives:** A native `<dialog>` element opened via a small inline `<script>` calling `showModal()` — more idiomatic and easier to make fully accessible (focus trapping, native Escape handling), but would introduce JavaScript into a page that has been JavaScript-free so far.
