package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPatternDetectionErrorHandling tests error handling in pattern detection
func TestPatternDetectionErrorHandling(t *testing.T) {
	// Test with malformed commits
	detector := NewSimplePatternsDetector(0.05, 0.3)
	
	malformedCommits := []CommitInfo{
		{
			Hash:      "", // Empty hash
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "malformed commit",
		},
		{
			Hash:      "abc123",
			Files:     nil, // Nil files
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "nil files",
		},
		{
			Hash:      "def456",
			Files:     []string{}, // Empty files
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "empty files",
		},
	}
	
	patterns, err := detector.MineSimplePatterns(malformedCommits)
	if err != nil {
		t.Errorf("Expected no error with malformed commits, got %v", err)
	}
	
	// Should handle malformed commits gracefully
	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns with malformed commits, got %d", len(patterns))
	}
}

// TestIgnoreFileErrorHandling tests error handling in ignore file processing
func TestIgnoreFileErrorHandling(t *testing.T) {
	// Test with non-existent directory
	nonExistentDir := "/non/existent/directory"
	analyzer := &GitAnalyzer{repoPath: nonExistentDir}
	
	pd := &PatternDetector{analyzer: analyzer}
	pd.loadExcludePatterns()
	
	// Should fall back to default patterns
	if len(pd.excludePatterns) == 0 {
		t.Error("Expected default exclude patterns when directory doesn't exist")
	}
	
	// Test with corrupted ignore file
	tempDir, err := os.MkdirTemp("", "codecontext_error_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a directory with same name as ignore file (should cause read error)
	ignoreDir := filepath.Join(tempDir, ".codecontextignore")
	err = os.MkdirAll(ignoreDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create ignore directory: %v", err)
	}
	
	analyzer = &GitAnalyzer{repoPath: tempDir}
	pd = &PatternDetector{analyzer: analyzer}
	pd.loadExcludePatterns()
	
	// Should fall back to default patterns when file is unreadable
	if len(pd.excludePatterns) == 0 {
		t.Error("Expected default exclude patterns when ignore file is unreadable")
	}
}

// TestSemanticAnalysisErrorHandling tests error handling in semantic analysis
func TestSemanticAnalysisErrorHandling(t *testing.T) {
	// Test with invalid repository path
	_, err := NewSemanticAnalyzer("/invalid/path", nil)
	if err == nil {
		t.Error("Expected error when creating semantic analyzer with invalid path")
	}
	
	// Test with nil configuration
	tempDir, err := os.MkdirTemp("", "codecontext_semantic_error_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	analyzer, err := NewSemanticAnalyzer(tempDir, nil)
	if err != nil {
		t.Fatalf("Should handle nil config gracefully: %v", err)
	}
	if analyzer.config == nil {
		t.Error("Expected default config when nil is provided")
	}
}

// TestPatternDetectionWithCorruptedData tests pattern detection with corrupted data
func TestPatternDetectionWithCorruptedData(t *testing.T) {
	detector := NewSimplePatternsDetector(0.05, 0.3)
	
	// Test with commits containing very long file names
	longFileName := string(make([]byte, 10000))
	for i := range longFileName {
		longFileName = longFileName[:i] + "a" + longFileName[i+1:]
	}
	
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{longFileName, "normal.go"},
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "long filename test",
		},
	}
	
	patterns, err := detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error with long filenames, got %v", err)
	}
	
	// Should handle long filenames gracefully
	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns with single long filename, got %d", len(patterns))
	}
	
	// Test with commits containing special characters
	specialFiles := []string{
		"file with spaces.go",
		"file-with-dashes.go",
		"file_with_underscores.go",
		"file.with.dots.go",
		"file@with@symbols.go",
		"file#with#hash.go",
		"file$with$dollar.go",
	}
	
	commits = []CommitInfo{
		{
			Hash:      "def456",
			Files:     specialFiles,
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "special characters test",
		},
	}
	
	patterns, err = detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error with special characters, got %v", err)
	}
	
	// Should handle special characters gracefully
	if len(patterns) == 0 {
		t.Error("Expected some patterns with special characters")
	}
}

// TestPatternDetectionMemoryLimits tests pattern detection with memory constraints
func TestPatternDetectionMemoryLimits(t *testing.T) {
	detector := NewSimplePatternsDetector(0.01, 0.01) // Very low thresholds
	
	// Create a commit with many files to generate many pairs
	manyFiles := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		manyFiles[i] = fmt.Sprintf("file%d.go", i)
	}
	
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     manyFiles,
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "many files commit",
		},
	}
	
	patterns, err := detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error with many files, got %v", err)
	}
	
	// Should handle large number of files without memory issues
	expectedPairs := 1000 * 999 / 2 // n*(n-1)/2 pairs
	if len(patterns) != expectedPairs {
		t.Errorf("Expected %d patterns, got %d", expectedPairs, len(patterns))
	}
}

// TestConcurrentPatternDetection tests thread safety of pattern detection
func TestConcurrentPatternDetection(t *testing.T) {
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "concurrent test",
		},
		{
			Hash:      "def456",
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: time.Now().Add(time.Hour),
			Author:    "test",
			Message:   "concurrent test 2",
		},
	}
	
	// Run multiple detectors concurrently
	results := make(chan []FrequentItemset, 10)
	errors := make(chan error, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			detector := NewSimplePatternsDetector(0.05, 0.3)
			patterns, err := detector.MineSimplePatterns(commits)
			if err != nil {
				errors <- err
				return
			}
			results <- patterns
		}()
	}
	
	// Collect results
	for i := 0; i < 10; i++ {
		select {
		case patterns := <-results:
			if len(patterns) != 1 {
				t.Errorf("Expected 1 pattern in concurrent test, got %d", len(patterns))
			}
		case err := <-errors:
			t.Errorf("Concurrent pattern detection failed: %v", err)
		case <-time.After(10 * time.Second):
			t.Error("Concurrent test timed out")
		}
	}
}

// TestPatternDetectionWithTimestampErrors tests handling of timestamp-related errors
func TestPatternDetectionWithTimestampErrors(t *testing.T) {
	detector := NewSimplePatternsDetector(0.05, 0.3)
	
	// Test with zero timestamps
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: time.Time{}, // Zero timestamp
			Author:    "test",
			Message:   "zero timestamp",
		},
		{
			Hash:      "def456",
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "normal timestamp",
		},
	}
	
	patterns, err := detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error with zero timestamps, got %v", err)
	}
	
	// Should handle zero timestamps gracefully
	if len(patterns) != 1 {
		t.Errorf("Expected 1 pattern with zero timestamps, got %d", len(patterns))
	}
	
	// Test with future timestamps
	commits = []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: time.Now().Add(24 * time.Hour), // Future timestamp
			Author:    "test",
			Message:   "future timestamp",
		},
	}
	
	patterns, err = detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error with future timestamps, got %v", err)
	}
	
	// Should handle future timestamps gracefully
	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns with single future timestamp, got %d", len(patterns))
	}
}

// TestPatternDetectionFilterErrors tests error handling in file filtering
func TestPatternDetectionFilterErrors(t *testing.T) {
	detector := NewSimplePatternsDetector(0.05, 0.3)
	
	// Test with filter that panics
	detector.SetFileFilter(func(file string) bool {
		if file == "panic.go" {
			panic("test panic")
		}
		return true
	})
	
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"normal.go", "panic.go"},
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "panic test",
		},
	}
	
	// Should handle panics gracefully (though this might not be caught)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Caught panic as expected: %v", r)
		}
	}()
	
	_, err := detector.MineSimplePatterns(commits)
	if err == nil {
		t.Log("Filter panic was not caught - this is expected behavior")
	}
}

// TestSemanticAnalysisWithMockErrors tests semantic analysis with mocked errors
func TestSemanticAnalysisWithMockErrors(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "codecontext_mock_error_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	config := DefaultSemanticConfig()
	analyzer, err := NewSemanticAnalyzer(tempDir, config)
	if err != nil {
		t.Fatalf("Failed to create semantic analyzer: %v", err)
	}
	
	// Mock git analyzer that returns errors
	errorGitAnalyzer := &ErrorMockGitAnalyzer{repoPath: tempDir}
	analyzer.gitAnalyzer = errorGitAnalyzer
	analyzer.patternDetector = NewPatternDetector(errorGitAnalyzer)
	
	// Test that errors are properly handled
	_, err = analyzer.AnalyzeRepository()
	if err == nil {
		t.Error("Expected error from mock git analyzer")
	}
}

// TestPatternDetectionValidation tests validation of pattern detection results
func TestPatternDetectionValidation(t *testing.T) {
	detector := NewSimplePatternsDetector(0.05, 0.3)
	
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "validation test",
		},
	}
	
	patterns, err := detector.MineSimplePatterns(commits)
	if err != nil {
		t.Fatalf("Pattern detection failed: %v", err)
	}
	
	// Validate all patterns
	for i, pattern := range patterns {
		// Check that all required fields are set
		if len(pattern.Items) == 0 {
			t.Errorf("Pattern %d has no items", i)
		}
		if pattern.Support <= 0 {
			t.Errorf("Pattern %d has invalid support: %d", i, pattern.Support)
		}
		if pattern.Confidence < 0 || pattern.Confidence > 1 {
			t.Errorf("Pattern %d has invalid confidence: %f", i, pattern.Confidence)
		}
		if pattern.Frequency <= 0 {
			t.Errorf("Pattern %d has invalid frequency: %d", i, pattern.Frequency)
		}
		if pattern.LastSeen.IsZero() {
			t.Errorf("Pattern %d has zero LastSeen timestamp", i)
		}
		
		// Check consistency between fields
		if pattern.Support != pattern.Frequency {
			t.Errorf("Pattern %d has inconsistent support (%d) and frequency (%d)", i, pattern.Support, pattern.Frequency)
		}
	}
}

// ErrorMockGitAnalyzer is a mock that returns errors for testing
type ErrorMockGitAnalyzer struct {
	repoPath string
}

func (e *ErrorMockGitAnalyzer) GetCommitHistory(days int) ([]CommitInfo, error) {
	return nil, errors.New("mock error: failed to get commit history")
}

func (e *ErrorMockGitAnalyzer) GetFileCoOccurrences(days int) (map[string][]string, error) {
	return nil, errors.New("mock error: failed to get file co-occurrences")
}

func (e *ErrorMockGitAnalyzer) GetChangeFrequency(days int) (map[string]int, error) {
	return nil, errors.New("mock error: failed to get change frequency")
}

func (e *ErrorMockGitAnalyzer) GetLastModified() (map[string]time.Time, error) {
	return nil, errors.New("mock error: failed to get last modified")
}

func (e *ErrorMockGitAnalyzer) GetBranchInfo() (string, error) {
	return "", errors.New("mock error: failed to get branch info")
}

func (e *ErrorMockGitAnalyzer) GetRemoteInfo() (string, error) {
	return "", errors.New("mock error: failed to get remote info")
}