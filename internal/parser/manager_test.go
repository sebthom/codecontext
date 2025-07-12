package parser

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Error("NewManager() returned nil")
	}

	if manager.parsers == nil {
		t.Error("Manager parsers not initialized")
	}

	if manager.languages == nil {
		t.Error("Manager languages not initialized")
	}

	if manager.cache == nil {
		t.Error("Manager cache not initialized")
	}
}

func TestGetSupportedLanguages(t *testing.T) {
	manager := NewManager()
	languages := manager.GetSupportedLanguages()

	if len(languages) == 0 {
		t.Error("No supported languages found")
	}

	// Check if TypeScript is supported
	var tsFound bool
	for _, lang := range languages {
		if lang.Name == "typescript" {
			tsFound = true
			if len(lang.Extensions) == 0 {
				t.Error("TypeScript language has no extensions")
			}
			break
		}
	}

	if !tsFound {
		t.Error("TypeScript language not found in supported languages")
	}
}

func TestDetectLanguage(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		name     string
		filePath string
		expected string
		wantNil  bool
	}{
		{
			name:     "typescript file",
			filePath: "test.ts",
			expected: "typescript",
			wantNil:  false,
		},
		{
			name:     "tsx file",
			filePath: "component.tsx",
			expected: "typescript",
			wantNil:  false,
		},
		{
			name:     "javascript file",
			filePath: "script.js",
			expected: "javascript",
			wantNil:  false,
		},
		{
			name:     "jsx file",
			filePath: "component.jsx",
			expected: "javascript",
			wantNil:  false,
		},
		{
			name:     "unsupported file",
			filePath: "document.txt",
			expected: "",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.detectLanguage(tt.filePath)

			if tt.wantNil && result != nil {
				t.Errorf("Expected nil result for %s, got %v", tt.filePath, result)
			}

			if !tt.wantNil && result == nil {
				t.Errorf("Expected non-nil result for %s, got nil", tt.filePath)
			}

			if !tt.wantNil && result != nil && result.Name != tt.expected {
				t.Errorf("Expected language %s for %s, got %s", tt.expected, tt.filePath, result.Name)
			}
		})
	}
}

func TestClassifyFile(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		name         string
		filePath     string
		expectedType string
		expectedTest bool
		expectError  bool
	}{
		{
			name:         "typescript source file",
			filePath:     "src/main.ts",
			expectedType: "source",
			expectedTest: false,
			expectError:  false,
		},
		{
			name:         "typescript test file",
			filePath:     "src/main.test.ts",
			expectedType: "test",
			expectedTest: true,
			expectError:  false,
		},
		{
			name:         "config file",
			filePath:     "config.json",
			expectedType: "config",
			expectedTest: false,
			expectError:  false,
		},
		{
			name:         "unsupported file",
			filePath:     "document.txt",
			expectedType: "",
			expectedTest: false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.ClassifyFile(tt.filePath)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, got nil", tt.filePath)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.filePath, err)
			}

			if !tt.expectError && result != nil {
				if result.FileType != tt.expectedType {
					t.Errorf("Expected file type %s for %s, got %s", tt.expectedType, tt.filePath, result.FileType)
				}

				if result.IsTest != tt.expectedTest {
					t.Errorf("Expected IsTest %v for %s, got %v", tt.expectedTest, tt.filePath, result.IsTest)
				}
			}
		})
	}
}

func TestParseFileVersioned(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		name        string
		filePath    string
		content     string
		version     string
		expectError bool
	}{
		{
			name:        "typescript function",
			filePath:    "test.ts",
			content:     "function hello() { return 'world'; }",
			version:     "1.0",
			expectError: false,
		},
		{
			name:        "unsupported file",
			filePath:    "test.txt",
			content:     "plain text",
			version:     "1.0",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.ParseFileVersioned(tt.filePath, tt.content, tt.version)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, got nil", tt.filePath)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.filePath, err)
			}

			if !tt.expectError && result != nil {
				if result.Version != tt.version {
					t.Errorf("Expected version %s, got %s", tt.version, result.Version)
				}

				if result.AST == nil {
					t.Error("Expected AST to be non-nil")
				}

				if result.AST.FilePath != tt.filePath {
					t.Errorf("Expected file path %s, got %s", tt.filePath, result.AST.FilePath)
				}
			}
		})
	}
}

func TestCalculateHash(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple content",
			content:  "hello world",
			expected: "hash-11",
		},
		{
			name:     "empty content",
			content:  "",
			expected: "hash-0",
		},
		{
			name:     "long content",
			content:  "this is a much longer piece of content for testing",
			expected: "hash-50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateHash(tt.content)
			if result != tt.expected {
				t.Errorf("Expected hash %s for content %s, got %s", tt.expected, tt.content, result)
			}
		})
	}
}
