# Compact Controller

The Compact Controller is a sophisticated context optimization system that reduces code graph size while preserving essential information. It employs multiple strategies to intelligently remove redundant or less relevant elements from code graphs.

## Overview

The Compact Controller addresses the challenge of managing large code graphs that can become unwieldy for analysis and processing. It provides configurable strategies to optimize graphs based on different criteria:

- **Relevance-based**: Removes elements based on their relevance to preserved items
- **Frequency-based**: Removes elements with low usage frequency
- **Dependency-based**: Removes isolated elements with weak dependencies
- **Size-based**: Removes largest elements to achieve target size
- **Hybrid**: Combines multiple strategies for balanced optimization
- **Adaptive**: Dynamically selects the best strategy based on graph characteristics

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                  Compact Controller                         │
├─────────────────┬─────────────────┬─────────────────────────┤
│   Strategy      │   Requirements  │    Configuration        │
│   Registry      │   Analysis      │    Management           │
│   - Registration│   - Preservation│    - Strategy Config    │
│   - Selection   │   - Constraints │    - Adaptive Rules     │
│   - Execution   │   - Validation  │    - Performance Tuning │
├─────────────────┼─────────────────┼─────────────────────────┤
│   Relevance     │   Frequency     │    Dependency           │
│   Strategy      │   Strategy      │    Strategy             │
│   - Propagation │   - Usage Count │    - Isolation Detection│
│   - Scoring     │   - Reference   │    - Connectivity       │
│   - Filtering   │     Analysis    │    - Graph Analysis     │
├─────────────────┼─────────────────┼─────────────────────────┤
│   Size          │   Hybrid        │    Adaptive             │
│   Strategy      │   Strategy      │    Strategy             │
│   - Target Size │   - Multi-pass  │    - Characteristic     │
│   - Priority    │   - Sequential  │      Analysis           │
│   - Optimization│   - Combination │    - Dynamic Selection  │
└─────────────────┴─────────────────┴─────────────────────────┘
```

### Key Interfaces

#### CompactController
```go
type CompactController struct {
    strategies map[string]Strategy
    config     *CompactConfig
    metrics    *CompactMetrics
}
```

#### Strategy Interface
```go
type Strategy interface {
    Compact(ctx context.Context, request *CompactRequest) (*CompactResult, error)
    GetName() string
    GetDescription() string
}
```

#### Core Operations
- `Compact(request)`: Optimize a graph using specified strategy
- `CompactMultiple(requests)`: Batch optimization with parallel processing
- `AnalyzeCompactionPotential(graph)`: Analyze optimization potential
- `RegisterStrategy(name, strategy)`: Register custom strategies

## Features

### 🎯 Strategy-Based Optimization

#### Available Strategies

| Strategy | Description | Best For | Compression | Speed |
|----------|-------------|----------|-------------|-------|
| **Relevance** | Removes elements based on relevance to preserved items | Focus on specific files/symbols | Medium | Fast |
| **Frequency** | Removes least used elements | General cleanup | High | Fast |
| **Dependency** | Removes isolated elements | Dependency optimization | Medium | Medium |
| **Size** | Removes largest elements first | Aggressive size reduction | High | Fast |
| **Hybrid** | Combines multiple strategies | Balanced optimization | High | Medium |
| **Adaptive** | Auto-selects best strategy | Unknown graph characteristics | Variable | Medium |

#### Strategy Selection
```go
// Manual strategy selection
request := &CompactRequest{
    Graph:    codeGraph,
    Strategy: "relevance",
    MaxSize:  5000,
}

// Adaptive strategy selection
request := &CompactRequest{
    Graph:   codeGraph,
    MaxSize: 5000,
    // Strategy will be selected automatically
}
```

### 📋 Preservation Requirements

#### File Preservation
```go
requirements := &CompactRequirements{
    PreserveFiles: []string{
        "src/core/engine.ts",
        "src/api/index.ts",
    },
    PreservePaths: []string{
        "src/critical/",
        "tests/integration/",
    },
}
```

#### Symbol Preservation
```go
requirements := &CompactRequirements{
    PreserveSymbols: []types.SymbolId{
        "main-function",
        "core-class",
        "api-interface",
    },
    RequiredTypes: []types.SymbolType{
        types.SymbolTypeInterface,
        types.SymbolTypeClass,
    },
}
```

#### Language Filtering
```go
requirements := &CompactRequirements{
    LanguageFilter: []string{
        "typescript",
        "javascript",
    },
    MinDepth: 2, // Minimum dependency depth to preserve
}
```

### 🔄 Batch Processing

#### Parallel Optimization
```go
requests := []*CompactRequest{
    {Graph: graph1, Strategy: "relevance", MaxSize: 5000},
    {Graph: graph2, Strategy: "frequency", MaxSize: 3000},
    {Graph: graph3, Strategy: "adaptive", MaxSize: 8000},
}

results, err := controller.CompactMultiple(ctx, requests)
```

#### Sequential Optimization
```go
config := &CompactConfig{
    ParallelProcessing: false, // Force sequential processing
    BatchSize:          10,
}
```

### 📊 Analysis and Metrics

#### Compaction Potential Analysis
```go
analysis := controller.AnalyzeCompactionPotential(graph)

fmt.Printf("Recommended strategy: %s\n", analysis.RecommendedStrategy)
fmt.Printf("Max compression ratio: %.2f\n", analysis.MaxCompressionRatio)
fmt.Printf("Estimated savings: %d elements\n", analysis.EstimatedSavings)

for strategy, analysis := range analysis.Strategies {
    fmt.Printf("%s: %.2f compression, %d removable files\n", 
        strategy, analysis.EstimatedCompression, analysis.RemovableFiles)
}
```

#### Performance Metrics
```go
metrics := controller.GetMetrics()

fmt.Printf("Total compactions: %d\n", metrics.TotalCompactions)
fmt.Printf("Average compression: %.2f\n", metrics.CompressionRatio)
fmt.Printf("Average time: %v\n", metrics.AverageTime)
fmt.Printf("Memory saved: %d bytes\n", metrics.MemorySaved)
fmt.Printf("Cache hit rate: %.2f%%\n", metrics.CacheHitRate*100)

// Strategy usage breakdown
for strategy, count := range metrics.StrategiesUsed {
    fmt.Printf("%s used %d times\n", strategy, count)
}
```

## Configuration

### Compact Configuration

```go
config := &CompactConfig{
    EnableCompaction:   true,              // Enable/disable compaction
    DefaultStrategy:    "hybrid",          // Default strategy to use
    MaxContextSize:     10000,             // Maximum allowed context size
    CompressionRatio:   0.7,               // Target compression ratio
    PriorityThreshold:  0.5,               // Priority threshold for preservation
    CacheEnabled:       true,              // Enable result caching
    CacheSize:          100,               // Cache size limit
    MetricsEnabled:     true,              // Enable metrics collection
    AdaptiveEnabled:    true,              // Enable adaptive strategy selection
    AdaptiveThreshold:  0.8,               // Threshold for adaptive triggers
    BatchSize:          50,                // Batch size for processing
    ParallelProcessing: true,              // Enable parallel processing
}
```

### Strategy-Specific Configuration

```go
config := &CompactConfig{
    StrategyConfig: map[string]interface{}{
        "relevance": map[string]interface{}{
            "threshold":        0.3,
            "propagation_depth": 3,
        },
        "frequency": map[string]interface{}{
            "removal_percentage": 0.3,
            "min_frequency":      1,
        },
        "size": map[string]interface{}{
            "aggressive_mode": true,
            "size_weight":     0.8,
        },
    },
}
```

## Usage Examples

### Basic Usage

```go
// Create controller
controller := NewCompactController(nil) // Uses default config

// Create compaction request
request := &CompactRequest{
    Graph:    codeGraph,
    Strategy: "hybrid",
    MaxSize:  5000,
    Requirements: &CompactRequirements{
        PreserveFiles: []string{"src/main.ts"},
    },
}

// Perform compaction
ctx := context.Background()
result, err := controller.Compact(ctx, request)
if err != nil {
    log.Fatal(err)
}

// Use compacted graph
compactedGraph := result.CompactedGraph
fmt.Printf("Reduced from %d to %d elements (%.2f compression)\n",
    result.OriginalSize, result.CompactedSize, result.CompressionRatio)
```

### Advanced Configuration

```go
config := &CompactConfig{
    EnableCompaction:   true,
    DefaultStrategy:    "adaptive",
    MaxContextSize:     15000,
    CompressionRatio:   0.6,
    PriorityThreshold:  0.4,
    CacheEnabled:       true,
    CacheSize:          200,
    MetricsEnabled:     true,
    AdaptiveEnabled:    true,
    AdaptiveThreshold:  0.75,
    BatchSize:          100,
    ParallelProcessing: true,
    StrategyConfig: map[string]interface{}{
        "relevance": map[string]interface{}{
            "threshold": 0.25,
        },
        "frequency": map[string]interface{}{
            "removal_percentage": 0.4,
        },
    },
}

controller := NewCompactController(config)
```

### Custom Strategy

```go
// Implement custom strategy
type CustomStrategy struct {
    *BaseStrategy
}

func NewCustomStrategy() *CustomStrategy {
    return &CustomStrategy{
        BaseStrategy: NewBaseStrategy("custom", "Custom optimization strategy"),
    }
}

func (cs *CustomStrategy) Compact(ctx context.Context, request *CompactRequest) (*CompactResult, error) {
    // Custom optimization logic
    compactedGraph := cs.copyGraph(request.Graph)
    
    // Apply custom optimization
    removedItems := &RemovedItems{
        Files:   []string{},
        Symbols: []types.SymbolId{},
        Reason:  "custom optimization",
    }
    
    return &CompactResult{
        CompactedGraph: compactedGraph,
        RemovedItems:   removedItems,
        Metadata: map[string]interface{}{
            "custom_metric": "value",
        },
    }, nil
}

// Register custom strategy
controller.RegisterStrategy("custom", NewCustomStrategy())
```

### Batch Processing

```go
// Process multiple graphs with different strategies
graphs := []*types.CodeGraph{graph1, graph2, graph3}
strategies := []string{"relevance", "frequency", "size"}

requests := make([]*CompactRequest, len(graphs))
for i, graph := range graphs {
    requests[i] = &CompactRequest{
        Graph:    graph,
        Strategy: strategies[i],
        MaxSize:  5000,
    }
}

// Process in parallel
results, err := controller.CompactMultiple(ctx, requests)
if err != nil {
    log.Fatal(err)
}

// Analyze results
for i, result := range results {
    fmt.Printf("Graph %d: %s strategy, %.2f compression\n",
        i+1, result.Strategy, result.CompressionRatio)
}
```

### Analysis and Monitoring

```go
// Analyze compaction potential before applying
analysis := controller.AnalyzeCompactionPotential(codeGraph)

fmt.Printf("Analysis Results:\n")
fmt.Printf("- Total elements: %d\n", 
    analysis.TotalFiles + analysis.TotalSymbols + analysis.TotalNodes + analysis.TotalEdges)
fmt.Printf("- Recommended strategy: %s\n", analysis.RecommendedStrategy)
fmt.Printf("- Expected compression: %.2f\n", analysis.MaxCompressionRatio)
fmt.Printf("- Estimated savings: %d elements\n", analysis.EstimatedSavings)

// Check if compaction is worthwhile
if analysis.MaxCompressionRatio < 0.8 {
    fmt.Println("Significant compaction potential detected")
    
    // Apply recommended strategy
    request := &CompactRequest{
        Graph:    codeGraph,
        Strategy: analysis.RecommendedStrategy,
        MaxSize:  analysis.EstimatedSavings,
    }
    
    result, err := controller.Compact(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Actual compression: %.2f\n", result.CompressionRatio)
}

// Monitor performance
metrics := controller.GetMetrics()
if metrics.AverageTime > 5*time.Second {
    fmt.Println("Warning: Compaction taking longer than expected")
}

if metrics.CompressionRatio > 0.9 {
    fmt.Println("Warning: Low compression ratio, consider different strategy")
}
```

## Performance Characteristics

### Time Complexity
- **Relevance Strategy**: O(n + e) where n = nodes, e = edges
- **Frequency Strategy**: O(n log n) for sorting by frequency
- **Dependency Strategy**: O(n + e) for dependency analysis
- **Size Strategy**: O(n log n) for sorting by size
- **Hybrid Strategy**: O(k × max(strategy_complexity)) where k = number of strategies
- **Adaptive Strategy**: O(analysis + selected_strategy)

### Space Complexity
- **Graph Copying**: O(n + e) - creates full copy of input graph
- **Strategy State**: O(n) - temporary data structures
- **Cache**: O(cache_size) - bounded by configuration
- **Metrics**: O(strategies) - constant per strategy

### Scalability
- **Nodes**: Tested up to 100k+ nodes efficiently
- **Strategies**: Up to 10 concurrent strategies without performance degradation
- **Batch Size**: Optimal batch sizes between 50-200 requests
- **Memory**: Configurable limits with automatic cleanup

## Best Practices

### 1. Strategy Selection
- **Small graphs** (< 1000 elements): Use `relevance` or `frequency`
- **Medium graphs** (1000-10000 elements): Use `hybrid`
- **Large graphs** (> 10000 elements): Use `adaptive` or `size`
- **Unknown characteristics**: Always start with `adaptive`

### 2. Preservation Requirements
- Be specific about what must be preserved
- Use path patterns instead of individual files when possible
- Consider dependency depth for interconnected code
- Test preservation requirements with sample data

### 3. Performance Optimization
- Enable caching for repeated operations
- Use parallel processing for batch operations
- Monitor metrics to identify performance bottlenecks
- Tune strategy-specific parameters based on your data

### 4. Configuration Tuning
- Start with default configuration
- Adjust compression ratio based on quality requirements
- Tune batch size based on available memory
- Enable adaptive mode for variable workloads

### 5. Error Handling
- Always check compaction results for warnings
- Validate preserved elements are still present
- Monitor compression ratios for quality degradation
- Implement fallback strategies for edge cases

## Integration with CodeContext

### CLI Integration
```bash
# Compact a code graph
codecontext compact --strategy hybrid --max-size 5000 --preserve "src/core/*"

# Analyze compaction potential
codecontext analyze-compact --graph output.json

# Batch compact multiple configurations
codecontext compact-batch --config compact-config.json
```

### Pipeline Integration
```go
// Integrate with analysis pipeline
analyzer := analyzer.NewAnalyzer(config)
controller := compact.NewCompactController(compactConfig)

// Analyze code
graph, err := analyzer.AnalyzeProject(projectPath)
if err != nil {
    return err
}

// Compact if graph is too large
if len(graph.Files) > 1000 {
    request := &compact.CompactRequest{
        Graph:    graph,
        Strategy: "adaptive",
        MaxSize:  800,
    }
    
    result, err := controller.Compact(ctx, request)
    if err != nil {
        return err
    }
    
    graph = result.CompactedGraph
}

// Continue with compacted graph
return generateOutput(graph)
```

## Testing

### Unit Tests
```bash
# Run compact controller tests
go test ./internal/compact/... -v

# Run with coverage
go test ./internal/compact/... -cover

# Run benchmarks
go test ./internal/compact/... -bench=.
```

### Integration Tests
```go
func TestCompactControllerIntegration(t *testing.T) {
    controller := NewCompactController(nil)
    largeGraph := createLargeTestGraph(10000) // 10k elements
    
    request := &CompactRequest{
        Graph:    largeGraph,
        Strategy: "adaptive",
        MaxSize:  1000,
    }
    
    ctx := context.Background()
    result, err := controller.Compact(ctx, request)
    require.NoError(t, err)
    
    // Verify significant compression
    assert.Less(t, result.CompressionRatio, 0.5)
    assert.LessOrEqual(t, result.CompactedSize, 1000)
    
    // Verify graph integrity
    verifyGraphIntegrity(t, result.CompactedGraph)
}
```

### Performance Tests
```go
func BenchmarkLargeGraphCompaction(b *testing.B) {
    controller := NewCompactController(nil)
    graph := createLargeTestGraph(50000) // 50k elements
    
    request := &CompactRequest{
        Graph:    graph,
        Strategy: "hybrid",
        MaxSize:  5000,
    }
    
    ctx := context.Background()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        controller.Compact(ctx, request)
    }
}
```

## Future Enhancements

### Planned Features
1. **Machine Learning**: ML-based relevance scoring
2. **Incremental Compaction**: Update existing compacted graphs
3. **Quality Metrics**: Advanced quality assessment
4. **Visual Analytics**: Compaction impact visualization
5. **Custom Preservers**: Pluggable preservation logic

### Performance Improvements
1. **Streaming Processing**: Process large graphs in chunks
2. **Memory Optimization**: Reduce memory footprint
3. **Parallel Strategies**: Run multiple strategies concurrently
4. **Smart Caching**: Predictive result caching

## References

- [Graph Compression Algorithms](https://en.wikipedia.org/wiki/Graph_compression)
- [Code Graph Analysis](https://docs.microsoft.com/en-us/visualstudio/code-quality/)
- [Software Metrics](https://en.wikipedia.org/wiki/Software_metric)
- [Dependency Analysis](https://en.wikipedia.org/wiki/Dependency_analysis)