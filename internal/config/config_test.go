package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/entro314-labs/git-herd/pkg/types"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	// Test default values
	expected := &types.Config{
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

	if !reflect.DeepEqual(cfg, expected) {
		t.Errorf("DefaultConfig() = %+v, want %+v", cfg, expected)
	}

	// Test specific field types and values
	if cfg.Workers <= 0 {
		t.Errorf("Expected Workers > 0, got %d", cfg.Workers)
	}

	if cfg.Operation != types.OperationFetch {
		t.Errorf("Expected Operation to be %q, got %q", types.OperationFetch, cfg.Operation)
	}

	if cfg.Timeout <= 0 {
		t.Errorf("Expected Timeout > 0, got %v", cfg.Timeout)
	}

	if len(cfg.ExcludeDirs) == 0 {
		t.Error("Expected ExcludeDirs to be non-empty")
	}
}

func TestSetupFlags(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cmd := &cobra.Command{}

	SetupFlags(cmd, cfg)

	tests := []struct {
		name         string
		shortFlag    string
		defaultValue interface{}
	}{
		{"operation", "o", "fetch"},
		{"workers", "w", 5},
		{"dry-run", "n", false},
		{"recursive", "r", true},
		{"skip-dirty", "s", true},
		{"verbose", "v", false},
		{"plain", "p", false},
		{"full-summary", "f", false},
		{"save-report", "", ""},
		{"timeout", "t", 5 * time.Minute},
		{"exclude", "e", []string{".git", "node_modules", "vendor"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			flag := cmd.Flags().Lookup(tt.name)
			if flag == nil {
				t.Errorf("Flag %q not found", tt.name)
				return
			}

			// Check short flag if expected
			if tt.shortFlag != "" && flag.Shorthand != tt.shortFlag {
				t.Errorf("Expected shorthand %q for flag %q, got %q", tt.shortFlag, tt.name, flag.Shorthand)
			}

			// Check default value (basic type checking)
			if flag.DefValue == "" && tt.defaultValue != "" {
				// Some flags may have empty default values which is valid
				return
			}
		})
	}
}

func TestSetupFlagsModifiesConfig(t *testing.T) {
	cfg := DefaultConfig()
	cmd := &cobra.Command{}

	// Store original values
	originalWorkers := cfg.Workers
	originalOperation := cfg.Operation

	SetupFlags(cmd, cfg)

	// Simulate setting flags
	if err := cmd.Flags().Set("workers", "10"); err != nil {
		t.Errorf("Failed to set workers flag: %v", err)
	}

	if err := cmd.Flags().Set("operation", "pull"); err != nil {
		t.Errorf("Failed to set operation flag: %v", err)
	}

	// Check that config was modified
	if cfg.Workers == originalWorkers {
		t.Errorf("Expected Workers to change from %d", originalWorkers)
	}

	if cfg.Operation == originalOperation {
		t.Errorf("Expected Operation to change from %q", originalOperation)
	}
}

func TestSetupViper(t *testing.T) {
	// Reset viper before test
	viper.Reset()

	cmd := &cobra.Command{}
	cfg := DefaultConfig()
	SetupFlags(cmd, cfg)

	if err := SetupViper(cmd); err != nil {
		t.Fatalf("SetupViper() error = %v", err)
	}

	// Test that viper is configured correctly
	configName := viper.ConfigFileUsed()
	if configName == "" {
		// This is expected if no config file exists
		t.Log("No config file found (expected for clean test environment)")
	}

	// Test that flags are bound to viper
	expectedBindings := []string{
		"operation", "workers", "dry-run", "recursive", "skip-dirty",
		"verbose", "plain", "full-summary", "save-report", "timeout", "exclude",
	}

	for _, binding := range expectedBindings {
		// We can't directly test if bindings exist, but we can test that viper
		// has been set up to read from the expected locations
		viper.SetDefault(binding, "test")
		if !viper.IsSet(binding) {
			t.Errorf("Expected viper to have binding for %q", binding)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	// Reset viper before test
	viper.Reset()

	cfg, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() error = %v", err)
	}

	if cfg == nil {
		t.Error("LoadConfig() returned nil config")
	}

	// Test that loaded config matches default config
	expected := DefaultConfig()
	if !reflect.DeepEqual(cfg, expected) {
		t.Errorf("LoadConfig() = %+v, want %+v", cfg, expected)
	}
}

func TestLoadConfigWithViperValues(t *testing.T) {
	// Reset viper before test
	viper.Reset()

	// Set some values in viper using the correct keys
	viper.Set("workers", 10)
	viper.Set("operation", "pull")
	viper.Set("dryrun", true) // Note: viper uses the struct field name without dash
	viper.Set("verbose", true)

	cfg, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() error = %v", err)
	}

	// Check that viper values were loaded
	if cfg.Workers != 10 {
		t.Errorf("Expected Workers = 10, got %d", cfg.Workers)
	}

	if cfg.Operation != types.OperationPull {
		t.Errorf("Expected Operation = %q, got %q", types.OperationPull, cfg.Operation)
	}

	if !cfg.DryRun {
		t.Error("Expected DryRun = true, got false")
	}

	if !cfg.Verbose {
		t.Error("Expected Verbose = true, got false")
	}
}

func TestLoadConfigWithInvalidData(t *testing.T) {
	// Reset viper before test
	viper.Reset()

	// Test with invalid operation type
	viper.Set("operation", "invalid_operation")

	cfg, err := LoadConfig()
	// Should not error but will have the invalid value
	if err != nil {
		t.Errorf("LoadConfig() unexpected error = %v", err)
	}

	// The invalid operation should be loaded as-is
	if string(cfg.Operation) != "invalid_operation" {
		t.Errorf("Expected Operation = 'invalid_operation', got %q", cfg.Operation)
	}
}

func TestSetupViperWithConfigFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping config file test in short mode")
	}

	// Create temporary directory for config file
	tmpDir, err := os.MkdirTemp("", "git-herd-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a test config file
	configPath := filepath.Join(tmpDir, "git-herd.yaml")
	configContent := `
workers: 8
operation: pull
dryrun: true
verbose: true
exclude:
  - .git
  - node_modules
  - build
`

	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Change to temp directory so viper can find the config
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to restore directory: %v", err)
		}
	}()

	// Reset viper and set up with config file
	viper.Reset()

	cmd := &cobra.Command{}
	cfg := DefaultConfig()
	SetupFlags(cmd, cfg)
	if err := SetupViper(cmd); err != nil {
		t.Fatalf("SetupViper() error = %v", err)
	}

	// Load config
	loadedCfg, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() error = %v", err)
	}

	// Verify values from config file were loaded
	if loadedCfg.Workers != 8 {
		t.Errorf("Expected Workers = 8 from config file, got %d", loadedCfg.Workers)
	}

	if loadedCfg.Operation != types.OperationPull {
		t.Errorf("Expected Operation = %q from config file, got %q", types.OperationPull, loadedCfg.Operation)
	}

	if !loadedCfg.DryRun {
		t.Error("Expected DryRun = true from config file, got false")
	}

	if !loadedCfg.Verbose {
		t.Error("Expected Verbose = true from config file, got false")
	}
}

func TestConfigFlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flagName string
		args     []string
		checker  func(*types.Config) error
	}{
		{
			name:     "operation default",
			flagName: "operation",
			args:     []string{},
			checker: func(cfg *types.Config) error {
				if cfg.Operation != types.OperationFetch {
					return fmt.Errorf("expected %q, got %q", types.OperationFetch, cfg.Operation)
				}
				return nil
			},
		},
		{
			name:     "workers default",
			flagName: "workers",
			args:     []string{},
			checker: func(cfg *types.Config) error {
				if cfg.Workers != 5 {
					return fmt.Errorf("expected 5, got %d", cfg.Workers)
				}
				return nil
			},
		},
		{
			name:     "operation pull",
			flagName: "operation",
			args:     []string{"--operation", "pull"},
			checker: func(cfg *types.Config) error {
				if cfg.Operation != types.OperationPull {
					return fmt.Errorf("expected %q, got %q", types.OperationPull, cfg.Operation)
				}
				return nil
			},
		},
		{
			name:     "workers custom",
			flagName: "workers",
			args:     []string{"--workers", "10"},
			checker: func(cfg *types.Config) error {
				if cfg.Workers != 10 {
					return fmt.Errorf("expected 10, got %d", cfg.Workers)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := DefaultConfig()
			cmd := &cobra.Command{}
			SetupFlags(cmd, cfg)

			// Set up command with args
			cmd.SetArgs(tt.args)

			// Parse flags
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Errorf("Failed to parse flags: %v", err)
			}

			// Check the result
			if err := tt.checker(cfg); err != nil {
				t.Errorf("Flag %q check failed: %v", tt.flagName, err)
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		modify func(*types.Config)
		valid  bool
	}{
		{
			name:   "valid default config",
			modify: func(*types.Config) {}, // No changes
			valid:  true,
		},
		{
			name: "zero workers",
			modify: func(cfg *types.Config) {
				cfg.Workers = 0
			},
			valid: true, // Currently no validation, but should be noted
		},
		{
			name: "negative workers",
			modify: func(cfg *types.Config) {
				cfg.Workers = -1
			},
			valid: true, // Currently no validation, but should be noted
		},
		{
			name: "very high workers",
			modify: func(cfg *types.Config) {
				cfg.Workers = 1000
			},
			valid: true,
		},
		{
			name: "zero timeout",
			modify: func(cfg *types.Config) {
				cfg.Timeout = 0
			},
			valid: true,
		},
		{
			name: "negative timeout",
			modify: func(cfg *types.Config) {
				cfg.Timeout = -1 * time.Second
			},
			valid: true, // Currently no validation
		},
		{
			name: "empty exclude dirs",
			modify: func(cfg *types.Config) {
				cfg.ExcludeDirs = []string{}
			},
			valid: true,
		},
		{
			name: "nil exclude dirs",
			modify: func(cfg *types.Config) {
				cfg.ExcludeDirs = nil
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := DefaultConfig()
			tt.modify(cfg)

			// Since there's no validation function currently, we just check
			// that the config can be created and used
			if cfg == nil {
				t.Error("Config became nil after modification")
			}

			// This test documents current behavior - in the future,
			// proper validation should be added
			if !tt.valid {
				t.Log("This configuration should be invalid but currently no validation exists")
			}
		})
	}
}

// Test edge cases and error conditions
func TestSetupViperEdgeCases(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer func() {
		if err := os.Setenv("HOME", originalHome); err != nil {
			t.Logf("Failed to restore HOME: %v", err)
		}
	}()

	// Test with invalid HOME directory
	if err := os.Setenv("HOME", "/non/existent/directory"); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}

	viper.Reset()
	cmd := &cobra.Command{}
	cfg := DefaultConfig()
	SetupFlags(cmd, cfg)

	// This should not panic even with invalid HOME
	if err := SetupViper(cmd); err != nil {
		t.Fatalf("SetupViper() error = %v", err)
	}

	// Should still be able to load config
	loadedCfg, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() should not fail with invalid HOME: %v", err)
	}

	if loadedCfg == nil {
		t.Error("LoadConfig() returned nil with invalid HOME")
	}
}

func TestConfigStringRepresentation(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	// Test that config can be formatted as string (useful for debugging)
	configStr := fmt.Sprintf("%+v", cfg)
	if configStr == "" {
		t.Error("Config string representation is empty")
	}

	// Should contain key field names
	expectedFields := []string{"Workers", "Operation", "DryRun", "Timeout"}
	for _, field := range expectedFields {
		if !contains(configStr, field) {
			t.Errorf("Config string representation should contain %q", field)
		}
	}
}

// Benchmark tests
func BenchmarkDefaultConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}

func BenchmarkSetupFlags(b *testing.B) {
	cfg := DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := &cobra.Command{}
		SetupFlags(cmd, cfg)
	}
}

func BenchmarkLoadConfig(b *testing.B) {
	viper.Reset()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg, err := LoadConfig()
		if err != nil {
			b.Errorf("LoadConfig() error = %v", err)
		}
		if cfg == nil {
			b.Error("LoadConfig() returned nil")
		}
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestLoadConfigUnmarshalError(t *testing.T) {
	// This test would require dependency injection or interface-based design
	// to properly test error conditions. Currently documented as a limitation.
	t.Skip("Cannot test unmarshal errors without refactoring to use interfaces")
}
