package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GitAnalyzer provides git repository analysis capabilities
type GitAnalyzer struct {
	repoPath string
	gitPath  string
}

// NewGitAnalyzer creates a new GitAnalyzer instance
func NewGitAnalyzer(repoPath string) (*GitAnalyzer, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git not found in PATH: %w", err)
	}

	analyzer := &GitAnalyzer{
		repoPath: repoPath,
		gitPath:  gitPath,
	}

	// Verify it's a git repository
	if !analyzer.IsGitRepository() {
		return nil, fmt.Errorf("not a git repository: %s", repoPath)
	}

	return analyzer, nil
}

// IsGitRepository checks if the directory is a git repository
func (g *GitAnalyzer) IsGitRepository() bool {
	cmd := exec.Command(g.gitPath, "rev-parse", "--git-dir")
	cmd.Dir = g.repoPath
	return cmd.Run() == nil
}

// FileChange represents a file change in a commit
type FileChange struct {
	FilePath   string
	ChangeType string // A, M, D, R, C (Added, Modified, Deleted, Renamed, Copied)
	CommitHash string
	Timestamp  time.Time
	Author     string
	Message    string
}

// CommitInfo represents commit information
type CommitInfo struct {
	Hash      string
	Author    string
	Email     string
	Timestamp time.Time
	Message   string
	Files     []string
}

// GetFileChangeHistory returns file changes for the specified time period
func (g *GitAnalyzer) GetFileChangeHistory(days int) ([]FileChange, error) {
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	
	cmd := exec.Command(g.gitPath, "log", 
		"--name-status", 
		"--pretty=format:%H|%an|%ae|%at|%s", 
		fmt.Sprintf("--since=%s", since),
		"--no-merges")
	cmd.Dir = g.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git log: %w", err)
	}

	return g.parseFileChanges(string(output))
}

// GetCommitHistory returns commit information for the specified time period
func (g *GitAnalyzer) GetCommitHistory(days int) ([]CommitInfo, error) {
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	
	cmd := exec.Command(g.gitPath, "log", 
		"--name-only",
		"--pretty=format:%H|%an|%ae|%at|%s",
		fmt.Sprintf("--since=%s", since),
		"--no-merges")
	cmd.Dir = g.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit history: %w", err)
	}

	return g.parseCommitHistory(string(output))
}

// GetFileCoOccurrences returns files that frequently change together
func (g *GitAnalyzer) GetFileCoOccurrences(days int) (map[string][]string, error) {
	commits, err := g.GetCommitHistory(days)
	if err != nil {
		return nil, err
	}

	// Count co-occurrences
	coOccurrences := make(map[string]map[string]int)
	
	for _, commit := range commits {
		if len(commit.Files) <= 1 {
			continue // Skip single-file commits
		}

		// Count each pair of files that changed together
		for i, file1 := range commit.Files {
			if coOccurrences[file1] == nil {
				coOccurrences[file1] = make(map[string]int)
			}
			
			for j, file2 := range commit.Files {
				if i != j {
					coOccurrences[file1][file2]++
				}
			}
		}
	}

	// Convert to ranked lists
	result := make(map[string][]string)
	for file, partners := range coOccurrences {
		// Sort partners by frequency
		type pair struct {
			file  string
			count int
		}
		
		var pairs []pair
		for partner, count := range partners {
			pairs = append(pairs, pair{partner, count})
		}
		
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].count > pairs[j].count
		})
		
		// Take top partners (minimum 2 co-occurrences)
		var topPartners []string
		for _, p := range pairs {
			if p.count >= 2 {
				topPartners = append(topPartners, p.file)
			}
		}
		
		if len(topPartners) > 0 {
			result[file] = topPartners
		}
	}

	return result, nil
}

// GetChangeFrequency returns how often each file changes
func (g *GitAnalyzer) GetChangeFrequency(days int) (map[string]int, error) {
	changes, err := g.GetFileChangeHistory(days)
	if err != nil {
		return nil, err
	}

	frequency := make(map[string]int)
	for _, change := range changes {
		frequency[change.FilePath]++
	}

	return frequency, nil
}

// GetLastModified returns the last modification time for each file
func (g *GitAnalyzer) GetLastModified() (map[string]time.Time, error) {
	cmd := exec.Command(g.gitPath, "log", "--name-only", "--pretty=format:%at", "-1")
	cmd.Dir = g.repoPath

	// Get individual file last modified times
	cmd = exec.Command(g.gitPath, "ls-files", "-z")
	cmd.Dir = g.repoPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	files := strings.Split(string(output), "\000")
	result := make(map[string]time.Time)
	
	for _, file := range files {
		if file == "" {
			continue
		}
		
		// Get last commit for this file
		cmd := exec.Command(g.gitPath, "log", "-1", "--pretty=format:%at", "--", file)
		cmd.Dir = g.repoPath
		
		output, err := cmd.Output()
		if err != nil {
			continue // Skip files that can't be queried
		}
		
		if timestamp, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64); err == nil {
			result[file] = time.Unix(timestamp, 0)
		}
	}

	return result, nil
}

// parseFileChanges parses git log output with file changes
func (g *GitAnalyzer) parseFileChanges(output string) ([]FileChange, error) {
	var changes []FileChange
	lines := strings.Split(output, "\n")
	
	var currentCommit CommitInfo
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Check if this is a commit line (contains |)
		if strings.Contains(line, "|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 5 {
				timestamp, _ := strconv.ParseInt(parts[3], 10, 64)
				currentCommit = CommitInfo{
					Hash:      parts[0],
					Author:    parts[1],
					Email:     parts[2],
					Timestamp: time.Unix(timestamp, 0),
					Message:   parts[4],
				}
			}
		} else {
			// This is a file change line
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				changes = append(changes, FileChange{
					FilePath:   parts[1],
					ChangeType: parts[0],
					CommitHash: currentCommit.Hash,
					Timestamp:  currentCommit.Timestamp,
					Author:     currentCommit.Author,
					Message:    currentCommit.Message,
				})
			}
		}
	}
	
	return changes, nil
}

// parseCommitHistory parses git log output with commit information
func (g *GitAnalyzer) parseCommitHistory(output string) ([]CommitInfo, error) {
	var commits []CommitInfo
	lines := strings.Split(output, "\n")
	
	var currentCommit *CommitInfo
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Check if this is a commit line (contains |)
		if strings.Contains(line, "|") {
			// Save previous commit if exists
			if currentCommit != nil {
				commits = append(commits, *currentCommit)
			}
			
			parts := strings.Split(line, "|")
			if len(parts) >= 5 {
				timestamp, _ := strconv.ParseInt(parts[3], 10, 64)
				currentCommit = &CommitInfo{
					Hash:      parts[0],
					Author:    parts[1],
					Email:     parts[2],
					Timestamp: time.Unix(timestamp, 0),
					Message:   parts[4],
					Files:     []string{},
				}
			}
		} else if currentCommit != nil {
			// This is a file in the current commit
			if line != "" {
				currentCommit.Files = append(currentCommit.Files, line)
			}
		}
	}
	
	// Don't forget the last commit
	if currentCommit != nil {
		commits = append(commits, *currentCommit)
	}
	
	return commits, nil
}

// ExecuteGitCommand executes a git command with proper error handling
func (g *GitAnalyzer) ExecuteGitCommand(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, g.gitPath, args...)
	cmd.Dir = g.repoPath
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git command failed: %w, stderr: %s", err, stderr.String())
	}
	
	return output, nil
}

// GetBranchInfo returns current branch information
func (g *GitAnalyzer) GetBranchInfo() (string, error) {
	output, err := g.ExecuteGitCommand(context.Background(), "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(string(output)), nil
}

// GetRemoteInfo returns remote repository information
func (g *GitAnalyzer) GetRemoteInfo() (string, error) {
	output, err := g.ExecuteGitCommand(context.Background(), "remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(string(output)), nil
}

// GetRepoPath returns the repository path
func (g *GitAnalyzer) GetRepoPath() string {
	return g.repoPath
}