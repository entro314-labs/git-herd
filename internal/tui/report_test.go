package tui

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/entro314-labs/git-herd/internal/config"
	"github.com/entro314-labs/git-herd/pkg/types"
)

func TestSaveReport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file I/O test in short mode")
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-report-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	cfg := config.DefaultConfig()
	cfg.SaveReport = tmpFile.Name()
	cfg.Operation = types.OperationFetch
	cfg.Workers = 5

	results := []types.GitRepo{
		{
			Path:     "/test/repo1",
			Name:     "repo1",
			Branch:   "main",
			Remote:   "origin",
			Duration: 150 * time.Millisecond,
			HasGit:   true,
			Clean:    true,
		},
		{
			Path:     "/test/repo2",
			Name:     "repo2",
			Branch:   "develop",
			Remote:   "upstream",
			Duration: 200 * time.Millisecond,
			HasGit:   true,
			Clean:    true,
		},
		{
			Path:  "/test/repo3",
			Name:  "repo3",
			Error: errors.New("operation failed"),
		},
	}

	err = saveReport(cfg, results, 2, 1, 0)
	if err != nil {
		t.Errorf("saveReport() error = %v", err)
	}

	// Read the file content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	contentStr := string(content)

	// Check header information
	expectedContent := []string{
		"git-herd Report",
		"Operation: fetch",
		"Workers: 5",
		"Total Repositories: 3",
		"Successful: 2, Failed: 1, Skipped: 0",
		"Repository Details:",
		"==================",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Expected report to contain %q, got:\n%s", expected, contentStr)
		}
	}

	// Check repository details
	expectedRepos := []string{
		"Repository: repo1",
		"Path: /test/repo1",
		"Branch: main",
		"Remote: origin",
		"Duration: 150ms",
		"Status: SUCCESS",
		"Repository: repo2",
		"Path: /test/repo2",
		"Branch: develop",
		"Remote: upstream",
		"Duration: 200ms",
		"Status: SUCCESS",
		"Repository: repo3",
		"Path: /test/repo3",
		"Status: FAILED - operation failed",
	}

	for _, expected := range expectedRepos {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Expected report to contain repository detail %q, got:\n%s", expected, contentStr)
		}
	}
}

func TestSaveReportDryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file I/O test in short mode")
	}

	tmpFile, err := os.CreateTemp("", "test-report-dry-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	cfg := config.DefaultConfig()
	cfg.SaveReport = tmpFile.Name()
	cfg.DryRun = true

	results := []types.GitRepo{
		{
			Path:     "/test/repo1",
			Name:     "repo1",
			Branch:   "main",
			Remote:   "origin",
			Duration: 150 * time.Millisecond,
		},
	}

	err = saveReport(cfg, results, 1, 0, 0)
	if err != nil {
		t.Errorf("saveReport() error = %v", err)
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "Status: DRY RUN - Would have succeeded") {
		t.Errorf("Expected dry run status in report, got:\n%s", contentStr)
	}
}

func TestSaveReportWithEmptyFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file I/O test in short mode")
	}

	tmpFile, err := os.CreateTemp("", "test-report-empty-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	cfg := config.DefaultConfig()
	cfg.SaveReport = tmpFile.Name()

	// Repository with empty branch and remote
	results := []types.GitRepo{
		{
			Path:     "/test/repo1",
			Name:     "repo1",
			Branch:   "", // Empty branch
			Remote:   "", // Empty remote
			Duration: 100 * time.Millisecond,
		},
	}

	err = saveReport(cfg, results, 1, 0, 0)
	if err != nil {
		t.Errorf("saveReport() error = %v", err)
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	contentStr := string(content)

	// Should contain the repository but not branch/remote lines
	if !strings.Contains(contentStr, "Repository: repo1") {
		t.Error("Expected report to contain repository name")
	}

	if !strings.Contains(contentStr, "Path: /test/repo1") {
		t.Error("Expected report to contain repository path")
	}

	// Should not contain empty branch or remote lines
	if strings.Contains(contentStr, "Branch: \n") {
		t.Error("Should not include empty branch line")
	}

	if strings.Contains(contentStr, "Remote: \n") {
		t.Error("Should not include empty remote line")
	}
}

func TestSaveReportCreateFileError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file I/O error test in short mode")
	}

	cfg := config.DefaultConfig()
	// Use an invalid path that should cause create to fail
	cfg.SaveReport = "/invalid/path/that/does/not/exist/report.txt"

	results := []types.GitRepo{
		{Path: "/test/repo1", Name: "repo1"},
	}

	err := saveReport(cfg, results, 1, 0, 0)
	if err == nil {
		t.Error("Expected error when creating file in invalid path")
	}

	if !strings.Contains(err.Error(), "failed to create report file") {
		t.Errorf("Expected error message about file creation, got: %v", err)
	}
}

func TestSaveReportEmptyResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file I/O test in short mode")
	}

	tmpFile, err := os.CreateTemp("", "test-report-empty-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	cfg := config.DefaultConfig()
	cfg.SaveReport = tmpFile.Name()

	// Empty results
	results := []types.GitRepo{}

	err = saveReport(cfg, results, 0, 0, 0)
	if err != nil {
		t.Errorf("saveReport() error = %v", err)
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	contentStr := string(content)

	// Should still contain header
	if !strings.Contains(contentStr, "Total Repositories: 0") {
		t.Error("Expected report to show 0 total repositories")
	}

	if !strings.Contains(contentStr, "Successful: 0, Failed: 0, Skipped: 0") {
		t.Error("Expected report to show all zero counts")
	}
}

func TestSaveReportLargeResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large data test in short mode")
	}

	tmpFile, err := os.CreateTemp("", "test-report-large-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	cfg := config.DefaultConfig()
	cfg.SaveReport = tmpFile.Name()

	// Create a large number of results
	results := make([]types.GitRepo, 1000)
	for i := 0; i < 1000; i++ {
		results[i] = types.GitRepo{
			Path:     fmt.Sprintf("/test/repo%d", i),
			Name:     fmt.Sprintf("repo%d", i),
			Branch:   "main",
			Remote:   "origin",
			Duration: time.Duration(i) * time.Millisecond,
		}
	}

	err = saveReport(cfg, results, 1000, 0, 0)
	if err != nil {
		t.Errorf("saveReport() error = %v", err)
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "Total Repositories: 1000") {
		t.Error("Expected report to show 1000 total repositories")
	}

	// Check that first and last repositories are included
	if !strings.Contains(contentStr, "Repository: repo0") {
		t.Error("Expected report to contain first repository")
	}

	if !strings.Contains(contentStr, "Repository: repo999") {
		t.Error("Expected report to contain last repository")
	}
}

func TestSaveReportDurationFormatting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file I/O test in short mode")
	}

	tmpFile, err := os.CreateTemp("", "test-report-duration-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	cfg := config.DefaultConfig()
	cfg.SaveReport = tmpFile.Name()

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "milliseconds",
			duration: 150 * time.Millisecond,
			expected: "150ms",
		},
		{
			name:     "seconds",
			duration: 2500 * time.Millisecond,
			expected: "2.5s",
		},
		{
			name:     "zero duration",
			duration: 0,
			expected: "0s",
		},
		{
			name:     "microseconds truncated",
			duration: 1500 * time.Microsecond,
			expected: "1ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := []types.GitRepo{
				{
					Path:     "/test/repo",
					Name:     "repo",
					Duration: tt.duration,
				},
			}

			err := saveReport(cfg, results, 1, 0, 0)
			if err != nil {
				t.Errorf("saveReport() error = %v", err)
			}

			content, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to read report file: %v", err)
			}

			if !strings.Contains(string(content), tt.expected) {
				t.Errorf("Expected report to contain duration %q, got:\n%s", tt.expected, content)
			}

			// Clean up for next test
			if err := tmpFile.Truncate(0); err != nil {
				t.Logf("Failed to truncate file: %v", err)
			}
			if _, err := tmpFile.Seek(0, 0); err != nil {
				t.Logf("Failed to seek file: %v", err)
			}
		})
	}
}

func TestSaveReportErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error handling test in short mode")
	}

	tmpFile, err := os.CreateTemp("", "test-report-error-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	cfg := config.DefaultConfig()
	cfg.SaveReport = tmpFile.Name()

	// Test various error types
	errorTests := []struct {
		name  string
		error error
	}{
		{
			name:  "simple error",
			error: errors.New("simple error message"),
		},
		{
			name:  "wrapped error",
			error: fmt.Errorf("wrapped: %w", errors.New("inner error")),
		},
		{
			name:  "long error message",
			error: errors.New("this is a very long error message that should be properly handled and not truncated or cause issues in the report generation"),
		},
		{
			name:  "empty error message",
			error: errors.New(""),
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			results := []types.GitRepo{
				{
					Path:  "/test/repo",
					Name:  "repo",
					Error: tt.error,
				},
			}

			err := saveReport(cfg, results, 0, 1, 0)
			if err != nil {
				t.Errorf("saveReport() error = %v", err)
			}

			content, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to read report file: %v", err)
			}

			contentStr := string(content)

			// Should contain the error message
			if !strings.Contains(contentStr, "Status: FAILED") {
				t.Error("Expected report to contain failed status")
			}

			if tt.error.Error() != "" && !strings.Contains(contentStr, tt.error.Error()) {
				t.Errorf("Expected report to contain error message %q, got:\n%s", tt.error.Error(), contentStr)
			}

			// Clean up for next test
			if err := tmpFile.Truncate(0); err != nil {
				t.Logf("Failed to truncate file: %v", err)
			}
			if _, err := tmpFile.Seek(0, 0); err != nil {
				t.Logf("Failed to seek file: %v", err)
			}
		})
	}
}

// Benchmark tests
func BenchmarkSaveReport(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	tmpFile, err := os.CreateTemp("", "bench-report-*.txt")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			b.Logf("Failed to close temp file: %v", err)
		}
		if err := os.Remove(tmpFile.Name()); err != nil {
			b.Logf("Failed to remove temp file: %v", err)
		}
	}()

	cfg := config.DefaultConfig()
	cfg.SaveReport = tmpFile.Name()

	// Create typical results
	results := make([]types.GitRepo, 100)
	for i := 0; i < 100; i++ {
		results[i] = types.GitRepo{
			Path:     fmt.Sprintf("/test/repo%d", i),
			Name:     fmt.Sprintf("repo%d", i),
			Branch:   "main",
			Remote:   "origin",
			Duration: time.Duration(i*10) * time.Millisecond,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Truncate file for each iteration
		if err := tmpFile.Truncate(0); err != nil {
			b.Fatalf("Failed to truncate file: %v", err)
		}
		if _, err := tmpFile.Seek(0, 0); err != nil {
			b.Fatalf("Failed to seek file: %v", err)
		}

		err := saveReport(cfg, results, 100, 0, 0)
		if err != nil {
			b.Errorf("saveReport() error = %v", err)
		}
	}
}

func BenchmarkSaveReportLarge(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping large benchmark in short mode")
	}

	tmpFile, err := os.CreateTemp("", "bench-report-large-*.txt")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			b.Logf("Failed to close temp file: %v", err)
		}
		if err := os.Remove(tmpFile.Name()); err != nil {
			b.Logf("Failed to remove temp file: %v", err)
		}
	}()

	cfg := config.DefaultConfig()
	cfg.SaveReport = tmpFile.Name()

	// Create large results set
	results := make([]types.GitRepo, 1000)
	for i := 0; i < 1000; i++ {
		results[i] = types.GitRepo{
			Path:     fmt.Sprintf("/test/repo%d", i),
			Name:     fmt.Sprintf("repo%d", i),
			Branch:   "main",
			Remote:   "origin",
			Duration: time.Duration(i*10) * time.Millisecond,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Truncate file for each iteration
		if err := tmpFile.Truncate(0); err != nil {
			b.Fatalf("Failed to truncate file: %v", err)
		}
		if _, err := tmpFile.Seek(0, 0); err != nil {
			b.Fatalf("Failed to seek file: %v", err)
		}

		err := saveReport(cfg, results, 1000, 0, 0)
		if err != nil {
			b.Errorf("saveReport() error = %v", err)
		}
	}
}
