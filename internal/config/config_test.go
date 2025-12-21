package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Defaults.DefaultAgent != "opencode" {
		t.Errorf("Defaults.DefaultAgent = %q; want %q", cfg.Defaults.DefaultAgent, "opencode")
	}

	if cfg.Defaults.BranchPrefix != "task/" {
		t.Errorf("Defaults.BranchPrefix = %q; want %q", cfg.Defaults.BranchPrefix, "task/")
	}

	if cfg.Defaults.BranchNaming != "template" {
		t.Errorf("Defaults.BranchNaming = %q; want %q", cfg.Defaults.BranchNaming, "template")
	}

	if cfg.Defaults.BranchTemplate != "{prefix}{slug}" {
		t.Errorf("Defaults.BranchTemplate = %q; want %q", cfg.Defaults.BranchTemplate, "{prefix}{slug}")
	}

	if cfg.Defaults.SlugMaxLength != 40 {
		t.Errorf("Defaults.SlugMaxLength = %d; want %d", cfg.Defaults.SlugMaxLength, 40)
	}

	if !cfg.Defaults.AutoSpawnAgent {
		t.Error("Defaults.AutoSpawnAgent should be true")
	}

	if !cfg.Defaults.AutoCreateBranch {
		t.Error("Defaults.AutoCreateBranch should be true")
	}

	for _, agent := range []string{"claude", "opencode", "aider"} {
		if _, ok := cfg.Agents[agent]; !ok {
			t.Errorf("expected agent %q to be defined", agent)
		}
	}

	claude := cfg.Agents["claude"]
	if claude.Command != "claude" {
		t.Errorf("claude.Command = %q; want %q", claude.Command, "claude")
	}

	opencode := cfg.Agents["opencode"]
	if opencode.Command != "opencode" {
		t.Errorf("opencode.Command = %q; want %q", opencode.Command, "opencode")
	}

	aider := cfg.Agents["aider"]
	if aider.Command != "aider" {
		t.Errorf("aider.Command = %q; want %q", aider.Command, "aider")
	}
	if len(aider.Args) != 1 || aider.Args[0] != "--yes" {
		t.Errorf("aider.Args = %v; want [--yes]", aider.Args)
	}

	if cfg.UI.Theme != "catppuccin-mocha" {
		t.Errorf("UI.Theme = %q; want %q", cfg.UI.Theme, "catppuccin-mocha")
	}

	if !cfg.UI.ShowAgentStatus {
		t.Error("UI.ShowAgentStatus should be true")
	}

	if cfg.UI.RefreshInterval != 5 {
		t.Errorf("UI.RefreshInterval = %d; want %d", cfg.UI.RefreshInterval, 5)
	}

	if cfg.UI.ColumnWidth != 40 {
		t.Errorf("UI.ColumnWidth = %d; want %d", cfg.UI.ColumnWidth, 40)
	}

	if !cfg.Cleanup.DeleteWorktree {
		t.Error("Cleanup.DeleteWorktree should be true")
	}

	if cfg.Cleanup.DeleteBranch {
		t.Error("Cleanup.DeleteBranch should be false")
	}

	if cfg.Cleanup.ForceWorktreeRemoval {
		t.Error("Cleanup.ForceWorktreeRemoval should be false")
	}
}

func TestConfigDir(t *testing.T) {
	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}

	if dir == "" {
		t.Error("ConfigDir() returned empty string")
	}

	if filepath.Base(dir) != "openkanban" {
		t.Errorf("ConfigDir() = %q; want to end with 'openkanban'", dir)
	}

	if filepath.Base(filepath.Dir(dir)) != ".config" {
		t.Errorf("ConfigDir() = %q; want parent to be '.config'", dir)
	}
}

func TestConfigPath(t *testing.T) {
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error: %v", err)
	}

	if filepath.Base(path) != "config.json" {
		t.Errorf("ConfigPath() = %q; want to end with 'config.json'", path)
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	defaults := DefaultConfig()
	if cfg.Defaults.DefaultAgent != defaults.Defaults.DefaultAgent {
		t.Errorf("Load() should return defaults when file not found")
	}
}

func TestLoad_EmptyPath(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\") error: %v", err)
	}

	if cfg == nil {
		t.Error("Load(\"\") should not return nil config")
	}
}

func TestLoad_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	customConfig := map[string]interface{}{
		"defaults": map[string]interface{}{
			"default_agent":   "claude",
			"branch_prefix":   "feature/",
			"slug_max_length": 30,
		},
		"ui": map[string]interface{}{
			"theme": "dark",
		},
	}

	data, err := json.Marshal(customConfig)
	if err != nil {
		t.Fatalf("failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Defaults.DefaultAgent != "claude" {
		t.Errorf("Defaults.DefaultAgent = %q; want %q", cfg.Defaults.DefaultAgent, "claude")
	}

	if cfg.Defaults.BranchPrefix != "feature/" {
		t.Errorf("Defaults.BranchPrefix = %q; want %q", cfg.Defaults.BranchPrefix, "feature/")
	}

	if cfg.Defaults.SlugMaxLength != 30 {
		t.Errorf("Defaults.SlugMaxLength = %d; want %d", cfg.Defaults.SlugMaxLength, 30)
	}

	if cfg.UI.Theme != "dark" {
		t.Errorf("UI.Theme = %q; want %q", cfg.UI.Theme, "dark")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	if err := os.WriteFile(configPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()
	cfg.Defaults.DefaultAgent = "custom-agent"

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Save() should create config file")
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.Defaults.DefaultAgent != "custom-agent" {
		t.Errorf("loaded.Defaults.DefaultAgent = %q; want %q", loaded.Defaults.DefaultAgent, "custom-agent")
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested", "dir", "config.json")

	cfg := DefaultConfig()

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Save() should create nested directories")
	}
}

func TestGetEffectiveInitPrompt(t *testing.T) {
	t.Run("returns agent-specific prompt when set", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Agents["claude"] = AgentConfig{
			Command:    "claude",
			InitPrompt: "custom claude prompt",
		}

		prompt := cfg.GetEffectiveInitPrompt("claude")
		if prompt != "custom claude prompt" {
			t.Errorf("GetEffectiveInitPrompt(\"claude\") = %q; want %q", prompt, "custom claude prompt")
		}
	})

	t.Run("falls back to global default when agent has no prompt", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Agents["custom"] = AgentConfig{
			Command:    "custom",
			InitPrompt: "",
		}
		cfg.Defaults.InitPrompt = "global prompt"

		prompt := cfg.GetEffectiveInitPrompt("custom")
		if prompt != "global prompt" {
			t.Errorf("GetEffectiveInitPrompt(\"custom\") = %q; want %q", prompt, "global prompt")
		}
	})

	t.Run("falls back to hardcoded default when no prompts set", func(t *testing.T) {
		cfg := &Config{
			Agents:   map[string]AgentConfig{},
			Defaults: BoardSettings{},
		}

		prompt := cfg.GetEffectiveInitPrompt("unknown")
		if prompt == "" {
			t.Error("GetEffectiveInitPrompt should return non-empty default prompt")
		}
	})

	t.Run("returns default for unknown agent", func(t *testing.T) {
		cfg := DefaultConfig()

		prompt := cfg.GetEffectiveInitPrompt("unknown-agent")
		if prompt == "" {
			t.Error("GetEffectiveInitPrompt should return non-empty default for unknown agent")
		}
	})
}

func TestMergeAgentDefaults(t *testing.T) {
	cfg := &Config{
		Agents: map[string]AgentConfig{
			"claude": {
				Command: "custom-claude",
				Args:    []string{"--custom"},
			},
		},
	}

	cfg.mergeAgentDefaults()

	if cfg.Agents["claude"].StatusFile != ".claude/status.json" {
		t.Errorf("claude.StatusFile = %q; want %q", cfg.Agents["claude"].StatusFile, ".claude/status.json")
	}

	if cfg.Agents["claude"].Env == nil {
		t.Error("claude.Env should not be nil after merge")
	}

	if cfg.Agents["claude"].Command != "custom-claude" {
		t.Errorf("claude.Command = %q; want %q", cfg.Agents["claude"].Command, "custom-claude")
	}
}

func TestConfigStructure(t *testing.T) {
	cfg := DefaultConfig()

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var unmarshaled Config
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if unmarshaled.Defaults.DefaultAgent != cfg.Defaults.DefaultAgent {
		t.Errorf("round-trip failed for Defaults.DefaultAgent")
	}

	if len(unmarshaled.Agents) != len(cfg.Agents) {
		t.Errorf("round-trip failed for Agents count")
	}
}

func TestAgentConfigFields(t *testing.T) {
	cfg := DefaultConfig()

	for name, agent := range cfg.Agents {
		if agent.Command == "" {
			t.Errorf("agent %q has empty Command", name)
		}
		if agent.Env == nil {
			t.Errorf("agent %q has nil Env", name)
		}
	}
}

func TestUIConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.UI.ColumnWidth <= 0 {
		t.Errorf("UI.ColumnWidth = %d; want positive value", cfg.UI.ColumnWidth)
	}

	if cfg.UI.TicketHeight <= 0 {
		t.Errorf("UI.TicketHeight = %d; want positive value", cfg.UI.TicketHeight)
	}

	if cfg.UI.RefreshInterval <= 0 {
		t.Errorf("UI.RefreshInterval = %d; want positive value", cfg.UI.RefreshInterval)
	}
}
