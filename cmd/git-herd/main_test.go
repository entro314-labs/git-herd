package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/entro314-labs/git-herd/internal/config"
)

func TestBuildVersion(t *testing.T) {
	// Note: Cannot use t.Parallel() on subtests because they modify global package variables
	tests := []struct {
		name     string
		version  string
		commit   string
		date     string
		builtBy  string
		expected string
	}{
		{
			name:     "development version",
			version:  "dev",
			commit:   "none",
			date:     "unknown",
			builtBy:  "unknown",
			expected: "dev (built from source)",
		},
		{
			name:     "release version with full info",
			version:  "v1.0.0",
			commit:   "abc123",
			date:     "2023-01-01T00:00:00Z",
			builtBy:  "goreleaser",
			expected: "v1.0.0 (commit: abc123, built: 2023-01-01T00:00:00Z, by: goreleaser)",
		},
		{
			name:     "release version with empty fields",
			version:  "v1.0.0",
			commit:   "",
			date:     "",
			builtBy:  "",
			expected: "v1.0.0 (commit: , built: , by: )",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			origVersion := version
			origCommit := commit
			origDate := date
			origBuiltBy := builtBy

			// Set test values
			version = tt.version
			commit = tt.commit
			date = tt.date
			builtBy = tt.builtBy

			// Test buildVersion function
			result := buildVersion()

			// Restore original values
			version = origVersion
			commit = origCommit
			date = origDate
			builtBy = origBuiltBy

			if result != tt.expected {
				t.Errorf("buildVersion() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRootCommandCreation(t *testing.T) {
	// Create a minimal test that verifies root command structure
	cfg := config.DefaultConfig()
	rootCmd := newRootCommand(cfg)

	// Test command properties
	if rootCmd.Use != "git-herd [path]" {
		t.Errorf("Expected Use to be 'git-herd [path]', got %q", rootCmd.Use)
	}

	if rootCmd.Short != "Bulk git operations on multiple repositories" {
		t.Errorf("Expected Short description to be 'Bulk git operations on multiple repositories', got %q", rootCmd.Short)
	}

	if !strings.Contains(rootCmd.Long, "git-herd performs git operations") {
		t.Errorf("Expected Long description to contain 'git-herd performs git operations', got %q", rootCmd.Long)
	}

	// Test that the command accepts maximum of 1 argument
	err := cobra.MaximumNArgs(1)(rootCmd, []string{"arg1", "arg2"})
	if err == nil {
		t.Error("Expected error for more than 1 argument, got nil")
	}

	err = cobra.MaximumNArgs(1)(rootCmd, []string{"arg1"})
	if err != nil {
		t.Errorf("Expected no error for 1 argument, got %v", err)
	}

	err = cobra.MaximumNArgs(1)(rootCmd, []string{})
	if err != nil {
		t.Errorf("Expected no error for 0 arguments, got %v", err)
	}
}

func TestRootCommandInvalidPath(t *testing.T) {
	cfg := config.DefaultConfig()
	rootCmd := newRootCommand(cfg)

	// Capture output
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Test with non-existent path
	rootCmd.SetArgs([]string{"--dry-run", "--plain", "/non/existent/path"})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("Expected error for non-existent path, got nil")
	}

	expectedError := "stat path /non/existent/path:"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain %q, got %q", expectedError, err.Error())
	}
}

func TestRootCommandValidPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "git-herd-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	cfg := config.DefaultConfig()
	rootCmd := newRootCommand(cfg)

	// Capture output
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Test with valid path
	rootCmd.SetArgs([]string{"--dry-run", "--plain", "--timeout", "1s", tmpDir})

	err = rootCmd.Execute()
	// We expect this to succeed (no error) even if no git repos are found
	if err != nil {
		t.Errorf("Expected no error for valid path, got %v", err)
	}
}

func TestRootCommandFlags(t *testing.T) {
	cfg := config.DefaultConfig()
	rootCmd := newRootCommand(cfg)

	// Test that flags are properly set up
	flags := rootCmd.Flags()

	expectedFlags := []string{
		"operation", "workers", "dry-run", "recursive", "skip-dirty",
		"verbose", "plain", "full-summary", "save-report", "timeout", "exclude", "discard-files", "export-scan",
	}

	for _, flagName := range expectedFlags {
		flag := flags.Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag %q to be defined", flagName)
		}
	}
}

func TestRootCommandVersion(t *testing.T) {
	cfg := config.DefaultConfig()
	rootCmd := newRootCommand(cfg)

	// Test version flag
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Expected no error for version flag, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, buildVersion()) {
		t.Errorf("Expected output to contain version info, got %q", output)
	}
}

func TestRootCommandHelp(t *testing.T) {
	cfg := config.DefaultConfig()
	rootCmd := newRootCommand(cfg)

	// Test help flag
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Expected no error for help flag, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Usage:") {
		t.Errorf("Expected output to contain usage info, got %q", output)
	}
}

func TestMainExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping main execution test in short mode")
	}

	// Test that main function doesn't panic
	// We'll use a subprocess approach to test main
	if os.Getenv("TEST_MAIN") != "1" {
		t.Skip("Skipping main execution test - use TEST_MAIN=1 to enable")
	}

	// This would be tested using exec.Command in a real scenario
	// For now, we just verify the main function exists and can be called
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("main() panicked: %v", r)
		}
	}()

	// We can't easily test main() directly without causing issues
	// This test exists to ensure the function is present and testable
	if testing.Verbose() {
		t.Log("main() function exists and is testable")
	}
}

func TestContextHandling(t *testing.T) {
	cfg := config.DefaultConfig()
	rootCmd := newRootCommand(cfg)

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "git-herd-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Capture output
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	rootCmd.SetArgs([]string{"--dry-run", "--plain", "--timeout", "100ms", tmpDir})

	// This should complete quickly due to dry run
	err = rootCmd.Execute()
	if err != nil {
		t.Errorf("Expected no error with timeout and dry run, got %v", err)
	}
}

func TestArgumentHandling(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		errMatch string
	}{
		{
			name:    "no arguments",
			args:    []string{"--dry-run", "--plain"},
			wantErr: false,
		},
		{
			name:    "one valid argument",
			args:    []string{"--dry-run", "--plain", "."},
			wantErr: false,
		},
		{
			name:     "non-existent path",
			args:     []string{"--dry-run", "--plain", "/non/existent/path"},
			wantErr:  true,
			errMatch: "stat path",
		},
		{
			name:     "too many arguments",
			args:     []string{"--dry-run", "--plain", "path1", "path2"},
			wantErr:  true,
			errMatch: "accepts at most 1 arg(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new config for each parallel test to avoid data races
			cfg := config.DefaultConfig()
			rootCmd := newRootCommand(cfg)
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			if tt.wantErr && err == nil {
				t.Errorf("Expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if tt.errMatch != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMatch) {
					t.Errorf("Expected error to contain %q, got %q", tt.errMatch, err.Error())
				}
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkBuildVersion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = buildVersion()
	}
}

func BenchmarkRootCommandCreation(b *testing.B) {
	cfg := config.DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = newRootCommand(cfg)
	}
}
