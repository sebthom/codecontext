package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPatternDetectionIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "codecontext_integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a mock git analyzer
	analyzer := &GitAnalyzer{repoPath: tempDir}
	
	// Create pattern detector
	pd := NewPatternDetector(analyzer)
	pd.SetThresholds(0.3, 0.3) // Lower thresholds for testing
	
	// Mock commits with realistic patterns
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"src/main.go", "src/utils.go"},
			Timestamp: time.Now().Add(-24 * time.Hour),
			Author:    "dev1",
			Message:   "Add main functionality",
		},
		{
			Hash:      "def456",
			Files:     []string{"src/main.go", "src/utils.go"},
			Timestamp: time.Now().Add(-20 * time.Hour),
			Author:    "dev2",
			Message:   "Update main and utils",
		},
		{
			Hash:      "ghi789",
			Files:     []string{"src/main.go", "src/utils.go"},
			Timestamp: time.Now().Add(-16 * time.Hour),
			Author:    "dev1",
			Message:   "Fix main and utils",
		},
		{
			Hash:      "jkl012",
			Files:     []string{"src/handler.go", "src/middleware.go"},
			Timestamp: time.Now().Add(-12 * time.Hour),
			Author:    "dev3",
			Message:   "Add handler and middleware",
		},
		{
			Hash:      "mno345",
			Files:     []string{"src/handler.go", "src/middleware.go"},
			Timestamp: time.Now().Add(-8 * time.Hour),
			Author:    "dev2",
			Message:   "Update handler and middleware",
		},
		{
			Hash:      "pqr678",
			Files:     []string{"src/main.go", "README.md"},
			Timestamp: time.Now().Add(-4 * time.Hour),
			Author:    "dev1",
			Message:   "Update main and docs",
		},
	}
	
	// Test simple patterns detection
	detector := NewSimplePatternsDetector(0.3, 0.3)
	detector.SetFileFilter(func(file string) bool {
		return filepath.Ext(file) == ".go" || filepath.Base(file) == "README.md"
	})
	
	patterns, err := detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if len(patterns) == 0 {
		t.Error("Expected to find some patterns")
	}
	
	// Verify pattern structure
	for i, pattern := range patterns {
		if len(pattern.Items) != 2 {
			t.Errorf("Pattern %d should have 2 items, got %d", i, len(pattern.Items))
		}
		if pattern.Support <= 0 {
			t.Errorf("Pattern %d should have positive support, got %d", i, pattern.Support)
		}
		if pattern.Confidence <= 0 {
			t.Errorf("Pattern %d should have positive confidence, got %f", i, pattern.Confidence)
		}
		if pattern.Frequency <= 0 {
			t.Errorf("Pattern %d should have positive frequency, got %d", i, pattern.Frequency)
		}
		if pattern.LastSeen.IsZero() {
			t.Errorf("Pattern %d should have LastSeen timestamp", i)
		}
	}
	
	// Test that patterns are sorted by frequency
	for i := 1; i < len(patterns); i++ {
		if patterns[i-1].Frequency < patterns[i].Frequency {
			t.Errorf("Patterns should be sorted by frequency (descending)")
		}
	}
	
	// Test specific pattern expectations
	foundMainUtils := false
	foundHandlerMiddleware := false
	
	for _, pattern := range patterns {
		if containsAll(pattern.Items, []string{"src/main.go", "src/utils.go"}) {
			foundMainUtils = true
			if pattern.Frequency != 3 {
				t.Errorf("main.go + utils.go pattern should have frequency 3, got %d", pattern.Frequency)
			}
		}
		if containsAll(pattern.Items, []string{"src/handler.go", "src/middleware.go"}) {
			foundHandlerMiddleware = true
			if pattern.Frequency != 2 {
				t.Errorf("handler.go + middleware.go pattern should have frequency 2, got %d", pattern.Frequency)
			}
		}
	}
	
	if !foundMainUtils {
		t.Error("Should have found main.go + utils.go pattern")
	}
	if !foundHandlerMiddleware {
		t.Error("Should have found handler.go + middleware.go pattern")
	}
}

func TestPatternDetectionWithIgnorePatterns(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "codecontext_ignore_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a .codecontextignore file
	ignoreFile := filepath.Join(tempDir, ".codecontextignore")
	ignoreContent := `# Test ignore file
node_modules/
*.log
dist/
build/`
	
	err = os.WriteFile(ignoreFile, []byte(ignoreContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create ignore file: %v", err)
	}
	
	// Create a mock git analyzer
	analyzer := &GitAnalyzer{repoPath: tempDir}
	
	// Create pattern detector (should load ignore patterns)
	pd := NewPatternDetector(analyzer)
	pd.SetThresholds(0.3, 0.3)
	
	// Test shouldIncludeFile with ignore patterns
	testCases := []struct {
		file     string
		expected bool
	}{
		{"src/main.go", true},
		{"node_modules/package/index.js", false},
		{"app.log", false},
		{"dist/bundle.js", false},
		{"build/output.js", false},
		{"package.json", true},
		{"README.md", false}, // Should be false because it doesn't match source extensions
	}
	
	for _, tc := range testCases {
		result := pd.shouldIncludeFile(tc.file)
		if result != tc.expected {
			t.Errorf("shouldIncludeFile(%q) = %v, expected %v", tc.file, result, tc.expected)
		}
	}
}

func TestPatternDetectionPerformance(t *testing.T) {
	// Create a large number of commits to test performance
	numCommits := 1000
	commits := make([]CommitInfo, numCommits)
	
	baseTime := time.Now().Add(-time.Duration(numCommits) * time.Hour)
	
	for i := 0; i < numCommits; i++ {
		commits[i] = CommitInfo{
			Hash:      fmt.Sprintf("commit%d", i),
			Files:     []string{fmt.Sprintf("file%d.go", i%10), fmt.Sprintf("file%d.go", (i+1)%10)},
			Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
			Author:    fmt.Sprintf("dev%d", i%5),
			Message:   fmt.Sprintf("Commit %d", i),
		}
	}
	
	detector := NewSimplePatternsDetector(0.05, 0.3)
	
	start := time.Now()
	patterns, err := detector.MineSimplePatterns(commits)
	elapsed := time.Since(start)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if len(patterns) == 0 {
		t.Error("Expected to find some patterns")
	}
	
	// Performance should be reasonable (less than 5 seconds for 1000 commits)
	if elapsed > 5*time.Second {
		t.Errorf("Pattern detection took too long: %v", elapsed)
	}
	
	t.Logf("Processed %d commits in %v, found %d patterns", numCommits, elapsed, len(patterns))
}

func TestPatternDetectionEdgeCases(t *testing.T) {
	detector := NewSimplePatternsDetector(0.05, 0.3)
	
	// Test with nil commits
	patterns, err := detector.MineSimplePatterns(nil)
	if err != nil {
		t.Errorf("Expected no error with nil commits, got %v", err)
	}
	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns with nil commits, got %d", len(patterns))
	}
	
	// Test with commits containing duplicate files
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"file1.go", "file2.go", "file1.go"}, // Duplicate file1.go
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "duplicate files",
		},
		{
			Hash:      "def456",
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: time.Now().Add(time.Hour),
			Author:    "test",
			Message:   "normal commit",
		},
	}
	
	patterns, err = detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error with duplicate files, got %v", err)
	}
	
	// Should handle duplicates gracefully
	if len(patterns) != 1 {
		t.Errorf("Expected 1 pattern with duplicate files, got %d", len(patterns))
	}
	
	// Test with empty file lists
	commits = []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{}, // Empty files
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "empty commit",
		},
	}
	
	patterns, err = detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error with empty files, got %v", err)
	}
	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns with empty files, got %d", len(patterns))
	}
}

func TestPatternDetectionWithFilters(t *testing.T) {
	detector := NewSimplePatternsDetector(0.3, 0.3)
	
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"main.go", "utils.go", "main.js", "utils.js"},
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "mixed languages",
		},
		{
			Hash:      "def456",
			Files:     []string{"main.go", "utils.go", "config.json"},
			Timestamp: time.Now().Add(time.Hour),
			Author:    "test",
			Message:   "go and config",
		},
	}
	
	// Test with Go files only filter
	detector.SetFileFilter(func(file string) bool {
		return filepath.Ext(file) == ".go"
	})
	
	patterns, err := detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if len(patterns) != 1 {
		t.Errorf("Expected 1 pattern with Go filter, got %d", len(patterns))
	}
	
	if len(patterns) > 0 {
		pattern := patterns[0]
		for _, file := range pattern.Items {
			if filepath.Ext(file) != ".go" {
				t.Errorf("Expected only .go files, found %s", file)
			}
		}
	}
	
	// Test with JavaScript files only filter
	detector.SetFileFilter(func(file string) bool {
		return filepath.Ext(file) == ".js"
	})
	
	patterns, err = detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Should find no patterns since JS files don't repeat in both commits
	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns with JS filter, got %d", len(patterns))
	}
}

func TestPatternDetectionConfidenceCalculation(t *testing.T) {
	detector := NewSimplePatternsDetector(0.1, 0.1)
	
	commits := []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"fileA.go", "fileB.go"},
			Timestamp: time.Now(),
			Author:    "test",
			Message:   "both files",
		},
		{
			Hash:      "def456",
			Files:     []string{"fileA.go", "fileB.go"},
			Timestamp: time.Now().Add(time.Hour),
			Author:    "test",
			Message:   "both files again",
		},
		{
			Hash:      "ghi789",
			Files:     []string{"fileA.go", "fileC.go"},
			Timestamp: time.Now().Add(2 * time.Hour),
			Author:    "test",
			Message:   "fileA with different file",
		},
	}
	
	patterns, err := detector.MineSimplePatterns(commits)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Find the fileA + fileB pattern
	var targetPattern *FrequentItemset
	for i := range patterns {
		if containsAll(patterns[i].Items, []string{"fileA.go", "fileB.go"}) {
			targetPattern = &patterns[i]
			break
		}
	}
	
	if targetPattern == nil {
		t.Fatal("Expected to find fileA + fileB pattern")
	}
	
	// Confidence should be 2/3 = 0.667 (2 commits with both files, 3 commits with fileA)
	expectedConfidence := 2.0 / 3.0
	if targetPattern.Confidence != expectedConfidence {
		t.Errorf("Expected confidence %f, got %f", expectedConfidence, targetPattern.Confidence)
	}
}

// Helper function to check if slice contains all elements
func containsAll(slice []string, elements []string) bool {
	for _, element := range elements {
		found := false
		for _, item := range slice {
			if item == element {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}