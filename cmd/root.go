package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/techdufus/openkanban/internal/app"
	"github.com/techdufus/openkanban/internal/config"
)

var (
	cfgFile     string
	projectPath string
)

var rootCmd = &cobra.Command{
	Use:   "openkanban",
	Short: "TUI kanban board for orchestrating AI coding agents",
	Long: `OpenKanban is a terminal-based kanban board that helps you manage
multiple AI coding agents across different tasks and git worktrees.

Each ticket spawns an embedded terminal pane with its own git worktree
for safe parallel development.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		cfg, result, err := config.LoadWithValidation(cfgFile)
		if err != nil {
			if result != nil && result.HasErrors() {
				fmt.Fprintf(os.Stderr, "Configuration errors:\n\n%s", result.FormatErrors())
				fmt.Fprintln(os.Stderr, "Run 'openkanban config validate' for details")
				return errors.New("invalid configuration")
			}
			return fmt.Errorf("failed to load config: %w", err)
		}

		if result != nil && result.HasErrors() {
			fmt.Fprintf(os.Stderr, "Configuration errors:\n\n%s", result.FormatErrors())
			fmt.Fprintln(os.Stderr, "Run 'openkanban config validate' for details")
			return errors.New("invalid configuration")
		}

		if result != nil && result.HasWarnings() {
			fmt.Fprintf(os.Stderr, "Config warnings:\n%s\n", result.FormatWarnings())
		}

		return app.Run(cfg, projectPath, Version)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/openkanban/config.json)")
	rootCmd.PersistentFlags().StringVarP(&projectPath, "project", "p", "", "project or repository path")

	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(deleteCmd)
}

var newCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new project",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, result, err := config.LoadWithValidation(cfgFile)
		if err != nil || (result != nil && result.HasErrors()) {
			if result != nil && result.HasErrors() {
				fmt.Fprintf(os.Stderr, "Configuration errors:\n\n%s", result.FormatErrors())
				return errors.New("invalid configuration")
			}
			return fmt.Errorf("failed to load config: %w", err)
		}

		repoPath := projectPath
		if repoPath == "" {
			repoPath, _ = os.Getwd()
		}

		repoPath, err = filepath.Abs(repoPath)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		name := filepath.Base(repoPath)
		if len(args) > 0 {
			name = args[0]
		}

		return app.CreateProject(cfg, name, repoPath)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.ListProjects()
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <name-or-id>",
	Short: "Delete a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.DeleteProject(args[0])
	},
}
