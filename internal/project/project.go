package project

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a git repository registered with OpenKanban.
// Each git repo is exactly one Project - this is the fundamental unit of organization.
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	RepoPath    string    `json:"repo_path"`    // Absolute path to git repo root
	WorktreeDir string    `json:"worktree_dir"` // Where worktrees go (default: {repo}-worktrees)
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Project-specific settings (overrides global defaults)
	Settings ProjectSettings `json:"settings"`
}

// ProjectSettings contains project-specific configuration.
// These override global defaults from config.Config.
type ProjectSettings struct {
	DefaultAgent     string `json:"default_agent,omitempty"`
	AutoSpawnAgent   bool   `json:"auto_spawn_agent"`
	AutoCreateBranch bool   `json:"auto_create_branch"`
	BranchPrefix     string `json:"branch_prefix,omitempty"`
	BranchNaming     string `json:"branch_naming,omitempty"`   // "template" | "ai" | "prompt"
	BranchTemplate   string `json:"branch_template,omitempty"` // e.g., "{prefix}{slug}"
	SlugMaxLength    int    `json:"slug_max_length,omitempty"` // default: 40
}

// NewProject creates a new project for a repository
func NewProject(name, repoPath string) *Project {
	now := time.Now()

	// Default worktree dir is sibling to repo: /path/to/repo -> /path/to/repo-worktrees
	worktreeDir := repoPath + "-worktrees"

	return &Project{
		ID:          uuid.New().String(),
		Name:        name,
		RepoPath:    repoPath,
		WorktreeDir: worktreeDir,
		CreatedAt:   now,
		UpdatedAt:   now,
		Settings: ProjectSettings{
			AutoSpawnAgent:   true,
			AutoCreateBranch: true,
			BranchPrefix:     "task/",
			BranchNaming:     "template",
			BranchTemplate:   "{prefix}{slug}",
			SlugMaxLength:    40,
		},
	}
}

// GetWorktreeDir returns the worktree directory, using default if not set
func (p *Project) GetWorktreeDir() string {
	if p.WorktreeDir != "" {
		return p.WorktreeDir
	}
	return p.RepoPath + "-worktrees"
}

// GetBranchPrefix returns the branch prefix, using default if not set
func (p *Project) GetBranchPrefix() string {
	if p.Settings.BranchPrefix != "" {
		return p.Settings.BranchPrefix
	}
	return "task/"
}

// GetBranchTemplate returns the branch template, using default if not set
func (p *Project) GetBranchTemplate() string {
	if p.Settings.BranchTemplate != "" {
		return p.Settings.BranchTemplate
	}
	return "{prefix}{slug}"
}

// GetSlugMaxLength returns the slug max length, using default if not set
func (p *Project) GetSlugMaxLength() int {
	if p.Settings.SlugMaxLength > 0 {
		return p.Settings.SlugMaxLength
	}
	return 40
}

// Touch updates the UpdatedAt timestamp
func (p *Project) Touch() {
	p.UpdatedAt = time.Now()
}
