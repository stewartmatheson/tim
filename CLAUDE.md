# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
go build -o tim .    # build binary
go install .         # install to $GOPATH/bin
go vet ./...         # static analysis (no linter configured beyond vet)
```

There are no tests yet. When adding tests, use standard `go test ./...`.

**Important:** `tim up` spawns tabs asynchronously via the current binary (`os.Executable()`). Do not use `go run .` — the temp binary may be deleted before tabs start. Use `go build -o ./tim . && ./tim up` for local dev, or `go install .` to update the installed binary.

## What tim Does

tim is a terminal multiplexer launcher for dev environments. It reads `.tim.yml` (searched from the current directory up to `$HOME`), opens a terminal tab per process, injects shared env vars, and tracks PIDs for graceful shutdown. A `.tim.yml` holds either one top-level project or several named ones under `projects`.

Commands: `tim up [project]` (start tabs), `tim down [project]` (SIGTERM via PID files), `tim exec <pid-file> <shell> <script>` (internal — runs a tab's commands and writes their PID to the given absolute path).

## Architecture

All code is in a single `main` package:

- **main.go** — CLI dispatch (`up`/`down`/`exec`) and PID file checks.
- **config.go** — finding and parsing `.tim.yml`, selecting a project, and resolving `working_dir` and the PID directory (`.tim/` or `.tim/<project>/` next to the config file).
- **process.go** — `tim exec`: runs the child process and maintains its PID file.
- **env.go** — captures the user's login, interactive shell environment for `tim exec`.
- **terminal.go** — `Terminal` interface (`OpenTab`), terminal auto-detection via env vars, shell command building (env export prefix + command), and direnv wrapping. Each terminal backend implements this interface.
- **tmux_terminal.go**, **kitty_terminal.go**, **windows_terminal.go** — terminal backends. Windows Terminal opens tabs via `wt.exe` → `wsl.exe` → `tim exec` chain.

Adding a new terminal: implement the `Terminal` interface and add detection logic to `DetectTerminal()` in terminal.go.

## Key Design Detail

`tim exec <pid-file> <shell> <script>` (env.go, process.go) first captures the environment of a login, interactive shell (`shell -l -i`, run in its own session with no terminal, output after a marker), then runs the script in a non-interactive `shell -c` with that environment. Don't run the script itself in an interactive shell: interactive shells ignore SIGTERM/SIGINT/SIGQUIT and put each command in its own process group, so `tim down` could not stop them. The child gets its own process group (made the terminal's foreground group), and its PID, which is also the group ID, goes in the PID file. `tim down` signals the whole group: SIGTERM, then SIGHUP (closes a leftover interactive shell after the commands finish), then SIGKILL.

PID files are cleaned up by `tim exec` via `defer` on normal exit and explicitly before `os.Exit` on error paths; it forwards SIGTERM/SIGINT/SIGQUIT/SIGHUP to the child so closing a tab cleans up too. As a safety net, `tim up` checks whether a PID file's process is still alive before skipping a tab, and `tim down` removes stale files.
