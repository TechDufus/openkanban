package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/techdufus/openkanban/internal/app"
	"github.com/techdufus/openkanban/internal/config"
)

var (
	cfgFile   string
	boardPath string
)

var rootCmd = &cobra.Command{
	Use:   "openkanban",
	Short: "TUI kanban board for orchestrating AI coding agents",
	Long: `OpenKanban is a terminal-based kanban board that helps you manage
multiple AI coding agents across different tasks and git worktrees.

Each ticket spawns an embedded terminal pane with its own git worktree
for safe parallel development.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := EnsureDaemon(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: daemon not available: %v\n", err)
		}

		bp := boardPath
		if bp == "" {
			cwd, _ := os.Getwd()
			bp = cwd
		}

		return app.Run(cfg, bp)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/openkanban/config.json)")
	rootCmd.PersistentFlags().StringVarP(&boardPath, "board", "b", "", "board or repository path")

	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(listCmd)
}

var newCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new board",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := "default"
		if len(args) > 0 {
			name = args[0]
		}

		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		repoPath := boardPath
		if repoPath == "" {
			repoPath, _ = os.Getwd()
		}

		repoPath, err = filepath.Abs(repoPath)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		return app.CreateBoard(cfg, name, repoPath)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all boards",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		return app.ListBoards(cfg)
	},
}
