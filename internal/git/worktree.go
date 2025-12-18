package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorktreeManager handles git worktree operations
type WorktreeManager struct {
	repoPath string
	baseDir  string
}

// NewWorktreeManager creates a new worktree manager
func NewWorktreeManager(repoPath, baseDir string) *WorktreeManager {
	return &WorktreeManager{
		repoPath: repoPath,
		baseDir:  baseDir,
	}
}

// CreateWorktree creates a new git worktree with a new branch
func (m *WorktreeManager) CreateWorktree(branchName, baseBranch string) (string, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(m.baseDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create worktree base directory: %w", err)
	}

	// Worktree path based on branch name
	worktreePath := filepath.Join(m.baseDir, sanitizeBranchName(branchName))

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return worktreePath, nil
	}

	// Create new branch and worktree
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath, baseBranch)
	cmd.Dir = m.repoPath

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to create worktree: %s: %w", string(output), err)
	}

	return worktreePath, nil
}

// RemoveWorktree removes a git worktree
func (m *WorktreeManager) RemoveWorktree(worktreePath string) error {
	// First, remove from git
	cmd := exec.Command("git", "worktree", "remove", worktreePath, "--force")
	cmd.Dir = m.repoPath

	if output, err := cmd.CombinedOutput(); err != nil {
		// If worktree not found, just clean up the directory
		if !strings.Contains(string(output), "not a working tree") {
			return fmt.Errorf("failed to remove worktree: %s: %w", string(output), err)
		}
	}

	// Clean up directory if it still exists
	if _, err := os.Stat(worktreePath); err == nil {
		if err := os.RemoveAll(worktreePath); err != nil {
			return fmt.Errorf("failed to remove worktree directory: %w", err)
		}
	}

	return nil
}

// ListWorktrees returns all worktrees for the repository
func (m *WorktreeManager) ListWorktrees() ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = m.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return parseWorktreeList(string(output)), nil
}

// Worktree represents a git worktree
type Worktree struct {
	Path   string
	HEAD   string
	Branch string
}

// parseWorktreeList parses git worktree list --porcelain output
func parseWorktreeList(output string) []Worktree {
	var worktrees []Worktree
	var current Worktree

	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			if current.Path != "" {
				worktrees = append(worktrees, current)
			}
			current = Worktree{Path: strings.TrimPrefix(line, "worktree ")}
		} else if strings.HasPrefix(line, "HEAD ") {
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		}
	}

	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees
}

// GetDefaultBranch returns the default branch of the repository
func (m *WorktreeManager) GetDefaultBranch() (string, error) {
	// Try to get from remote
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = m.repoPath

	output, err := cmd.Output()
	if err == nil {
		branch := strings.TrimSpace(string(output))
		branch = strings.TrimPrefix(branch, "refs/remotes/origin/")
		return branch, nil
	}

	// Fall back to common defaults
	for _, branch := range []string{"main", "master"} {
		cmd := exec.Command("git", "rev-parse", "--verify", branch)
		cmd.Dir = m.repoPath
		if err := cmd.Run(); err == nil {
			return branch, nil
		}
	}

	return "main", nil
}

// DeleteBranch deletes a git branch
func (m *WorktreeManager) DeleteBranch(branchName string) error {
	cmd := exec.Command("git", "branch", "-D", branchName)
	cmd.Dir = m.repoPath

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete branch: %s: %w", string(output), err)
	}

	return nil
}

// HasUncommittedChanges checks if the worktree has uncommitted changes
func (m *WorktreeManager) HasUncommittedChanges(worktreePath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = worktreePath

	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	return len(strings.TrimSpace(string(output))) > 0, nil
}

// sanitizeBranchName converts a branch name to a safe directory name
func sanitizeBranchName(name string) string {
	// Remove common prefixes
	name = strings.TrimPrefix(name, "refs/heads/")
	name = strings.TrimPrefix(name, "agent/")
	name = strings.TrimPrefix(name, "feature/")

	// Replace slashes with dashes
	name = strings.ReplaceAll(name, "/", "-")

	return name
}
