# Virtual Graph Engine (VGE)

The Virtual Graph Engine is a sophisticated system inspired by Virtual DOM concepts that enables efficient incremental updates to code graphs. It maintains a shadow representation of the graph and applies minimal patches to synchronize changes.

## Overview

The VGE implements a shadow/actual graph pattern where:

- **Shadow Graph**: Virtual representation where changes are applied immediately
- **Actual Graph**: Committed state that reflects the true graph structure
- **Reconciliation**: Process of computing minimal differences and applying patches
- **Batching**: Intelligent grouping of changes for optimal performance

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                  Virtual Graph Engine                       │
├─────────────────┬─────────────────┬─────────────────────────┤
│   Shadow Graph  │  Change Batcher │    AST Differ           │
│   - Virtual Rep │  - Batching     │    - Myers Algorithm    │
│   - Immediate   │  - Timing       │    - Impact Analysis    │
│     Updates     │  - Priority     │    - Symbol Tracking    │
├─────────────────┼─────────────────┼─────────────────────────┤
│   Actual Graph  │   Reconciler    │    Metrics & Config     │
│   - Committed   │  - Plan Gen     │    - Performance        │
│   - Stable      │  - Validation   │    - Memory Usage       │
│   - Production  │  - Rollback     │    - Cache Stats        │
└─────────────────┴─────────────────┴─────────────────────────┘
```

### Key Interfaces

#### VirtualGraphEngine
```go
type VirtualGraphEngine struct {
    shadow         *types.CodeGraph
    actual         *types.CodeGraph
    pendingChanges []ChangeSet
    differ         *ASTDiffer
    reconciler     *Reconciler
    batcher        *ChangeBatcher
    config         *VGEConfig
    metrics        *VGEMetrics
}
```

#### Core Operations
- `Initialize(actualGraph)`: Initialize with existing graph
- `QueueChange(change)`: Add change to pending queue
- `ProcessPendingChanges()`: Apply batched changes
- `GetShadowGraph()`: Read-only access to shadow
- `GetActualGraph()`: Read-only access to actual
- `GetMetrics()`: Performance metrics

## Features

### 🔄 Change Management

#### Change Types
| Type | Description | Priority |
|------|-------------|----------|
| `file_add` | New file added | 3 |
| `file_modify` | File content changed | 5 |
| `file_delete` | File removed | 1 (highest) |
| `symbol_add` | New symbol added | 4 |
| `symbol_modify` | Symbol changed | 6 (lowest) |
| `symbol_delete` | Symbol removed | 2 |

#### Change Processing
```go
// Queue a change
change := ChangeSet{
    ID:       "change-1",
    Type:     ChangeTypeFileModify,
    FilePath: "src/user.ts",
    Changes: []Change{
        {
            Type:     ChangeTypeFileModify,
            Target:   "src/user.ts",
            OldValue: oldFileNode,
            NewValue: newFileNode,
        },
    },
    Timestamp: time.Now(),
}

err := vge.QueueChange(change)
```

### 📦 Intelligent Batching

#### Batching Strategies
- **Size-based**: Batch when threshold reached (default: 5 changes)
- **Time-based**: Batch after timeout (default: 500ms)
- **Priority-based**: High priority changes trigger immediate processing
- **Adaptive**: Dynamic batching based on system load

#### Batch Configuration
```go
config := &VGEConfig{
    BatchThreshold:  5,                    // Changes before batching
    BatchTimeout:    500 * time.Millisecond, // Max wait time
    MaxShadowMemory: 100 * 1024 * 1024,    // 100MB memory limit
    DiffAlgorithm:   "myers",              // myers, patience, histogram
    EnableMetrics:   true,                 // Performance tracking
    GCThreshold:     0.8,                  // GC trigger threshold
}
```

### 🔍 AST Diffing

#### Diff Algorithms
1. **Myers Algorithm** (default): Classic LCS-based diffing
2. **Patience Algorithm**: Better for large files with many unique lines
3. **Histogram Algorithm**: Optimized for specific patterns

#### Impact Analysis
```go
type ImpactAnalysis struct {
    AffectedFiles   []string
    AffectedSymbols []types.SymbolId
    PropagationTree *PropagationNode
    RiskScore       float64
    Recommendations []string
}
```

#### Symbol Change Tracking
- **Added**: New symbols introduced
- **Removed**: Symbols deleted
- **Modified**: Symbols with changed properties
- **Renamed**: Symbols that moved/renamed (detected automatically)

### 🔧 Reconciliation

#### Reconciliation Process
1. **Plan Generation**: Compute minimal patches between shadow and actual
2. **Dependency Ordering**: Sort patches to respect dependencies
3. **Validation**: Pre-flight checks for consistency
4. **Application**: Apply patches in order
5. **Rollback**: Restore previous state if errors occur

#### Patch Types
```go
type GraphPatch struct {
    ID           string
    Type         PatchType  // add, remove, modify, reorder
    TargetNode   types.NodeId
    Changes      []PropertyChange
    Dependencies []types.NodeId
    Priority     int
}
```

#### Conflict Resolution
- **Abort**: Stop on first conflict
- **Force**: Override conflicts with latest changes
- **Merge**: Intelligent merging of conflicting changes

### 📊 Performance & Metrics

#### Performance Metrics
```go
type VGEMetrics struct {
    TotalChanges      int64
    BatchesProcessed  int64
    AverageBatchSize  float64
    DiffTime          time.Duration
    ReconcileTime     time.Duration
    CommitTime        time.Duration
    ShadowMemoryBytes int64
    CacheHitRate      float64
    LastUpdate        time.Time
}
```

#### Memory Management
- **Shadow Graph**: ~10% overhead of actual graph
- **Change Queue**: Bounded by batch thresholds
- **Cache**: LRU cache with configurable size
- **GC**: Automatic cleanup at configurable thresholds

## Usage Examples

### Basic Usage

```go
// Create and initialize VGE
config := DefaultVGEConfig()
vge := NewVirtualGraphEngine(config)

// Initialize with existing graph
err := vge.Initialize(actualGraph)
if err != nil {
    log.Fatal(err)
}

// Queue changes
change := ChangeSet{
    ID:       "file-update",
    Type:     ChangeTypeFileModify,
    FilePath: "src/user.ts",
    Changes:  []Change{ /* ... */ },
}

err = vge.QueueChange(change)
if err != nil {
    log.Fatal(err)
}

// Process changes (can be automatic via batching)
ctx := context.Background()
err = vge.ProcessPendingChanges(ctx)
if err != nil {
    log.Fatal(err)
}

// Get updated graph
updatedGraph := vge.GetActualGraph()
```

### Advanced Configuration

```go
config := &VGEConfig{
    BatchThreshold:  10,                   // Larger batches
    BatchTimeout:    1 * time.Second,      // Longer wait
    MaxShadowMemory: 200 * 1024 * 1024,    // 200MB limit
    DiffAlgorithm:   "patience",           // Different algorithm
    EnableMetrics:   true,                 // Track performance
    GCThreshold:     0.9,                  // More aggressive GC
}

vge := NewVirtualGraphEngine(config)
```

### Integration with File Watcher

```go
// In file watcher callback
func onFileChange(filePath string, newContent []byte) {
    // Parse the file to get new symbols
    newFileNode := parseFile(filePath, newContent)
    
    // Create change
    change := ChangeSet{
        ID:       fmt.Sprintf("file-change-%s", filePath),
        Type:     ChangeTypeFileModify,
        FilePath: filePath,
        Changes: []Change{
            {
                Type:     ChangeTypeFileModify,
                Target:   filePath,
                OldValue: currentFileNode,
                NewValue: newFileNode,
            },
        },
        Timestamp: time.Now(),
    }
    
    // Queue for processing
    vge.QueueChange(change)
}
```

### Metrics and Monitoring

```go
// Get performance metrics
metrics := vge.GetMetrics()

fmt.Printf("Total changes: %d\n", metrics.TotalChanges)
fmt.Printf("Batches processed: %d\n", metrics.BatchesProcessed)
fmt.Printf("Average batch size: %.2f\n", metrics.AverageBatchSize)
fmt.Printf("Shadow memory: %d bytes\n", metrics.ShadowMemoryBytes)
fmt.Printf("Cache hit rate: %.2f%%\n", metrics.CacheHitRate*100)

// Check if performance is degrading
if metrics.CommitTime > 1*time.Second {
    log.Warn("VGE commit time is high, consider tuning batch size")
}

if metrics.ShadowMemoryBytes > config.MaxShadowMemory {
    log.Warn("Shadow graph memory usage is high")
}
```

## Performance Characteristics

### Time Complexity
- **Change Queuing**: O(1)
- **Batch Processing**: O(changes) for batching, O(changes * log(nodes)) for reconciliation
- **Diff Computation**: O(n*m) where n,m are tree sizes (varies by algorithm)
- **Patch Application**: O(patches)

### Space Complexity
- **Shadow Graph**: O(nodes + edges) - approximately same as actual graph
- **Change Queue**: O(pending_changes)
- **Diff Cache**: O(cached_diffs) - bounded by LRU

### Scalability
- **Nodes**: Tested up to 100k+ nodes
- **Changes**: Handles 1000+ changes per batch efficiently
- **Memory**: Configurable limits with automatic GC
- **Concurrency**: Thread-safe with read-write locks

## Configuration

### VGE Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `BatchThreshold` | 5 | Changes to accumulate before processing |
| `BatchTimeout` | 500ms | Maximum wait time for batching |
| `MaxShadowMemory` | 100MB | Memory limit for shadow graph |
| `DiffAlgorithm` | myers | Algorithm for AST diffing |
| `EnableMetrics` | true | Performance metrics collection |
| `GCThreshold` | 0.8 | Memory threshold for garbage collection |

### Reconciler Configuration

| Option | Default | Description |
|--------|---------|-------------|
| `MaxConcurrency` | 4 | Concurrent patch operations |
| `ConflictResolution` | merge | How to handle conflicts |
| `DependencyOrdering` | true | Order patches by dependencies |
| `ValidationEnabled` | true | Pre/post validation |
| `RollbackEnabled` | true | Automatic rollback on errors |
| `MaxPatchSize` | 1000 | Maximum patches per plan |
| `BatchTimeout` | 30s | Timeout for batch operations |

## Testing

### Unit Tests

```bash
# Run VGE tests
go test ./internal/vgraph/... -v

# Run with coverage
go test ./internal/vgraph/... -cover

# Run benchmarks
go test ./internal/vgraph/... -bench=.
```

### Integration Tests

```go
func TestVGEIntegration(t *testing.T) {
    vge := NewVirtualGraphEngine(nil)
    graph := createLargeTestGraph(1000) // 1000 nodes
    
    // Initialize
    err := vge.Initialize(graph)
    require.NoError(t, err)
    
    // Apply many changes
    for i := 0; i < 100; i++ {
        change := createRandomChange()
        err := vge.QueueChange(change)
        require.NoError(t, err)
    }
    
    // Process and verify
    ctx := context.Background()
    err = vge.ProcessPendingChanges(ctx)
    require.NoError(t, err)
    
    // Verify graph consistency
    verifyGraphConsistency(t, vge.GetActualGraph())
}
```

### Performance Tests

```go
func BenchmarkVGELargeGraph(b *testing.B) {
    vge := NewVirtualGraphEngine(nil)
    graph := createLargeTestGraph(10000) // 10k nodes
    vge.Initialize(graph)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        change := createRandomChange()
        vge.QueueChange(change)
        
        if i%100 == 0 { // Process every 100 changes
            ctx := context.Background()
            vge.ProcessPendingChanges(ctx)
        }
    }
}
```

## Error Handling

### Common Errors

1. **Initialization Errors**
   - Invalid graph structure
   - Memory allocation failures
   - Configuration validation errors

2. **Change Processing Errors**
   - Invalid change format
   - Dependency conflicts
   - Memory limit exceeded

3. **Reconciliation Errors**
   - Patch application failures
   - Validation errors
   - Rollback failures

### Error Recovery

```go
// Automatic rollback on errors
err := vge.ProcessPendingChanges(ctx)
if err != nil {
    log.Printf("Processing failed, rollback triggered: %v", err)
    // VGE automatically restores previous state
    
    // Check metrics for details
    metrics := vge.GetMetrics()
    if metrics.PlansRolledBack > 0 {
        log.Printf("Rollbacks: %d", metrics.PlansRolledBack)
    }
}
```

## Best Practices

### 1. Batch Size Tuning
- **Small batches** (1-5 changes): Low latency, higher overhead
- **Medium batches** (5-20 changes): Good balance
- **Large batches** (20+ changes): Higher latency, better throughput

### 2. Memory Management
- Monitor shadow graph memory usage
- Set appropriate `MaxShadowMemory` limits
- Use `GCThreshold` to control cleanup frequency

### 3. Change Ordering
- Group related changes together
- Use priority levels appropriately
- Consider dependency relationships

### 4. Error Handling
- Enable rollback for production use
- Monitor reconciliation failures
- Implement circuit breakers for cascading failures

### 5. Performance Monitoring
- Track key metrics regularly
- Set up alerting for performance degradation
- Profile memory usage under load

## Future Enhancements

### Planned Features
1. **Distributed VGE**: Multi-node virtual graph support
2. **Persistent Shadow**: Disk-backed shadow graphs
3. **Advanced Batching**: ML-based adaptive batching
4. **Conflict Resolution**: Improved merge strategies
5. **Visual Debugging**: Graph diff visualization

### Performance Improvements
1. **Parallel Processing**: Multi-threaded reconciliation
2. **Memory Optimization**: Compressed shadow graphs
3. **Cache Optimization**: Predictive caching
4. **Algorithm Improvements**: Custom diff algorithms

## References

- [Virtual DOM Concepts](https://reactjs.org/docs/faq-internals.html)
- [Myers Diff Algorithm](https://blog.jcoglan.com/2017/02/12/the-myers-diff-algorithm-part-1/)
- [Tree Diffing Algorithms](https://grfia.dlsi.ua.es/ml/algorithms/references/editsurvey_bille.pdf)
- [LCS and Edit Distance](https://en.wikipedia.org/wiki/Longest_common_subsequence_problem)