package config

import (
	"fmt"
	"os/exec"
	"strings"
	"text/template"
)

// ValidationError represents a single config validation issue
type ValidationError struct {
	Section string // "defaults", "agents.claude", "ui", etc.
	Field   string // "command", "branch_naming", etc.
	Message string // Human-readable error
	Value   any    // The invalid value (for display)
}

// ValidationResult holds all validation errors and warnings
type ValidationResult struct {
	Errors   []ValidationError
	Warnings []ValidationError
}

// HasErrors returns true if there are any validation errors
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if there are any validation warnings
func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// AddError adds a validation error
func (r *ValidationResult) AddError(section, field, message string, value any) {
	r.Errors = append(r.Errors, ValidationError{
		Section: section,
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// AddWarning adds a validation warning
func (r *ValidationResult) AddWarning(section, field, message string, value any) {
	r.Warnings = append(r.Warnings, ValidationError{
		Section: section,
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// FormatErrors returns a formatted string of all errors for CLI output
func (r *ValidationResult) FormatErrors() string {
	var sb strings.Builder
	for _, e := range r.Errors {
		if e.Field != "" {
			sb.WriteString(fmt.Sprintf("  [%s] %s\n", e.Section, e.Field))
		} else {
			sb.WriteString(fmt.Sprintf("  [%s]\n", e.Section))
		}
		sb.WriteString(fmt.Sprintf("    %s\n", e.Message))
		if e.Value != nil {
			sb.WriteString(fmt.Sprintf("    got: %v\n", e.Value))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// FormatWarnings returns a formatted string of all warnings for CLI output
func (r *ValidationResult) FormatWarnings() string {
	var sb strings.Builder
	for _, w := range r.Warnings {
		if w.Field != "" {
			sb.WriteString(fmt.Sprintf("  [%s] %s\n", w.Section, w.Field))
		} else {
			sb.WriteString(fmt.Sprintf("  [%s]\n", w.Section))
		}
		sb.WriteString(fmt.Sprintf("    %s\n", w.Message))
		if w.Value != nil {
			sb.WriteString(fmt.Sprintf("    got: %v\n", w.Value))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// Validate performs full config validation and returns all errors and warnings
func (c *Config) Validate() *ValidationResult {
	result := &ValidationResult{}
	c.validateDefaults(result)
	c.validateAgents(result)
	c.validateUI(result)
	c.validateOpencode(result)
	return result
}

// validateDefaults validates the defaults section
func (c *Config) validateDefaults(r *ValidationResult) {
	// BranchNaming must be a valid enum value
	validNaming := map[string]bool{"template": true, "ai": true, "prompt": true, "": true}
	if !validNaming[c.Defaults.BranchNaming] {
		r.AddError("defaults", "branch_naming",
			fmt.Sprintf("must be one of: template, ai, prompt (got %q)", c.Defaults.BranchNaming),
			c.Defaults.BranchNaming)
	}

	// SlugMaxLength must be positive if set
	if c.Defaults.SlugMaxLength < 0 {
		r.AddError("defaults", "slug_max_length",
			"must be a positive number",
			c.Defaults.SlugMaxLength)
	}

	// DefaultAgent must reference a defined agent (if set)
	if c.Defaults.DefaultAgent != "" {
		if _, exists := c.Agents[c.Defaults.DefaultAgent]; !exists {
			r.AddError("defaults", "default_agent",
				fmt.Sprintf("references undefined agent %q", c.Defaults.DefaultAgent),
				c.Defaults.DefaultAgent)
		}
	}

	// BranchTemplate should contain placeholders (warning only)
	if c.Defaults.BranchTemplate != "" {
		if !strings.Contains(c.Defaults.BranchTemplate, "{slug}") &&
			!strings.Contains(c.Defaults.BranchTemplate, "{prefix}") {
			r.AddWarning("defaults", "branch_template",
				"should contain {slug} or {prefix} placeholder",
				c.Defaults.BranchTemplate)
		}
	}

	// Validate InitPrompt template syntax
	if c.Defaults.InitPrompt != "" {
		if err := validateTemplate(c.Defaults.InitPrompt); err != nil {
			r.AddError("defaults", "init_prompt",
				fmt.Sprintf("invalid Go template syntax: %v", err),
				nil)
		}
	}
}

func (c *Config) validateAgents(r *ValidationResult) {
	for name, agent := range c.Agents {
		section := fmt.Sprintf("agents.%s", name)

		if agent.Command == "" {
			r.AddError(section, "command", "is required but missing", nil)
		} else if name == c.Defaults.DefaultAgent {
			if _, err := exec.LookPath(agent.Command); err != nil {
				r.AddWarning(section, "command",
					fmt.Sprintf("executable %q not found in PATH", agent.Command),
					agent.Command)
			}
		}

		if agent.InitPrompt != "" {
			if err := validateTemplate(agent.InitPrompt); err != nil {
				r.AddError(section, "init_prompt",
					fmt.Sprintf("invalid Go template syntax: %v", err),
					nil)
			}
		}
	}
}

// validateUI validates the UI section
func (c *Config) validateUI(r *ValidationResult) {
	if c.UI.Theme != "" && !IsValidTheme(c.UI.Theme) {
		r.AddWarning("ui", "theme",
			fmt.Sprintf("unknown theme %q, falling back to catppuccin-mocha. Available: %v",
				c.UI.Theme, ThemeNames()),
			c.UI.Theme)
	}

	if c.UI.ColumnWidth <= 0 {
		r.AddError("ui", "column_width",
			"must be a positive number",
			c.UI.ColumnWidth)
	}

	if c.UI.TicketHeight <= 0 {
		r.AddError("ui", "ticket_height",
			"must be a positive number",
			c.UI.TicketHeight)
	}

	if c.UI.RefreshInterval <= 0 {
		r.AddError("ui", "refresh_interval",
			"must be a positive number",
			c.UI.RefreshInterval)
	}
}

// validateOpencode validates the opencode server settings
func (c *Config) validateOpencode(r *ValidationResult) {
	if c.Opencode.ServerPort < 0 || c.Opencode.ServerPort > 65535 {
		r.AddError("opencode", "server_port",
			"must be between 1 and 65535",
			c.Opencode.ServerPort)
	}

	if c.Opencode.PollInterval < 0 {
		r.AddError("opencode", "poll_interval",
			"must be a positive number",
			c.Opencode.PollInterval)
	}
}

// validateTemplate checks if a string is a valid Go template
func validateTemplate(tmpl string) error {
	_, err := template.New("check").Parse(tmpl)
	return err
}
