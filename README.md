# tim

Spin up your dev environment in a terminal tab per process.

tim reads a `.tim.yml` config file and opens each process in its own
tab, with shared environment variables injected automatically.

## Requirements

tim currently supports:

-   **Windows Terminal** (`wt.exe`) via WSL2
-   [tmux](https://github.com/tmux/tmux)
-   [kitty](https://sw.kovidgoyal.net/kitty/) (requires `allow_remote_control` enabled in `kitty.conf`)

Support for additional terminal multiplexers is planned, including:

-   [zellij](https://zellij.dev/)

## Installation

``` bash
go install .
```

Make sure `tim` is on your `$PATH` inside WSL.

## Usage

``` bash
tim up [project]      # start all processes
tim down [project]    # gracefully stop all running processes
```

tim looks for `.tim.yml` in the current directory, then in each parent
directory up to your home folder, so you can run it from anywhere inside
a project. The project name is only needed when `.tim.yml` defines more
than one project.

## Configuration

Create a `.tim.yml` in your project root:

``` yaml
tabs:
  API Server:
    - cd backend
    - npm install
    - npm run dev
  Worker: go run ./worker
  Database: docker compose up db

env:
  NODE_ENV: development
  DATABASE_URL: postgres://localhost/myapp
```

Each key under `tabs` becomes the tab title. The value is either a
single command string or a list of commands run sequentially (joined with
`&&`). Variables defined under `env` are available to all commands and
can be referenced with `$VAR` syntax.

By default tim auto-detects the terminal from the environment. To use a
specific terminal, set `terminal`:

``` yaml
terminal: tmux  # or "wt" for Windows Terminal, "kitty" for the kitty terminal

tabs:
  API Server: npm run dev
```

Supported values: `tmux`, `wt`, `kitty`.

### Working directory

Tabs start in the directory containing `.tim.yml`. Set `working_dir` to
start them somewhere else. Relative paths are resolved from the
`.tim.yml`'s directory, and `~` expands to your home folder:

``` yaml
working_dir: backend

tabs:
  API Server: npm run dev
```

A tab can also set its own `working_dir`. Write the tab as a mapping
with `working_dir` and `commands` (a string or a list). A relative path
is resolved from the project's `working_dir`:

``` yaml
working_dir: backend

tabs:
  API Server: npm run dev        # runs in backend/
  Worker:
    working_dir: worker          # runs in backend/worker/
    commands:
      - go build
      - ./worker
  Frontend:
    working_dir: ../frontend     # runs in frontend/
    commands: npm run dev
```

### Multiple projects

To keep several configurations in one file, put them under `projects`.
Each project takes the same keys as a top-level config (`terminal`,
`working_dir`, `tabs`, `env`), and its tabs can each set their own
`working_dir`, so a project can mix tabs from different directories:

``` yaml
projects:
  web:
    working_dir: frontend
    tabs:
      Dev Server: npm run dev
      Storybook: npm run storybook

  full-stack:
    terminal: tmux
    tabs:
      API:
        working_dir: backend
        commands: go run .
      Worker:
        working_dir: backend/worker
        commands:
          - go build
          - ./worker
      Frontend:
        working_dir: frontend
        commands: npm run dev
      Database:
        working_dir: infra
        commands: docker compose up db
    env:
      DATABASE_URL: postgres://localhost/myapp
```

Start and stop a project by name:

``` bash
tim up full-stack
tim down full-stack
```

-   The project name can be left out when the file has only one project.
    With several, tim lists the available projects and asks you to
    choose.
-   Without a `working_dir`, a project's tabs start in the directory
    containing `.tim.yml`. A tab's relative `working_dir` is resolved
    from its project's `working_dir`.
-   Every tab's `working_dir` is checked before any tabs open, so a typo
    stops `tim up` with an error naming the tab instead of opening a
    partial set of tabs.
-   Each project tracks its processes separately (in
    `.tim/<project>/`), so you can run several projects at once and
    `tim down` stops only the one you name.
-   With tmux, each project gets its own session, named after the
    project.
-   A file uses either `projects` or top-level `tabs`, not both.

## How it works

`tim up` opens a new terminal tab for each entry, launching the command
via `tim exec` so the process PID is tracked in `.tim/<name>.pid` (or
`.tim/<project>/<name>.pid`), next to `.tim.yml`.

Before running a tab's commands, `tim exec` starts your shell once as a
login, interactive shell (`$SHELL -l -i`) and copies the environment it
ends up with, so everything your startup files set up (`.zprofile`,
`.zshrc`, `.bashrc`, ...: Homebrew, nvm, asdf, ...) is available. The
commands themselves then run in a plain `$SHELL -c` with that environment,
in their own process group that is the tab's foreground group, so Ctrl-C
reaches them directly. If the startup files fail or take more than 15
seconds, tim prints a warning and runs the commands in a login shell
(`$SHELL -l -c`) instead, which reads `.zprofile` but not `.zshrc`. When
direnv is installed and the tab's directory has a `.envrc`, the commands
run through `direnv exec` too. In kitty, tabs also inherit the environment
of the shell you ran `tim up` from.

`tim down` sends `SIGTERM` to each tab's process group for a graceful
shutdown. Anything still running after 3 seconds gets `SIGHUP` (which also
closes a tab whose commands have finished, leaving its interactive shell
open), then `SIGKILL` after 2 more. PID files are removed once their
processes are gone.

## Known Issues

**Unsanitised tab names used as filenames** --- Tab names are used
directly in PID file paths (`.tim/<name>.pid`) with no sanitisation. A
tab name like `../../etc/foo` would write outside the `.tim/` directory.
