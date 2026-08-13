package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/entro314-labs/git-herd/pkg/types"
)

// gitCmd runs a git command in dir with a hermetic environment and fails the
// test on error.
func gitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()

	base := []string{
		"-c", "user.name=git-herd-test",
		"-c", "user.email=test@example.com",
		"-c", "commit.gpgsign=false",
		"-c", "protocol.file.allow=always",
	}
	cmd := exec.Command("git", append(base, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}

// initTestRepo creates a repository with one committed file (tracked.txt).
func initTestRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	gitCmd(t, dir, "init", "-q", "-b", "main")
	writeFile(t, dir, "tracked.txt", "original\n")
	gitCmd(t, dir, "add", ".")
	gitCmd(t, dir, "commit", "-q", "-m", "init")
	return dir
}

// addLocalRemote wires the repository to a local bare upstream so fetch and
// pull work offline.
func addLocalRemote(t *testing.T, workDir, remoteName string) {
	t.Helper()

	upstream := t.TempDir()
	gitCmd(t, upstream, "init", "-q", "--bare", "-b", "main")
	gitCmd(t, workDir, "remote", "add", remoteName, upstream)
	gitCmd(t, workDir, "push", "-q", "-u", remoteName, "main")
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(content)
}

func fileExists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

func processTestRepo(cfg *types.Config, dir string) types.GitRepo {
	processor := NewProcessor(cfg)
	return processor.ProcessRepo(context.Background(), types.GitRepo{
		Path:   dir,
		Name:   filepath.Base(dir),
		HasGit: true,
	})
}

func TestProcessRepoSkipRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		operation   types.OperationType
		skipDirty   bool
		dirty       bool
		withRemote  bool
		wantSkipped bool
		wantErrPart string
	}{
		{
			name:       "fetch on dirty repo is not skipped",
			operation:  types.OperationFetch,
			skipDirty:  true,
			dirty:      true,
			withRemote: true,
		},
		{
			name:        "pull on dirty repo is skipped",
			operation:   types.OperationPull,
			skipDirty:   true,
			dirty:       true,
			withRemote:  true,
			wantSkipped: true,
			wantErrPart: "uncommitted changes",
		},
		{
			name:       "pull on clean repo succeeds",
			operation:  types.OperationPull,
			skipDirty:  true,
			withRemote: true,
		},
		{
			name:        "fetch without remote is skipped",
			operation:   types.OperationFetch,
			skipDirty:   true,
			wantSkipped: true,
			wantErrPart: "no remote configured",
		},
		{
			name:        "pull without remote is skipped",
			operation:   types.OperationPull,
			skipDirty:   true,
			wantSkipped: true,
			wantErrPart: "no remote configured",
		},
		{
			name:      "scan on dirty repo without remote is not skipped",
			operation: types.OperationScan,
			skipDirty: true,
			dirty:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := initTestRepo(t)
			if tt.withRemote {
				addLocalRemote(t, dir, "origin")
			}
			if tt.dirty {
				writeFile(t, dir, "tracked.txt", "local edit\n")
			}

			cfg := &types.Config{
				Operation: tt.operation,
				SkipDirty: tt.skipDirty,
				Workers:   1,
			}

			result := processTestRepo(cfg, dir)

			if tt.wantSkipped {
				if !errors.Is(result.Error, types.ErrRepoSkipped) {
					t.Fatalf("expected ErrRepoSkipped, got %v", result.Error)
				}
				if !strings.Contains(result.Error.Error(), tt.wantErrPart) {
					t.Fatalf("expected error to contain %q, got %v", tt.wantErrPart, result.Error)
				}
			} else if result.Error != nil {
				t.Fatalf("expected success, got %v", result.Error)
			}
		})
	}
}

func TestProcessRepoDiscardRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		operation    types.OperationType
		dryRun       bool
		discardFiles []string
		setup        func(t *testing.T, dir string)
		verify       func(t *testing.T, dir string, result types.GitRepo)
	}{
		{
			name:         "discard restores tracked file before fetch",
			operation:    types.OperationFetch,
			discardFiles: []string{"tracked.txt"},
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "tracked.txt", "local edit\n")
			},
			verify: func(t *testing.T, dir string, result types.GitRepo) {
				if result.Error != nil {
					t.Fatalf("expected success, got %v", result.Error)
				}
				if got := readFile(t, dir, "tracked.txt"); got != "original\n" {
					t.Fatalf("expected tracked.txt restored to original, got %q", got)
				}
			},
		},
		{
			name:         "discard removes matching untracked file",
			operation:    types.OperationFetch,
			discardFiles: []string{".DS_Store"},
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, ".DS_Store", "junk")
			},
			verify: func(t *testing.T, dir string, result types.GitRepo) {
				if result.Error != nil {
					t.Fatalf("expected success, got %v", result.Error)
				}
				if fileExists(dir, ".DS_Store") {
					t.Fatal("expected untracked .DS_Store to be removed")
				}
			},
		},
		{
			name:         "discard bails out when non-matching changes exist",
			operation:    types.OperationPull,
			discardFiles: []string{"package.json"},
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "package.json", "{}")
				writeFile(t, dir, "tracked.txt", "important local work\n")
			},
			verify: func(t *testing.T, dir string, result types.GitRepo) {
				if !errors.Is(result.Error, types.ErrRepoSkipped) {
					t.Fatalf("expected dirty repo to be skipped for pull, got %v", result.Error)
				}
				if got := readFile(t, dir, "tracked.txt"); got != "important local work\n" {
					t.Fatalf("expected tracked.txt preserved, got %q", got)
				}
				if !fileExists(dir, "package.json") {
					t.Fatal("expected package.json preserved when discard bails out")
				}
			},
		},
		{
			name:         "dry-run does not mutate the worktree",
			operation:    types.OperationPull,
			dryRun:       true,
			discardFiles: []string{"tracked.txt"},
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "tracked.txt", "local edit\n")
			},
			verify: func(t *testing.T, dir string, result types.GitRepo) {
				if result.Error != nil {
					t.Fatalf("expected dry-run success, got %v", result.Error)
				}
				if got := readFile(t, dir, "tracked.txt"); got != "local edit\n" {
					t.Fatalf("expected dry-run to leave tracked.txt untouched, got %q", got)
				}
			},
		},
		{
			name:         "scan never discards even with matching patterns",
			operation:    types.OperationScan,
			discardFiles: []string{"tracked.txt"},
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "tracked.txt", "local edit\n")
			},
			verify: func(t *testing.T, dir string, result types.GitRepo) {
				if result.Error != nil {
					t.Fatalf("expected scan success, got %v", result.Error)
				}
				if got := readFile(t, dir, "tracked.txt"); got != "local edit\n" {
					t.Fatalf("expected scan to leave tracked.txt untouched, got %q", got)
				}
				if len(result.ModifiedFiles) != 1 || result.ModifiedFiles[0] != "tracked.txt" {
					t.Fatalf("expected scan to report tracked.txt as modified, got %v", result.ModifiedFiles)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := initTestRepo(t)
			if tt.operation != types.OperationScan {
				addLocalRemote(t, dir, "origin")
			}
			tt.setup(t, dir)

			cfg := &types.Config{
				Operation:    tt.operation,
				SkipDirty:    true,
				DryRun:       tt.dryRun,
				DiscardFiles: tt.discardFiles,
				Workers:      1,
			}

			result := processTestRepo(cfg, dir)
			tt.verify(t, dir, result)
		})
	}
}

func TestProcessRepoCancellationIsSkipped(t *testing.T) {
	t.Parallel()

	dir := initTestRepo(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	processor := NewProcessor(&types.Config{Operation: types.OperationFetch, Workers: 1})
	result := processor.ProcessRepo(ctx, types.GitRepo{Path: dir, Name: "repo", HasGit: true})

	if !errors.Is(result.Error, types.ErrRepoSkipped) {
		t.Fatalf("expected cancellation to be reported as skipped, got %v", result.Error)
	}
}

func TestAnalyzeRepoRemoteSelection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		remotes    []string
		wantRemote string
	}{
		{name: "no remotes", remotes: nil, wantRemote: ""},
		{name: "origin preferred over others", remotes: []string{"zzz", "origin", "aaa"}, wantRemote: "origin"},
		{name: "alphabetical fallback without origin", remotes: []string{"zzz", "upstream"}, wantRemote: "upstream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := initTestRepo(t)
			for _, remote := range tt.remotes {
				gitCmd(t, dir, "remote", "add", remote, t.TempDir())
			}

			repo := types.GitRepo{Path: dir, Name: "repo", HasGit: true}
			NewProcessor(&types.Config{}).AnalyzeRepo(&repo)

			if repo.Error != nil {
				t.Fatalf("unexpected analyze error: %v", repo.Error)
			}
			if repo.Remote != tt.wantRemote {
				t.Fatalf("expected remote %q, got %q", tt.wantRemote, repo.Remote)
			}
		})
	}
}

func TestOperationContextWithoutTimeout(t *testing.T) {
	t.Parallel()

	processor := NewProcessor(&types.Config{})
	parentCtx := context.Background()

	ctx, cancel := processor.operationContext(parentCtx)
	defer cancel()

	if _, ok := ctx.Deadline(); ok {
		t.Fatal("expected no deadline when timeout is disabled")
	}

	if ctx != parentCtx {
		t.Fatal("expected operationContext to reuse the parent context when timeout is disabled")
	}
}

func TestOperationContextWithTimeout(t *testing.T) {
	t.Parallel()

	processor := NewProcessor(&types.Config{Timeout: 250 * time.Millisecond})
	parentCtx := context.Background()

	ctx, cancel := processor.operationContext(parentCtx)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline when timeout is configured")
	}

	remaining := time.Until(deadline)
	if remaining <= 0 {
		t.Fatalf("expected future deadline, got %v", remaining)
	}

	if remaining > 400*time.Millisecond {
		t.Fatalf("expected deadline close to configured timeout, got %v", remaining)
	}

	if _, ok := parentCtx.Deadline(); ok {
		t.Fatal("expected parent context to remain without a deadline")
	}
}

func TestOperationContextHonorsEarlierParentDeadline(t *testing.T) {
	t.Parallel()

	processor := NewProcessor(&types.Config{Timeout: 2 * time.Second})
	parentCtx, parentCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer parentCancel()

	ctx, cancel := processor.operationContext(parentCtx)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected derived context to have a deadline")
	}

	remaining := time.Until(deadline)
	if remaining > 300*time.Millisecond {
		t.Fatalf("expected parent deadline to win, got %v", remaining)
	}
}

func TestProcessRepo_SkipsSubmoduleByDefault(t *testing.T) {
	// A healthy repository, flagged as a submodule the way the scanner would
	// after resolving a gitlink.
	dir := initTestRepo(t)
	addLocalRemote(t, dir, "origin")
	repo := types.GitRepo{Path: dir, Name: "sub", HasGit: true, IsSubmodule: true}

	p := NewProcessor(&types.Config{Operation: types.OperationPull, Timeout: time.Minute})
	got := p.ProcessRepo(context.Background(), repo)

	if !errors.Is(got.Error, types.ErrRepoSkipped) {
		t.Errorf("Expected a submodule to be skipped on pull, got %v", got.Error)
	}

	// The skip must still report what was found, so the repo appears in
	// summaries rather than vanishing.
	if got.Branch != "main" {
		t.Errorf("Expected a skipped submodule to still report its branch, got %q", got.Branch)
	}
}

func TestProcessRepo_ProcessesSubmoduleWhenRequested(t *testing.T) {
	dir := initTestRepo(t)
	addLocalRemote(t, dir, "origin")
	repo := types.GitRepo{Path: dir, Name: "sub", HasGit: true, IsSubmodule: true}

	p := NewProcessor(&types.Config{
		Operation:      types.OperationPull,
		Timeout:        time.Minute,
		WithSubmodules: true,
	})
	got := p.ProcessRepo(context.Background(), repo)

	if errors.Is(got.Error, types.ErrRepoSkipped) {
		t.Errorf("Expected --with-submodules to process the submodule, got %v", got.Error)
	}
}

func TestProcessRepo_ScanAlwaysReportsSubmodules(t *testing.T) {
	dir := initTestRepo(t)
	addLocalRemote(t, dir, "origin")
	repo := types.GitRepo{Path: dir, Name: "sub", HasGit: true, IsSubmodule: true}

	p := NewProcessor(&types.Config{Operation: types.OperationScan, Timeout: time.Minute})
	got := p.ProcessRepo(context.Background(), repo)

	if got.Error != nil {
		t.Errorf("Expected scan to report a submodule without error, got %v", got.Error)
	}
}

func TestProcessRepo_BrokenGitlinkErrorSurvivesAnalysis(t *testing.T) {
	// A gitlink diagnosed at scan time must not be overwritten by the generic
	// "failed to open repository" error AnalyzeRepo would otherwise produce.
	repo := types.GitRepo{
		Path:        t.TempDir(),
		Name:        "sub",
		HasGit:      true,
		IsSubmodule: true,
		Error:       fmt.Errorf("%w: gitdir missing", types.ErrBrokenGitlink),
	}

	p := NewProcessor(&types.Config{Operation: types.OperationPull, Timeout: time.Minute})
	got := p.ProcessRepo(context.Background(), repo)

	if !errors.Is(got.Error, types.ErrBrokenGitlink) {
		t.Errorf("Expected ErrBrokenGitlink to be preserved, got %v", got.Error)
	}
}

func TestInvalidPathHint(t *testing.T) {
	if got := invalidPathHint(nil); got != "" {
		t.Errorf("Expected no hint for nil error, got %q", got)
	}
	if got := invalidPathHint(errors.New("some other failure")); got != "" {
		t.Errorf("Expected no hint for an unrelated error, got %q", got)
	}
	got := invalidPathHint(errors.New(`invalid path "Icon\r": contains control character`))
	if !strings.Contains(got, "Icon") {
		t.Errorf("Expected an actionable hint mentioning the Icon file, got %q", got)
	}
}
