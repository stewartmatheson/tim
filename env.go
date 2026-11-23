package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const envMarker = "__TIM_ENV_START__"

// loginEnvTimeout bounds how long a user's startup files may take.
const loginEnvTimeout = 15 * time.Second

// loginEnv returns the environment a login, interactive shell ends up with
// after reading the user's startup files (.zprofile and .zshrc,
// .bash_profile and .bashrc, ...). Many tools (nvm, asdf, pyenv, ...) are set
// up only in the interactive files, so a plain `shell -l -c` would miss them.
//
// The shell runs in its own session with no terminal, so it cannot take over
// the tab's terminal, and prints a marker before the environment so anything
// the startup files print is skipped.
func loginEnv(shell string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), loginEnvTimeout)
	defer cancel()

	script := fmt.Sprintf(`printf '%%s\0' %s; exec env -0`, envMarker)
	cmd := exec.CommandContext(ctx, shell, "-l", "-i", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("%s startup files took longer than %s", shell, loginEnvTimeout)
	}
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("%w: %s", err, msg)
		}
		return nil, err
	}

	marker := []byte(envMarker + "\x00")
	i := bytes.LastIndex(out, marker)
	if i < 0 {
		return nil, errors.New("no environment in shell output")
	}

	var env []string
	for _, kv := range bytes.Split(out[i+len(marker):], []byte{0}) {
		s := string(kv)
		name, _, ok := strings.Cut(s, "=")
		if !ok || name == "" {
			continue
		}
		switch name {
		case "PWD", "OLDPWD", "SHLVL", "_":
			// Describe the capturing shell, not the tab.
			continue
		}
		env = append(env, s)
	}
	return env, nil
}

// envValue returns the value of name in env, the last one winning as with
// os/exec.
func envValue(env []string, name string) string {
	value := ""
	for _, kv := range env {
		if k, v, ok := strings.Cut(kv, "="); ok && k == name {
			value = v
		}
	}
	return value
}

// lookPath is exec.LookPath against a given PATH rather than tim's own.
func lookPath(file, path string) (string, error) {
	if strings.Contains(file, "/") {
		return file, nil
	}
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			dir = "."
		}
		p := filepath.Join(dir, file)
		if info, err := os.Stat(p); err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
			return p, nil
		}
	}
	return "", fmt.Errorf("%s: not found in PATH", file)
}
