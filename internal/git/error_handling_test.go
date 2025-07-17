package git

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockErrorGitAnalyzer implements GitAnalyzerInterface but returns errors
type MockErrorGitAnalyzer struct {
	repoPath string
}

func (m *MockErrorGitAnalyzer) IsGitRepository() bool {
	return false
}

func (m *MockErrorGitAnalyzer) GetFileChangeHistory(days int) ([]FileChange, error) {
	return nil, errors.New("mock error: failed to get file change history")
}

func (m *MockErrorGitAnalyzer) GetCommitHistory(days int) ([]CommitInfo, error) {
	return nil, errors.New("mock error: failed to get commit history")
}

func (m *MockErrorGitAnalyzer) GetFileCoOccurrences(days int) (map[string][]string, error) {
	return nil, errors.New("mock error: failed to get file co-occurrences")
}

func (m *MockErrorGitAnalyzer) GetChangeFrequency(days int) (map[string]int, error) {
	return nil, errors.New("mock error: failed to get change frequency")
}

func (m *MockErrorGitAnalyzer) GetLastModified() (map[string]time.Time, error) {
	return nil, errors.New("mock error: failed to get last modified")
}

func (m *MockErrorGitAnalyzer) GetBranchInfo() (string, error) {
	return "", errors.New("mock error: failed to get branch info")
}

func (m *MockErrorGitAnalyzer) GetRemoteInfo() (string, error) {
	return "", errors.New("mock error: failed to get remote info")
}

func (m *MockErrorGitAnalyzer) ExecuteGitCommand(ctx context.Context, args ...string) ([]byte, error) {
	return nil, errors.New("mock error: failed to execute git command")
}

func (m *MockErrorGitAnalyzer) GetRepoPath() string {
	return m.repoPath
}

// TestPatternDetectionErrorHandling tests error handling in pattern detection
func TestPatternDetectionErrorHandling(t *testing.T) {
	mockAnalyzer := &MockErrorGitAnalyzer{repoPath: "."}
	detector := NewPatternDetector(mockAnalyzer)

	// Test DetectChangePatterns with error
	patterns, err := detector.DetectChangePatterns(30)
	if err == nil {
		t.Error("Expected error from DetectChangePatterns")
	}
	if patterns != nil {
		t.Error("Expected nil patterns on error")
	}

	// Test DetectFileRelationships with error
	relationships, err := detector.DetectFileRelationships(30)
	if err == nil {
		t.Error("Expected error from DetectFileRelationships")
	}
	if relationships != nil {
		t.Error("Expected nil relationships on error")
	}
}

// TestSemanticAnalysisErrorHandling tests error handling in semantic analysis
func TestSemanticAnalysisErrorHandling(t *testing.T) {
	// Use current directory which is a git repository
	config := DefaultSemanticConfig()
	analyzer, err := NewSemanticAnalyzer(".", config)
	if err != nil {
		t.Fatalf("Failed to create semantic analyzer: %v", err)
	}

	// Replace with mock that returns errors
	mockAnalyzer := &MockErrorGitAnalyzer{repoPath: "."}
	analyzer.gitAnalyzer = mockAnalyzer
	analyzer.patternDetector = NewPatternDetector(mockAnalyzer)

	// Test that errors are properly handled
	_, err = analyzer.AnalyzeRepository()
	if err == nil {
		t.Error("Expected error from AnalyzeRepository")
	}
}

// TestGitAnalyzerErrorRecovery tests error recovery in GitAnalyzer
func TestGitAnalyzerErrorRecovery(t *testing.T) {
	// Test with non-existent repository
	analyzer, err := NewGitAnalyzer("/non/existent/path")
	if err != nil {
		t.Skip("Expected behavior - git analyzer creation failed for non-existent path")
		return
	}

	// Test IsGitRepository with invalid path
	if analyzer.IsGitRepository() {
		t.Error("Expected false for non-existent repository")
	}

	// Test other operations should handle errors gracefully
	_, err = analyzer.GetCommitHistory(30)
	if err == nil {
		t.Error("Expected error for non-existent repository")
	}
}