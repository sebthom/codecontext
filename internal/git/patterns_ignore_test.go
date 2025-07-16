package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExcludePatterns(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "codecontext_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a mock git analyzer
	analyzer := &GitAnalyzer{repoPath: tempDir}
	
	// Test with no .codecontextignore file (should use defaults)
	pd := &PatternDetector{analyzer: analyzer}
	pd.loadExcludePatterns()
	
	if len(pd.excludePatterns) == 0 {
		t.Error("Expected default exclude patterns when no ignore file exists")
	}
	
	// Check that default patterns are loaded
	expectedDefaults := []string{"node_modules/", "dist/", "build/", "target/", ".git/", "vendor/"}
	for _, expected := range expectedDefaults {
		found := false
		for _, pattern := range pd.excludePatterns {
			if pattern == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected default pattern %s not found in exclude patterns", expected)
		}
	}
}

func TestLoadExcludePatternsFromFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "codecontext_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a test .codecontextignore file
	ignoreFile := filepath.Join(tempDir, ".codecontextignore")
	content := `# Test ignore file
node_modules/
*.log
temp/
# Another comment
build/
*.tmp`
	
	err = os.WriteFile(ignoreFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test ignore file: %v", err)
	}
	
	// Create a mock git analyzer
	analyzer := &GitAnalyzer{repoPath: tempDir}
	
	// Test loading from file
	pd := &PatternDetector{analyzer: analyzer}
	pd.loadExcludePatterns()
	
	expectedPatterns := []string{"node_modules/", "*.log", "temp/", "build/", "*.tmp"}
	if len(pd.excludePatterns) != len(expectedPatterns) {
		t.Errorf("Expected %d patterns, got %d", len(expectedPatterns), len(pd.excludePatterns))
	}
	
	for _, expected := range expectedPatterns {
		found := false
		for _, pattern := range pd.excludePatterns {
			if pattern == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected pattern %s not found in exclude patterns", expected)
		}
	}
}

func TestMatchesPattern(t *testing.T) {
	pd := &PatternDetector{}
	
	tests := []struct {
		file     string
		pattern  string
		expected bool
	}{
		// Directory patterns
		{"node_modules/package/index.js", "node_modules/", true},
		{"src/node_modules/package/index.js", "node_modules/", true},
		{"src/main.go", "node_modules/", false},
		
		// Wildcard patterns
		{"app.log", "*.log", true},
		{"error.log", "*.log", true},
		{"app.js", "*.log", false},
		{"temp.tmp", "*.tmp", true},
		{"cache.cache", "*.cache", true},
		
		// Exact matches
		{"build/output.js", "build/", true},
		{"target/classes/Main.class", "target/", true},
		{"vendor/package.go", "vendor/", true},
		{"src/main.go", "build/", false},
		
		// Substring matches
		{"__pycache__/module.pyc", "__pycache__/", true},
		{".git/config", ".git/", true},
		{"src/main.go", ".git/", false},
	}
	
	for _, tt := range tests {
		result := pd.matchesPattern(tt.file, tt.pattern)
		if result != tt.expected {
			t.Errorf("matchesPattern(%q, %q) = %v, expected %v", tt.file, tt.pattern, result, tt.expected)
		}
	}
}

func TestShouldIncludeFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "codecontext_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a mock git analyzer
	analyzer := &GitAnalyzer{repoPath: tempDir}
	
	// Create pattern detector with default patterns
	pd := NewPatternDetector(analyzer)
	
	tests := []struct {
		file     string
		expected bool
	}{
		// Source files should be included
		{"src/main.go", true},
		{"app.js", true},
		{"component.tsx", true},
		{"utils.py", true},
		
		// Hidden files should be excluded (except .codecontextignore)
		{".hidden", false},
		{".git/config", false},
		{".codecontextignore", false}, // This should be excluded by pattern, not hidden rule
		
		// Build artifacts should be excluded
		{"node_modules/package/index.js", false},
		{"dist/bundle.js", false},
		{"build/output.js", false},
		{"target/classes/Main.class", false},
		{"vendor/package.go", false},
		{"__pycache__/module.pyc", false},
		
		// Log and temp files should be excluded
		{"app.log", false},
		{"error.log", false},
		{"temp.tmp", false},
		{"cache.cache", false},
		
		// Config files should be included
		{"package.json", true},
		{"Dockerfile", true},
		{"Makefile", true},
	}
	
	for _, tt := range tests {
		result := pd.shouldIncludeFile(tt.file)
		if result != tt.expected {
			t.Errorf("shouldIncludeFile(%q) = %v, expected %v", tt.file, result, tt.expected)
		}
	}
}