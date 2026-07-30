---
status: todo
---
# Avoid Blocking Stat Calls During Site Regeneration

## Description
Every write to the registry through the web UI (edit, protect, a media change, a deletion) triggers a full `site.Generate()` over the *entire* registry, and `site.GroupBySystem` performs up to 4 `os.Stat` calls per game to resolve which media are really on disk — all of it synchronous, and all of it while `webUI.mu`'s write lock is held. Flagged by a `/vibe:review` pass (2026-07-30): a single-game correction on a large registry does thousands of blocking stat calls for games the correction never touched, while every concurrent reader (`serveHome`, `serveSystem`, `serveGame`) queues behind it.

## Acceptance Criteria
- [ ] A single-game correction no longer performs a stat check for every other, unrelated game's media in the same request
- [ ] A concurrent `GET` request is not blocked for the duration of a full site regeneration triggered by an unrelated write
- [ ] The regenerated site remains byte-for-byte correct and complete for every game (no media reference silently dropped or missed)

## Notes
`serveHome` already avoids the equivalent full-registry walk for its own page (see the existing comment in `internal/webui/webui.go` about "thousands of `os.Stat` calls for a page that shows no media at all") — this item is about every *write* path still triggering it via `store.Save`. Two directions worth weighing: regenerate only the affected system's section, or move regeneration off the request path (background/debounced) — either changes when the served site reflects a write, which is worth discussing before committing to one.
