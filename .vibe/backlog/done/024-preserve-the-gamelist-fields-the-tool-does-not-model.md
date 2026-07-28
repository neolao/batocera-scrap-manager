---
status: done
---
# Preserve The Gamelist Fields The Tool Does Not Model

## Description
Rewriting a system's `gamelist.xml` re-encodes the document from the parsed `gamelist.Game` struct alone, so every element that struct does not declare is silently dropped: `<favorite>`, `<playcount>`, `<lastplayed>`, and whatever else Batocera or EmulationStation writes there. The user loses data this tool never had any business touching — their favourites and their play history — in exchange for metadata they asked to have completed. Observed for real while building item 023: a gamelist holding `<favorite>true</favorite>` and `<playcount>17</playcount>` came back without either after a completion.

This is pre-existing behaviour of the `scrape` command, not a regression, but item 023 put a button in the web UI that runs the same completion — so it is now one click away instead of a deliberate command-line invocation. `update` reads those files without ever writing them, so only the completion flow is concerned.

## Acceptance Criteria
- [ ] A `gamelist.xml` holding `<favorite>`, `<playcount>` and `<lastplayed>` still holds them, with their original values, after a completion has filled other fields of the same game
- [ ] An element the tool does not model that sits on a game the completion did not touch is preserved as well
- [ ] Completing a game whose local entry carries an unknown element still fills the gaps the registry can fill, and the resulting document is valid XML that `ParseFile` reads back without error
- [ ] The registry keeps ignoring those elements: they are not imported by `update`, not stored in a game's JSON file, and not shown in the web UI

## Notes
The stake is the user's only copy of a file under no version control, which is also why writing it was made all-or-nothing ([`decisions/026`](../decisions/026-a-gamelist-is-written-beside-then-swapped-in.md)) — that guarantees the file is never truncated, but says nothing about what the rewritten document contains.

`encoding/xml` has no round-trip mode, so the shape of the fix is the real question and should be settled before implementing. Adding the three known fields to `gamelist.Game` is the cheapest option but only moves the boundary: the next unknown tag is dropped just the same, and `<favorite>`/`<playcount>` would then be tempted into the registry, which has no use for them. Capturing the unknown children of each `<game>` (`,any` into a `[]xml.Token` or a raw `,innerxml` remainder) and writing them back preserves everything, at the cost of a field on `Game` that every other package must ignore — and of deciding where the preserved elements land in the re-encoded element order, since Batocera's own reader may or may not care.

Whichever shape wins, it belongs in `internal/gamelist`: `internal/registry` should stay unaware that such fields exist. Worth checking what Batocera actually tolerates in element ordering before assuming the preserved tags can simply be appended at the end.
