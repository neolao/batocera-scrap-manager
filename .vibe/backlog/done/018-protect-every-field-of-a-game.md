---
status: done
depends_on: [015]
---
# Protect Every Field Of A Game

## Description
Protection against being overwritten by `update` is currently decided field by field, and only as a side effect of a correction made in the web UI: only the fields whose value really changes on save are marked in `manual_fields` (decision 017). There is no way to say "this game is good, leave it alone" without artificially modifying each of its fields. It must be possible to protect a whole game — that is, to mark every one of its editable fields at once — from the command line and from the game's page in the web UI, and to lift that protection.

## Acceptance Criteria
- [ ] A CLI command protects a game named by its system and its ROM file: every one of its editable fields ends up in `manual_fields`, without any of their values changing
- [ ] The same command can lift the protection: `manual_fields` becomes empty again and the game is refreshed by `update` once more
- [ ] After protecting then running `update` on a ROMs folder whose `gamelist.xml` carries different values, no editable field of the game is modified in the registry
- [ ] The game's page in the web UI offers the protection and its lift, states the current state, and the change takes effect without restarting `serve`
- [ ] Protecting an unknown game answers `ErrGameNotFound` on the CLI side and 404 on the web side, without modifying anything in the registry

## Notes
The logic belongs to the registry, not to the interfaces: add to `internal/registry/metadata.go` what it takes to mark/unmark the whole of `editableFields`, reused as-is by the command and by the web handler — a guard placed in the web UI would be bypassed by the CLI (the same reasoning as decision 017). Whole-game protection must stay compatible with the existing per-field protection: it is its limit case, not a second mechanism, and the edit form's "hand back" checkbox must keep working on a fully protected game.

Open question: the shape of the command — `protect <system> <rom-filename>` with a symmetrical `unprotect`, or a single command with a flag. Follow the convention of the existing commands (`remove <system> <rom-filename>`) and provide a `--help` as item 013 requires.

Also worth noting for the help and the docs: protecting a game does not stop Batocera from displaying the value held by the ROMs folder's `gamelist.xml`, which is left untouched.
