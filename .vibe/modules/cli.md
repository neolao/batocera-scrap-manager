# Module: cli
**Role:** Entry point and command-line interface of batocera-scrap-manager — parses arguments and dispatches to commands.
**Files:** `main.go`, `internal/cli/cli.go`, `internal/cli/common.go`, `internal/cli/config.go`, `internal/cli/update.go`, `internal/cli/scrape.go`, `internal/cli/remove.go`, `internal/cli/protect.go`, `internal/cli/serve.go`
**Exports:** `cli.Execute(args []string, out io.Writer) int`
**Depends on:** [`modules/config.md`](config.md), [`modules/registry.md`](registry.md), [`modules/store.md`](store.md), [`modules/webui.md`](webui.md)

`internal/cli/common.go` factors out logic shared by `update`, `scrape`, and `remove`: `loadConfigAndRegistry` (load the config, verify `RegistryFolder` is set, load the registry — writing the same error message and returning `ok=false` on any failure), `saveAndGenerateSite` (a thin wrapper over [`modules/store.md`](store.md)'s `Save`, which writes the registry then regenerates the HTML site; both failure modes are reported the same way here), `newCompletionProgressReporter` (the `registry.CompletionEvent` line-printing closure shared by `scrape`'s batch and targeted modes), and `newImportProgressReporter` (the `registry.ProgressEvent` line-printing closure used by `update`'s batch mode — its targeted mode keeps its own inline closure, since it always prints the header regardless of the event's `GameIndex`, unlike the batch closure which only prints on `GameIndex == 1`).

## `config` subcommand
`internal/cli/config.go` implements `runConfig(args []string, out io.Writer) int`, dispatched by `Execute` on `args[0] == "config"`, which itself dispatches to `runConfigSetRegistry`, `runConfigAddRomsFolder`, or `runConfigList`.

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
  - Saves the updated registry and (re)generates the HTML consultation site inside the registry folder, through [`modules/store.md`](store.md) — see [`modules/site.md`](site.md) and [`decisions/006`](../decisions/006-auto-regenerate-html-site-on-update.md) — failing with exit code 1 if that fails, then prints a summary `"%d added, %d updated, %d unchanged"`.
  - No configured ROMs folder is a valid case (not an error): it prints a zero summary (and still (re)generates the site, e.g. showing "No games in the registry yet." on a first run), with no progress lines.
- **With one argument** (targeted mode, backlog item 012): the argument is a real path on disk to a single ROM; `runUpdateTargeted` resolves it via `resolveGamePath` (shared with `scrape`, see below) and calls `registry.ImportGame` for just that game, always printing its progress line unconditionally (there is at most one). Returns exit code 1 with a clear message if the path is outside every configured ROMs folder, or if `registry.ImportGame` returns `registry.ErrGameNotFound`. On success, saves the registry, regenerates the site, and prints the same `"%d added, %d updated, %d unchanged"` summary as the batch mode.

## `remove` subcommand
`internal/cli/remove.go` implements `runRemove(args []string, out io.Writer) int`, dispatched by `Execute` on `args[0] == "remove"`.

- `remove --help` — prints `removeUsage` and returns exit code 0, without removing anything (backlog item 013).
- Expects two positional arguments, `<system> <rom-filename>` (e.g. `Sonic.zip` — no need for the original subfolder, if any); prints the same usage message and returns exit code 1 if either is missing.
- Loads the config, fails with exit code 1 if `RegistryFolder` is not set (same message as `update`/`scrape`).
- Loads the registry, then calls `registry.Remove`. On `registry.ErrGameNotFound`, prints an error naming the system and filename and returns exit code 1; on any other error, prints it and returns exit code 1.
- `registry.ErrMediaLeftBehind` is the exception: the game itself is gone, so it prints the normal confirmation followed by a `warning:` line naming the leftover files, and returns exit code **0** — reporting it as a failure would have the user retry a removal that already happened.
- On success, prints a one-line confirmation (`"removed <rom-filename> (system: <system>)"`) and returns exit code 0.
- Unlike the web UI's deletion, it does **not** regenerate the consultation site: `index.html` still lists the removed game until the next `update`. Pre-existing behaviour, untouched by backlog item 016.

## `protect` and `unprotect` subcommands
`internal/cli/protect.go` implements `runProtect` and `runUnprotect`, dispatched by `Execute` on `args[0] == "protect"` / `"unprotect"`. Both are one call to `applyProtection`, which differs only by the usage constant, the confirmation verb, and the `registry.Protect`/`registry.Unprotect` function it is handed — two verbs rather than one command with a flag, so both halves of the feature appear in the top-level `--help` (backlog item 018).

- `--help` — prints `protectUsage` / `unprotectUsage` and returns exit code 0, before any config or registry loading. `unprotectUsage` states that the lift also clears the marks left by corrections made in the web UI, since that is not recoverable from the CLI.
- Expects exactly two positional arguments, `<system> <rom-filename>`. `parseProtectionArgs` is deliberately stricter than `runRemove`'s reading: any argument starting with `-` is refused by name, and a count other than 2 is refused rather than silently ignored (`remove nes a.zip b.zip` still swallows `b.zip`). Both refusals print `error: ...` then the usage, and return exit code 1.
- Converts the ROM filename to the registry's game ID with the exported `registry.GameID` — the CLI addresses a game by filename while the registry addresses it by ID, and that rule is never re-derived locally (see [`decisions/014`](../decisions/014-dedupe-by-extension-stripped-filename-too.md)).
- Applies the change, then persists through `saveAndGenerateSite`. On `registry.ErrGameNotFound`, prints `remove`'s exact wording (`error: no game found for system %q and filename %q`) and returns exit code 1.
- On success, prints `"protected <rom-filename> (system: <system>)"` / `"unprotected ..."` and returns exit code 0. Both are idempotent: re-running either succeeds and stores the same state.

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

## `serve` subcommand
`internal/cli/serve.go` implements `runServe(args []string, out io.Writer) int`, dispatched by `Execute` on `args[0] == "serve"` (backlog item 014). It is the only long-running foreground command of the tool.

- `serve --help` — prints `serveUsage` and returns exit code 0, checked before any flag parsing or config/registry loading, like every other command.
- Options are parsed with a `flag.FlagSet` (`ContinueOnError`, output discarded, `Usage` silenced) rather than by hand — the only command with a flag so far. `parseServeArgs` returns a three-state `serveArgsOutcome` (`serveArgsParsed` / `serveArgsHelpRequested` / `serveArgsInvalid`) so the flag package's implicit `-h` exits 0 with the usage, while a malformed command line exits 1: an unknown flag, a missing `--addr` value, an `--addr` without a port (rejected upfront by `net.SplitHostPort`, suggesting the `:8080` form), or any leftover positional argument each print `error: ...` followed by `serveUsage`.
- Loads the config and registry via `loadConfigAndRegistry` — same error message as the other commands when the registry is not configured. The registry is a startup snapshot: it is never reloaded while the server runs, which `serveUsage` states explicitly.
- Installs the signal handler (`signal.NotifyContext` on `os.Interrupt`/`SIGTERM`) *before* listening and printing, so an interrupt arriving right after the address line is always caught.
- Listens with `net.Listen("tcp", addr)` (default `0.0.0.0:8080`), exiting 1 on failure (e.g. address already in use), then prints, via `listeningLines`, the address taken from `listener.Addr()` — not from the flag, so `:0` and wildcard resolution are truthful — plus a browser URL, substituting `localhost` for a wildcard host (`0.0.0.0`, `::`, empty) since that URL is not usable as-is on every platform.
- `serveUntil` runs an `http.Server` (`ReadHeaderTimeout`, `ErrorLog` discarded so per-connection noise never pollutes the command's own output) over the handler built by [`modules/webui.md`](webui.md), shuts it down gracefully when the context is cancelled (5s grace period for in-flight requests), and returns exit code 0 on `http.ErrServerClosed` — so a service manager stopping the process does not read it as a crash.
