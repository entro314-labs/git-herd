package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/entro314-labs/git-herd/pkg/types"
)

// Scanner handles discovering git repositories in a directory tree
type Scanner struct {
	config *types.Config
}

// NewScanner creates a new git repository scanner
func NewScanner(config *types.Config) *Scanner {
	return &Scanner{
		config: config,
	}
}

// FindRepos discovers all git repositories in the given directory
func (s *Scanner) FindRepos(ctx context.Context, rootPath string, onProgress func(int)) ([]types.GitRepo, error) {
	var repos []types.GitRepo
	var mu sync.Mutex
	var foundCount int

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip if not a directory
		if !d.IsDir() {
			return nil
		}

		// Check if we should exclude this directory
		for _, exclude := range s.config.ExcludeDirs {
			if strings.Contains(path, exclude) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Check if this is a git repository
		gitPath := filepath.Join(path, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			repo := types.GitRepo{
				Path:   path,
				Name:   filepath.Base(path),
				HasGit: true,
			}

			// Don't analyze repo here - defer to processing phase for better performance
			mu.Lock()
			repos = append(repos, repo)
			foundCount++
			currentCount := foundCount
			mu.Unlock()

			if onProgress != nil {
				onProgress(currentCount)
			}

			// Skip subdirectories if not recursive
			if !s.config.Recursive {
				return filepath.SkipDir
			}
		}

		return nil
	})

	return repos, err
}
