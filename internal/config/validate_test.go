package config

import (
	"strings"
	"testing"
)

func TestValidate_ValidDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	result := cfg.Validate()

	if result.HasErrors() {
		t.Errorf("default config should be valid, got errors:\n%s", result.FormatErrors())
	}
}

func TestValidate_InvalidBranchNaming(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Defaults.BranchNaming = "invalid"

	result := cfg.Validate()

	if !result.HasErrors() {
		t.Error("expected validation error for invalid branch_naming")
	}

	found := false
	for _, e := range result.Errors {
		if e.Section == "defaults" && e.Field == "branch_naming" {
			found = true
			if !strings.Contains(e.Message, "template, ai, prompt") {
				t.Errorf("error message should list valid values; got %q", e.Message)
			}
		}
	}
	if !found {
		t.Error("expected error for defaults.branch_naming")
	}
}

func TestValidate_NegativeSlugMaxLength(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Defaults.SlugMaxLength = -1

	result := cfg.Validate()

	if !result.HasErrors() {
		t.Error("expected validation error for negative slug_max_length")
	}

	found := false
	for _, e := range result.Errors {
		if e.Section == "defaults" && e.Field == "slug_max_length" {
			found = true
		}
	}
	if !found {
		t.Error("expected error for defaults.slug_max_length")
	}
}

func TestValidate_NonexistentDefaultAgent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Defaults.DefaultAgent = "nonexistent-agent"

	result := cfg.Validate()

	if !result.HasErrors() {
		t.Error("expected validation error for nonexistent default_agent")
	}

	found := false
	for _, e := range result.Errors {
		if e.Section == "defaults" && e.Field == "default_agent" {
			found = true
			if !strings.Contains(e.Message, "nonexistent-agent") {
				t.Errorf("error message should mention the agent name; got %q", e.Message)
			}
		}
	}
	if !found {
		t.Error("expected error for defaults.default_agent")
	}
}

func TestValidate_BranchTemplateMissingPlaceholders(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Defaults.BranchTemplate = "feature-branch"

	result := cfg.Validate()

	// This should be a warning, not an error
	if result.HasErrors() {
		t.Errorf("missing placeholders should be a warning, not error:\n%s", result.FormatErrors())
	}

	if !result.HasWarnings() {
		t.Error("expected warning for branch_template without placeholders")
	}

	found := false
	for _, w := range result.Warnings {
		if w.Section == "defaults" && w.Field == "branch_template" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for defaults.branch_template")
	}
}

func TestValidate_MissingAgentCommand(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Agents["custom"] = AgentConfig{
		Command: "",
		Args:    []string{},
	}

	result := cfg.Validate()

	if !result.HasErrors() {
		t.Error("expected validation error for missing agent command")
	}

	found := false
	for _, e := range result.Errors {
		if e.Section == "agents.custom" && e.Field == "command" {
			found = true
			if !strings.Contains(e.Message, "required") {
				t.Errorf("error message should mention 'required'; got %q", e.Message)
			}
		}
	}
	if !found {
		t.Error("expected error for agents.custom.command")
	}
}

func TestValidate_CommandNotInPath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Agents["custom"] = AgentConfig{
		Command: "nonexistent-binary-12345",
		Args:    []string{},
	}

	result := cfg.Validate()

	// This should be a warning, not an error
	hasCommandWarning := false
	for _, w := range result.Warnings {
		if w.Section == "agents.custom" && w.Field == "command" {
			hasCommandWarning = true
			if !strings.Contains(w.Message, "not found in PATH") {
				t.Errorf("warning should mention PATH; got %q", w.Message)
			}
		}
	}
	if !hasCommandWarning {
		t.Error("expected warning for command not in PATH")
	}
}

func TestValidate_InvalidTemplatePrompt(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Agents["custom"] = AgentConfig{
		Command:    "echo",
		InitPrompt: "{{.Invalid syntax",
	}

	result := cfg.Validate()

	if !result.HasErrors() {
		t.Error("expected validation error for invalid template syntax")
	}

	found := false
	for _, e := range result.Errors {
		if e.Section == "agents.custom" && e.Field == "init_prompt" {
			found = true
		}
	}
	if !found {
		t.Error("expected error for agents.custom.init_prompt")
	}
}

func TestValidate_InvalidDefaultsInitPrompt(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Defaults.InitPrompt = "{{.Broken"

	result := cfg.Validate()

	if !result.HasErrors() {
		t.Error("expected validation error for invalid defaults.init_prompt")
	}

	found := false
	for _, e := range result.Errors {
		if e.Section == "defaults" && e.Field == "init_prompt" {
			found = true
		}
	}
	if !found {
		t.Error("expected error for defaults.init_prompt")
	}
}

func TestValidate_ZeroUIColumnWidth(t *testing.T) {
	cfg := DefaultConfig()
	cfg.UI.ColumnWidth = 0

	result := cfg.Validate()

	if !result.HasErrors() {
		t.Error("expected validation error for zero column_width")
	}

	found := false
	for _, e := range result.Errors {
		if e.Section == "ui" && e.Field == "column_width" {
			found = true
		}
	}
	if !found {
		t.Error("expected error for ui.column_width")
	}
}

func TestValidate_ZeroUITicketHeight(t *testing.T) {
	cfg := DefaultConfig()
	cfg.UI.TicketHeight = 0

	result := cfg.Validate()

	if !result.HasErrors() {
		t.Error("expected validation error for zero ticket_height")
	}

	found := false
	for _, e := range result.Errors {
		if e.Section == "ui" && e.Field == "ticket_height" {
			found = true
		}
	}
	if !found {
		t.Error("expected error for ui.ticket_height")
	}
}

func TestValidate_ZeroUIRefreshInterval(t *testing.T) {
	cfg := DefaultConfig()
	cfg.UI.RefreshInterval = 0

	result := cfg.Validate()

	if !result.HasErrors() {
		t.Error("expected validation error for zero refresh_interval")
	}

	found := false
	for _, e := range result.Errors {
		if e.Section == "ui" && e.Field == "refresh_interval" {
			found = true
		}
	}
	if !found {
		t.Error("expected error for ui.refresh_interval")
	}
}

func TestValidate_InvalidServerPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"negative port", -1},
		{"port too high", 70000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Opencode.ServerPort = tt.port

			result := cfg.Validate()

			if !result.HasErrors() {
				t.Error("expected validation error for invalid server_port")
			}

			found := false
			for _, e := range result.Errors {
				if e.Section == "opencode" && e.Field == "server_port" {
					found = true
				}
			}
			if !found {
				t.Errorf("expected error for opencode.server_port with value %d", tt.port)
			}
		})
	}
}

func TestValidate_NegativePollInterval(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Opencode.PollInterval = -1

	result := cfg.Validate()

	if !result.HasErrors() {
		t.Error("expected validation error for negative poll_interval")
	}

	found := false
	for _, e := range result.Errors {
		if e.Section == "opencode" && e.Field == "poll_interval" {
			found = true
		}
	}
	if !found {
		t.Error("expected error for opencode.poll_interval")
	}
}

func TestValidationResult_FormatErrors(t *testing.T) {
	r := &ValidationResult{}
	r.AddError("defaults", "branch_naming", "must be valid", "invalid")
	r.AddError("agents.custom", "command", "is required", nil)

	output := r.FormatErrors()

	if !strings.Contains(output, "defaults") {
		t.Error("formatted errors should contain section name")
	}
	if !strings.Contains(output, "branch_naming") {
		t.Error("formatted errors should contain field name")
	}
	if !strings.Contains(output, "must be valid") {
		t.Error("formatted errors should contain message")
	}
	if !strings.Contains(output, "invalid") {
		t.Error("formatted errors should contain value")
	}
	if !strings.Contains(output, "agents.custom") {
		t.Error("formatted errors should contain nested section")
	}
}

func TestValidationResult_FormatWarnings(t *testing.T) {
	r := &ValidationResult{}
	r.AddWarning("agents.custom", "command", "not found in PATH", "custom-agent")

	output := r.FormatWarnings()

	if !strings.Contains(output, "agents.custom") {
		t.Error("formatted warnings should contain section name")
	}
	if !strings.Contains(output, "command") {
		t.Error("formatted warnings should contain field name")
	}
	if !strings.Contains(output, "not found in PATH") {
		t.Error("formatted warnings should contain message")
	}
}

func TestValidationResult_HasErrors(t *testing.T) {
	r := &ValidationResult{}

	if r.HasErrors() {
		t.Error("empty result should not have errors")
	}

	r.AddError("test", "field", "message", nil)

	if !r.HasErrors() {
		t.Error("result with error should have errors")
	}
}

func TestValidationResult_HasWarnings(t *testing.T) {
	r := &ValidationResult{}

	if r.HasWarnings() {
		t.Error("empty result should not have warnings")
	}

	r.AddWarning("test", "field", "message", nil)

	if !r.HasWarnings() {
		t.Error("result with warning should have warnings")
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := &Config{
		Defaults: BoardSettings{
			DefaultAgent:   "nonexistent",
			BranchNaming:   "invalid",
			SlugMaxLength:  -1,
			BranchTemplate: "no-placeholders",
		},
		Agents: map[string]AgentConfig{
			"bad": {Command: ""},
		},
		UI: UIConfig{
			ColumnWidth:     0,
			TicketHeight:    0,
			RefreshInterval: 0,
		},
		Opencode: OpencodeSettings{
			ServerPort:   -1,
			PollInterval: -1,
		},
	}

	result := cfg.Validate()

	// Should have multiple errors
	if len(result.Errors) < 5 {
		t.Errorf("expected at least 5 errors, got %d:\n%s", len(result.Errors), result.FormatErrors())
	}

	// Should have at least one warning (branch_template)
	if len(result.Warnings) < 1 {
		t.Error("expected at least 1 warning for branch_template")
	}
}

func TestValidate_EmptyBranchNamingIsValid(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Defaults.BranchNaming = ""

	result := cfg.Validate()

	for _, e := range result.Errors {
		if e.Field == "branch_naming" {
			t.Error("empty branch_naming should be valid (uses default)")
		}
	}
}

func TestValidate_ValidTemplatePrompt(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Agents["custom"] = AgentConfig{
		Command:    "echo",
		InitPrompt: "Working on: {{.Title}}\nDescription: {{.Description}}",
	}

	result := cfg.Validate()

	for _, e := range result.Errors {
		if e.Section == "agents.custom" && e.Field == "init_prompt" {
			t.Errorf("valid template should not produce error: %s", e.Message)
		}
	}
}
