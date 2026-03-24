# tim

Spin up your dev environment in a terminal tab per process.

tim reads a `.tim.yml` config file and opens each process in its own Windows Terminal tab, with shared environment variables injected automatically.

## Requirements

- Windows Terminal (`wt.exe`)
- WSL2

## Installation

```bash
go install .
```

Make sure `tim` is on your `$PATH` inside WSL.

## Usage

```bash
tim up      # start all processes
tim down    # gracefully stop all running processes
```

## Configuration

Create a `.tim.yml` in your project root:

```yaml
tabs:
  API Server: npm run dev
  Worker: go run ./worker
  Database: docker compose up db

env:
  NODE_ENV: development
  DATABASE_URL: postgres://localhost/myapp
```

Each key under `tabs` becomes the tab title; the value is the command to run. Variables defined under `env` are available to all commands and can be referenced with `$VAR` syntax.

## How it works

`tim up` opens a new Windows Terminal tab for each entry, launching the command via `tim exec` so the process PID is tracked in `.tim/<name>.pid`.

`tim down` reads those PID files and sends `SIGTERM` to each process for a graceful shutdown.
