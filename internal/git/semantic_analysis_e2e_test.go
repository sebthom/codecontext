package git

import (
	"context"
	"testing"
	"time"
)

// MockSuccessGitAnalyzer implements GitAnalyzerInterface with successful responses
type MockSuccessGitAnalyzer struct {
	repoPath string
	commits  []CommitInfo
}

func (m *MockSuccessGitAnalyzer) IsGitRepository() bool {
	return true
}

func (m *MockSuccessGitAnalyzer) GetFileChangeHistory(days int) ([]FileChange, error) {
	return []FileChange{
		{
			FilePath:   "src/main.go",
			ChangeType: "M",
			CommitHash: "abc123",
			Timestamp:  time.Now().Add(-1 * time.Hour),
			Author:     "developer1",
			Message:    "Update main logic",
		},
		{
			FilePath:   "src/utils.go",
			ChangeType: "M",
			CommitHash: "def456",
			Timestamp:  time.Now().Add(-2 * time.Hour),
			Author:     "developer2",
			Message:    "Fix utilities",
		},
	}, nil
}

func (m *MockSuccessGitAnalyzer) GetCommitHistory(days int) ([]CommitInfo, error) {
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	var filtered []CommitInfo
	
	for _, commit := range m.commits {
		if commit.Timestamp.After(cutoff) {
			filtered = append(filtered, commit)
		}
	}
	
	return filtered, nil
}

func (m *MockSuccessGitAnalyzer) GetFileCoOccurrences(days int) (map[string][]string, error) {
	return map[string][]string{
		"src/main.go":  {"src/utils.go", "src/config.go"},
		"src/utils.go": {"src/main.go", "src/types.go"},
	}, nil
}

func (m *MockSuccessGitAnalyzer) GetChangeFrequency(days int) (map[string]int, error) {
	return map[string]int{
		"src/main.go":   5,
		"src/utils.go":  3,
		"src/config.go": 2,
	}, nil
}

func (m *MockSuccessGitAnalyzer) GetLastModified() (map[string]time.Time, error) {
	now := time.Now()
	return map[string]time.Time{
		"src/main.go":   now.Add(-1 * time.Hour),
		"src/utils.go":  now.Add(-2 * time.Hour),
		"src/config.go": now.Add(-3 * time.Hour),
	}, nil
}

func (m *MockSuccessGitAnalyzer) GetBranchInfo() (string, error) {
	return "main", nil
}

func (m *MockSuccessGitAnalyzer) GetRemoteInfo() (string, error) {
	return "https://github.com/test/repo.git", nil
}

func (m *MockSuccessGitAnalyzer) ExecuteGitCommand(ctx context.Context, args ...string) ([]byte, error) {
	return []byte("mock git command output"), nil
}

func (m *MockSuccessGitAnalyzer) GetRepoPath() string {
	return m.repoPath
}

// TestSemanticAnalysisEndToEnd tests complete semantic analysis workflow
func TestSemanticAnalysisEndToEnd(t *testing.T) {
	// Use current directory which is a git repository
	config := DefaultSemanticConfig()
	analyzer, err := NewSemanticAnalyzer(".", config)
	if err != nil {
		t.Fatalf("Failed to create semantic analyzer: %v", err)
	}

	// Create mock with test data
	mockAnalyzer := &MockSuccessGitAnalyzer{
		repoPath: ".",
		commits: []CommitInfo{
			{
				Hash:      "abc123",
				Files:     []string{"src/main.go", "src/utils.go"},
				Timestamp: time.Now().Add(-1 * time.Hour),
				Author:    "developer1",
				Message:   "Update main logic",
			},
			{
				Hash:      "def456",
				Files:     []string{"src/utils.go", "src/config.go"},
				Timestamp: time.Now().Add(-2 * time.Hour),
				Author:    "developer2",
				Message:   "Fix configuration",
			},
		},
	}

	analyzer.gitAnalyzer = mockAnalyzer
	analyzer.patternDetector = NewPatternDetector(mockAnalyzer)

	// Run semantic analysis
	result, err := analyzer.AnalyzeRepository()
	if err != nil {
		t.Fatalf("Semantic analysis failed: %v", err)
	}

	// Verify results
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Basic checks - patterns may be empty with mock data
	t.Logf("Found %d file relationships, %d change patterns, %d total files", 
		len(result.FileRelationships), len(result.ChangePatterns), result.AnalysisSummary.TotalFiles)
}

// TestSemanticAnalysisConfiguration tests different configurations
func TestSemanticAnalysisConfiguration(t *testing.T) {
	// Test with conservative configuration
	conservativeConfig := &SemanticConfig{
		AnalysisPeriodDays:   7,
		MinChangeCorrelation: 0.8,
		MinPatternSupport:    0.3,
		MinPatternConfidence: 0.8,
	}

	analyzer, err := NewSemanticAnalyzer(".", conservativeConfig)
	if err != nil {
		t.Fatalf("Failed to create semantic analyzer: %v", err)
	}

	mockAnalyzer := &MockSuccessGitAnalyzer{
		repoPath: ".",
		commits: []CommitInfo{
			{
				Hash:      "abc123",
				Files:     []string{"src/main.go"},
				Timestamp: time.Now().Add(-1 * time.Hour),
				Author:    "developer1",
				Message:   "Update main",
			},
		},
	}

	analyzer.gitAnalyzer = mockAnalyzer
	analyzer.patternDetector = NewPatternDetector(mockAnalyzer)

	result, err := analyzer.AnalyzeRepository()
	if err != nil {
		t.Fatalf("Conservative analysis failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

// TestSemanticAnalysisContextRecommendations tests context recommendations
func TestSemanticAnalysisContextRecommendations(t *testing.T) {
	config := DefaultSemanticConfig()
	analyzer, err := NewSemanticAnalyzer(".", config)
	if err != nil {
		t.Fatalf("Failed to create semantic analyzer: %v", err)
	}

	mockAnalyzer := &MockSuccessGitAnalyzer{
		repoPath: ".",
		commits: []CommitInfo{
			{
				Hash:      "abc123",
				Files:     []string{"src/main.go", "src/utils.go"},
				Timestamp: time.Now().Add(-1 * time.Hour),
				Author:    "developer1",
				Message:   "Update main logic",
			},
		},
	}

	analyzer.gitAnalyzer = mockAnalyzer
	analyzer.patternDetector = NewPatternDetector(mockAnalyzer)

	// Test getting context recommendations for specific file
	recommendations, err := analyzer.GetContextRecommendationsForFile("src/main.go")
	if err != nil {
		t.Fatalf("Failed to get context recommendations: %v", err)
	}

	if len(recommendations) == 0 {
		t.Error("Expected context recommendations")
	}
}

// TestSemanticAnalysisPerformance tests performance characteristics
func TestSemanticAnalysisPerformance(t *testing.T) {
	config := DefaultSemanticConfig()
	analyzer, err := NewSemanticAnalyzer(".", config)
	if err != nil {
		t.Fatalf("Failed to create semantic analyzer: %v", err)
	}

	// Create mock with larger dataset
	commits := make([]CommitInfo, 100)
	for i := 0; i < 100; i++ {
		commits[i] = CommitInfo{
			Hash:      "commit" + string(rune('0'+i%10)),
			Files:     []string{"file" + string(rune('0'+i%10)) + ".go"},
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
			Author:    "developer" + string(rune('0'+i%5)),
			Message:   "Commit " + string(rune('0'+i%10)),
		}
	}

	mockAnalyzer := &MockSuccessGitAnalyzer{
		repoPath: ".",
		commits:  commits,
	}

	analyzer.gitAnalyzer = mockAnalyzer
	analyzer.patternDetector = NewPatternDetector(mockAnalyzer)

	// Measure performance
	start := time.Now()
	result, err := analyzer.AnalyzeRepository()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Performance test failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Performance should be reasonable (adjust threshold as needed)
	if duration > 10*time.Second {
		t.Errorf("Analysis took too long: %v", duration)
	}

	t.Logf("Analysis completed in %v", duration)
}