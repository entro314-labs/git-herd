package tui

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/entro314-labs/git-herd/internal/config"
	"github.com/entro314-labs/git-herd/pkg/types"
)

func TestNewModel(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	rootPath := "/test/path"

	model := NewModel(cfg, rootPath)

	// Test basic initialization
	if model == nil {
		t.Fatal("NewModel() returned nil")
	}

	if model.config != cfg {
		t.Errorf("Expected config to be %v, got %v", cfg, model.config)
	}

	if model.rootPath != rootPath {
		t.Errorf("Expected rootPath to be %q, got %q", rootPath, model.rootPath)
	}

	if model.phase != "initializing" {
		t.Errorf("Expected initial phase to be 'initializing', got %q", model.phase)
	}

	if !model.scanning {
		t.Error("Expected scanning to be true initially")
	}

	if model.processing {
		t.Error("Expected processing to be false initially")
	}

	if model.done {
		t.Error("Expected done to be false initially")
	}

	if model.ctx == nil {
		t.Error("Expected context to be initialized")
	}

	if model.cancel == nil {
		t.Error("Expected cancel function to be initialized")
	}

	if model.scanner == nil {
		t.Error("Expected scanner to be initialized")
	}

	if model.processor == nil {
		t.Error("Expected processor to be initialized")
	}

	// Test UI components initialization
	if reflect.DeepEqual(model.spinner, spinner.Model{}) {
		t.Error("Expected spinner to be initialized")
	}

	if reflect.DeepEqual(model.progress, progress.Model{}) {
		t.Error("Expected progress to be initialized")
	}
}

func TestNewModelWithTimeout(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.Timeout = 5 * time.Second
	rootPath := "/test/path"

	model := NewModel(cfg, rootPath)

	if model.ctx == nil {
		t.Error("Expected context to be initialized")
	}

	// Check that context has deadline
	deadline, ok := model.ctx.Deadline()
	if !ok {
		t.Error("Expected context to have deadline when timeout is set")
	}

	if time.Until(deadline) > 6*time.Second {
		t.Error("Expected context deadline to be approximately 5 seconds from now")
	}
}

func TestModelInit(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")

	cmd := model.Init()
	if cmd == nil {
		t.Error("Expected Init() to return a command")
	}

	// Init should return a batch command with spinner tick and repo scanning
	// We can't easily test the exact commands without refactoring, but we can
	// verify that a command is returned
}

func TestModelUpdateKeyMessages(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")

	tests := []struct {
		name    string
		keyMsg  tea.KeyMsg
		wantCmd bool
	}{
		{
			name:    "ctrl+c quits",
			keyMsg:  tea.KeyMsg{Type: tea.KeyCtrlC},
			wantCmd: true,
		},
		{
			name:    "q quits",
			keyMsg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			wantCmd: true,
		},
		{
			name:    "other key no action",
			keyMsg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}},
			wantCmd: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			newModel, cmd := model.Update(tt.keyMsg)
			if newModel == nil {
				t.Error("Expected Update to return a model")
			}

			if tt.wantCmd && cmd == nil {
				t.Error("Expected Update to return a command")
			}

			if !tt.wantCmd && cmd != nil {
				t.Error("Expected Update to return nil command")
			}
		})
	}
}

func TestModelUpdateSpinnerTick(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")

	// Create a spinner tick message
	tickMsg := spinner.TickMsg{
		Time: time.Now(),
		ID:   1,
	}

	newModel, cmd := model.Update(tickMsg)
	if newModel == nil {
		t.Error("Expected Update to return a model")
	}

	// The spinner might or might not return a command depending on internal state
	// This is acceptable behavior, so we just test that the model is returned
	_ = cmd // Command may or may not be present
}

func TestModelUpdateReposFound(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")

	repos := []types.GitRepo{
		{
			Path:   "/test/repo1",
			Name:   "repo1",
			HasGit: true,
			Clean:  true,
		},
		{
			Path:   "/test/repo2",
			Name:   "repo2",
			HasGit: true,
			Clean:  true,
		},
	}

	msg := reposFoundMsg(repos)
	newModel, cmd := model.Update(msg)

	if newModel == nil {
		t.Error("Expected Update to return a model")
	}

	updatedModel := newModel.(*Model)

	// Check state changes
	if updatedModel.scanning {
		t.Error("Expected scanning to be false after repos found")
	}

	if !updatedModel.processing {
		t.Error("Expected processing to be true after repos found")
	}

	if updatedModel.phase != "processing" {
		t.Errorf("Expected phase to be 'processing', got %q", updatedModel.phase)
	}

	if len(updatedModel.repos) != len(repos) {
		t.Errorf("Expected %d repos, got %d", len(repos), len(updatedModel.repos))
	}

	// Should return a command to start processing
	if cmd == nil {
		t.Error("Expected Update to return a processing command")
	}
}

func TestModelUpdateReposFoundEmpty(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")

	msg := reposFoundMsg([]types.GitRepo{})
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*Model)

	// Should complete immediately with no repos
	if !updatedModel.done {
		t.Error("Expected done to be true when no repos found")
	}

	if updatedModel.phase != "complete" {
		t.Errorf("Expected phase to be 'complete', got %q", updatedModel.phase)
	}

	// Should quit
	if cmd == nil {
		t.Error("Expected Update to return a quit command")
	}
}

func TestModelUpdateRepoProcessed(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")

	// Set up model with repos
	model.repos = []types.GitRepo{
		{Path: "/test/repo1", Name: "repo1"},
		{Path: "/test/repo2", Name: "repo2"},
	}
	model.processing = true
	model.phase = "processing"

	processedRepo := types.GitRepo{
		Path:     "/test/repo1",
		Name:     "repo1",
		Duration: 100 * time.Millisecond,
	}

	msg := repoProcessedMsg(processedRepo)
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*Model)

	// Check that results were updated
	if len(updatedModel.results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(updatedModel.results))
	}

	if updatedModel.processed != 1 {
		t.Errorf("Expected processed count to be 1, got %d", updatedModel.processed)
	}

	// Should still be processing since there are more repos
	if !updatedModel.processing {
		t.Error("Expected processing to still be true")
	}

	if updatedModel.done {
		t.Error("Expected done to be false until all repos processed")
	}

	// Should return command to process next repo
	if cmd == nil {
		t.Error("Expected Update to return a command to process next repo")
	}
}

func TestModelUpdateRepoProcessedComplete(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")

	// Set up model with one repo
	model.repos = []types.GitRepo{
		{Path: "/test/repo1", Name: "repo1"},
	}
	model.processing = true
	model.phase = "processing"
	model.processed = 0

	processedRepo := types.GitRepo{
		Path:     "/test/repo1",
		Name:     "repo1",
		Duration: 100 * time.Millisecond,
	}

	msg := repoProcessedMsg(processedRepo)
	newModel, cmd := model.Update(msg)

	updatedModel := newModel.(*Model)

	// Should be done after processing the only repo
	if updatedModel.processing {
		t.Error("Expected processing to be false after all repos processed")
	}

	if !updatedModel.done {
		t.Error("Expected done to be true after all repos processed")
	}

	if updatedModel.phase != "complete" {
		t.Errorf("Expected phase to be 'complete', got %q", updatedModel.phase)
	}

	// Should return quit command
	if cmd == nil {
		t.Error("Expected Update to return a quit command")
	}
}

func TestModelUpdateProcessingDone(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")

	testErr := errors.New("processing error")
	msg := processingDoneMsg{err: testErr}

	newModel, cmd := model.Update(msg)
	updatedModel := newModel.(*Model)

	// Check state changes
	if updatedModel.processing {
		t.Error("Expected processing to be false after processing done")
	}

	if !updatedModel.done {
		t.Error("Expected done to be true after processing done")
	}

	if updatedModel.phase != "complete" {
		t.Errorf("Expected phase to be 'complete', got %q", updatedModel.phase)
	}

	if updatedModel.err != testErr {
		t.Errorf("Expected error to be %v, got %v", testErr, updatedModel.err)
	}

	// Should return quit command
	if cmd == nil {
		t.Error("Expected Update to return a quit command")
	}
}

func TestModelProcessMethods(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping process method tests in short mode")
	}

	// These tests require more complex setup with actual git operations
	// and are primarily integration tests

	cfg := config.DefaultConfig()
	cfg.DryRun = true // Use dry run to avoid actual git operations

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "tui-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	model := NewModel(cfg, tmpDir)

	// Test scanRepos
	cmd := model.scanRepos()
	if cmd == nil {
		t.Error("Expected scanRepos to return a command")
	}

	// Execute the command to test it
	msg := cmd()
	if msg == nil {
		t.Error("Expected scanRepos command to return a message")
	}

	// The message should be either reposFoundMsg or processingDoneMsg
	switch msg := msg.(type) {
	case reposFoundMsg:
		// Expected for successful scan
		t.Logf("Found %d repos", len(msg))
	case processingDoneMsg:
		if msg.err == nil {
			t.Log("No repos found (expected for empty directory)")
		} else {
			t.Errorf("Unexpected error from scanRepos: %v", msg.err)
		}
	default:
		t.Errorf("Unexpected message type from scanRepos: %T", msg)
	}
}

func TestModelProcessReposEmpty(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")

	// Empty repos list
	model.repos = []types.GitRepo{}

	cmd := model.processRepos()
	if cmd == nil {
		t.Error("Expected processRepos to return a command")
	}

	msg := cmd()
	if msg == nil {
		t.Error("Expected processRepos command to return a message")
	}

	// Should return processingDoneMsg with no error
	doneMsg, ok := msg.(processingDoneMsg)
	if !ok {
		t.Errorf("Expected processingDoneMsg, got %T", msg)
	}

	if doneMsg.err != nil {
		t.Errorf("Expected no error for empty repos, got %v", doneMsg.err)
	}
}

func TestModelProcessNextRepo(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.DryRun = true
	model := NewModel(cfg, "/test/path")

	// Set up repos
	model.repos = []types.GitRepo{
		{Path: "/test/repo1", Name: "repo1", HasGit: true},
		{Path: "/test/repo2", Name: "repo2", HasGit: true},
	}
	model.processed = 1 // One repo already processed
	model.nextIndex = 1

	cmd := model.processNextRepo()
	if cmd == nil {
		t.Error("Expected processNextRepo to return a command")
	}

	// The command execution would require actual git operations
	// so we just test that it returns a command
}

func TestModelProcessNextRepoComplete(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")

	// Set up repos with all processed
	model.repos = []types.GitRepo{
		{Path: "/test/repo1", Name: "repo1"},
	}
	model.processed = 1 // All repos processed
	model.nextIndex = 1

	cmd := model.processNextRepo()
	if cmd != nil {
		t.Error("Expected processNextRepo to return nil when all repos processed")
	}
}

func TestModelMessageTypes(t *testing.T) {
	t.Parallel()

	// Test message type conversions
	repos := []types.GitRepo{
		{Path: "/test", Name: "test"},
	}

	reposMsg := reposFoundMsg(repos)
	convertedRepos := []types.GitRepo(reposMsg)

	if len(convertedRepos) != len(repos) {
		t.Errorf("Expected %d repos after conversion, got %d", len(repos), len(convertedRepos))
	}

	repo := types.GitRepo{Path: "/test", Name: "test"}
	repoMsg := repoProcessedMsg(repo)
	convertedRepo := types.GitRepo(repoMsg)

	if convertedRepo.Path != repo.Path {
		t.Errorf("Expected path %q after conversion, got %q", repo.Path, convertedRepo.Path)
	}

	// Test processingDoneMsg
	err := errors.New("test error")
	doneMsg := processingDoneMsg{err: err}

	if doneMsg.err.Error() != err.Error() {
		t.Errorf("Expected error %q, got %q", err.Error(), doneMsg.err.Error())
	}
}

func TestModelCancel(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")

	// Test that context is not cancelled initially
	select {
	case <-model.ctx.Done():
		t.Error("Context should not be cancelled initially")
	default:
		// Expected
	}

	// Cancel the context
	model.cancel()

	// Test that context is now cancelled
	select {
	case <-model.ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled after cancel() call")
	}
}

func TestModelConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	cfg := config.DefaultConfig()
	cfg.Timeout = 1 * time.Second
	model := NewModel(cfg, "/test/path")

	// Test that multiple operations can be performed safely
	done := make(chan bool, 2)

	// Simulate concurrent updates
	go func() {
		for i := 0; i < 10; i++ {
			tickMsg := spinner.TickMsg{Time: time.Now(), ID: i}
			_, _ = model.Update(tickMsg)
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 5; i++ {
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
			_, _ = model.Update(keyMsg)
			time.Sleep(20 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	timeout := time.After(2 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case <-done:
			// Expected
		case <-timeout:
			t.Error("Concurrent operations timed out")
		}
	}
}

// Benchmark tests
func BenchmarkNewModel(b *testing.B) {
	cfg := config.DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model := NewModel(cfg, "/test/path")
		model.cancel() // Clean up context
	}
}

func BenchmarkModelUpdate(b *testing.B) {
	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")
	defer model.cancel()

	tickMsg := spinner.TickMsg{Time: time.Now(), ID: 1}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = model.Update(tickMsg)
	}
}

func BenchmarkRepoProcessedUpdate(b *testing.B) {
	cfg := config.DefaultConfig()
	model := NewModel(cfg, "/test/path")
	defer model.cancel()

	// Set up repos for processing
	model.repos = make([]types.GitRepo, 1000)
	for i := 0; i < 1000; i++ {
		model.repos[i] = types.GitRepo{
			Path: fmt.Sprintf("/test/repo%d", i),
			Name: fmt.Sprintf("repo%d", i),
		}
	}
	model.processing = true
	model.phase = "processing"

	repoMsg := repoProcessedMsg(types.GitRepo{
		Path: "/test/repo1",
		Name: "repo1",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset processed count to avoid completing
		model.processed = 0
		model.results = model.results[:0]
		_, _ = model.Update(repoMsg)
	}
}
