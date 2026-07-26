# Module: store
**Role:** Commits a change made to the registry — writes the registry itself and regenerates the static consultation site derived from it — so the two never drift apart, whichever caller made the change.
**Files:** `internal/store/store.go`
**Exports:**
- `store.Save(reg *registry.Registry, registryFolder string) error` — `registry.Save` followed by `site.Generate`. A registry write failure is returned as-is; when only the site regeneration failed, the returned error wraps `ErrSiteNotRegenerated`
- `store.ErrSiteNotRegenerated` — sentinel marking the half-failure worth telling apart: the registry (the source of truth) was written, but the consultation site is now stale. A caller that reported it as a plain failure would tell the user nothing was saved and have them redo a change that already applied

**Depends on:** [`modules/registry.md`](registry.md), [`modules/site.md`](site.md)

**Architecture note:** extracted for backlog item 015 — see [`decisions/020`](../decisions/020-one-place-committing-a-registry-change.md). The sequence previously lived only in [`modules/cli.md`](cli.md)'s `saveAndGenerateSite`, which now delegates to it, and [`modules/webui.md`](webui.md) needed the same one without depending on the CLI layer. `internal/registry` could not host it either: `internal/site` already imports `registry`, so the dependency would be a cycle. The CLI still treats both failures the same (an error printed, exit code 1); only the web UI distinguishes them, since it is the one that must not claim a correction was lost.
