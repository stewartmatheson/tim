package main

import (
	"os/exec"
)

type WindowsTerminal struct {
	profileID  string
	timBinPath string
}

func (t *WindowsTerminal) OpenTab(title, cwd, pidFile string, commands Commands, env Env) error {
	shellCmd := buildShellCommand(commands, env)
	args := []string{
		"wt.exe", "-w", "0", "nt", "--title", title,
		"--profile", t.profileID,
		"--", "wsl.exe", "--cd", cwd, // Linux path inside WSL
		"--",
	}
	args = append(args, execArgs(t.timBinPath, pidFile, userShell(), shellCmd)...)
	cmd := exec.Command(args[0], args[1:]...)
	return cmd.Start()
}
