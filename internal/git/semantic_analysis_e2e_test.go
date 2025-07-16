package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSemanticAnalysisEndToEnd(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "codecontext_e2e_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a mock git repository structure
	createMockRepository(t, tempDir)
	
	// Create semantic analyzer
	config := DefaultSemanticConfig()
	config.AnalysisPeriodDays = 30
	config.MinPatternSupport = 0.1
	config.MinPatternConfidence = 0.2
	
	analyzer, err := NewSemanticAnalyzer(tempDir, config)
	if err != nil {
		t.Fatalf("Failed to create semantic analyzer: %v", err)
	}
	
	// Mock the git analyzer's methods for testing
	mockGitAnalyzer := &MockGitAnalyzer{
		repoPath: tempDir,
		commits:  createMockCommits(),
	}
	analyzer.gitAnalyzer = mockGitAnalyzer
	analyzer.patternDetector = NewPatternDetector(mockGitAnalyzer)
	analyzer.patternDetector.SetThresholds(config.MinPatternSupport, config.MinPatternConfidence)
	
	// Run semantic analysis
	result, err := analyzer.AnalyzeRepository()
	if err != nil {
		t.Fatalf("Semantic analysis failed: %v", err)
	}
	
	// Verify results structure
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	
	// Test neighborhoods
	if len(result.Neighborhoods) == 0 {
		t.Error("Expected to find some neighborhoods")
	}
	
	for i, neighborhood := range result.Neighborhoods {
		if neighborhood.Name == "" {
			t.Errorf("Neighborhood %d should have a name", i)
		}
		if len(neighborhood.Files) == 0 {
			t.Errorf("Neighborhood %d should have files", i)
		}
		if neighborhood.ChangeFrequency < 0 {
			t.Errorf("Neighborhood %d should have non-negative change frequency", i)
		}
		if neighborhood.Confidence < 0 || neighborhood.Confidence > 1 {
			t.Errorf("Neighborhood %d should have confidence between 0 and 1", i)
		}
		if neighborhood.Metadata == nil {
			t.Errorf("Neighborhood %d should have metadata", i)
		}
	}
	
	// Test change patterns
	if len(result.ChangePatterns) == 0 {
		t.Error("Expected to find some change patterns")
	}
	
	for i, pattern := range result.ChangePatterns {
		if pattern.Name == "" {
			t.Errorf("Pattern %d should have a name", i)
		}
		if len(pattern.Files) < 2 {
			t.Errorf("Pattern %d should have at least 2 files", i)
		}
		if pattern.Frequency <= 0 {
			t.Errorf("Pattern %d should have positive frequency", i)
		}
		if pattern.Confidence < 0 || pattern.Confidence > 1 {
			t.Errorf("Pattern %d should have confidence between 0 and 1", i)
		}
	}
	
	// Test context recommendations
	if len(result.ContextRecommendations) == 0 {
		t.Error("Expected to find some context recommendations")
	}
	
	for i, rec := range result.ContextRecommendations {
		if rec.ForFile == "" {
			t.Errorf("Recommendation %d should have ForFile", i)
		}
		if len(rec.IncludeFiles) == 0 {
			t.Errorf("Recommendation %d should have IncludeFiles", i)
		}
		if rec.Reason == "" {
			t.Errorf("Recommendation %d should have Reason", i)
		}
		if rec.Confidence < 0 || rec.Confidence > 1 {
			t.Errorf("Recommendation %d should have confidence between 0 and 1", i)
		}
		if rec.Priority == "" {
			t.Errorf("Recommendation %d should have Priority", i)
		}
	}
	
	// Test analysis summary
	summary := result.AnalysisSummary
	if summary.TotalFiles <= 0 {
		t.Error("Summary should have positive total files")
	}
	if summary.NeighborhoodsFound != len(result.Neighborhoods) {
		t.Errorf("Summary neighborhoods count mismatch: %d vs %d", summary.NeighborhoodsFound, len(result.Neighborhoods))
	}
	if summary.PatternsFound != len(result.ChangePatterns) {
		t.Errorf("Summary patterns count mismatch: %d vs %d", summary.PatternsFound, len(result.ChangePatterns))
	}
	if summary.AnalysisDate.IsZero() {
		t.Error("Summary should have analysis date")
	}
	if summary.PerformanceMetrics.AnalysisTime <= 0 {
		t.Error("Summary should have positive analysis time")
	}
}

func TestSemanticAnalysisWithDifferentConfigurations(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "codecontext_config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	createMockRepository(t, tempDir)
	
	// Test with conservative configuration
	conservativeConfig := &SemanticConfig{
		AnalysisPeriodDays:   30,
		MinChangeCorrelation: 0.8,
		MinPatternSupport:    0.2,
		MinPatternConfidence: 0.8,
		MaxNeighborhoodSize:  5,
		IncludeTestFiles:     false,
		IncludeDocFiles:      false,
		IncludeConfigFiles:   false,
	}
	
	analyzer, err := NewSemanticAnalyzer(tempDir, conservativeConfig)
	if err != nil {
		t.Fatalf("Failed to create semantic analyzer: %v", err)
	}
	
	// Mock the git analyzer
	mockGitAnalyzer := &MockGitAnalyzer{
		repoPath: tempDir,
		commits:  createMockCommits(),
	}
	analyzer.gitAnalyzer = mockGitAnalyzer
	analyzer.patternDetector = NewPatternDetector(mockGitAnalyzer)
	analyzer.patternDetector.SetThresholds(conservativeConfig.MinPatternSupport, conservativeConfig.MinPatternConfidence)
	
	conservativeResult, err := analyzer.AnalyzeRepository()
	if err != nil {
		t.Fatalf("Conservative analysis failed: %v", err)
	}
	
	// Test with liberal configuration
	liberalConfig := &SemanticConfig{
		AnalysisPeriodDays:   30,
		MinChangeCorrelation: 0.2,
		MinPatternSupport:    0.05,
		MinPatternConfidence: 0.2,
		MaxNeighborhoodSize:  20,
		IncludeTestFiles:     true,
		IncludeDocFiles:      true,
		IncludeConfigFiles:   true,
	}
	
	analyzer, err = NewSemanticAnalyzer(tempDir, liberalConfig)
	if err != nil {
		t.Fatalf("Failed to create semantic analyzer: %v", err)
	}
	
	// Mock the git analyzer
	analyzer.gitAnalyzer = mockGitAnalyzer
	analyzer.patternDetector = NewPatternDetector(mockGitAnalyzer)
	analyzer.patternDetector.SetThresholds(liberalConfig.MinPatternSupport, liberalConfig.MinPatternConfidence)
	
	liberalResult, err := analyzer.AnalyzeRepository()
	if err != nil {
		t.Fatalf("Liberal analysis failed: %v", err)
	}
	
	// Liberal configuration should find more patterns
	if len(liberalResult.ChangePatterns) < len(conservativeResult.ChangePatterns) {
		t.Errorf("Liberal config should find more patterns: %d vs %d", len(liberalResult.ChangePatterns), len(conservativeResult.ChangePatterns))
	}
	
	// Liberal configuration should find more neighborhoods
	if len(liberalResult.Neighborhoods) < len(conservativeResult.Neighborhoods) {
		t.Errorf("Liberal config should find more neighborhoods: %d vs %d", len(liberalResult.Neighborhoods), len(conservativeResult.Neighborhoods))
	}
}

func TestSemanticAnalysisFileFiltering(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "codecontext_filter_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	createMockRepository(t, tempDir)
	
	// Create configuration that excludes test files
	config := DefaultSemanticConfig()
	config.IncludeTestFiles = false
	config.IncludeDocFiles = false
	config.IncludeConfigFiles = false
	
	analyzer, err := NewSemanticAnalyzer(tempDir, config)
	if err != nil {
		t.Fatalf("Failed to create semantic analyzer: %v", err)
	}
	
	// Test shouldIncludeFile method
	testCases := []struct {
		file     string
		expected bool
	}{
		{"main.go", true},
		{"main_test.go", false},
		{"README.md", false},
		{"config.json", false},
		{"src/handler.go", true},
		{"src/handler_test.go", false},
		{".hidden", false},
		{"node_modules/package.js", false},
	}
	
	for _, tc := range testCases {
		result := analyzer.shouldIncludeFile(tc.file)
		if result != tc.expected {
			t.Errorf("shouldIncludeFile(%q) = %v, expected %v", tc.file, result, tc.expected)
		}
	}
	
	// Test with include test files enabled
	config.IncludeTestFiles = true
	config.IncludeDocFiles = true
	config.IncludeConfigFiles = true
	
	analyzer, err = NewSemanticAnalyzer(tempDir, config)
	if err != nil {
		t.Fatalf("Failed to create semantic analyzer: %v", err)
	}
	
	// Now test files should be included
	if !analyzer.shouldIncludeFile("main_test.go") {
		t.Error("Test files should be included when IncludeTestFiles is true")
	}
	if !analyzer.shouldIncludeFile("README.md") {
		t.Error("Doc files should be included when IncludeDocFiles is true")
	}
	if !analyzer.shouldIncludeFile("config.json") {
		t.Error("Config files should be included when IncludeConfigFiles is true")
	}
}

func TestSemanticAnalysisContextRecommendations(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "codecontext_context_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	createMockRepository(t, tempDir)
	
	config := DefaultSemanticConfig()
	analyzer, err := NewSemanticAnalyzer(tempDir, config)
	if err != nil {
		t.Fatalf("Failed to create semantic analyzer: %v", err)
	}
	
	// Mock the git analyzer
	mockGitAnalyzer := &MockGitAnalyzer{
		repoPath: tempDir,
		commits:  createMockCommits(),
	}
	analyzer.gitAnalyzer = mockGitAnalyzer
	analyzer.patternDetector = NewPatternDetector(mockGitAnalyzer)
	analyzer.patternDetector.SetThresholds(config.MinPatternSupport, config.MinPatternConfidence)
	
	// Test getting context recommendations for specific file
	recommendations, err := analyzer.GetContextRecommendationsForFile("src/main.go")
	if err != nil {
		t.Fatalf("Failed to get context recommendations: %v", err)
	}
	
	// Should find recommendations for main.go
	if len(recommendations) == 0 {
		t.Error("Expected to find context recommendations for main.go")
	}
	
	for i, rec := range recommendations {
		if rec.ForFile != "src/main.go" {
			t.Errorf("Recommendation %d should be for main.go, got %s", i, rec.ForFile)
		}
		if len(rec.IncludeFiles) == 0 {
			t.Errorf("Recommendation %d should have include files", i)
		}
		
		// Should not include the target file in recommendations
		for _, includeFile := range rec.IncludeFiles {
			if includeFile == "src/main.go" {
				t.Errorf("Recommendation %d should not include target file in include files", i)
			}
		}
	}
	
	// Test with non-existent file
	recommendations, err = analyzer.GetContextRecommendationsForFile("nonexistent.go")
	if err != nil {
		t.Fatalf("Failed to get context recommendations for nonexistent file: %v", err)
	}
	
	// Should return empty recommendations
	if len(recommendations) != 0 {
		t.Errorf("Expected no recommendations for nonexistent file, got %d", len(recommendations))
	}
}

func TestSemanticAnalysisPerformance(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "codecontext_perf_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	createMockRepository(t, tempDir)
	
	config := DefaultSemanticConfig()
	analyzer, err := NewSemanticAnalyzer(tempDir, config)
	if err != nil {
		t.Fatalf("Failed to create semantic analyzer: %v", err)
	}
	
	// Create large number of commits for performance testing
	largeCommits := createLargeCommitSet(1000)
	
	mockGitAnalyzer := &MockGitAnalyzer{
		repoPath: tempDir,
		commits:  largeCommits,
	}
	analyzer.gitAnalyzer = mockGitAnalyzer
	analyzer.patternDetector = NewPatternDetector(mockGitAnalyzer)
	analyzer.patternDetector.SetThresholds(config.MinPatternSupport, config.MinPatternConfidence)
	
	// Measure performance
	start := time.Now()
	result, err := analyzer.AnalyzeRepository()
	elapsed := time.Since(start)
	
	if err != nil {
		t.Fatalf("Performance test failed: %v", err)
	}
	
	// Should complete in reasonable time (less than 30 seconds for 1000 commits)
	if elapsed > 30*time.Second {
		t.Errorf("Analysis took too long: %v", elapsed)
	}
	
	// Should find some results
	if len(result.ChangePatterns) == 0 {
		t.Error("Expected to find some patterns with large commit set")
	}
	
	// Performance metrics should be recorded
	if result.AnalysisSummary.PerformanceMetrics.AnalysisTime != elapsed {
		t.Errorf("Performance metrics mismatch: recorded %v, actual %v", result.AnalysisSummary.PerformanceMetrics.AnalysisTime, elapsed)
	}
	
	t.Logf("Analyzed %d commits in %v", len(largeCommits), elapsed)
}

// Helper functions for testing

func createMockRepository(t *testing.T, tempDir string) {
	dirs := []string{
		filepath.Join(tempDir, "src"),
		filepath.Join(tempDir, "test"),
		filepath.Join(tempDir, "docs"),
	}
	
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}
	
	files := []string{
		filepath.Join(tempDir, "src", "main.go"),
		filepath.Join(tempDir, "src", "utils.go"),
		filepath.Join(tempDir, "src", "handler.go"),
		filepath.Join(tempDir, "test", "main_test.go"),
		filepath.Join(tempDir, "README.md"),
		filepath.Join(tempDir, "package.json"),
	}
	
	for _, file := range files {
		err := os.WriteFile(file, []byte("// Mock file content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}
}

func createMockCommits() []CommitInfo {
	now := time.Now()
	return []CommitInfo{
		{
			Hash:      "abc123",
			Files:     []string{"src/main.go", "src/utils.go"},
			Timestamp: now.Add(-24 * time.Hour),
			Author:    "dev1",
			Message:   "Add main functionality",
		},
		{
			Hash:      "def456",
			Files:     []string{"src/main.go", "src/utils.go"},
			Timestamp: now.Add(-20 * time.Hour),
			Author:    "dev2",
			Message:   "Update main and utils",
		},
		{
			Hash:      "ghi789",
			Files:     []string{"src/main.go", "src/utils.go"},
			Timestamp: now.Add(-16 * time.Hour),
			Author:    "dev1",
			Message:   "Fix main and utils",
		},
		{
			Hash:      "jkl012",
			Files:     []string{"src/handler.go", "src/middleware.go"},
			Timestamp: now.Add(-12 * time.Hour),
			Author:    "dev3",
			Message:   "Add handler and middleware",
		},
		{
			Hash:      "mno345",
			Files:     []string{"src/handler.go", "src/middleware.go"},
			Timestamp: now.Add(-8 * time.Hour),
			Author:    "dev2",
			Message:   "Update handler and middleware",
		},
		{
			Hash:      "pqr678",
			Files:     []string{"src/main.go", "README.md"},
			Timestamp: now.Add(-4 * time.Hour),
			Author:    "dev1",
			Message:   "Update main and docs",
		},
	}
}

func createLargeCommitSet(numCommits int) []CommitInfo {
	commits := make([]CommitInfo, numCommits)
	now := time.Now()
	
	for i := 0; i < numCommits; i++ {
		commits[i] = CommitInfo{
			Hash:      fmt.Sprintf("commit%d", i),
			Files:     []string{fmt.Sprintf("file%d.go", i%20), fmt.Sprintf("file%d.go", (i+1)%20)},
			Timestamp: now.Add(-time.Duration(i) * time.Hour),
			Author:    fmt.Sprintf("dev%d", i%10),
			Message:   fmt.Sprintf("Commit %d", i),
		}
	}
	
	return commits
}

// MockGitAnalyzer for testing
type MockGitAnalyzer struct {
	repoPath string
	commits  []CommitInfo
}

func (m *MockGitAnalyzer) GetCommitHistory(days int) ([]CommitInfo, error) {
	// Return commits within the specified days
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	var filtered []CommitInfo
	
	for _, commit := range m.commits {
		if commit.Timestamp.After(cutoff) {
			filtered = append(filtered, commit)
		}
	}
	
	return filtered, nil
}

func (m *MockGitAnalyzer) GetFileCoOccurrences(days int) (map[string][]string, error) {
	// Simple mock implementation
	return map[string][]string{
		"src/main.go":    {"src/utils.go"},
		"src/utils.go":   {"src/main.go"},
		"src/handler.go": {"src/middleware.go"},
	}, nil
}

func (m *MockGitAnalyzer) GetChangeFrequency(days int) (map[string]int, error) {
	// Simple mock implementation
	return map[string]int{
		"src/main.go":       4,
		"src/utils.go":      3,
		"src/handler.go":    2,
		"src/middleware.go": 2,
		"README.md":         1,
	}, nil
}

func (m *MockGitAnalyzer) GetLastModified() (map[string]time.Time, error) {
	// Simple mock implementation
	now := time.Now()
	return map[string]time.Time{
		"src/main.go":       now.Add(-4 * time.Hour),
		"src/utils.go":      now.Add(-16 * time.Hour),
		"src/handler.go":    now.Add(-8 * time.Hour),
		"src/middleware.go": now.Add(-8 * time.Hour),
		"README.md":         now.Add(-4 * time.Hour),
	}, nil
}

func (m *MockGitAnalyzer) GetBranchInfo() (string, error) {
	return "main", nil
}

func (m *MockGitAnalyzer) GetRemoteInfo() (string, error) {
	return "https://github.com/test/repo.git", nil
}