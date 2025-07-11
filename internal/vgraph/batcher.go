package vgraph

import (
	"sync"
	"time"
)

// ChangeBatcher manages the batching of changes for efficient processing
type ChangeBatcher struct {
	config   *VGEConfig
	batches  map[string]*BatchInfo
	mutex    sync.RWMutex
	timers   map[string]*time.Timer
	callback func([]ChangeSet) error
}

// BatchInfo holds information about a batch
type BatchInfo struct {
	ID        string      `json:"id"`
	Changes   []ChangeSet `json:"changes"`
	StartTime time.Time   `json:"start_time"`
	Size      int         `json:"size"`
	Priority  int         `json:"priority"`
}

// BatchStrategy represents different batching strategies
type BatchStrategy string

const (
	BatchStrategySize     BatchStrategy = "size"      // Batch by size
	BatchStrategyTime     BatchStrategy = "time"      // Batch by time
	BatchStrategyPriority BatchStrategy = "priority"  // Batch by priority
	BatchStrategyAdaptive BatchStrategy = "adaptive"  // Adaptive batching
)

// NewChangeBatcher creates a new change batcher
func NewChangeBatcher(config *VGEConfig) *ChangeBatcher {
	return &ChangeBatcher{
		config:  config,
		batches: make(map[string]*BatchInfo),
		timers:  make(map[string]*time.Timer),
	}
}

// SetCallback sets the callback function for batch processing
func (cb *ChangeBatcher) SetCallback(callback func([]ChangeSet) error) {
	cb.callback = callback
}

// AddChange adds a change to the appropriate batch
func (cb *ChangeBatcher) AddChange(change ChangeSet) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	batchKey := cb.getBatchKey(change)
	
	// Get or create batch
	batch, exists := cb.batches[batchKey]
	if !exists {
		batch = &BatchInfo{
			ID:        batchKey,
			Changes:   make([]ChangeSet, 0),
			StartTime: time.Now(),
			Size:      0,
			Priority:  cb.calculatePriority(change),
		}
		cb.batches[batchKey] = batch
		
		// Start timer for this batch
		cb.startBatchTimer(batchKey)
	}

	// Add change to batch
	batch.Changes = append(batch.Changes, change)
	batch.Size++

	// Check if batch should be processed immediately
	if cb.shouldProcessBatch(batch) {
		return cb.processBatch(batchKey)
	}

	return nil
}

// ProcessAllBatches processes all pending batches
func (cb *ChangeBatcher) ProcessAllBatches() error {
	cb.mutex.Lock()
	batchKeys := make([]string, 0, len(cb.batches))
	for key := range cb.batches {
		batchKeys = append(batchKeys, key)
	}
	cb.mutex.Unlock()

	for _, key := range batchKeys {
		err := cb.processBatch(key)
		if err != nil {
			return err
		}
	}

	return nil
}

// getBatchKey determines which batch a change belongs to
func (cb *ChangeBatcher) getBatchKey(change ChangeSet) string {
	// For now, batch by file path
	// More sophisticated strategies could be implemented
	return change.FilePath
}

// calculatePriority calculates the priority of a change
func (cb *ChangeBatcher) calculatePriority(change ChangeSet) int {
	switch change.Type {
	case ChangeTypeFileDelete:
		return 1 // Highest priority
	case ChangeTypeSymbolDel:
		return 2
	case ChangeTypeFileAdd:
		return 3
	case ChangeTypeSymbolAdd:
		return 4
	case ChangeTypeFileModify:
		return 5
	case ChangeTypeSymbolMod:
		return 6 // Lowest priority
	default:
		return 10
	}
}

// shouldProcessBatch determines if a batch should be processed immediately
func (cb *ChangeBatcher) shouldProcessBatch(batch *BatchInfo) bool {
	// Process if batch size exceeds threshold
	if batch.Size >= cb.config.BatchThreshold {
		return true
	}

	// Process if batch is high priority and has been waiting
	if batch.Priority <= 2 && time.Since(batch.StartTime) > cb.config.BatchTimeout/2 {
		return true
	}

	return false
}

// processBatch processes a specific batch
func (cb *ChangeBatcher) processBatch(batchKey string) error {
	cb.mutex.Lock()
	batch, exists := cb.batches[batchKey]
	if !exists {
		cb.mutex.Unlock()
		return nil
	}

	// Remove batch from pending batches
	delete(cb.batches, batchKey)
	
	// Cancel timer if it exists
	if timer, exists := cb.timers[batchKey]; exists {
		timer.Stop()
		delete(cb.timers, batchKey)
	}
	cb.mutex.Unlock()

	// Process the batch
	if cb.callback != nil {
		return cb.callback(batch.Changes)
	}

	return nil
}

// startBatchTimer starts a timer for a batch
func (cb *ChangeBatcher) startBatchTimer(batchKey string) {
	timer := time.AfterFunc(cb.config.BatchTimeout, func() {
		cb.processBatch(batchKey)
	})
	cb.timers[batchKey] = timer
}

// GetBatchInfo returns information about all current batches
func (cb *ChangeBatcher) GetBatchInfo() map[string]*BatchInfo {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	info := make(map[string]*BatchInfo)
	for key, batch := range cb.batches {
		// Create a copy to avoid race conditions
		batchCopy := &BatchInfo{
			ID:        batch.ID,
			Changes:   make([]ChangeSet, len(batch.Changes)),
			StartTime: batch.StartTime,
			Size:      batch.Size,
			Priority:  batch.Priority,
		}
		copy(batchCopy.Changes, batch.Changes)
		info[key] = batchCopy
	}

	return info
}

// ClearBatches clears all pending batches
func (cb *ChangeBatcher) ClearBatches() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// Stop all timers
	for _, timer := range cb.timers {
		timer.Stop()
	}

	// Clear all data structures
	cb.batches = make(map[string]*BatchInfo)
	cb.timers = make(map[string]*time.Timer)
}