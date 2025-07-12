package analyzer

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nuthan-ms/codecontext/internal/parser"
	"github.com/nuthan-ms/codecontext/internal/vgraph"
	"github.com/nuthan-ms/codecontext/pkg/types"
)

// IncrementalAnalyzer provides incremental analysis using the Virtual Graph Engine
type IncrementalAnalyzer struct {
	vge           *vgraph.VirtualGraphEngine
	parser        *parser.Manager
	baseDir       string
	config        *IncrementalConfig
	fileVersions  map[string]string // Track file versions for change detection
	lastAnalysis  time.Time
	analysisCache map[string]*types.AST
}

// IncrementalConfig holds configuration for incremental analysis
type IncrementalConfig struct {
	EnableVGE          bool          `json:"enable_vge"`          // Enable Virtual Graph Engine
	DiffAlgorithm      string        `json:"diff_algorithm"`      // Diffing algorithm to use
	BatchSize          int           `json:"batch_size"`          // Batch size for changes
	BatchTimeout       time.Duration `json:"batch_timeout"`       // Batch timeout
	CacheEnabled       bool          `json:"cache_enabled"`       // Enable AST caching
	MaxCacheSize       int           `json:"max_cache_size"`      // Maximum cached ASTs
	ChangeDetection    string        `json:"change_detection"`    // "mtime", "hash", "content"
	IncrementalDepth   int           `json:"incremental_depth"`   // How deep to analyze dependencies
	ParallelProcessing bool          `json:"parallel_processing"` // Enable parallel processing
}

// FileChange represents a detected file change
type FileChange struct {
	Path          string                 `json:"path"`
	Type          ChangeType             `json:"type"`
	OldVersion    string                 `json:"old_version"`
	NewVersion    string                 `json:"new_version"`
	OldAST        *types.AST             `json:"old_ast,omitempty"`
	NewAST        *types.AST             `json:"new_ast,omitempty"`
	Diff          *vgraph.ASTDiff        `json:"diff,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	AffectedFiles []string               `json:"affected_files"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ChangeType represents the type of file change
type ChangeType string

const (
	ChangeTypeAdded    ChangeType = "added"
	ChangeTypeModified ChangeType = "modified"
	ChangeTypeRemoved  ChangeType = "removed"
	ChangeTypeRenamed  ChangeType = "renamed"
)

// IncrementalResult represents the result of incremental analysis
type IncrementalResult struct {
	UpdatedGraph     *types.CodeGraph    `json:"updated_graph"`
	ProcessedChanges []FileChange        `json:"processed_changes"`
	ImpactAnalysis   *ImpactSummary      `json:"impact_analysis"`
	Performance      *PerformanceMetrics `json:"performance"`
	Errors           []string            `json:"errors"`
}

// ImpactSummary summarizes the impact of changes
type ImpactSummary struct {
	TotalChanges      int      `json:"total_changes"`
	FilesAffected     int      `json:"files_affected"`
	SymbolsAffected   int      `json:"symbols_affected"`
	HighImpactChanges int      `json:"high_impact_changes"`
	RiskScore         float64  `json:"risk_score"`
	Recommendations   []string `json:"recommendations"`
}

// PerformanceMetrics tracks performance of incremental analysis
type PerformanceMetrics struct {
	TotalTime       time.Duration `json:"total_time"`
	ChangeDetection time.Duration `json:"change_detection"`
	ASTGeneration   time.Duration `json:"ast_generation"`
	DiffComputation time.Duration `json:"diff_computation"`
	GraphUpdate     time.Duration `json:"graph_update"`
	FilesProcessed  int           `json:"files_processed"`
	CacheHitRate    float64       `json:"cache_hit_rate"`
	MemoryUsage     int64         `json:"memory_usage"`
}

// NewIncrementalAnalyzer creates a new incremental analyzer
func NewIncrementalAnalyzer(baseDir string, config *IncrementalConfig) (*IncrementalAnalyzer, error) {
	if config == nil {
		config = DefaultIncrementalConfig()
	}

	// Create VGE configuration
	vgeConfig := &vgraph.VGEConfig{
		BatchThreshold:  config.BatchSize,
		BatchTimeout:    config.BatchTimeout,
		DiffAlgorithm:   config.DiffAlgorithm,
		EnableMetrics:   true,
		MaxShadowMemory: 200 * 1024 * 1024, // 200MB
		GCThreshold:     0.8,
	}

	analyzer := &IncrementalAnalyzer{
		vge:           vgraph.NewVirtualGraphEngine(vgeConfig),
		parser:        parser.NewManager(),
		baseDir:       baseDir,
		config:        config,
		fileVersions:  make(map[string]string),
		analysisCache: make(map[string]*types.AST),
		lastAnalysis:  time.Now(),
	}

	return analyzer, nil
}

// DefaultIncrementalConfig returns default configuration
func DefaultIncrementalConfig() *IncrementalConfig {
	return &IncrementalConfig{
		EnableVGE:          true,
		DiffAlgorithm:      "myers",
		BatchSize:          5,
		BatchTimeout:       500 * time.Millisecond,
		CacheEnabled:       true,
		MaxCacheSize:       1000,
		ChangeDetection:    "mtime",
		IncrementalDepth:   3,
		ParallelProcessing: true,
	}
}

// Initialize initializes the incremental analyzer with an existing graph
func (ia *IncrementalAnalyzer) Initialize(graph *types.CodeGraph) error {
	if !ia.config.EnableVGE {
		return fmt.Errorf("VGE is disabled in configuration")
	}

	// Initialize the Virtual Graph Engine
	err := ia.vge.Initialize(graph)
	if err != nil {
		return fmt.Errorf("failed to initialize VGE: %w", err)
	}

	// Build initial file version map
	for filePath := range graph.Files {
		version, err := ia.getFileVersion(filePath)
		if err != nil {
			continue // Skip files that can't be accessed
		}
		ia.fileVersions[filePath] = version
	}

	return nil
}

// AnalyzeChanges analyzes a set of file changes incrementally
func (ia *IncrementalAnalyzer) AnalyzeChanges(ctx context.Context, changedPaths []string) (*IncrementalResult, error) {
	start := time.Now()

	result := &IncrementalResult{
		ProcessedChanges: make([]FileChange, 0),
		Errors:           make([]string, 0),
		Performance:      &PerformanceMetrics{},
	}

	// Detect changes
	changeStart := time.Now()
	changes, err := ia.detectChanges(changedPaths)
	if err != nil {
		return nil, fmt.Errorf("change detection failed: %w", err)
	}
	result.Performance.ChangeDetection = time.Since(changeStart)

	// Process changes through VGE
	for _, change := range changes {
		err := ia.processFileChange(ctx, change, result)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to process %s: %v", change.Path, err))
			continue
		}
		result.ProcessedChanges = append(result.ProcessedChanges, change)
	}

	// Process pending changes in VGE
	err = ia.vge.ProcessPendingChanges(ctx)
	if err != nil {
		return nil, fmt.Errorf("VGE processing failed: %w", err)
	}

	// Get updated graph
	result.UpdatedGraph = ia.vge.GetActualGraph()

	// Compute impact analysis
	result.ImpactAnalysis = ia.computeImpactSummary(result.ProcessedChanges)

	// Update performance metrics
	result.Performance.TotalTime = time.Since(start)
	result.Performance.FilesProcessed = len(result.ProcessedChanges)

	// Get VGE metrics
	vgeMetrics := ia.vge.GetMetrics()
	result.Performance.MemoryUsage = vgeMetrics.ShadowMemoryBytes

	return result, nil
}

// detectChanges detects what changed in the specified files
func (ia *IncrementalAnalyzer) detectChanges(changedPaths []string) ([]FileChange, error) {
	changes := make([]FileChange, 0)

	for _, path := range changedPaths {
		change, err := ia.detectFileChange(path)
		if err != nil {
			continue // Skip files with errors
		}
		if change != nil {
			changes = append(changes, *change)
		}
	}

	return changes, nil
}

// detectFileChange detects changes in a specific file
func (ia *IncrementalAnalyzer) detectFileChange(filePath string) (*FileChange, error) {
	// Get current file version
	currentVersion, err := ia.getFileVersion(filePath)
	if err != nil {
		// File might have been deleted
		if os.IsNotExist(err) {
			if oldVersion, exists := ia.fileVersions[filePath]; exists {
				delete(ia.fileVersions, filePath)
				return &FileChange{
					Path:       filePath,
					Type:       ChangeTypeRemoved,
					OldVersion: oldVersion,
					NewVersion: "",
					Timestamp:  time.Now(),
				}, nil
			}
		}
		return nil, err
	}

	// Check if file is new or changed
	oldVersion, exists := ia.fileVersions[filePath]
	if !exists {
		// New file
		ia.fileVersions[filePath] = currentVersion
		return &FileChange{
			Path:       filePath,
			Type:       ChangeTypeAdded,
			OldVersion: "",
			NewVersion: currentVersion,
			Timestamp:  time.Now(),
		}, nil
	}

	if oldVersion != currentVersion {
		// Modified file
		ia.fileVersions[filePath] = currentVersion
		return &FileChange{
			Path:       filePath,
			Type:       ChangeTypeModified,
			OldVersion: oldVersion,
			NewVersion: currentVersion,
			Timestamp:  time.Now(),
		}, nil
	}

	// No change
	return nil, nil
}

// processFileChange processes a single file change
func (ia *IncrementalAnalyzer) processFileChange(ctx context.Context, change FileChange, result *IncrementalResult) error {
	switch change.Type {
	case ChangeTypeAdded:
		return ia.processFileAdded(ctx, change, result)
	case ChangeTypeModified:
		return ia.processFileModified(ctx, change, result)
	case ChangeTypeRemoved:
		return ia.processFileRemoved(ctx, change, result)
	default:
		return fmt.Errorf("unknown change type: %s", change.Type)
	}
}

// processFileAdded processes a newly added file
func (ia *IncrementalAnalyzer) processFileAdded(ctx context.Context, change FileChange, result *IncrementalResult) error {
	astStart := time.Now()

	// Parse the new file
	classification, err := ia.parser.ClassifyFile(change.Path)
	if err != nil {
		return fmt.Errorf("failed to classify file: %w", err)
	}

	ast, err := ia.parser.ParseFile(change.Path, classification.Language)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	result.Performance.ASTGeneration += time.Since(astStart)

	// Extract symbols
	symbols, err := ia.parser.ExtractSymbols(ast)
	if err != nil {
		return fmt.Errorf("failed to extract symbols: %w", err)
	}

	// Extract imports
	imports, err := ia.parser.ExtractImports(ast)
	if err != nil {
		return fmt.Errorf("failed to extract imports: %w", err)
	}

	// Create file node
	fileNode := &types.FileNode{
		Path:         change.Path,
		Language:     classification.Language.Name,
		Size:         len(ast.Content),
		Lines:        strings.Count(ast.Content, "\n") + 1,
		SymbolCount:  len(symbols),
		ImportCount:  len(imports),
		IsTest:       classification.IsTest,
		IsGenerated:  classification.IsGenerated,
		LastModified: time.Now(),
		Symbols:      make([]types.SymbolId, 0, len(symbols)),
		Imports:      imports,
	}

	// Create VGE change set for file addition
	vgeChange := vgraph.ChangeSet{
		ID:       fmt.Sprintf("add-file-%s", change.Path),
		Type:     vgraph.ChangeTypeFileAdd,
		FilePath: change.Path,
		Changes: []vgraph.Change{
			{
				Type:     vgraph.ChangeTypeFileAdd,
				Target:   change.Path,
				OldValue: nil,
				NewValue: fileNode,
			},
		},
		Timestamp: time.Now(),
	}

	// Add symbol changes
	for _, symbol := range symbols {
		vgeChange.Changes = append(vgeChange.Changes, vgraph.Change{
			Type:     vgraph.ChangeTypeSymbolAdd,
			Target:   string(symbol.Id),
			OldValue: nil,
			NewValue: symbol,
		})
		fileNode.Symbols = append(fileNode.Symbols, symbol.Id)
	}

	// Queue change to VGE
	return ia.vge.QueueChange(vgeChange)
}

// processFileModified processes a modified file
func (ia *IncrementalAnalyzer) processFileModified(ctx context.Context, change FileChange, result *IncrementalResult) error {
	astStart := time.Now()

	// Parse the modified file
	classification, err := ia.parser.ClassifyFile(change.Path)
	if err != nil {
		return fmt.Errorf("failed to classify file: %w", err)
	}

	newAST, err := ia.parser.ParseFile(change.Path, classification.Language)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	result.Performance.ASTGeneration += time.Since(astStart)

	// Get old AST from cache or actual graph
	oldAST := ia.getCachedAST(change.Path)
	if oldAST == nil {
		// Create a placeholder old AST if not found
		oldAST = &types.AST{
			FilePath: change.Path,
			Version:  change.OldVersion,
			Content:  "",
		}
	}

	// Cache new AST
	if ia.config.CacheEnabled {
		ia.cacheAST(change.Path, newAST)
	}

	// Compute diff
	diffStart := time.Now()
	differ := vgraph.NewASTDiffer()
	diff, err := differ.ComputeDiff(oldAST, newAST)
	if err != nil {
		return fmt.Errorf("failed to compute diff: %w", err)
	}
	result.Performance.DiffComputation += time.Since(diffStart)

	// Extract new symbols and imports
	symbols, err := ia.parser.ExtractSymbols(newAST)
	if err != nil {
		return fmt.Errorf("failed to extract symbols: %w", err)
	}

	imports, err := ia.parser.ExtractImports(newAST)
	if err != nil {
		return fmt.Errorf("failed to extract imports: %w", err)
	}

	// Create updated file node
	fileNode := &types.FileNode{
		Path:         change.Path,
		Language:     classification.Language.Name,
		Size:         len(newAST.Content),
		Lines:        strings.Count(newAST.Content, "\n") + 1,
		SymbolCount:  len(symbols),
		ImportCount:  len(imports),
		IsTest:       classification.IsTest,
		IsGenerated:  classification.IsGenerated,
		LastModified: time.Now(),
		Symbols:      make([]types.SymbolId, 0, len(symbols)),
		Imports:      imports,
	}

	// Create VGE change set for file modification
	vgeChange := vgraph.ChangeSet{
		ID:       fmt.Sprintf("mod-file-%s", change.Path),
		Type:     vgraph.ChangeTypeFileModify,
		FilePath: change.Path,
		Changes: []vgraph.Change{
			{
				Type:     vgraph.ChangeTypeFileModify,
				Target:   change.Path,
				OldValue: nil, // Would need to get old file node from graph
				NewValue: fileNode,
			},
		},
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"diff":                   diff,
			"has_structural_changes": diff.StructuralChanges,
			"similarity":             diff.Similarity,
		},
	}

	// Add symbol changes based on diff
	for _, symbol := range symbols {
		vgeChange.Changes = append(vgeChange.Changes, vgraph.Change{
			Type:     vgraph.ChangeTypeSymbolMod,
			Target:   string(symbol.Id),
			OldValue: nil, // Would need to get old symbol from graph
			NewValue: symbol,
		})
		fileNode.Symbols = append(fileNode.Symbols, symbol.Id)
	}

	// Queue change to VGE
	return ia.vge.QueueChange(vgeChange)
}

// processFileRemoved processes a removed file
func (ia *IncrementalAnalyzer) processFileRemoved(ctx context.Context, change FileChange, result *IncrementalResult) error {
	// Create VGE change set for file removal
	vgeChange := vgraph.ChangeSet{
		ID:       fmt.Sprintf("del-file-%s", change.Path),
		Type:     vgraph.ChangeTypeFileDelete,
		FilePath: change.Path,
		Changes: []vgraph.Change{
			{
				Type:     vgraph.ChangeTypeFileDelete,
				Target:   change.Path,
				OldValue: nil, // Would get from actual graph
				NewValue: nil,
			},
		},
		Timestamp: time.Now(),
	}

	// Queue change to VGE
	return ia.vge.QueueChange(vgeChange)
}

// computeImpactSummary computes impact summary from processed changes
func (ia *IncrementalAnalyzer) computeImpactSummary(changes []FileChange) *ImpactSummary {
	summary := &ImpactSummary{
		TotalChanges:      len(changes),
		FilesAffected:     len(changes),
		SymbolsAffected:   0,
		HighImpactChanges: 0,
		RiskScore:         0.0,
		Recommendations:   make([]string, 0),
	}

	for _, change := range changes {
		switch change.Type {
		case ChangeTypeRemoved:
			summary.HighImpactChanges++
			summary.RiskScore += 0.8
		case ChangeTypeModified:
			if change.Diff != nil && change.Diff.StructuralChanges {
				summary.HighImpactChanges++
				summary.RiskScore += 0.5
			}
		case ChangeTypeAdded:
			summary.RiskScore += 0.1
		}
	}

	// Generate recommendations
	if summary.HighImpactChanges > 0 {
		summary.Recommendations = append(summary.Recommendations,
			"High impact changes detected - review for breaking changes")
	}
	if summary.RiskScore > 2.0 {
		summary.Recommendations = append(summary.Recommendations,
			"High risk score - consider additional testing")
	}

	return summary
}

// Helper methods

func (ia *IncrementalAnalyzer) getFileVersion(filePath string) (string, error) {
	switch ia.config.ChangeDetection {
	case "mtime":
		info, err := os.Stat(filePath)
		if err != nil {
			return "", err
		}
		return info.ModTime().Format(time.RFC3339Nano), nil
	case "hash":
		// Would implement file hash here
		return "", fmt.Errorf("hash change detection not implemented")
	case "content":
		// Would implement content-based detection here
		return "", fmt.Errorf("content change detection not implemented")
	default:
		return "", fmt.Errorf("unknown change detection method: %s", ia.config.ChangeDetection)
	}
}

func (ia *IncrementalAnalyzer) getCachedAST(filePath string) *types.AST {
	if !ia.config.CacheEnabled {
		return nil
	}
	return ia.analysisCache[filePath]
}

func (ia *IncrementalAnalyzer) cacheAST(filePath string, ast *types.AST) {
	if !ia.config.CacheEnabled {
		return
	}

	// Simple cache size management
	if len(ia.analysisCache) >= ia.config.MaxCacheSize {
		// Remove oldest entry (simple FIFO)
		for key := range ia.analysisCache {
			delete(ia.analysisCache, key)
			break
		}
	}

	ia.analysisCache[filePath] = ast
}

// GetVGEMetrics returns Virtual Graph Engine metrics
func (ia *IncrementalAnalyzer) GetVGEMetrics() *vgraph.VGEMetrics {
	return ia.vge.GetMetrics()
}

// GetCurrentGraph returns the current graph state
func (ia *IncrementalAnalyzer) GetCurrentGraph() *types.CodeGraph {
	return ia.vge.GetActualGraph()
}
