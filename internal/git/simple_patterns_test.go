package git

import (
	"fmt"
	"testing"
	"time"
)

func TestNewSimplePatternsDetector(t *testing.T) {
	detector := NewSimplePatternsDetector(0.05, 0.3)
	
	if detector.minSupport != 0.05 {
		t.Errorf("Expected minSupport 0.05, got %f", detector.minSupport)
	}
	
	if detector.minConfidence != 0.3 {
		t.Errorf("Expected minConfidence 0.3, got %f", detector.minConfidence)
	}
	
	if detector.filterFunc == nil {
		t.Error("Expected filterFunc to be initialized")
	}
}

func TestSimplePatternsDetector_SetFileFilter(t *testing.T) {
	detector := NewSimplePatternsDetector(0.05, 0.3)
	
	// Test default filter (should accept all files)
	if !detector.filterFunc("test.go") {
		t.Error("Default filter should accept all files")
	}
	
	// Set custom filter
	detector.SetFileFilter(func(file string) bool {
		return file == "allowed.go"
	})
	
	if !detector.filterFunc("allowed.go") {
		t.Error("Custom filter should accept allowed.go")
	}
	
	if detector.filterFunc("denied.go") {
		t.Error("Custom filter should reject denied.go")
	}
}

func TestSimplePatternsDetector_MineSimplePatterns(t *testing.T) {
	detector := NewSimplePatternsDetector(0.5, 0.5) // High thresholds for testing
	
	// Test with empty commits
	patterns, err := detector.MineSimplePatterns([]CommitInfo{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns for empty commits, got %d", len(patterns))
	}
	
	// Test with single file commits (should be filtered out)
	singleFileCommits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"single.go"},
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "single file commit",
		},
	}
	
	patterns, err = detector.MineSimplePatterns(singleFileCommits)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns for single file commits, got %d", len(patterns))
	}
	
	// Test with multiple file commits
	multiFileCommits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "multi file commit 1",
		},
		{
			Hash:      "def456",
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: time.Now().Add(time.Hour),
			Author:    "test",
			Message:   "multi file commit 2",
		},
	}
	
	patterns, err = detector.MineSimplePatterns(multiFileCommits)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(patterns) != 1 {
		t.Errorf("Expected 1 pattern, got %d", len(patterns))
	}
	
	if len(patterns) > 0 {
		pattern := patterns[0]
		if len(pattern.Items) != 2 {
			t.Errorf("Expected 2 items in pattern, got %d", len(pattern.Items))
		}
		if pattern.Support != 2 {
			t.Errorf("Expected support 2, got %d", pattern.Support)
		}
		if pattern.Frequency != 2 {
			t.Errorf("Expected frequency 2, got %d", pattern.Frequency)
		}
		if pattern.Confidence != 1.0 {
			t.Errorf("Expected confidence 1.0, got %f", pattern.Confidence)
		}
	}
}

func TestSimplePatternsDetector_MineSimplePatternsWithFilter(t *testing.T) {
	detector := NewSimplePatternsDetector(0.3, 0.3)
	
	// Set filter to only include .go files
	detector.SetFileFilter(func(file string) bool {
		return file[len(file)-3:] == ".go"
	})
	
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"main.go", "utils.go", "README.md"},
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "mixed file types",
		},
		{
			Hash:      "def456",
			Files:     []string{"main.go", "utils.go", "config.json"},
			Timestamp: time.Now().Add(time.Hour),
			Author:    "test",
			Message:   "mixed file types 2",
		},
	}
	
	patterns, err := detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Should find pattern for main.go + utils.go (both .go files)
	if len(patterns) != 1 {
		t.Errorf("Expected 1 pattern, got %d", len(patterns))
	}
	
	if len(patterns) > 0 {
		pattern := patterns[0]
		expectedFiles := []string{"main.go", "utils.go"}
		if len(pattern.Items) != len(expectedFiles) {
			t.Errorf("Expected %d items, got %d", len(expectedFiles), len(pattern.Items))
		}
		for i, expected := range expectedFiles {
			if pattern.Items[i] != expected {
				t.Errorf("Expected item %d to be %s, got %s", i, expected, pattern.Items[i])
			}
		}
	}
}

func TestSimplePatternsDetector_calculatePairConfidence(t *testing.T) {
	detector := NewSimplePatternsDetector(0.05, 0.3)
	
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "both files",
		},
		{
			Hash:      "def456",
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: time.Now().Add(time.Hour),
			Author:    "test",
			Message:   "both files again",
		},
		{
			Hash:      "ghi789",
			Files:     []string{"file1.go"},
			Timestamp: time.Now().Add(2 * time.Hour),
			Author:    "test",
			Message:   "only file1",
		},
	}
	
	// Test with valid pair
	confidence := detector.calculatePairConfidence([]string{"file1.go", "file2.go"}, commits)
	expected := 2.0 / 3.0 // 2 commits with both files, 3 commits with file1
	if confidence != expected {
		t.Errorf("Expected confidence %f, got %f", expected, confidence)
	}
	
	// Test with invalid pair (not exactly 2 files)
	confidence = detector.calculatePairConfidence([]string{"file1.go"}, commits)
	if confidence != 0.0 {
		t.Errorf("Expected confidence 0.0 for single file, got %f", confidence)
	}
	
	confidence = detector.calculatePairConfidence([]string{"file1.go", "file2.go", "file3.go"}, commits)
	if confidence != 0.0 {
		t.Errorf("Expected confidence 0.0 for three files, got %f", confidence)
	}
}

func TestSimplePatternName(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			name:     "empty files",
			files:    []string{},
			expected: "Empty Pattern",
		},
		{
			name:     "single file",
			files:    []string{"main.go"},
			expected: "main",
		},
		{
			name:     "two files",
			files:    []string{"main.go", "utils.go"},
			expected: "main + utils",
		},
		{
			name:     "files with paths",
			files:    []string{"src/main.go", "src/utils.go"},
			expected: "main + utils",
		},
		{
			name:     "files with different extensions",
			files:    []string{"app.js", "styles.css"},
			expected: "app + styles",
		},
		{
			name:     "files already sorted",
			files:    []string{"a.go", "b.go", "c.go"},
			expected: "a + b + c",
		},
		{
			name:     "files not sorted",
			files:    []string{"c.go", "a.go", "b.go"},
			expected: "a + b + c",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SimplePatternName(tt.files)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestFrequentItemsetSorting(t *testing.T) {
	detector := NewSimplePatternsDetector(0.1, 0.1)
	
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"high1.go", "high2.go"},
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "high frequency pair 1",
		},
		{
			Hash:      "def456",
			Files:     []string{"high1.go", "high2.go"},
			Timestamp: time.Now().Add(time.Hour),
			Author:    "test",
			Message:   "high frequency pair 2",
		},
		{
			Hash:      "ghi789",
			Files:     []string{"high1.go", "high2.go"},
			Timestamp: time.Now().Add(2 * time.Hour),
			Author:    "test",
			Message:   "high frequency pair 3",
		},
		{
			Hash:      "jkl012",
			Files:     []string{"low1.go", "low2.go"},
			Timestamp: time.Now().Add(3 * time.Hour),
			Author:    "test",
			Message:   "low frequency pair 1",
		},
	}
	
	patterns, err := detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if len(patterns) < 2 {
		t.Fatalf("Expected at least 2 patterns, got %d", len(patterns))
	}
	
	// Should be sorted by frequency (descending)
	if patterns[0].Frequency < patterns[1].Frequency {
		t.Errorf("Patterns should be sorted by frequency (descending)")
	}
	
	// First pattern should be the high frequency one
	if patterns[0].Frequency != 3 {
		t.Errorf("Expected first pattern frequency 3, got %d", patterns[0].Frequency)
	}
}

func TestSimplePatternsDetector_EdgeCases(t *testing.T) {
	detector := NewSimplePatternsDetector(0.05, 0.3)
	
	// Test with very low support threshold
	detector.minSupport = 0.01
	
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"file1.go", "file2.go", "file3.go"},
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "three files",
		},
	}
	
	patterns, err := detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Should find 3 patterns (all pairs: 1-2, 1-3, 2-3)
	if len(patterns) != 3 {
		t.Errorf("Expected 3 patterns for 3 files, got %d", len(patterns))
	}
	
	// Test with very high thresholds using multiple commits with different patterns
	detector.minSupport = 0.99
	detector.minConfidence = 0.99
	
	// Create commits where patterns don't meet high thresholds
	multiCommits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "first commit",
		},
		{
			Hash:      "def456", 
			Files:     []string{"file1.go", "file3.go"},
			Timestamp: time.Now().Add(time.Hour),
			Author:    "test",
			Message:   "second commit",
		},
		{
			Hash:      "ghi789",
			Files:     []string{"file2.go", "file3.go"},
			Timestamp: time.Now().Add(2 * time.Hour),
			Author:    "test",
			Message:   "third commit",
		},
	}
	
	patterns, err = detector.MineSimplePatterns(multiCommits)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Should find no patterns with high thresholds - no pair appears in 99% of commits
	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns with high thresholds, got %d", len(patterns))
	}
}

func TestSimplePatternsDetector_LargeCommitHandling(t *testing.T) {
	detector := NewSimplePatternsDetector(0.05, 0.3)
	
	// Create commit with many files
	manyFiles := make([]string, 50)
	for i := 0; i < 50; i++ {
		manyFiles[i] = fmt.Sprintf("file%d.go", i)
	}
	
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     manyFiles,
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "large commit",
		},
	}
	
	patterns, err := detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Should handle large commits without issues
	// With 50 files, we get 50*49/2 = 1225 pairs
	expectedPairs := 50 * 49 / 2
	if len(patterns) != expectedPairs {
		t.Errorf("Expected %d patterns for 50 files, got %d", expectedPairs, len(patterns))
	}
}

func TestSimplePatternsDetector_TimestampHandling(t *testing.T) {
	detector := NewSimplePatternsDetector(0.05, 0.3)
	
	now := time.Now()
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: now,
			Author:    "test",
			Message:   "first commit",
		},
		{
			Hash:      "def456",
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: now.Add(time.Hour),
			Author:    "test",
			Message:   "second commit",
		},
	}
	
	patterns, err := detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}
	
	pattern := patterns[0]
	
	// LastSeen should be the most recent timestamp
	if !pattern.LastSeen.Equal(now.Add(time.Hour)) {
		t.Errorf("Expected LastSeen to be %v, got %v", now.Add(time.Hour), pattern.LastSeen)
	}
}