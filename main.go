package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func up(projectName string) {
	project, err := loadProject(projectName)
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	term, err := DetectTerminal(project.Terminal, project.sessionName())
	if err != nil {
		fmt.Println("Error detecting terminal:", err)
		os.Exit(1)
	}

	for title, tab := range project.Tabs {
		pidFile := project.pidFile(title)
		if _, err := os.Stat(pidFile); err == nil {
			if isProcessRunning(pidFile) {
				fmt.Printf("Skipping %q: already running (pid file exists)\n", title)
				continue
			}
			fmt.Printf("Removing stale pid file for %q\n", title)
			os.Remove(pidFile)
		}
		if err := term.OpenTab(title, tab.Dir, pidFile, tab.Commands, project.Env); err != nil {
			fmt.Printf("Error opening tab %q: %v\n", title, err)
		}
	}

	if t, ok := term.(*TmuxTerminal); ok {
		if err := t.Attach(); err != nil {
			fmt.Printf("Error attaching to tmux session: %v\n", err)
		}
	}
}

// readPid returns the PID in pidFile, or 0 if it does not hold a usable one.
func readPid(pidFile string) int {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 1 {
		return 0
	}
	return pid
}

// alive reports whether a process (pid > 0) or process group (pid < 0)
// exists. EPERM means it exists but belongs to someone else.
func alive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func isProcessRunning(pidFile string) bool {
	pid := readPid(pidFile)
	return pid != 0 && alive(pid)
}

// stopTarget is one tab being shut down by `tim down`.
type stopTarget struct {
	pidFile string
	pid     int
	target  int // what to signal: -pgid for the tab's process group, or pid
}

func (t stopTarget) running() bool { return alive(t.target) }

// Grace periods for `tim down`: SIGTERM first, then SIGHUP (which also
// closes a tab whose commands have finished and left an interactive shell
// behind, since interactive shells ignore SIGTERM), then SIGKILL.
var stopSteps = []struct {
	sig  syscall.Signal
	name string
	wait time.Duration
}{
	{syscall.SIGTERM, "SIGTERM", 3 * time.Second},
	{syscall.SIGHUP, "SIGHUP", 2 * time.Second},
	{syscall.SIGKILL, "SIGKILL", 1 * time.Second},
}

func down(projectName string) {
	project, err := loadProject(projectName)
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	pidFiles, err := filepath.Glob(filepath.Join(project.PidDir, "*.pid"))
	if err != nil || len(pidFiles) == 0 {
		fmt.Println("No running processes found")
		return
	}

	var targets []stopTarget
	for _, pidFile := range pidFiles {
		pid := readPid(pidFile)
		if pid == 0 || !alive(pid) {
			fmt.Printf("Removing stale pid file %s\n", pidFile)
			os.Remove(pidFile)
			continue
		}
		t := stopTarget{pidFile: pidFile, pid: pid, target: pid}
		// tim exec makes the child its own process group leader; signal the
		// whole group so the commands the shell started stop too. Anything
		// else (an old pid file, or a reused PID) gets only the process.
		if pgid, err := syscall.Getpgid(pid); err == nil && pgid == pid {
			t.target = -pid
		}
		targets = append(targets, t)
	}

	for i, step := range stopSteps {
		var remaining []stopTarget
		for _, t := range targets {
			if !t.running() {
				continue
			}
			if err := syscall.Kill(t.target, step.sig); err != nil && !errors.Is(err, syscall.ESRCH) {
				fmt.Printf("Error sending %s to %d: %v\n", step.name, t.pid, err)
				continue
			}
			if i == 0 {
				fmt.Printf("Sent %s to %d (%s)\n", step.name, t.pid, t.pidFile)
			} else {
				fmt.Printf("%d still running, sent %s\n", t.pid, step.name)
			}
			remaining = append(remaining, t)
		}
		targets = remaining
		waitStopped(targets, step.wait)
	}

	// tim exec removes its own pid file when its child exits; clear up any
	// it could not (if it was killed itself, for instance).
	for _, pidFile := range pidFiles {
		if pid := readPid(pidFile); pid == 0 || !alive(pid) {
			os.Remove(pidFile)
		}
	}
	for _, t := range targets {
		if t.running() {
			fmt.Printf("Could not stop %d (%s)\n", t.pid, t.pidFile)
		}
	}
}

// waitStopped waits until none of targets are running, or timeout passes.
func waitStopped(targets []stopTarget, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		running := false
		for _, t := range targets {
			if t.running() {
				running = true
				break
			}
		}
		if !running {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: tim <up|down> [project]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "exec":
		if len(os.Args) != 5 {
			fmt.Println("Usage: tim exec <pid-file> <shell> <script>")
			os.Exit(1)
		}
		startProcess(os.Args[2], os.Args[3], os.Args[4])
	case "up", "down":
		if len(os.Args) > 3 {
			fmt.Printf("Usage: tim %s [project]\n", os.Args[1])
			os.Exit(1)
		}
		projectName := ""
		if len(os.Args) == 3 {
			projectName = os.Args[2]
		}
		if os.Args[1] == "up" {
			up(projectName)
		} else {
			down(projectName)
		}
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
