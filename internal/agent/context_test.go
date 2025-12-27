package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/techdufus/openkanban/internal/board"
)

func TestBuildContextPrompt(t *testing.T) {
	tests := []struct {
		name           string
		template       string
		ticket         *board.Ticket
		expectContains []string
		expectEmpty    bool
	}{
		{
			name:        "empty template returns empty",
			template:    "",
			ticket:      &board.Ticket{Title: "Test"},
			expectEmpty: true,
		},
		{
			name:           "simple title substitution",
			template:       "Work on: {{.Title}}",
			ticket:         &board.Ticket{Title: "Fix the bug"},
			expectContains: []string{"Work on: Fix the bug"},
		},
		{
			name:     "multiple field substitution",
			template: "Title: {{.Title}}\nBranch: {{.BranchName}}\nBase: {{.BaseBranch}}",
			ticket: &board.Ticket{
				Title:      "New feature",
				BranchName: "feature/new-thing",
				BaseBranch: "main",
			},
			expectContains: []string{
				"Title: New feature",
				"Branch: feature/new-thing",
				"Base: main",
			},
		},
		{
			name:     "description field",
			template: "{{.Title}}: {{.Description}}",
			ticket: &board.Ticket{
				Title:       "Bug fix",
				Description: "Fix the critical issue",
			},
			expectContains: []string{"Bug fix: Fix the critical issue"},
		},
		{
			name:     "all fields",
			template: "ID={{.TicketID}} Title={{.Title}} Status={{.Status}} Path={{.WorktreePath}}",
			ticket: &board.Ticket{
				ID:           "abc-123",
				Title:        "Test",
				Status:       board.StatusInProgress,
				WorktreePath: "/path/to/worktree",
			},
			expectContains: []string{
				"ID=abc-123",
				"Title=Test",
				"Status=in_progress",
				"Path=/path/to/worktree",
			},
		},
		{
			name:     "handles empty fields gracefully",
			template: "Title={{.Title}} Desc={{.Description}}",
			ticket: &board.Ticket{
				Title:       "Only title",
				Description: "",
			},
			expectContains: []string{"Title=Only title", "Desc="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildContextPrompt(tt.template, tt.ticket)

			if tt.expectEmpty {
				if result != "" {
					t.Errorf("BuildContextPrompt() = %q; want empty", result)
				}
				return
			}

			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("BuildContextPrompt() = %q; want to contain %q", result, expected)
				}
			}
		})
	}
}

func TestBuildContextPrompt_InvalidTemplate(t *testing.T) {
	ticket := &board.Ticket{
		Title:       "Test ticket",
		Description: "Some description",
	}

	result := BuildContextPrompt("{{.InvalidSyntax", ticket)

	if result == "" {
		t.Error("BuildContextPrompt with invalid template should return fallback, not empty")
	}
	if !strings.Contains(result, "Test ticket") {
		t.Errorf("fallback should contain ticket title; got %q", result)
	}
}

func TestBuildFallbackPrompt(t *testing.T) {
	tests := []struct {
		name           string
		ticket         *board.Ticket
		expectContains []string
	}{
		{
			name: "title only",
			ticket: &board.Ticket{
				Title: "Simple task",
			},
			expectContains: []string{"Task: Simple task"},
		},
		{
			name: "title and description",
			ticket: &board.Ticket{
				Title:       "Complex task",
				Description: "Do these things",
			},
			expectContains: []string{"Task: Complex task", "Do these things"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFallbackPrompt(tt.ticket)
			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("buildFallbackPrompt() = %q; want to contain %q", result, expected)
				}
			}
		})
	}
}

func TestShouldInjectContext(t *testing.T) {
	tests := []struct {
		name     string
		ticket   *board.Ticket
		expected bool
	}{
		{
			name:     "new ticket without spawn time",
			ticket:   &board.Ticket{AgentSpawnedAt: nil},
			expected: true,
		},
		{
			name: "previously spawned ticket",
			ticket: &board.Ticket{
				AgentSpawnedAt: func() *time.Time { t := time.Now(); return &t }(),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldInjectContext(tt.ticket)
			if result != tt.expected {
				t.Errorf("ShouldInjectContext() = %v; want %v", result, tt.expected)
			}
		})
	}
}

func TestContextData_AllFieldsMapped(t *testing.T) {
	ticket := &board.Ticket{
		ID:           "test-id-123",
		Title:        "Test Title",
		Description:  "Test Description",
		BranchName:   "feature/test",
		BaseBranch:   "main",
		Status:       board.StatusInProgress,
		WorktreePath: "/home/user/project-worktrees/test",
	}

	template := "{{.TicketID}}|{{.Title}}|{{.Description}}|{{.BranchName}}|{{.BaseBranch}}|{{.Status}}|{{.WorktreePath}}"
	result := BuildContextPrompt(template, ticket)

	expected := "test-id-123|Test Title|Test Description|feature/test|main|in_progress|/home/user/project-worktrees/test"
	if result != expected {
		t.Errorf("All fields mapping:\ngot:  %q\nwant: %q", result, expected)
	}
}
