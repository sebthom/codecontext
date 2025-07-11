package analyzer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

func TestNewIncrementalAnalyzer(t *testing.T) {
	tempDir := t.TempDir()
	
	tests := []struct {
		name   string
		config *IncrementalConfig
	}{
		{
			name:   "default config",
			config: nil,
		},
		{
			name: "custom config",
			config: &IncrementalConfig{
				EnableVGE:          true,
				DiffAlgorithm:      "patience",
				BatchSize:          10,
				BatchTimeout:       1 * time.Second,
				CacheEnabled:       false,
				MaxCacheSize:       500,
				ChangeDetection:    "mtime",
				IncrementalDepth:   5,
				ParallelProcessing: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer, err := NewIncrementalAnalyzer(tempDir, tt.config)
			if err != nil {
				t.Fatalf("NewIncrementalAnalyzer() error = %v", err)
			}

			if analyzer == nil {
				t.Fatal("NewIncrementalAnalyzer() returned nil")
			}

			if analyzer.vge == nil {
				t.Error("VGE should be initialized")
			}

			if analyzer.parser == nil {
				t.Error("Parser should be initialized")
			}

			if analyzer.baseDir != tempDir {
				t.Errorf("Expected baseDir %s, got %s", tempDir, analyzer.baseDir)
			}

			// Check config
			if tt.config == nil {
				// Should use default config
				if !analyzer.config.EnableVGE {
					t.Error("Expected EnableVGE to be true in default config")
				}
			} else {
				// Should use provided config
				if analyzer.config.BatchSize != tt.config.BatchSize {
					t.Errorf("Expected BatchSize %d, got %d", tt.config.BatchSize, analyzer.config.BatchSize)
				}
			}
		})
	}
}

func TestDefaultIncrementalConfig(t *testing.T) {
	config := DefaultIncrementalConfig()
	
	if config == nil {
		t.Fatal("DefaultIncrementalConfig() returned nil")
	}

	// Check default values
	if !config.EnableVGE {
		t.Error("Expected EnableVGE to be true")
	}

	if config.DiffAlgorithm != "myers" {
		t.Errorf("Expected DiffAlgorithm 'myers', got %s", config.DiffAlgorithm)
	}

	if config.BatchSize != 5 {
		t.Errorf("Expected BatchSize 5, got %d", config.BatchSize)
	}

	if config.BatchTimeout != 500*time.Millisecond {
		t.Errorf("Expected BatchTimeout 500ms, got %v", config.BatchTimeout)
	}

	if !config.CacheEnabled {
		t.Error("Expected CacheEnabled to be true")
	}

	if config.MaxCacheSize != 1000 {
		t.Errorf("Expected MaxCacheSize 1000, got %d", config.MaxCacheSize)
	}

	if config.ChangeDetection != "mtime" {
		t.Errorf("Expected ChangeDetection 'mtime', got %s", config.ChangeDetection)
	}

	if config.IncrementalDepth != 3 {
		t.Errorf("Expected IncrementalDepth 3, got %d", config.IncrementalDepth)
	}

	if !config.ParallelProcessing {
		t.Error("Expected ParallelProcessing to be true")
	}
}

func TestIncrementalAnalyzer_Initialize(t *testing.T) {
	tempDir := t.TempDir()
	analyzer, err := NewIncrementalAnalyzer(tempDir, nil)
	if err != nil {
		t.Fatalf("NewIncrementalAnalyzer() error = %v", err)
	}

	// Create test graph
	testGraph := createTestCodeGraph()
	
	err = analyzer.Initialize(testGraph)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Check that file versions are tracked
	if len(analyzer.fileVersions) == 0 {
		t.Error("Expected file versions to be tracked")
	}

	// Verify VGE is initialized
	actualGraph := analyzer.GetCurrentGraph()
	if actualGraph == nil {
		t.Error("Expected current graph to be available")
	}
}

func TestIncrementalAnalyzer_DetectChanges(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create test files
	testFile1 := filepath.Join(tempDir, "test1.ts")
	testFile2 := filepath.Join(tempDir, "test2.ts")
	
	err := os.WriteFile(testFile1, []byte("// Test file 1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	err = os.WriteFile(testFile2, []byte("// Test file 2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	analyzer, err := NewIncrementalAnalyzer(tempDir, nil)
	if err != nil {
		t.Fatalf("NewIncrementalAnalyzer() error = %v", err)
	}

	// Test detecting new files
	changes, err := analyzer.detectChanges([]string{testFile1, testFile2})
	if err != nil {
		t.Fatalf("detectChanges() error = %v", err)
	}

	if len(changes) != 2 {
		t.Errorf("Expected 2 changes, got %d", len(changes))
	}

	for _, change := range changes {
		if change.Type != ChangeTypeAdded {
			t.Errorf("Expected change type %s, got %s", ChangeTypeAdded, change.Type)
		}
	}

	// Test detecting modified files
	time.Sleep(10 * time.Millisecond) // Ensure different mtime
	err = os.WriteFile(testFile1, []byte("// Modified test file 1"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	changes, err = analyzer.detectChanges([]string{testFile1, testFile2})
	if err != nil {
		t.Fatalf("detectChanges() error = %v", err)
	}

	// Should detect one modified file and one unchanged
	modifiedCount := 0
	for _, change := range changes {
		if change.Type == ChangeTypeModified {
			modifiedCount++
		}
	}

	if modifiedCount != 1 {
		t.Errorf("Expected 1 modified file, got %d", modifiedCount)
	}

	// Test detecting removed files
	err = os.Remove(testFile1)
	if err != nil {
		t.Fatalf("Failed to remove test file: %v", err)
	}

	changes, err = analyzer.detectChanges([]string{testFile1})
	if err != nil {
		t.Fatalf("detectChanges() error = %v", err)
	}

	if len(changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(changes))
	}

	if changes[0].Type != ChangeTypeRemoved {
		t.Errorf("Expected change type %s, got %s", ChangeTypeRemoved, changes[0].Type)
	}
}

func TestIncrementalAnalyzer_AnalyzeChanges(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create test TypeScript file
	testFile := filepath.Join(tempDir, "test.ts")
	testContent := `
export interface User {
    id: number;
    name: string;
}

export function createUser(id: number, name: string): User {
    return { id, name };
}

export class UserService {
    getUser(id: number): User | null {
        return null;
    }
}
`
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	analyzer, err := NewIncrementalAnalyzer(tempDir, DefaultIncrementalConfig())
	if err != nil {
		t.Fatalf("NewIncrementalAnalyzer() error = %v", err)
	}

	// Initialize with empty graph
	testGraph := &types.CodeGraph{
		Nodes:   make(map[types.NodeId]*types.GraphNode),
		Edges:   make(map[types.EdgeId]*types.GraphEdge),
		Files:   make(map[string]*types.FileNode),
		Symbols: make(map[types.SymbolId]*types.Symbol),
		Metadata: &types.GraphMetadata{
			Generated: time.Now(),
			Version:   "test",
		},
	}

	err = analyzer.Initialize(testGraph)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Analyze the new file
	ctx := context.Background()
	result, err := analyzer.AnalyzeChanges(ctx, []string{testFile})
	if err != nil {
		t.Fatalf("AnalyzeChanges() error = %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("AnalyzeChanges() returned nil result")
	}

	if len(result.ProcessedChanges) == 0 {
		t.Error("Expected processed changes")
	}

	if result.UpdatedGraph == nil {
		t.Error("Expected updated graph")
	}

	if result.ImpactAnalysis == nil {
		t.Error("Expected impact analysis")
	}

	if result.Performance == nil {
		t.Error("Expected performance metrics")
	}

	// Verify change was processed correctly
	change := result.ProcessedChanges[0]
	if change.Type != ChangeTypeAdded {
		t.Errorf("Expected change type %s, got %s", ChangeTypeAdded, change.Type)
	}

	if change.Path != testFile {
		t.Errorf("Expected path %s, got %s", testFile, change.Path)
	}

	// Verify graph was updated
	if len(result.UpdatedGraph.Files) == 0 {
		t.Error("Expected files in updated graph")
	}
}

func TestChangeTypes(t *testing.T) {
	tests := []struct {
		changeType ChangeType
		expected   string
	}{
		{ChangeTypeAdded, "added"},
		{ChangeTypeModified, "modified"},
		{ChangeTypeRemoved, "removed"},
		{ChangeTypeRenamed, "renamed"},
	}

	for _, tt := range tests {
		if string(tt.changeType) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.changeType))
		}
	}
}

func TestIncrementalAnalyzer_GetVGEMetrics(t *testing.T) {
	tempDir := t.TempDir()
	analyzer, err := NewIncrementalAnalyzer(tempDir, nil)
	if err != nil {
		t.Fatalf("NewIncrementalAnalyzer() error = %v", err)
	}

	metrics := analyzer.GetVGEMetrics()
	if metrics == nil {
		t.Fatal("GetVGEMetrics() returned nil")
	}

	// Check basic metrics structure
	if metrics.LastUpdate.IsZero() {
		t.Error("LastUpdate should not be zero")
	}
}

func TestIncrementalAnalyzer_CacheManagement(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultIncrementalConfig()
	config.CacheEnabled = true
	config.MaxCacheSize = 2 // Small cache for testing

	analyzer, err := NewIncrementalAnalyzer(tempDir, config)
	if err != nil {
		t.Fatalf("NewIncrementalAnalyzer() error = %v", err)
	}

	// Test caching
	ast1 := &types.AST{FilePath: "file1.ts", Content: "content1"}
	ast2 := &types.AST{FilePath: "file2.ts", Content: "content2"}
	ast3 := &types.AST{FilePath: "file3.ts", Content: "content3"}

	analyzer.cacheAST("file1.ts", ast1)
	analyzer.cacheAST("file2.ts", ast2)

	// Should be cached
	if cached := analyzer.getCachedAST("file1.ts"); cached == nil {
		t.Error("Expected file1.ts to be cached")
	}

	if cached := analyzer.getCachedAST("file2.ts"); cached == nil {
		t.Error("Expected file2.ts to be cached")
	}

	// Add third file - should evict first due to cache size limit
	analyzer.cacheAST("file3.ts", ast3)

	// Check cache size
	if len(analyzer.analysisCache) > config.MaxCacheSize {
		t.Errorf("Cache size %d exceeds limit %d", len(analyzer.analysisCache), config.MaxCacheSize)
	}
}

func TestIncrementalAnalyzer_DisabledVGE(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultIncrementalConfig()
	config.EnableVGE = false

	analyzer, err := NewIncrementalAnalyzer(tempDir, config)
	if err != nil {
		t.Fatalf("NewIncrementalAnalyzer() error = %v", err)
	}

	testGraph := createTestCodeGraph()
	
	err = analyzer.Initialize(testGraph)
	if err == nil {
		t.Error("Expected error when VGE is disabled")
	}
}

func TestImpactSummary(t *testing.T) {
	analyzer := &IncrementalAnalyzer{}
	
	changes := []FileChange{
		{Type: ChangeTypeAdded, Path: "new.ts"},
		{Type: ChangeTypeModified, Path: "mod.ts"},
		{Type: ChangeTypeRemoved, Path: "del.ts"},
	}

	summary := analyzer.computeImpactSummary(changes)
	
	if summary == nil {
		t.Fatal("computeImpactSummary() returned nil")
	}

	if summary.TotalChanges != 3 {
		t.Errorf("Expected TotalChanges 3, got %d", summary.TotalChanges)
	}

	if summary.FilesAffected != 3 {
		t.Errorf("Expected FilesAffected 3, got %d", summary.FilesAffected)
	}

	if summary.HighImpactChanges == 0 {
		t.Error("Expected some high impact changes due to removal")
	}

	if summary.RiskScore == 0 {
		t.Error("Expected non-zero risk score")
	}

	if len(summary.Recommendations) == 0 {
		t.Error("Expected some recommendations")
	}
}

// Benchmark tests

func BenchmarkIncrementalAnalyzer_DetectChanges(b *testing.B) {
	tempDir := b.TempDir()
	
	// Create many test files
	files := make([]string, 100)
	for i := 0; i < 100; i++ {
		filePath := filepath.Join(tempDir, fmt.Sprintf("test%d.ts", i))
		err := os.WriteFile(filePath, []byte(fmt.Sprintf("// Test file %d", i)), 0644)
		if err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
		files[i] = filePath
	}

	analyzer, err := NewIncrementalAnalyzer(tempDir, nil)
	if err != nil {
		b.Fatalf("NewIncrementalAnalyzer() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.detectChanges(files)
	}
}

func BenchmarkIncrementalAnalyzer_ProcessFileChange(b *testing.B) {
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "benchmark.ts")
	
	content := `
export interface User {
    id: number;
    name: string;
    email: string;
}

export class UserService {
    private users: User[] = [];

    addUser(user: User): void {
        this.users.push(user);
    }

    getUser(id: number): User | undefined {
        return this.users.find(u => u.id === id);
    }

    removeUser(id: number): boolean {
        const index = this.users.findIndex(u => u.id === id);
        if (index !== -1) {
            this.users.splice(index, 1);
            return true;
        }
        return false;
    }
}
`
	
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	analyzer, err := NewIncrementalAnalyzer(tempDir, nil)
	if err != nil {
		b.Fatalf("NewIncrementalAnalyzer() error = %v", err)
	}

	testGraph := createTestCodeGraph()
	err = analyzer.Initialize(testGraph)
	if err != nil {
		b.Fatalf("Initialize() error = %v", err)
	}

	change := FileChange{
		Path:       testFile,
		Type:       ChangeTypeAdded,
		NewVersion: "v1",
		Timestamp:  time.Now(),
	}

	result := &IncrementalResult{
		ProcessedChanges: make([]FileChange, 0),
		Performance:      &PerformanceMetrics{},
	}

	ctx := context.Background()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		analyzer.processFileChange(ctx, change, result)
	}
}

// Helper function for tests
func createTestCodeGraph() *types.CodeGraph {
	graph := &types.CodeGraph{
		Nodes:   make(map[types.NodeId]*types.GraphNode),
		Edges:   make(map[types.EdgeId]*types.GraphEdge),
		Files:   make(map[string]*types.FileNode),
		Symbols: make(map[types.SymbolId]*types.Symbol),
		Metadata: &types.GraphMetadata{
			TotalFiles:   2,
			TotalSymbols: 3,
			Generated:    time.Now(),
			Version:      "test",
		},
	}

	// Add test files
	graph.Files["test1.ts"] = &types.FileNode{
		Path:        "test1.ts",
		Language:    "typescript",
		Size:        500,
		Lines:       25,
		SymbolCount: 2,
		ImportCount: 1,
	}

	graph.Files["test2.ts"] = &types.FileNode{
		Path:        "test2.ts",
		Language:    "typescript",
		Size:        800,
		Lines:       40,
		SymbolCount: 1,
		ImportCount: 2,
	}

	// Add test symbols
	graph.Symbols["symbol1"] = &types.Symbol{
		Id:   "symbol1",
		Name: "TestFunction",
		Type: types.SymbolTypeFunction,
		Location: types.Location{
			StartLine:   10,
			StartColumn: 1,
			EndLine:     10,
			EndColumn:   30,
		},
		Signature: "function TestFunction(): void",
		Language:  "typescript",
	}

	graph.Symbols["symbol2"] = &types.Symbol{
		Id:   "symbol2",
		Name: "TestClass",
		Type: types.SymbolTypeClass,
		Location: types.Location{
			StartLine:   20,
			StartColumn: 1,
			EndLine:     20,
			EndColumn:   30,
		},
		Signature: "class TestClass",
		Language:  "typescript",
	}

	graph.Symbols["symbol3"] = &types.Symbol{
		Id:   "symbol3",
		Name: "TestVariable",
		Type: types.SymbolTypeVariable,
		Location: types.Location{
			StartLine:   5,
			StartColumn: 1,
			EndLine:     5,
			EndColumn:   30,
		},
		Signature: "const TestVariable: string",
		Language:  "typescript",
	}

	return graph
}