package git

import (
	"testing"
	"time"
)

func TestPatternDetector_DetectChangePatterns(t *testing.T) {
	analyzer, err := NewGitAnalyzer(".")
	if err != nil {
		t.Skipf("skipping test: %v", err)
	}

	detector := NewPatternDetector(analyzer)
	detector.SetThresholds(0.05, 0.3) // Lower thresholds for testing

	patterns, err := detector.DetectChangePatterns(30)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Validate patterns
	for _, pattern := range patterns {
		if pattern.Name == "" {
			t.Error("expected non-empty pattern name")
		}
		if len(pattern.Files) == 0 {
			t.Error("expected at least one file in pattern")
		}
		if pattern.Frequency <= 0 {
			t.Error("expected positive frequency")
		}
		if pattern.Confidence < 0 || pattern.Confidence > 1 {
			t.Error("expected confidence between 0 and 1")
		}
		if pattern.LastOccurrence.IsZero() {
			t.Error("expected non-zero last occurrence")
		}
	}

	t.Logf("Found %d change patterns", len(patterns))
}

func TestPatternDetector_DetectFileRelationships(t *testing.T) {
	analyzer, err := NewGitAnalyzer(".")
	if err != nil {
		t.Skipf("skipping test: %v", err)
	}

	detector := NewPatternDetector(analyzer)
	relationships, err := detector.DetectFileRelationships(30)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Validate relationships
	for _, rel := range relationships {
		if rel.File1 == "" || rel.File2 == "" {
			t.Error("expected non-empty file names")
		}
		if rel.File1 == rel.File2 {
			t.Error("file should not be related to itself")
		}
		if rel.Correlation < 0 || rel.Correlation > 1 {
			t.Error("expected correlation between 0 and 1")
		}
		if rel.Frequency < 0 {
			t.Error("expected non-negative frequency")
		}
		if rel.Strength != "strong" && rel.Strength != "moderate" && rel.Strength != "weak" {
			t.Error("expected valid strength classification")
		}
	}

	t.Logf("Found %d file relationships", len(relationships))
}

func TestPatternDetector_DetectModuleGroups(t *testing.T) {
	analyzer, err := NewGitAnalyzer(".")
	if err != nil {
		t.Skipf("skipping test: %v", err)
	}

	detector := NewPatternDetector(analyzer)
	groups, err := detector.DetectModuleGroups(30)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Validate groups
	for _, group := range groups {
		if group.Name == "" {
			t.Error("expected non-empty group name")
		}
		if len(group.Files) < 2 {
			t.Error("expected at least 2 files in group")
		}
		if group.CohesionScore < 0 || group.CohesionScore > 1 {
			t.Error("expected cohesion score between 0 and 1")
		}
		if group.ChangeFrequency < 0 {
			t.Error("expected non-negative change frequency")
		}
		if group.InternalConnections < 0 {
			t.Error("expected non-negative internal connections")
		}
		if group.ExternalConnections < 0 {
			t.Error("expected non-negative external connections")
		}
	}

	t.Logf("Found %d module groups", len(groups))
}

func TestCalculateConfidence(t *testing.T) {
	detector := &PatternDetector{
		minSupport:    0.1,
		minConfidence: 0.6,
	}

	pattern := &ChangePattern{
		Files: []string{"file1.go", "file2.go"},
	}

	commits := []CommitInfo{
		{
			Files: []string{"file1.go", "file2.go"},
		},
		{
			Files: []string{"file1.go", "file2.go"},
		},
		{
			Files: []string{"file1.go", "file3.go"},
		},
		{
			Files: []string{"file2.go", "file3.go"},
		},
	}

	confidence := detector.calculateConfidence(pattern, commits)
	
	// Expected: 2 together, 2 separate = 2/(2+2) = 0.5
	expected := 0.5
	if confidence != expected {
		t.Errorf("expected confidence %f, got %f", expected, confidence)
	}
}

func TestCalculateAvgInterval(t *testing.T) {
	detector := &PatternDetector{
		minSupport:    0.1,
		minConfidence: 0.6,
	}

	pattern := &ChangePattern{
		Files: []string{"file1.go", "file2.go"},
	}

	baseTime := time.Unix(1640995200, 0)
	commits := []CommitInfo{
		{
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: baseTime,
		},
		{
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: baseTime.Add(time.Hour),
		},
		{
			Files:     []string{"file1.go", "file2.go"},
			Timestamp: baseTime.Add(2 * time.Hour),
		},
	}

	interval := detector.calculateAvgInterval(pattern, commits)
	
	// Expected: (1h + 1h) / 2 = 1h
	expected := time.Hour
	if interval != expected {
		t.Errorf("expected interval %v, got %v", expected, interval)
	}
}

func TestFindConnectedComponent(t *testing.T) {
	detector := &PatternDetector{}

	adjacency := map[string][]string{
		"file1.go": {"file2.go", "file3.go"},
		"file2.go": {"file1.go"},
		"file3.go": {"file1.go"},
		"file4.go": {"file5.go"},
		"file5.go": {"file4.go"},
	}

	visited := make(map[string]bool)
	component := detector.findConnectedComponent("file1.go", adjacency, visited)

	expectedFiles := []string{"file1.go", "file2.go", "file3.go"}
	if len(component) != len(expectedFiles) {
		t.Errorf("expected %d files, got %d", len(expectedFiles), len(component))
	}

	for _, file := range expectedFiles {
		found := false
		for _, componentFile := range component {
			if file == componentFile {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected file %s in component", file)
		}
	}
}


func TestGenerateModuleName(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			name:     "empty files",
			files:    []string{},
			expected: "empty-module",
		},
		{
			name:     "common directory",
			files:    []string{"internal/auth/handler.go", "internal/auth/middleware.go"},
			expected: "internal-auth-module",
		},
		{
			name:     "no common directory",
			files:    []string{"main.go", "README.md"},
			expected: "module-2-files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateModuleName(tt.files)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestClassifyStrength(t *testing.T) {
	tests := []struct {
		correlation float64
		expected    string
	}{
		{0.8, "strong"},
		{0.7, "strong"},
		{0.6, "moderate"},
		{0.4, "moderate"},
		{0.3, "weak"},
		{0.1, "weak"},
	}

	for _, tt := range tests {
		result := classifyStrength(tt.correlation)
		if result != tt.expected {
			t.Errorf("correlation %f: expected %s, got %s", tt.correlation, tt.expected, result)
		}
	}
}

func TestCommonPrefix(t *testing.T) {
	tests := []struct {
		a, b     string
		expected string
	}{
		{"internal/auth/handler.go", "internal/auth/middleware.go", "internal/auth/"},
		{"main.go", "main_test.go", "main"},
		{"file1.go", "file2.go", "file"},
		{"completely", "different", ""},
		{"same", "same", "same"},
	}

	for _, tt := range tests {
		result := commonPrefix(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("commonPrefix(%s, %s): expected %s, got %s", tt.a, tt.b, tt.expected, result)
		}
	}
}

func TestContains(t *testing.T) {
	slice := []string{"file1.go", "file2.go", "file3.go"}

	if !contains(slice, "file2.go") {
		t.Error("expected to find file2.go in slice")
	}

	if contains(slice, "file4.go") {
		t.Error("expected not to find file4.go in slice")
	}
}

// Benchmark tests
func BenchmarkDetectChangePatterns(b *testing.B) {
	analyzer, err := NewGitAnalyzer(".")
	if err != nil {
		b.Skipf("skipping benchmark: %v", err)
	}

	detector := NewPatternDetector(analyzer)
	detector.SetThresholds(0.05, 0.3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := detector.DetectChangePatterns(30)
		if err != nil {
			b.Errorf("unexpected error: %v", err)
		}
	}
}

func BenchmarkDetectFileRelationships(b *testing.B) {
	analyzer, err := NewGitAnalyzer(".")
	if err != nil {
		b.Skipf("skipping benchmark: %v", err)
	}

	detector := NewPatternDetector(analyzer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := detector.DetectFileRelationships(30)
		if err != nil {
			b.Errorf("unexpected error: %v", err)
		}
	}
}