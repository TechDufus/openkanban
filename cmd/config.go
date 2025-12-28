package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/techdufus/openkanban/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  "Commands for managing OpenKanban configuration files.",
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Long:  "Check the configuration file for errors and display helpful messages.",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgFile
		if path == "" {
			var err error
			path, err = config.ConfigPath()
			if err != nil {
				return fmt.Errorf("failed to determine config path: %w", err)
			}
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("No config file found at %s\n", path)
			fmt.Println("Using default configuration.")
			fmt.Println("\nRun 'openkanban config generate' to create a config file.")
			return nil
		}

		cfg, result, err := config.LoadWithValidation(path)
		if err != nil && result == nil {
			return fmt.Errorf("failed to read config: %w", err)
		}

		if result != nil && result.HasErrors() {
			fmt.Fprintf(os.Stderr, "Config errors in %s:\n\n", path)
			fmt.Fprint(os.Stderr, result.FormatErrors())
			os.Exit(1)
		}

		if result != nil && result.HasWarnings() {
			fmt.Printf("Config valid with %d warning(s) in %s:\n\n", len(result.Warnings), path)
			fmt.Print(result.FormatWarnings())
			return nil
		}

		_ = cfg
		fmt.Printf("Configuration is valid: %s\n", path)
		return nil
	},
}

var forceGenerate bool

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate default configuration file",
	Long:  "Create a new configuration file with default values.",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgFile
		if path == "" {
			var err error
			path, err = config.ConfigPath()
			if err != nil {
				return fmt.Errorf("failed to determine config path: %w", err)
			}
		}

		if _, err := os.Stat(path); err == nil {
			if !forceGenerate {
				return fmt.Errorf("config file already exists at %s (use --force to overwrite)", path)
			}
		}

		cfg := config.DefaultConfig()
		if err := cfg.Save(path); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		fmt.Printf("Generated config at %s\n", path)
		return nil
	},
}

var showPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgFile
		if path == "" {
			var err error
			path, err = config.ConfigPath()
			if err != nil {
				return fmt.Errorf("failed to determine config path: %w", err)
			}
		}

		fmt.Println(path)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "(file does not exist)")
		}
		return nil
	},
}

func init() {
	configCmd.AddCommand(validateCmd)
	configCmd.AddCommand(generateCmd)
	configCmd.AddCommand(showPathCmd)

	generateCmd.Flags().BoolVarP(&forceGenerate, "force", "f", false, "overwrite existing config file")

	rootCmd.AddCommand(configCmd)
}
