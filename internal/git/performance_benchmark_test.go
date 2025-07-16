package git

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// BenchmarkSimplePatternDetection benchmarks the core pattern detection algorithm
func BenchmarkSimplePatternDetection(b *testing.B) {
	sizes := []int{10, 50, 100, 500, 1000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("commits_%d", size), func(b *testing.B) {
			commits := generateBenchmarkCommits(size)
			detector := NewSimplePatternsDetector(0.05, 0.3)
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				_, err := detector.MineSimplePatterns(commits)
				if err != nil {
					b.Fatalf("Pattern detection failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkPatternDetectionWithFilters benchmarks pattern detection with file filtering
func BenchmarkPatternDetectionWithFilters(b *testing.B) {
	commits := generateBenchmarkCommits(500)
	detector := NewSimplePatternsDetector(0.05, 0.3)
	
	// Set filter to only include Go files
	detector.SetFileFilter(func(file string) bool {
		return len(file) > 3 && file[len(file)-3:] == ".go"
	})
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := detector.MineSimplePatterns(commits)
		if err != nil {
			b.Fatalf("Pattern detection with filters failed: %v", err)
		}
	}
}

// BenchmarkConfidenceCalculation benchmarks the confidence calculation
func BenchmarkConfidenceCalculation(b *testing.B) {
	commits := generateBenchmarkCommits(1000)
	detector := NewSimplePatternsDetector(0.05, 0.3)
	
	files := []string{"file1.go", "file2.go"}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		detector.calculatePairConfidence(files, commits)
	}
}

// BenchmarkPatternNameGeneration benchmarks pattern name generation
func BenchmarkPatternNameGeneration(b *testing.B) {
	testCases := [][]string{
		{"main.go", "utils.go"},
		{"src/main.go", "src/utils.go", "src/handler.go"},
		{"frontend/components/Header.tsx", "frontend/components/Footer.tsx"},
		{"backend/services/UserService.go", "backend/services/AuthService.go", "backend/services/EmailService.go"},
	}
	
	for _, files := range testCases {
		b.Run(fmt.Sprintf("files_%d", len(files)), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				SimplePatternName(files)
			}
		})
	}
}

// BenchmarkIgnorePatternMatching benchmarks ignore pattern matching
func BenchmarkIgnorePatternMatching(b *testing.B) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "codecontext_bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	analyzer := &GitAnalyzer{repoPath: tempDir}
	pd := NewPatternDetector(analyzer)
	
	testFiles := []string{
		"src/main.go",
		"node_modules/package/index.js",
		"dist/bundle.js",
		"build/output.js",
		"target/classes/Main.class",
		"vendor/package.go",
		"app.log",
		"temp.tmp",
		"cache.cache",
		"__pycache__/module.pyc",
		".git/config",
		"src/components/Header.tsx",
		"src/utils/helpers.ts",
		"test/unit/main_test.go",
		"docs/README.md",
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, file := range testFiles {
			pd.shouldIncludeFile(file)
		}
	}
}

// BenchmarkSemanticAnalysis benchmarks the full semantic analysis pipeline
func BenchmarkSemanticAnalysis(b *testing.B) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "codecontext_semantic_bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	config := DefaultSemanticConfig()
	analyzer, err := NewSemanticAnalyzer(tempDir, config)
	if err != nil {
		b.Fatalf("Failed to create semantic analyzer: %v", err)
	}
	
	// Mock the git analyzer
	commits := generateBenchmarkCommits(200)
	mockGitAnalyzer := &MockGitAnalyzer{
		repoPath: tempDir,
		commits:  commits,
	}
	analyzer.gitAnalyzer = mockGitAnalyzer
	analyzer.patternDetector = NewPatternDetector(mockGitAnalyzer)
	analyzer.patternDetector.SetThresholds(config.MinPatternSupport, config.MinPatternConfidence)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzeRepository()
		if err != nil {
			b.Fatalf("Semantic analysis failed: %v", err)
		}
	}
}

// BenchmarkMemoryUsage benchmarks memory usage during pattern detection
func BenchmarkMemoryUsage(b *testing.B) {
	sizes := []int{100, 500, 1000, 2000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("memory_commits_%d", size), func(b *testing.B) {
			commits := generateBenchmarkCommits(size)
			detector := NewSimplePatternsDetector(0.05, 0.3)
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				patterns, err := detector.MineSimplePatterns(commits)
				if err != nil {
					b.Fatalf("Pattern detection failed: %v", err)
				}
				
				// Force garbage collection to measure actual memory usage
				if i%10 == 0 {
					_ = patterns // Use the patterns to prevent optimization
				}
			}
		})
	}
}

// BenchmarkConcurrentPatternDetection benchmarks concurrent pattern detection
func BenchmarkConcurrentPatternDetection(b *testing.B) {
	commits := generateBenchmarkCommits(500)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			detector := NewSimplePatternsDetector(0.05, 0.3)
			_, err := detector.MineSimplePatterns(commits)
			if err != nil {
				b.Fatalf("Concurrent pattern detection failed: %v", err)
			}
		}
	})
}

// BenchmarkPatternSorting benchmarks the sorting of patterns by frequency
func BenchmarkPatternSorting(b *testing.B) {
	// Generate a large number of patterns
	patterns := make([]FrequentItemset, 1000)
	for i := 0; i < 1000; i++ {
		patterns[i] = FrequentItemset{
			Items:      []string{fmt.Sprintf("file%d.go", i), fmt.Sprintf("file%d.go", (i+1)%1000)},
			Support:    i % 100,
			Confidence: float64(i%100) / 100.0,
			Frequency:  i % 100,
			LastSeen:   time.Now().Add(-time.Duration(i) * time.Hour),
		}
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Copy patterns to avoid modifying the original slice
		testPatterns := make([]FrequentItemset, len(patterns))
		copy(testPatterns, patterns)
		
		// Sort by frequency (this is what the actual code does)
		for j := 0; j < len(testPatterns)-1; j++ {
			for k := j + 1; k < len(testPatterns); k++ {
				if testPatterns[j].Frequency < testPatterns[k].Frequency {
					testPatterns[j], testPatterns[k] = testPatterns[k], testPatterns[j]
				}
			}
		}
	}
}

// BenchmarkLargeFileSetHandling benchmarks handling of commits with many files
func BenchmarkLargeFileSetHandling(b *testing.B) {
	fileCounts := []int{10, 20, 50, 100}
	
	for _, fileCount := range fileCounts {
		b.Run(fmt.Sprintf("files_per_commit_%d", fileCount), func(b *testing.B) {
			// Create commits with many files each
			commits := make([]CommitInfo, 50)
			for i := 0; i < 50; i++ {
				files := make([]string, fileCount)
				for j := 0; j < fileCount; j++ {
					files[j] = fmt.Sprintf("file%d_%d.go", i, j)
				}
				commits[i] = CommitInfo{
					Hash:      fmt.Sprintf("commit%d", i),
					Files:     files,
					Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
					Author:    fmt.Sprintf("dev%d", i%5),
					Message:   fmt.Sprintf("Commit %d with %d files", i, fileCount),
				}
			}
			
			detector := NewSimplePatternsDetector(0.05, 0.3)
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				_, err := detector.MineSimplePatterns(commits)
				if err != nil {
					b.Fatalf("Large file set handling failed: %v", err)
				}
			}
		})
	}
}

// generateBenchmarkCommits creates a set of commits for benchmarking
func generateBenchmarkCommits(count int) []CommitInfo {
	commits := make([]CommitInfo, count)
	now := time.Now()
	
	// Create realistic file patterns
	filePatterns := [][]string{
		{"src/main.go", "src/utils.go"},
		{"src/handler.go", "src/middleware.go"},
		{"src/service.go", "src/repository.go"},
		{"frontend/app.js", "frontend/components.js"},
		{"backend/server.go", "backend/routes.go"},
		{"test/main_test.go", "test/utils_test.go"},
		{"docs/README.md", "docs/CHANGELOG.md"},
		{"config/app.json", "config/database.json"},
	}
	
	for i := 0; i < count; i++ {
		pattern := filePatterns[i%len(filePatterns)]
		
		// Add some variation to make it realistic
		files := make([]string, len(pattern))
		copy(files, pattern)
		
		// Occasionally add extra files
		if i%5 == 0 {
			files = append(files, fmt.Sprintf("extra/file%d.go", i))
		}
		
		commits[i] = CommitInfo{
			Hash:      fmt.Sprintf("commit%d", i),
			Files:     files,
			Timestamp: now.Add(-time.Duration(i) * time.Hour),
			Author:    fmt.Sprintf("dev%d", i%10),
			Message:   fmt.Sprintf("Commit %d", i),
		}
	}
	
	return commits
}