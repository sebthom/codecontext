package compact

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

// Strategy defines the interface for compaction strategies
type Strategy interface {
	// Compact performs the actual compaction
	Compact(ctx context.Context, request *CompactRequest) (*CompactResult, error)
	
	// GetName returns the strategy name
	GetName() string
	
	// GetDescription returns a description of the strategy
	GetDescription() string
}

// StrategyAnalyzer is an optional interface for strategies that can analyze potential
type StrategyAnalyzer interface {
	AnalyzePotential(graph *types.CodeGraph) *StrategyAnalysis
}

// BaseStrategy provides common functionality for all strategies
type BaseStrategy struct {
	name        string
	description string
}

// NewBaseStrategy creates a new base strategy
func NewBaseStrategy(name, description string) *BaseStrategy {
	return &BaseStrategy{
		name:        name,
		description: description,
	}
}

// GetName returns the strategy name
func (bs *BaseStrategy) GetName() string {
	return bs.name
}

// GetDescription returns the strategy description
func (bs *BaseStrategy) GetDescription() string {
	return bs.description
}

// RelevanceStrategy removes elements based on relevance to preserved items
type RelevanceStrategy struct {
	*BaseStrategy
}

// NewRelevanceStrategy creates a new relevance-based strategy
func NewRelevanceStrategy() *RelevanceStrategy {
	return &RelevanceStrategy{
		BaseStrategy: NewBaseStrategy("relevance", "Removes elements based on relevance to preserved items"),
	}
}

// AnalyzePotential implements StrategyAnalyzer
func (rs *RelevanceStrategy) AnalyzePotential(graph *types.CodeGraph) *StrategyAnalysis {
	return &StrategyAnalysis{
		EstimatedCompression: 0.7,
		RemovableFiles:       int(float64(len(graph.Files)) * 0.3),
		RemovableSymbols:     int(float64(len(graph.Symbols)) * 0.3),
		Confidence:           0.8,
		Warnings:             []string{},
	}
}

// Compact implements the Strategy interface
func (rs *RelevanceStrategy) Compact(ctx context.Context, request *CompactRequest) (*CompactResult, error) {
	graph := request.Graph
	requirements := request.Requirements
	
	// Create a copy of the graph to modify
	compactedGraph := rs.copyGraph(graph)
	removedItems := &RemovedItems{
		Files:   make([]string, 0),
		Symbols: make([]types.SymbolId, 0),
		Edges:   make([]types.EdgeId, 0),
		Nodes:   make([]types.NodeId, 0),
		Reason:  "relevance-based removal",
	}
	
	// Calculate relevance scores for all elements
	relevanceScores := rs.calculateRelevanceScores(graph, requirements)
	
	// Remove elements with low relevance scores
	threshold := 0.3 // Below this threshold, elements are considered irrelevant
	
	// Remove files
	for filePath, score := range relevanceScores.Files {
		if score < threshold && !rs.isFilePreserved(filePath, requirements) {
			delete(compactedGraph.Files, filePath)
			removedItems.Files = append(removedItems.Files, filePath)
		}
	}
	
	// Remove symbols
	for symbolId, score := range relevanceScores.Symbols {
		if score < threshold && !rs.isSymbolPreserved(symbolId, requirements) {
			delete(compactedGraph.Symbols, symbolId)
			removedItems.Symbols = append(removedItems.Symbols, symbolId)
		}
	}
	
	// Clean up orphaned nodes and edges
	rs.cleanupOrphans(compactedGraph, removedItems)
	
	return &CompactResult{
		CompactedGraph: compactedGraph,
		RemovedItems:   removedItems,
		Metadata: map[string]interface{}{
			"relevance_threshold": threshold,
			"scores_calculated":   len(relevanceScores.Files) + len(relevanceScores.Symbols),
		},
	}, nil
}

// FrequencyStrategy removes elements based on usage frequency
type FrequencyStrategy struct {
	*BaseStrategy
}

// NewFrequencyStrategy creates a new frequency-based strategy
func NewFrequencyStrategy() *FrequencyStrategy {
	return &FrequencyStrategy{
		BaseStrategy: NewBaseStrategy("frequency", "Removes elements based on usage frequency"),
	}
}

// AnalyzePotential implements StrategyAnalyzer
func (fs *FrequencyStrategy) AnalyzePotential(graph *types.CodeGraph) *StrategyAnalysis {
	return &StrategyAnalysis{
		EstimatedCompression: 0.6,
		RemovableFiles:       int(float64(len(graph.Files)) * 0.4),
		RemovableSymbols:     int(float64(len(graph.Symbols)) * 0.4),
		Confidence:           0.9,
		Warnings:             []string{},
	}
}

// Compact implements the Strategy interface
func (fs *FrequencyStrategy) Compact(ctx context.Context, request *CompactRequest) (*CompactResult, error) {
	graph := request.Graph
	requirements := request.Requirements
	
	compactedGraph := fs.copyGraph(graph)
	removedItems := &RemovedItems{
		Files:   make([]string, 0),
		Symbols: make([]types.SymbolId, 0),
		Edges:   make([]types.EdgeId, 0),
		Nodes:   make([]types.NodeId, 0),
		Reason:  "frequency-based removal",
	}
	
	// Calculate usage frequencies
	frequencies := fs.calculateFrequencies(graph)
	
	// Sort by frequency and remove least used items
	fileFreqs := fs.sortFilesByFrequency(frequencies.Files)
	symbolFreqs := fs.sortSymbolsByFrequency(frequencies.Symbols)
	
	// Remove bottom 30% of files by frequency
	removeCount := int(float64(len(fileFreqs)) * 0.3)
	for i := 0; i < removeCount && i < len(fileFreqs); i++ {
		filePath := fileFreqs[i].FilePath
		if !fs.isFilePreserved(filePath, requirements) {
			delete(compactedGraph.Files, filePath)
			removedItems.Files = append(removedItems.Files, filePath)
		}
	}
	
	// Remove bottom 30% of symbols by frequency
	removeCount = int(float64(len(symbolFreqs)) * 0.3)
	for i := 0; i < removeCount && i < len(symbolFreqs); i++ {
		symbolId := symbolFreqs[i].SymbolId
		if !fs.isSymbolPreserved(symbolId, requirements) {
			delete(compactedGraph.Symbols, symbolId)
			removedItems.Symbols = append(removedItems.Symbols, symbolId)
		}
	}
	
	fs.cleanupOrphans(compactedGraph, removedItems)
	
	return &CompactResult{
		CompactedGraph: compactedGraph,
		RemovedItems:   removedItems,
		Metadata: map[string]interface{}{
			"removal_percentage": 0.3,
			"files_analyzed":     len(fileFreqs),
			"symbols_analyzed":   len(symbolFreqs),
		},
	}, nil
}

// DependencyStrategy removes elements based on dependency analysis
type DependencyStrategy struct {
	*BaseStrategy
}

// NewDependencyStrategy creates a new dependency-based strategy
func NewDependencyStrategy() *DependencyStrategy {
	return &DependencyStrategy{
		BaseStrategy: NewBaseStrategy("dependency", "Removes elements based on dependency analysis"),
	}
}

// Compact implements the Strategy interface
func (ds *DependencyStrategy) Compact(ctx context.Context, request *CompactRequest) (*CompactResult, error) {
	graph := request.Graph
	requirements := request.Requirements
	
	compactedGraph := ds.copyGraph(graph)
	removedItems := &RemovedItems{
		Files:   make([]string, 0),
		Symbols: make([]types.SymbolId, 0),
		Edges:   make([]types.EdgeId, 0),
		Nodes:   make([]types.NodeId, 0),
		Reason:  "dependency-based removal",
	}
	
	// Analyze dependencies
	depAnalysis := ds.analyzeDependencies(graph)
	
	// Remove isolated nodes (no dependencies)
	for filePath, deps := range depAnalysis.FileDependencies {
		if deps.InDegree == 0 && deps.OutDegree == 0 && !ds.isFilePreserved(filePath, requirements) {
			delete(compactedGraph.Files, filePath)
			removedItems.Files = append(removedItems.Files, filePath)
		}
	}
	
	// Remove symbols with weak dependencies
	for symbolId, deps := range depAnalysis.SymbolDependencies {
		if deps.InDegree <= 1 && deps.OutDegree == 0 && !ds.isSymbolPreserved(symbolId, requirements) {
			delete(compactedGraph.Symbols, symbolId)
			removedItems.Symbols = append(removedItems.Symbols, symbolId)
		}
	}
	
	ds.cleanupOrphans(compactedGraph, removedItems)
	
	return &CompactResult{
		CompactedGraph: compactedGraph,
		RemovedItems:   removedItems,
		Metadata: map[string]interface{}{
			"isolated_files":   depAnalysis.IsolatedFiles,
			"isolated_symbols": depAnalysis.IsolatedSymbols,
			"dependency_depth": depAnalysis.MaxDepth,
		},
	}, nil
}

// SizeStrategy removes elements based on size considerations
type SizeStrategy struct {
	*BaseStrategy
}

// NewSizeStrategy creates a new size-based strategy
func NewSizeStrategy() *SizeStrategy {
	return &SizeStrategy{
		BaseStrategy: NewBaseStrategy("size", "Removes elements to achieve target size"),
	}
}

// Compact implements the Strategy interface
func (ss *SizeStrategy) Compact(ctx context.Context, request *CompactRequest) (*CompactResult, error) {
	graph := request.Graph
	requirements := request.Requirements
	maxSize := request.MaxSize
	
	compactedGraph := ss.copyGraph(graph)
	removedItems := &RemovedItems{
		Files:   make([]string, 0),
		Symbols: make([]types.SymbolId, 0),
		Edges:   make([]types.EdgeId, 0),
		Nodes:   make([]types.NodeId, 0),
		Reason:  "size-based removal",
	}
	
	currentSize := ss.calculateGraphSize(compactedGraph)
	
	// If already under target size, no action needed
	if currentSize <= maxSize {
		return &CompactResult{
			CompactedGraph: compactedGraph,
			RemovedItems:   removedItems,
			Metadata: map[string]interface{}{
				"target_size":    maxSize,
				"no_action":      true,
			},
		}, nil
	}
	
	// Calculate removal priorities based on size impact
	fileSizes := ss.calculateFileSizes(graph)
	
	// Sort files by size (largest first) and remove until target is reached
	type FileSizeInfo struct {
		FilePath string
		Size     int
	}
	
	fileInfos := make([]FileSizeInfo, 0, len(fileSizes))
	for filePath, size := range fileSizes {
		if !ss.isFilePreserved(filePath, requirements) {
			fileInfos = append(fileInfos, FileSizeInfo{filePath, size})
		}
	}
	
	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].Size > fileInfos[j].Size
	})
	
	// Remove largest files until target size is reached
	for _, fileInfo := range fileInfos {
		if currentSize <= maxSize {
			break
		}
		
		delete(compactedGraph.Files, fileInfo.FilePath)
		removedItems.Files = append(removedItems.Files, fileInfo.FilePath)
		currentSize -= fileInfo.Size
	}
	
	ss.cleanupOrphans(compactedGraph, removedItems)
	
	return &CompactResult{
		CompactedGraph: compactedGraph,
		RemovedItems:   removedItems,
		Metadata: map[string]interface{}{
			"target_size":    maxSize,
			"achieved_size":  ss.calculateGraphSize(compactedGraph),
			"files_removed":  len(removedItems.Files),
		},
	}, nil
}

// HybridStrategy combines multiple strategies
type HybridStrategy struct {
	*BaseStrategy
	strategies []Strategy
}

// NewHybridStrategy creates a new hybrid strategy
func NewHybridStrategy() *HybridStrategy {
	return &HybridStrategy{
		BaseStrategy: NewBaseStrategy("hybrid", "Combines multiple compaction strategies"),
		strategies: []Strategy{
			NewRelevanceStrategy(),
			NewFrequencyStrategy(),
			NewDependencyStrategy(),
		},
	}
}

// Compact implements the Strategy interface
func (hs *HybridStrategy) Compact(ctx context.Context, request *CompactRequest) (*CompactResult, error) {
	// Apply strategies in sequence, each working on the result of the previous
	currentGraph := request.Graph
	allRemovedItems := &RemovedItems{
		Files:   make([]string, 0),
		Symbols: make([]types.SymbolId, 0),
		Edges:   make([]types.EdgeId, 0),
		Nodes:   make([]types.NodeId, 0),
		Reason:  "hybrid strategy combination",
	}
	
	metadata := make(map[string]interface{})
	
	for _, strategy := range hs.strategies {
		// Create a new request for this strategy
		strategyRequest := &CompactRequest{
			Graph:        currentGraph,
			Strategy:     strategy.GetName(),
			MaxSize:      request.MaxSize,
			Priorities:   request.Priorities,
			Requirements: request.Requirements,
			Context:      request.Context,
		}
		
		result, err := strategy.Compact(ctx, strategyRequest)
		if err != nil {
			return nil, err
		}
		
		// Accumulate removed items
		allRemovedItems.Files = append(allRemovedItems.Files, result.RemovedItems.Files...)
		allRemovedItems.Symbols = append(allRemovedItems.Symbols, result.RemovedItems.Symbols...)
		allRemovedItems.Edges = append(allRemovedItems.Edges, result.RemovedItems.Edges...)
		allRemovedItems.Nodes = append(allRemovedItems.Nodes, result.RemovedItems.Nodes...)
		
		// Store strategy metadata
		metadata[strategy.GetName()] = result.Metadata
		
		// Update graph for next strategy
		currentGraph = result.CompactedGraph
	}
	
	return &CompactResult{
		CompactedGraph: currentGraph,
		RemovedItems:   allRemovedItems,
		Metadata: map[string]interface{}{
			"strategies_applied": len(hs.strategies),
			"strategy_results":   metadata,
		},
	}, nil
}

// AdaptiveStrategy dynamically selects the best approach
type AdaptiveStrategy struct {
	*BaseStrategy
}

// NewAdaptiveStrategy creates a new adaptive strategy
func NewAdaptiveStrategy() *AdaptiveStrategy {
	return &AdaptiveStrategy{
		BaseStrategy: NewBaseStrategy("adaptive", "Dynamically selects optimal compaction approach"),
	}
}

// Compact implements the Strategy interface
func (as *AdaptiveStrategy) Compact(ctx context.Context, request *CompactRequest) (*CompactResult, error) {
	graph := request.Graph
	
	// Analyze graph characteristics
	characteristics := as.analyzeGraphCharacteristics(graph)
	
	// Select best strategy based on characteristics
	var strategy Strategy
	
	if characteristics.HasHighConnectivity {
		strategy = NewDependencyStrategy()
	} else if characteristics.HasLargeFiles {
		strategy = NewSizeStrategy()
	} else if characteristics.HasManyUnusedSymbols {
		strategy = NewFrequencyStrategy()
	} else {
		strategy = NewRelevanceStrategy()
	}
	
	// Apply selected strategy
	result, err := strategy.Compact(ctx, request)
	if err != nil {
		return nil, err
	}
	
	// Add adaptive metadata
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.Metadata["adaptive_choice"] = strategy.GetName()
	result.Metadata["graph_characteristics"] = characteristics
	
	return result, nil
}

// Helper types and methods

type RelevanceScores struct {
	Files   map[string]float64
	Symbols map[types.SymbolId]float64
}

type FrequencyInfo struct {
	Files   map[string]int
	Symbols map[types.SymbolId]int
}

type FileFrequency struct {
	FilePath  string
	Frequency int
}

type SymbolFrequency struct {
	SymbolId  types.SymbolId
	Frequency int
}

type DependencyInfo struct {
	InDegree  int
	OutDegree int
}

type DependencyAnalysis struct {
	FileDependencies   map[string]DependencyInfo
	SymbolDependencies map[types.SymbolId]DependencyInfo
	IsolatedFiles      int
	IsolatedSymbols    int
	MaxDepth           int
}

type GraphCharacteristics struct {
	HasHighConnectivity   bool
	HasLargeFiles        bool
	HasManyUnusedSymbols bool
	AverageFileSize      float64
	ConnectivityRatio    float64
	UnusedSymbolRatio    float64
}

// Common helper methods that all strategies can use

func (bs *BaseStrategy) copyGraph(graph *types.CodeGraph) *types.CodeGraph {
	// Create a deep copy of the graph
	copied := &types.CodeGraph{
		Nodes:   make(map[types.NodeId]*types.GraphNode),
		Edges:   make(map[types.EdgeId]*types.GraphEdge),
		Files:   make(map[string]*types.FileNode),
		Symbols: make(map[types.SymbolId]*types.Symbol),
		Metadata: &types.GraphMetadata{
			TotalFiles:   graph.Metadata.TotalFiles,
			TotalSymbols: graph.Metadata.TotalSymbols,
			Generated:    graph.Metadata.Generated,
			Version:      graph.Metadata.Version,
		},
	}
	
	// Copy nodes
	for id, node := range graph.Nodes {
		copied.Nodes[id] = &types.GraphNode{
			Id:       node.Id,
			Type:     node.Type,
			Label:    node.Label,
			Metadata: node.Metadata,
		}
	}
	
	// Copy edges
	for id, edge := range graph.Edges {
		copied.Edges[id] = &types.GraphEdge{
			Id:     edge.Id,
			From:   edge.From,
			To:     edge.To,
			Type:   edge.Type,
			Weight: edge.Weight,
		}
	}
	
	// Copy files
	for path, file := range graph.Files {
		copied.Files[path] = &types.FileNode{
			Path:         file.Path,
			Language:     file.Language,
			Size:         file.Size,
			Lines:        file.Lines,
			SymbolCount:  file.SymbolCount,
			ImportCount:  file.ImportCount,
			IsTest:       file.IsTest,
			IsGenerated:  file.IsGenerated,
			LastModified: file.LastModified,
			Symbols:      make([]types.SymbolId, len(file.Symbols)),
			Imports:      make([]*types.Import, len(file.Imports)),
		}
		copy(copied.Files[path].Symbols, file.Symbols)
		copy(copied.Files[path].Imports, file.Imports)
	}
	
	// Copy symbols
	for id, symbol := range graph.Symbols {
		copied.Symbols[id] = &types.Symbol{
			Id:        symbol.Id,
			Name:      symbol.Name,
			Type:      symbol.Type,
			Location:  symbol.Location,
			Signature: symbol.Signature,
			Language:  symbol.Language,
		}
	}
	
	return copied
}

func (bs *BaseStrategy) isFilePreserved(filePath string, requirements *CompactRequirements) bool {
	if requirements == nil {
		return false
	}
	
	for _, preserved := range requirements.PreserveFiles {
		if preserved == filePath {
			return true
		}
	}
	
	for _, pattern := range requirements.PreservePaths {
		if bs.matchesPattern(filePath, pattern) {
			return true
		}
	}
	
	return false
}

func (bs *BaseStrategy) isSymbolPreserved(symbolId types.SymbolId, requirements *CompactRequirements) bool {
	if requirements == nil {
		return false
	}
	
	for _, preserved := range requirements.PreserveSymbols {
		if preserved == symbolId {
			return true
		}
	}
	
	return false
}

func (bs *BaseStrategy) matchesPattern(path, pattern string) bool {
	// Simple pattern matching - could be enhanced with proper glob support
	return strings.Contains(path, pattern)
}

func (bs *BaseStrategy) cleanupOrphans(graph *types.CodeGraph, removedItems *RemovedItems) {
	// Remove orphaned nodes and edges
	// This is a simplified implementation
	
	// Track which files and symbols still exist
	existingFiles := make(map[string]bool)
	existingSymbols := make(map[types.SymbolId]bool)
	
	for filePath := range graph.Files {
		existingFiles[filePath] = true
	}
	
	for symbolId := range graph.Symbols {
		existingSymbols[symbolId] = true
	}
	
	// Remove nodes and edges that reference removed files/symbols
	for nodeId := range graph.Nodes {
		// Check if node references removed items (simplified check)
		shouldRemove := false
		// Add logic here to determine if node should be removed
		
		if shouldRemove {
			delete(graph.Nodes, nodeId)
			removedItems.Nodes = append(removedItems.Nodes, nodeId)
		}
	}
	
	for edgeId, edge := range graph.Edges {
		// Check if edge references removed nodes
		_, sourceExists := graph.Nodes[edge.From]
		_, targetExists := graph.Nodes[edge.To]
		
		if !sourceExists || !targetExists {
			delete(graph.Edges, edgeId)
			removedItems.Edges = append(removedItems.Edges, edgeId)
		}
	}
}

func (bs *BaseStrategy) calculateGraphSize(graph *types.CodeGraph) int {
	return len(graph.Files) + len(graph.Symbols) + len(graph.Nodes) + len(graph.Edges)
}

// Strategy-specific helper methods

func (rs *RelevanceStrategy) calculateRelevanceScores(graph *types.CodeGraph, requirements *CompactRequirements) *RelevanceScores {
	scores := &RelevanceScores{
		Files:   make(map[string]float64),
		Symbols: make(map[types.SymbolId]float64),
	}
	
	// Initialize all scores to 0.1 (base relevance)
	for filePath := range graph.Files {
		scores.Files[filePath] = 0.1
	}
	
	for symbolId := range graph.Symbols {
		scores.Symbols[symbolId] = 0.1
	}
	
	// Increase scores for preserved items
	if requirements != nil {
		for _, filePath := range requirements.PreserveFiles {
			scores.Files[filePath] = 1.0
		}
		
		for _, symbolId := range requirements.PreserveSymbols {
			scores.Symbols[symbolId] = 1.0
		}
	}
	
	// Propagate relevance through dependencies
	// This is a simplified implementation
	for _, edge := range graph.Edges {
		if sourceScore, exists := scores.Files[string(edge.From)]; exists {
			if targetScore, exists := scores.Files[string(edge.To)]; exists {
				// Propagate relevance
				newScore := math.Min(1.0, targetScore + sourceScore*0.5)
				scores.Files[string(edge.To)] = newScore
			}
		}
	}
	
	return scores
}

func (fs *FrequencyStrategy) calculateFrequencies(graph *types.CodeGraph) *FrequencyInfo {
	frequencies := &FrequencyInfo{
		Files:   make(map[string]int),
		Symbols: make(map[types.SymbolId]int),
	}
	
	// Initialize frequencies
	for filePath := range graph.Files {
		frequencies.Files[filePath] = 0
	}
	
	for symbolId := range graph.Symbols {
		frequencies.Symbols[symbolId] = 0
	}
	
	// Count references through edges
	for _, edge := range graph.Edges {
		// Increment frequency for referenced items
		if sourceFile := string(edge.From); sourceFile != "" {
			frequencies.Files[sourceFile]++
		}
		if targetFile := string(edge.To); targetFile != "" {
			frequencies.Files[targetFile]++
		}
	}
	
	// Count symbol references through imports
	for _, file := range graph.Files {
		for _, symbolId := range file.Symbols {
			frequencies.Symbols[symbolId]++
		}
	}
	
	return frequencies
}

func (fs *FrequencyStrategy) sortFilesByFrequency(frequencies map[string]int) []FileFrequency {
	fileFreqs := make([]FileFrequency, 0, len(frequencies))
	
	for filePath, freq := range frequencies {
		fileFreqs = append(fileFreqs, FileFrequency{filePath, freq})
	}
	
	sort.Slice(fileFreqs, func(i, j int) bool {
		return fileFreqs[i].Frequency < fileFreqs[j].Frequency
	})
	
	return fileFreqs
}

func (fs *FrequencyStrategy) sortSymbolsByFrequency(frequencies map[types.SymbolId]int) []SymbolFrequency {
	symbolFreqs := make([]SymbolFrequency, 0, len(frequencies))
	
	for symbolId, freq := range frequencies {
		symbolFreqs = append(symbolFreqs, SymbolFrequency{symbolId, freq})
	}
	
	sort.Slice(symbolFreqs, func(i, j int) bool {
		return symbolFreqs[i].Frequency < symbolFreqs[j].Frequency
	})
	
	return symbolFreqs
}

func (ds *DependencyStrategy) analyzeDependencies(graph *types.CodeGraph) *DependencyAnalysis {
	analysis := &DependencyAnalysis{
		FileDependencies:   make(map[string]DependencyInfo),
		SymbolDependencies: make(map[types.SymbolId]DependencyInfo),
	}
	
	// Initialize dependency info
	for filePath := range graph.Files {
		analysis.FileDependencies[filePath] = DependencyInfo{}
	}
	
	for symbolId := range graph.Symbols {
		analysis.SymbolDependencies[symbolId] = DependencyInfo{}
	}
	
	// Count dependencies through edges
	for _, edge := range graph.Edges {
		// Update in-degree and out-degree
		if sourceFile := string(edge.From); sourceFile != "" {
			if info, exists := analysis.FileDependencies[sourceFile]; exists {
				info.OutDegree++
				analysis.FileDependencies[sourceFile] = info
			}
		}
		
		if targetFile := string(edge.To); targetFile != "" {
			if info, exists := analysis.FileDependencies[targetFile]; exists {
				info.InDegree++
				analysis.FileDependencies[targetFile] = info
			}
		}
	}
	
	// Count isolated nodes
	for _, deps := range analysis.FileDependencies {
		if deps.InDegree == 0 && deps.OutDegree == 0 {
			analysis.IsolatedFiles++
		}
	}
	
	for _, deps := range analysis.SymbolDependencies {
		if deps.InDegree == 0 && deps.OutDegree == 0 {
			analysis.IsolatedSymbols++
		}
	}
	
	return analysis
}

func (ss *SizeStrategy) calculateFileSizes(graph *types.CodeGraph) map[string]int {
	sizes := make(map[string]int)
	
	for filePath, file := range graph.Files {
		// Use a combination of actual size and symbol count as a size metric
		size := file.Size + file.SymbolCount*10 + file.Lines
		sizes[filePath] = size
	}
	
	return sizes
}

func (as *AdaptiveStrategy) analyzeGraphCharacteristics(graph *types.CodeGraph) *GraphCharacteristics {
	characteristics := &GraphCharacteristics{}
	
	// Calculate connectivity ratio
	totalPossibleEdges := len(graph.Nodes) * (len(graph.Nodes) - 1)
	if totalPossibleEdges > 0 {
		characteristics.ConnectivityRatio = float64(len(graph.Edges)) / float64(totalPossibleEdges)
		characteristics.HasHighConnectivity = characteristics.ConnectivityRatio > 0.1
	}
	
	// Calculate average file size
	totalSize := 0
	for _, file := range graph.Files {
		totalSize += file.Size
	}
	if len(graph.Files) > 0 {
		characteristics.AverageFileSize = float64(totalSize) / float64(len(graph.Files))
		characteristics.HasLargeFiles = characteristics.AverageFileSize > 10000
	}
	
	// Calculate unused symbol ratio (simplified)
	// In a real implementation, this would analyze actual usage
	characteristics.UnusedSymbolRatio = 0.3 // Placeholder
	characteristics.HasManyUnusedSymbols = characteristics.UnusedSymbolRatio > 0.2
	
	return characteristics
}