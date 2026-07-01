# Module: cli
**Role:** Entry point and command-line interface of batocera-scrap-manager — parses arguments and dispatches to commands.
**Files:** `main.go`, `internal/cli/cli.go`, `internal/cli/config.go`, `internal/cli/update.go`
**Exports:** `cli.Execute(args []string, out io.Writer) int`
**Depends on:** [`modules/config.md`](config.md), [`modules/registry.md`](registry.md)

## `config` subcommand
`internal/cli/config.go` implements `runConfig(args []string, out io.Writer) int`, dispatched by `Execute` on `args[0] == "config"`.

- `config set-registry <path>` — sets the registry path (converted to an absolute path via `internal/config`), persisted to the config file.
- `config add-roms-folder <path>` — adds a Batocera ROMs folder to watch (deduplicated by absolute path).
- `config list` — displays the configured registry (or "(not set)") and the list of ROMs folders (or "(none)").
- Any missing or unknown subcommand returns exit code 1.

The config file path is resolved via `config.DefaultPath()`: the `BATOCERA_SCRAP_MANAGER_CONFIG` environment variable if set, otherwise `os.UserConfigDir()/batocera-scrap-manager/config.json`.

## `update` subcommand
`internal/cli/update.go` implements `runUpdate(out io.Writer) int`, dispatched by `Execute` on `args[0] == "update"`.

- Loads the config, fails with exit code 1 if `RegistryPath` is not set (explicit error message).
- Loads the registry, then calls `registry.ImportFromRomsFolder` for each configured ROMs folder; stops and returns exit code 1 as soon as a folder is not found.
- Saves the updated registry, then prints a summary `"%d added, %d updated, %d unchanged"`.
- No configured ROMs folder is a valid case (not an error): it prints a zero summary.
