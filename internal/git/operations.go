package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		repo.LastCommit = head.Hash().String()[:8] // Short hash
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

	// Get remote information
	remotes, err := gitRepo.Remotes()
	if err == nil && len(remotes) > 0 {
		repo.Remote = remotes[0].Config().Name
	}
}

// ProcessRepo performs the git operation on a single repository
func (p *Processor) ProcessRepo(ctx context.Context, repo types.GitRepo) types.GitRepo {
	start := time.Now()
	defer func() {
		repo.Duration = time.Since(start)
	}()

	// Analyze repo first (moved from scanning phase for better performance)
	p.AnalyzeRepo(&repo)

	if repo.Error != nil {
		return repo
	}

	// Discard specific files if configured
	if len(p.config.DiscardFiles) > 0 && !repo.Clean {
		gitRepo, err := gogit.PlainOpen(repo.Path)
		if err != nil {
			repo.Error = fmt.Errorf("failed to open repository for discard: %w", err)
			return repo
		}

		if err := p.discardFiles(gitRepo, &repo); err != nil {
			repo.Error = fmt.Errorf("failed to discard files: %w", err)
			return repo
		}

		// Re-analyze after discarding files
		p.AnalyzeRepo(&repo)
	}

	// Skip dirty repos if configured (but not for scan operation)
	if p.config.SkipDirty && !repo.Clean && p.config.Operation != types.OperationScan {
		repo.Error = fmt.Errorf("repository has uncommitted changes (skipped)")
		return repo
	}

	if p.config.DryRun {
		return repo
	}

	gitRepo, err := gogit.PlainOpen(repo.Path)
	if err != nil {
		repo.Error = fmt.Errorf("failed to open repository: %w", err)
		return repo
	}

	switch p.config.Operation {
	case types.OperationFetch:
		err = p.fetchRepo(ctx, gitRepo)
	case types.OperationPull:
		err = p.pullRepo(ctx, gitRepo)
	case types.OperationScan:
		// Scan operation - analysis already done in AnalyzeRepo
		return repo
	}

	if err != nil {
		repo.Error = err
	}

	return repo
}

// fetchRepo performs git fetch on a repository
func (p *Processor) fetchRepo(ctx context.Context, repo *gogit.Repository) error {
	err := repo.FetchContext(ctx, &gogit.FetchOptions{
		RemoteName: "origin",
		Progress:   nil, // We could add progress reporting here
	})

	if err != nil && err != gogit.NoErrAlreadyUpToDate {
		return fmt.Errorf("fetch failed: %w", err)
	}

	return nil
}

// pullRepo performs git pull on a repository
func (p *Processor) pullRepo(ctx context.Context, repo *gogit.Repository) error {
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = worktree.PullContext(ctx, &gogit.PullOptions{
		RemoteName: "origin",
		Progress:   nil,
	})

	if err != nil && err != gogit.NoErrAlreadyUpToDate {
		return fmt.Errorf("pull failed: %w", err)
	}

	return nil
}

// discardFiles discards changes to specific files matching the configured patterns
func (p *Processor) discardFiles(gitRepo *gogit.Repository, repo *types.GitRepo) error {
	worktree, err := gitRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Track which files were discarded
	var discardedFiles []string

	// Iterate through modified files and discard those matching patterns
	for file, fileStatus := range status {
		if fileStatus.Worktree == gogit.Unmodified && fileStatus.Staging == gogit.Unmodified {
			continue
		}

		// Check if file matches any discard pattern
		shouldDiscard := false
		for _, pattern := range p.config.DiscardFiles {
			// Support both exact matches and glob patterns
			matched, err := filepath.Match(pattern, filepath.Base(file))
			if err == nil && matched {
				shouldDiscard = true
				break
			}
			// Also check exact match
			if file == pattern || filepath.Base(file) == pattern {
				shouldDiscard = true
				break
			}
		}

		if shouldDiscard {
			discardedFiles = append(discardedFiles, file)
		}
	}

	// If we have files to discard, use git checkout to reset them
	if len(discardedFiles) > 0 {
		for _, file := range discardedFiles {
			// Use git command to discard changes to specific file
			cmd := exec.Command("git", "checkout", "HEAD", "--", file)
			cmd.Dir = repo.Path
			cmd.Env = os.Environ()

			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to discard %s: %w (output: %s)", file, err, string(output))
			}
		}

		if p.config.Verbose {
			fmt.Printf("  Discarded changes in %s: %v\n", repo.Name, discardedFiles)
		}
	}

	return nil
}
