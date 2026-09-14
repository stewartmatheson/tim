package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Terminal interface {
	OpenTab(title string, commands Commands, env Env) error
}

func DetectTerminal(name string) (Terminal, error) {
	timPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("could not determine tim binary path: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not determine working directory: %w", err)
	}
	sessionName := filepath.Base(cwd)

	switch name {
	case "tmux":
		return &TmuxTerminal{timBinPath: timPath, sessionName: sessionName, cwd: cwd}, nil
	case "wt":
		return &WindowsTerminal{
			profileID:  os.Getenv("WT_PROFILE_ID"),
			timBinPath: timPath,
		}, nil
	case "kitty":
		t, err := newKittyTerminal(timPath)
		if err != nil {
			return nil, err
		}
		return t, nil
	case "":
		// Auto-detect
	default:
		return nil, fmt.Errorf("unknown terminal %q: supported values are \"tmux\", \"wt\", and \"kitty\"", name)
	}

	if os.Getenv("TMUX") != "" {
		return &TmuxTerminal{timBinPath: timPath, sessionName: sessionName, cwd: cwd}, nil
	}

	if os.Getenv("WT_SESSION") != "" {
		return &WindowsTerminal{
			profileID:  os.Getenv("WT_PROFILE_ID"),
			timBinPath: timPath,
		}, nil
	}

	if os.Getenv("KITTY_WINDOW_ID") != "" {
		t, err := newKittyTerminal(timPath)
		if err != nil {
			return nil, err
		}
		return t, nil
	}

	return nil, fmt.Errorf("could not detect terminal: set WT_SESSION, TMUX, or KITTY_WINDOW_ID, or set \"terminal\" in .tim.yml")
}

// userShell returns the shell tim should use to run tab commands. It prefers
// the shell that actually invoked tim (tim's parent process) over $SHELL,
// since $SHELL just records the login shell and goes stale the moment
// someone execs into a different shell (e.g. a terminal that inherits
// $SHELL=/bin/bash but is actually running a separately compiled zsh).
// It falls back to $SHELL, then a hardcoded default, when the parent
// process can't be resolved (e.g. non-Linux, no /proc).
func userShell() string {
	if shell := parentShell(); shell != "" {
		return shell
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "bash"
}

// parentShell resolves the executable path of tim's parent process via
// /proc, which is only available on Linux (including WSL).
func parentShell() string {
	exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", os.Getppid()))
	if err != nil {
		return ""
	}
	return exe
}

// direnvWrap prepends "direnv exec <cwd>" to args when direnv is installed
// and cwd has a .envrc, so tab commands see the same environment direnv's
// shell hook would normally export. The hook itself never fires for tab
// commands because they run via a non-interactive `shell -c`, which skips
// the prompt cycle direnv hooks into.
func direnvWrap(cwd string, args []string) []string {
	if _, err := exec.LookPath("direnv"); err != nil {
		return args
	}
	if _, err := os.Stat(filepath.Join(cwd, ".envrc")); err != nil {
		return args
	}
	return append([]string{"direnv", "exec", cwd}, args...)
}

func buildEnvPrefix(env Env) string {
	parts := make([]string, 0, len(env))
	for k, v := range env {
		parts = append(parts, fmt.Sprintf("export %s=%s", k, v))
	}
	return strings.Join(parts, " && ")
}

func buildShellCommand(commands Commands, env Env) string {
	expanded := make([]string, len(commands))
	for i, cmd := range commands {
		expanded[i] = os.Expand(cmd, func(key string) string {
			return env[key]
		})
	}

	parts := []string{}
	if envPrefix := buildEnvPrefix(env); envPrefix != "" {
		parts = append(parts, envPrefix)
	}
	parts = append(parts, expanded...)
	return strings.Join(parts, " && ")
}
