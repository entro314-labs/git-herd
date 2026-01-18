package tui

import (
	"fmt"
	"os"
	"time"

	"github.com/entro314-labs/git-herd/pkg/types"
)

// saveReport saves a detailed report to a file
func saveReport(config *types.Config, results []types.GitRepo, successful, failed, skipped int) (err error) {
	file, err := os.Create(config.SaveReport)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close report file: %w", closeErr)
		}
	}()

	var writeErr error
	fprintf := func(format string, a ...interface{}) {
		if writeErr != nil {
			return
		}
		_, writeErr = fmt.Fprintf(file, format, a...)
	}

	// Write header
	fprintf("git-herd Report - %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fprintf("Operation: %s\n", config.Operation)
	fprintf("Workers: %d\n", config.Workers)
	fprintf("Total Repositories: %d\n", len(results))
	fprintf("Successful: %d, Failed: %d, Skipped: %d\n\n", successful, failed, skipped)

	fprintf("Repository Details:\n")
	fprintf("==================\n\n")

	for _, result := range results {
		fprintf("Repository: %s\n", result.Name)
		fprintf("Path: %s\n", result.Path)

		if result.Branch != "" {
			fprintf("Branch: %s\n", result.Branch)
		}
		if result.Remote != "" {
			fprintf("Remote: %s\n", result.Remote)
		}

		fprintf("Duration: %v\n", result.Duration.Truncate(time.Millisecond))

		if result.Error != nil {
			fprintf("Status: FAILED - %v\n", result.Error)
		} else if config.DryRun {
			fprintf("Status: DRY RUN - Would have succeeded\n")
		} else {
			fprintf("Status: SUCCESS\n")
		}

		fprintf("\n")
	}

	if writeErr != nil {
		return fmt.Errorf("failed to write to report file: %w", writeErr)
	}

	return nil
}
