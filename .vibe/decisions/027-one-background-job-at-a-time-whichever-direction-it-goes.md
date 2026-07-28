---
date: 2026-07-28
status: accepted
---
# One background job at a time, whichever direction it goes

**Context:** Importing the ROMs folders from the web UI (backlog item 022) is the
second long operation the server detaches from its request, after completing them
(decisions/025). The two flow in opposite directions over the same files: the
completion reads the registry snapshot and rewrites every `gamelist.xml`, the
import reads those same `gamelist.xml` files and rewrites the registry. Each
already refuses a second run of its own kind, through a slot of its own.

**Decision:** The two share one slot. A run of either kind takes it, and the other
kind's page then renders neither its start button nor a silent refusal, but a
sentence naming the operation in flight and a link to the page following it.

**Reason:** Left independent, an import could read a `gamelist.xml` the completion
is halfway through replacing, and a completion could write out a snapshot the
import has already superseded. Neither corrupts anything — a gamelist is swapped
in atomically (decisions/026) and a registry change applies to a `Clone()` — but
the outcome depends on which run wins, and no page could tell the user which one
they got. One slot makes the answer knowable without any locking discipline
spanning the two flows. It also costs the user nothing real: the two operations
are the two halves of one round trip, and running them at once is never what
someone means.

**Rejected alternatives:**
- *Independent slots, documented as safe.* The interleaving is harmless in the
  data but unexplainable in the UI, and "harmless" rests on an argument about
  atomic renames that no reader of the page can check.
- *Queueing the second run behind the first.* A run lasts minutes; a button whose
  effect starts at an unstated time later is worse than one that says it cannot
  run now.
- *Refusing the submission with a 409.* The refusal has to be worded on the page
  the user is already looking at, not on an error page they have to leave.
