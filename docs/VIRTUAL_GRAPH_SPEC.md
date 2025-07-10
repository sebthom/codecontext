# Virtual Graph Engine Specification

**Version:** 2.0  
**Component:** Virtual Graph Engine  
**Status:** Design Complete, Implementation in Progress

## Overview

The Virtual Graph Engine is the core innovation of CodeContext v2.0, implementing a Virtual DOM-inspired architecture for efficient incremental updates. This component achieves O(changes) complexity instead of O(repository_size) for update operations.

## Architecture Pattern

### Virtual DOM Inspiration
The Virtual Graph Engine borrows concepts from React's Virtual DOM:

1. **Shadow Graph**: Virtual representation of the code graph
2. **Actual Graph**: Committed state used for output generation
3. **Diffing**: Efficient computation of changes between states
4. **Reconciliation**: Minimal application of changes to actual state

### Core Components

```go
type VirtualGraphEngine struct {
    // State Management
    shadow      *CodeGraph       // Virtual representation
    actual      *CodeGraph       // Committed state
    pending     []ChangeSet      // Batched changes
    
    // Processing Components  
    differ      *ASTDiffer       // AST-level diffing
    reconciler  *Reconciler      // Change reconciliation
    patcher     *PatchManager    // Patch application
    
    // Configuration
    batchThreshold int            // Changes before reconciliation
    batchTimeout   time.Duration  // Max time before reconciliation
    maxShadowSize  int           // Memory limit for shadow graph
}
```

## Change Detection Flow

### 1. File Change Detection
```
File System Event → AST Parser → New AST → Shadow Graph Update
```

### 2. AST Diffing
```go
type ASTDiff struct {
    FileId            string
    FromVersion       string  
    ToVersion         string
    Additions         []*ASTNode
    Deletions         []*ASTNode
    Modifications     []*ASTModification
    StructuralChanges bool
    ImpactRadius      *ImpactAnalysis
}
```

### 3. Impact Analysis
The engine computes the impact radius of changes:

```go
type ImpactAnalysis struct {
    AffectedNodes     []NodeId    // Direct impact
    AffectedFiles     []string    // File-level impact
    PropagationDepth  int         // How far changes propagate
    Severity          string      // "low", "medium", "high"
    EstimatedTokens   int         // Token count impact
    Dependencies      []NodeId    // Upstream dependencies
    Dependents        []NodeId    // Downstream dependents
}
```

## Batching Strategy

### Change Accumulation
Changes are batched based on:
- **Count Threshold**: Default 5 changes
- **Time Threshold**: Default 500ms timeout
- **Memory Threshold**: Shadow graph size limit

### Batching Logic
```go
func (vge *VirtualGraphEngine) ShouldBatch(change Change) bool {
    // Check if we should batch this change or reconcile immediately
    pendingCount := len(vge.pending)
    lastChange := vge.getLastChangeTime()
    
    return pendingCount < vge.batchThreshold && 
           time.Since(lastChange) < vge.batchTimeout &&
           vge.getShadowMemoryUsage() < vge.maxShadowSize
}
```

## Reconciliation Process

### 1. Plan Generation
```go
type ReconciliationPlan struct {
    Id               string
    Patches          []GraphPatch
    UpdateOrder      []NodeId           // Dependency-sorted updates
    Invalidations    []CacheInvalidation
    EstimatedDuration time.Duration
    TokenImpact      TokenDelta
    Reversible       bool
}
```

### 2. Patch Types
```go
type GraphPatch struct {
    Type         string      // "add", "remove", "modify", "reorder"
    TargetNode   NodeId
    Changes      []PropertyChange
    Dependencies []NodeId
    Metadata     map[string]interface{}
}
```

### 3. Dependency Resolution
The reconciler ensures patches are applied in dependency order:
- Symbol definitions before references
- Parent nodes before children
- Imports before usage

## Performance Optimizations

### 1. Structural Hashing
```go
type StructuralHash struct {
    NodeType     string
    ChildHashes  []string
    PropertyHash string
    Combined     string    // SHA-256 of above
}
```

### 2. Lazy Evaluation
- Shadow graph nodes created on-demand
- Diff computation only for changed subtrees
- Cache-aware reconciliation planning

### 3. Memory Management
```go
type MemoryManager struct {
    shadowSizeLimit   int
    gcThreshold      float64    // Trigger GC at 80% usage
    compressionLevel int        // 0-9, higher = more CPU, less memory
}
```

## Diff Algorithms

### 1. Tree Diffing
Based on Myers' algorithm with optimizations for ASTs:

```go
type TreeDiffer struct {
    algorithm    string      // "myers", "patience", "histogram"
    maxDepth     int         // Limit recursion depth
    useHashing   bool        // Use structural hashes
    memoization  bool        // Cache diff results
}
```

### 2. Symbol-Level Diffing
```go
type SymbolDiff struct {
    Added     map[SymbolId]*Symbol
    Removed   map[SymbolId]*Symbol  
    Modified  map[SymbolId]*SymbolModification
    Renamed   map[SymbolId]*RenameInfo
    Moved     map[SymbolId]*MoveInfo
}
```

### 3. Semantic Diffing
Beyond structural changes, the engine detects semantic changes:
- Function signature modifications
- Type system changes
- Access modifier changes
- Documentation updates

## State Management

### 1. Shadow Graph Lifecycle
```
Create → Update → Batch → Reconcile → Commit → Cleanup
```

### 2. State Synchronization
```go
func (vge *VirtualGraphEngine) Sync() error {
    // Ensure shadow and actual graphs are synchronized
    diff := vge.computeGraphDiff(vge.shadow, vge.actual)
    if !diff.IsEmpty() {
        return vge.forceReconciliation()
    }
    return nil
}
```

### 3. Checkpoint Management
```go
type GraphCheckpoint struct {
    Id        string
    Graph     *CodeGraph
    Timestamp time.Time
    Changes   []ChangeSet
    Metadata  map[string]interface{}
}
```

## Error Handling and Recovery

### 1. Corruption Detection
```go
func (vge *VirtualGraphEngine) ValidateIntegrity() []ValidationError {
    var errors []ValidationError
    
    // Check shadow-actual consistency
    // Validate graph structure
    // Check for orphaned nodes
    // Verify change history
    
    return errors
}
```

### 2. Recovery Strategies
- **Soft Recovery**: Rebuild shadow from actual
- **Hard Recovery**: Full regeneration from source
- **Partial Recovery**: Quarantine corrupted nodes

### 3. Rollback Support
```go
func (vge *VirtualGraphEngine) Rollback(checkpoint *GraphCheckpoint) error {
    // Restore to previous known good state
    vge.actual = checkpoint.Graph.Clone()
    vge.shadow = vge.createShadowFrom(vge.actual)
    vge.clearPendingChanges()
    return nil
}
```

## Configuration

### Virtual Graph Settings
```yaml
virtual_graph:
  enabled: true
  batch_threshold: 5
  batch_timeout: 500ms
  max_shadow_memory: 100MB
  diff_algorithm: myers
  use_structural_hashing: true
  memoize_diffs: true
  compression_level: 6
  
incremental_update:
  enabled: true
  min_change_size: 10
  max_patch_history: 1000
  compact_patches: true
  async_reconciliation: false
```

### Performance Tuning
```yaml
performance:
  parallel_diffing: true
  max_diff_workers: 4
  chunk_size: 1000
  memory_pressure_threshold: 0.8
  gc_frequency: "5m"
```

## Metrics and Monitoring

### Key Metrics
```go
type VirtualGraphMetrics struct {
    // Performance
    DiffComputeTime      time.Duration
    ReconciliationTime   time.Duration
    BatchSize           int
    CacheHitRate        float64
    
    // Memory
    ShadowGraphSize     int
    ActualGraphSize     int
    PendingChangesSize  int
    
    // Operations
    TotalPatches        int
    SuccessfulPatches   int
    FailedPatches       int
    RollbacksExecuted   int
}
```

### Health Checks
```bash
# CLI commands for monitoring
codecontext graph --show-shadow
codecontext graph --reconcile --dry-run
codecontext graph --validate
codecontext graph --stats
codecontext graph --memory-usage
```

## Testing Strategy

### Unit Tests
- Individual component testing
- Mock implementations for dependencies
- Property-based testing for diff algorithms

### Integration Tests
- End-to-end workflow testing
- Performance benchmarking
- Memory leak detection

### Stress Tests
```go
func TestVirtualGraphStress(t *testing.T) {
    // Test with 10k file changes
    // Measure memory usage over time
    // Verify no performance degradation
}
```

## Future Enhancements

### Phase 2
- Distributed virtual graphs for team collaboration
- Predictive change impact analysis
- Machine learning-based optimization

### Phase 3
- Real-time collaborative editing support
- Advanced conflict resolution
- Cross-repository virtual graphs

## Implementation Checklist

- [x] Core interfaces defined
- [x] Type system complete
- [ ] AST differ implementation
- [ ] Shadow graph management
- [ ] Reconciliation engine
- [ ] Patch application system
- [ ] Memory management
- [ ] Error recovery
- [ ] Performance monitoring
- [ ] Configuration system

---

*This specification serves as the definitive guide for Virtual Graph Engine implementation and should be updated with any architectural changes.*