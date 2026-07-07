package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"

	"github.com/entro314-labs/git-herd/pkg/types"
)

// Processor handles git operations on repositories
type Processor struct {
	config *types.Config
}

// NewProcessor creates a new git operations processor
func NewProcessor(config *types.Config) *Processor {
	return &Processor{
		config: config,
	}
}

// AnalyzeRepo analyzes a git repository to determine its status
func (p *Processor) AnalyzeRepo(repo *types.GitRepo) {
	start := time.Now()
	defer func() {
		repo.Duration = time.Since(start)
	}()

	gitRepo, err := gogit.PlainOpen(repo.Path)
	if err != nil {
		repo.Error = fmt.Errorf("failed to open repository: %w", err)
		return
	}

	// Get current branch
	head, err := gitRepo.Head()
	if err != nil {
		repo.Error = fmt.Errorf("failed to get HEAD: %w", err)
		return
	}

	if head.Name().IsBranch() {
		repo.Branch = head.Name().Short()
	} else {
		repo.Branch = "detached"
	}

	// Get last commit information
	commit, err := gitRepo.CommitObject(head.Hash())
	if err == nil {
		repo.LastCommit = head.Hash().String()[:8]                  // Short hash
		repo.LastCommitMsg = strings.Split(commit.Message, "\n")[0] // First line only
	}

	// Check working tree status
	worktree, err := gitRepo.Worktree()
	if err != nil {
		repo.Error = fmt.Errorf("failed to get worktree: %w", err)
		return
	}

	status, err := worktree.Status()
	if err != nil {
		repo.Error = fmt.Errorf("failed to get status: %w", err)
		return
	}

	repo.Clean = status.IsClean()

	// Collect modified files
	repo.ModifiedFiles = []string{}
	for file, fileStatus := range status {
		if fileStatus.Worktree != gogit.Unmodified || fileStatus.Staging != gogit.Unmodified {
			repo.ModifiedFiles = append(repo.ModifiedFiles, file)
		}
	}

	// Get remote information, preferring "origin" and falling back to the
	// alphabetically first remote for deterministic behavior.
	remotes, err := gitRepo.Remotes()
	if err == nil && len(remotes) > 0 {
		names := make([]string, 0, len(remotes))
		for _, remote := range remotes {
			names = append(names, remote.Config().Name)
		}
		slices.Sort(names)
		if slices.Contains(names, gogit.DefaultRemoteName) {
			repo.Remote = gogit.DefaultRemoteName
		} else {
			repo.Remote = names[0]
		}
	}
}

// ProcessRepo performs the git operation on a single repository
func (p *Processor) ProcessRepo(ctx context.Context, repo types.GitRepo) types.GitRepo {
	start := time.Now()
	defer func() {
		repo.Duration = time.Since(start)
	}()

	repoCtx, cancel := p.operationContext(ctx)
	defer cancel()

	if err := repoCtx.Err(); err != nil {
		repo.Error = classifyContextErr(err)
		return repo
	}

	// Analyze repo first (moved from scanning phase for better performance)
	p.AnalyzeRepo(&repo)

	if repo.Error != nil {
		return repo
	}

	// Scan is read-only: analysis is already done, never mutate the worktree.
	if p.config.Operation == types.OperationScan {
		return repo
	}

	// Discard specific files if configured. Dry-run previews the discard
	// outcome without mutating the worktree.
	if len(p.config.DiscardFiles) > 0 && !repo.Clean {
		if err := repoCtx.Err(); err != nil {
			repo.Error = classifyContextErr(err)
			return repo
		}

		gitRepo, err := gogit.PlainOpen(repo.Path)
		if err != nil {
			repo.Error = fmt.Errorf("failed to open repository for discard: %w", err)
			return repo
		}

		tracked, untracked, allMatch, err := p.planDiscard(gitRepo, &repo)
		if err != nil {
			repo.Error = fmt.Errorf("failed to discard files: %w", err)
			return repo
		}

		if allMatch {
			if p.config.DryRun {
				// The discard would leave the worktree clean, so the skip
				// decision below must reflect the post-discard state.
				repo.Clean = true
			} else {
				if err := p.executeDiscard(repoCtx, &repo, tracked, untracked); err != nil {
					repo.Error = fmt.Errorf("failed to discard files: %w", err)
					return repo
				}

				if err := repoCtx.Err(); err != nil {
					repo.Error = classifyContextErr(err)
					return repo
				}

				// Re-analyze after discarding files
				p.AnalyzeRepo(&repo)
			}
		}
	}

	// Skip dirty repos if configured. Only pull merges into the worktree;
	// fetch is safe on dirty repositories.
	if p.config.SkipDirty && !repo.Clean && p.config.Operation == types.OperationPull {
		repo.Error = fmt.Errorf("%w: uncommitted changes", types.ErrRepoSkipped)
		return repo
	}

	// Repositories without a remote have nothing to fetch or pull from.
	if repo.Remote == "" {
		repo.Error = fmt.Errorf("%w: no remote configured", types.ErrRepoSkipped)
		return repo
	}

	if p.config.DryRun {
		return repo
	}

	var err error
	switch p.config.Operation {
	case types.OperationFetch:
		err = p.fetchRepo(repoCtx, repo.Path, repo.Remote)
	case types.OperationPull:
		err = p.pullRepo(repoCtx, repo.Path, repo.Remote)
	}

	if err != nil {
		// A command killed by user cancellation is not a repository failure.
		if ctxErr := ctx.Err(); errors.Is(ctxErr, context.Canceled) {
			err = ctxErr
		}
		repo.Error = classifyContextErr(err)
	}

	return repo
}

// classifyContextErr converts user cancellation into a skip so interrupted
// repositories are reported as skipped instead of failed. Per-repository
// timeouts (context.DeadlineExceeded) remain genuine failures.
func classifyContextErr(err error) error {
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("%w: operation cancelled", types.ErrRepoSkipped)
	}
	return err
}

func (p *Processor) operationContext(parent context.Context) (context.Context, context.CancelFunc) {
	if p.config.Timeout <= 0 {
		return parent, func() {}
	}

	return context.WithTimeout(parent, p.config.Timeout)
}

// gitEnv returns the environment for git subprocesses with interactive
// credential prompts disabled so commands fail fast instead of hanging
// when authentication is required.
func gitEnv() []string {
	return append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=",
		"SSH_ASKPASS=",
	)
}

// fetchRepo performs git fetch on a repository
func (p *Processor) fetchRepo(ctx context.Context, repoPath, remote string) error {
	cmd := exec.CommandContext(ctx, "git", "fetch", remote)
	cmd.Dir = repoPath
	cmd.Env = gitEnv()

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("fetch failed: %w (output: %s)", err, string(output))
	}

	return nil
}

// pullRepo performs git pull on a repository
func (p *Processor) pullRepo(ctx context.Context, repoPath, remote string) error {
	cmd := exec.CommandContext(ctx, "git", "pull", remote)
	cmd.Dir = repoPath
	cmd.Env = gitEnv()

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pull failed: %w (output: %s)", err, string(output))
	}

	return nil
}

// planDiscard determines which modified files match the configured discard
// patterns, separating tracked files (restored from HEAD) from untracked
// files (removed, since HEAD has no version of them). allMatch is false when
// any modified file does not match the patterns: the user has legitimate
// changes, so nothing should be discarded.
func (p *Processor) planDiscard(gitRepo *gogit.Repository, repo *types.GitRepo) (tracked, untracked []string, allMatch bool, err error) {
	worktree, err := gitRepo.Worktree()
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to get status: %w", err)
	}

	for file, fileStatus := range status {
		if fileStatus.Worktree == gogit.Unmodified && fileStatus.Staging == gogit.Unmodified {
			continue
		}

		cleanPath := filepath.Clean(file)
		if filepath.IsAbs(cleanPath) || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(os.PathSeparator)) {
			return nil, nil, false, fmt.Errorf("invalid file path from git status: %q", file)
		}

		// Check if file matches any discard pattern
		matches := false
		for _, pattern := range p.config.DiscardFiles {
			// Support both exact matches and glob patterns
			matched, matchErr := filepath.Match(pattern, filepath.Base(file))
			if matchErr != nil {
				return nil, nil, false, fmt.Errorf("invalid discard pattern %q: %w", pattern, matchErr)
			}
			if matched || file == pattern || filepath.Base(file) == pattern {
				matches = true
				break
			}
		}

		if !matches {
			// Found a modified file that we shouldn't discard
			// Bail out entirely - we only discard if ALL modified files match the patterns
			if p.config.Verbose {
				fmt.Printf("  Keeping changes in %s because %s is modified and doesn't match discard patterns\n", repo.Name, file)
			}
			return nil, nil, false, nil
		}

		if fileStatus.Staging == gogit.Untracked && fileStatus.Worktree == gogit.Untracked {
			untracked = append(untracked, file)
		} else {
			tracked = append(tracked, file)
		}
	}

	return tracked, untracked, true, nil
}

// executeDiscard restores tracked files from HEAD and removes untracked ones.
func (p *Processor) executeDiscard(ctx context.Context, repo *types.GitRepo, tracked, untracked []string) error {
	for _, file := range tracked {
		cmd := exec.CommandContext(ctx, "git", "checkout", "HEAD", "--", file)
		cmd.Dir = repo.Path
		cmd.Env = gitEnv()

		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to discard %s: %w (output: %s)", file, err, string(output))
		}
	}

	// `git checkout HEAD` would fail for untracked files because HEAD has no
	// version of them, so remove them instead. -x also removes files covered
	// by ignore rules: the discard pattern matched them explicitly.
	for _, file := range untracked {
		cmd := exec.CommandContext(ctx, "git", "clean", "-f", "-x", "--", file)
		cmd.Dir = repo.Path
		cmd.Env = gitEnv()

		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to remove untracked %s: %w (output: %s)", file, err, string(output))
		}
	}

	if p.config.Verbose && (len(tracked) > 0 || len(untracked) > 0) {
		fmt.Printf("  Discarded changes in %s: %v\n", repo.Name, append(tracked, untracked...))
	}

	return nil
}
