package diff

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

// DiffEngine provides advanced diffing capabilities for ASTs and code structures
type DiffEngine struct {
	config       *Config
	semanticDiff *SemanticDiffer
	astDiff      *ASTDiffer
	renameDetect *RenameDetector
	depTracker   *DependencyTracker
	// cache        *DiffCache  // TODO: Implement caching
}

// Config holds configuration for the diff engine
type Config struct {
	EnableSemanticDiff    bool          `json:"enable_semantic_diff"`
	EnableStructuralDiff  bool          `json:"enable_structural_diff"`
	EnableRenameDetection bool          `json:"enable_rename_detection"`
	EnableDepTracking     bool          `json:"enable_dep_tracking"`
	SimilarityThreshold   float64       `json:"similarity_threshold"` // 0.0-1.0
	RenameThreshold       float64       `json:"rename_threshold"`     // 0.0-1.0
	MaxDiffDepth          int           `json:"max_diff_depth"`       // Maximum AST depth to diff
	Timeout               time.Duration `json:"timeout"`              // Per-diff timeout
	EnableCaching         bool          `json:"enable_caching"`
	CacheTTL              time.Duration `json:"cache_ttl"`
}

// DiffResult represents the result of comparing two code structures
type DiffResult struct {
	Type          DiffType           `json:"type"`
	FilePath      string             `json:"file_path"`
	Language      string             `json:"language"`
	Changes       []Change           `json:"changes"`
	Additions     []Addition         `json:"additions"`
	Deletions     []Deletion         `json:"deletions"`
	Modifications []Modification     `json:"modifications"`
	Renames       []Rename           `json:"renames"`
	Dependencies  []DependencyChange `json:"dependencies"`
	Metrics       DiffMetrics        `json:"metrics"`
	Timestamp     time.Time          `json:"timestamp"`
	ComputeTime   time.Duration      `json:"compute_time"`
}

// DiffType represents the type of diff operation
type DiffType string

const (
	DiffTypeStructural DiffType = "structural"
	DiffTypeSemantic   DiffType = "semantic"
	DiffTypeHybrid     DiffType = "hybrid"
)

// Change represents a single change in the diff
type Change struct {
	Type     ChangeType             `json:"type"`
	Path     string                 `json:"path"` // AST path to the change
	OldValue interface{}            `json:"old_value"`
	NewValue interface{}            `json:"new_value"`
	Position Position               `json:"position"`
	Impact   ImpactLevel            `json:"impact"`
	Context  ChangeContext          `json:"context"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ChangeType represents the type of change
type ChangeType string

const (
	ChangeTypeAdd    ChangeType = "add"
	ChangeTypeDelete ChangeType = "delete"
	ChangeTypeModify ChangeType = "modify"
	ChangeTypeMove   ChangeType = "move"
	ChangeTypeRename ChangeType = "rename"
)

// ImpactLevel represents the impact level of a change
type ImpactLevel string

const (
	ImpactLow      ImpactLevel = "low"
	ImpactMedium   ImpactLevel = "medium"
	ImpactHigh     ImpactLevel = "high"
	ImpactCritical ImpactLevel = "critical"
)

// Position represents a position in the source code
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
	Offset int `json:"offset"`
}

// ChangeContext provides context around a change
type ChangeContext struct {
	Function string   `json:"function"`
	Class    string   `json:"class"`
	Module   string   `json:"module"`
	Scope    string   `json:"scope"`
	Imports  []string `json:"imports"`
	Exports  []string `json:"exports"`
	Tags     []string `json:"tags"`
}

// Addition represents an added element
type Addition struct {
	Change
	Symbol *types.Symbol `json:"symbol"`
}

// Deletion represents a deleted element
type Deletion struct {
	Change
	Symbol *types.Symbol `json:"symbol"`
}

// Modification represents a modified element
type Modification struct {
	Change
	OldSymbol *types.Symbol `json:"old_symbol"`
	NewSymbol *types.Symbol `json:"new_symbol"`
}

// Rename represents a renamed element
type Rename struct {
	Change
	OldName    string        `json:"old_name"`
	NewName    string        `json:"new_name"`
	Symbol     *types.Symbol `json:"symbol"`
	Confidence float64       `json:"confidence"`
	Reason     string        `json:"reason"`
}

// DiffMetrics provides metrics about the diff operation
type DiffMetrics struct {
	TotalChanges     int     `json:"total_changes"`
	AddedLines       int     `json:"added_lines"`
	DeletedLines     int     `json:"deleted_lines"`
	ModifiedLines    int     `json:"modified_lines"`
	Similarity       float64 `json:"similarity"`        // 0.0-1.0
	Complexity       float64 `json:"complexity"`        // Diff complexity metric
	SemanticDistance float64 `json:"semantic_distance"` // Semantic similarity
}

// NewDiffEngine creates a new diff engine
func NewDiffEngine(config *Config) *DiffEngine {
	if config == nil {
		config = DefaultConfig()
	}

	engine := &DiffEngine{
		config: config,
	}

	if config.EnableSemanticDiff {
		engine.semanticDiff = NewSemanticDiffer(config)
	}

	if config.EnableStructuralDiff {
		engine.astDiff = NewASTDiffer(config)
	}

	if config.EnableRenameDetection {
		engine.renameDetect = NewRenameDetector(config)
	}

	if config.EnableDepTracking {
		engine.depTracker = NewDependencyTracker(config)
	}

	// TODO: Implement caching
	// if config.EnableCaching {
	//	engine.cache = NewDiffCache(config)
	// }

	return engine
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		EnableSemanticDiff:    true,
		EnableStructuralDiff:  true,
		EnableRenameDetection: true,
		EnableDepTracking:     true,
		SimilarityThreshold:   0.7,
		RenameThreshold:       0.8,
		MaxDiffDepth:          10,
		Timeout:               5 * time.Second,
		EnableCaching:         true,
		CacheTTL:              1 * time.Hour,
	}
}

// CompareFiles performs a comprehensive diff between two files
func (de *DiffEngine) CompareFiles(ctx context.Context, oldFile, newFile *types.FileInfo) (*DiffResult, error) {
	start := time.Now()

	// TODO: Implement caching
	// Check cache first
	// if de.cache != nil {
	//	if cached := de.cache.Get(oldFile, newFile); cached != nil {
	//		return cached, nil
	//	}
	// }

	// Create context with timeout
	diffCtx, cancel := context.WithTimeout(ctx, de.config.Timeout)
	defer cancel()

	result := &DiffResult{
		FilePath:      newFile.Path,
		Language:      newFile.Language,
		Changes:       make([]Change, 0),
		Additions:     make([]Addition, 0),
		Deletions:     make([]Deletion, 0),
		Modifications: make([]Modification, 0),
		Renames:       make([]Rename, 0),
		Dependencies:  make([]DependencyChange, 0),
		Timestamp:     time.Now(),
	}

	// Determine diff type based on configuration
	if de.config.EnableSemanticDiff && de.config.EnableStructuralDiff {
		result.Type = DiffTypeHybrid
	} else if de.config.EnableSemanticDiff {
		result.Type = DiffTypeSemantic
	} else {
		result.Type = DiffTypeStructural
	}

	// Perform structural diff if enabled
	if de.astDiff != nil {
		structuralChanges, err := de.astDiff.Compare(diffCtx, oldFile, newFile)
		if err != nil {
			return nil, fmt.Errorf("structural diff failed: %w", err)
		}
		result.Changes = append(result.Changes, structuralChanges...)
	}

	// Perform semantic diff if enabled
	if de.semanticDiff != nil {
		semanticChanges, err := de.semanticDiff.Compare(diffCtx, oldFile, newFile)
		if err != nil {
			return nil, fmt.Errorf("semantic diff failed: %w", err)
		}
		result.Changes = append(result.Changes, semanticChanges...)
	}

	// Detect renames if enabled
	if de.renameDetect != nil {
		renames, err := de.renameDetect.DetectRenames(diffCtx, oldFile, newFile)
		if err != nil {
			return nil, fmt.Errorf("rename detection failed: %w", err)
		}
		result.Renames = renames
	}

	// Track dependency changes if enabled
	if de.config.EnableDepTracking && de.depTracker != nil {
		depChanges, err := de.depTracker.TrackDependencyChanges(diffCtx, oldFile, newFile)
		if err != nil {
			return nil, fmt.Errorf("dependency tracking failed: %w", err)
		}
		// Convert changes to dependency changes for the result
		var dependencyChanges []DependencyChange
		for _, change := range depChanges {
			if depChange, ok := change.Metadata["dependency_change"].(DependencyChange); ok {
				dependencyChanges = append(dependencyChanges, depChange)
			}
		}
		result.Dependencies = dependencyChanges
		result.Changes = append(result.Changes, depChanges...)
	}

	// Categorize changes
	de.categorizeChanges(result)

	// Calculate metrics
	result.Metrics = de.calculateMetrics(oldFile, newFile, result)
	result.ComputeTime = time.Since(start)

	// TODO: Implement caching
	// Cache result if caching is enabled
	// if de.cache != nil {
	//	de.cache.Put(oldFile, newFile, result)
	// }

	return result, nil
}

// CompareSymbols performs a detailed comparison between two symbols
func (de *DiffEngine) CompareSymbols(ctx context.Context, oldSymbol, newSymbol *types.Symbol) (*DiffResult, error) {
	start := time.Now()

	result := &DiffResult{
		FilePath:      newSymbol.FullyQualifiedName, // Use qualified name as path for symbol comparison
		Language:      newSymbol.Language,
		Changes:       make([]Change, 0),
		Modifications: make([]Modification, 0),
		Timestamp:     time.Now(),
		Type:          DiffTypeSemantic,
	}

	// Compare symbol properties
	changes := de.compareSymbolProperties(oldSymbol, newSymbol)
	result.Changes = append(result.Changes, changes...)

	// Check for renames
	if oldSymbol.Name != newSymbol.Name {
		confidence := de.calculateRenameConfidence(oldSymbol, newSymbol)
		rename := Rename{
			Change: Change{
				Type:     ChangeTypeRename,
				Path:     newSymbol.FullyQualifiedName,
				OldValue: oldSymbol.Name,
				NewValue: newSymbol.Name,
				Position: Position{
					Line:   newSymbol.Location.StartLine,
					Column: newSymbol.Location.StartColumn,
				},
				Impact: de.calculateImpactLevel(ChangeTypeRename, oldSymbol, newSymbol),
			},
			OldName:    oldSymbol.Name,
			NewName:    newSymbol.Name,
			Symbol:     newSymbol,
			Confidence: confidence,
			Reason:     de.determineRenameReason(oldSymbol, newSymbol),
		}
		result.Renames = append(result.Renames, rename)
	}

	// Calculate metrics
	result.Metrics = de.calculateSymbolMetrics(oldSymbol, newSymbol, result)
	result.ComputeTime = time.Since(start)

	return result, nil
}

// categorizeChanges categorizes changes into additions, deletions, and modifications
func (de *DiffEngine) categorizeChanges(result *DiffResult) {
	for _, change := range result.Changes {
		switch change.Type {
		case ChangeTypeAdd:
			result.Additions = append(result.Additions, Addition{Change: change})
		case ChangeTypeDelete:
			result.Deletions = append(result.Deletions, Deletion{Change: change})
		case ChangeTypeModify:
			result.Modifications = append(result.Modifications, Modification{Change: change})
		}
	}
}

// compareSymbolProperties compares individual properties of symbols
func (de *DiffEngine) compareSymbolProperties(oldSymbol, newSymbol *types.Symbol) []Change {
	var changes []Change

	// Compare documentation
	if oldSymbol.Documentation != newSymbol.Documentation {
		changes = append(changes, Change{
			Type:     ChangeTypeModify,
			Path:     newSymbol.FullyQualifiedName + ".documentation",
			OldValue: oldSymbol.Documentation,
			NewValue: newSymbol.Documentation,
			Position: Position{
				Line:   newSymbol.Location.StartLine,
				Column: newSymbol.Location.StartColumn,
			},
			Impact: ImpactLow,
		})
	}

	// Compare signature
	if oldSymbol.Signature != newSymbol.Signature {
		changes = append(changes, Change{
			Type:     ChangeTypeModify,
			Path:     newSymbol.FullyQualifiedName + ".signature",
			OldValue: oldSymbol.Signature,
			NewValue: newSymbol.Signature,
			Position: Position{
				Line:   newSymbol.Location.StartLine,
				Column: newSymbol.Location.StartColumn,
			},
			Impact: de.calculateSignatureImpact(oldSymbol.Signature, newSymbol.Signature),
		})
	}

	// Compare visibility
	if oldSymbol.Visibility != newSymbol.Visibility {
		changes = append(changes, Change{
			Type:     ChangeTypeModify,
			Path:     newSymbol.FullyQualifiedName + ".visibility",
			OldValue: oldSymbol.Visibility,
			NewValue: newSymbol.Visibility,
			Position: Position{
				Line:   newSymbol.Location.StartLine,
				Column: newSymbol.Location.StartColumn,
			},
			Impact: de.calculateVisibilityImpact(oldSymbol.Visibility, newSymbol.Visibility),
		})
	}

	return changes
}

// calculateMetrics calculates diff metrics
func (de *DiffEngine) calculateMetrics(oldFile, newFile *types.FileInfo, result *DiffResult) DiffMetrics {
	metrics := DiffMetrics{
		TotalChanges: len(result.Changes),
	}

	// Count different types of changes
	for _, change := range result.Changes {
		switch change.Type {
		case ChangeTypeAdd:
			metrics.AddedLines++
		case ChangeTypeDelete:
			metrics.DeletedLines++
		case ChangeTypeModify:
			metrics.ModifiedLines++
		}
	}

	// Calculate similarity
	if oldFile != nil && newFile != nil {
		metrics.Similarity = de.calculateSimilarity(oldFile, newFile)
		metrics.SemanticDistance = de.calculateSemanticDistance(oldFile, newFile)
	}

	// Calculate complexity (based on number and types of changes)
	metrics.Complexity = de.calculateComplexity(result)

	return metrics
}

// calculateSymbolMetrics calculates metrics for symbol comparison
func (de *DiffEngine) calculateSymbolMetrics(oldSymbol, newSymbol *types.Symbol, result *DiffResult) DiffMetrics {
	return DiffMetrics{
		TotalChanges:     len(result.Changes),
		Similarity:       de.calculateSymbolSimilarity(oldSymbol, newSymbol),
		SemanticDistance: de.calculateSymbolSemanticDistance(oldSymbol, newSymbol),
		Complexity:       de.calculateSymbolComplexity(result),
	}
}

// calculateRenameConfidence calculates confidence score for rename detection
func (de *DiffEngine) calculateRenameConfidence(oldSymbol, newSymbol *types.Symbol) float64 {
	// Compare signatures
	sigSimilarity := de.calculateStringSimilarity(oldSymbol.Signature, newSymbol.Signature)

	// Compare documentation
	docSimilarity := de.calculateStringSimilarity(oldSymbol.Documentation, newSymbol.Documentation)

	// Compare location (relative position in file)
	locSimilarity := de.calculateLocationSimilarity(oldSymbol.Location, newSymbol.Location)

	// Weighted average
	confidence := (sigSimilarity*0.5 + docSimilarity*0.3 + locSimilarity*0.2)

	return confidence
}

// calculateStringSimilarity calculates similarity between two strings
func (de *DiffEngine) calculateStringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	if s1 == "" || s2 == "" {
		return 0.0
	}

	// Use Levenshtein distance-based similarity
	maxLen := max(len(s1), len(s2))
	distance := de.levenshteinDistance(s1, s2)

	return 1.0 - float64(distance)/float64(maxLen)
}

// calculateLocationSimilarity calculates similarity based on symbol locations
func (de *DiffEngine) calculateLocationSimilarity(loc1, loc2 types.Location) float64 {
	lineDiff := abs(loc1.StartLine - loc2.StartLine)

	// Similarity decreases with line distance
	if lineDiff == 0 {
		return 1.0
	} else if lineDiff <= 5 {
		return 0.8
	} else if lineDiff <= 20 {
		return 0.5
	} else {
		return 0.1
	}
}

// Helper functions

func (de *DiffEngine) calculateImpactLevel(changeType ChangeType, oldSymbol, newSymbol *types.Symbol) ImpactLevel {
	switch changeType {
	case ChangeTypeRename:
		if oldSymbol.Visibility == "public" || newSymbol.Visibility == "public" {
			return ImpactHigh
		}
		return ImpactMedium
	case ChangeTypeDelete:
		if oldSymbol.Visibility == "public" {
			return ImpactCritical
		}
		return ImpactHigh
	case ChangeTypeAdd:
		return ImpactLow
	default:
		return ImpactMedium
	}
}

func (de *DiffEngine) calculateSignatureImpact(oldSig, newSig string) ImpactLevel {
	// Simple heuristic: breaking changes are high impact
	if strings.Contains(oldSig, "public") && !strings.Contains(newSig, "public") {
		return ImpactCritical
	}

	// Parameter changes are medium to high impact
	if strings.Count(oldSig, ",") != strings.Count(newSig, ",") {
		return ImpactHigh
	}

	return ImpactMedium
}

func (de *DiffEngine) calculateVisibilityImpact(oldVis, newVis string) ImpactLevel {
	// Visibility changes can have significant impact
	if oldVis == "public" && newVis != "public" {
		return ImpactCritical
	}
	if oldVis != "public" && newVis == "public" {
		return ImpactMedium
	}
	return ImpactLow
}

func (de *DiffEngine) calculateSimilarity(oldFile, newFile *types.FileInfo) float64 {
	// Simple similarity based on file size and line count
	if oldFile.Lines == 0 && newFile.Lines == 0 {
		return 1.0
	}

	sizeDiff := abs(int(oldFile.Size - newFile.Size))
	lineDiff := abs(oldFile.Lines - newFile.Lines)

	maxSize := max(int(oldFile.Size), int(newFile.Size))
	maxLines := max(oldFile.Lines, newFile.Lines)

	sizeSim := 1.0 - float64(sizeDiff)/float64(maxSize)
	lineSim := 1.0 - float64(lineDiff)/float64(maxLines)

	return (sizeSim + lineSim) / 2.0
}

func (de *DiffEngine) calculateSemanticDistance(oldFile, newFile *types.FileInfo) float64 {
	// Placeholder for semantic distance calculation
	// This would involve more sophisticated analysis
	return 0.5
}

func (de *DiffEngine) calculateComplexity(result *DiffResult) float64 {
	// Complexity based on number and types of changes
	complexity := 0.0

	for _, change := range result.Changes {
		switch change.Impact {
		case ImpactLow:
			complexity += 1.0
		case ImpactMedium:
			complexity += 2.0
		case ImpactHigh:
			complexity += 3.0
		case ImpactCritical:
			complexity += 5.0
		}
	}

	// Normalize by total changes
	if len(result.Changes) > 0 {
		complexity /= float64(len(result.Changes))
	}

	return complexity
}

func (de *DiffEngine) calculateSymbolSimilarity(oldSymbol, newSymbol *types.Symbol) float64 {
	nameSim := de.calculateStringSimilarity(oldSymbol.Name, newSymbol.Name)
	sigSim := de.calculateStringSimilarity(oldSymbol.Signature, newSymbol.Signature)

	return (nameSim + sigSim) / 2.0
}

func (de *DiffEngine) calculateSymbolSemanticDistance(oldSymbol, newSymbol *types.Symbol) float64 {
	// Placeholder for symbol semantic distance
	return 0.5
}

func (de *DiffEngine) calculateSymbolComplexity(result *DiffResult) float64 {
	return de.calculateComplexity(result)
}

func (de *DiffEngine) determineRenameReason(oldSymbol, newSymbol *types.Symbol) string {
	if oldSymbol.Kind != newSymbol.Kind {
		return "kind_change"
	}
	if oldSymbol.Signature != newSymbol.Signature {
		return "signature_change"
	}
	return "simple_rename"
}

func (de *DiffEngine) levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}

	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// GenerateHash generates a hash for caching purposes
func (de *DiffEngine) GenerateHash(oldFile, newFile *types.FileInfo) string {
	hasher := sha256.New()
	hasher.Write([]byte(oldFile.Path))
	hasher.Write([]byte(newFile.Path))
	hasher.Write([]byte(fmt.Sprintf("%d", oldFile.ModTime.Unix())))
	hasher.Write([]byte(fmt.Sprintf("%d", newFile.ModTime.Unix())))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// Utility functions
// Utility functions moved to utils.go
