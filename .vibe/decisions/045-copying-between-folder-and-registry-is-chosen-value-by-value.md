---
date: 2026-11-18
status: accepted
---
# Copying between a ROMs folder and the registry is chosen value by value, either way

**Context:** On a ROMs folder's system page, "Retrieve into the registry" did not say what it did. It also replaced the whole registry entry with the folder's version, erasing values the folder lacked. The send link reached a different, all-or-nothing confirmation page. The user asked to copy in both directions and to pick what is completed or overwritten, the way the import from a web page works (decisions/044).

**Decision:** Each game row offers "Folder → Registry" (shown when the folder lists the game) and "Registry → Folder" (shown when the registry knows it). Both open the same review page. It shows the twelve values the source holds, each next to the destination's value: ticked when the destination is empty, unticked when it would overwrite, "Same" when identical (never for a medium, since one reference may name two different files). Applying copies only the ticked values, through `registry.RetrieveFields` or `registry.SendFields`. Both share one write rule, `pickFields`, which checks that every medium reference stays inside its system folder before anything changes. Copying into the registry leaves the hand-edited marks as they are, because the user picked each value. Copying into the folder goes through the existing `sendGame`, so it still creates the missing `gamelist.xml` entry when the ROM is present (decisions/037).

**Reason:** The review page is itself the confirmation: nothing is written without the user seeing each value and ticking it. That makes the separate send confirmation reached from this page (decisions/040) and the one-click retrieve (decisions/041) redundant. One page and one rule for both directions keeps them symmetric.

**Rejected alternatives:**
- Two fill-gaps-only buttons: simpler, but the user wants to overwrite selectively.
- A fill or replace mode per direction: all or nothing again, which is what the user rejected.
- Keeping "Retrieve" and only renaming it: it still replaced the whole entry.
