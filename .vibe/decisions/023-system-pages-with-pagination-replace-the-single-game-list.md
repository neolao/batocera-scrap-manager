---
date: 2026-07-27
status: accepted
---
# System pages with pagination replace the single game list

**Context:** The served web UI's home page (backlog item 014) rendered every game of every system into one document, with an in-page `#<system>` anchor as its only navigation. On a phone that page is unusable — below 480px each game became a full-width card with a 4:3 cover, so roughly one game fitted per screen — and it is also the most expensive page of the whole UI: it goes through `site.GroupBySystem`, which `os.Stat`s each of a game's four media references, so a registry of two thousand games costs some eight thousand stat calls and a megabyte of HTML on every single load. Backlog item 020 asks for a browsable-with-a-thumb UI.

**Decision:** `/` becomes a summary listing each system and its game count, and every system gets its own route `/system/{system}`, paginated at 60 games a page through `?page=N`. The summary counts entries directly instead of grouping them, and a system page groups only the sixty entries it is about to render. An unknown system, and a page number outside `[1, pageCount]`, both answer the themed 404. The static site is untouched structurally: it stays the single self-contained `index.html` of [`decisions/008`](008-move-consultation-site-to-registry-root.md), and only inherits the shared theme's new compact rows.

**Reason:** Splitting by system is what makes both problems go away at once — the home page stops touching the disk at all, a system page stats sixty games instead of two thousand, and each page is short enough to scroll with a thumb. A page out of bounds is treated as a wrong URL rather than an empty list because the alternative hides broken links: a `200` on `?page=99` would read as a system that lost its games. The first page is addressed without a parameter, so a system keeps one canonical URL, and `?page=1` is still accepted rather than redirected — a redirect would buy nothing and cost a round trip on a phone.

**Rejected alternatives:**
- Keeping the single all-games page and only compacting the cards: 60 rows per screen still means scrolling past several thousand of them to reach the last system, and the per-load stat cost — the reason the page is slow to open as the registry grows — would be untouched.
- Alphabetical paging (`?letter=S`) instead of numbered pages: more meaningful for finding a known game, but produces wildly uneven pages (a letter can hold three games or three hundred), which defeats the point of bounding page size.
- Rendering all of a system's games and letting the browser lazy-load: `loading="lazy"` already defers the images, but the HTML and the media-presence lookups are paid up front regardless — that is the actual cost.
- A client-side search or filter: the served UI ships no JavaScript, and adding it here for navigation alone would break that property for every other page.
