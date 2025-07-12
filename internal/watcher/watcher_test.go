package watcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewFileWatcher(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   string
	}{
		{
			name: "basic config",
			config: Config{
				TargetDir:  "/tmp/test",
				OutputFile: "/tmp/test/output.md",
			},
			want: "/tmp/test",
		},
		{
			name: "with custom debounce",
			config: Config{
				TargetDir:    "/tmp/test",
				OutputFile:   "/tmp/test/output.md",
				DebounceTime: 1 * time.Second,
			},
			want: "/tmp/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			watcher, err := NewFileWatcher(tt.config)
			if err != nil {
				t.Errorf("NewFileWatcher() error = %v", err)
				return
			}
			defer watcher.Stop()

			if watcher.targetDir != tt.want {
				t.Errorf("NewFileWatcher() targetDir = %v, want %v", watcher.targetDir, tt.want)
			}

			// Test default debounce time
			if tt.config.DebounceTime == 0 && watcher.debounce != 500*time.Millisecond {
				t.Errorf("NewFileWatcher() debounce = %v, want %v", watcher.debounce, 500*time.Millisecond)
			}

			// Test default extensions
			if len(tt.config.IncludeExts) == 0 && len(watcher.includeExts) == 0 {
				t.Error("NewFileWatcher() should have default extensions")
			}
		})
	}
}

func TestFileWatcher_shouldExclude(t *testing.T) {
	config := Config{
		TargetDir:       "/tmp/test",
		OutputFile:      "/tmp/test/output.md",
		ExcludePatterns: []string{"node_modules", ".git", "*.log"},
	}

	watcher, err := NewFileWatcher(config)
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}
	defer watcher.Stop()

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "exclude node_modules",
			path: "/tmp/test/node_modules/package.json",
			want: true,
		},
		{
			name: "exclude .git",
			path: "/tmp/test/.git/config",
			want: true,
		},
		{
			name: "include source file",
			path: "/tmp/test/src/index.ts",
			want: false,
		},
		{
			name: "include test file",
			path: "/tmp/test/test/sample.ts",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := watcher.shouldExclude(tt.path); got != tt.want {
				t.Errorf("FileWatcher.shouldExclude() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileWatcher_shouldInclude(t *testing.T) {
	config := Config{
		TargetDir:   "/tmp/test",
		OutputFile:  "/tmp/test/output.md",
		IncludeExts: []string{".ts", ".tsx", ".js", ".json"},
	}

	watcher, err := NewFileWatcher(config)
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}
	defer watcher.Stop()

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "include typescript",
			path: "/tmp/test/src/index.ts",
			want: true,
		},
		{
			name: "include javascript",
			path: "/tmp/test/src/index.js",
			want: true,
		},
		{
			name: "include json",
			path: "/tmp/test/package.json",
			want: true,
		},
		{
			name: "exclude text file",
			path: "/tmp/test/README.txt",
			want: false,
		},
		{
			name: "exclude binary file",
			path: "/tmp/test/image.png",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := watcher.shouldInclude(tt.path); got != tt.want {
				t.Errorf("FileWatcher.shouldInclude() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileWatcher_Integration(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "codecontext-watcher-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFile := filepath.Join(tmpDir, "test.ts")
	err = os.WriteFile(testFile, []byte(`
export function hello() {
  return "Hello, World!";
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "output.md")

	// Create watcher
	config := Config{
		TargetDir:    tmpDir,
		OutputFile:   outputFile,
		DebounceTime: 100 * time.Millisecond,
	}

	watcher, err := NewFileWatcher(config)
	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v", err)
	}
	defer watcher.Stop()

	// Start watcher
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = watcher.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	// Give watcher time to initialize
	time.Sleep(200 * time.Millisecond)

	// Modify test file
	err = os.WriteFile(testFile, []byte(`
export function hello() {
  return "Hello, World! (Modified)";
}

export function goodbye() {
  return "Goodbye!";
}
`), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Wait for debounce and processing
	time.Sleep(500 * time.Millisecond)

	// Check if output file was created/updated
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Check if output file has content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Output file is empty")
	}

	// Basic content validation
	contentStr := string(content)
	if !strings.Contains(contentStr, "CodeContext Map") {
		t.Error("Output file does not contain expected header")
	}
}

func TestFileChange(t *testing.T) {
	change := FileChange{
		Path:      "/tmp/test/file.ts",
		Operation: "WRITE",
		Timestamp: time.Now(),
	}

	if change.Path != "/tmp/test/file.ts" {
		t.Errorf("FileChange.Path = %v, want %v", change.Path, "/tmp/test/file.ts")
	}

	if change.Operation != "WRITE" {
		t.Errorf("FileChange.Operation = %v, want %v", change.Operation, "WRITE")
	}

	if change.Timestamp.IsZero() {
		t.Error("FileChange.Timestamp should not be zero")
	}
}
