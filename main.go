package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"syscall"
)

type Env map[string]string

type Commands []string

func (c *Commands) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		*c = Commands{value.Value}
		return nil
	case yaml.SequenceNode:
		var list []string
		if err := value.Decode(&list); err != nil {
			return err
		}
		*c = list
		return nil
	default:
		return fmt.Errorf("expected string or list of strings, got %v", value.Kind)
	}
}

type Config struct {
	Terminal string              `yaml:"terminal"`
	Tabs     map[string]Commands `yaml:"tabs"`
	Env      Env                 `yaml:"env"`
}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile(".tim.yml")
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func up() {

	config, err := loadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	if config == nil {
		fmt.Println("No .tim.yml found")
		os.Exit(1)
	}

	term, err := DetectTerminal(config.Terminal)
	if err != nil {
		fmt.Println("Error detecting terminal:", err)
		os.Exit(1)
	}

	for title, commands := range config.Tabs {
		pidFile := fmt.Sprintf(".tim/%s.pid", title)
		if _, err := os.Stat(pidFile); err == nil {
			fmt.Printf("Skipping %q: already running (pid file exists)\n", title)
			continue
		}
		if err := term.OpenTab(title, commands, config.Env); err != nil {
			fmt.Printf("Error opening tab %q: %v\n", title, err)
		}
	}

	if t, ok := term.(*TmuxTerminal); ok {
		if err := t.Attach(); err != nil {
			fmt.Printf("Error attaching to tmux session: %v\n", err)
		}
	}
}

func down() {
	pidFiles, err := filepath.Glob(".tim/*.pid")
	if err != nil || len(pidFiles) == 0 {
		fmt.Println("No running processes found")
		return
	}

	for _, pidFile := range pidFiles {
		data, err := os.ReadFile(pidFile)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", pidFile, err)
			continue
		}

		var pid int
		fmt.Sscanf(string(data), "%d", &pid)

		process, err := os.FindProcess(pid)
		if err != nil {
			fmt.Printf("Error finding process %d: %v\n", pid, err)
			os.Remove(pidFile)
			continue
		}

		if err := process.Signal(syscall.SIGTERM); err != nil {
			fmt.Printf("Error sending SIGTERM to %d: %v\n", pid, err)
			os.Remove(pidFile)
			continue
		}

		fmt.Printf("Sent SIGTERM to %d (%s)\n", pid, pidFile)
		os.Remove(pidFile)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: tim <exec|up>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "exec":
		startProcess(os.Args[2], os.Args[3:])
	case "up":
		up()
	case "down":
		down()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
