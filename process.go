package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func startProcess(processName string, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: tim exec <command> [args...]")
		os.Exit(1)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	os.MkdirAll(".tim", 0755)
	pidFileName := fmt.Sprintf("./.tim/%s.pid", processName)
	os.WriteFile(pidFileName, fmt.Appendf(nil, "%d", cmd.Process.Pid), 0644)
	defer os.Remove(pidFileName)

	// Forward signals to the child process
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	go func() {
		for sig := range sigs {
			cmd.Process.Signal(sig)
		}
	}()

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
