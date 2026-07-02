# Module: cli
**Role:** Entry point and command-line interface of batocera-scrap-manager — parses arguments and dispatches to commands.
**Files:** `main.go`, `internal/cli/cli.go`, `internal/cli/config.go`, `internal/cli/update.go`, `internal/cli/scrape.go`, `internal/cli/remove.go`
**Exports:** `cli.Execute(args []string, out io.Writer) int`
**Depends on:** [`modules/config.md`](config.md), [`modules/registry.md`](registry.md), [`modules/site.md`](site.md)

## `config` subcommand
`internal/cli/config.go` implements `runConfig(args []string, out io.Writer) int`, dispatched by `Execute` on `args[0] == "config"`.

- `config --help` — prints `configUsage` (the subcommand syntax) and returns exit code 0, without touching the config file (backlog item 013).
- `config set-registry <path>` — sets the registry folder (converted to an absolute path via `internal/config`), persisted to the config file.
- `config add-roms-folder <path>` — adds a Batocera ROMs folder to watch (deduplicated by absolute path).
- `config list` — displays the configured registry (or "(not set)") and the list of ROMs folders (or "(none)").
- Any missing or unknown subcommand returns exit code 1.

The config file path is resolved via `config.DefaultPath()`: the `BATOCERA_SCRAP_MANAGER_CONFIG` environment variable if set, otherwise `os.UserConfigDir()/batocera-scrap-manager/config.json`.

## `update` subcommand
`internal/cli/update.go` implements `runUpdate(args []string, out io.Writer) int`, dispatched by `Execute` on `args[0] == "update"`.

- `update --help` — prints `updateUsage` and returns exit code 0, checked before any config/registry loading so it works even with no registry configured (backlog item 013).
- Loads the config, fails with exit code 1 if `RegistryFolder` is not set (explicit error message).
- Loads the registry.
- **Without an argument** (batch mode): calls `registry.ImportFromRomsFolder` (passing the ROMs folder, the registry folder, and a progress callback) for each configured ROMs folder; stops and returns exit code 1 as soon as a folder is not found.
  - The progress callback prints one line per system when its first game starts (`"<system>: <N> game(s)"`) and one line per game processed (`"  [<index>/<count>] <name>"`), as plain sequential output (no carriage-return overwrites or ANSI codes), so it stays readable when redirected to a file.
  - Saves the updated registry, then calls `site.Generate` to (re)generate the HTML consultation site inside the registry folder — see [`modules/site.md`](site.md) and [`decisions/006`](../decisions/006-auto-regenerate-html-site-on-update.md) — failing with exit code 1 if that fails, then prints a summary `"%d added, %d updated, %d unchanged"`.
  - No configured ROMs folder is a valid case (not an error): it prints a zero summary (and still (re)generates the site, e.g. showing "No games in the registry yet." on a first run), with no progress lines.
- **With one argument** (targeted mode, backlog item 012): the argument is a real path on disk to a single ROM; `runUpdateTargeted` resolves it via `resolveGamePath` (shared with `scrape`, see below) and calls `registry.ImportGame` for just that game, always printing its progress line unconditionally (there is at most one). Returns exit code 1 with a clear message if the path is outside every configured ROMs folder, or if `registry.ImportGame` returns `registry.ErrGameNotFound`. On success, saves the registry, regenerates the site, and prints the same `"%d added, %d updated, %d unchanged"` summary as the batch mode.

## `remove` subcommand
`internal/cli/remove.go` implements `runRemove(args []string, out io.Writer) int`, dispatched by `Execute` on `args[0] == "remove"`.

- `remove --help` — prints `removeUsage` and returns exit code 0, without removing anything (backlog item 013).
- Expects two positional arguments, `<system> <rom-filename>` (e.g. `Sonic.zip` — no need for the original subfolder, if any); prints the same usage message and returns exit code 1 if either is missing.
- Loads the config, fails with exit code 1 if `RegistryFolder` is not set (same message as `update`/`scrape`).
- Loads the registry, then calls `registry.Remove`. On `registry.ErrGameNotFound`, prints an error naming the system and filename and returns exit code 1; on any other error, prints it and returns exit code 1.
- On success, prints a one-line confirmation (`"removed <rom-filename> (system: <system>)"`) and returns exit code 0.

## `scrape` subcommand
`internal/cli/scrape.go` implements `runScrape(args []string, out io.Writer) int`, dispatched by `Execute` on `args[0] == "scrape"`.

- `scrape --help` — prints `scrapeUsage` and returns exit code 0, checked before any config/registry loading so it works even with no registry configured (backlog item 013).
- Loads the config, fails with exit code 1 if `RegistryFolder` is not set (same message as `update`).
- Loads the registry (read-only — never saved back).
- **Without an argument** (batch mode): calls `registry.CompleteRomsFolder` (passing the ROMs folder and the registry folder) for each configured ROMs folder; stops and returns exit code 1 as soon as a folder is not found.
  - The progress callback follows the same line format as `update`'s (`"<system>: <N> game(s)"` header, `"  [<index>/<count>] <romsFolder>: <name>"` per game), but unlike `update` it only fires — and only prints — for games that actually had a field completed from the registry; a game already fully complete, or unknown to the registry, produces no line, so identical metadata is silently skipped. The per-system header is triggered by the first event carrying a new system name, not by a fixed game index, since that first event does not always fall on that system's first game. Unlike `update`, each game line also repeats the ROMs folder currently being processed (rather than a one-time header per folder), so every printed change stays unambiguous even when several configured ROMs folders share the same system name (backlog item 006, see [`decisions/012`](../decisions/012-repeat-roms-folder-on-every-scrape-game-line.md)).
  - Prints a summary `"%d processed, %d completed, %d failed"`, still counting every game examined regardless of whether it produced a progress line.
  - No configured ROMs folder is a valid case (not an error): it prints a zero summary, with no progress lines.
- **With one argument** (targeted mode, backlog item 011): the argument is a real path on disk (resolved via `filepath.Abs`) to a single ROM; `runScrapeTargeted` finds which configured ROMs folder it falls under and derives the system from the next path component (`resolveGamePath`, defined in `scrape.go` and reused by `update.go`'s own targeted mode), then calls `registry.CompleteGame` for just that game, reusing the same progress-line format. Returns exit code 1 with a clear message if the path is outside every configured ROMs folder, or if `registry.CompleteGame` returns `registry.ErrGameNotFound` (no local entry, or no matching registry entry). On success, prints the same summary format with `processed` always `1` — see [`decisions/013`](../decisions/013-target-single-game-by-real-disk-path.md).
