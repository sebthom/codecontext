package git

import (
	"strings"
	"testing"
)

func TestNewSemanticAnalyzer(t *testing.T) {
	config := DefaultSemanticConfig()
	analyzer, err := NewSemanticAnalyzer(".", config)
	if err != nil {
		t.Skipf("skipping test: %v", err)
	}

	if analyzer == nil {
		t.Error("expected non-nil analyzer")
	}
	if analyzer.config != config {
		t.Error("expected config to be set")
	}
	if analyzer.gitAnalyzer == nil {
		t.Error("expected git analyzer to be set")
	}
	if analyzer.patternDetector == nil {
		t.Error("expected pattern detector to be set")
	}
}

func TestDefaultSemanticConfig(t *testing.T) {
	config := DefaultSemanticConfig()
	
	if config.AnalysisPeriodDays != 30 {
		t.Errorf("expected analysis period 30 days, got %d", config.AnalysisPeriodDays)
	}
	if config.MinChangeCorrelation != 0.6 {
		t.Errorf("expected min correlation 0.6, got %f", config.MinChangeCorrelation)
	}
	if config.MinPatternSupport != 0.1 {
		t.Errorf("expected min support 0.1, got %f", config.MinPatternSupport)
	}
	if config.MinPatternConfidence != 0.6 {
		t.Errorf("expected min confidence 0.6, got %f", config.MinPatternConfidence)
	}
	if config.MaxNeighborhoodSize != 10 {
		t.Errorf("expected max neighborhood size 10, got %d", config.MaxNeighborhoodSize)
	}
	if !config.IncludeTestFiles {
		t.Error("expected include test files to be true")
	}
	if config.IncludeDocFiles {
		t.Error("expected include doc files to be false")
	}
	if config.IncludeConfigFiles {
		t.Error("expected include config files to be false")
	}
}

func TestSemanticAnalyzer_AnalyzeRepository(t *testing.T) {
	config := DefaultSemanticConfig()
	config.AnalysisPeriodDays = 30 // Reasonable period for testing
	
	analyzer, err := NewSemanticAnalyzer(".", config)
	if err != nil {
		t.Skipf("skipping test: %v", err)
	}

	result, err := analyzer.AnalyzeRepository()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if result == nil {
		t.Error("expected non-nil result")
		return
	}

	// Validate neighborhoods
	for _, neighborhood := range result.Neighborhoods {
		if neighborhood.Name == "" {
			t.Error("expected non-empty neighborhood name")
		}
		if len(neighborhood.Files) == 0 {
			t.Error("expected at least one file in neighborhood")
		}
		if neighborhood.ChangeFrequency < 0 {
			t.Error("expected non-negative change frequency")
		}
		if neighborhood.CorrelationStrength < 0 || neighborhood.CorrelationStrength > 1 {
			t.Error("expected correlation strength between 0 and 1")
		}
		if neighborhood.Confidence < 0 || neighborhood.Confidence > 1 {
			t.Error("expected confidence between 0 and 1")
		}
	}

	// Validate context recommendations
	for _, rec := range result.ContextRecommendations {
		if rec.ForFile == "" {
			t.Error("expected non-empty for file")
		}
		if len(rec.IncludeFiles) == 0 {
			t.Error("expected at least one include file")
		}
		if rec.Reason == "" {
			t.Error("expected non-empty reason")
		}
		if rec.Confidence < 0 || rec.Confidence > 1 {
			t.Error("expected confidence between 0 and 1")
		}
		if rec.Priority != "high" && rec.Priority != "medium" && rec.Priority != "low" {
			t.Error("expected valid priority")
		}
	}

	// Validate analysis summary
	summary := result.AnalysisSummary
	if summary.TotalFiles < 0 {
		t.Error("expected non-negative total files")
	}
	if summary.NeighborhoodsFound != len(result.Neighborhoods) {
		t.Error("expected neighborhoods found to match actual neighborhoods")
	}
	if summary.PatternsFound != len(result.ChangePatterns) {
		t.Error("expected patterns found to match actual patterns")
	}
	if summary.AnalysisPeriodDays != config.AnalysisPeriodDays {
		t.Error("expected analysis period to match config")
	}
	if summary.AnalysisDate.IsZero() {
		t.Error("expected non-zero analysis date")
	}

	t.Logf("Found %d neighborhoods, %d patterns, %d recommendations", 
		len(result.Neighborhoods), len(result.ChangePatterns), len(result.ContextRecommendations))
}

func TestSemanticAnalyzer_GetContextRecommendationsForFile(t *testing.T) {
	config := DefaultSemanticConfig()
	analyzer, err := NewSemanticAnalyzer(".", config)
	if err != nil {
		t.Skipf("skipping test: %v", err)
	}

	// Test with a specific file (use a common filename)
	testFile := "main.go"
	recommendations, err := analyzer.GetContextRecommendationsForFile(testFile)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Validate recommendations
	for _, rec := range recommendations {
		if rec.ForFile != testFile {
			t.Errorf("expected for file %s, got %s", testFile, rec.ForFile)
		}
		if rec.Reason == "" {
			t.Error("expected non-empty reason")
		}
		if rec.Confidence < 0 || rec.Confidence > 1 {
			t.Error("expected confidence between 0 and 1")
		}
	}

	t.Logf("Found %d recommendations for file %s", len(recommendations), testFile)
}

func TestSemanticAnalyzer_shouldIncludeFile(t *testing.T) {
	config := DefaultSemanticConfig()
	analyzer := &SemanticAnalyzer{config: config}

	tests := []struct {
		file     string
		expected bool
	}{
		{"main.go", true},
		{"handler.go", true},
		{"main_test.go", true}, // IncludeTestFiles = true
		{"README.md", false},   // IncludeDocFiles = false
		{"config.json", false}, // IncludeConfigFiles = false
		{".gitignore", false},  // Hidden file
		{"test.yaml", false},   // IncludeConfigFiles = false
	}

	for _, tt := range tests {
		result := analyzer.shouldIncludeFile(tt.file)
		if result != tt.expected {
			t.Errorf("shouldIncludeFile(%s): expected %v, got %v", tt.file, tt.expected, result)
		}
	}

	// Test with different config
	config.IncludeTestFiles = false
	config.IncludeDocFiles = true
	config.IncludeConfigFiles = true

	tests2 := []struct {
		file     string
		expected bool
	}{
		{"main_test.go", false}, // IncludeTestFiles = false
		{"README.md", true},     // IncludeDocFiles = true
		{"config.json", true},   // IncludeConfigFiles = true
	}

	for _, tt := range tests2 {
		result := analyzer.shouldIncludeFile(tt.file)
		if result != tt.expected {
			t.Errorf("shouldIncludeFile(%s): expected %v, got %v", tt.file, tt.expected, result)
		}
	}
}

func TestSemanticAnalyzer_filterFiles(t *testing.T) {
	config := DefaultSemanticConfig()
	config.IncludeTestFiles = false
	config.IncludeDocFiles = false
	config.IncludeConfigFiles = false

	analyzer := &SemanticAnalyzer{config: config}

	input := []string{
		"main.go",
		"handler.go",
		"main_test.go",
		"README.md",
		"config.json",
		".gitignore",
		"service.go",
	}

	expected := []string{
		"main.go",
		"handler.go",
		"service.go",
	}

	result := analyzer.filterFiles(input)
	if len(result) != len(expected) {
		t.Errorf("expected %d files, got %d", len(expected), len(result))
		return
	}

	for i, file := range result {
		if file != expected[i] {
			t.Errorf("expected file %s, got %s", expected[i], file)
		}
	}
}

func TestSemanticAnalyzer_calculateFileOverlap(t *testing.T) {
	analyzer := &SemanticAnalyzer{}

	tests := []struct {
		files1   []string
		files2   []string
		expected float64
	}{
		{
			[]string{"a", "b", "c"},
			[]string{"b", "c", "d"},
			0.5, // 2 intersection, 4 union = 0.5
		},
		{
			[]string{"a", "b"},
			[]string{"a", "b"},
			1.0, // perfect overlap
		},
		{
			[]string{"a", "b"},
			[]string{"c", "d"},
			0.0, // no overlap
		},
		{
			[]string{},
			[]string{"a", "b"},
			0.0, // empty files1
		},
		{
			[]string{"a"},
			[]string{},
			0.0, // empty files2
		},
	}

	for _, tt := range tests {
		result := analyzer.calculateFileOverlap(tt.files1, tt.files2)
		if result != tt.expected {
			t.Errorf("calculateFileOverlap(%v, %v): expected %f, got %f", tt.files1, tt.files2, tt.expected, result)
		}
	}
}

func TestSemanticAnalyzer_classifyPriority(t *testing.T) {
	analyzer := &SemanticAnalyzer{}

	tests := []struct {
		confidence float64
		expected   string
	}{
		{0.9, "high"},
		{0.8, "high"},
		{0.7, "medium"},
		{0.6, "medium"},
		{0.5, "low"},
		{0.1, "low"},
	}

	for _, tt := range tests {
		result := analyzer.classifyPriority(tt.confidence)
		if result != tt.expected {
			t.Errorf("classifyPriority(%f): expected %s, got %s", tt.confidence, tt.expected, result)
		}
	}
}

func TestSemanticAnalyzer_generateReasonText(t *testing.T) {
	analyzer := &SemanticAnalyzer{}

	tests := []struct {
		neighborhood SemanticNeighborhood
		expectContains string
	}{
		{
			SemanticNeighborhood{
				Name: "auth-module",
				ChangeFrequency: 25,
				CorrelationStrength: 0.7,
			},
			"frequently change together",
		},
		{
			SemanticNeighborhood{
				Name: "core-module",
				ChangeFrequency: 5,
				CorrelationStrength: 0.9,
			},
			"strong correlation",
		},
		{
			SemanticNeighborhood{
				Name: "utils-module",
				ChangeFrequency: 3,
				CorrelationStrength: 0.5,
			},
			"change patterns",
		},
	}

	for _, tt := range tests {
		result := analyzer.generateReasonText(tt.neighborhood)
		if !strings.Contains(result, tt.expectContains) {
			t.Errorf("generateReasonText for %s: expected to contain '%s', got '%s'", tt.neighborhood.Name, tt.expectContains, result)
		}
	}
}

func TestSemanticAnalyzer_mergeSimilarNeighborhoods(t *testing.T) {
	analyzer := &SemanticAnalyzer{}

	neighborhoods := []SemanticNeighborhood{
		{
			Name: "group1",
			Files: []string{"a.go", "b.go", "c.go"},
		},
		{
			Name: "group2",
			Files: []string{"a.go", "b.go", "c.go"}, // Same files - should be merged
		},
		{
			Name: "group3",
			Files: []string{"d.go", "e.go"},
		},
		{
			Name: "group4",
			Files: []string{"a.go", "b.go", "f.go"}, // High overlap with group1
		},
	}

	result := analyzer.mergeSimilarNeighborhoods(neighborhoods)
	
	// Should have fewer neighborhoods after merging
	if len(result) >= len(neighborhoods) {
		t.Errorf("expected fewer neighborhoods after merging, got %d from %d", len(result), len(neighborhoods))
	}

	// Should have at least the distinct groups
	if len(result) < 2 {
		t.Errorf("expected at least 2 distinct groups, got %d", len(result))
	}
}

func TestSemanticAnalyzer_extractCommonOperations(t *testing.T) {
	analyzer := &SemanticAnalyzer{}

	tests := []struct {
		pattern  ChangePattern
		expected []string
	}{
		{
			ChangePattern{
				Frequency: 15,
				Confidence: 0.9,
				Metadata: map[string]string{
					"operations": "add_feature,refactor,test",
				},
			},
			[]string{"add_feature", "refactor", "test"},
		},
		{
			ChangePattern{
				Frequency: 15,
				Confidence: 0.9,
				Metadata: nil,
			},
			[]string{"frequent_updates", "coordinated_changes"},
		},
		{
			ChangePattern{
				Frequency: 5,
				Confidence: 0.5,
				Metadata: nil,
			},
			[]string{},
		},
	}

	for _, tt := range tests {
		result := analyzer.extractCommonOperations(tt.pattern)
		if len(result) != len(tt.expected) {
			t.Errorf("expected %d operations, got %d", len(tt.expected), len(result))
			continue
		}
		
		for i, op := range result {
			if op != tt.expected[i] {
				t.Errorf("expected operation %s, got %s", tt.expected[i], op)
			}
		}
	}
}

// Benchmark tests
func BenchmarkSemanticAnalyzer_AnalyzeRepository(b *testing.B) {
	config := DefaultSemanticConfig()
	config.AnalysisPeriodDays = 7 // Shorter period for benchmarking
	
	analyzer, err := NewSemanticAnalyzer(".", config)
	if err != nil {
		b.Skipf("skipping benchmark: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzeRepository()
		if err != nil {
			b.Errorf("unexpected error: %v", err)
		}
	}
}

func BenchmarkSemanticAnalyzer_GetContextRecommendationsForFile(b *testing.B) {
	config := DefaultSemanticConfig()
	analyzer, err := NewSemanticAnalyzer(".", config)
	if err != nil {
		b.Skipf("skipping benchmark: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.GetContextRecommendationsForFile("main.go")
		if err != nil {
			b.Errorf("unexpected error: %v", err)
		}
	}
}