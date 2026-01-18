package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/entro314-labs/git-herd/pkg/types"
)

func TestScanner_FindRepos_EmptyDirectory(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "git-herd-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &types.Config{
		Workers:     5,
		Operation:   types.OperationFetch,
		Recursive:   true,
		ExcludeDirs: []string{".git"},
	}

	scanner := NewScanner(config)
	ctx := context.Background()

	repos, err := scanner.FindRepos(ctx, tmpDir, nil)
	if err != nil {
		t.Fatalf("FindRepos failed: %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("Expected 0 repos, got %d", len(repos))
	}
}

func TestScanner_FindRepos_WithGitRepo(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "git-herd-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a mock .git directory
	gitDir := filepath.Join(tmpDir, "testrepo", ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	config := &types.Config{
		Workers:     5,
		Operation:   types.OperationFetch,
		Recursive:   true,
		ExcludeDirs: []string{},
	}

	scanner := NewScanner(config)
	ctx := context.Background()

	repos, err := scanner.FindRepos(ctx, tmpDir, nil)
	if err != nil {
		t.Fatalf("FindRepos failed: %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("Expected 1 repo, got %d", len(repos))
	}

	if len(repos) > 0 {
		expectedPath := filepath.Join(tmpDir, "testrepo")
		if repos[0].Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, repos[0].Path)
		}

		if repos[0].Name != "testrepo" {
			t.Errorf("Expected name 'testrepo', got %s", repos[0].Name)
		}

		if !repos[0].HasGit {
			t.Error("Expected HasGit to be true")
		}
	}
}

func TestScanner_ExcludeDirectories(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "git-herd-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create directories that should be excluded
	excludedDirs := []string{"node_modules", "vendor"}
	for _, dir := range excludedDirs {
		dirPath := filepath.Join(tmpDir, dir, ".git")
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	// Create a legitimate git repo
	gitDir := filepath.Join(tmpDir, "project", ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create project/.git dir: %v", err)
	}

	config := &types.Config{
		Workers:     5,
		Operation:   types.OperationFetch,
		Recursive:   true,
		ExcludeDirs: excludedDirs,
	}

	scanner := NewScanner(config)
	ctx := context.Background()

	repos, err := scanner.FindRepos(ctx, tmpDir, nil)
	if err != nil {
		t.Fatalf("FindRepos failed: %v", err)
	}

	// Should only find the 'project' repo, not the excluded directories
	if len(repos) != 1 {
		t.Errorf("Expected 1 repo, got %d", len(repos))
		for i, repo := range repos {
			t.Logf("Repo %d: %s (%s)", i, repo.Name, repo.Path)
		}
	}

	if len(repos) > 0 && repos[0].Name != "project" {
		t.Errorf("Expected to find 'project', got %s", repos[0].Name)
	}
}
