package app

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/techdufus/openkanban/internal/agent"
	"github.com/techdufus/openkanban/internal/board"
	"github.com/techdufus/openkanban/internal/config"
	"github.com/techdufus/openkanban/internal/git"
	"github.com/techdufus/openkanban/internal/ui"
)

// Run starts the TUI application
func Run(cfg *config.Config, boardPath string) error {
	// Find or create board
	b, err := findBoard(cfg, boardPath)
	if err != nil {
		return err
	}

	boardDir, err := boardsDir()
	if err != nil {
		return err
	}

	// Initialize managers
	agentMgr := agent.NewManager(cfg)
	worktreeMgr := git.NewWorktreeManager(b.RepoPath, b.Settings.WorktreeBase)

	// Create the TUI model
	model := ui.NewModel(cfg, b, boardDir, agentMgr, worktreeMgr)

	// Run the Bubbletea program
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	return err
}

// CreateBoard creates a new board for a repository
func CreateBoard(cfg *config.Config, name, repoPath string) error {
	// Verify it's a git repository
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Create board with default settings
	settings := board.BoardSettings{
		DefaultAgent:     cfg.Defaults.DefaultAgent,
		WorktreeBase:     cfg.Defaults.WorktreeBase,
		AutoSpawnAgent:   cfg.Defaults.AutoSpawnAgent,
		AutoCreateBranch: cfg.Defaults.AutoCreateBranch,
		BranchPrefix:     cfg.Defaults.BranchPrefix,
	}

	if settings.WorktreeBase == "" {
		settings.WorktreeBase = repoPath + "-worktrees"
	}

	b := board.NewBoard(name, repoPath, settings)

	// Save board
	boardDir, err := boardsDir()
	if err != nil {
		return err
	}

	if err := b.Save(boardDir); err != nil {
		return fmt.Errorf("failed to save board: %w", err)
	}

	fmt.Printf("Created board '%s' for %s\n", name, repoPath)
	fmt.Printf("Board ID: %s\n", b.ID)
	return nil
}

// ListBoards displays all available boards
func ListBoards(cfg *config.Config) error {
	dir, err := boardsDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No boards found. Create one with: openkanban new")
			return nil
		}
		return err
	}

	fmt.Println("Available boards:")
	fmt.Println()

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			path := filepath.Join(dir, entry.Name())
			b, err := board.LoadBoard(path)
			if err != nil {
				continue
			}

			ticketCount := len(b.Tickets)
			inProgress := len(b.GetTicketsByStatus(board.StatusInProgress))

			fmt.Printf("  %s (%s)\n", b.Name, b.ID[:8])
			fmt.Printf("    Path: %s\n", b.RepoPath)
			fmt.Printf("    Tickets: %d total, %d in progress\n", ticketCount, inProgress)
			fmt.Println()
		}
	}

	return nil
}

// findBoard locates or prompts for a board
func findBoard(cfg *config.Config, path string) (*board.Board, error) {
	dir, err := boardsDir()
	if err != nil {
		return nil, err
	}

	// If path is a board ID, load directly
	boardFile := filepath.Join(dir, path+".json")
	if b, err := board.LoadBoard(boardFile); err == nil {
		return b, nil
	}

	// If path is a repo path, find matching board
	absPath, _ := filepath.Abs(path)
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			b, err := board.LoadBoard(filepath.Join(dir, entry.Name()))
			if err == nil && b.RepoPath == absPath {
				return b, nil
			}
		}
	}

	// No board found, offer to create one
	return nil, fmt.Errorf("no board found for %s. Create one with: openkanban new", path)
}

// boardsDir returns the directory where boards are stored
func boardsDir() (string, error) {
	cfgDir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, "boards"), nil
}
