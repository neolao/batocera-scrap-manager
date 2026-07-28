---
date: 2026-07-28
status: accepted
---
# Completing ROMs folders runs in the background, behind one page

**Context:** Backlog item 023 lets the web UI trigger `registry.CompleteRomsFolder` over every configured ROMs folder — rewriting `gamelist.xml` files and copying media back into the user's Batocera folders. Unlike every mutating route so far, this one is a batch that can run for minutes on a large library, and the web UI ships no JavaScript, so there is no client-side progress to lean on. The item left the execution model as an open question, to be settled here and reused by item 022 (the import in the opposite direction) so both commands look alike.

**Decision:** The submission starts the completion in a **detached goroutine** and redirects (`303`) to `/complete`, which is a single URL rendering four states: no ROMs folder configured, no run in flight (the confirmation), a run in progress, and the report of the last run. Only the running state emits `<meta http-equiv="refresh">`; the report never does. One run at a time: the slot is reserved before the `303`, and a submission arriving while a run is in flight starts nothing and redirects to the same page. The run captures the served registry snapshot under the read lock and releases it immediately, then works against that pointer for its whole duration. The report of the last run stays readable until the next one replaces it.

**Reason:** Running synchronously would hold `webUI.mu.RLock` for the whole batch; Go's `sync.RWMutex` blocks new readers as soon as a writer waits, so one metadata correction submitted during a long run would freeze every served page. It would also show nothing at all — a blank tab for minutes, which invites a reload, which is a second write over the same unversioned `gamelist.xml`. And the outcome would die with the connection: this operation writes outside the registry with no way back, so the report is the only evidence of what was changed on the user's disk, and it has to outlive the browser. One URL rather than a confirmation URL plus a status URL keeps "run it again" as the very control that started it, and matches how the existing delete flow reads (confirm on a page of its own, act, land on a page that states the outcome).

**Rejected alternatives:**
- *Synchronous execution rendering the report as the POST response* — freezes the UI through the shared lock, shows no progress, and loses the report to any dropped connection (a reverse proxy's read timeout, a phone suspending Wi-Fi).
- *A separate status URL under `/complete/status`* — two pages that must agree on four states, and a report page with no way to run again without navigating back.
- *A job history keyed by an identifier in the URL* — a home tool with one user has no use for more than the last report; storing several would be state to expire with nothing asking for it.
- *Continuing to the next ROMs folder after one turns out to be unreachable* (the operations expert's recommendation) — the acceptance criterion asks that the report reflect the folders processed **before** the failure, and the CLI's `scrape` stops the same way. Two entry points onto one domain operation must not differ in what they mean.
- *Tracing the run on the `serve` command's output* — would thread an `io.Writer` through `webui.Handler` for a record the page already holds; a run report lost to a server restart is acceptable here.
