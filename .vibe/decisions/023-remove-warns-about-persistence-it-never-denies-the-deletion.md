---
date: 2026-07-27
status: accepted
---
# `remove` warns about persistence, it never denies the deletion

**Context:** Backlog item 019 makes the CLI's `remove` regenerate the consultation
site, which every other registry-modifying command already does through
`saveAndGenerateSite` — the thin `internal/cli` wrapper over `store.Save` that prints
`error: %v` and reports failure to its caller, which then returns exit code 1.

That helper cannot be reused here. `registry.Remove` erases the game's JSON file
*before* `store.Save` is ever reached, so by the time persistence can fail the
deletion has already committed (see decisions/022). A `store.Save` failure reported
as `error:` with exit code 1 would tell the user the removal failed while the game is
already gone, and have them retry a deletion that would then answer
`no game found` — the exact confusion `ErrMediaLeftBehind` is already handled to
avoid in this same command.

**Decision:** After a successful `registry.Remove`, `runRemove` calls `store.Save`
directly rather than through `saveAndGenerateSite`. Every failure it can return is a
`warning:` line appended after the `removed ...` confirmation, and the exit code
stays 0. Non-zero remains reserved for the cases where nothing was deleted at all
(bad arguments, unreadable config or registry, `ErrGameNotFound`, a JSON file that
could not be erased).

The two failures are worded apart rather than folded into one `warning: %v`:

- wrapping `store.ErrSiteNotRegenerated` → the registry holds, only `index.html` is
  stale; the warning names `update` as the way to rebuild it.
- any other error → `registry.Save` itself failed, so the registry folder may hold a
  partial write concerning *other* games; the warning says so and does not send the
  user to `update` as if it were a display problem.

**Reason:** The recovery actions differ, so one message cannot serve both. Telling a
user "the site is stale, run update" when the registry write failed sends them to
rebuild a page from a registry whose state is the actual problem. The deleted game
itself is safe either way — its file is gone and `registry.Save` only ever writes the
entries it is given, never resurrects one that left.

**Rejected alternatives:**
- *Reuse `saveAndGenerateSite` as-is* — its "print an error, fail the command"
  semantics assume the whole operation failed, which is never true here.
- *Widen `saveAndGenerateSite` with a mode telling it whether a failure is fatal* —
  a boolean parameter deciding what a helper means at each call site, for two callers
  that share three lines. The divergence is the point; a comment marks it so it is
  not "aligned" back later.
- *A distinct exit code for the stale-site case* — no caller can act on it that could
  not act on the warning line, and it would break scripts treating 0 as "the game is
  gone", which it is.
