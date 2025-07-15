package git

import (
	"testing"
	"time"
)

func TestNewGitAnalyzer(t *testing.T) {
	tests := []struct {
		name      string
		repoPath  string
		wantError bool
	}{
		{
			name:      "valid git repository",
			repoPath:  ".",
			wantError: false,
		},
		{
			name:      "non-git directory",
			repoPath:  "/tmp",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer, err := NewGitAnalyzer(tt.repoPath)
			
			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if analyzer == nil {
				t.Error("expected analyzer but got nil")
			}
		})
	}
}

func TestGitAnalyzer_IsGitRepository(t *testing.T) {
	// Test with current directory (should be a git repo)
	analyzer := &GitAnalyzer{
		repoPath: ".",
		gitPath:  "git",
	}
	
	if !analyzer.IsGitRepository() {
		t.Error("expected current directory to be a git repository")
	}
	
	// Test with non-git directory
	analyzer.repoPath = "/tmp"
	if analyzer.IsGitRepository() {
		t.Error("expected /tmp to not be a git repository")
	}
}

func TestGitAnalyzer_GetFileChangeHistory(t *testing.T) {
	analyzer, err := NewGitAnalyzer(".")
	if err != nil {
		t.Skipf("skipping test: %v", err)
	}

	changes, err := analyzer.GetFileChangeHistory(30)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Basic validation
	for _, change := range changes {
		if change.FilePath == "" {
			t.Error("expected non-empty file path")
		}
		if change.CommitHash == "" {
			t.Error("expected non-empty commit hash")
		}
		if change.Timestamp.IsZero() {
			t.Error("expected non-zero timestamp")
		}
		if change.ChangeType == "" {
			t.Error("expected non-empty change type")
		}
	}

	t.Logf("Found %d file changes in last 30 days", len(changes))
}

func TestGitAnalyzer_GetCommitHistory(t *testing.T) {
	analyzer, err := NewGitAnalyzer(".")
	if err != nil {
		t.Skipf("skipping test: %v", err)
	}

	commits, err := analyzer.GetCommitHistory(30)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Basic validation
	for _, commit := range commits {
		if commit.Hash == "" {
			t.Error("expected non-empty commit hash")
		}
		if commit.Author == "" {
			t.Error("expected non-empty author")
		}
		if commit.Timestamp.IsZero() {
			t.Error("expected non-zero timestamp")
		}
		// Files can be empty for some commits
	}

	t.Logf("Found %d commits in last 30 days", len(commits))
}

func TestGitAnalyzer_GetFileCoOccurrences(t *testing.T) {
	analyzer, err := NewGitAnalyzer(".")
	if err != nil {
		t.Skipf("skipping test: %v", err)
	}

	coOccurrences, err := analyzer.GetFileCoOccurrences(30)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Validate structure
	for file, partners := range coOccurrences {
		if file == "" {
			t.Error("expected non-empty file path")
		}
		if len(partners) == 0 {
			t.Error("expected at least one partner")
		}
		
		for _, partner := range partners {
			if partner == "" {
				t.Error("expected non-empty partner file path")
			}
			if partner == file {
				t.Error("file should not be its own partner")
			}
		}
	}

	t.Logf("Found co-occurrence patterns for %d files", len(coOccurrences))
}

func TestGitAnalyzer_GetChangeFrequency(t *testing.T) {
	analyzer, err := NewGitAnalyzer(".")
	if err != nil {
		t.Skipf("skipping test: %v", err)
	}

	frequency, err := analyzer.GetChangeFrequency(30)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Validate structure
	for file, count := range frequency {
		if file == "" {
			t.Error("expected non-empty file path")
		}
		if count <= 0 {
			t.Error("expected positive change count")
		}
	}

	t.Logf("Found change frequencies for %d files", len(frequency))
}

func TestGitAnalyzer_GetLastModified(t *testing.T) {
	analyzer, err := NewGitAnalyzer(".")
	if err != nil {
		t.Skipf("skipping test: %v", err)
	}

	lastModified, err := analyzer.GetLastModified()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Validate structure
	for file, timestamp := range lastModified {
		if file == "" {
			t.Error("expected non-empty file path")
		}
		if timestamp.IsZero() {
			t.Error("expected non-zero timestamp")
		}
	}

	t.Logf("Found last modified times for %d files", len(lastModified))
}

func TestGitAnalyzer_GetBranchInfo(t *testing.T) {
	analyzer, err := NewGitAnalyzer(".")
	if err != nil {
		t.Skipf("skipping test: %v", err)
	}

	branch, err := analyzer.GetBranchInfo()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if branch == "" {
		t.Error("expected non-empty branch name")
	}

	t.Logf("Current branch: %s", branch)
}

func TestGitAnalyzer_GetRemoteInfo(t *testing.T) {
	analyzer, err := NewGitAnalyzer(".")
	if err != nil {
		t.Skipf("skipping test: %v", err)
	}

	remote, err := analyzer.GetRemoteInfo()
	if err != nil {
		// Remote might not exist, that's okay
		t.Logf("No remote found (expected for some repos): %v", err)
		return
	}

	if remote == "" {
		t.Error("expected non-empty remote URL")
	}

	t.Logf("Remote URL: %s", remote)
}

func TestParseFileChanges(t *testing.T) {
	analyzer := &GitAnalyzer{
		repoPath: ".",
		gitPath:  "git",
	}

	// Sample git log output with file changes
	output := `abc123|John Doe|john@example.com|1640995200|Initial commit
A	README.md
A	main.go

def456|Jane Smith|jane@example.com|1640995300|Add tests
M	main.go
A	main_test.go`

	changes, err := analyzer.parseFileChanges(output)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	expected := []FileChange{
		{
			FilePath:   "README.md",
			ChangeType: "A",
			CommitHash: "abc123",
			Author:     "John Doe",
			Message:    "Initial commit",
		},
		{
			FilePath:   "main.go",
			ChangeType: "A",
			CommitHash: "abc123",
			Author:     "John Doe",
			Message:    "Initial commit",
		},
		{
			FilePath:   "main.go",
			ChangeType: "M",
			CommitHash: "def456",
			Author:     "Jane Smith",
			Message:    "Add tests",
		},
		{
			FilePath:   "main_test.go",
			ChangeType: "A",
			CommitHash: "def456",
			Author:     "Jane Smith",
			Message:    "Add tests",
		},
	}

	if len(changes) != len(expected) {
		t.Errorf("expected %d changes, got %d", len(expected), len(changes))
		return
	}

	for i, change := range changes {
		if change.FilePath != expected[i].FilePath {
			t.Errorf("expected file path %s, got %s", expected[i].FilePath, change.FilePath)
		}
		if change.ChangeType != expected[i].ChangeType {
			t.Errorf("expected change type %s, got %s", expected[i].ChangeType, change.ChangeType)
		}
		if change.CommitHash != expected[i].CommitHash {
			t.Errorf("expected commit hash %s, got %s", expected[i].CommitHash, change.CommitHash)
		}
		if change.Author != expected[i].Author {
			t.Errorf("expected author %s, got %s", expected[i].Author, change.Author)
		}
		if change.Message != expected[i].Message {
			t.Errorf("expected message %s, got %s", expected[i].Message, change.Message)
		}
	}
}

func TestParseCommitHistory(t *testing.T) {
	analyzer := &GitAnalyzer{
		repoPath: ".",
		gitPath:  "git",
	}

	// Sample git log output with commit history
	output := `abc123|John Doe|john@example.com|1640995200|Initial commit
README.md
main.go

def456|Jane Smith|jane@example.com|1640995300|Add tests
main.go
main_test.go`

	commits, err := analyzer.parseCommitHistory(output)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	expected := []CommitInfo{
		{
			Hash:      "abc123",
			Author:    "John Doe",
			Email:     "john@example.com",
			Timestamp: time.Unix(1640995200, 0),
			Message:   "Initial commit",
			Files:     []string{"README.md", "main.go"},
		},
		{
			Hash:      "def456",
			Author:    "Jane Smith",
			Email:     "jane@example.com",
			Timestamp: time.Unix(1640995300, 0),
			Message:   "Add tests",
			Files:     []string{"main.go", "main_test.go"},
		},
	}

	if len(commits) != len(expected) {
		t.Errorf("expected %d commits, got %d", len(expected), len(commits))
		return
	}

	for i, commit := range commits {
		if commit.Hash != expected[i].Hash {
			t.Errorf("expected hash %s, got %s", expected[i].Hash, commit.Hash)
		}
		if commit.Author != expected[i].Author {
			t.Errorf("expected author %s, got %s", expected[i].Author, commit.Author)
		}
		if commit.Email != expected[i].Email {
			t.Errorf("expected email %s, got %s", expected[i].Email, commit.Email)
		}
		if commit.Message != expected[i].Message {
			t.Errorf("expected message %s, got %s", expected[i].Message, commit.Message)
		}
		if len(commit.Files) != len(expected[i].Files) {
			t.Errorf("expected %d files, got %d", len(expected[i].Files), len(commit.Files))
			continue
		}
		for j, file := range commit.Files {
			if file != expected[i].Files[j] {
				t.Errorf("expected file %s, got %s", expected[i].Files[j], file)
			}
		}
	}
}

// Benchmark tests
func BenchmarkGetFileChangeHistory(b *testing.B) {
	analyzer, err := NewGitAnalyzer(".")
	if err != nil {
		b.Skipf("skipping benchmark: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.GetFileChangeHistory(30)
		if err != nil {
			b.Errorf("unexpected error: %v", err)
		}
	}
}

func BenchmarkGetFileCoOccurrences(b *testing.B) {
	analyzer, err := NewGitAnalyzer(".")
	if err != nil {
		b.Skipf("skipping benchmark: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.GetFileCoOccurrences(30)
		if err != nil {
			b.Errorf("unexpected error: %v", err)
		}
	}
}