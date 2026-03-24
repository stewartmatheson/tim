package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"gopkg.in/yaml.v3"
)

type Env map[string]string

type Config struct {
	Tabs map[string]string `yaml:"tabs"`
	Env  Env               `yaml:"env"`
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

func buildEnvPrefix(env Env) string {
	parts := make([]string, 0, len(env))
	for k, v := range env {
		parts = append(parts, fmt.Sprintf("export %s=%s", k, v))
	}
	return strings.Join(parts, " && ")
}

func openNewTab(title string, command string, env Env) error {
	command = os.Expand(command, func(key string) string {
		return env[key]
	})
	environmentPrefix := buildEnvPrefix(env)
	profileID := os.Getenv("WT_PROFILE_ID")

	preamble := "echo 'Executing: " + command + "'"
	divider := "echo '-------------------------------------'"

	// returnToExit := "echo 'Press any key to exit' && read"
	commandsToAppend := []string{}
	for _, part := range []string{environmentPrefix, preamble, divider, command} {
		if part != "" {
			commandsToAppend = append(commandsToAppend, part)
		}
	}

	timBinaryLocation, err := exec.LookPath("tim")
	if err != nil {
		fmt.Println("Error: Can't find tim in path", err)
		os.Exit(1)
	}

	commandToExecute := strings.Join(commandsToAppend, " && ")
	cmd := exec.Command(
		"wt.exe", "-w", "0", "nt", "--title", title,
		"--profile", profileID,
		"--", "wsl.exe", "--",
		timBinaryLocation, "exec", title, "bash", "-c", commandToExecute,
	)

	fmt.Println(cmd)

	return cmd.Start()
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

	for title, tabCommand := range config.Tabs {
		openNewTab(title, tabCommand, config.Env)
	}
}

func execCommand(processName string, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: tim exec <command> [args...]")
		os.Exit(1)
	}

	config, err := loadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	env := os.Environ()
	if config != nil {
		for k, v := range config.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	binary, err := exec.LookPath(args[0])
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	os.MkdirAll(".tim", 0755)
	pidFileName := fmt.Sprintf("./.tim/%s.pid", processName)
	os.WriteFile(pidFileName, fmt.Appendf(nil, "%d", os.Getpid()), 0644)

	if err := syscall.Exec(binary, args, env); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: tim <exec|up>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "exec":
		execCommand(os.Args[2], os.Args[3:])
	case "up":
		up()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
