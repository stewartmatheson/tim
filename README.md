# tim

Spin up your dev environment in a terminal tab per process.

tim reads a `.tim.yml` config file and opens each process in its own
tab, with shared environment variables injected automatically.

## Requirements

tim currently supports:

-   **Windows Terminal** (`wt.exe`) via WSL2

Support for additional terminal multiplexers is planned, including:

-   [kitty](https://sw.kovidgoyal.net/kitty/)
-   [tmux](https://github.com/tmux/tmux)
-   [zellij](https://zellij.dev/)

## Installation

``` bash
go install .
```

Make sure `tim` is on your `$PATH` inside WSL.

## Usage

``` bash
tim up      # start all processes
tim down    # gracefully stop all running processes
```

## Configuration

Create a `.tim.yml` in your project root:

``` yaml
tabs:
  API Server: npm run dev
  Worker: go run ./worker
  Database: docker compose up db

env:
  NODE_ENV: development
  DATABASE_URL: postgres://localhost/myapp
```

Each key under `tabs` becomes the tab title; the value is the command to
run. Variables defined under `env` are available to all commands and can
be referenced with `$VAR` syntax.

## How it works

`tim up` opens a new terminal tab for each entry, launching the command
via `tim exec` so the process PID is tracked in `.tim/<name>.pid`.

`tim down` reads those PID files and sends `SIGTERM` to each process for
a graceful shutdown.

## Known Issues

**Orphaned PID files** --- If a tab is closed manually or the process is
killed with Ctrl-C, the `.tim/<name>.pid` file is left behind. On the
next `tim up`, that process will be skipped as if it were still running.
The fix is to run `tim down` to clear the stale PID files, or delete
them from `.tim/` by hand.

A better long-term solution would be to replace `syscall.Exec` with a
managed subprocess: spawn the process, forward signals to it, and clean
up the PID file on exit regardless of how the process terminates.
