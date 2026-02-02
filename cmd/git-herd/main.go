package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/entro314-labs/git-herd/internal/config"
	"github.com/entro314-labs/git-herd/internal/worker"
	"github.com/entro314-labs/git-herd/pkg/types"
)

// Version information - populated at build time by GoReleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

// buildVersion returns a formatted version string
func buildVersion() string {
	if version == "dev" {
		return fmt.Sprintf("%s (built from source)", version)
	}
	return fmt.Sprintf("%s (commit: %s, built: %s, by: %s)", version, commit, date, builtBy)
}

func main() {
	cfg := config.DefaultConfig()
	rootCmd := newRootCommand(cfg)
	cobra.CheckErr(rootCmd.Execute())
}

func newRootCommand(cfg *types.Config) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "git-herd [path]",
		Short: "Bulk git operations on multiple repositories",
		Long: `git-herd performs git operations (fetch/pull) on all git repositories
found in the specified directory and its subdirectories.`,
		Version: buildVersion(),
		Args:    cobra.MaximumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := config.SetupViper(cmd); err != nil {
				return err
			}

			loadedCfg, err := config.LoadConfig()
			if err != nil {
				return err
			}

			*cfg = *loadedCfg
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Setup signal handling for graceful shutdown
			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			// Add timeout if specified
			if cfg.Timeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
				defer cancel()
			}

			// Determine root path
			rootPath := "."
			if len(args) > 0 {
				rootPath = args[0]
			}

			// Validate path
			info, err := os.Stat(rootPath)
			if err != nil {
				return fmt.Errorf("stat path %s: %w", rootPath, err)
			}
			if !info.IsDir() {
				return fmt.Errorf("path is not a directory: %s", rootPath)
			}

			// Create and execute manager
			manager := worker.New(cfg)
			return manager.Execute(ctx, rootPath)
		},
	}

	// Setup configuration flags
	config.SetupFlags(rootCmd, cfg)

	return rootCmd
}
