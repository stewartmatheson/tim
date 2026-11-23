package main

import (
	"fmt"
	"os/exec"
	"strings"
)

type KittyTerminal struct {
	timBinPath string
}

// newKittyTerminal checks that kitty's remote-control socket is reachable
// before tim commits to using it, so a misconfigured kitty.conf surfaces as
// one clear error instead of a confusing failure per tab.
func newKittyTerminal(timBinPath string) (*KittyTerminal, error) {
	cmd := exec.Command("kitty", "@", "ls")
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf(
			"kitty remote control is not available (%s): %s\n"+
				"tim needs remote control to open tabs. Add `allow_remote_control yes` "+
				"(or `allow_remote_control socket-only` with a `listen_on` socket) to your "+
				"kitty.conf, then restart kitty",
			err, strings.TrimSpace(string(out)),
		)
	}
	return &KittyTerminal{timBinPath: timBinPath}, nil
}

func (t *KittyTerminal) OpenTab(title, cwd, pidFile string, commands Commands, env Env) error {
	shell := userShell()
	shellCmd := buildShellCommand(commands, env) + "; exec " + shell

	tabCmd := execArgs(t.timBinPath, pidFile, shell, shellCmd)

	args := []string{
		"kitty", "@", "launch",
		"--type=tab",
		"--tab-title", title,
		"--cwd", cwd,
		// Over remote control this copies the environment of the
		// `kitty @` process (i.e. tim's, inherited from the shell that ran
		// `tim up`), not the environment kitty itself started with.
		"--copy-env",
		"--",
	}
	args = append(args, tabCmd...)

	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kitty @ launch failed: %w (%s)", err, string(out))
	}
	return nil
}
