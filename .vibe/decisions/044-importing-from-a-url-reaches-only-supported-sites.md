---
date: 2026-11-18
status: accepted
---
# Importing from a URL reaches only supported sites, and the user picks each value

**Context:** A game's values can now be filled or completed from a web page whose address the user pastes. This is the first time the server itself makes requests to the network, and `serve` trusts anyone on the local network (decisions/032).

**Decision:** A new `internal/scrape` package is the only code that makes network requests. Before any request, it accepts only `https` addresses, on the default port, without credentials, on a host one of its parsers reads; today that is `retrogaming.neolao.com`. A redirect must stay on that same host. Pages are capped at 2 MB and images at 20 MB, an image must look like one by its content, and a request times out after 20 s. The web UI first shows a review where every value found has a tick box: ticked when the game has nothing there, unticked when the box would overwrite a value, absent when the value is identical. Applying fetches the page and the ticked images again, outside the registry lock, then writes only the ticked values through `UpdateMetadata` (so they become hand-edited) and `WriteMedium`. HTML is parsed with `golang.org/x/net/html`.

**Reason:** A host allowlist checked before any network access means no pasted address can turn the server into a way to reach a machine on the local network. Re-reading the page when applying means a submission can never inject values the site does not hold. Per-value choice was the user's explicit request, so a correction made by hand is never lost silently. Each site is a parser added to a list, so supporting a second site means adding a parser, not changing the flow.

**Rejected alternatives:**
- Fetching any address: SSRF from a server that has no authentication.
- Passing the reviewed values through hidden fields: it trusts the browser with data the server can re-read itself.
- Parsing the HTML with regular expressions: brittle against a generated page, when the Go project maintains a real parser.
- Keeping every screenshot: the registry has a single thumbnail slot, so only the first screenshot is kept.
