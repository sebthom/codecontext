package git

import (
	"context"
	"testing"
	"time"
)

// MockBenchmarkGitAnalyzer implements GitAnalyzerInterface for benchmarking
type MockBenchmarkGitAnalyzer struct {
	repoPath string
	commits  []CommitInfo
}

func (m *MockBenchmarkGitAnalyzer) IsGitRepository() bool {
	return true
}

func (m *MockBenchmarkGitAnalyzer) GetFileChangeHistory(days int) ([]FileChange, error) {
	changes := make([]FileChange, 0, 1000)
	for i := 0; i < 1000; i++ {
		changes = append(changes, FileChange{
			FilePath:   "src/file" + string(rune('0'+i%10)) + ".go",
			ChangeType: "M",
			CommitHash: "commit" + string(rune('0'+i%10)),
			Timestamp:  time.Now().Add(-time.Duration(i) * time.Hour),
			Author:     "developer" + string(rune('0'+i%5)),
			Message:    "Update file",
		})
	}
	return changes, nil
}

func (m *MockBenchmarkGitAnalyzer) GetCommitHistory(days int) ([]CommitInfo, error) {
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	var filtered []CommitInfo
	
	for _, commit := range m.commits {
		if commit.Timestamp.After(cutoff) {
			filtered = append(filtered, commit)
		}
	}
	
	return filtered, nil
}

func (m *MockBenchmarkGitAnalyzer) GetFileCoOccurrences(days int) (map[string][]string, error) {
	coOccurrences := make(map[string][]string)
	for i := 0; i < 100; i++ {
		file := "src/file" + string(rune('0'+i%10)) + ".go"
		coOccurrences[file] = []string{
			"src/file" + string(rune('0'+(i+1)%10)) + ".go",
			"src/file" + string(rune('0'+(i+2)%10)) + ".go",
		}
	}
	return coOccurrences, nil
}

func (m *MockBenchmarkGitAnalyzer) GetChangeFrequency(days int) (map[string]int, error) {
	frequency := make(map[string]int)
	for i := 0; i < 100; i++ {
		file := "src/file" + string(rune('0'+i%10)) + ".go"
		frequency[file] = i + 1
	}
	return frequency, nil
}

func (m *MockBenchmarkGitAnalyzer) GetLastModified() (map[string]time.Time, error) {
	lastModified := make(map[string]time.Time)
	now := time.Now()
	for i := 0; i < 100; i++ {
		file := "src/file" + string(rune('0'+i%10)) + ".go"
		lastModified[file] = now.Add(-time.Duration(i) * time.Hour)
	}
	return lastModified, nil
}

func (m *MockBenchmarkGitAnalyzer) GetBranchInfo() (string, error) {
	return "main", nil
}

func (m *MockBenchmarkGitAnalyzer) GetRemoteInfo() (string, error) {
	return "https://github.com/test/repo.git", nil
}

func (m *MockBenchmarkGitAnalyzer) ExecuteGitCommand(ctx context.Context, args ...string) ([]byte, error) {
	return []byte("benchmark git command output"), nil
}

func (m *MockBenchmarkGitAnalyzer) GetRepoPath() string {
	return m.repoPath
}

// BenchmarkSemanticAnalysis benchmarks the semantic analysis process
func BenchmarkSemanticAnalysis(b *testing.B) {
	// Create mock analyzer with realistic data
	commits := make([]CommitInfo, 1000)
	for i := 0; i < 1000; i++ {
		commits[i] = CommitInfo{
			Hash:      "commit" + string(rune('0'+i%10)),
			Files:     []string{"src/file" + string(rune('0'+i%10)) + ".go"},
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
			Author:    "developer" + string(rune('0'+i%5)),
			Message:   "Commit " + string(rune('0'+i%10)),
		}
	}

	mockAnalyzer := &MockBenchmarkGitAnalyzer{
		repoPath: ".",
		commits:  commits,
	}

	config := DefaultSemanticConfig()
	analyzer, err := NewSemanticAnalyzer(".", config)
	if err != nil {
		b.Fatalf("Failed to create semantic analyzer: %v", err)
	}

	analyzer.gitAnalyzer = mockAnalyzer
	analyzer.patternDetector = NewPatternDetector(mockAnalyzer)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzeRepository()
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

// BenchmarkPatternDetection benchmarks pattern detection specifically
func BenchmarkPatternDetection(b *testing.B) {
	commits := make([]CommitInfo, 500)
	for i := 0; i < 500; i++ {
		commits[i] = CommitInfo{
			Hash:      "commit" + string(rune('0'+i%10)),
			Files:     []string{"src/file" + string(rune('0'+i%10)) + ".go"},
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
			Author:    "developer" + string(rune('0'+i%5)),
			Message:   "Commit " + string(rune('0'+i%10)),
		}
	}

	mockAnalyzer := &MockBenchmarkGitAnalyzer{
		repoPath: ".",
		commits:  commits,
	}

	detector := NewPatternDetector(mockAnalyzer)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := detector.DetectChangePatterns(30)
		if err != nil {
			b.Fatalf("Pattern detection benchmark failed: %v", err)
		}
	}
}

// BenchmarkFileRelationships benchmarks file relationship detection
func BenchmarkFileRelationships(b *testing.B) {
	commits := make([]CommitInfo, 300)
	for i := 0; i < 300; i++ {
		commits[i] = CommitInfo{
			Hash:      "commit" + string(rune('0'+i%10)),
			Files:     []string{"src/file" + string(rune('0'+i%10)) + ".go"},
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
			Author:    "developer" + string(rune('0'+i%5)),
			Message:   "Commit " + string(rune('0'+i%10)),
		}
	}

	mockAnalyzer := &MockBenchmarkGitAnalyzer{
		repoPath: ".",
		commits:  commits,
	}

	detector := NewPatternDetector(mockAnalyzer)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := detector.DetectFileRelationships(30)
		if err != nil {
			b.Fatalf("File relationships benchmark failed: %v", err)
		}
	}
}

// BenchmarkContextRecommendations benchmarks context recommendations
func BenchmarkContextRecommendations(b *testing.B) {
	commits := make([]CommitInfo, 200)
	for i := 0; i < 200; i++ {
		commits[i] = CommitInfo{
			Hash:      "commit" + string(rune('0'+i%10)),
			Files:     []string{"src/main.go", "src/utils.go"},
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
			Author:    "developer" + string(rune('0'+i%5)),
			Message:   "Update main logic",
		}
	}

	mockAnalyzer := &MockBenchmarkGitAnalyzer{
		repoPath: ".",
		commits:  commits,
	}

	config := DefaultSemanticConfig()
	analyzer, err := NewSemanticAnalyzer(".", config)
	if err != nil {
		b.Fatalf("Failed to create semantic analyzer: %v", err)
	}

	analyzer.gitAnalyzer = mockAnalyzer
	analyzer.patternDetector = NewPatternDetector(mockAnalyzer)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := analyzer.GetContextRecommendationsForFile("src/main.go")
		if err != nil {
			b.Fatalf("Context recommendations benchmark failed: %v", err)
		}
	}
}