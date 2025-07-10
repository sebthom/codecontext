package parser

import (
	"testing"
	"time"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

func TestNewASTCache(t *testing.T) {
	cache := NewASTCache()
	
	if cache == nil {
		t.Error("NewASTCache() returned nil")
	}
	
	if cache.astCache == nil {
		t.Error("astCache not initialized")
	}
	
	if cache.diffCache == nil {
		t.Error("diffCache not initialized")
	}
	
	if cache.timestamps == nil {
		t.Error("timestamps not initialized")
	}
}

func TestASTCache_SetAndGet(t *testing.T) {
	cache := NewASTCache()
	
	ast := &types.VersionedAST{
		AST: &types.AST{
			FilePath: "test.ts",
			Language: "typescript",
			Content:  "function test() {}",
		},
		Version: "1.0",
		Hash:    "test-hash",
	}
	
	// Test Set
	err := cache.Set("test.ts", ast)
	if err != nil {
		t.Errorf("Set() failed: %v", err)
	}
	
	// Test Get
	result, err := cache.Get("test.ts", "1.0")
	if err != nil {
		t.Errorf("Get() failed: %v", err)
	}
	
	if result == nil {
		t.Error("Get() returned nil")
	}
	
	if result.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", result.Version)
	}
	
	if result.AST.FilePath != "test.ts" {
		t.Errorf("Expected file path test.ts, got %s", result.AST.FilePath)
	}
}

func TestASTCache_GetNonExistent(t *testing.T) {
	cache := NewASTCache()
	
	result, err := cache.Get("non-existent.ts")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
	
	if result != nil {
		t.Error("Expected nil result for non-existent file")
	}
}

func TestASTCache_Invalidate(t *testing.T) {
	cache := NewASTCache()
	
	ast := &types.VersionedAST{
		AST: &types.AST{
			FilePath: "test.ts",
			Language: "typescript",
		},
		Version: "1.0",
		Hash:    "test-hash",
	}
	
	// Set an entry
	err := cache.Set("test.ts", ast)
	if err != nil {
		t.Errorf("Set() failed: %v", err)
	}
	
	// Verify it exists
	_, err = cache.Get("test.ts", "1.0")
	if err != nil {
		t.Errorf("Get() failed before invalidation: %v", err)
	}
	
	// Invalidate
	err = cache.Invalidate("test.ts")
	if err != nil {
		t.Errorf("Invalidate() failed: %v", err)
	}
	
	// Verify it's gone
	_, err = cache.Get("test.ts", "1.0")
	if err == nil {
		t.Error("Expected error after invalidation, got nil")
	}
}

func TestASTCache_Clear(t *testing.T) {
	cache := NewASTCache()
	
	ast := &types.VersionedAST{
		AST: &types.AST{
			FilePath: "test.ts",
			Language: "typescript",
		},
		Version: "1.0",
		Hash:    "test-hash",
	}
	
	// Set multiple entries
	cache.Set("test1.ts", ast)
	cache.Set("test2.ts", ast)
	
	// Verify size
	if cache.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cache.Size())
	}
	
	// Clear
	err := cache.Clear()
	if err != nil {
		t.Errorf("Clear() failed: %v", err)
	}
	
	// Verify it's empty
	if cache.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", cache.Size())
	}
}

func TestASTCache_DiffCache(t *testing.T) {
	cache := NewASTCache()
	
	diffs := []*types.ASTDiff{
		{
			FileId:      "test.ts",
			FromVersion: "1.0",
			ToVersion:   "2.0",
		},
	}
	
	// Set diff cache
	err := cache.SetDiffCache("test.ts", diffs)
	if err != nil {
		t.Errorf("SetDiffCache() failed: %v", err)
	}
	
	// Get diff cache
	result, err := cache.GetDiffCache("test.ts")
	if err != nil {
		t.Errorf("GetDiffCache() failed: %v", err)
	}
	
	if len(result) != 1 {
		t.Errorf("Expected 1 diff, got %d", len(result))
	}
	
	if result[0].FileId != "test.ts" {
		t.Errorf("Expected file ID test.ts, got %s", result[0].FileId)
	}
}

func TestASTCache_TTL(t *testing.T) {
	cache := NewASTCache()
	cache.SetTTL(time.Millisecond * 100) // Very short TTL for testing
	
	ast := &types.VersionedAST{
		AST: &types.AST{
			FilePath: "test.ts",
			Language: "typescript",
		},
		Version: "1.0",
		Hash:    "test-hash",
	}
	
	// Set entry
	err := cache.Set("test.ts", ast)
	if err != nil {
		t.Errorf("Set() failed: %v", err)
	}
	
	// Get immediately - should work
	_, err = cache.Get("test.ts", "1.0")
	if err != nil {
		t.Errorf("Get() failed immediately after set: %v", err)
	}
	
	// Wait for TTL to expire
	time.Sleep(time.Millisecond * 150)
	
	// Get after TTL - should fail
	_, err = cache.Get("test.ts", "1.0")
	if err == nil {
		t.Error("Expected error after TTL expiry, got nil")
	}
}

func TestASTCache_MaxSize(t *testing.T) {
	cache := NewASTCache()
	cache.SetMaxSize(2) // Small size for testing
	
	ast1 := &types.VersionedAST{
		AST: &types.AST{FilePath: "test1.ts", Language: "typescript"},
		Version: "1.0",
		Hash:    "hash1",
	}
	
	ast2 := &types.VersionedAST{
		AST: &types.AST{FilePath: "test2.ts", Language: "typescript"},
		Version: "1.0",
		Hash:    "hash2",
	}
	
	ast3 := &types.VersionedAST{
		AST: &types.AST{FilePath: "test3.ts", Language: "typescript"},
		Version: "1.0",
		Hash:    "hash3",
	}
	
	// Add entries
	cache.Set("test1.ts", ast1)
	cache.Set("test2.ts", ast2)
	
	// Should be at max size
	if cache.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cache.Size())
	}
	
	// Add another entry - should evict the oldest
	cache.Set("test3.ts", ast3)
	
	// Should still be at max size
	if cache.Size() != 2 {
		t.Errorf("Expected size 2 after eviction, got %d", cache.Size())
	}
	
	// The oldest entry should be gone
	_, err := cache.Get("test1.ts", "1.0")
	if err == nil {
		t.Error("Expected first entry to be evicted")
	}
	
	// The newest entry should still be there
	_, err = cache.Get("test3.ts", "1.0")
	if err != nil {
		t.Errorf("Expected newest entry to be present: %v", err)
	}
}

func TestASTCache_Stats(t *testing.T) {
	cache := NewASTCache()
	
	ast := &types.VersionedAST{
		AST: &types.AST{
			FilePath: "test.ts",
			Language: "typescript",
		},
		Version: "1.0",
		Hash:    "test-hash",
	}
	
	// Add some entries
	cache.Set("test1.ts", ast)
	cache.Set("test2.ts", ast)
	
	stats := cache.Stats()
	
	if stats["ast_entries"] != 2 {
		t.Errorf("Expected 2 AST entries, got %v", stats["ast_entries"])
	}
	
	if stats["diff_entries"] != 0 {
		t.Errorf("Expected 0 diff entries, got %v", stats["diff_entries"])
	}
	
	if stats["max_size"] != 1000 {
		t.Errorf("Expected max size 1000, got %v", stats["max_size"])
	}
}