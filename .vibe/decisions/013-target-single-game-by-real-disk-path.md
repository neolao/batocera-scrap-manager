---
date: 2026-07-02
status: accepted
---
# Target a single game for `scrape` by its real disk path

**Context:** Backlog item 011 requires `scrape` to optionally complete a single
game instead of every game in every configured ROMs folder. The game must be
identified from the command line, but the backlog note left open whether the
path should be relative to a configured ROMs folder, an absolute path on
disk, or expressed as an explicit `<system> <rom-filename>` pair (as `remove`
already does).

**Decision:** The path argument is a real filesystem path (absolute, or
relative to the current working directory, resolved via `filepath.Abs`) to
the ROM file as it actually exists inside one of the configured ROMs
folders. The system is derived automatically: the tool finds which
configured ROMs folder the path falls under, and takes the next path
component as the system name. The ROM filename used to match the registry is
the path's base name, consistent with the existing base-name-only matching
rule (see `decisions/005`).

**Reason:** This mirrors how a user or an external trigger (e.g. a file
manager action, a udev rule, a "scrape this ROM" hook) would naturally refer
to a specific ROM — by its real location on disk — without requiring them to
already know or type the system name. It keeps the command's interface to a
single argument.

**Rejected alternatives:**
- A path relative to a configured ROMs folder: rejected because it still
  requires the caller to know which configured folder is relevant, without
  actually simplifying anything over the absolute-path approach, while being
  more error-prone (ambiguous when several ROMs folders are configured).
- An explicit `<system> <rom-filename>` pair, symmetric with `remove`:
  rejected because it forces the caller to already know the system, which
  defeats the purpose of pointing at a file that already encodes it in its
  path.
