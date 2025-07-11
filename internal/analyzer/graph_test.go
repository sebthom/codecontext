package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewGraphBuilder(t *testing.T) {
	builder := NewGraphBuilder()
	
	if builder == nil {
		t.Fatal("NewGraphBuilder returned nil")
	}
	
	if builder.parser == nil {
		t.Error("GraphBuilder.parser is nil")
	}
	
	if builder.graph == nil {
		t.Error("GraphBuilder.graph is nil")
	}
	
	if builder.graph.Nodes == nil {
		t.Error("GraphBuilder.graph.Nodes is nil")
	}
	
	if builder.graph.Edges == nil {
		t.Error("GraphBuilder.graph.Edges is nil")
	}
	
	if builder.graph.Files == nil {
		t.Error("GraphBuilder.graph.Files is nil")
	}
	
	if builder.graph.Symbols == nil {
		t.Error("GraphBuilder.graph.Symbols is nil")
	}
}

func TestAnalyzeDirectory(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()
	
	// Create a test TypeScript file
	testFile := filepath.Join(tmpDir, "test.ts")
	testContent := `// Test TypeScript file
export class TestClass {
  private value: number = 0;
  
  public getValue(): number {
    return this.value;
  }
  
  public setValue(newValue: number): void {
    this.value = newValue;
  }
}

export function testFunction(param: string): string {
  return "test: " + param;
}

const testConstant = 42;
`
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Create another test file with imports
	testFile2 := filepath.Join(tmpDir, "importer.ts")
	testContent2 := `import { TestClass, testFunction } from './test';
import * as fs from 'fs';

const instance = new TestClass();
const result = testFunction("hello");
`
	
	err = os.WriteFile(testFile2, []byte(testContent2), 0644)
	if err != nil {
		t.Fatalf("Failed to create second test file: %v", err)
	}
	
	// Test the analyzer
	builder := NewGraphBuilder()
	graph, err := builder.AnalyzeDirectory(tmpDir)
	if err != nil {
		t.Fatalf("AnalyzeDirectory failed: %v", err)
	}
	
	// Verify graph structure
	if graph == nil {
		t.Fatal("Returned graph is nil")
	}
	
	if graph.Metadata == nil {
		t.Fatal("Graph metadata is nil")
	}
	
	// Check that files were processed
	if len(graph.Files) == 0 {
		t.Error("No files were analyzed")
	}
	
	// Check that symbols were extracted
	if len(graph.Symbols) == 0 {
		t.Error("No symbols were extracted")
	}
	
	// Verify specific file was processed
	found := false
	for filePath := range graph.Files {
		if filepath.Base(filePath) == "test.ts" {
			found = true
			break
		}
	}
	if !found {
		t.Error("test.ts was not found in analyzed files")
	}
	
	t.Logf("Analyzed %d files with %d symbols", 
		len(graph.Files), len(graph.Symbols))
	
	// Log symbol details for debugging
	for _, symbol := range graph.Symbols {
		t.Logf("Symbol: %s (%s) at %s:%d", 
			symbol.Name, symbol.Type, 
			filepath.Base(symbol.FullyQualifiedName), symbol.Location.StartLine)
	}
}

func TestIsSupportedFile(t *testing.T) {
	builder := NewGraphBuilder()
	
	tests := []struct {
		path     string
		expected bool
	}{
		{"test.ts", true},
		{"test.tsx", true},
		{"test.js", true},
		{"test.jsx", true},
		{"test.json", true},
		{"test.yaml", true},
		{"test.yml", true},
		{"test.txt", false},
		{"test.py", false},
		{"test.go", false},
		{"README.md", false},
	}
	
	for _, test := range tests {
		result := builder.isSupportedFile(test.path)
		if result != test.expected {
			t.Errorf("isSupportedFile(%q) = %v, expected %v", 
				test.path, result, test.expected)
		}
	}
}

func TestShouldSkipPath(t *testing.T) {
	builder := NewGraphBuilder()
	
	tests := []struct {
		path     string
		expected bool
	}{
		{"src/index.ts", false},
		{"node_modules/package/index.js", true},
		{".git/config", true},
		{"dist/bundle.js", true},
		{"coverage/report.html", true},
		{"test/unit.spec.ts", false},
		{".codecontext/config.yaml", true},
	}
	
	for _, test := range tests {
		result := builder.shouldSkipPath(test.path)
		if result != test.expected {
			t.Errorf("shouldSkipPath(%q) = %v, expected %v", 
				test.path, result, test.expected)
		}
	}
}

func TestGetSupportedLanguages(t *testing.T) {
	builder := NewGraphBuilder()
	languages := builder.GetSupportedLanguages()
	
	if len(languages) == 0 {
		t.Error("GetSupportedLanguages returned empty slice")
	}
	
	// Should include at least JavaScript and TypeScript
	foundJS := false
	foundTS := false
	
	for _, lang := range languages {
		if lang.Name == "javascript" {
			foundJS = true
		}
		if lang.Name == "typescript" {
			foundTS = true
		}
	}
	
	if !foundJS {
		t.Error("JavaScript language not found in supported languages")
	}
	
	if !foundTS {
		t.Error("TypeScript language not found in supported languages")
	}
}