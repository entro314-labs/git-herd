package config

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/entro314-labs/git-herd/pkg/types"
)

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *types.Config {
	return &types.Config{
		Workers:     5,
		Operation:   types.OperationFetch,
		DryRun:      false,
		Recursive:   true,
		SkipDirty:   true,
		Verbose:     false,
		Timeout:     5 * time.Minute,
		ExcludeDirs: []string{".git", "node_modules", "vendor"},
		PlainMode:   false,
		FullSummary: false,
		SaveReport:  "",
	}
}

// SetupFlags configures command line flags for the root command
func SetupFlags(cmd *cobra.Command, config *types.Config) {
	// Flags
	cmd.Flags().StringVarP((*string)(&config.Operation), "operation", "o", "fetch", "Operation to perform: fetch or pull")
	cmd.Flags().IntVarP(&config.Workers, "workers", "w", 5, "Number of concurrent workers")
	cmd.Flags().BoolVarP(&config.DryRun, "dry-run", "n", false, "Show what would be done without executing")
	cmd.Flags().BoolVarP(&config.Recursive, "recursive", "r", true, "Process repositories recursively")
	cmd.Flags().BoolVarP(&config.SkipDirty, "skip-dirty", "s", true, "Skip repositories with uncommitted changes")
	cmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", false, "Enable verbose logging")
	cmd.Flags().BoolVarP(&config.PlainMode, "plain", "p", false, "Use plain text output instead of TUI")
	cmd.Flags().BoolVarP(&config.FullSummary, "full-summary", "f", false, "Display full summary of all repositories")
	cmd.Flags().StringVarP(&config.SaveReport, "save-report", "", "", "Save detailed report to file (e.g., report.txt)")
	cmd.Flags().DurationVarP(&config.Timeout, "timeout", "t", 5*time.Minute, "Overall operation timeout")
	cmd.Flags().StringSliceVarP(&config.ExcludeDirs, "exclude", "e", []string{".git", "node_modules", "vendor"}, "Directories to exclude")
}

// SetupViper configures viper for configuration file support
func SetupViper(cmd *cobra.Command) error {
	// Setup viper for configuration file support
	viper.SetConfigName("git-herd")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/git-herd")

	// Bind flags to viper
	flags := []string{
		"operation", "workers", "dry-run", "recursive", "skip-dirty",
		"verbose", "plain", "full-summary", "save-report", "timeout", "exclude",
	}

	for _, name := range flags {
		if err := viper.BindPFlag(name, cmd.Flags().Lookup(name)); err != nil {
			return fmt.Errorf("failed to bind flag %s: %v", name, err)
		}
	}

	// Try to read config file (ignore error if file doesn't exist)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error occurred
			fmt.Printf("Warning: failed to parse config file: %v\n", err)
		}
	}

	return nil
}

// LoadConfig loads and validates configuration
func LoadConfig() (*types.Config, error) {
	config := DefaultConfig()

	// Load from viper (which includes file and flags)
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}
