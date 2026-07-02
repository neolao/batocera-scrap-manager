---
date: 2026-07-02
status: accepted
---
# Match registry entries by ROM filename, not full relative path

**Context:** While designing the `remove` command (backlog item 005), the Product Owner pointed out that the registry stores each game's metadata as a flat file per system (`<registryFolder>/<system>/<basename>.json`, via `gameFileName`), discarding any subfolder structure the original ROM path might have had (e.g. `./sub/Sonic.zip`). Yet the in-memory matching key used everywhere (`indexOf`, backing `Import`, `ImportFromRomsFolder`, and `CompleteRomsFolder`) compared the *full* `Game.Path`, including that subfolder prefix. This meant two ROMs sharing the same filename in different subfolders of the same system were treated as distinct entries in memory, while silently colliding on disk (one file overwriting the other), since storage never reproduced the subfolder path in the first place.

**Decision:** Registry entries are now matched by system + ROM filename (`filepath.Base(Game.Path)`), not by the full relative path. This is used consistently by `indexOf` (and therefore `Import`, `ImportFromRomsFolder`, `CompleteRomsFolder`) and by the `remove` CLI command, which now takes a ROM filename (e.g. `Sonic.zip`) instead of a full path.

**Reason:** This aligns the in-memory identity of a game with what the registry actually reproduces on disk, closing the silent-collision gap described above, and lets users designate a game to remove without needing to know or reconstruct its original subfolder path.

**Rejected alternatives:** Keeping the full path as the matching key and instead making storage mirror the ROM's subfolder structure — rejected as unnecessary complexity for a registry that intentionally keeps a flat, self-contained per-system layout (see `decisions/001`). Also rejected: exposing the registry's internal JSON file path directly as the `remove` argument — it would leak a storage implementation detail as a stable CLI contract.
