package main

import (
	"fmt"
	"os"
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
		return &TmuxTerminal{timBinPath: timPath, sessionName: sessionName}, nil
	case "wt":
		return &WindowsTerminal{
			profileID:  os.Getenv("WT_PROFILE_ID"),
			timBinPath: timPath,
		}, nil
	case "":
		// Auto-detect
	default:
		return nil, fmt.Errorf("unknown terminal %q: supported values are \"tmux\" and \"wt\"", name)
	}

	if os.Getenv("TMUX") != "" {
		return &TmuxTerminal{timBinPath: timPath, sessionName: sessionName}, nil
	}

	if os.Getenv("WT_SESSION") != "" {
		return &WindowsTerminal{
			profileID:  os.Getenv("WT_PROFILE_ID"),
			timBinPath: timPath,
		}, nil
	}

	return nil, fmt.Errorf("could not detect terminal: set WT_SESSION or TMUX, or set \"terminal\" in .tim.yml")
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
