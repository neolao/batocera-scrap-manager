# Module: cli
**Role:** Entry point and command-line interface of batocera-scrap-manager — parses arguments and dispatches to commands.
**Files:** `main.go`, `internal/cli/cli.go`, `internal/cli/config.go`, `internal/cli/update.go`, `internal/cli/scrape.go`, `internal/cli/remove.go`
**Exports:** `cli.Execute(args []string, out io.Writer) int`
**Depends on:** [`modules/config.md`](config.md), [`modules/registry.md`](registry.md)

## `config` subcommand
`internal/cli/config.go` implements `runConfig(args []string, out io.Writer) int`, dispatched by `Execute` on `args[0] == "config"`.

- `config set-registry <path>` — sets the registry folder (converted to an absolute path via `internal/config`), persisted to the config file.
- `config add-roms-folder <path>` — adds a Batocera ROMs folder to watch (deduplicated by absolute path).
- `config list` — displays the configured registry (or "(not set)") and the list of ROMs folders (or "(none)").
- Any missing or unknown subcommand returns exit code 1.

The config file path is resolved via `config.DefaultPath()`: the `BATOCERA_SCRAP_MANAGER_CONFIG` environment variable if set, otherwise `os.UserConfigDir()/batocera-scrap-manager/config.json`.

## `update` subcommand
`internal/cli/update.go` implements `runUpdate(out io.Writer) int`, dispatched by `Execute` on `args[0] == "update"`.

- Loads the config, fails with exit code 1 if `RegistryFolder` is not set (explicit error message).
- Loads the registry, then calls `registry.ImportFromRomsFolder` (passing the ROMs folder, the registry folder, and a progress callback) for each configured ROMs folder; stops and returns exit code 1 as soon as a folder is not found.
- The progress callback prints one line per system when its first game starts (`"<system>: <N> game(s)"`) and one line per game processed (`"  [<index>/<count>] <name>"`), as plain sequential output (no carriage-return overwrites or ANSI codes), so it stays readable when redirected to a file.
- Saves the updated registry, then prints a summary `"%d added, %d updated, %d unchanged"`.
- No configured ROMs folder is a valid case (not an error): it prints a zero summary, with no progress lines.

## `remove` subcommand
`internal/cli/remove.go` implements `runRemove(args []string, out io.Writer) int`, dispatched by `Execute` on `args[0] == "remove"`.

- Expects two positional arguments, `<system> <rom-filename>` (e.g. `Sonic.zip` — no need for the original subfolder, if any); prints a usage message and returns exit code 1 if either is missing.
- Loads the config, fails with exit code 1 if `RegistryFolder` is not set (same message as `update`/`scrape`).
- Loads the registry, then calls `registry.Remove`. On `registry.ErrGameNotFound`, prints an error naming the system and filename and returns exit code 1; on any other error, prints it and returns exit code 1.
- On success, prints a one-line confirmation (`"removed <rom-filename> (system: <system>)"`) and returns exit code 0.

## `scrape` subcommand
`internal/cli/scrape.go` implements `runScrape(out io.Writer) int`, dispatched by `Execute` on `args[0] == "scrape"`.

- Loads the config, fails with exit code 1 if `RegistryFolder` is not set (same message as `update`).
- Loads the registry (read-only — never saved back), then calls `registry.CompleteRomsFolder` (passing the ROMs folder and the registry folder) for each configured ROMs folder; stops and returns exit code 1 as soon as a folder is not found.
- The progress callback follows the same line format as `update`'s (`"<system>: <N> game(s)"` header, `"  [<index>/<count>] <name>"` per game), but unlike `update` it only fires — and only prints — for games that actually had a field completed from the registry; a game already fully complete, or unknown to the registry, produces no line, so identical metadata is silently skipped. The per-system header is triggered by the first event carrying a new system name, not by a fixed game index, since that first event does not always fall on that system's first game.
- Prints a summary `"%d processed, %d completed, %d failed"`, still counting every game examined regardless of whether it produced a progress line.
- No configured ROMs folder is a valid case (not an error): it prints a zero summary, with no progress lines.
