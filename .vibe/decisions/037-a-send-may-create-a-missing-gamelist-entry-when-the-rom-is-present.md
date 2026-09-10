---
date: 2026-09-10
status: accepted
---
# A send may create a missing gamelist entry when the ROM is present

**Context:** Sending one game (`registry.CompleteGame`/`registry.ReplaceGame`,
over the shared `sendGame` core) refused outright whenever the target
folder's `gamelist.xml` had no entry for the ROM, even when the registry
held a matching entry and the ROM file was actually sitting in that folder
— a game added to a second ROMs folder but never yet scraped there. The
glossary already worded the old rule narrowly ("it never adds a game to a
folder that does not hold its ROM"), leaving the door open for exactly this
case: a folder that *does* hold the ROM.

**Decision:** When the local `gamelist.xml` has no entry for the ROM (by
base filename) but a matching registry entry exists, `sendGame` now checks
whether the ROM file is actually present at the exact relative path used
for both the registry lookup and the new entry's `Path` — via the existing
`mediumPath` containment helper, never a path rebuilt from the registry's
own idea of the file's location. If present, a blank `gamelist.Game{Path:
romFilename}` is appended in memory and goes through the same merge/
overwrite, media-copy and atomic-rewrite path an existing entry already
does, in one `gamelist.UpdateFile` call — never an intermediate write of a
bare, unmerged entry. A `gamelist.xml` entirely absent for that system is
now treated as an empty list rather than an immediate error, so the same
logic creates the file; a `gamelist.xml` that exists but fails to parse is
still a hard `ErrGameNotFound`. `CompleteGame` and `ReplaceGame` both gain
an `added bool` return alongside their existing result shape, so the CLI
and the web UI can word a creation distinctly from a fill or a replacement
rather than reusing either sentence.

`CompleteRomsFolder` (the whole-folder `scrape` and the web UI's
`/complete`) is untouched: it walks a folder's *already-parsed* local
entries and only ever fills gaps of games already listed there, never
consulting `sendGame` — so this change cannot make a background pass
manufacture entries in bulk. It only ever applies to a single targeted
send.

**Reason:** The old refusal conflated two different situations under one
error — a ROM genuinely missing from the folder, and a ROM present but
never listed — the second of which is exactly what a user copying a ROM
into a second Batocera folder runs into, with no mechanism in the tool to
resolve it short of relaunching Batocera's own scraper. Checking the exact
same path used to look the game up (rather than trusting the registry's
memory of where the file lives) is what keeps the new entry from ever
pointing at a file that was not actually confirmed present — the safety
property the old message was already promising.

**Rejected alternatives:**
- *Pre-reading the target gamelist.xml on the confirmation page to warn "this
  will create a new entry" before the send.* Confirmed with the product
  owner as unnecessary — the after-the-fact wording on the outcome banner is
  enough, and the confirmation page stays as it is.
- *Toning down "Replace"'s red warning for the creation case, since nothing
  is actually overwritten.* Confirmed with the product owner as unnecessary
  complexity — the mode keeps one look regardless of whether it fills,
  replaces, or creates.
- *Extending `CompleteRomsFolder` the same way, so a whole-folder pass could
  also pick up ROMs Batocera never scraped.* Out of scope: it changes the
  blast radius of an unattended background operation from "fixes what's
  already listed" to "can list new games on its own", which is a
  meaningfully different risk profile than fixing one targeted send.
- *Matching the ROM file loosely (by base name anywhere under the system
  folder) rather than at the exact relative path.* Rejected per the data
  consultation: the path that gets stat'd must be the same one written into
  the new entry, or the containment check can pass on one string while the
  file on disk is a different one.
