package cache

import (
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

// Config holds configuration for the persistent cache
type Config struct {
	Directory     string        `json:"directory"`
	MaxSize       int           `json:"max_size"`       // Maximum number of cached items
	TTL           time.Duration `json:"ttl"`            // Time to live for cache entries
	EnableLRU     bool          `json:"enable_lru"`     // Enable LRU eviction
	EnableMetrics bool          `json:"enable_metrics"` // Enable metrics collection
	Compression   bool          `json:"compression"`    // Enable compression (future)
}

// PersistentCache provides disk-backed caching for CodeGraph objects
type PersistentCache struct {
	config  *Config
	items   map[string]*CacheItem
	access  map[string]time.Time // For LRU tracking
	mutex   sync.RWMutex
	metrics *CacheMetrics
}

// CacheItem represents a cached item with metadata
type CacheItem struct {
	Key         string                 `json:"key"`
	Graph       *types.CodeGraph       `json:"graph"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	AccessedAt  time.Time              `json:"accessed_at"`
	AccessCount int64                  `json:"access_count"`
	Size        int64                  `json:"size"` // Estimated size in bytes
	Hash        string                 `json:"hash"` // Content hash for validation
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	Hits        int64     `json:"hits"`
	Misses      int64     `json:"misses"`
	Evictions   int64     `json:"evictions"`
	TotalSize   int64     `json:"total_size"`
	HitRate     float64   `json:"hit_rate"`
	LastCleanup time.Time `json:"last_cleanup"`
	mutex       sync.RWMutex
}

// NewPersistentCache creates a new persistent cache
func NewPersistentCache(config *Config) (*PersistentCache, error) {
	if config == nil {
		config = DefaultCacheConfig()
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(config.Directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &PersistentCache{
		config:  config,
		items:   make(map[string]*CacheItem),
		access:  make(map[string]time.Time),
		metrics: &CacheMetrics{},
	}

	// Load existing cache from disk
	if err := cache.loadFromDisk(); err != nil {
		// Log error but don't fail - start with empty cache
		fmt.Printf("Warning: failed to load cache from disk: %v\n", err)
	}

	// Start background cleanup if TTL is enabled
	if config.TTL > 0 {
		go cache.startCleanupWorker()
	}

	return cache, nil
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() *Config {
	return &Config{
		Directory:     ".codecontext/cache",
		MaxSize:       1000,
		TTL:           24 * time.Hour,
		EnableLRU:     true,
		EnableMetrics: true,
		Compression:   false,
	}
}

// GetGraph retrieves a cached graph by key
func (pc *PersistentCache) GetGraph(key string) *types.CodeGraph {
	pc.mutex.RLock()
	item, exists := pc.items[key]
	pc.mutex.RUnlock()

	if !exists {
		pc.recordMiss()
		return nil
	}

	// Check TTL
	if pc.config.TTL > 0 && time.Since(item.CreatedAt) > pc.config.TTL {
		pc.mutex.Lock()
		delete(pc.items, key)
		delete(pc.access, key)
		pc.mutex.Unlock()
		pc.recordMiss()
		return nil
	}

	// Update access information
	pc.mutex.Lock()
	item.AccessedAt = time.Now()
	item.AccessCount++
	if pc.config.EnableLRU {
		pc.access[key] = time.Now()
	}
	pc.mutex.Unlock()

	pc.recordHit()
	return item.Graph
}

// SetGraph stores a graph in the cache
func (pc *PersistentCache) SetGraph(key string, graph *types.CodeGraph) error {
	if graph == nil {
		return fmt.Errorf("cannot cache nil graph")
	}

	// Calculate content hash for validation
	hash := pc.calculateGraphHash(graph)

	// Estimate size
	size := pc.estimateGraphSize(graph)

	item := &CacheItem{
		Key:         key,
		Graph:       graph,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 1,
		Size:        size,
		Hash:        hash,
	}

	pc.mutex.Lock()
	defer pc.mutex.Unlock()

	// Check if we need to evict items
	if len(pc.items) >= pc.config.MaxSize {
		if err := pc.evictItems(); err != nil {
			return fmt.Errorf("failed to evict items: %w", err)
		}
	}

	pc.items[key] = item
	if pc.config.EnableLRU {
		pc.access[key] = time.Now()
	}

	// Update metrics
	pc.metrics.mutex.Lock()
	pc.metrics.TotalSize += size
	pc.metrics.mutex.Unlock()

	// Persist to disk
	return pc.saveToDisk(key, item)
}

// GetAST retrieves a cached AST by file path
func (pc *PersistentCache) GetAST(filePath string) *types.AST {
	key := "ast:" + filePath

	pc.mutex.RLock()
	item, exists := pc.items[key]
	pc.mutex.RUnlock()

	if !exists {
		pc.recordMiss()
		return nil
	}

	// Check TTL
	if pc.config.TTL > 0 && time.Since(item.CreatedAt) > pc.config.TTL {
		pc.mutex.Lock()
		delete(pc.items, key)
		delete(pc.access, key)
		pc.mutex.Unlock()
		pc.recordMiss()
		return nil
	}

	// Extract AST from metadata
	if astData, exists := item.Metadata["ast"]; exists {
		if ast, ok := astData.(*types.AST); ok {
			pc.recordHit()
			return ast
		}
	}

	pc.recordMiss()
	return nil
}

// SetAST stores an AST in the cache
func (pc *PersistentCache) SetAST(filePath string, ast *types.AST) error {
	if ast == nil {
		return fmt.Errorf("cannot cache nil AST")
	}

	key := "ast:" + filePath
	size := int64(len(ast.Content) + 1000) // Estimate AST overhead

	item := &CacheItem{
		Key:         key,
		Metadata:    map[string]interface{}{"ast": ast},
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 1,
		Size:        size,
		Hash:        pc.calculateASTHash(ast),
	}

	pc.mutex.Lock()
	defer pc.mutex.Unlock()

	// Check if we need to evict items
	if len(pc.items) >= pc.config.MaxSize {
		if err := pc.evictItems(); err != nil {
			return fmt.Errorf("failed to evict items: %w", err)
		}
	}

	pc.items[key] = item
	if pc.config.EnableLRU {
		pc.access[key] = time.Now()
	}

	// Update metrics
	pc.metrics.mutex.Lock()
	pc.metrics.TotalSize += size
	pc.metrics.mutex.Unlock()

	return nil // ASTs are not persisted to disk by default (too much data)
}

// Clear removes all items from the cache
func (pc *PersistentCache) Clear() {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()

	pc.items = make(map[string]*CacheItem)
	pc.access = make(map[string]time.Time)

	pc.metrics.mutex.Lock()
	pc.metrics.TotalSize = 0
	pc.metrics.mutex.Unlock()

	// Clear disk cache
	pc.clearDiskCache()
}

// GetMetrics returns current cache metrics
func (pc *PersistentCache) GetMetrics() *CacheMetrics {
	pc.metrics.mutex.RLock()
	defer pc.metrics.mutex.RUnlock()

	// Calculate hit rate
	total := pc.metrics.Hits + pc.metrics.Misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(pc.metrics.Hits) / float64(total)
	}

	return &CacheMetrics{
		Hits:        pc.metrics.Hits,
		Misses:      pc.metrics.Misses,
		Evictions:   pc.metrics.Evictions,
		TotalSize:   pc.metrics.TotalSize,
		HitRate:     hitRate,
		LastCleanup: pc.metrics.LastCleanup,
	}
}

// Close gracefully shuts down the cache
func (pc *PersistentCache) Close() error {
	// Save current state to disk
	return pc.saveIndexToDisk()
}

// Private methods

func (pc *PersistentCache) recordHit() {
	if !pc.config.EnableMetrics {
		return
	}

	pc.metrics.mutex.Lock()
	pc.metrics.Hits++
	pc.metrics.mutex.Unlock()
}

func (pc *PersistentCache) recordMiss() {
	if !pc.config.EnableMetrics {
		return
	}

	pc.metrics.mutex.Lock()
	pc.metrics.Misses++
	pc.metrics.mutex.Unlock()
}

func (pc *PersistentCache) evictItems() error {
	// Evict items based on strategy
	if pc.config.EnableLRU {
		return pc.evictLRU()
	}

	// Simple eviction: remove oldest items
	return pc.evictOldest()
}

func (pc *PersistentCache) evictLRU() error {
	if len(pc.items) == 0 {
		return nil
	}

	// Find least recently used item
	var oldestKey string
	var oldestTime time.Time = time.Now()

	for key, accessTime := range pc.access {
		if accessTime.Before(oldestTime) {
			oldestTime = accessTime
			oldestKey = key
		}
	}

	if oldestKey != "" {
		if item, exists := pc.items[oldestKey]; exists {
			pc.metrics.mutex.Lock()
			pc.metrics.TotalSize -= item.Size
			pc.metrics.Evictions++
			pc.metrics.mutex.Unlock()
		}

		delete(pc.items, oldestKey)
		delete(pc.access, oldestKey)

		// Remove from disk
		pc.removeFromDisk(oldestKey)
	}

	return nil
}

func (pc *PersistentCache) evictOldest() error {
	if len(pc.items) == 0 {
		return nil
	}

	// Find oldest item by creation time
	var oldestKey string
	var oldestTime time.Time = time.Now()

	for key, item := range pc.items {
		if item.CreatedAt.Before(oldestTime) {
			oldestTime = item.CreatedAt
			oldestKey = key
		}
	}

	if oldestKey != "" {
		if item, exists := pc.items[oldestKey]; exists {
			pc.metrics.mutex.Lock()
			pc.metrics.TotalSize -= item.Size
			pc.metrics.Evictions++
			pc.metrics.mutex.Unlock()
		}

		delete(pc.items, oldestKey)
		delete(pc.access, oldestKey)

		// Remove from disk
		pc.removeFromDisk(oldestKey)
	}

	return nil
}

func (pc *PersistentCache) calculateGraphHash(graph *types.CodeGraph) string {
	// Simple hash based on graph metadata
	data := fmt.Sprintf("%d-%d-%s",
		len(graph.Files),
		len(graph.Symbols),
		graph.Metadata.Generated.Format(time.RFC3339))

	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (pc *PersistentCache) calculateASTHash(ast *types.AST) string {
	data := fmt.Sprintf("%s-%s-%d", ast.FilePath, ast.Version, len(ast.Content))
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (pc *PersistentCache) estimateGraphSize(graph *types.CodeGraph) int64 {
	// Rough estimation of graph size in memory
	size := int64(0)

	// Estimate file nodes
	size += int64(len(graph.Files)) * 1000 // ~1KB per file node

	// Estimate symbols
	size += int64(len(graph.Symbols)) * 500 // ~500B per symbol

	// Estimate nodes and edges
	size += int64(len(graph.Nodes)) * 300 // ~300B per node
	size += int64(len(graph.Edges)) * 200 // ~200B per edge

	return size
}

func (pc *PersistentCache) startCleanupWorker() {
	ticker := time.NewTicker(time.Hour) // Cleanup every hour
	defer ticker.Stop()

	for range ticker.C {
		pc.cleanup()
	}
}

func (pc *PersistentCache) cleanup() {
	if pc.config.TTL <= 0 {
		return
	}

	pc.mutex.Lock()
	defer pc.mutex.Unlock()

	expired := make([]string, 0)
	cutoff := time.Now().Add(-pc.config.TTL)

	for key, item := range pc.items {
		if item.CreatedAt.Before(cutoff) {
			expired = append(expired, key)
		}
	}

	// Remove expired items
	for _, key := range expired {
		if item, exists := pc.items[key]; exists {
			pc.metrics.mutex.Lock()
			pc.metrics.TotalSize -= item.Size
			pc.metrics.Evictions++
			pc.metrics.mutex.Unlock()
		}

		delete(pc.items, key)
		delete(pc.access, key)
		pc.removeFromDisk(key)
	}

	pc.metrics.mutex.Lock()
	pc.metrics.LastCleanup = time.Now()
	pc.metrics.mutex.Unlock()
}

// Disk persistence methods

func (pc *PersistentCache) loadFromDisk() error {
	indexPath := filepath.Join(pc.config.Directory, "index.gob")

	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return nil // No existing cache
	}

	file, err := os.Open(indexPath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)

	var items map[string]*CacheItem
	if err := decoder.Decode(&items); err != nil {
		return err
	}

	// Load individual cache files
	for key, item := range items {
		// Only load graphs, not ASTs
		if item.Graph != nil {
			itemPath := pc.getCacheFilePath(key)
			if err := pc.loadItemFromDisk(key, itemPath); err == nil {
				pc.items[key] = item
				if pc.config.EnableLRU {
					pc.access[key] = item.AccessedAt
				}
			}
		}
	}

	return nil
}

func (pc *PersistentCache) saveToDisk(key string, item *CacheItem) error {
	// Only save graphs to disk, not ASTs
	if item.Graph == nil {
		return nil
	}

	itemPath := pc.getCacheFilePath(key)

	file, err := os.Create(itemPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(item)
}

func (pc *PersistentCache) loadItemFromDisk(key, itemPath string) error {
	file, err := os.Open(itemPath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)

	var item CacheItem
	if err := decoder.Decode(&item); err != nil {
		return err
	}

	pc.items[key] = &item
	return nil
}

func (pc *PersistentCache) removeFromDisk(key string) {
	itemPath := pc.getCacheFilePath(key)
	os.Remove(itemPath) // Ignore errors
}

func (pc *PersistentCache) saveIndexToDisk() error {
	indexPath := filepath.Join(pc.config.Directory, "index.gob")

	file, err := os.Create(indexPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)

	// Only save items that have graphs (not ASTs)
	persistentItems := make(map[string]*CacheItem)
	for key, item := range pc.items {
		if item.Graph != nil {
			persistentItems[key] = item
		}
	}

	return encoder.Encode(persistentItems)
}

func (pc *PersistentCache) clearDiskCache() {
	// Remove all cache files
	os.RemoveAll(pc.config.Directory)
	os.MkdirAll(pc.config.Directory, 0755)
}

func (pc *PersistentCache) getCacheFilePath(key string) string {
	// Create safe filename from key
	hash := md5.Sum([]byte(key))
	filename := hex.EncodeToString(hash[:]) + ".gob"
	return filepath.Join(pc.config.Directory, filename)
}
