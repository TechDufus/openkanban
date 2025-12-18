package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the global application configuration
type Config struct {
	// Default settings for new boards
	Defaults BoardSettings `json:"defaults"`

	// Agent configurations by name
	Agents map[string]AgentConfig `json:"agents"`

	// UI preferences
	UI UIConfig `json:"ui"`

	// Custom keybindings
	Keys map[string]string `json:"keys,omitempty"`
}

// BoardSettings contains default settings for boards
type BoardSettings struct {
	DefaultAgent     string `json:"default_agent"`
	WorktreeBase     string `json:"worktree_base"`
	AutoSpawnAgent   bool   `json:"auto_spawn_agent"`
	AutoCreateBranch bool   `json:"auto_create_branch"`
	BranchPrefix     string `json:"branch_prefix"`
	BranchNaming     string `json:"branch_naming"`   // "template" | "ai" | "prompt"
	BranchTemplate   string `json:"branch_template"` // e.g., "{prefix}{slug}"
	SlugMaxLength    int    `json:"slug_max_length"` // default: 40
}

// AgentConfig defines how to spawn and monitor an AI agent
type AgentConfig struct {
	Command    string            `json:"command"`
	Args       []string          `json:"args"`
	Env        map[string]string `json:"env"`
	StatusFile string            `json:"status_file"`
	InitPrompt string            `json:"init_prompt"`
}

// UIConfig holds UI-related preferences
type UIConfig struct {
	Theme           string `json:"theme"`
	ShowAgentStatus bool   `json:"show_agent_status"`
	RefreshInterval int    `json:"refresh_interval"`
	ColumnWidth     int    `json:"column_width"`
	TicketHeight    int    `json:"ticket_height"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Defaults: BoardSettings{
			DefaultAgent:     "opencode",
			WorktreeBase:     "",
			AutoSpawnAgent:   true,
			AutoCreateBranch: true,
			BranchPrefix:     "task/",
			BranchNaming:     "template",
			BranchTemplate:   "{prefix}{slug}",
			SlugMaxLength:    40,
		},
		Agents: map[string]AgentConfig{
			"claude": {
				Command:    "claude",
				Args:       []string{"--dangerously-skip-permissions"},
				Env:        map[string]string{},
				StatusFile: ".claude/status.json",
				InitPrompt: "You are working on: {{.Title}}\n\nDescription:\n{{.Description}}\n\nBranch: {{.BranchName}}\nBase: {{.BaseBranch}}",
			},
			"opencode": {
				Command:    "opencode",
				Args:       []string{},
				Env:        map[string]string{},
				StatusFile: ".opencode/status.json",
				InitPrompt: "Task: {{.Title}}\n\n{{.Description}}",
			},
			"aider": {
				Command:    "aider",
				Args:       []string{"--yes"},
				Env:        map[string]string{},
				StatusFile: "",
				InitPrompt: "",
			},
		},
		UI: UIConfig{
			Theme:           "catppuccin-mocha",
			ShowAgentStatus: true,
			RefreshInterval: 5,
			ColumnWidth:     40,
			TicketHeight:    4,
		},
	}
}

// ConfigDir returns the configuration directory path
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "openkanban"), nil
}

// ConfigPath returns the default config file path
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads configuration from file or returns defaults
func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = ConfigPath()
		if err != nil {
			return DefaultConfig(), nil
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes configuration to file
func (c *Config) Save(path string) error {
	if path == "" {
		var err error
		path, err = ConfigPath()
		if err != nil {
			return err
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
