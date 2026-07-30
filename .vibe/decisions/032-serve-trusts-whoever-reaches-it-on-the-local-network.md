---
date: 2026-07-30
status: accepted
---
# `serve` trusts whoever reaches it on the local network

**Context:** A code review (`/vibe:review`) and a local penetration test both flagged that every state-changing route of `internal/webui` — deleting a game, editing its metadata, replacing a medium, sending a game to a ROMs folder, running import or completion — accepts any request with no account, session, or token, and that `serve` listens on `0.0.0.0:8080` by default ([`internal/cli/serve.go`](../../internal/cli/serve.go)), so any device on the same network reaches it. The pentest proved this end to end with bare `curl` calls, no headers required.

**Decision:** This is the intended shape of `serve`, not a gap. The command exists precisely so a game's metadata and media can be corrected from a phone or laptop while the tool runs on the Batocera machine itself — the wildcard default address is what makes that reachable, and is documented as such in `serve`'s own usage text and in `defaultServeAddr`'s comment. No authentication layer is added. The one control kept is the one already in place: a same-origin check (`crossSite`, `internal/webui/save.go`) that refuses a submission whose `Sec-Fetch-Site`/`Origin` names another site, closing the *cross-site* case (a browser tab on an unrelated page submitting to `serve` on the user's behalf) without touching the *same-network* case, which is the one this decision accepts.

**Reason:** A shared secret, a login form, or a per-run token would all require the user to carry or type something before reaching a page whose entire purpose is convenience — correcting one game's box art without touching a keyboard. Batocera's own exposed services (Samba, SSH in its default images) hold the same trust boundary: the local network is the perimeter, not the process. Moving that boundary into `serve` would protect against a threat model (another device on the user's own network) this tool was never scoped to resist, at the cost of the workflow it exists for.

**Consequence:** Whoever is on the same network as a running `serve` can read, edit, and delete anything in the registry and overwrite a configured ROMs folder's `gamelist.xml`, with no confirmation beyond the same unauthenticated request the web UI itself would send. This is acceptable on a home network; it is not on a network shared with untrusted devices. `serve`'s usage text and README should tell the user exactly that — the tool does not gatekeep who is inside their LAN.

**Rejected alternatives:**
- *A per-run token embedded in the served pages, required on every state-changing request.* Closes the network-reachability case in addition to the cross-site one already handled, but adds a secret the user must copy from the terminal into a browser on another device before the page becomes usable — the exact friction this command's UX is built to avoid.
- *Bind to `127.0.0.1` by default, opt into `0.0.0.0` with a flag.* Defeats the primary use case (correcting games from a phone) for the common case, to protect the uncommon one (a shared or hostile LAN); the user carries the risk either way once they opt back in, so the default only delays the same decision.
