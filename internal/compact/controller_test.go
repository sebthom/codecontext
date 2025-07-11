package compact

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

func TestNewCompactController(t *testing.T) {
	tests := []struct {
		name   string
		config *CompactConfig
	}{
		{
			name:   "default config",
			config: nil,
		},
		{
			name: "custom config",
			config: &CompactConfig{
				EnableCompaction:   true,
				DefaultStrategy:    "relevance",
				MaxContextSize:     5000,
				CompressionRatio:   0.5,
				PriorityThreshold:  0.3,
				CacheEnabled:       false,
				CacheSize:          50,
				MetricsEnabled:     false,
				AdaptiveEnabled:    false,
				AdaptiveThreshold:  0.9,
				BatchSize:          25,
				ParallelProcessing: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := NewCompactController(tt.config)
			
			if controller == nil {
				t.Fatal("NewCompactController returned nil")
			}

			if controller.strategies == nil {
				t.Error("Strategies map should be initialized")
			}

			if controller.config == nil {
				t.Error("Config should be initialized")
			}

			if controller.metrics == nil {
				t.Error("Metrics should be initialized")
			}

			// Check that default strategies are registered
			expectedStrategies := []string{"relevance", "frequency", "dependency", "size", "hybrid", "adaptive"}
			for _, strategy := range expectedStrategies {
				if _, exists := controller.strategies[strategy]; !exists {
					t.Errorf("Expected strategy %s to be registered", strategy)
				}
			}

			// Check config values
			if tt.config == nil {
				// Should use default config
				if !controller.config.EnableCompaction {
					t.Error("Expected EnableCompaction to be true in default config")
				}
				if controller.config.DefaultStrategy != "hybrid" {
					t.Errorf("Expected DefaultStrategy 'hybrid', got %s", controller.config.DefaultStrategy)
				}
			} else {
				// Should use provided config
				if controller.config.MaxContextSize != tt.config.MaxContextSize {
					t.Errorf("Expected MaxContextSize %d, got %d", tt.config.MaxContextSize, controller.config.MaxContextSize)
				}
			}
		})
	}
}

func TestDefaultCompactConfig(t *testing.T) {
	config := DefaultCompactConfig()
	
	if config == nil {
		t.Fatal("DefaultCompactConfig returned nil")
	}

	// Check default values
	if !config.EnableCompaction {
		t.Error("Expected EnableCompaction to be true")
	}

	if config.DefaultStrategy != "hybrid" {
		t.Errorf("Expected DefaultStrategy 'hybrid', got %s", config.DefaultStrategy)
	}

	if config.MaxContextSize != 10000 {
		t.Errorf("Expected MaxContextSize 10000, got %d", config.MaxContextSize)
	}

	if config.CompressionRatio != 0.7 {
		t.Errorf("Expected CompressionRatio 0.7, got %f", config.CompressionRatio)
	}

	if config.PriorityThreshold != 0.5 {
		t.Errorf("Expected PriorityThreshold 0.5, got %f", config.PriorityThreshold)
	}

	if !config.CacheEnabled {
		t.Error("Expected CacheEnabled to be true")
	}

	if config.CacheSize != 100 {
		t.Errorf("Expected CacheSize 100, got %d", config.CacheSize)
	}

	if !config.MetricsEnabled {
		t.Error("Expected MetricsEnabled to be true")
	}

	if !config.AdaptiveEnabled {
		t.Error("Expected AdaptiveEnabled to be true")
	}

	if config.AdaptiveThreshold != 0.8 {
		t.Errorf("Expected AdaptiveThreshold 0.8, got %f", config.AdaptiveThreshold)
	}

	if config.BatchSize != 50 {
		t.Errorf("Expected BatchSize 50, got %d", config.BatchSize)
	}

	if !config.ParallelProcessing {
		t.Error("Expected ParallelProcessing to be true")
	}
}

func TestCompactController_RegisterStrategy(t *testing.T) {
	controller := NewCompactController(nil)
	
	// Create a mock strategy
	mockStrategy := &MockStrategy{
		name:        "mock",
		description: "Mock strategy for testing",
	}
	
	controller.RegisterStrategy("mock", mockStrategy)
	
	// Verify strategy was registered
	strategy, exists := controller.GetStrategy("mock")
	if !exists {
		t.Error("Expected mock strategy to be registered")
	}
	
	if strategy != mockStrategy {
		t.Error("Retrieved strategy should be the same instance")
	}
}

func TestCompactController_ListStrategies(t *testing.T) {
	controller := NewCompactController(nil)
	
	strategies := controller.ListStrategies()
	
	if len(strategies) == 0 {
		t.Error("Expected at least one strategy to be listed")
	}
	
	// Check that strategies are sorted
	for i := 1; i < len(strategies); i++ {
		if strategies[i-1] > strategies[i] {
			t.Error("Strategies should be sorted alphabetically")
		}
	}
	
	// Check that default strategies are present
	expectedStrategies := map[string]bool{
		"relevance":  true,
		"frequency":  true,
		"dependency": true,
		"size":       true,
		"hybrid":     true,
		"adaptive":   true,
	}
	
	for _, strategy := range strategies {
		if expectedStrategies[strategy] {
			delete(expectedStrategies, strategy)
		}
	}
	
	if len(expectedStrategies) > 0 {
		t.Errorf("Missing expected strategies: %v", expectedStrategies)
	}
}

func TestCompactController_Compact(t *testing.T) {
	controller := NewCompactController(nil)
	
	// Create test graph
	testGraph := createTestCodeGraph()
	
	tests := []struct {
		name     string
		request  *CompactRequest
		wantErr  bool
		disabled bool
	}{
		{
			name: "basic compaction",
			request: &CompactRequest{
				Graph:    testGraph,
				Strategy: "relevance",
				MaxSize:  5,
			},
			wantErr: false,
		},
		{
			name: "default strategy",
			request: &CompactRequest{
				Graph:   testGraph,
				MaxSize: 5,
			},
			wantErr: false,
		},
		{
			name: "unknown strategy",
			request: &CompactRequest{
				Graph:    testGraph,
				Strategy: "unknown",
				MaxSize:  5,
			},
			wantErr: true,
		},
		{
			name: "nil graph",
			request: &CompactRequest{
				Graph:    nil,
				Strategy: "relevance",
				MaxSize:  5,
			},
			wantErr: true,
		},
		{
			name: "compaction disabled",
			request: &CompactRequest{
				Graph:    testGraph,
				Strategy: "relevance",
				MaxSize:  5,
			},
			disabled: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.disabled {
				controller.config.EnableCompaction = false
				defer func() { controller.config.EnableCompaction = true }()
			}

			ctx := context.Background()
			result, err := controller.Compact(ctx, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Fatal("Expected result but got nil")
			}

			if result.CompactedGraph == nil {
				t.Error("Expected compacted graph")
			}

			if result.ExecutionTime == 0 && !tt.disabled {
				t.Error("Expected execution time to be recorded")
			}

			if tt.disabled {
				// When disabled, should return original graph
				if result.CompressionRatio != 1.0 {
					t.Error("Expected compression ratio 1.0 when compaction is disabled")
				}
			} else {
				// When enabled, should have strategy set
				if result.Strategy == "" {
					t.Error("Expected strategy to be set")
				}
			}
		})
	}
}

func TestCompactController_CompactMultiple(t *testing.T) {
	controller := NewCompactController(nil)
	testGraph := createTestCodeGraph()
	
	requests := []*CompactRequest{
		{
			Graph:    testGraph,
			Strategy: "relevance",
			MaxSize:  5,
		},
		{
			Graph:    testGraph,
			Strategy: "frequency",
			MaxSize:  8,
		},
		{
			Graph:    testGraph,
			Strategy: "size",
			MaxSize:  3,
		},
	}
	
	ctx := context.Background()
	results, err := controller.CompactMultiple(ctx, requests)
	
	if err != nil {
		t.Fatalf("CompactMultiple failed: %v", err)
	}
	
	if len(results) != len(requests) {
		t.Errorf("Expected %d results, got %d", len(requests), len(results))
	}
	
	for i, result := range results {
		if result == nil {
			t.Errorf("Result %d is nil", i)
			continue
		}
		
		if result.CompactedGraph == nil {
			t.Errorf("Result %d has nil compacted graph", i)
		}
		
		if result.Strategy == "" {
			t.Errorf("Result %d has empty strategy", i)
		}
	}
}

func TestCompactController_AnalyzeCompactionPotential(t *testing.T) {
	controller := NewCompactController(nil)
	testGraph := createTestCodeGraph()
	
	analysis := controller.AnalyzeCompactionPotential(testGraph)
	
	if analysis == nil {
		t.Fatal("AnalyzeCompactionPotential returned nil")
	}
	
	if analysis.TotalFiles != len(testGraph.Files) {
		t.Errorf("Expected TotalFiles %d, got %d", len(testGraph.Files), analysis.TotalFiles)
	}
	
	if analysis.TotalSymbols != len(testGraph.Symbols) {
		t.Errorf("Expected TotalSymbols %d, got %d", len(testGraph.Symbols), analysis.TotalSymbols)
	}
	
	if analysis.TotalNodes != len(testGraph.Nodes) {
		t.Errorf("Expected TotalNodes %d, got %d", len(testGraph.Nodes), analysis.TotalNodes)
	}
	
	if analysis.TotalEdges != len(testGraph.Edges) {
		t.Errorf("Expected TotalEdges %d, got %d", len(testGraph.Edges), analysis.TotalEdges)
	}
	
	if len(analysis.Strategies) == 0 {
		t.Error("Expected strategy analyses")
	}
	
	if analysis.RecommendedStrategy == "" {
		t.Error("Expected recommended strategy")
	}
	
	if analysis.MaxCompressionRatio < 0 || analysis.MaxCompressionRatio > 1 {
		t.Errorf("Invalid compression ratio: %f", analysis.MaxCompressionRatio)
	}
}

func TestCompactController_GetMetrics(t *testing.T) {
	controller := NewCompactController(nil)
	
	metrics := controller.GetMetrics()
	
	if metrics == nil {
		t.Fatal("GetMetrics returned nil")
	}
	
	if metrics.StrategiesUsed == nil {
		t.Error("StrategiesUsed should be initialized")
	}
	
	// Initial metrics should be empty
	if metrics.TotalCompactions != 0 {
		t.Error("Expected TotalCompactions to be 0 initially")
	}
	
	if metrics.CompressionRatio != 0 {
		t.Error("Expected CompressionRatio to be 0 initially")
	}
}

func TestCompactController_ResetMetrics(t *testing.T) {
	controller := NewCompactController(nil)
	
	// Set some metrics
	controller.metrics.TotalCompactions = 10
	controller.metrics.CompressionRatio = 0.5
	controller.metrics.StrategiesUsed["test"] = 5
	
	// Reset metrics
	controller.ResetMetrics()
	
	// Verify reset
	if controller.metrics.TotalCompactions != 0 {
		t.Error("Expected TotalCompactions to be reset to 0")
	}
	
	if controller.metrics.CompressionRatio != 0 {
		t.Error("Expected CompressionRatio to be reset to 0")
	}
	
	if len(controller.metrics.StrategiesUsed) != 0 {
		t.Error("Expected StrategiesUsed to be reset")
	}
}

func TestCompactController_AdaptiveSelection(t *testing.T) {
	config := DefaultCompactConfig()
	config.AdaptiveEnabled = true
	controller := NewCompactController(config)
	
	// Create a large graph to trigger adaptive selection
	largeGraph := createLargeTestGraph()
	
	request := &CompactRequest{
		Graph:   largeGraph,
		MaxSize: 10,
		// No strategy specified - should trigger adaptive selection
	}
	
	ctx := context.Background()
	result, err := controller.Compact(ctx, request)
	
	if err != nil {
		t.Fatalf("Adaptive compaction failed: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result")
	}
	
	// Should have triggered adaptive strategy selection
	if controller.metrics.AdaptiveTriggers == 0 {
		t.Error("Expected adaptive triggers to be incremented")
	}
	
	// Should have metadata about adaptive choice
	if adaptiveChoice, exists := result.Metadata["adaptive_choice"]; !exists {
		t.Error("Expected adaptive_choice in metadata")
	} else if adaptiveChoice == "" {
		t.Error("Expected non-empty adaptive choice")
	}
}

// Mock strategy for testing
type MockStrategy struct {
	name        string
	description string
}

func (ms *MockStrategy) Compact(ctx context.Context, request *CompactRequest) (*CompactResult, error) {
	// Return a simple result that removes nothing
	return &CompactResult{
		CompactedGraph:   request.Graph,
		OriginalSize:     10,
		CompactedSize:    10,
		CompressionRatio: 1.0,
		Strategy:         ms.name,
		ExecutionTime:    time.Millisecond,
		RemovedItems:     &RemovedItems{},
		Metadata:         map[string]interface{}{"mock": true},
	}, nil
}

func (ms *MockStrategy) GetName() string {
	return ms.name
}

func (ms *MockStrategy) GetDescription() string {
	return ms.description
}

// Helper functions

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
		Symbols:     []types.SymbolId{"symbol1", "symbol2"},
		Imports:     []*types.Import{{Path: "./test2"}},
	}

	graph.Files["test2.ts"] = &types.FileNode{
		Path:        "test2.ts",
		Language:    "typescript",
		Size:        800,
		Lines:       40,
		SymbolCount: 1,
		ImportCount: 0,
		Symbols:     []types.SymbolId{"symbol3"},
		Imports:     []*types.Import{},
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

	// Add test nodes
	graph.Nodes["node1"] = &types.GraphNode{
		Id:    "node1",
		Type:  "file",
		Label: "test1.ts",
	}

	graph.Nodes["node2"] = &types.GraphNode{
		Id:    "node2",
		Type:  "file",
		Label: "test2.ts",
	}

	// Add test edge
	graph.Edges["edge1"] = &types.GraphEdge{
		Id:     "edge1",
		From:   "node1",
		To:     "node2",
		Type:   "imports",
		Weight: 1.0,
	}

	return graph
}

func createLargeTestGraph() *types.CodeGraph {
	graph := createTestCodeGraph()
	
	// Add many more files to make it "large"
	for i := 3; i <= 100; i++ {
		fileName := fmt.Sprintf("file%d.ts", i)
		symbolId := types.SymbolId(fmt.Sprintf("symbol%d", i))
		nodeId := types.NodeId(fmt.Sprintf("node%d", i))
		
		graph.Files[fileName] = &types.FileNode{
			Path:        fileName,
			Language:    "typescript",
			Size:        1000,
			Lines:       50,
			SymbolCount: 1,
			ImportCount: 0,
			Symbols:     []types.SymbolId{symbolId},
			Imports:     []*types.Import{},
		}
		
		graph.Symbols[symbolId] = &types.Symbol{
			Id:   symbolId,
			Name: fmt.Sprintf("Symbol%d", i),
			Type: types.SymbolTypeFunction,
			Location: types.Location{
				StartLine:   10,
				StartColumn: 1,
				EndLine:     10,
				EndColumn:   30,
			},
			Language: "typescript",
		}
		
		graph.Nodes[nodeId] = &types.GraphNode{
			Id:    nodeId,
			Type:  "file",
			Label: fileName,
		}
	}
	
	// Update metadata
	graph.Metadata.TotalFiles = len(graph.Files)
	graph.Metadata.TotalSymbols = len(graph.Symbols)
	
	return graph
}

// Benchmark tests

func BenchmarkCompactController_Compact(b *testing.B) {
	controller := NewCompactController(nil)
	testGraph := createTestCodeGraph()
	
	request := &CompactRequest{
		Graph:    testGraph,
		Strategy: "relevance",
		MaxSize:  5,
	}
	
	ctx := context.Background()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		controller.Compact(ctx, request)
	}
}

func BenchmarkCompactController_CompactMultiple(b *testing.B) {
	controller := NewCompactController(nil)
	testGraph := createTestCodeGraph()
	
	requests := []*CompactRequest{
		{Graph: testGraph, Strategy: "relevance", MaxSize: 5},
		{Graph: testGraph, Strategy: "frequency", MaxSize: 8},
		{Graph: testGraph, Strategy: "size", MaxSize: 3},
	}
	
	ctx := context.Background()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		controller.CompactMultiple(ctx, requests)
	}
}

func BenchmarkCompactController_AnalyzeCompactionPotential(b *testing.B) {
	controller := NewCompactController(nil)
	testGraph := createLargeTestGraph()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		controller.AnalyzeCompactionPotential(testGraph)
	}
}