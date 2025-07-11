package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

func TestNewPersistentCache(t *testing.T) {
	tempDir := t.TempDir()
	
	config := &Config{
		Directory:     tempDir,
		MaxSize:       10,
		TTL:           time.Hour,
		EnableLRU:     true,
		EnableMetrics: true,
	}
	
	cache, err := NewPersistentCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()
	
	if cache == nil {
		t.Fatal("Cache should not be nil")
	}
	
	// Check that directory was created
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Cache directory should be created")
	}
}

func TestDefaultCacheConfig(t *testing.T) {
	config := DefaultCacheConfig()
	
	if config == nil {
		t.Fatal("Default config should not be nil")
	}
	
	if config.Directory != ".codecontext/cache" {
		t.Errorf("Expected directory '.codecontext/cache', got %s", config.Directory)
	}
	
	if config.MaxSize != 1000 {
		t.Errorf("Expected max size 1000, got %d", config.MaxSize)
	}
	
	if config.TTL != 24*time.Hour {
		t.Errorf("Expected TTL 24h, got %v", config.TTL)
	}
	
	if !config.EnableLRU {
		t.Error("Expected LRU to be enabled")
	}
	
	if !config.EnableMetrics {
		t.Error("Expected metrics to be enabled")
	}
}

func TestPersistentCache_SetGetGraph(t *testing.T) {
	tempDir := t.TempDir()
	
	config := &Config{
		Directory:     tempDir,
		MaxSize:       10,
		TTL:           time.Hour,
		EnableLRU:     true,
		EnableMetrics: true,
	}
	
	cache, err := NewPersistentCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()
	
	// Create test graph
	graph := createTestGraph()
	
	// Set graph
	err = cache.SetGraph("test-key", graph)
	if err != nil {
		t.Fatalf("Failed to set graph: %v", err)
	}
	
	// Get graph
	retrieved := cache.GetGraph("test-key")
	if retrieved == nil {
		t.Fatal("Retrieved graph should not be nil")
	}
	
	// Verify graph content
	if len(retrieved.Files) != len(graph.Files) {
		t.Errorf("Expected %d files, got %d", len(graph.Files), len(retrieved.Files))
	}
	
	if len(retrieved.Symbols) != len(graph.Symbols) {
		t.Errorf("Expected %d symbols, got %d", len(graph.Symbols), len(retrieved.Symbols))
	}
}

func TestPersistentCache_SetGetAST(t *testing.T) {
	tempDir := t.TempDir()
	
	config := &Config{
		Directory:     tempDir,
		MaxSize:       10,
		TTL:           time.Hour,
		EnableLRU:     true,
		EnableMetrics: true,
	}
	
	cache, err := NewPersistentCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()
	
	// Create test AST
	ast := &types.AST{
		FilePath: "test.ts",
		Version:  "v1",
		Content:  "export function test() { return 42; }",
	}
	
	// Set AST
	err = cache.SetAST("test.ts", ast)
	if err != nil {
		t.Fatalf("Failed to set AST: %v", err)
	}
	
	// Get AST
	retrieved := cache.GetAST("test.ts")
	if retrieved == nil {
		t.Fatal("Retrieved AST should not be nil")
	}
	
	// Verify AST content
	if retrieved.FilePath != ast.FilePath {
		t.Errorf("Expected file path %s, got %s", ast.FilePath, retrieved.FilePath)
	}
	
	if retrieved.Content != ast.Content {
		t.Errorf("Expected content %s, got %s", ast.Content, retrieved.Content)
	}
}

func TestPersistentCache_TTL(t *testing.T) {
	tempDir := t.TempDir()
	
	config := &Config{
		Directory:     tempDir,
		MaxSize:       10,
		TTL:           100 * time.Millisecond, // Very short TTL for testing
		EnableLRU:     true,
		EnableMetrics: true,
	}
	
	cache, err := NewPersistentCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()
	
	// Create test graph
	graph := createTestGraph()
	
	// Set graph
	err = cache.SetGraph("test-key", graph)
	if err != nil {
		t.Fatalf("Failed to set graph: %v", err)
	}
	
	// Should be available immediately
	retrieved := cache.GetGraph("test-key")
	if retrieved == nil {
		t.Fatal("Graph should be available immediately")
	}
	
	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)
	
	// Should be expired now
	expired := cache.GetGraph("test-key")
	if expired != nil {
		t.Error("Graph should be expired")
	}
}

func TestPersistentCache_LRUEviction(t *testing.T) {
	tempDir := t.TempDir()
	
	config := &Config{
		Directory:     tempDir,
		MaxSize:       2, // Small size to trigger eviction
		TTL:           time.Hour,
		EnableLRU:     true,
		EnableMetrics: true,
	}
	
	cache, err := NewPersistentCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()
	
	// Create test graphs
	graph1 := createTestGraph()
	graph2 := createTestGraph()
	graph3 := createTestGraph()
	
	// Set first two graphs
	err = cache.SetGraph("key1", graph1)
	if err != nil {
		t.Fatalf("Failed to set graph1: %v", err)
	}
	
	err = cache.SetGraph("key2", graph2)
	if err != nil {
		t.Fatalf("Failed to set graph2: %v", err)
	}
	
	// Access key1 to make it more recently used
	_ = cache.GetGraph("key1")
	
	// Set third graph - should evict key2 (least recently used)
	err = cache.SetGraph("key3", graph3)
	if err != nil {
		t.Fatalf("Failed to set graph3: %v", err)
	}
	
	// key1 and key3 should be available, key2 should be evicted
	if cache.GetGraph("key1") == nil {
		t.Error("key1 should still be available")
	}
	
	if cache.GetGraph("key3") == nil {
		t.Error("key3 should be available")
	}
	
	if cache.GetGraph("key2") != nil {
		t.Error("key2 should be evicted")
	}
}

func TestPersistentCache_Clear(t *testing.T) {
	tempDir := t.TempDir()
	
	config := &Config{
		Directory:     tempDir,
		MaxSize:       10,
		TTL:           time.Hour,
		EnableLRU:     true,
		EnableMetrics: true,
	}
	
	cache, err := NewPersistentCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()
	
	// Set some items
	graph := createTestGraph()
	cache.SetGraph("key1", graph)
	cache.SetGraph("key2", graph)
	
	ast := &types.AST{FilePath: "test.ts", Content: "test"}
	cache.SetAST("test.ts", ast)
	
	// Verify items exist
	if cache.GetGraph("key1") == nil {
		t.Error("key1 should exist before clear")
	}
	
	if cache.GetAST("test.ts") == nil {
		t.Error("AST should exist before clear")
	}
	
	// Clear cache
	cache.Clear()
	
	// Verify items are gone
	if cache.GetGraph("key1") != nil {
		t.Error("key1 should be gone after clear")
	}
	
	if cache.GetAST("test.ts") != nil {
		t.Error("AST should be gone after clear")
	}
}

func TestPersistentCache_Metrics(t *testing.T) {
	tempDir := t.TempDir()
	
	config := &Config{
		Directory:     tempDir,
		MaxSize:       10,
		TTL:           time.Hour,
		EnableLRU:     true,
		EnableMetrics: true,
	}
	
	cache, err := NewPersistentCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()
	
	// Initial metrics should be zero
	metrics := cache.GetMetrics()
	if metrics.Hits != 0 {
		t.Errorf("Expected 0 hits, got %d", metrics.Hits)
	}
	
	if metrics.Misses != 0 {
		t.Errorf("Expected 0 misses, got %d", metrics.Misses)
	}
	
	// Test miss
	_ = cache.GetGraph("nonexistent")
	metrics = cache.GetMetrics()
	if metrics.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", metrics.Misses)
	}
	
	// Test hit
	graph := createTestGraph()
	cache.SetGraph("test", graph)
	_ = cache.GetGraph("test")
	
	metrics = cache.GetMetrics()
	if metrics.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", metrics.Hits)
	}
	
	// Check hit rate
	if metrics.HitRate != 0.5 { // 1 hit out of 2 total
		t.Errorf("Expected hit rate 0.5, got %f", metrics.HitRate)
	}
}

func TestPersistentCache_Persistence(t *testing.T) {
	tempDir := t.TempDir()
	
	config := &Config{
		Directory:     tempDir,
		MaxSize:       10,
		TTL:           time.Hour,
		EnableLRU:     true,
		EnableMetrics: true,
	}
	
	// Create first cache instance
	cache1, err := NewPersistentCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache1: %v", err)
	}
	
	// Set a graph
	graph := createTestGraph()
	err = cache1.SetGraph("persistent-key", graph)
	if err != nil {
		t.Fatalf("Failed to set graph: %v", err)
	}
	
	// Close first cache
	cache1.Close()
	
	// Create second cache instance (should load from disk)
	cache2, err := NewPersistentCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache2: %v", err)
	}
	defer cache2.Close()
	
	// Retrieve graph from second cache
	retrieved := cache2.GetGraph("persistent-key")
	if retrieved == nil {
		t.Fatal("Graph should be persisted and loaded")
	}
	
	// Verify graph content
	if len(retrieved.Files) != len(graph.Files) {
		t.Errorf("Expected %d files, got %d", len(graph.Files), len(retrieved.Files))
	}
}

func TestPersistentCache_NilHandling(t *testing.T) {
	tempDir := t.TempDir()
	
	config := &Config{
		Directory:     tempDir,
		MaxSize:       10,
		TTL:           time.Hour,
		EnableLRU:     true,
		EnableMetrics: true,
	}
	
	cache, err := NewPersistentCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()
	
	// Test setting nil graph
	err = cache.SetGraph("nil-graph", nil)
	if err == nil {
		t.Error("Setting nil graph should return error")
	}
	
	// Test setting nil AST
	err = cache.SetAST("nil-ast", nil)
	if err == nil {
		t.Error("Setting nil AST should return error")
	}
	
	// Test getting non-existent items
	if cache.GetGraph("nonexistent") != nil {
		t.Error("Getting non-existent graph should return nil")
	}
	
	if cache.GetAST("nonexistent") != nil {
		t.Error("Getting non-existent AST should return nil")
	}
}

// Helper function to create test graph
func createTestGraph() *types.CodeGraph {
	return &types.CodeGraph{
		Nodes:   make(map[types.NodeId]*types.GraphNode),
		Edges:   make(map[types.EdgeId]*types.GraphEdge),
		Files:   map[string]*types.FileNode{
			"test.ts": {
				Path:        "test.ts",
				Language:    "typescript",
				Size:        100,
				Lines:       10,
				SymbolCount: 2,
			},
		},
		Symbols: map[types.SymbolId]*types.Symbol{
			"test-symbol": {
				Id:   "test-symbol",
				Name: "TestFunction",
				Type: types.SymbolTypeFunction,
				Location: types.FileLocation{
					FilePath: "test.ts",
					Line:     5,
				},
			},
		},
		Metadata: &types.GraphMetadata{
			TotalFiles:   1,
			TotalSymbols: 1,
			Generated:    time.Now(),
			Version:      "test",
		},
	}
}

// Benchmark tests

func BenchmarkPersistentCache_SetGraph(b *testing.B) {
	tempDir := b.TempDir()
	
	config := &Config{
		Directory:     tempDir,
		MaxSize:       1000,
		TTL:           time.Hour,
		EnableLRU:     true,
		EnableMetrics: true,
	}
	
	cache, err := NewPersistentCache(config)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()
	
	graph := createTestGraph()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.SetGraph(fmt.Sprintf("key-%d", i), graph)
	}
}

func BenchmarkPersistentCache_GetGraph(b *testing.B) {
	tempDir := b.TempDir()
	
	config := &Config{
		Directory:     tempDir,
		MaxSize:       1000,
		TTL:           time.Hour,
		EnableLRU:     true,
		EnableMetrics: true,
	}
	
	cache, err := NewPersistentCache(config)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()
	
	// Populate cache
	graph := createTestGraph()
	for i := 0; i < 100; i++ {
		cache.SetGraph(fmt.Sprintf("key-%d", i), graph)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetGraph(fmt.Sprintf("key-%d", i%100))
	}
}