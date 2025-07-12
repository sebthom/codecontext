package vgraph

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

// Reconciler handles the reconciliation of changes between shadow and actual graphs
type Reconciler struct {
	vge     *VirtualGraphEngine
	config  *ReconcilerConfig
	metrics *ReconcilerMetrics
}

// ReconcilerConfig holds configuration for the reconciler
type ReconcilerConfig struct {
	MaxConcurrency     int           `json:"max_concurrency"`     // Maximum concurrent operations
	ConflictResolution string        `json:"conflict_resolution"` // "abort", "force", "merge"
	DependencyOrdering bool          `json:"dependency_ordering"` // Enable dependency-aware ordering
	ValidationEnabled  bool          `json:"validation_enabled"`  // Enable pre/post validation
	RollbackEnabled    bool          `json:"rollback_enabled"`    // Enable rollback capability
	MaxPatchSize       int           `json:"max_patch_size"`      // Maximum patch size
	BatchTimeout       time.Duration `json:"batch_timeout"`       // Timeout for batch operations
}

// ReconcilerMetrics tracks reconciliation performance
type ReconcilerMetrics struct {
	PlansGenerated     int64         `json:"plans_generated"`
	PlansApplied       int64         `json:"plans_applied"`
	PlansRolledBack    int64         `json:"plans_rolled_back"`
	AvgPlanTime        time.Duration `json:"avg_plan_time"`
	AvgApplyTime       time.Duration `json:"avg_apply_time"`
	ConflictsDetected  int64         `json:"conflicts_detected"`
	ConflictsResolved  int64         `json:"conflicts_resolved"`
	LastReconciliation time.Time     `json:"last_reconciliation"`
}

// ConflictResolution represents a conflict resolution strategy
type ConflictResolution struct {
	Type       ConflictType           `json:"type"`
	Strategy   string                 `json:"strategy"`
	Resolution string                 `json:"resolution"`
	Confidence float64                `json:"confidence"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ConflictType represents the type of conflict
type ConflictType string

const (
	ConflictTypeValueMismatch ConflictType = "value_mismatch"
	ConflictTypeStructural    ConflictType = "structural"
	ConflictTypeDependency    ConflictType = "dependency"
	ConflictTypeOrderingIssue ConflictType = "ordering"
	ConflictTypeResourceLock  ConflictType = "resource_lock"
)

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid       bool                `json:"valid"`
	Errors      []ValidationError   `json:"errors"`
	Warnings    []ValidationWarning `json:"warnings"`
	Suggestions []string            `json:"suggestions"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	NodeID   string `json:"node_id,omitempty"`
	Severity string `json:"severity"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	NodeID  string `json:"node_id,omitempty"`
}

// NewReconciler creates a new reconciler
func NewReconciler(vge *VirtualGraphEngine) *Reconciler {
	return &Reconciler{
		vge: vge,
		config: &ReconcilerConfig{
			MaxConcurrency:     4,
			ConflictResolution: "merge",
			DependencyOrdering: true,
			ValidationEnabled:  true,
			RollbackEnabled:    true,
			MaxPatchSize:       1000,
			BatchTimeout:       30 * time.Second,
		},
		metrics: &ReconcilerMetrics{
			LastReconciliation: time.Now(),
		},
	}
}

// GeneratePlan generates a reconciliation plan between shadow and actual graphs
func (r *Reconciler) GeneratePlan(ctx context.Context, actual, shadow *types.CodeGraph) (*ReconciliationPlan, error) {
	start := time.Now()
	defer func() {
		r.metrics.PlansGenerated++
		r.metrics.AvgPlanTime = (r.metrics.AvgPlanTime + time.Since(start)) / 2
	}()

	plan := &ReconciliationPlan{
		Patches:       make([]GraphPatch, 0),
		UpdateOrder:   make([]types.NodeId, 0),
		Invalidations: make([]CacheInvalidation, 0),
		Dependencies:  make(map[string][]string),
	}

	// Generate patches for different components
	err := r.generateFilePatches(actual, shadow, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to generate file patches: %w", err)
	}

	err = r.generateSymbolPatches(actual, shadow, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to generate symbol patches: %w", err)
	}

	err = r.generateNodePatches(actual, shadow, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to generate node patches: %w", err)
	}

	err = r.generateEdgePatches(actual, shadow, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to generate edge patches: %w", err)
	}

	// Order patches by dependencies
	if r.config.DependencyOrdering {
		err = r.orderPatchesByDependencies(plan)
		if err != nil {
			return nil, fmt.Errorf("failed to order patches: %w", err)
		}
	}

	// Validate plan
	if r.config.ValidationEnabled {
		validation := r.validatePlan(plan)
		if !validation.Valid {
			return nil, fmt.Errorf("plan validation failed: %v", validation.Errors)
		}
	}

	// Estimate execution time and token impact
	plan.EstimatedTime = r.estimateExecutionTime(plan)
	plan.TokenImpact = r.calculateTokenImpact(actual, shadow)

	return plan, nil
}

// ApplyPlan applies a reconciliation plan to the actual graph
func (r *Reconciler) ApplyPlan(ctx context.Context, plan *ReconciliationPlan, actual *types.CodeGraph) error {
	start := time.Now()
	defer func() {
		r.metrics.PlansApplied++
		r.metrics.AvgApplyTime = (r.metrics.AvgApplyTime + time.Since(start)) / 2
		r.metrics.LastReconciliation = time.Now()
	}()

	// Create rollback point if enabled
	var rollbackData *RollbackData
	if r.config.RollbackEnabled {
		var err error
		rollbackData, err = r.createRollbackPoint(actual)
		if err != nil {
			return fmt.Errorf("failed to create rollback point: %w", err)
		}
	}

	// Apply patches in order
	appliedPatches := make([]GraphPatch, 0)
	for _, patch := range plan.Patches {
		select {
		case <-ctx.Done():
			// Context cancelled, rollback if enabled
			if r.config.RollbackEnabled && rollbackData != nil {
				r.rollbackToPoint(actual, rollbackData)
			}
			return ctx.Err()
		default:
		}

		err := r.applyPatch(ctx, patch, actual)
		if err != nil {
			// Patch failed, rollback if enabled
			if r.config.RollbackEnabled && rollbackData != nil {
				r.rollbackToPoint(actual, rollbackData)
				r.metrics.PlansRolledBack++
			}
			return fmt.Errorf("failed to apply patch %s: %w", patch.ID, err)
		}

		appliedPatches = append(appliedPatches, patch)
	}

	// Apply cache invalidations
	for _, invalidation := range plan.Invalidations {
		err := r.applyCacheInvalidation(invalidation)
		if err != nil {
			return fmt.Errorf("failed to apply cache invalidation: %w", err)
		}
	}

	return nil
}

// generateFilePatches generates patches for file changes
func (r *Reconciler) generateFilePatches(actual, shadow *types.CodeGraph, plan *ReconciliationPlan) error {
	// Compare files in actual vs shadow
	for filePath, shadowFile := range shadow.Files {
		actualFile, exists := actual.Files[filePath]

		if !exists {
			// File added in shadow
			patch := GraphPatch{
				ID:         fmt.Sprintf("file-add-%s", filePath),
				Type:       PatchTypeAdd,
				TargetNode: types.NodeId(fmt.Sprintf("file-%s", filePath)),
				Changes: []PropertyChange{
					{
						Property: "file",
						OldValue: nil,
						NewValue: shadowFile,
					},
				},
				Priority: 1,
			}
			plan.Patches = append(plan.Patches, patch)
		} else if r.filesAreDifferent(actualFile, shadowFile) {
			// File modified in shadow
			patch := GraphPatch{
				ID:         fmt.Sprintf("file-mod-%s", filePath),
				Type:       PatchTypeModify,
				TargetNode: types.NodeId(fmt.Sprintf("file-%s", filePath)),
				Changes:    r.generateFilePropertyChanges(actualFile, shadowFile),
				Priority:   2,
			}
			plan.Patches = append(plan.Patches, patch)
		}
	}

	// Check for files removed in shadow
	for filePath := range actual.Files {
		if _, exists := shadow.Files[filePath]; !exists {
			patch := GraphPatch{
				ID:         fmt.Sprintf("file-del-%s", filePath),
				Type:       PatchTypeRemove,
				TargetNode: types.NodeId(fmt.Sprintf("file-%s", filePath)),
				Changes: []PropertyChange{
					{
						Property: "file",
						OldValue: actual.Files[filePath],
						NewValue: nil,
					},
				},
				Priority: 3,
			}
			plan.Patches = append(plan.Patches, patch)
		}
	}

	return nil
}

// generateSymbolPatches generates patches for symbol changes
func (r *Reconciler) generateSymbolPatches(actual, shadow *types.CodeGraph, plan *ReconciliationPlan) error {
	// Compare symbols in actual vs shadow
	for symbolId, shadowSymbol := range shadow.Symbols {
		actualSymbol, exists := actual.Symbols[symbolId]

		if !exists {
			// Symbol added in shadow
			patch := GraphPatch{
				ID:         fmt.Sprintf("symbol-add-%s", symbolId),
				Type:       PatchTypeAdd,
				TargetNode: types.NodeId(fmt.Sprintf("symbol-%s", symbolId)),
				Changes: []PropertyChange{
					{
						Property: "symbol",
						OldValue: nil,
						NewValue: shadowSymbol,
					},
				},
				Priority: 1,
			}
			plan.Patches = append(plan.Patches, patch)
		} else if r.symbolsAreDifferent(actualSymbol, shadowSymbol) {
			// Symbol modified in shadow
			patch := GraphPatch{
				ID:         fmt.Sprintf("symbol-mod-%s", symbolId),
				Type:       PatchTypeModify,
				TargetNode: types.NodeId(fmt.Sprintf("symbol-%s", symbolId)),
				Changes:    r.generateSymbolPropertyChanges(actualSymbol, shadowSymbol),
				Priority:   2,
			}
			plan.Patches = append(plan.Patches, patch)
		}
	}

	// Check for symbols removed in shadow
	for symbolId := range actual.Symbols {
		if _, exists := shadow.Symbols[symbolId]; !exists {
			patch := GraphPatch{
				ID:         fmt.Sprintf("symbol-del-%s", symbolId),
				Type:       PatchTypeRemove,
				TargetNode: types.NodeId(fmt.Sprintf("symbol-%s", symbolId)),
				Changes: []PropertyChange{
					{
						Property: "symbol",
						OldValue: actual.Symbols[symbolId],
						NewValue: nil,
					},
				},
				Priority: 3,
			}
			plan.Patches = append(plan.Patches, patch)
		}
	}

	return nil
}

// generateNodePatches generates patches for node changes
func (r *Reconciler) generateNodePatches(actual, shadow *types.CodeGraph, plan *ReconciliationPlan) error {
	// Compare nodes (similar pattern to files and symbols)
	// Implementation would follow the same pattern as generateFilePatches
	return nil
}

// generateEdgePatches generates patches for edge changes
func (r *Reconciler) generateEdgePatches(actual, shadow *types.CodeGraph, plan *ReconciliationPlan) error {
	// Compare edges (similar pattern to files and symbols)
	// Implementation would follow the same pattern as generateFilePatches
	return nil
}

// orderPatchesByDependencies orders patches based on dependencies
func (r *Reconciler) orderPatchesByDependencies(plan *ReconciliationPlan) error {
	// Sort patches by priority first, then by dependencies
	sort.Slice(plan.Patches, func(i, j int) bool {
		if plan.Patches[i].Priority != plan.Patches[j].Priority {
			return plan.Patches[i].Priority < plan.Patches[j].Priority
		}
		// Further ordering logic based on dependencies would go here
		return plan.Patches[i].ID < plan.Patches[j].ID
	})

	// Build update order
	for _, patch := range plan.Patches {
		plan.UpdateOrder = append(plan.UpdateOrder, patch.TargetNode)
	}

	return nil
}

// validatePlan validates a reconciliation plan
func (r *Reconciler) validatePlan(plan *ReconciliationPlan) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationError, 0),
		Warnings:    make([]ValidationWarning, 0),
		Suggestions: make([]string, 0),
	}

	// Check for patch conflicts
	patchTargets := make(map[types.NodeId]bool)
	for _, patch := range plan.Patches {
		if patchTargets[patch.TargetNode] {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:     "conflict",
				Message:  fmt.Sprintf("Multiple patches target the same node: %s", patch.TargetNode),
				NodeID:   string(patch.TargetNode),
				Severity: "high",
			})
		}
		patchTargets[patch.TargetNode] = true
	}

	// Check patch size
	if len(plan.Patches) > r.config.MaxPatchSize {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Type:    "performance",
			Message: fmt.Sprintf("Large patch set (%d patches) may impact performance", len(plan.Patches)),
		})
	}

	// Add suggestions
	if len(plan.Patches) > 10 {
		result.Suggestions = append(result.Suggestions, "Consider breaking large changes into smaller batches")
	}

	return result
}

// applyPatch applies a single patch to the graph
func (r *Reconciler) applyPatch(ctx context.Context, patch GraphPatch, graph *types.CodeGraph) error {
	switch patch.Type {
	case PatchTypeAdd:
		return r.applyAddPatch(patch, graph)
	case PatchTypeRemove:
		return r.applyRemovePatch(patch, graph)
	case PatchTypeModify:
		return r.applyModifyPatch(patch, graph)
	case PatchTypeReorder:
		return r.applyReorderPatch(patch, graph)
	default:
		return fmt.Errorf("unknown patch type: %s", patch.Type)
	}
}

// applyAddPatch applies an add patch
func (r *Reconciler) applyAddPatch(patch GraphPatch, graph *types.CodeGraph) error {
	for _, change := range patch.Changes {
		switch change.Property {
		case "file":
			if file, ok := change.NewValue.(*types.FileNode); ok {
				if graph.Files == nil {
					graph.Files = make(map[string]*types.FileNode)
				}
				// Extract file path from target node ID
				filePath := string(patch.TargetNode)[5:] // Remove "file-" prefix
				graph.Files[filePath] = file
			}
		case "symbol":
			if symbol, ok := change.NewValue.(*types.Symbol); ok {
				if graph.Symbols == nil {
					graph.Symbols = make(map[types.SymbolId]*types.Symbol)
				}
				graph.Symbols[symbol.Id] = symbol
			}
		}
	}
	return nil
}

// applyRemovePatch applies a remove patch
func (r *Reconciler) applyRemovePatch(patch GraphPatch, graph *types.CodeGraph) error {
	for _, change := range patch.Changes {
		switch change.Property {
		case "file":
			filePath := string(patch.TargetNode)[5:] // Remove "file-" prefix
			if graph.Files != nil {
				delete(graph.Files, filePath)
			}
		case "symbol":
			symbolId := types.SymbolId(string(patch.TargetNode)[7:]) // Remove "symbol-" prefix
			if graph.Symbols != nil {
				delete(graph.Symbols, symbolId)
			}
		}
	}
	return nil
}

// applyModifyPatch applies a modify patch
func (r *Reconciler) applyModifyPatch(patch GraphPatch, graph *types.CodeGraph) error {
	// Apply property changes to existing entities
	for _, change := range patch.Changes {
		switch change.Property {
		case "file":
			if file, ok := change.NewValue.(*types.FileNode); ok {
				filePath := string(patch.TargetNode)[5:] // Remove "file-" prefix
				if graph.Files != nil {
					graph.Files[filePath] = file
				}
			}
		case "symbol":
			if symbol, ok := change.NewValue.(*types.Symbol); ok {
				if graph.Symbols != nil {
					graph.Symbols[symbol.Id] = symbol
				}
			}
		}
	}
	return nil
}

// applyReorderPatch applies a reorder patch
func (r *Reconciler) applyReorderPatch(patch GraphPatch, graph *types.CodeGraph) error {
	// Reordering logic would be implemented here
	return nil
}

// Helper functions

func (r *Reconciler) filesAreDifferent(file1, file2 *types.FileNode) bool {
	return file1.Size != file2.Size ||
		file1.Lines != file2.Lines ||
		file1.SymbolCount != file2.SymbolCount ||
		file1.ImportCount != file2.ImportCount
}

func (r *Reconciler) symbolsAreDifferent(sym1, sym2 *types.Symbol) bool {
	return sym1.Name != sym2.Name ||
		sym1.Type != sym2.Type ||
		sym1.Signature != sym2.Signature ||
		sym1.Documentation != sym2.Documentation
}

func (r *Reconciler) generateFilePropertyChanges(oldFile, newFile *types.FileNode) []PropertyChange {
	changes := make([]PropertyChange, 0)

	if oldFile.Size != newFile.Size {
		changes = append(changes, PropertyChange{
			Property: "size",
			OldValue: oldFile.Size,
			NewValue: newFile.Size,
		})
	}

	if oldFile.Lines != newFile.Lines {
		changes = append(changes, PropertyChange{
			Property: "lines",
			OldValue: oldFile.Lines,
			NewValue: newFile.Lines,
		})
	}

	return changes
}

func (r *Reconciler) generateSymbolPropertyChanges(oldSymbol, newSymbol *types.Symbol) []PropertyChange {
	changes := make([]PropertyChange, 0)

	if oldSymbol.Name != newSymbol.Name {
		changes = append(changes, PropertyChange{
			Property: "name",
			OldValue: oldSymbol.Name,
			NewValue: newSymbol.Name,
		})
	}

	if oldSymbol.Signature != newSymbol.Signature {
		changes = append(changes, PropertyChange{
			Property: "signature",
			OldValue: oldSymbol.Signature,
			NewValue: newSymbol.Signature,
		})
	}

	return changes
}

func (r *Reconciler) estimateExecutionTime(plan *ReconciliationPlan) time.Duration {
	// Simple estimation based on patch count
	baseTime := 10 * time.Millisecond
	return baseTime * time.Duration(len(plan.Patches))
}

func (r *Reconciler) calculateTokenImpact(actual, shadow *types.CodeGraph) *TokenDelta {
	// Calculate token impact (simplified)
	beforeTokens := len(actual.Files)*100 + len(actual.Symbols)*50
	afterTokens := len(shadow.Files)*100 + len(shadow.Symbols)*50

	return &TokenDelta{
		Before: beforeTokens,
		After:  afterTokens,
		Delta:  afterTokens - beforeTokens,
	}
}

func (r *Reconciler) applyCacheInvalidation(invalidation CacheInvalidation) error {
	// Apply cache invalidation logic
	return nil
}

// Rollback functionality

type RollbackData struct {
	Timestamp time.Time
	Files     map[string]*types.FileNode
	Symbols   map[types.SymbolId]*types.Symbol
	Nodes     map[types.NodeId]*types.GraphNode
	Edges     map[types.EdgeId]*types.GraphEdge
}

func (r *Reconciler) createRollbackPoint(graph *types.CodeGraph) (*RollbackData, error) {
	rollback := &RollbackData{
		Timestamp: time.Now(),
		Files:     make(map[string]*types.FileNode),
		Symbols:   make(map[types.SymbolId]*types.Symbol),
		Nodes:     make(map[types.NodeId]*types.GraphNode),
		Edges:     make(map[types.EdgeId]*types.GraphEdge),
	}

	// Deep copy current state
	for path, file := range graph.Files {
		fileCopy := *file
		rollback.Files[path] = &fileCopy
	}

	for id, symbol := range graph.Symbols {
		symbolCopy := *symbol
		rollback.Symbols[id] = &symbolCopy
	}

	return rollback, nil
}

func (r *Reconciler) rollbackToPoint(graph *types.CodeGraph, rollback *RollbackData) error {
	// Restore from rollback point
	graph.Files = rollback.Files
	graph.Symbols = rollback.Symbols
	graph.Nodes = rollback.Nodes
	graph.Edges = rollback.Edges

	return nil
}
