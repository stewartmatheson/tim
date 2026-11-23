package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
)

// startProcess runs script in shell as a child process, recording the
// child's PID in pidFile (an absolute path chosen by `tim up`) for as long
// as it runs.
//
// The script runs in a plain, non-interactive shell (`shell -c`) with the
// environment captured from a login, interactive one (see loginEnv). An
// interactive shell would ignore SIGTERM and run each command in its own
// process group, so `tim down` could not stop it. The child is put in its
// own process group, made the terminal's foreground group so Ctrl-C and
// keyboard input reach it, and `tim down` signals that whole group.
func startProcess(pidFileName, shell, script string) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	env, err := loginEnv(shell)
	shellCmd := []string{shell, "-c", script}
	if err != nil {
		fmt.Printf("tim: could not load your shell's environment (%v); running with a login shell only\n", err)
		env = os.Environ()
		shellCmd = []string{shell, "-l", "-c", script}
	}
	args := direnvWrap(cwd, envValue(env, "PATH"), shellCmd)

	path, err := lookPath(args[0], envValue(env, "PATH"))
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	cmd := exec.Command(path, args[1:]...)
	cmd.Args[0] = args[0]
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if isTerminal(os.Stdin) {
		cmd.SysProcAttr.Foreground = true
		cmd.SysProcAttr.Ctty = int(os.Stdin.Fd())
	}

	if err := cmd.Start(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(pidFileName), 0755); err != nil {
		fmt.Println("tim: could not create pid directory:", err)
	}
	if err := os.WriteFile(pidFileName, fmt.Appendf(nil, "%d", cmd.Process.Pid), 0644); err != nil {
		fmt.Println("tim: could not write pid file:", err)
	}
	defer os.Remove(pidFileName)

	// Forward signals sent to tim exec itself (including SIGHUP when the tab
	// is closed) to the child, then wait for it so the PID file is removed.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP)
	go func() {
		for sig := range sigs {
			cmd.Process.Signal(sig)
		}
	}()

	if err := cmd.Wait(); err != nil {
		os.Remove(pidFileName)
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
