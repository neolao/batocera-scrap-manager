---
date: 2026-07-29
status: accepted
---
# Replacing is its own operation, not a mode of completing

**Context:** Sending one game to a chosen ROMs folder (backlog item 026) was
specified as a web UI entry point onto `registry.CompleteGame`. The user then
asked for the choice between filling the folder's gaps and *replacing* what it
holds. Completion is defined — in the code, in the glossary, on both of its
confirmation pages — as the operation that fills gaps and **never** overwrites a
value already present locally. Overwriting is therefore not a setting of it; it
contradicts what the word means everywhere else in this codebase.

**Decision:** `registry.ReplaceGame` is a second exported operation alongside
`CompleteGame`, with the same signature and the same result shape. Both delegate
to one unexported core (`sendGame`) parameterized by the merge rule alone — the
parsing, the lookup, the media copy and the gamelist rewrite are shared word for
word. Two rules bound what replacing may destroy:

- a field the registry holds **empty** never blanks the folder's value — not
  knowing something is no reason to make the user lose it;
- a medium is written only when the destination is missing or holds different
  bytes, so a repeat send rewrites nothing and can truthfully report "already up
  to date".

`CompleteGame`'s behaviour, signature and callers are unchanged, and `scrape`
still only completes.

**Reason:** Two operations that mean opposite things about the user's own files
must be nameable apart at the call site, in the tests, and in the glossary — a
boolean parameter would make `CompleteGame(…, true)` mean the one thing its
documentation swears it never does. What they genuinely share is mechanism, not
meaning, so the sharing happens one level down, in a private helper neither name
leaks into. That is the same split the codebase already applies between
`ImportFromRomsFolder` and `CompleteRomsFolder`.

Comparing bytes before writing a medium costs one read of a file already being
copied, and buys the honest verdict: without it, every send after the first would
claim to have changed something, in the one flow that writes into files under no
version control.

**Rejected alternatives:**
- *A `replace bool` (or a mode enum) parameter on `CompleteGame`.* Makes the
  function's own name a lie in half its calls, and forces every existing caller
  and test to state a mode they have no opinion about.
- *Replacing by blanking the local entry and then completing it.* Simple to
  write, but a field the registry does not know would be erased — the folder's
  `gamelist.xml` is the user's only copy of what this tool does not model.
- *Overwriting media unconditionally.* Cheaper, but rewrites the user's files on
  every repeat and makes "already up to date" unsayable.
- *Deleting the media a replacement orphans* (the folder pointed at another
  filename). Erasing a user's file that nothing asked about; the orphan is inert
  and visible, the deletion would not be.
- *Adding the same choice to `scrape`.* A separate entry point with its own
  flag design; out of scope here, and nothing about this decision blocks it.
