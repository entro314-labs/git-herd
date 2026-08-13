package git

import (
	"context"
	"errors"
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

func TestScanner_NonRecursive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git-herd-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Root repository, a direct-subdirectory repository, and a repository
	// nested two levels down inside a non-repository directory.
	for _, dir := range []string{
		filepath.Join(tmpDir, ".git"),
		filepath.Join(tmpDir, "direct", ".git"),
		filepath.Join(tmpDir, "group", "nested", ".git"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	tests := []struct {
		name      string
		recursive bool
		wantNames []string
	}{
		{
			name:      "recursive finds nested repos",
			recursive: true,
			wantNames: []string{filepath.Base(tmpDir), "direct", "nested"},
		},
		{
			name:      "non-recursive only considers root and direct subdirectories",
			recursive: false,
			wantNames: []string{filepath.Base(tmpDir), "direct"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &types.Config{
				Workers:   5,
				Operation: types.OperationFetch,
				Recursive: tt.recursive,
			}

			repos, err := NewScanner(config).FindRepos(context.Background(), tmpDir, nil)
			if err != nil {
				t.Fatalf("FindRepos failed: %v", err)
			}

			names := make(map[string]bool, len(repos))
			for _, repo := range repos {
				names[repo.Name] = true
			}

			if len(repos) != len(tt.wantNames) {
				t.Errorf("Expected %d repos %v, got %d: %v", len(tt.wantNames), tt.wantNames, len(repos), names)
			}
			for _, want := range tt.wantNames {
				if !names[want] {
					t.Errorf("Expected repo %q to be found, got %v", want, names)
				}
			}
		})
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

// writeGitlink creates a worktree directory whose ".git" is a gitlink file
// pointing at target, mirroring how git records a submodule.
func writeGitlink(t *testing.T, root, name, target string) string {
	t.Helper()

	worktree := filepath.Join(root, name)
	if err := os.MkdirAll(worktree, 0755); err != nil {
		t.Fatalf("Failed to create worktree %s: %v", worktree, err)
	}

	contents := "gitdir: " + target + "\n"
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte(contents), 0644); err != nil {
		t.Fatalf("Failed to write gitlink: %v", err)
	}

	return worktree
}

func TestScanner_SubmoduleGitlinkResolves(t *testing.T) {
	tmpDir := t.TempDir()

	// The real git directory, as a parent repo's .git/modules/<name> would be.
	realGitDir := filepath.Join(tmpDir, "parent", ".git", "modules", "sub")
	if err := os.MkdirAll(realGitDir, 0755); err != nil {
		t.Fatalf("Failed to create module dir: %v", err)
	}

	writeGitlink(t, filepath.Join(tmpDir, "parent"), "sub", "../.git/modules/sub")

	config := &types.Config{Workers: 1, Operation: types.OperationScan, Recursive: true}
	repos, err := NewScanner(config).FindRepos(context.Background(), tmpDir, nil)
	if err != nil {
		t.Fatalf("FindRepos failed: %v", err)
	}

	var sub *types.GitRepo
	for i := range repos {
		if repos[i].Name == "sub" {
			sub = &repos[i]
		}
	}
	if sub == nil {
		t.Fatal("Expected the submodule worktree to be discovered")
	}

	if !sub.IsSubmodule {
		t.Error("Expected IsSubmodule to be true for a gitlink worktree")
	}
	if sub.Error != nil {
		t.Errorf("Expected no error for a resolvable gitlink, got %v", sub.Error)
	}
	if sub.GitDir != filepath.Clean(realGitDir) {
		t.Errorf("Expected GitDir %s, got %s", filepath.Clean(realGitDir), sub.GitDir)
	}
}

func TestScanner_SubmoduleGitlinkBroken(t *testing.T) {
	tmpDir := t.TempDir()

	// Point at a modules directory that was never created, the state left by a
	// clone without --recurse-submodules.
	writeGitlink(t, filepath.Join(tmpDir, "parent"), "sub", "../.git/modules/sub")

	config := &types.Config{Workers: 1, Operation: types.OperationScan, Recursive: true}
	repos, err := NewScanner(config).FindRepos(context.Background(), tmpDir, nil)
	if err != nil {
		t.Fatalf("FindRepos failed: %v", err)
	}

	var sub *types.GitRepo
	for i := range repos {
		if repos[i].Name == "sub" {
			sub = &repos[i]
		}
	}
	if sub == nil {
		t.Fatal("Expected the submodule worktree to be discovered")
	}

	if !sub.IsSubmodule {
		t.Error("Expected IsSubmodule to be true for a gitlink worktree")
	}
	if !errors.Is(sub.Error, types.ErrBrokenGitlink) {
		t.Errorf("Expected ErrBrokenGitlink, got %v", sub.Error)
	}
	if sub.GitDir != "" {
		t.Errorf("Expected empty GitDir for a broken gitlink, got %s", sub.GitDir)
	}
}

func TestScanner_MalformedGitlinkIsReported(t *testing.T) {
	tmpDir := t.TempDir()

	worktree := filepath.Join(tmpDir, "weird")
	if err := os.MkdirAll(worktree, 0755); err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte("not a gitlink\n"), 0644); err != nil {
		t.Fatalf("Failed to write .git file: %v", err)
	}

	config := &types.Config{Workers: 1, Operation: types.OperationScan, Recursive: true}
	repos, err := NewScanner(config).FindRepos(context.Background(), tmpDir, nil)
	if err != nil {
		t.Fatalf("FindRepos failed: %v", err)
	}

	if len(repos) != 1 {
		t.Fatalf("Expected 1 repo, got %d", len(repos))
	}
	if !errors.Is(repos[0].Error, types.ErrBrokenGitlink) {
		t.Errorf("Expected ErrBrokenGitlink for a .git file without a gitdir marker, got %v", repos[0].Error)
	}
}
