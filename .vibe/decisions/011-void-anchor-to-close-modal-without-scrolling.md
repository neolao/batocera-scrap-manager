---
date: 2026-07-02
status: accepted
---
# Void anchor to close the detail modal without scrolling the page
**Context:** The CSS-only detail modal (see [`decisions/010`](010-css-only-detail-modal-via-target.md)) originally closed by linking back to the current system's section anchor (`#<system>`). Navigating to that anchor made the browser scroll the page to that section, which was jarring since the page had not actually moved while the modal was open.
**Decision:** The modal's close button and backdrop link to a fragment that matches no element in the page (`#_modal-close`) instead of a real anchor. Navigating to a non-existent fragment stops the CSS `:target` match (closing the modal) without triggering any scroll, since the browser has no matching element to scroll to.
**Reason:** Keeps the JavaScript-free `:target` technique while removing an unwanted scroll jump on close; no other CSS-only mechanism exists to unset `:target` without changing the URL fragment.
**Rejected alternatives:** Linking back to `#top` — same scroll-jump problem, just always to the top of the page instead of the system section.
