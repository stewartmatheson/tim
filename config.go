package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
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

// Tab is one entry under "tabs". In YAML it is either a command string, a
// list of commands, or a mapping with "commands" and an optional
// "working_dir".
type Tab struct {
	WorkingDir string   `yaml:"working_dir"`
	Commands   Commands `yaml:"commands"`
}

func (t *Tab) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return value.Decode(&t.Commands)
	}
	type plain Tab // drops this method so Decode doesn't recurse
	return value.Decode((*plain)(t))
}

// Project is one set of tabs that tim starts and stops together.
type Project struct {
	Terminal   string         `yaml:"terminal"`
	WorkingDir string         `yaml:"working_dir"`
	Tabs       map[string]Tab `yaml:"tabs"`
	Env        Env            `yaml:"env"`
}

// Config is the contents of .tim.yml. It either describes a single project
// at the top level, or several named projects under "projects".
type Config struct {
	Project  `yaml:",inline"`
	Projects map[string]Project `yaml:"projects"`
}

// ResolvedProject is a Project with its paths worked out relative to the
// .tim.yml it came from.
type ResolvedProject struct {
	Name     string // empty for a single top-level project
	Terminal string
	Env      Env
	Dir      string // absolute project working directory (tabs default to it)
	PidDir   string // absolute directory holding the PID files
	Tabs     map[string]ResolvedTab
}

type ResolvedTab struct {
	Dir      string // absolute working directory for this tab
	Commands Commands
}

func (p *ResolvedProject) pidFile(tab string) string {
	return filepath.Join(p.PidDir, tab+".pid")
}

// sessionName names the tmux session: the project name when there is one,
// otherwise the working directory's base name.
func (p *ResolvedProject) sessionName() string {
	if p.Name != "" {
		return p.Name
	}
	return filepath.Base(p.Dir)
}

// findConfig searches for .tim.yml starting in the current directory and
// walking up through parent directories, stopping at the user's home
// directory (or the filesystem root if the current directory is not under home).
func findConfig() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	home, _ := os.UserHomeDir()

	for {
		path := filepath.Join(dir, ".tim.yml")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}

		parent := filepath.Dir(dir)
		if dir == home || parent == dir {
			return "", fmt.Errorf("no .tim.yml file found")
		}
		dir = parent
	}
}

func loadConfig() (*Config, string, error) {
	path, err := findConfig()
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, "", fmt.Errorf("%s: %w", path, err)
	}
	return &config, path, nil
}

// loadProject finds .tim.yml and picks the project to work on. name may be
// empty when the file has a single top-level project or exactly one entry
// under "projects".
func loadProject(name string) (*ResolvedProject, error) {
	config, path, err := loadConfig()
	if err != nil {
		return nil, err
	}
	configDir := filepath.Dir(path)

	var project Project
	timDir := filepath.Join(configDir, ".tim")
	pidDir := timDir

	if len(config.Projects) == 0 {
		if name != "" {
			return nil, fmt.Errorf("%s has no projects; run without a project name", path)
		}
		project = config.Project
	} else {
		if len(config.Tabs) > 0 {
			return nil, fmt.Errorf("%s: put tabs inside a project under \"projects\", not at the top level", path)
		}
		names := make([]string, 0, len(config.Projects))
		for n := range config.Projects {
			names = append(names, n)
		}
		sort.Strings(names)

		if name == "" {
			if len(names) > 1 {
				return nil, fmt.Errorf("%s has several projects, choose one: %s", path, strings.Join(names, ", "))
			}
			name = names[0]
		}
		p, ok := config.Projects[name]
		if !ok {
			return nil, fmt.Errorf("no project %q in %s; available: %s", name, path, strings.Join(names, ", "))
		}
		project = p
		pidDir = filepath.Join(timDir, name)
	}

	if len(project.Tabs) == 0 {
		return nil, fmt.Errorf("%s: no tabs defined", path)
	}

	dir, err := resolveWorkingDir(configDir, project.WorkingDir)
	if err != nil {
		return nil, err
	}

	tabs := make(map[string]ResolvedTab, len(project.Tabs))
	for title, tab := range project.Tabs {
		if len(tab.Commands) == 0 {
			return nil, fmt.Errorf("tab %q: no commands", title)
		}
		tabDir, err := resolveWorkingDir(dir, tab.WorkingDir)
		if err != nil {
			return nil, fmt.Errorf("tab %q: %w", title, err)
		}
		tabs[title] = ResolvedTab{Dir: tabDir, Commands: tab.Commands}
	}

	return &ResolvedProject{
		Name:     name,
		Terminal: project.Terminal,
		Env:      project.Env,
		Dir:      dir,
		PidDir:   pidDir,
		Tabs:     tabs,
	}, nil
}

// resolveWorkingDir turns working_dir into an absolute directory. Relative
// paths are taken from baseDir, which is also the default: the .tim.yml's
// directory for a project, and the project's directory for a tab.
func resolveWorkingDir(baseDir, workingDir string) (string, error) {
	dir := workingDir
	if dir == "~" || strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("working_dir %q: %w", workingDir, err)
		}
		dir = filepath.Join(home, dir[1:])
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(baseDir, dir)
	}
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("working_dir %q: %w", workingDir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("working_dir %q is not a directory", workingDir)
	}
	return dir, nil
}
