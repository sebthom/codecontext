package compact

import (
	"context"
	"testing"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

func TestRelevanceStrategy_Compact(t *testing.T) {
	strategy := NewRelevanceStrategy()
	testGraph := createTestCodeGraph()

	request := &CompactRequest{
		Graph:   testGraph,
		MaxSize: 5,
		Requirements: &CompactRequirements{
			PreserveFiles: []string{"test1.ts"},
		},
	}

	ctx := context.Background()
	result, err := strategy.Compact(ctx, request)

	if err != nil {
		t.Fatalf("RelevanceStrategy.Compact failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result")
	}

	if result.CompactedGraph == nil {
		t.Error("Expected compacted graph")
	}

	if result.RemovedItems == nil {
		t.Error("Expected removed items")
	}

	// Preserved file should still exist
	if _, exists := result.CompactedGraph.Files["test1.ts"]; !exists {
		t.Error("Preserved file should still exist")
	}

	// Should have metadata about relevance threshold
	if threshold, exists := result.Metadata["relevance_threshold"]; !exists {
		t.Error("Expected relevance_threshold in metadata")
	} else if threshold.(float64) <= 0 {
		t.Error("Expected positive relevance threshold")
	}
}

func TestFrequencyStrategy_Compact(t *testing.T) {
	strategy := NewFrequencyStrategy()
	testGraph := createTestCodeGraph()

	request := &CompactRequest{
		Graph:   testGraph,
		MaxSize: 5,
	}

	ctx := context.Background()
	result, err := strategy.Compact(ctx, request)

	if err != nil {
		t.Fatalf("FrequencyStrategy.Compact failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result")
	}

	if result.CompactedGraph == nil {
		t.Error("Expected compacted graph")
	}

	if result.RemovedItems == nil {
		t.Error("Expected removed items")
	}

	// Should have metadata about removal percentage
	if percentage, exists := result.Metadata["removal_percentage"]; !exists {
		t.Error("Expected removal_percentage in metadata")
	} else if percentage.(float64) != 0.3 {
		t.Errorf("Expected removal percentage 0.3, got %f", percentage.(float64))
	}
}

func TestDependencyStrategy_Compact(t *testing.T) {
	strategy := NewDependencyStrategy()
	testGraph := createTestCodeGraph()

	request := &CompactRequest{
		Graph:   testGraph,
		MaxSize: 5,
	}

	ctx := context.Background()
	result, err := strategy.Compact(ctx, request)

	if err != nil {
		t.Fatalf("DependencyStrategy.Compact failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result")
	}

	if result.CompactedGraph == nil {
		t.Error("Expected compacted graph")
	}

	if result.RemovedItems == nil {
		t.Error("Expected removed items")
	}

	// Should have metadata about dependency analysis
	if _, exists := result.Metadata["isolated_files"]; !exists {
		t.Error("Expected isolated_files in metadata")
	}

	if _, exists := result.Metadata["isolated_symbols"]; !exists {
		t.Error("Expected isolated_symbols in metadata")
	}
}

func TestSizeStrategy_Compact(t *testing.T) {
	strategy := NewSizeStrategy()
	testGraph := createTestCodeGraph()

	// Test with size already under target
	request := &CompactRequest{
		Graph:   testGraph,
		MaxSize: 1000, // Large target size
	}

	ctx := context.Background()
	result, err := strategy.Compact(ctx, request)

	if err != nil {
		t.Fatalf("SizeStrategy.Compact failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result")
	}

	// Should have no action metadata
	if noAction, exists := result.Metadata["no_action"]; !exists || !noAction.(bool) {
		t.Error("Expected no_action to be true when size is already under target")
	}

	// Test with size over target
	request.MaxSize = 2 // Very small target
	result, err = strategy.Compact(ctx, request)

	if err != nil {
		t.Fatalf("SizeStrategy.Compact failed: %v", err)
	}

	// Should have target size metadata
	if targetSize, exists := result.Metadata["target_size"]; !exists {
		t.Error("Expected target_size in metadata")
	} else if targetSize.(int) != 2 {
		t.Errorf("Expected target size 2, got %d", targetSize.(int))
	}

	// Should have removed some files
	if len(result.RemovedItems.Files) == 0 {
		t.Error("Expected some files to be removed")
	}
}

func TestHybridStrategy_Compact(t *testing.T) {
	strategy := NewHybridStrategy()
	testGraph := createTestCodeGraph()

	request := &CompactRequest{
		Graph:   testGraph,
		MaxSize: 5,
	}

	ctx := context.Background()
	result, err := strategy.Compact(ctx, request)

	if err != nil {
		t.Fatalf("HybridStrategy.Compact failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result")
	}

	if result.CompactedGraph == nil {
		t.Error("Expected compacted graph")
	}

	if result.RemovedItems == nil {
		t.Error("Expected removed items")
	}

	// Should have metadata about strategies applied
	if strategiesApplied, exists := result.Metadata["strategies_applied"]; !exists {
		t.Error("Expected strategies_applied in metadata")
	} else if strategiesApplied.(int) != 3 {
		t.Errorf("Expected 3 strategies applied, got %d", strategiesApplied.(int))
	}

	// Should have strategy results
	if _, exists := result.Metadata["strategy_results"]; !exists {
		t.Error("Expected strategy_results in metadata")
	}
}

func TestAdaptiveStrategy_Compact(t *testing.T) {
	strategy := NewAdaptiveStrategy()

	tests := []struct {
		name  string
		graph *types.CodeGraph
	}{
		{
			name:  "small graph",
			graph: createTestCodeGraph(),
		},
		{
			name:  "large graph",
			graph: createLargeTestGraph(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &CompactRequest{
				Graph:   tt.graph,
				MaxSize: 5,
			}

			ctx := context.Background()
			result, err := strategy.Compact(ctx, request)

			if err != nil {
				t.Fatalf("AdaptiveStrategy.Compact failed: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result")
			}

			if result.CompactedGraph == nil {
				t.Error("Expected compacted graph")
			}

			// Should have adaptive choice metadata
			if adaptiveChoice, exists := result.Metadata["adaptive_choice"]; !exists {
				t.Error("Expected adaptive_choice in metadata")
			} else if adaptiveChoice.(string) == "" {
				t.Error("Expected non-empty adaptive choice")
			}

			// Should have graph characteristics
			if _, exists := result.Metadata["graph_characteristics"]; !exists {
				t.Error("Expected graph_characteristics in metadata")
			}
		})
	}
}

func TestBaseStrategy_GetName(t *testing.T) {
	strategy := NewBaseStrategy("test", "Test strategy")

	if strategy.GetName() != "test" {
		t.Errorf("Expected name 'test', got %s", strategy.GetName())
	}
}

func TestBaseStrategy_GetDescription(t *testing.T) {
	strategy := NewBaseStrategy("test", "Test strategy")

	if strategy.GetDescription() != "Test strategy" {
		t.Errorf("Expected description 'Test strategy', got %s", strategy.GetDescription())
	}
}

func TestBaseStrategy_CopyGraph(t *testing.T) {
	strategy := NewBaseStrategy("test", "Test strategy")
	original := createTestCodeGraph()

	copied := strategy.copyGraph(original)

	if copied == nil {
		t.Fatal("Expected copied graph")
	}

	// Verify it's a different instance
	if copied == original {
		t.Error("Expected different instance")
	}

	// Verify same content
	if len(copied.Files) != len(original.Files) {
		t.Errorf("Expected %d files, got %d", len(original.Files), len(copied.Files))
	}

	if len(copied.Symbols) != len(original.Symbols) {
		t.Errorf("Expected %d symbols, got %d", len(original.Symbols), len(copied.Symbols))
	}

	if len(copied.Nodes) != len(original.Nodes) {
		t.Errorf("Expected %d nodes, got %d", len(original.Nodes), len(copied.Nodes))
	}

	if len(copied.Edges) != len(original.Edges) {
		t.Errorf("Expected %d edges, got %d", len(original.Edges), len(copied.Edges))
	}

	// Verify files are copied correctly
	for path, originalFile := range original.Files {
		copiedFile, exists := copied.Files[path]
		if !exists {
			t.Errorf("File %s not found in copied graph", path)
			continue
		}

		if copiedFile.Path != originalFile.Path {
			t.Errorf("File path mismatch: expected %s, got %s", originalFile.Path, copiedFile.Path)
		}

		if copiedFile.Language != originalFile.Language {
			t.Errorf("File language mismatch: expected %s, got %s", originalFile.Language, copiedFile.Language)
		}
	}

	// Verify symbols are copied correctly
	for id, originalSymbol := range original.Symbols {
		copiedSymbol, exists := copied.Symbols[id]
		if !exists {
			t.Errorf("Symbol %s not found in copied graph", id)
			continue
		}

		if copiedSymbol.Name != originalSymbol.Name {
			t.Errorf("Symbol name mismatch: expected %s, got %s", originalSymbol.Name, copiedSymbol.Name)
		}

		if copiedSymbol.Type != originalSymbol.Type {
			t.Errorf("Symbol type mismatch: expected %s, got %s", originalSymbol.Type, copiedSymbol.Type)
		}
	}
}

func TestBaseStrategy_IsFilePreserved(t *testing.T) {
	strategy := NewBaseStrategy("test", "Test strategy")

	requirements := &CompactRequirements{
		PreserveFiles: []string{"important.ts", "critical.js"},
		PreservePaths: []string{"src/core"},
	}

	tests := []struct {
		filePath string
		expected bool
	}{
		{"important.ts", true},
		{"critical.js", true},
		{"src/core/module.ts", true},
		{"other.ts", false},
		{"test/spec.ts", false},
	}

	for _, tt := range tests {
		result := strategy.isFilePreserved(tt.filePath, requirements)
		if result != tt.expected {
			t.Errorf("isFilePreserved(%s) = %v, expected %v", tt.filePath, result, tt.expected)
		}
	}

	// Test with nil requirements
	result := strategy.isFilePreserved("any.ts", nil)
	if result != false {
		t.Error("Expected false when requirements is nil")
	}
}

func TestBaseStrategy_IsSymbolPreserved(t *testing.T) {
	strategy := NewBaseStrategy("test", "Test strategy")

	requirements := &CompactRequirements{
		PreserveSymbols: []types.SymbolId{"important-symbol", "critical-function"},
	}

	tests := []struct {
		symbolId types.SymbolId
		expected bool
	}{
		{"important-symbol", true},
		{"critical-function", true},
		{"other-symbol", false},
	}

	for _, tt := range tests {
		result := strategy.isSymbolPreserved(tt.symbolId, requirements)
		if result != tt.expected {
			t.Errorf("isSymbolPreserved(%s) = %v, expected %v", tt.symbolId, result, tt.expected)
		}
	}

	// Test with nil requirements
	result := strategy.isSymbolPreserved("any-symbol", nil)
	if result != false {
		t.Error("Expected false when requirements is nil")
	}
}

func TestBaseStrategy_CalculateGraphSize(t *testing.T) {
	strategy := NewBaseStrategy("test", "Test strategy")
	graph := createTestCodeGraph()

	size := strategy.calculateGraphSize(graph)
	expectedSize := len(graph.Files) + len(graph.Symbols) + len(graph.Nodes) + len(graph.Edges)

	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}
}

func TestRelevanceStrategy_CalculateRelevanceScores(t *testing.T) {
	strategy := NewRelevanceStrategy()
	graph := createTestCodeGraph()

	requirements := &CompactRequirements{
		PreserveFiles:   []string{"test1.ts"},
		PreserveSymbols: []types.SymbolId{"symbol1"},
	}

	scores := strategy.calculateRelevanceScores(graph, requirements)

	if scores == nil {
		t.Fatal("Expected relevance scores")
	}

	if len(scores.Files) == 0 {
		t.Error("Expected file scores")
	}

	if len(scores.Symbols) == 0 {
		t.Error("Expected symbol scores")
	}

	// Preserved file should have high score
	if score, exists := scores.Files["test1.ts"]; !exists {
		t.Error("Expected score for preserved file")
	} else if score != 1.0 {
		t.Errorf("Expected score 1.0 for preserved file, got %f", score)
	}

	// Preserved symbol should have high score
	if score, exists := scores.Symbols["symbol1"]; !exists {
		t.Error("Expected score for preserved symbol")
	} else if score != 1.0 {
		t.Errorf("Expected score 1.0 for preserved symbol, got %f", score)
	}
}

func TestFrequencyStrategy_CalculateFrequencies(t *testing.T) {
	strategy := NewFrequencyStrategy()
	graph := createTestCodeGraph()

	frequencies := strategy.calculateFrequencies(graph)

	if frequencies == nil {
		t.Fatal("Expected frequency info")
	}

	if len(frequencies.Files) == 0 {
		t.Error("Expected file frequencies")
	}

	if len(frequencies.Symbols) == 0 {
		t.Error("Expected symbol frequencies")
	}

	// All files should have frequency entries
	for filePath := range graph.Files {
		if _, exists := frequencies.Files[filePath]; !exists {
			t.Errorf("Expected frequency for file %s", filePath)
		}
	}

	// All symbols should have frequency entries
	for symbolId := range graph.Symbols {
		if _, exists := frequencies.Symbols[symbolId]; !exists {
			t.Errorf("Expected frequency for symbol %s", symbolId)
		}
	}
}

func TestFrequencyStrategy_SortFilesByFrequency(t *testing.T) {
	strategy := NewFrequencyStrategy()

	frequencies := map[string]int{
		"file1.ts": 5,
		"file2.ts": 2,
		"file3.ts": 8,
		"file4.ts": 1,
	}

	sorted := strategy.sortFilesByFrequency(frequencies)

	if len(sorted) != len(frequencies) {
		t.Errorf("Expected %d items, got %d", len(frequencies), len(sorted))
	}

	// Should be sorted by frequency (ascending)
	for i := 1; i < len(sorted); i++ {
		if sorted[i-1].Frequency > sorted[i].Frequency {
			t.Errorf("Items not sorted correctly: %d > %d", sorted[i-1].Frequency, sorted[i].Frequency)
		}
	}

	// First item should be lowest frequency
	if sorted[0].Frequency != 1 {
		t.Errorf("Expected lowest frequency 1, got %d", sorted[0].Frequency)
	}
}

func TestDependencyStrategy_AnalyzeDependencies(t *testing.T) {
	strategy := NewDependencyStrategy()
	graph := createTestCodeGraph()

	analysis := strategy.analyzeDependencies(graph)

	if analysis == nil {
		t.Fatal("Expected dependency analysis")
	}

	if len(analysis.FileDependencies) == 0 {
		t.Error("Expected file dependencies")
	}

	if len(analysis.SymbolDependencies) == 0 {
		t.Error("Expected symbol dependencies")
	}

	// All files should have dependency info
	for filePath := range graph.Files {
		if _, exists := analysis.FileDependencies[filePath]; !exists {
			t.Errorf("Expected dependency info for file %s", filePath)
		}
	}

	// All symbols should have dependency info
	for symbolId := range graph.Symbols {
		if _, exists := analysis.SymbolDependencies[symbolId]; !exists {
			t.Errorf("Expected dependency info for symbol %s", symbolId)
		}
	}
}

func TestSizeStrategy_CalculateFileSizes(t *testing.T) {
	strategy := NewSizeStrategy()
	graph := createTestCodeGraph()

	sizes := strategy.calculateFileSizes(graph)

	if len(sizes) == 0 {
		t.Error("Expected file sizes")
	}

	// All files should have size entries
	for filePath := range graph.Files {
		if _, exists := sizes[filePath]; !exists {
			t.Errorf("Expected size for file %s", filePath)
		}
	}

	// Sizes should be positive
	for filePath, size := range sizes {
		if size <= 0 {
			t.Errorf("Expected positive size for file %s, got %d", filePath, size)
		}
	}
}

func TestAdaptiveStrategy_AnalyzeGraphCharacteristics(t *testing.T) {
	strategy := NewAdaptiveStrategy()

	tests := []struct {
		name  string
		graph *types.CodeGraph
	}{
		{
			name:  "small graph",
			graph: createTestCodeGraph(),
		},
		{
			name:  "large graph",
			graph: createLargeTestGraph(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			characteristics := strategy.analyzeGraphCharacteristics(tt.graph)

			if characteristics == nil {
				t.Fatal("Expected graph characteristics")
			}

			// Connectivity ratio should be between 0 and 1
			if characteristics.ConnectivityRatio < 0 || characteristics.ConnectivityRatio > 1 {
				t.Errorf("Invalid connectivity ratio: %f", characteristics.ConnectivityRatio)
			}

			// Average file size should be non-negative
			if characteristics.AverageFileSize < 0 {
				t.Errorf("Invalid average file size: %f", characteristics.AverageFileSize)
			}

			// Unused symbol ratio should be between 0 and 1
			if characteristics.UnusedSymbolRatio < 0 || characteristics.UnusedSymbolRatio > 1 {
				t.Errorf("Invalid unused symbol ratio: %f", characteristics.UnusedSymbolRatio)
			}
		})
	}
}

// Benchmark tests

func BenchmarkRelevanceStrategy_Compact(b *testing.B) {
	strategy := NewRelevanceStrategy()
	testGraph := createTestCodeGraph()

	request := &CompactRequest{
		Graph:   testGraph,
		MaxSize: 5,
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		strategy.Compact(ctx, request)
	}
}

func BenchmarkFrequencyStrategy_Compact(b *testing.B) {
	strategy := NewFrequencyStrategy()
	testGraph := createTestCodeGraph()

	request := &CompactRequest{
		Graph:   testGraph,
		MaxSize: 5,
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		strategy.Compact(ctx, request)
	}
}

func BenchmarkHybridStrategy_Compact(b *testing.B) {
	strategy := NewHybridStrategy()
	testGraph := createTestCodeGraph()

	request := &CompactRequest{
		Graph:   testGraph,
		MaxSize: 5,
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		strategy.Compact(ctx, request)
	}
}

func BenchmarkAdaptiveStrategy_Compact(b *testing.B) {
	strategy := NewAdaptiveStrategy()
	testGraph := createLargeTestGraph()

	request := &CompactRequest{
		Graph:   testGraph,
		MaxSize: 20,
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		strategy.Compact(ctx, request)
	}
}
