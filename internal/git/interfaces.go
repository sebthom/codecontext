package git

import (
	"context"
	"time"
)

// GitAnalyzerInterface defines the interface for git analysis operations
type GitAnalyzerInterface interface {
	IsGitRepository() bool
	GetFileChangeHistory(days int) ([]FileChange, error)
	GetCommitHistory(days int) ([]CommitInfo, error)
	GetFileCoOccurrences(days int) (map[string][]string, error)
	GetChangeFrequency(days int) (map[string]int, error)
	GetLastModified() (map[string]time.Time, error)
	GetBranchInfo() (string, error)
	GetRemoteInfo() (string, error)
	ExecuteGitCommand(ctx context.Context, args ...string) ([]byte, error)
	GetRepoPath() string
}

// Ensure GitAnalyzer implements GitAnalyzerInterface
var _ GitAnalyzerInterface = (*GitAnalyzer)(nil)