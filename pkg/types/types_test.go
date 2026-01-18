package types

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestOperationType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation OperationType
		expected  string
	}{
		{
			name:      "fetch operation",
			operation: OperationFetch,
			expected:  "fetch",
		},
		{
			name:      "pull operation",
			operation: OperationPull,
			expected:  "pull",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if string(tt.operation) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(tt.operation))
			}
		})
	}
}

func TestOperationTypeConstants(t *testing.T) {
	t.Parallel()

	// Test that constants are properly defined
	if OperationFetch != "fetch" {
		t.Errorf("OperationFetch should be 'fetch', got %q", OperationFetch)
	}

	if OperationPull != "pull" {
		t.Errorf("OperationPull should be 'pull', got %q", OperationPull)
	}

	// Test that constants are different
	if OperationFetch == OperationPull {
		t.Error("OperationFetch and OperationPull should be different")
	}
}

func TestGitRepo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		repo GitRepo
	}{
		{
			name: "empty repo",
			repo: GitRepo{},
		},
		{
			name: "basic repo",
			repo: GitRepo{
				Path:   "/path/to/repo",
				Name:   "test-repo",
				HasGit: true,
				Clean:  true,
				Branch: "main",
				Remote: "origin",
			},
		},
		{
			name: "repo with error",
			repo: GitRepo{
				Path:   "/path/to/repo",
				Name:   "test-repo",
				HasGit: false,
				Clean:  false,
				Error:  errors.New("test error"),
			},
		},
		{
			name: "repo with duration",
			repo: GitRepo{
				Path:     "/path/to/repo",
				Name:     "test-repo",
				HasGit:   true,
				Clean:    true,
				Duration: 5 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := tt.repo

			// Test field access
			if repo.Path != tt.repo.Path {
				t.Errorf("Expected Path %q, got %q", tt.repo.Path, repo.Path)
			}

			if repo.Name != tt.repo.Name {
				t.Errorf("Expected Name %q, got %q", tt.repo.Name, repo.Name)
			}

			if repo.HasGit != tt.repo.HasGit {
				t.Errorf("Expected HasGit %v, got %v", tt.repo.HasGit, repo.HasGit)
			}

			if repo.Clean != tt.repo.Clean {
				t.Errorf("Expected Clean %v, got %v", tt.repo.Clean, repo.Clean)
			}

			if repo.Branch != tt.repo.Branch {
				t.Errorf("Expected Branch %q, got %q", tt.repo.Branch, repo.Branch)
			}

			if repo.Remote != tt.repo.Remote {
				t.Errorf("Expected Remote %q, got %q", tt.repo.Remote, repo.Remote)
			}

			if repo.Duration != tt.repo.Duration {
				t.Errorf("Expected Duration %v, got %v", tt.repo.Duration, repo.Duration)
			}

			// Test error handling
			if (repo.Error == nil) != (tt.repo.Error == nil) {
				t.Errorf("Expected Error presence %v, got %v", tt.repo.Error != nil, repo.Error != nil)
			}

			if repo.Error != nil && tt.repo.Error != nil {
				if repo.Error.Error() != tt.repo.Error.Error() {
					t.Errorf("Expected Error %q, got %q", tt.repo.Error.Error(), repo.Error.Error())
				}
			}
		})
	}
}

func TestConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "empty config",
			config: Config{},
		},
		{
			name: "full config",
			config: Config{
				Workers:     10,
				Operation:   OperationPull,
				DryRun:      true,
				Recursive:   true,
				SkipDirty:   false,
				Verbose:     true,
				Timeout:     10 * time.Minute,
				ExcludeDirs: []string{".git", "node_modules"},
				PlainMode:   true,
				FullSummary: true,
				SaveReport:  "/path/to/report.txt",
			},
		},
		{
			name: "config with zero values",
			config: Config{
				Workers:     0,
				Operation:   "",
				DryRun:      false,
				Recursive:   false,
				SkipDirty:   false,
				Verbose:     false,
				Timeout:     0,
				ExcludeDirs: nil,
				PlainMode:   false,
				FullSummary: false,
				SaveReport:  "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := tt.config

			// Test all field access
			if config.Workers != tt.config.Workers {
				t.Errorf("Expected Workers %d, got %d", tt.config.Workers, config.Workers)
			}

			if config.Operation != tt.config.Operation {
				t.Errorf("Expected Operation %q, got %q", tt.config.Operation, config.Operation)
			}

			if config.DryRun != tt.config.DryRun {
				t.Errorf("Expected DryRun %v, got %v", tt.config.DryRun, config.DryRun)
			}

			if config.Recursive != tt.config.Recursive {
				t.Errorf("Expected Recursive %v, got %v", tt.config.Recursive, config.Recursive)
			}

			if config.SkipDirty != tt.config.SkipDirty {
				t.Errorf("Expected SkipDirty %v, got %v", tt.config.SkipDirty, config.SkipDirty)
			}

			if config.Verbose != tt.config.Verbose {
				t.Errorf("Expected Verbose %v, got %v", tt.config.Verbose, config.Verbose)
			}

			if config.Timeout != tt.config.Timeout {
				t.Errorf("Expected Timeout %v, got %v", tt.config.Timeout, config.Timeout)
			}

			if config.PlainMode != tt.config.PlainMode {
				t.Errorf("Expected PlainMode %v, got %v", tt.config.PlainMode, config.PlainMode)
			}

			if config.FullSummary != tt.config.FullSummary {
				t.Errorf("Expected FullSummary %v, got %v", tt.config.FullSummary, config.FullSummary)
			}

			if config.SaveReport != tt.config.SaveReport {
				t.Errorf("Expected SaveReport %q, got %q", tt.config.SaveReport, config.SaveReport)
			}

			// Test slice comparison
			if len(config.ExcludeDirs) != len(tt.config.ExcludeDirs) {
				t.Errorf("Expected ExcludeDirs length %d, got %d", len(tt.config.ExcludeDirs), len(config.ExcludeDirs))
			}

			for i, expected := range tt.config.ExcludeDirs {
				if i < len(config.ExcludeDirs) && config.ExcludeDirs[i] != expected {
					t.Errorf("Expected ExcludeDirs[%d] %q, got %q", i, expected, config.ExcludeDirs[i])
				}
			}
		})
	}
}

func TestGitRepoResult(t *testing.T) {
	t.Parallel()

	now := time.Now()
	later := now.Add(5 * time.Second)

	tests := []struct {
		name   string
		result GitRepoResult
	}{
		{
			name:   "empty result",
			result: GitRepoResult{},
		},
		{
			name: "successful result",
			result: GitRepoResult{
				Repo: GitRepo{
					Path:   "/path/to/repo",
					Name:   "test-repo",
					HasGit: true,
					Clean:  true,
				},
				Success:   true,
				Skipped:   false,
				StartTime: now,
				EndTime:   later,
			},
		},
		{
			name: "failed result",
			result: GitRepoResult{
				Repo: GitRepo{
					Path:  "/path/to/repo",
					Name:  "test-repo",
					Error: errors.New("operation failed"),
				},
				Success:   false,
				Skipped:   false,
				StartTime: now,
				EndTime:   later,
			},
		},
		{
			name: "skipped result",
			result: GitRepoResult{
				Repo: GitRepo{
					Path:   "/path/to/repo",
					Name:   "test-repo",
					HasGit: true,
					Clean:  false, // Dirty repo
				},
				Success:   false,
				Skipped:   true,
				StartTime: now,
				EndTime:   now, // Same time for skipped
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.result

			// Test field access
			if result.Success != tt.result.Success {
				t.Errorf("Expected Success %v, got %v", tt.result.Success, result.Success)
			}

			if result.Skipped != tt.result.Skipped {
				t.Errorf("Expected Skipped %v, got %v", tt.result.Skipped, result.Skipped)
			}

			if !result.StartTime.Equal(tt.result.StartTime) {
				t.Errorf("Expected StartTime %v, got %v", tt.result.StartTime, result.StartTime)
			}

			if !result.EndTime.Equal(tt.result.EndTime) {
				t.Errorf("Expected EndTime %v, got %v", tt.result.EndTime, result.EndTime)
			}

			// Test embedded repo
			if result.Repo.Path != tt.result.Repo.Path {
				t.Errorf("Expected Repo.Path %q, got %q", tt.result.Repo.Path, result.Repo.Path)
			}
		})
	}
}

func TestProcessingStats(t *testing.T) {
	t.Parallel()

	now := time.Now()
	later := now.Add(10 * time.Second)

	tests := []struct {
		name  string
		stats ProcessingStats
	}{
		{
			name:  "empty stats",
			stats: ProcessingStats{},
		},
		{
			name: "basic stats",
			stats: ProcessingStats{
				Total:      10,
				Successful: 8,
				Failed:     1,
				Skipped:    1,
				StartTime:  now,
				EndTime:    later,
			},
		},
		{
			name: "all failed stats",
			stats: ProcessingStats{
				Total:      5,
				Successful: 0,
				Failed:     5,
				Skipped:    0,
				StartTime:  now,
				EndTime:    later,
			},
		},
		{
			name: "all skipped stats",
			stats: ProcessingStats{
				Total:      3,
				Successful: 0,
				Failed:     0,
				Skipped:    3,
				StartTime:  now,
				EndTime:    now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stats := tt.stats

			// Test field access
			if stats.Total != tt.stats.Total {
				t.Errorf("Expected Total %d, got %d", tt.stats.Total, stats.Total)
			}

			if stats.Successful != tt.stats.Successful {
				t.Errorf("Expected Successful %d, got %d", tt.stats.Successful, stats.Successful)
			}

			if stats.Failed != tt.stats.Failed {
				t.Errorf("Expected Failed %d, got %d", tt.stats.Failed, stats.Failed)
			}

			if stats.Skipped != tt.stats.Skipped {
				t.Errorf("Expected Skipped %d, got %d", tt.stats.Skipped, stats.Skipped)
			}

			if !stats.StartTime.Equal(tt.stats.StartTime) {
				t.Errorf("Expected StartTime %v, got %v", tt.stats.StartTime, stats.StartTime)
			}

			if !stats.EndTime.Equal(tt.stats.EndTime) {
				t.Errorf("Expected EndTime %v, got %v", tt.stats.EndTime, stats.EndTime)
			}

			// Test invariant: Total should equal sum of Successful, Failed, and Skipped
			sum := stats.Successful + stats.Failed + stats.Skipped
			if stats.Total > 0 && sum != stats.Total {
				t.Errorf("Expected Total (%d) to equal sum of Successful+Failed+Skipped (%d)", stats.Total, sum)
			}
		})
	}
}

func TestProcessingStatsSummary(t *testing.T) {
	t.Parallel()

	stats := ProcessingStats{
		Total:      10,
		Successful: 8,
		Failed:     1,
		Skipped:    1,
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(5 * time.Second),
	}

	summary := stats.Summary()

	// Currently the Summary method returns empty string
	// This test documents the current behavior and should be updated
	// when the Summary method is implemented
	if summary != "" {
		t.Errorf("Expected empty summary (not implemented), got %q", summary)
	}

	// Test that Summary method doesn't panic
	emptyStats := ProcessingStats{}
	emptySummary := emptyStats.Summary()
	if emptySummary != "" {
		t.Errorf("Expected empty summary for empty stats, got %q", emptySummary)
	}
}

func TestTypeStringConversions(t *testing.T) {
	t.Parallel()

	// Test OperationType string conversion
	fetch := OperationFetch
	pull := OperationPull

	if string(fetch) != "fetch" {
		t.Errorf("Expected 'fetch', got %q", string(fetch))
	}

	if string(pull) != "pull" {
		t.Errorf("Expected 'pull', got %q", string(pull))
	}

	// Test custom OperationType values
	custom := OperationType("custom")
	if string(custom) != "custom" {
		t.Errorf("Expected 'custom', got %q", string(custom))
	}
}

func TestTypeComparison(t *testing.T) {
	t.Parallel()

	// Test OperationType comparison
	if OperationFetch == OperationPull {
		t.Error("OperationFetch should not equal OperationPull")
	}

	// Test with variables
	op1 := OperationFetch
	op2 := OperationFetch
	op3 := OperationPull

	if op1 != op2 {
		t.Error("Same operations should be equal")
	}

	if op1 == op3 {
		t.Error("Different operations should not be equal")
	}
}

func TestGitRepoZeroValues(t *testing.T) {
	t.Parallel()

	var repo GitRepo

	// Test zero values
	if repo.Path != "" {
		t.Errorf("Expected empty Path, got %q", repo.Path)
	}

	if repo.Name != "" {
		t.Errorf("Expected empty Name, got %q", repo.Name)
	}

	if repo.HasGit {
		t.Error("Expected HasGit to be false")
	}

	if repo.Clean {
		t.Error("Expected Clean to be false")
	}

	if repo.Branch != "" {
		t.Errorf("Expected empty Branch, got %q", repo.Branch)
	}

	if repo.Remote != "" {
		t.Errorf("Expected empty Remote, got %q", repo.Remote)
	}

	if repo.Error != nil {
		t.Errorf("Expected nil Error, got %v", repo.Error)
	}

	if repo.Duration != 0 {
		t.Errorf("Expected zero Duration, got %v", repo.Duration)
	}
}

func TestConfigZeroValues(t *testing.T) {
	t.Parallel()

	var config Config

	// Test zero values
	if config.Workers != 0 {
		t.Errorf("Expected Workers to be 0, got %d", config.Workers)
	}

	if config.Operation != "" {
		t.Errorf("Expected empty Operation, got %q", config.Operation)
	}

	if config.DryRun {
		t.Error("Expected DryRun to be false")
	}

	if config.Recursive {
		t.Error("Expected Recursive to be false")
	}

	if config.SkipDirty {
		t.Error("Expected SkipDirty to be false")
	}

	if config.Verbose {
		t.Error("Expected Verbose to be false")
	}

	if config.Timeout != 0 {
		t.Errorf("Expected Timeout to be 0, got %v", config.Timeout)
	}

	if config.ExcludeDirs != nil {
		t.Errorf("Expected ExcludeDirs to be nil, got %v", config.ExcludeDirs)
	}

	if config.PlainMode {
		t.Error("Expected PlainMode to be false")
	}

	if config.FullSummary {
		t.Error("Expected FullSummary to be false")
	}

	if config.SaveReport != "" {
		t.Errorf("Expected empty SaveReport, got %q", config.SaveReport)
	}
}

// Benchmark tests
func BenchmarkOperationTypeString(b *testing.B) {
	op := OperationFetch
	for i := 0; i < b.N; i++ {
		_ = string(op)
	}
}

func BenchmarkGitRepoCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GitRepo{
			Path:   "/path/to/repo",
			Name:   "test-repo",
			HasGit: true,
			Clean:  true,
			Branch: "main",
			Remote: "origin",
		}
	}
}

func BenchmarkConfigCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Config{
			Workers:     5,
			Operation:   OperationFetch,
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
}

func BenchmarkProcessingStatsCalculation(b *testing.B) {
	stats := ProcessingStats{
		Total:      1000,
		Successful: 800,
		Failed:     150,
		Skipped:    50,
		StartTime:  time.Now().Add(-10 * time.Minute),
		EndTime:    time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate calculations that might be done with stats
		_ = stats.Total - stats.Successful - stats.Failed - stats.Skipped
		_ = float64(stats.Successful) / float64(stats.Total) * 100
		_ = stats.EndTime.Sub(stats.StartTime)
	}
}

// Test struct field tags if they exist (they don't currently, but this documents the expectation)
func TestStructTags(t *testing.T) {
	t.Parallel()

	// This test documents that struct tags might be useful for serialization
	// but are not currently implemented
	t.Log("Struct tags are not currently implemented but might be useful for JSON/YAML serialization")

	// Future enhancement: Add struct tags like:
	// `json:"workers" yaml:"workers"`
	// `json:"operation" yaml:"operation"`
	// etc.
}

func TestOperationTypeValidation(t *testing.T) {
	t.Parallel()

	validOperations := []OperationType{OperationFetch, OperationPull}

	for _, op := range validOperations {
		switch op {
		case OperationFetch, OperationPull:
			// Valid operations
		default:
			t.Errorf("Unexpected operation type: %q", op)
		}
	}

	// Test invalid operation handling (currently no validation exists)
	invalidOp := OperationType("invalid")
	if string(invalidOp) != "invalid" {
		t.Errorf("Invalid operation should preserve its value, got %q", string(invalidOp))
	}

	// This documents that validation should be added in the future
	t.Log("Operation type validation should be implemented to reject invalid operations")
}

func TestDurationHandling(t *testing.T) {
	t.Parallel()

	// Test various duration values in GitRepo
	durations := []time.Duration{
		0,
		time.Millisecond,
		time.Second,
		time.Minute,
		time.Hour,
		-time.Second, // Negative duration (edge case)
	}

	for _, d := range durations {
		repo := GitRepo{Duration: d}
		if repo.Duration != d {
			t.Errorf("Expected Duration %v, got %v", d, repo.Duration)
		}
	}

	// Test duration in Config
	config := Config{Timeout: 5 * time.Minute}
	if config.Timeout != 5*time.Minute {
		t.Errorf("Expected Timeout %v, got %v", 5*time.Minute, config.Timeout)
	}
}

func TestSliceHandling(t *testing.T) {
	t.Parallel()

	// Test ExcludeDirs slice handling
	tests := []struct {
		name        string
		excludeDirs []string
	}{
		{
			name:        "nil slice",
			excludeDirs: nil,
		},
		{
			name:        "empty slice",
			excludeDirs: []string{},
		},
		{
			name:        "single item",
			excludeDirs: []string{".git"},
		},
		{
			name:        "multiple items",
			excludeDirs: []string{".git", "node_modules", "vendor", "build"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := Config{ExcludeDirs: tt.excludeDirs}

			if len(config.ExcludeDirs) != len(tt.excludeDirs) {
				t.Errorf("Expected ExcludeDirs length %d, got %d", len(tt.excludeDirs), len(config.ExcludeDirs))
			}

			for i, expected := range tt.excludeDirs {
				if config.ExcludeDirs[i] != expected {
					t.Errorf("Expected ExcludeDirs[%d] = %q, got %q", i, expected, config.ExcludeDirs[i])
				}
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	t.Parallel()

	// Test GitRepo with various error types
	errors := []error{
		nil,
		fmt.Errorf("simple error"),
		fmt.Errorf("wrapped error: %w", fmt.Errorf("inner error")),
		errors.New("standard error"),
	}

	for i, err := range errors {
		t.Run(fmt.Sprintf("error_%d", i), func(t *testing.T) {
			t.Parallel()

			repo := GitRepo{Error: err}

			if (repo.Error == nil) != (err == nil) {
				t.Errorf("Expected Error nil status %v, got %v", err == nil, repo.Error == nil)
			}

			if repo.Error != nil && err != nil {
				if repo.Error.Error() != err.Error() {
					t.Errorf("Expected Error %q, got %q", err.Error(), repo.Error.Error())
				}
			}
		})
	}
}
