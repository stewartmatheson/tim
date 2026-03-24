package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Terminal interface {
	OpenTab(title string, command string, env Env) error
}

func DetectTerminal() (Terminal, error) {
	if os.Getenv("WT_SESSION") != "" {
		timPath, err := exec.LookPath("tim")
		if err != nil {
			return nil, fmt.Errorf("tim not found in PATH: %w", err)
		}
		return &WindowsTerminal{
			profileID:  os.Getenv("WT_PROFILE_ID"),
			timBinPath: timPath,
		}, nil
	}
	return nil, fmt.Errorf("unsupported terminal: set WT_SESSION or add an interface to tim for your terminal")
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
	preamble := "echo 'Executing: " + command + "'"
	divider := "echo '-------------------------------------'"

	parts := []string{}
	for _, part := range []string{envPrefix, preamble, divider, command} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, " && ")
}
