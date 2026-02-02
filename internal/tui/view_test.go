package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/entro314-labs/git-herd/internal/config"
	"github.com/entro314-labs/git-herd/pkg/types"
)

func TestModelView(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()

	// Test view in different phases
	tests := []struct {
		name              string
		setupModel        func(*Model)
		expectContains    []string
		expectNotContains []string
	}{
		{
			name: "initializing phase",
			setupModel: func(m *Model) {
				m.phase = "initializing"
			},
			expectContains: []string{
				"git-herd",
				"Fetch Operation",
				"Scanning for Git repositories",
				"/test/path",
				"Press 'q' or Ctrl+C to quit",
			},
		},
		{
			name: "scanning phase",
			setupModel: func(m *Model) {
				m.phase = "scanning"
			},
			expectContains: []string{
				"Scanning for Git repositories",
				"/test/path",
			},
		},
		{
			name: "processing phase with repos",
			setupModel: func(m *Model) {
				m.phase = "processing"
				m.repos = []types.GitRepo{
					{Path: "/test/repo1", Name: "repo1"},
					{Path: "/test/repo2", Name: "repo2"},
				}
				m.processed = 1
				m.results = []types.GitRepo{
					{
						Path:     "/test/repo1",
						Name:     "repo1",
						Branch:   "main",
						Remote:   "origin",
						Duration: 150 * time.Millisecond,
					},
				}
			},
			expectContains: []string{
				"Processing repositories",
				"(1/2)",
				"✓",
				"repo1",
				"main@origin",
				"150ms",
			},
		},
		{
			name: "processing phase with error",
			setupModel: func(m *Model) {
				m.phase = "processing"
				m.repos = []types.GitRepo{
					{Path: "/test/repo1", Name: "repo1"},
				}
				m.processed = 1
				m.results = []types.GitRepo{
					{
						Path:  "/test/repo1",
						Name:  "repo1",
						Error: &testError{msg: "operation failed"},
					},
				}
			},
			expectContains: []string{
				"Processing repositories",
				"✗",
				"repo1",
				"operation failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a copy of the model for this test
			testModel := NewModel(cfg, "/test/path")
			tt.setupModel(testModel)

			view := testModel.View()

			for _, expected := range tt.expectContains {
				if !strings.Contains(view, expected) {
					t.Errorf("Expected view to contain %q, got:\n%s", expected, view)
				}
			}

			for _, notExpected := range tt.expectNotContains {
				if strings.Contains(view, notExpected) {
					t.Errorf("Expected view to not contain %q, got:\n%s", notExpected, view)
				}
			}
		})
	}
}

func TestModelViewDone(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")
	model.done = true

	view := model.View()

	// When done, should call renderSummary
	// We can't test the exact content without setting up results,
	// but we can verify it doesn't show the regular processing view
	if strings.Contains(view, "Press 'q' or Ctrl+C to quit") {
		t.Error("Done view should not contain quit instructions")
	}
}

func TestModelRenderSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupModel     func(*Model)
		expectContains []string
	}{
		{
			name: "no repositories found",
			setupModel: func(m *Model) {
				m.repos = []types.GitRepo{}
				m.results = []types.GitRepo{}
			},
			expectContains: []string{
				"git-herd",
				"No Git repositories found",
				"/test/path",
			},
		},
		{
			name: "successful results",
			setupModel: func(m *Model) {
				m.repos = []types.GitRepo{
					{Path: "/test/repo1", Name: "repo1"},
					{Path: "/test/repo2", Name: "repo2"},
				}
				m.results = []types.GitRepo{
					{
						Path:     "/test/repo1",
						Name:     "repo1",
						Branch:   "main",
						Remote:   "origin",
						Duration: 150 * time.Millisecond,
					},
					{
						Path:     "/test/repo2",
						Name:     "repo2",
						Branch:   "develop",
						Remote:   "origin",
						Duration: 200 * time.Millisecond,
					},
				}
			},
			expectContains: []string{
				"🎉 git-herd Results",
				"✓",
				"repo1",
				"/test/repo1",
				"main@origin",
				"150ms",
				"repo2",
				"/test/repo2",
				"develop@origin",
				"200ms",
				"📊 Summary:",
				"2 successful",
				"0 failed",
				"0 skipped",
				"2 total",
			},
		},
		{
			name: "mixed results with errors",
			setupModel: func(m *Model) {
				m.repos = []types.GitRepo{
					{Path: "/test/repo1", Name: "repo1"},
					{Path: "/test/repo2", Name: "repo2"},
				}
				m.results = []types.GitRepo{
					{
						Path:     "/test/repo1",
						Name:     "repo1",
						Branch:   "main",
						Remote:   "origin",
						Duration: 150 * time.Millisecond,
					},
					{
						Path:  "/test/repo2",
						Name:  "repo2",
						Error: &testError{msg: "operation failed"},
					},
				}
			},
			expectContains: []string{
				"✓",
				"repo1",
				"✗",
				"repo2",
				"operation failed",
				"1 successful",
				"1 failed",
				"0 skipped",
			},
		},
		{
			name: "skipped repositories",
			setupModel: func(m *Model) {
				m.repos = []types.GitRepo{
					{Path: "/test/repo1", Name: "repo1"},
					{Path: "/test/repo2", Name: "repo2"},
				}
				m.results = []types.GitRepo{
					{
						Path:     "/test/repo1",
						Name:     "repo1",
						Branch:   "main",
						Remote:   "origin",
						Duration: 150 * time.Millisecond,
					},
					{
						Path:  "/test/repo2",
						Name:  "repo2",
						Error: &testError{msg: "skipped: dirty working directory"},
					},
				}
			},
			expectContains: []string{
				"✓",
				"repo1",
				"⊝",
				"repo2",
				"skipped: dirty working directory",
				"1 successful",
				"0 failed",
				"1 skipped",
			},
		},
		{
			name: "dry run mode",
			setupModel: func(m *Model) {
				m.config.DryRun = true
				m.repos = []types.GitRepo{
					{Path: "/test/repo1", Name: "repo1"},
				}
				m.results = []types.GitRepo{
					{
						Path:     "/test/repo1",
						Name:     "repo1",
						Branch:   "main",
						Remote:   "origin",
						Duration: 150 * time.Millisecond,
					},
				}
			},
			expectContains: []string{
				"👁", // Dry run icon
				"repo1",
				"1 successful",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.DefaultConfig()
			model := NewModel(cfg, "/test/path")
			model.done = true
			tt.setupModel(model)

			summary := model.renderSummary()

			for _, expected := range tt.expectContains {
				if !strings.Contains(summary, expected) {
					t.Errorf("Expected summary to contain %q, got:\n%s", expected, summary)
				}
			}
		})
	}
}

func TestModelRenderSummaryWithReport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file I/O test in short mode")
	}

	cfg := config.DefaultConfig()

	// Create temporary file for report
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

	cfg.SaveReport = tmpFile.Name()

	model := NewModel(cfg, "/test/path")
	model.done = true
	model.repos = []types.GitRepo{
		{Path: "/test/repo1", Name: "repo1"},
	}
	model.results = []types.GitRepo{
		{
			Path:     "/test/repo1",
			Name:     "repo1",
			Branch:   "main",
			Remote:   "origin",
			Duration: 150 * time.Millisecond,
		},
	}

	summary := model.renderSummary()

	// Should mention the report file
	if !strings.Contains(summary, "Detailed report saved to:") {
		t.Error("Expected summary to mention saved report")
	}

	if !strings.Contains(summary, tmpFile.Name()) {
		t.Error("Expected summary to contain report file path")
	}
}

func TestViewStyles(t *testing.T) {
	t.Parallel()

	// Test that styles are properly initialized and don't panic when used
	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")

	// Test different view states to exercise style usage
	model.phase = "initializing"
	view1 := model.View()
	if view1 == "" {
		t.Error("View should not be empty")
	}

	model.phase = "processing"
	model.repos = []types.GitRepo{{Path: "/test", Name: "test"}}
	view2 := model.View()
	if view2 == "" {
		t.Error("Processing view should not be empty")
	}

	model.done = true
	view3 := model.View()
	if view3 == "" {
		t.Error("Done view should not be empty")
	}
}

func TestViewProgress(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")
	model.phase = "processing"
	model.repos = make([]types.GitRepo, 10)
	for i := 0; i < 10; i++ {
		model.repos[i] = types.GitRepo{
			Path: fmt.Sprintf("/test/repo%d", i),
			Name: fmt.Sprintf("repo%d", i),
		}
	}

	// Test progress display at different completion levels
	testCases := []struct {
		processed int
		expected  float64
	}{
		{0, 0.0},
		{5, 0.5},
		{10, 1.0},
	}

	for _, tc := range testCases {
		model.processed = tc.processed
		view := model.View()

		// View should contain progress information
		if !strings.Contains(view, fmt.Sprintf("(%d/10)", tc.processed)) {
			t.Errorf("Expected view to show progress (%d/10)", tc.processed)
		}
	}
}

func TestViewRecentResults(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")
	model.phase = "processing"
	model.repos = make([]types.GitRepo, 5)

	// Add more than 3 results to test the "recent results" limiting
	model.results = []types.GitRepo{
		{Name: "repo1", Branch: "main", Remote: "origin", Duration: 100 * time.Millisecond},
		{Name: "repo2", Branch: "main", Remote: "origin", Duration: 150 * time.Millisecond},
		{Name: "repo3", Branch: "main", Remote: "origin", Duration: 200 * time.Millisecond},
		{Name: "repo4", Branch: "main", Remote: "origin", Duration: 250 * time.Millisecond},
		{Name: "repo5", Branch: "main", Remote: "origin", Duration: 300 * time.Millisecond},
	}

	view := model.View()

	// Should only show the last 3 results
	if strings.Contains(view, "repo1") {
		t.Error("Should not show oldest result when more than 3 results exist")
	}

	if strings.Contains(view, "repo2") {
		t.Error("Should not show second oldest result when more than 3 results exist")
	}

	// Should show the most recent 3
	expectedRecent := []string{"repo3", "repo4", "repo5"}
	for _, repo := range expectedRecent {
		if !strings.Contains(view, repo) {
			t.Errorf("Should show recent result %s", repo)
		}
	}
}

func TestViewOperationTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		operation types.OperationType
		expected  string
	}{
		{types.OperationFetch, "Fetch Operation"},
		{types.OperationPull, "Pull Operation"},
	}

	for _, tt := range tests {
		t.Run(string(tt.operation), func(t *testing.T) {
			t.Parallel()

			cfg := config.DefaultConfig()
			cfg.Operation = tt.operation
			model := NewModel(cfg, "/test/path")

			view := model.View()

			if !strings.Contains(view, tt.expected) {
				t.Errorf("Expected view to contain %q for operation %s", tt.expected, tt.operation)
			}
		})
	}
}

func TestViewEmptyStates(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")

	// Test view with no repos in processing phase
	model.phase = "processing"
	model.repos = []types.GitRepo{}

	view := model.View()
	if view == "" {
		t.Error("View should not be empty even with no repos")
	}

	// Test view with repos but no results yet
	model.repos = []types.GitRepo{{Path: "/test", Name: "test"}}
	model.results = []types.GitRepo{}

	view = model.View()
	if !strings.Contains(view, "Processing repositories") {
		t.Error("Should show processing message when repos exist but no results yet")
	}
}

// Test helper for error interface
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// Benchmark tests
func BenchmarkModelView(b *testing.B) {
	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")
	defer model.cancel()

	// Set up a typical processing state
	model.phase = "processing"
	model.repos = make([]types.GitRepo, 10)
	model.results = make([]types.GitRepo, 5)
	for i := 0; i < 5; i++ {
		model.results[i] = types.GitRepo{
			Name:     fmt.Sprintf("repo%d", i),
			Branch:   "main",
			Remote:   "origin",
			Duration: time.Duration(i*100) * time.Millisecond,
		}
	}
	model.processed = 5

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.View()
	}
}

func BenchmarkModelRenderSummary(b *testing.B) {
	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")
	defer model.cancel()

	// Set up results for summary
	model.done = true
	model.repos = make([]types.GitRepo, 100)
	model.results = make([]types.GitRepo, 100)
	for i := 0; i < 100; i++ {
		model.results[i] = types.GitRepo{
			Path:     fmt.Sprintf("/test/repo%d", i),
			Name:     fmt.Sprintf("repo%d", i),
			Branch:   "main",
			Remote:   "origin",
			Duration: time.Duration(i*10) * time.Millisecond,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.renderSummary()
	}
}
