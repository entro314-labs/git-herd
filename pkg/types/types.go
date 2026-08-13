package types

import (
	"errors"
	"fmt"
	"time"
)

// OperationType defines the type of git operation to perform
type OperationType string

const (
	OperationFetch OperationType = "fetch"
	OperationPull  OperationType = "pull"
	OperationScan  OperationType = "scan"
)

var ErrRepoSkipped = errors.New("repository skipped")

// ErrBrokenGitlink marks a working tree whose ".git" is a gitlink file pointing
// at a git directory that does not exist. This is common for submodules in
// repositories cloned without --recurse-submodules, where ".git/modules/<name>"
// was never populated. Reported separately so it is not confused with a
// genuinely missing repository.
var ErrBrokenGitlink = errors.New("broken gitlink")

// GitRepo represents a git repository with its path and status
type GitRepo struct {
	Path          string
	Name          string
	HasGit        bool
	IsSubmodule   bool   // ".git" is a gitlink file rather than a directory
	GitDir        string // Resolved git directory (differs from Path/.git for submodules)
	Clean         bool
	Branch        string
	Remote        string
	Error         error
	Duration      time.Duration
	LastCommit    string   // Last commit hash
	LastCommitMsg string   // Last commit message
	ModifiedFiles []string // List of modified files
}

// Config holds application configuration
type Config struct {
	Workers        int           `mapstructure:"workers" json:"workers,omitzero"`
	Operation      OperationType `mapstructure:"operation" json:"operation,omitzero"`
	DryRun         bool          `mapstructure:"dry-run" json:"dry_run,omitzero"`
	Recursive      bool          `mapstructure:"recursive" json:"recursive,omitzero"`
	SkipDirty      bool          `mapstructure:"skip-dirty" json:"skip_dirty,omitzero"`
	WithSubmodules bool          `mapstructure:"with-submodules" json:"with_submodules,omitzero"` // Operate on submodules as standalone repos
	Verbose        bool          `mapstructure:"verbose" json:"verbose,omitzero"`
	Timeout        time.Duration `mapstructure:"timeout" json:"timeout,omitzero"`
	ExcludeDirs    []string      `mapstructure:"exclude" json:"exclude_dirs,omitzero"`
	PlainMode      bool          `mapstructure:"plain" json:"plain_mode,omitzero"`            // Disable TUI for plain text output
	FullSummary    bool          `mapstructure:"full-summary" json:"full_summary,omitzero"`   // Show full summary of all repositories
	SaveReport     string        `mapstructure:"save-report" json:"save_report,omitzero"`     // File path to save detailed report
	DiscardFiles   []string      `mapstructure:"discard-files" json:"discard_files,omitzero"` // File patterns to discard before pull/fetch
	ExportScan     string        `mapstructure:"export-scan" json:"export_scan,omitzero"`     // Export scan results to markdown file
}

// GitRepoResult represents the result of processing a git repository
type GitRepoResult struct {
	Repo      GitRepo
	Success   bool
	Skipped   bool
	StartTime time.Time
	EndTime   time.Time
}

// ProcessingStats holds statistics about the processing session
type ProcessingStats struct {
	Total      int
	Successful int
	Failed     int
	Skipped    int
	StartTime  time.Time
	EndTime    time.Time
}

// Summary returns a formatted summary of the stats
func (s *ProcessingStats) Summary() string {
	if s == nil {
		return ""
	}

	duration := time.Duration(0)
	if !s.StartTime.IsZero() && !s.EndTime.IsZero() && s.EndTime.After(s.StartTime) {
		duration = s.EndTime.Sub(s.StartTime).Truncate(time.Millisecond)
	}

	summary := fmt.Sprintf("Processed %d repositories", s.Total)
	if duration > 0 {
		summary = fmt.Sprintf("%s in %s", summary, duration)
	}

	return fmt.Sprintf("%s: %d successful, %d failed, %d skipped", summary, s.Successful, s.Failed, s.Skipped)
}
