package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type TmuxTerminal struct {
	timBinPath     string
	sessionName    string
	sessionCreated bool
}

func (t *TmuxTerminal) hasSession() bool {
	cmd := exec.Command("tmux", "has-session", "-t", t.sessionName)
	return cmd.Run() == nil
}

func (t *TmuxTerminal) OpenTab(title string, commands Commands, env Env) error {
	shellCmd := buildShellCommand(commands, env) + "; exec $SHELL"

	tabCmd := []string{t.timBinPath, "exec", title, "bash", "-c", shellCmd}

	var args []string
	if !t.sessionCreated && !t.hasSession() {
		args = append([]string{"tmux", "new-session", "-d", "-s", t.sessionName, "-n", title, "--"}, tabCmd...)
		t.sessionCreated = true
	} else {
		args = append([]string{"tmux", "new-window", "-t", t.sessionName, "-n", title, "--"}, tabCmd...)
	}

	fmt.Printf("[debug] %s\n", strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	return cmd.Run()
}

func (t *TmuxTerminal) Attach() error {
	if os.Getenv("TMUX") != "" {
		return nil
	}
	fmt.Printf("Attaching to tmux session %q...\n", t.sessionName)
	cmd := exec.Command("tmux", "attach-session", "-t", t.sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
