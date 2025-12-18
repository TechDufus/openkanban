package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const defaultGlobalPrompt = `You have been spawned by OpenKanban to work on a ticket.

**Title:** {{.Title}}

**Description:**
{{.Description}}

**Branch:** {{.BranchName}} (from {{.BaseBranch}})

Focus on completing this ticket. Ask clarifying questions if the description is unclear.`

const defaultOpencodePrompt = `You have been spawned by OpenKanban, a kanban board system for managing development tasks.

## Your Assignment

You are now working on a specific ticket. This ticket represents a discrete unit of work that needs to be completed.

**Ticket Title:** {{.Title}}

**Ticket Description:**
{{.Description}}

## Technical Context

- **Git Branch:** {{.BranchName}}
- **Base Branch:** {{.BaseBranch}}
- **Working Directory:** This session is scoped to an isolated git worktree for this ticket

## Expectations

1. Focus exclusively on completing the work described in this ticket
2. The ticket description above is your primary specification - implement what it describes
3. If the description is unclear or incomplete, ask clarifying questions before proceeding
4. Make commits as appropriate for the work being done
5. When the work is complete, summarize what was accomplished

Begin by analyzing the ticket requirements and proposing your approach.`

const defaultClaudePrompt = `You have been spawned by OpenKanban, a kanban board system for managing development tasks.

## Your Assignment

You are now working on a specific ticket. This ticket represents a discrete unit of work that needs to be completed.

**Ticket Title:** {{.Title}}

**Ticket Description:**
{{.Description}}

## Technical Context

- **Git Branch:** {{.BranchName}}
- **Base Branch:** {{.BaseBranch}}
- **Working Directory:** This session is scoped to an isolated git worktree for this ticket

## Expectations

1. Focus exclusively on completing the work described in this ticket
2. The ticket description above is your primary specification - implement what it describes
3. If the description is unclear or incomplete, ask clarifying questions before proceeding
4. Make commits as appropriate for the work being done
5. When the work is complete, summarize what was accomplished

Begin by analyzing the ticket requirements and proposing your approach.`

const defaultAiderPrompt = `OpenKanban Ticket: {{.Title}}

Description:
{{.Description}}

Branch: {{.BranchName}} (from {{.BaseBranch}})

This is your assigned task. Implement what the description specifies.`

// Config holds the global application configuration
type Config struct {
	Defaults BoardSettings          `json:"defaults"`
	Agents   map[string]AgentConfig `json:"agents"`
	UI       UIConfig               `json:"ui"`
	Cleanup  CleanupSettings        `json:"cleanup"`
	Keys     map[string]string      `json:"keys,omitempty"`
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
	InitPrompt       string `json:"init_prompt"`
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

// CleanupSettings controls cleanup behavior when deleting tickets
type CleanupSettings struct {
	DeleteWorktree       bool `json:"delete_worktree"`        // Remove git worktree on ticket delete
	DeleteBranch         bool `json:"delete_branch"`          // Delete git branch after worktree removal
	ForceWorktreeRemoval bool `json:"force_worktree_removal"` // Force removal even with uncommitted changes
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
			InitPrompt:       defaultGlobalPrompt,
		},
		Agents: map[string]AgentConfig{
			"claude": {
				Command:    "claude",
				Args:       []string{"--dangerously-skip-permissions"},
				Env:        map[string]string{},
				StatusFile: ".claude/status.json",
				InitPrompt: defaultClaudePrompt,
			},
			"opencode": {
				Command:    "opencode",
				Args:       []string{},
				Env:        map[string]string{},
				StatusFile: ".opencode/status.json",
				InitPrompt: defaultOpencodePrompt,
			},
			"aider": {
				Command:    "aider",
				Args:       []string{"--yes"},
				Env:        map[string]string{},
				StatusFile: "",
				InitPrompt: defaultAiderPrompt,
			},
		},
		UI: UIConfig{
			Theme:           "catppuccin-mocha",
			ShowAgentStatus: true,
			RefreshInterval: 5,
			ColumnWidth:     40,
			TicketHeight:    4,
		},
		Cleanup: CleanupSettings{
			DeleteWorktree:       true,
			DeleteBranch:         false,
			ForceWorktreeRemoval: false,
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

	cfg.mergeAgentDefaults()

	return cfg, nil
}

func (c *Config) mergeAgentDefaults() {
	defaults := DefaultConfig()

	for name, defaultCfg := range defaults.Agents {
		if userCfg, exists := c.Agents[name]; exists {
			if userCfg.StatusFile == "" {
				userCfg.StatusFile = defaultCfg.StatusFile
			}
			if userCfg.Env == nil {
				userCfg.Env = defaultCfg.Env
			}
			c.Agents[name] = userCfg
		}
	}
}

func (c *Config) GetEffectiveInitPrompt(agentType string) string {
	if agentCfg, ok := c.Agents[agentType]; ok && agentCfg.InitPrompt != "" {
		return agentCfg.InitPrompt
	}
	if c.Defaults.InitPrompt != "" {
		return c.Defaults.InitPrompt
	}
	return defaultGlobalPrompt
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
