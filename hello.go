package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"
)

type Env map[string]string

type Config struct {
	Tabs []string `yaml:"tabs"`
	Env  Env      `yaml:"env"`
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

func openNewTab(command string, env Env) error {
	environmentPrefix := buildEnvPrefix(env)
	profileID := os.Getenv("WT_PROFILE_ID")
	commandToExecute := environmentPrefix + " && " + command

	cmd := exec.Command("wt.exe", "-w", "0", "nt", "--profile", profileID,
		"--", "wsl.exe", "--", "bash", "-c", commandToExecute)
	return cmd.Start()
}

func main() {
	config, err := loadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	if config == nil {
		fmt.Println("No .tim.yml found")
		os.Exit(1)
	}

	for _, tabCommand := range config.Tabs {
		fmt.Println(tabCommand)
		openNewTab(tabCommand, config.Env)
	}
}
