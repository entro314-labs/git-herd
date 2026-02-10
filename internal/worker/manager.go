package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/sync/errgroup"

	"github.com/entro314-labs/git-herd/internal/git"
	"github.com/entro314-labs/git-herd/internal/tui"
	"github.com/entro314-labs/git-herd/pkg/types"
)

// Manager handles bulk git operations with worker pools
type Manager struct {
	config    *types.Config
	logger    *slog.Logger
	scanner   *git.Scanner
	processor *git.Processor
}

// New creates a new Manager instance
func New(config *types.Config) *Manager {
	level := slog.LevelInfo
	if config.Verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return &Manager{
		config:    config,
		logger:    slog.New(handler),
		scanner:   git.NewScanner(config),
		processor: git.NewProcessor(config),
	}
}

// Execute runs the bulk git operation
func (m *Manager) Execute(ctx context.Context, rootPath string) error {
	// Use TUI if not in plain mode and not verbose (TUI doesn't work well with verbose logging)
	if !m.config.PlainMode && !m.config.Verbose {
		model := tui.NewModel(ctx, m.config, rootPath)
		defer model.Cancel()
		p := tea.NewProgram(model)

		finalModel, err := p.Run()
		if err != nil {
			// Fallback to plain mode if TUI fails
			m.logger.Error("TUI failed, falling back to plain mode", "error", err)
			return m.executeInPlainMode(ctx, rootPath)
		}

		finalTUI, ok := finalModel.(*tui.Model)
		if !ok {
			return fmt.Errorf("unexpected TUI model type: %T", finalModel)
		}

		results := finalTUI.Results()
		finalErr := finalTUI.Error()
		if ctx.Err() != nil {
			finalErr = errors.Join(finalErr, ctx.Err())
		}

		if len(results) == 0 {
			return finalErr
		}

		successful, failed, skipped := summarizeResults(results)
		if persistErr := m.persistArtifacts(ctx, results, successful, failed, skipped); persistErr != nil {
			finalErr = errors.Join(finalErr, persistErr)
		}
		if failed > 0 {
			finalErr = errors.Join(finalErr, fmt.Errorf("%d repositories failed", failed))
		}

		return finalErr
	}

	return m.executeInPlainMode(ctx, rootPath)
}

// executeInPlainMode runs the operation with plain text output
func (m *Manager) executeInPlainMode(ctx context.Context, rootPath string) error {
	m.logger.InfoContext(ctx, "Starting bulk git operation",
		"operation", m.config.Operation,
		"path", rootPath,
		"workers", m.config.Workers)

	// Find all git repositories
	if m.config.PlainMode || m.config.Verbose {
		fmt.Printf("🔍 Scanning for Git repositories in %s...\n", rootPath)
	}

	repos, err := m.scanner.FindRepos(ctx, rootPath, func(count int) {
		if (m.config.PlainMode || m.config.Verbose) && count%10 == 0 {
			fmt.Printf("   Found %d repositories so far...\n", count)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to find repositories: %w", err)
	}

	if m.config.PlainMode || m.config.Verbose {
		fmt.Printf("✅ Scan complete: found %d Git repositories\n", len(repos))
	}

	if len(repos) == 0 {
		m.logger.InfoContext(ctx, "No git repositories found")
		return nil
	}

	m.logger.InfoContext(ctx, "Found repositories", "count", len(repos))

	// Process repositories concurrently
	return m.processReposConcurrently(ctx, repos)
}

// processReposConcurrently processes repositories using worker pools
func (m *Manager) processReposConcurrently(ctx context.Context, repos []types.GitRepo) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(m.config.Workers)

	resultChan := make(chan types.GitRepo, len(repos))
	groupErrChan := make(chan error, 1)

	// Start workers
	for _, repo := range repos {
		g.Go(func() error {
			processedRepo := m.processor.ProcessRepo(ctx, repo)
			select {
			case resultChan <- processedRepo:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}

	// Start result collector
	go func() {
		err := g.Wait()
		if err != nil {
			m.logger.Error("Worker group failed", "error", err)
		}
		groupErrChan <- err
		close(resultChan)
	}()

	// Collect and display results
	resultErr := m.displayResults(ctx, resultChan, len(repos))
	groupErr := <-groupErrChan
	if groupErr != nil {
		return errors.Join(groupErr, resultErr)
	}
	return resultErr
}

// displayResults shows the results of the operations
func (m *Manager) displayResults(ctx context.Context, resultChan <-chan types.GitRepo, total int) error {
	var successful, failed, skipped int
	var allResults []types.GitRepo

	fmt.Printf("\n📊 Processing Results:\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	for result := range resultChan {
		allResults = append(allResults, result)

		if result.Error != nil {
			if errors.Is(result.Error, types.ErrRepoSkipped) {
				skipped++
			} else {
				failed++
			}
			if m.config.FullSummary {
				fmt.Printf("❌ %s (%s): %v\n", result.Name, result.Path, result.Error)
			}
		} else {
			successful++
			status := "✅"
			if m.config.DryRun {
				status = "🔍"
			}
			if m.config.FullSummary {
				fmt.Printf("%s %s (%s) [%s@%s] - %v\n",
					status, result.Name, result.Path, result.Branch, result.Remote, result.Duration.Truncate(time.Millisecond))
			}
		}
	}

	// Show condensed view if not full summary
	if !m.config.FullSummary {
		// Show only first few and last few results
		displayCount := 5
		if len(allResults) <= displayCount*2 {
			displayCount = len(allResults) / 2
		}

		for i, result := range allResults[:displayCount] {
			m.displaySingleResult(result, i == 0)
		}

		if len(allResults) > displayCount*2 {
			fmt.Printf("... (%d more repositories) ...\n", len(allResults)-displayCount*2)
		}

		if len(allResults) > displayCount {
			start := len(allResults) - displayCount
			if len(allResults) <= displayCount*2 {
				start = displayCount
			}
			for _, result := range allResults[start:] {
				m.displaySingleResult(result, false)
			}
		}
	}

	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📈 Summary: %d successful, %d failed, %d skipped, %d total\n", successful, failed, skipped, total)

	// Save report/export scan results if requested
	finalErr := m.persistArtifacts(ctx, allResults, successful, failed, skipped)

	if !m.config.FullSummary && len(allResults) > 10 {
		fmt.Printf("💡 Use --full-summary flag to see all %d repositories\n", len(allResults))
	}

	if failed > 0 {
		finalErr = errors.Join(finalErr, fmt.Errorf("%d repositories failed", failed))
	}

	return finalErr
}

func summarizeResults(results []types.GitRepo) (successful, failed, skipped int) {
	for _, result := range results {
		if result.Error != nil {
			if errors.Is(result.Error, types.ErrRepoSkipped) {
				skipped++
			} else {
				failed++
			}
		} else {
			successful++
		}
	}

	return successful, failed, skipped
}

func (m *Manager) persistArtifacts(ctx context.Context, results []types.GitRepo, successful, failed, skipped int) error {
	var finalErr error
	if m.config.SaveReport != "" {
		if err := tui.SaveReport(m.config, results, successful, failed, skipped); err != nil {
			m.logger.ErrorContext(ctx, "Failed to save report", "error", err)
			fmt.Fprintf(os.Stderr, "Error saving report: %v\n", err)
			finalErr = errors.Join(finalErr, err)
		} else {
			fmt.Printf("📄 Detailed report saved to: %s\n", m.config.SaveReport)
		}
	}

	if m.config.ExportScan != "" {
		if err := m.exportScanToMarkdown(results, m.config.ExportScan); err != nil {
			m.logger.ErrorContext(ctx, "Failed to export scan", "error", err)
			fmt.Fprintf(os.Stderr, "Error exporting scan: %v\n", err)
			finalErr = errors.Join(finalErr, err)
		} else {
			fmt.Printf("📋 Scan report exported to: %s\n", m.config.ExportScan)
		}
	}

	return finalErr
}

// displaySingleResult displays a single repository result
func (m *Manager) displaySingleResult(result types.GitRepo, isFirst bool) {
	if result.Error != nil {
		if errors.Is(result.Error, types.ErrRepoSkipped) {
			fmt.Printf("⊝ %s (%s): %v\n", result.Name, result.Path, result.Error)
		} else {
			fmt.Printf("❌ %s (%s): %v\n", result.Name, result.Path, result.Error)
		}
	} else {
		status := "✅"
		if m.config.DryRun {
			status = "🔍"
		}
		fmt.Printf("%s %s (%s) [%s@%s] - %v\n",
			status, result.Name, result.Path, result.Branch, result.Remote, result.Duration.Truncate(time.Millisecond))
	}
}

// exportScanToMarkdown exports repository scan results to a markdown file
func (m *Manager) exportScanToMarkdown(results []types.GitRepo, filePath string) (err error) {
	rootDir := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)
	if fileName == "." || fileName == string(os.PathSeparator) {
		return fmt.Errorf("export file path is invalid: %s", filePath)
	}
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer func() {
		err = errors.Join(err, root.Close())
	}()

	file, err := root.OpenFile(fileName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	var writeErr error
	writef := func(label, format string, a ...any) {
		if writeErr != nil {
			return
		}
		_, writeErr = fmt.Fprintf(file, format, a...)
		if writeErr != nil {
			writeErr = fmt.Errorf("failed to write %s: %w", label, writeErr)
		}
	}

	// Write header
	writef("header", "# Git Repository Scan Report\n\n")
	writef("timestamp", "Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	writef("total", "Total Repositories: %d\n\n", len(results))
	writef("separator", "---\n\n")

	// Write repository details
	for _, repo := range results {
		writef("repo name", "## %s\n\n", repo.Name)
		writef("path", "**Path:** `%s`\n\n", repo.Path)

		if repo.Branch != "" {
			writef("branch", "**Branch:** %s\n\n", repo.Branch)
		}

		if repo.Remote != "" {
			writef("remote", "**Remote:** %s\n\n", repo.Remote)
		}

		if repo.LastCommit != "" {
			writef("commit", "**Last Commit:** `%s`\n\n", repo.LastCommit)
			if repo.LastCommitMsg != "" {
				writef("commit message", "**Commit Message:** %s\n\n", repo.LastCommitMsg)
			}
		}

		if len(repo.ModifiedFiles) > 0 {
			writef("modified files header", "**Modified Files:**\n\n")
			for _, modFile := range repo.ModifiedFiles {
				writef("modified file", "- `%s`\n", modFile)
			}
			writef("modified files newline", "\n")
		} else {
			writef("clean status", "**Status:** Clean (no local changes)\n\n")
		}

		if repo.Error != nil {
			writef("error", "**Error:** %v\n\n", repo.Error)
		}

		writef("separator", "---\n\n")
	}

	if writeErr != nil {
		return writeErr
	}

	return nil
}
