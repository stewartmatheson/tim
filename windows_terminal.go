package main

import (
	"os/exec"
)

type WindowsTerminal struct {
	profileID  string
	timBinPath string
}

func (t *WindowsTerminal) OpenTab(title string, commands Commands, env Env) error {
	shellCmd := buildShellCommand(commands, env)
	cmd := exec.Command(
		"wt.exe", "-w", "0", "nt", "--title", title,
		"--profile", t.profileID,
		"--", "wsl.exe", "--",
		t.timBinPath, "exec", title, "bash", "-c", shellCmd,
	)
	return cmd.Start()
}
