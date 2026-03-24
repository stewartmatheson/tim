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

**Panic on `tim exec` with no arguments** --- Running `tim exec` without
a command causes an index-out-of-bounds panic. The args check inside
`execCommand` is too late since `os.Args[2]` is accessed at the call
site before the function can validate.

**Signalling PID 0 on corrupted PID file** --- `tim down` parses PID
files with `fmt.Sscanf` without checking for errors. If the file
contains invalid data, `pid` remains 0 and `SIGTERM` is sent to PID 0,
which on Linux signals the entire process group.

**Shell injection via environment variables** --- Values in the `env`
config are interpolated directly into `export k=v` shell statements
without quoting or escaping. A malicious or accidental value like
`foo; rm -rf /` would execute arbitrary commands.

**Shell injection via command echo** --- The command string is embedded
in an `echo` statement without escaping. A command containing a single
quote can break out of the string.

**Unsanitised tab names used as filenames** --- Tab names are used
directly in PID file paths (`.tim/<name>.pid`) with no sanitisation. A
tab name like `../../etc/foo` would write outside the `.tim/` directory.

**Ignored errors in exec setup** --- `os.MkdirAll` and `os.WriteFile`
errors are silently ignored when creating the `.tim/` directory and
writing PID files. If either fails, the process runs but is not tracked.

**No tests** --- The project has no test coverage.
