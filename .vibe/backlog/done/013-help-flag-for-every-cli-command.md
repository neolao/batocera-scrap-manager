---
status: done
---
# --help For Every CLI Command

## Description
Only `batocera-scrap-manager --help` exists today, and it merely lists the available commands. Every subcommand (`config`, `update`, `scrape`, `remove`) must accept its own `--help` flag, printing a detailed usage message specific to that subcommand (expected arguments, options, examples), so the user does not have to guess the exact syntax.

## Acceptance Criteria
- [ ] The user can run `batocera-scrap-manager config --help`, `update --help`, `scrape --help` and `remove --help`, and gets a usage message specific to that command (not the generic global one)
- [ ] Every help message describes the expected arguments/subcommands (e.g. `remove --help` mentions `<system> <rom-filename>`)
- [ ] Running `<command> --help` returns exit code 0, without performing the command's real action
- [ ] The existing behavior of `batocera-scrap-manager --help` (with no subcommand) stays unchanged

## Notes
Several subcommands (`remove`) already print a usage message when arguments are missing (`removeUsage`); it could be reused for `--help`. To settle: should that message also be printed automatically on an argument error (the behavior already in place for `remove`), or only on explicit request through `--help`?
