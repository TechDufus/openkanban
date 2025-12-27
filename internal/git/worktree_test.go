package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsValidWorktree(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := &WorktreeManager{repoPath: tmpDir, baseDir: tmpDir}

	tests := []struct {
		name     string
		setup    func(path string) error
		expected bool
	}{
		{
			name: "valid worktree with .git file",
			setup: func(path string) error {
				if err := os.MkdirAll(path, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(path, ".git"), []byte("gitdir: /some/path"), 0644)
			},
			expected: true,
		},
		{
			name: "invalid - .git is directory (normal repo)",
			setup: func(path string) error {
				return os.MkdirAll(filepath.Join(path, ".git"), 0755)
			},
			expected: false,
		},
		{
			name: "invalid - no .git at all",
			setup: func(path string) error {
				return os.MkdirAll(path, 0755)
			},
			expected: false,
		},
		{
			name: "invalid - path does not exist",
			setup: func(path string) error {
				return nil
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPath := filepath.Join(tmpDir, tt.name)
			if err := tt.setup(testPath); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			result := mgr.isValidWorktree(testPath)
			if result != tt.expected {
				t.Errorf("isValidWorktree(%q) = %v; want %v", testPath, result, tt.expected)
			}
		})
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name unchanged",
			input:    "my-branch",
			expected: "my-branch",
		},
		{
			name:     "strips refs/heads/ prefix",
			input:    "refs/heads/my-branch",
			expected: "my-branch",
		},
		{
			name:     "strips agent/ prefix",
			input:    "agent/my-task",
			expected: "my-task",
		},
		{
			name:     "strips feature/ prefix",
			input:    "feature/new-thing",
			expected: "new-thing",
		},
		{
			name:     "replaces slashes with dashes",
			input:    "user/feature/thing",
			expected: "user-feature-thing",
		},
		{
			name:     "combines prefix stripping and slash replacement",
			input:    "refs/heads/feature/my/nested/branch",
			expected: "my-nested-branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeBranchName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeBranchName(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseWorktreeList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Worktree
	}{
		{
			name:     "empty output",
			input:    "",
			expected: nil,
		},
		{
			name: "single worktree",
			input: `worktree /home/user/project
HEAD abc123
branch refs/heads/main
`,
			expected: []Worktree{
				{Path: "/home/user/project", HEAD: "abc123", Branch: "main"},
			},
		},
		{
			name: "multiple worktrees",
			input: `worktree /home/user/project
HEAD abc123
branch refs/heads/main

worktree /home/user/project-worktrees/feature
HEAD def456
branch refs/heads/feature/new-thing
`,
			expected: []Worktree{
				{Path: "/home/user/project", HEAD: "abc123", Branch: "main"},
				{Path: "/home/user/project-worktrees/feature", HEAD: "def456", Branch: "feature/new-thing"},
			},
		},
		{
			name: "detached HEAD worktree",
			input: `worktree /home/user/project
HEAD abc123
detached
`,
			expected: []Worktree{
				{Path: "/home/user/project", HEAD: "abc123", Branch: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseWorktreeList(tt.input)

			if len(result) != len(tt.expected) {
				t.Fatalf("parseWorktreeList() returned %d worktrees; want %d", len(result), len(tt.expected))
			}

			for i, exp := range tt.expected {
				if result[i].Path != exp.Path {
					t.Errorf("worktree[%d].Path = %q; want %q", i, result[i].Path, exp.Path)
				}
				if result[i].HEAD != exp.HEAD {
					t.Errorf("worktree[%d].HEAD = %q; want %q", i, result[i].HEAD, exp.HEAD)
				}
				if result[i].Branch != exp.Branch {
					t.Errorf("worktree[%d].Branch = %q; want %q", i, result[i].Branch, exp.Branch)
				}
			}
		})
	}
}

func TestResolveMainRepo(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("normal repo returns same path", func(t *testing.T) {
		repoPath := filepath.Join(tmpDir, "normal-repo")
		if err := os.MkdirAll(filepath.Join(repoPath, ".git"), 0755); err != nil {
			t.Fatal(err)
		}

		result := ResolveMainRepo(repoPath)
		if result != repoPath {
			t.Errorf("ResolveMainRepo(%q) = %q; want same path", repoPath, result)
		}
	})

	t.Run("worktree resolves to main repo", func(t *testing.T) {
		worktreePath := filepath.Join(tmpDir, "worktree")
		if err := os.MkdirAll(worktreePath, 0755); err != nil {
			t.Fatal(err)
		}

		gitContent := "gitdir: /home/user/main-repo/.git/worktrees/feature-branch"
		if err := os.WriteFile(filepath.Join(worktreePath, ".git"), []byte(gitContent), 0644); err != nil {
			t.Fatal(err)
		}

		result := ResolveMainRepo(worktreePath)
		if result != "/home/user/main-repo" {
			t.Errorf("ResolveMainRepo(%q) = %q; want %q", worktreePath, result, "/home/user/main-repo")
		}
	})

	t.Run("no .git returns same path", func(t *testing.T) {
		noGitPath := filepath.Join(tmpDir, "no-git")
		if err := os.MkdirAll(noGitPath, 0755); err != nil {
			t.Fatal(err)
		}

		result := ResolveMainRepo(noGitPath)
		if result != noGitPath {
			t.Errorf("ResolveMainRepo(%q) = %q; want same path", noGitPath, result)
		}
	})
}

func TestNewWorktreeManagerFromPaths(t *testing.T) {
	mgr := NewWorktreeManagerFromPaths("/repo/path", "/worktrees/path")

	if mgr.repoPath != "/repo/path" {
		t.Errorf("repoPath = %q; want %q", mgr.repoPath, "/repo/path")
	}
	if mgr.baseDir != "/worktrees/path" {
		t.Errorf("baseDir = %q; want %q", mgr.baseDir, "/worktrees/path")
	}
}
