package vgraph

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

func TestNewVirtualGraphEngine(t *testing.T) {
	tests := []struct {
		name   string
		config *VGEConfig
	}{
		{
			name:   "default config",
			config: nil,
		},
		{
			name: "custom config",
			config: &VGEConfig{
				BatchThreshold:  10,
				BatchTimeout:    1 * time.Second,
				MaxShadowMemory: 200 * 1024 * 1024,
				DiffAlgorithm:   "patience",
				EnableMetrics:   false,
				GCThreshold:     0.9,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vge := NewVirtualGraphEngine(tt.config)
			
			if vge == nil {
				t.Fatal("NewVirtualGraphEngine returned nil")
			}

			if vge.shadow == nil {
				t.Error("Shadow graph should be initialized")
			}

			if vge.actual == nil {
				t.Error("Actual graph should be initialized")
			}

			if vge.differ == nil {
				t.Error("Differ should be initialized")
			}

			if vge.reconciler == nil {
				t.Error("Reconciler should be initialized")
			}

			if vge.batcher == nil {
				t.Error("Batcher should be initialized")
			}

			// Check config
			if tt.config == nil {
				// Should use default config
				if vge.config.BatchThreshold != 5 {
					t.Errorf("Expected default BatchThreshold 5, got %d", vge.config.BatchThreshold)
				}
			} else {
				// Should use provided config
				if vge.config.BatchThreshold != tt.config.BatchThreshold {
					t.Errorf("Expected BatchThreshold %d, got %d", tt.config.BatchThreshold, vge.config.BatchThreshold)
				}
			}
		})
	}
}

func TestVirtualGraphEngine_Initialize(t *testing.T) {
	vge := NewVirtualGraphEngine(nil)
	
	// Create a test graph
	testGraph := createTestCodeGraph()
	
	err := vge.Initialize(testGraph)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Check that actual graph is set
	actualGraph := vge.GetActualGraph()
	if actualGraph == nil {
		t.Fatal("Actual graph should not be nil after initialization")
	}

	// Check that shadow graph is a copy of actual
	shadowGraph := vge.GetShadowGraph()
	if shadowGraph == nil {
		t.Fatal("Shadow graph should not be nil after initialization")
	}

	// Verify they have the same number of elements but are different instances
	if len(actualGraph.Files) != len(shadowGraph.Files) {
		t.Errorf("Files count mismatch: actual=%d, shadow=%d", 
			len(actualGraph.Files), len(shadowGraph.Files))
	}

	if len(actualGraph.Symbols) != len(shadowGraph.Symbols) {
		t.Errorf("Symbols count mismatch: actual=%d, shadow=%d", 
			len(actualGraph.Symbols), len(shadowGraph.Symbols))
	}
}

func TestVirtualGraphEngine_QueueChange(t *testing.T) {
	vge := NewVirtualGraphEngine(&VGEConfig{
		BatchThreshold: 3, // Low threshold for testing
		BatchTimeout:   100 * time.Millisecond,
	})

	testGraph := createTestCodeGraph()
	err := vge.Initialize(testGraph)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Create test changes
	change1 := ChangeSet{
		ID:       "change1",
		Type:     ChangeTypeFileModify,
		FilePath: "test1.ts",
		Changes: []Change{
			{
				Type:     ChangeTypeFileModify,
				Target:   "test1.ts",
				OldValue: "old content",
				NewValue: "new content",
			},
		},
		Timestamp: time.Now(),
	}

	change2 := ChangeSet{
		ID:       "change2",
		Type:     ChangeTypeSymbolAdd,
		FilePath: "test2.ts",
		Changes: []Change{
			{
				Type:     ChangeTypeSymbolAdd,
				Target:   "newSymbol",
				OldValue: nil,
				NewValue: &types.Symbol{
					Id:   "test-symbol",
					Name: "TestSymbol",
					Type: types.SymbolTypeFunction,
				},
			},
		},
		Timestamp: time.Now(),
	}

	// Queue changes
	err = vge.QueueChange(change1)
	if err != nil {
		t.Errorf("QueueChange failed: %v", err)
	}

	err = vge.QueueChange(change2)
	if err != nil {
		t.Errorf("QueueChange failed: %v", err)
	}

	// Check metrics
	metrics := vge.GetMetrics()
	if metrics.TotalChanges != 2 {
		t.Errorf("Expected 2 total changes, got %d", metrics.TotalChanges)
	}
}

func TestVirtualGraphEngine_ProcessPendingChanges(t *testing.T) {
	vge := NewVirtualGraphEngine(nil)
	testGraph := createTestCodeGraph()
	err := vge.Initialize(testGraph)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Add a file change
	newFile := &types.FileNode{
		Path:        "new-file.ts",
		Language:    "typescript",
		Size:        1000,
		Lines:       50,
		SymbolCount: 5,
		ImportCount: 2,
	}

	change := ChangeSet{
		ID:       "test-change",
		Type:     ChangeTypeFileAdd,
		FilePath: "new-file.ts",
		Changes: []Change{
			{
				Type:     ChangeTypeFileAdd,
				Target:   "new-file.ts",
				OldValue: nil,
				NewValue: newFile,
			},
		},
		Timestamp: time.Now(),
	}

	// Queue the change
	err = vge.QueueChange(change)
	if err != nil {
		t.Fatalf("QueueChange failed: %v", err)
	}

	// Process pending changes
	ctx := context.Background()
	err = vge.ProcessPendingChanges(ctx)
	if err != nil {
		t.Fatalf("ProcessPendingChanges failed: %v", err)
	}

	// Verify the change was applied to actual graph
	actualGraph := vge.GetActualGraph()
	if _, exists := actualGraph.Files["new-file.ts"]; !exists {
		t.Error("File should have been added to actual graph")
	}

	// Check metrics
	metrics := vge.GetMetrics()
	if metrics.BatchesProcessed == 0 {
		t.Error("Expected at least one batch to be processed")
	}
}

func TestVirtualGraphEngine_GetMetrics(t *testing.T) {
	vge := NewVirtualGraphEngine(nil)
	
	metrics := vge.GetMetrics()
	if metrics == nil {
		t.Fatal("GetMetrics should not return nil")
	}

	// Check default values
	if metrics.TotalChanges != 0 {
		t.Errorf("Expected TotalChanges to be 0, got %d", metrics.TotalChanges)
	}

	if metrics.BatchesProcessed != 0 {
		t.Errorf("Expected BatchesProcessed to be 0, got %d", metrics.BatchesProcessed)
	}

	if metrics.LastUpdate.IsZero() {
		t.Error("LastUpdate should not be zero")
	}
}

func TestVirtualGraphEngine_Reset(t *testing.T) {
	vge := NewVirtualGraphEngine(nil)
	testGraph := createTestCodeGraph()
	
	// Initialize with test data
	err := vge.Initialize(testGraph)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Add some changes
	change := ChangeSet{
		ID:       "test-change",
		Type:     ChangeTypeFileAdd,
		FilePath: "test.ts",
		Changes:  []Change{},
	}
	
	err = vge.QueueChange(change)
	if err != nil {
		t.Fatalf("QueueChange failed: %v", err)
	}

	// Reset the engine
	err = vge.Reset()
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	// Verify everything is reset
	actualGraph := vge.GetActualGraph()
	if len(actualGraph.Files) != 0 {
		t.Error("Actual graph should be empty after reset")
	}

	shadowGraph := vge.GetShadowGraph()
	if len(shadowGraph.Files) != 0 {
		t.Error("Shadow graph should be empty after reset")
	}

	metrics := vge.GetMetrics()
	if metrics.TotalChanges != 0 {
		t.Error("Metrics should be reset")
	}
}

func TestDefaultVGEConfig(t *testing.T) {
	config := DefaultVGEConfig()
	
	if config == nil {
		t.Fatal("DefaultVGEConfig should not return nil")
	}

	// Check default values
	if config.BatchThreshold != 5 {
		t.Errorf("Expected BatchThreshold 5, got %d", config.BatchThreshold)
	}

	if config.BatchTimeout != 500*time.Millisecond {
		t.Errorf("Expected BatchTimeout 500ms, got %v", config.BatchTimeout)
	}

	if config.MaxShadowMemory != 100*1024*1024 {
		t.Errorf("Expected MaxShadowMemory 100MB, got %d", config.MaxShadowMemory)
	}

	if config.DiffAlgorithm != "myers" {
		t.Errorf("Expected DiffAlgorithm 'myers', got %s", config.DiffAlgorithm)
	}

	if !config.EnableMetrics {
		t.Error("Expected EnableMetrics to be true")
	}

	if config.GCThreshold != 0.8 {
		t.Errorf("Expected GCThreshold 0.8, got %f", config.GCThreshold)
	}
}

func TestChangeTypes(t *testing.T) {
	// Test change type constants
	tests := []struct {
		changeType ChangeType
		expected   string
	}{
		{ChangeTypeFileAdd, "file_add"},
		{ChangeTypeFileModify, "file_modify"},
		{ChangeTypeFileDelete, "file_delete"},
		{ChangeTypeSymbolAdd, "symbol_add"},
		{ChangeTypeSymbolMod, "symbol_modify"},
		{ChangeTypeSymbolDel, "symbol_delete"},
	}

	for _, tt := range tests {
		if string(tt.changeType) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.changeType))
		}
	}
}

func TestPatchTypes(t *testing.T) {
	// Test patch type constants
	tests := []struct {
		patchType PatchType
		expected  string
	}{
		{PatchTypeAdd, "add"},
		{PatchTypeRemove, "remove"},
		{PatchTypeModify, "modify"},
		{PatchTypeReorder, "reorder"},
	}

	for _, tt := range tests {
		if string(tt.patchType) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.patchType))
		}
	}
}

// Helper function to create a test code graph
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

// Benchmark tests

func BenchmarkVirtualGraphEngine_QueueChange(b *testing.B) {
	vge := NewVirtualGraphEngine(nil)
	testGraph := createTestCodeGraph()
	vge.Initialize(testGraph)

	change := ChangeSet{
		ID:       "bench-change",
		Type:     ChangeTypeFileModify,
		FilePath: "test.ts",
		Changes: []Change{
			{
				Type:     ChangeTypeFileModify,
				Target:   "test.ts",
				OldValue: "old",
				NewValue: "new",
			},
		},
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		change.ID = fmt.Sprintf("change-%d", i)
		vge.QueueChange(change)
	}
}

func BenchmarkVirtualGraphEngine_ProcessPendingChanges(b *testing.B) {
	vge := NewVirtualGraphEngine(nil)
	testGraph := createTestCodeGraph()
	vge.Initialize(testGraph)

	// Pre-populate with changes
	for i := 0; i < 10; i++ {
		change := ChangeSet{
			ID:       fmt.Sprintf("change-%d", i),
			Type:     ChangeTypeFileModify,
			FilePath: "test.ts",
			Changes: []Change{
				{
					Type:     ChangeTypeFileModify,
					Target:   "test.ts",
					OldValue: "old",
					NewValue: "new",
				},
			},
			Timestamp: time.Now(),
		}
		vge.QueueChange(change)
	}

	ctx := context.Background()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		vge.ProcessPendingChanges(ctx)
	}
}