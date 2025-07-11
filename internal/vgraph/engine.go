package vgraph

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

// VirtualGraphEngine implements the Virtual Graph pattern for efficient incremental updates
type VirtualGraphEngine struct {
	shadow         *types.CodeGraph         // Virtual representation
	actual         *types.CodeGraph         // Committed state
	pendingChanges []ChangeSet              // Batched changes
	differ         *ASTDiffer               // Diff computation
	reconciler     *Reconciler              // Change application
	batcher        *ChangeBatcher           // Change batching logic
	config         *VGEConfig               // Configuration
	metrics        *VGEMetrics              // Performance metrics
	mu             sync.RWMutex             // Thread safety
}

// VGEConfig holds configuration for the Virtual Graph Engine
type VGEConfig struct {
	BatchThreshold   int           `json:"batch_threshold"`   // Number of changes to batch
	BatchTimeout     time.Duration `json:"batch_timeout"`     // Max time to wait for batching
	MaxShadowMemory  int64         `json:"max_shadow_memory"` // Maximum shadow graph memory (bytes)
	DiffAlgorithm    string        `json:"diff_algorithm"`    // myers, patience, histogram
	EnableMetrics    bool          `json:"enable_metrics"`    // Enable performance metrics
	GCThreshold      float64       `json:"gc_threshold"`      // GC trigger threshold (0.0-1.0)
}

// VGEMetrics tracks performance metrics for the Virtual Graph Engine
type VGEMetrics struct {
	TotalChanges      int64         `json:"total_changes"`
	BatchesProcessed  int64         `json:"batches_processed"`
	AverageBatchSize  float64       `json:"average_batch_size"`
	DiffTime          time.Duration `json:"diff_time"`
	ReconcileTime     time.Duration `json:"reconcile_time"`
	CommitTime        time.Duration `json:"commit_time"`
	ShadowMemoryBytes int64         `json:"shadow_memory_bytes"`
	CacheHitRate      float64       `json:"cache_hit_rate"`
	LastUpdate        time.Time     `json:"last_update"`
}

// ChangeSet represents a set of changes to apply
type ChangeSet struct {
	ID        string                 `json:"id"`
	Type      ChangeType             `json:"type"`
	FilePath  string                 `json:"file_path"`
	Changes   []Change               `json:"changes"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// ChangeType represents the type of change
type ChangeType string

const (
	ChangeTypeFileAdd    ChangeType = "file_add"
	ChangeTypeFileModify ChangeType = "file_modify"
	ChangeTypeFileDelete ChangeType = "file_delete"
	ChangeTypeSymbolAdd  ChangeType = "symbol_add"
	ChangeTypeSymbolMod  ChangeType = "symbol_modify"
	ChangeTypeSymbolDel  ChangeType = "symbol_delete"
)

// Change represents a specific change to apply
type Change struct {
	Type     ChangeType             `json:"type"`
	Target   string                 `json:"target"`   // File path or symbol ID
	OldValue interface{}            `json:"old_value"`
	NewValue interface{}            `json:"new_value"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ReconciliationPlan represents a plan for applying changes
type ReconciliationPlan struct {
	Patches        []GraphPatch       `json:"patches"`
	UpdateOrder    []types.NodeId     `json:"update_order"`
	Invalidations  []CacheInvalidation `json:"invalidations"`
	EstimatedTime  time.Duration      `json:"estimated_time"`
	TokenImpact    *TokenDelta        `json:"token_impact"`
	Dependencies   map[string][]string `json:"dependencies"`
}

// GraphPatch represents a change to apply to the graph
type GraphPatch struct {
	ID           string                 `json:"id"`
	Type         PatchType              `json:"type"`
	TargetNode   types.NodeId           `json:"target_node"`
	Changes      []PropertyChange       `json:"changes"`
	Dependencies []types.NodeId         `json:"dependencies"`
	Priority     int                    `json:"priority"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// PatchType represents the type of patch
type PatchType string

const (
	PatchTypeAdd    PatchType = "add"
	PatchTypeRemove PatchType = "remove"
	PatchTypeModify PatchType = "modify"
	PatchTypeReorder PatchType = "reorder"
)

// PropertyChange represents a change to a property
type PropertyChange struct {
	Property string      `json:"property"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

// CacheInvalidation represents a cache invalidation
type CacheInvalidation struct {
	Type    string   `json:"type"`
	Keys    []string `json:"keys"`
	Reason  string   `json:"reason"`
}

// TokenDelta represents the impact on token count
type TokenDelta struct {
	Before int `json:"before"`
	After  int `json:"after"`
	Delta  int `json:"delta"`
}

// NewVirtualGraphEngine creates a new Virtual Graph Engine
func NewVirtualGraphEngine(config *VGEConfig) *VirtualGraphEngine {
	if config == nil {
		config = DefaultVGEConfig()
	}

	vge := &VirtualGraphEngine{
		shadow:         &types.CodeGraph{},
		actual:         &types.CodeGraph{},
		pendingChanges: make([]ChangeSet, 0),
		config:         config,
		metrics:        &VGEMetrics{LastUpdate: time.Now()},
	}

	vge.differ = NewASTDiffer()
	vge.reconciler = NewReconciler(vge)
	vge.batcher = NewChangeBatcher(config)

	return vge
}

// DefaultVGEConfig returns default configuration for VGE
func DefaultVGEConfig() *VGEConfig {
	return &VGEConfig{
		BatchThreshold:  5,
		BatchTimeout:    500 * time.Millisecond,
		MaxShadowMemory: 100 * 1024 * 1024, // 100MB
		DiffAlgorithm:   "myers",
		EnableMetrics:   true,
		GCThreshold:     0.8,
	}
}

// Initialize initializes the Virtual Graph Engine with an actual graph
func (vge *VirtualGraphEngine) Initialize(actualGraph *types.CodeGraph) error {
	vge.mu.Lock()
	defer vge.mu.Unlock()

	// Deep copy the actual graph to create shadow
	shadowCopy, err := vge.deepCopyGraph(actualGraph)
	if err != nil {
		return fmt.Errorf("failed to create shadow graph: %w", err)
	}

	vge.actual = actualGraph
	vge.shadow = shadowCopy
	vge.pendingChanges = make([]ChangeSet, 0)

	if vge.config.EnableMetrics {
		vge.updateMetrics()
	}

	return nil
}

// QueueChange adds a change to the pending changes queue
func (vge *VirtualGraphEngine) QueueChange(change ChangeSet) error {
	vge.mu.Lock()
	defer vge.mu.Unlock()

	vge.pendingChanges = append(vge.pendingChanges, change)
	vge.metrics.TotalChanges++

	// Check if we should trigger a batch
	if vge.shouldTriggerBatch() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vge.ProcessPendingChanges(ctx)
		}()
	}

	return nil
}

// ProcessPendingChanges processes all pending changes
func (vge *VirtualGraphEngine) ProcessPendingChanges(ctx context.Context) error {
	vge.mu.Lock()
	if len(vge.pendingChanges) == 0 {
		vge.mu.Unlock()
		return nil
	}

	changes := make([]ChangeSet, len(vge.pendingChanges))
	copy(changes, vge.pendingChanges)
	vge.pendingChanges = make([]ChangeSet, 0)
	vge.mu.Unlock()

	start := time.Now()

	// Apply changes to shadow graph
	err := vge.applyChangesToShadow(changes)
	if err != nil {
		return fmt.Errorf("failed to apply changes to shadow: %w", err)
	}

	// Generate reconciliation plan
	plan, err := vge.reconciler.GeneratePlan(ctx, vge.actual, vge.shadow)
	if err != nil {
		return fmt.Errorf("failed to generate reconciliation plan: %w", err)
	}

	// Apply plan to actual graph
	err = vge.reconciler.ApplyPlan(ctx, plan, vge.actual)
	if err != nil {
		return fmt.Errorf("failed to apply reconciliation plan: %w", err)
	}

	// Update metrics
	if vge.config.EnableMetrics {
		vge.metrics.BatchesProcessed++
		vge.metrics.AverageBatchSize = float64(vge.metrics.TotalChanges) / float64(vge.metrics.BatchesProcessed)
		vge.metrics.CommitTime += time.Since(start)
		vge.updateMetrics()
	}

	return nil
}

// GetShadowGraph returns a read-only copy of the shadow graph
func (vge *VirtualGraphEngine) GetShadowGraph() *types.CodeGraph {
	vge.mu.RLock()
	defer vge.mu.RUnlock()

	// Return a deep copy to prevent external modifications
	copy, _ := vge.deepCopyGraph(vge.shadow)
	return copy
}

// GetActualGraph returns a read-only copy of the actual graph
func (vge *VirtualGraphEngine) GetActualGraph() *types.CodeGraph {
	vge.mu.RLock()
	defer vge.mu.RUnlock()

	// Return a deep copy to prevent external modifications
	copy, _ := vge.deepCopyGraph(vge.actual)
	return copy
}

// GetMetrics returns current performance metrics
func (vge *VirtualGraphEngine) GetMetrics() *VGEMetrics {
	vge.mu.RLock()
	defer vge.mu.RUnlock()

	// Return a copy of metrics
	metricsCopy := *vge.metrics
	return &metricsCopy
}

// Reset resets the Virtual Graph Engine to initial state
func (vge *VirtualGraphEngine) Reset() error {
	vge.mu.Lock()
	defer vge.mu.Unlock()

	vge.shadow = &types.CodeGraph{}
	vge.actual = &types.CodeGraph{}
	vge.pendingChanges = make([]ChangeSet, 0)
	vge.metrics = &VGEMetrics{LastUpdate: time.Now()}

	return nil
}

// shouldTriggerBatch determines if a batch should be triggered
func (vge *VirtualGraphEngine) shouldTriggerBatch() bool {
	return len(vge.pendingChanges) >= vge.config.BatchThreshold
}

// applyChangesToShadow applies a set of changes to the shadow graph
func (vge *VirtualGraphEngine) applyChangesToShadow(changes []ChangeSet) error {
	for _, changeSet := range changes {
		for _, change := range changeSet.Changes {
			err := vge.applySingleChange(change)
			if err != nil {
				return fmt.Errorf("failed to apply change %s: %w", change.Target, err)
			}
		}
	}
	return nil
}

// applySingleChange applies a single change to the shadow graph
func (vge *VirtualGraphEngine) applySingleChange(change Change) error {
	switch change.Type {
	case ChangeTypeFileAdd:
		return vge.addFileToShadow(change)
	case ChangeTypeFileModify:
		return vge.modifyFileInShadow(change)
	case ChangeTypeFileDelete:
		return vge.deleteFileFromShadow(change)
	case ChangeTypeSymbolAdd:
		return vge.addSymbolToShadow(change)
	case ChangeTypeSymbolMod:
		return vge.modifySymbolInShadow(change)
	case ChangeTypeSymbolDel:
		return vge.deleteSymbolFromShadow(change)
	default:
		return fmt.Errorf("unknown change type: %s", change.Type)
	}
}

// addFileToShadow adds a file to the shadow graph
func (vge *VirtualGraphEngine) addFileToShadow(change Change) error {
	fileNode, ok := change.NewValue.(*types.FileNode)
	if !ok {
		return fmt.Errorf("invalid file node type for add operation")
	}

	if vge.shadow.Files == nil {
		vge.shadow.Files = make(map[string]*types.FileNode)
	}

	vge.shadow.Files[change.Target] = fileNode
	return nil
}

// modifyFileInShadow modifies a file in the shadow graph
func (vge *VirtualGraphEngine) modifyFileInShadow(change Change) error {
	fileNode, ok := change.NewValue.(*types.FileNode)
	if !ok {
		return fmt.Errorf("invalid file node type for modify operation")
	}

	if vge.shadow.Files == nil {
		vge.shadow.Files = make(map[string]*types.FileNode)
	}

	vge.shadow.Files[change.Target] = fileNode
	return nil
}

// deleteFileFromShadow deletes a file from the shadow graph
func (vge *VirtualGraphEngine) deleteFileFromShadow(change Change) error {
	if vge.shadow.Files != nil {
		delete(vge.shadow.Files, change.Target)
	}
	return nil
}

// addSymbolToShadow adds a symbol to the shadow graph
func (vge *VirtualGraphEngine) addSymbolToShadow(change Change) error {
	symbol, ok := change.NewValue.(*types.Symbol)
	if !ok {
		return fmt.Errorf("invalid symbol type for add operation")
	}

	if vge.shadow.Symbols == nil {
		vge.shadow.Symbols = make(map[types.SymbolId]*types.Symbol)
	}

	vge.shadow.Symbols[symbol.Id] = symbol
	return nil
}

// modifySymbolInShadow modifies a symbol in the shadow graph
func (vge *VirtualGraphEngine) modifySymbolInShadow(change Change) error {
	symbol, ok := change.NewValue.(*types.Symbol)
	if !ok {
		return fmt.Errorf("invalid symbol type for modify operation")
	}

	if vge.shadow.Symbols == nil {
		vge.shadow.Symbols = make(map[types.SymbolId]*types.Symbol)
	}

	vge.shadow.Symbols[symbol.Id] = symbol
	return nil
}

// deleteSymbolFromShadow deletes a symbol from the shadow graph
func (vge *VirtualGraphEngine) deleteSymbolFromShadow(change Change) error {
	symbolId := types.SymbolId(change.Target)
	if vge.shadow.Symbols != nil {
		delete(vge.shadow.Symbols, symbolId)
	}
	return nil
}

// deepCopyGraph creates a deep copy of a code graph
func (vge *VirtualGraphEngine) deepCopyGraph(graph *types.CodeGraph) (*types.CodeGraph, error) {
	// For now, this is a basic implementation
	// In production, we'd use a more efficient serialization method
	newGraph := &types.CodeGraph{
		Nodes:   make(map[types.NodeId]*types.GraphNode),
		Edges:   make(map[types.EdgeId]*types.GraphEdge),
		Files:   make(map[string]*types.FileNode),
		Symbols: make(map[types.SymbolId]*types.Symbol),
	}

	// Copy metadata if it exists
	if graph.Metadata != nil {
		newGraph.Metadata = &types.GraphMetadata{}
		*newGraph.Metadata = *graph.Metadata
	}

	// Copy nodes
	for id, node := range graph.Nodes {
		newNode := &types.GraphNode{}
		*newNode = *node
		newGraph.Nodes[id] = newNode
	}

	// Copy edges
	for id, edge := range graph.Edges {
		newEdge := &types.GraphEdge{}
		*newEdge = *edge
		newGraph.Edges[id] = newEdge
	}

	// Copy files
	for path, file := range graph.Files {
		newFile := &types.FileNode{}
		*newFile = *file
		newGraph.Files[path] = newFile
	}

	// Copy symbols
	for id, symbol := range graph.Symbols {
		newSymbol := &types.Symbol{}
		*newSymbol = *symbol
		newGraph.Symbols[id] = newSymbol
	}

	return newGraph, nil
}

// updateMetrics updates internal metrics
func (vge *VirtualGraphEngine) updateMetrics() {
	vge.metrics.LastUpdate = time.Now()
	
	// Calculate shadow memory usage (approximate)
	shadowMemory := int64(0)
	shadowMemory += int64(len(vge.shadow.Nodes) * 200)    // Approximate size per node
	shadowMemory += int64(len(vge.shadow.Edges) * 150)    // Approximate size per edge
	shadowMemory += int64(len(vge.shadow.Files) * 300)    // Approximate size per file
	shadowMemory += int64(len(vge.shadow.Symbols) * 250)  // Approximate size per symbol
	
	vge.metrics.ShadowMemoryBytes = shadowMemory
}