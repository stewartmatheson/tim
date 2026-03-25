package main

import (
	"fmt"
	"os"
	"strings"
)

type Terminal interface {
	OpenTab(title string, command string, env Env) error
}

func DetectTerminal() (Terminal, error) {
	timPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("could not determine tim binary path: %w", err)
	}

	if os.Getenv("TMUX") != "" {
		return &TmuxTerminal{
			timBinPath: timPath,
		}, nil
	}

	if os.Getenv("WT_SESSION") != "" {
		return &WindowsTerminal{
			profileID:  os.Getenv("WT_PROFILE_ID"),
			timBinPath: timPath,
		}, nil
	}

	return nil, fmt.Errorf("unsupported terminal: set WT_SESSION or TMUX, or add an interface to tim for your terminal")
}

func buildEnvPrefix(env Env) string {
	parts := make([]string, 0, len(env))
	for k, v := range env {
		parts = append(parts, fmt.Sprintf("export %s=%s", k, v))
	}
	return strings.Join(parts, " && ")
}

func buildShellCommand(title string, command string, env Env) string {
	command = os.Expand(command, func(key string) string {
		return env[key]
	})
	envPrefix := buildEnvPrefix(env)

	parts := []string{}
	for _, part := range []string{envPrefix, command} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, " && ")
}
