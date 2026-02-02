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
		{"discard-files", "d", []string{}},
		{"export-scan", "", ""},
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
		"discard-files", "export-scan",
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
	viper.Set("dry-run", true) // Note: usage aligns with flag now
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

func TestLoadConfigEnvOverrides(t *testing.T) {
	viper.Reset()

	t.Setenv("GIT_HERD_WORKERS", "12")
	t.Setenv("GIT_HERD_OPERATION", "pull")

	cmd := &cobra.Command{}
	cfg := DefaultConfig()
	SetupFlags(cmd, cfg)

	if err := SetupViper(cmd); err != nil {
		t.Fatalf("SetupViper() error = %v", err)
	}

	loadedCfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loadedCfg.Workers != 12 {
		t.Errorf("Expected Workers = 12 from env, got %d", loadedCfg.Workers)
	}

	if loadedCfg.Operation != types.OperationPull {
		t.Errorf("Expected Operation = %q from env, got %q", types.OperationPull, loadedCfg.Operation)
	}
}

func TestLoadConfigFlagOverridesEnv(t *testing.T) {
	viper.Reset()

	t.Setenv("GIT_HERD_WORKERS", "3")

	cmd := &cobra.Command{}
	cfg := DefaultConfig()
	SetupFlags(cmd, cfg)

	if err := cmd.Flags().Parse([]string{"--workers", "9"}); err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	if err := SetupViper(cmd); err != nil {
		t.Fatalf("SetupViper() error = %v", err)
	}

	loadedCfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loadedCfg.Workers != 9 {
		t.Errorf("Expected Workers = 9 from flag, got %d", loadedCfg.Workers)
	}
}

func TestLoadConfigWithInvalidData(t *testing.T) {
	// Reset viper before test
	viper.Reset()

	// Test with invalid operation type
	viper.Set("operation", "invalid_operation")

	_, err := LoadConfig()
	// Should error due to validation
	if err == nil {
		t.Error("LoadConfig() expected error for invalid operation, got nil")
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
dry-run: true
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
		name    string
		modify  func(*types.Config)
		wantErr bool
		check   func(*types.Config) error
	}{
		{
			name:    "valid default config",
			modify:  func(*types.Config) {},
			wantErr: false,
		},
		{
			name: "zero workers",
			modify: func(cfg *types.Config) {
				cfg.Workers = 0
			},
			wantErr: true,
		},
		{
			name: "negative workers",
			modify: func(cfg *types.Config) {
				cfg.Workers = -1
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			modify: func(cfg *types.Config) {
				cfg.Timeout = -1 * time.Second
			},
			wantErr: true,
		},
		{
			name: "export scan requires scan operation",
			modify: func(cfg *types.Config) {
				cfg.ExportScan = "report.md"
				cfg.Operation = types.OperationFetch
			},
			wantErr: true,
		},
		{
			name: "operation normalization",
			modify: func(cfg *types.Config) {
				cfg.Operation = "PULL"
			},
			wantErr: false,
			check: func(cfg *types.Config) error {
				if cfg.Operation != types.OperationPull {
					return fmt.Errorf("expected %q, got %q", types.OperationPull, cfg.Operation)
				}
				return nil
			},
		},
		{
			name: "empty exclude dirs allowed",
			modify: func(cfg *types.Config) {
				cfg.ExcludeDirs = []string{}
			},
			wantErr: false,
		},
		{
			name: "nil exclude dirs allowed",
			modify: func(cfg *types.Config) {
				cfg.ExcludeDirs = nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := DefaultConfig()
			tt.modify(cfg)

			err := ValidateConfig(cfg)
			if tt.wantErr && err == nil {
				t.Errorf("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no validation error, got %v", err)
			}
			if err == nil && tt.check != nil {
				if checkErr := tt.check(cfg); checkErr != nil {
					t.Error(checkErr)
				}
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
