---
date: 2026-07-26
status: accepted
---
# One place committing a registry change

**Context:** Every command that modifies the registry must both persist it (`registry.Save`) and
regenerate the static consultation site derived from it (`site.Generate`) — a two-step invariant
held so far by `saveAndGenerateSite` in `internal/cli/common.go`. The web UI's save now needs the
exact same sequence, and it must not depend on the `cli` package. `internal/registry` cannot host it
either: `site` already imports `registry`, so the dependency would be a cycle.

**Decision:** A small `internal/store` package holds that sequence (`store.Save`), used by both
`internal/cli` and `internal/webui`. When the registry was written but the site regeneration failed,
it returns an error wrapping `store.ErrSiteNotRegenerated`, so a caller can tell a total failure
("nothing was saved") from a partial one ("saved, but the consultation site is stale") instead of
reporting a save that did happen as an error.

**Reason:** Duplicating the sequence in a second package is exactly the kind of drift between two
independently-maintained rules that [`014`](014-dedupe-by-extension-stripped-filename-too.md) turned
into a real data-loss bug. Distinguishing the partial failure matters because the registry is the
source of truth: a user told "save failed" would re-submit an edit that already applied.

**Rejected alternatives:**
- *`internal/webui` calls `registry.Save` + `site.Generate` itself* — two callers, one invariant, no
  single place to change it.
- *Exporting the helper from `internal/cli`* — `webui` would depend on the CLI layer, and `cli`
  already depends on `webui`.
- *Injecting a save callback into `webui.Handler`* — inverts the dependency correctly, but pushes the
  choice of what "saving" means into the CLI and makes the web UI's own tests assert through a stub.
