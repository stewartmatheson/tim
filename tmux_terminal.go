package main

import (
	"fmt"
	"os/exec"
	"strings"
)

type TmuxTerminal struct {
	timBinPath string
}

func (t *TmuxTerminal) OpenTab(title string, command string, env Env) error {
	shellCmd := buildShellCommand(title, command, env)

	args := []string{
		"tmux",
		"new-window",
		"-n",
		title,
		"--",
		t.timBinPath,
		"exec",
		title,
		"bash",
		"-c",
		shellCmd,
	}

	fmt.Printf("[debug] %s\n", strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	return cmd.Run()
}
