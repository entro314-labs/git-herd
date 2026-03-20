package git

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
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

	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, fmt.Errorf("open root %s: %w", rootPath, err)
	}
	defer func() { _ = root.Close() }()

	rootFS := root.FS()

	err = fs.WalkDir(rootFS, ".", func(entryPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
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
		if s.isExcludedPath(entryPath) {
			if entryPath == "." {
				return nil
			}
			return fs.SkipDir
		}

		// Check if this is a git repository
		gitPath := path.Join(entryPath, ".git")
		if _, err := root.Stat(gitPath); err == nil {
			repoPath := rootPath
			if entryPath != "." {
				repoPath = filepath.Join(rootPath, filepath.FromSlash(entryPath))
			}
			repo := types.GitRepo{
				Path:   repoPath,
				Name:   filepath.Base(repoPath),
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
				return fs.SkipDir
			}
		} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return repos, nil
}

func (s *Scanner) isExcludedPath(entryPath string) bool {
	if len(s.config.ExcludeDirs) == 0 {
		return false
	}

	segments := strings.SplitSeq(entryPath, "/")
	for segment := range segments {
		if segment == "." || segment == "" {
			continue
		}
		for _, exclude := range s.config.ExcludeDirs {
			if exclude == "" {
				continue
			}
			if segment == exclude {
				return true
			}
		}
	}

	return false
}
