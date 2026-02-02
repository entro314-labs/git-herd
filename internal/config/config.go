package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/entro314-labs/git-herd/pkg/types"
)

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *types.Config {
	return &types.Config{
		Workers:      5,
		Operation:    types.OperationFetch,
		DryRun:       false,
		Recursive:    true,
		SkipDirty:    true,
		Verbose:      false,
		Timeout:      5 * time.Minute,
		ExcludeDirs:  []string{".git", "node_modules", "vendor"},
		PlainMode:    false,
		FullSummary:  false,
		SaveReport:   "",
		DiscardFiles: []string{},
		ExportScan:   "",
	}
}

// SetupFlags configures command line flags for the root command
func SetupFlags(cmd *cobra.Command, config *types.Config) {
	// Flags
	cmd.Flags().VarP(newOperationValue(&config.Operation), "operation", "o", "Operation to perform: fetch, pull, or scan")
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
	cmd.Flags().StringSliceVarP(&config.DiscardFiles, "discard-files", "d", []string{}, "File patterns to discard changes before pull/fetch (e.g., package.json,package-lock.json)")
	cmd.Flags().StringVarP(&config.ExportScan, "export-scan", "", "", "Export repository scan to markdown file (use with -o scan)")
}

// operationValue implements pflag.Value for OperationType
type operationValue struct {
	target *types.OperationType
}

func newOperationValue(target *types.OperationType) *operationValue {
	return &operationValue{target: target}
}

func (o *operationValue) String() string {
	return string(*o.target)
}

func (o *operationValue) Set(s string) error {
	*o.target = types.OperationType(s)
	return nil
}

func (o *operationValue) Type() string {
	return "string"
}

// SetupViper configures viper for configuration file support
func SetupViper(cmd *cobra.Command) error {
	// Setup viper for configuration file support
	viper.SetConfigName("git-herd")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if configDir, err := os.UserConfigDir(); err == nil {
		viper.AddConfigPath(filepath.Join(configDir, "git-herd"))
	}

	viper.SetEnvPrefix("GIT_HERD")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// Bind flags to viper
	flags := []string{
		"operation", "workers", "dry-run", "recursive", "skip-dirty",
		"verbose", "plain", "full-summary", "save-report", "timeout", "exclude",
		"discard-files", "export-scan",
	}

	for _, name := range flags {
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			return fmt.Errorf("missing flag definition: %s", name)
		}
		if err := viper.BindPFlag(name, flag); err != nil {
			return fmt.Errorf("bind flag %s: %w", name, err)
		}
		if err := viper.BindEnv(name); err != nil {
			return fmt.Errorf("bind env %s: %w", name, err)
		}
	}

	// Try to read config file (ignore error if file doesn't exist)
	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			return nil
		}
		return fmt.Errorf("read config: %w", err)
	}

	return nil
}

// LoadConfig loads and validates configuration
func LoadConfig() (*types.Config, error) {
	config := DefaultConfig()

	// Load from viper (which includes file and flags)
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// ValidateConfig validates and normalizes configuration
func ValidateConfig(config *types.Config) error {
	if config.Workers <= 0 {
		return fmt.Errorf("workers must be greater than 0")
	}

	if config.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative")
	}

	operation := strings.ToLower(strings.TrimSpace(string(config.Operation)))
	if operation == "" {
		config.Operation = types.OperationFetch
	} else {
		config.Operation = types.OperationType(operation)
		switch config.Operation {
		case types.OperationFetch, types.OperationPull, types.OperationScan:
			// valid
		default:
			return fmt.Errorf("invalid operation: %s (must be 'fetch', 'pull', or 'scan')", config.Operation)
		}
	}

	if config.ExportScan != "" && config.Operation != types.OperationScan {
		return fmt.Errorf("export-scan requires operation 'scan'")
	}

	return nil
}
