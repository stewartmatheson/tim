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

tim is a terminal multiplexer launcher for dev environments. It reads `.tim.yml`, opens a terminal tab per process, injects shared env vars, and tracks PIDs for graceful shutdown.

Commands: `tim up` (start tabs), `tim down` (SIGTERM via PID files), `tim exec <name> <cmd...>` (internal — runs a command and writes a PID file).

## Architecture

All code is in a single `main` package (three files):

- **main.go** — CLI dispatch (`up`/`down`/`exec`), config loading from `.tim.yml`, PID file management in `.tim/` directory.
- **terminal.go** — `Terminal` interface (`OpenTab`), terminal auto-detection via env vars, and shell command building (env export prefix + command). Each terminal backend implements this interface.
- **terminal_wt.go** — Windows Terminal implementation. Opens tabs via `wt.exe` → `wsl.exe` → `tim exec` chain.

Adding a new terminal: implement the `Terminal` interface and add detection logic to `DetectTerminal()` in terminal.go.

## Key Design Detail

`tim exec` uses `syscall.Exec` (not `os/exec`) — it replaces the Go process entirely with the target command. This means no Go code runs after exec succeeds, so cleanup (like PID file removal) cannot happen from within the process. This is the root cause of the orphaned PID file problem.
