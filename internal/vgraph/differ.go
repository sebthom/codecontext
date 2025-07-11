package vgraph

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

// ASTDiffer computes structural differences between AST versions
type ASTDiffer struct {
	config        *DiffConfig
	cache         map[string]*ASTDiff
	hashCache     map[string]string
}

// DiffConfig holds configuration for AST diffing
type DiffConfig struct {
	Algorithm       string  `json:"algorithm"`        // myers, patience, histogram
	MaxDepth        int     `json:"max_depth"`        // Maximum diff depth
	UseTreeHashing  bool    `json:"use_tree_hashing"` // Enable tree hashing optimization
	UseMemoization  bool    `json:"use_memoization"`  // Enable memoization
	SimilarityThreshold float64 `json:"similarity_threshold"` // Threshold for similarity detection
}

// ASTDiff represents the difference between two AST versions
type ASTDiff struct {
	FileID           string             `json:"file_id"`
	FromVersion      string             `json:"from_version"`
	ToVersion        string             `json:"to_version"`
	Additions        []ASTNode          `json:"additions"`
	Deletions        []ASTNode          `json:"deletions"`
	Modifications    []ASTModification  `json:"modifications"`
	StructuralChanges bool              `json:"structural_changes"`
	ImpactRadius     *ImpactAnalysis    `json:"impact_radius"`
	Similarity       float64            `json:"similarity"`
	ComputationTime  time.Duration      `json:"computation_time"`
	Hash             string             `json:"hash"`
}

// ASTNode represents a node in the AST
type ASTNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Value    string                 `json:"value"`
	Children []ASTNode              `json:"children"`
	Location types.FileLocation     `json:"location"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ASTModification represents a modification to an AST node
type ASTModification struct {
	NodeID      string             `json:"node_id"`
	Type        ModificationType   `json:"type"`
	OldValue    interface{}        `json:"old_value"`
	NewValue    interface{}        `json:"new_value"`
	FieldName   string             `json:"field_name"`
	Impact      ImpactLevel        `json:"impact"`
}

// ModificationType represents the type of modification
type ModificationType string

const (
	ModificationValueChange    ModificationType = "value_change"
	ModificationTypeChange     ModificationType = "type_change"
	ModificationLocationChange ModificationType = "location_change"
	ModificationChildAdd       ModificationType = "child_add"
	ModificationChildRemove    ModificationType = "child_remove"
	ModificationChildReorder   ModificationType = "child_reorder"
)

// ImpactLevel represents the impact level of a change
type ImpactLevel string

const (
	ImpactLow      ImpactLevel = "low"
	ImpactMedium   ImpactLevel = "medium"
	ImpactHigh     ImpactLevel = "high"
	ImpactCritical ImpactLevel = "critical"
)

// ImpactAnalysis represents the impact analysis of changes
type ImpactAnalysis struct {
	AffectedFiles   []string               `json:"affected_files"`
	AffectedSymbols []types.SymbolId       `json:"affected_symbols"`
	PropagationTree *PropagationNode       `json:"propagation_tree"`
	RiskScore       float64                `json:"risk_score"`
	Recommendations []string               `json:"recommendations"`
}

// PropagationNode represents a node in the change propagation tree
type PropagationNode struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Impact       ImpactLevel        `json:"impact"`
	Children     []*PropagationNode `json:"children"`
	Probability  float64            `json:"probability"`
}

// SymbolChangeSet represents a set of symbol changes
type SymbolChangeSet struct {
	Added    map[types.SymbolId]*types.Symbol        `json:"added"`
	Removed  map[types.SymbolId]*types.Symbol        `json:"removed"`
	Modified map[types.SymbolId]*SymbolModification  `json:"modified"`
	Renamed  map[types.SymbolId]*RenameInfo          `json:"renamed"`
}

// SymbolModification represents a modification to a symbol
type SymbolModification struct {
	Symbol      *types.Symbol      `json:"symbol"`
	Changes     []PropertyChange   `json:"changes"`
	Impact      ImpactLevel        `json:"impact"`
	Confidence  float64            `json:"confidence"`
}

// RenameInfo represents information about a renamed symbol
type RenameInfo struct {
	OldName    string  `json:"old_name"`
	NewName    string  `json:"new_name"`
	Confidence float64 `json:"confidence"`
}

// NewASTDiffer creates a new AST differ
func NewASTDiffer() *ASTDiffer {
	return &ASTDiffer{
		config: &DiffConfig{
			Algorithm:           "myers",
			MaxDepth:           50,
			UseTreeHashing:     true,
			UseMemoization:     true,
			SimilarityThreshold: 0.8,
		},
		cache:     make(map[string]*ASTDiff),
		hashCache: make(map[string]string),
	}
}

// ComputeDiff computes the difference between two ASTs
func (d *ASTDiffer) ComputeDiff(oldAST, newAST *types.AST) (*ASTDiff, error) {
	start := time.Now()

	// Generate cache key
	cacheKey := d.generateCacheKey(oldAST, newAST)
	
	// Check cache first
	if d.config.UseMemoization {
		if cached, exists := d.cache[cacheKey]; exists {
			return cached, nil
		}
	}

	// Convert ASTs to internal format
	oldNodes, err := d.convertAST(oldAST)
	if err != nil {
		return nil, fmt.Errorf("failed to convert old AST: %w", err)
	}

	newNodes, err := d.convertAST(newAST)
	if err != nil {
		return nil, fmt.Errorf("failed to convert new AST: %w", err)
	}

	// Compute structural diff
	diff := &ASTDiff{
		FileID:           oldAST.FilePath,
		FromVersion:      oldAST.Version,
		ToVersion:        newAST.Version,
		ComputationTime:  0, // Will be set at the end
	}

	// Perform diff computation based on algorithm
	switch d.config.Algorithm {
	case "myers":
		err = d.myersDiff(oldNodes, newNodes, diff)
	case "patience":
		err = d.patienceDiff(oldNodes, newNodes, diff)
	case "histogram":
		err = d.histogramDiff(oldNodes, newNodes, diff)
	default:
		err = d.myersDiff(oldNodes, newNodes, diff)
	}

	if err != nil {
		return nil, fmt.Errorf("diff computation failed: %w", err)
	}

	// Compute impact analysis
	diff.ImpactRadius, err = d.computeImpactAnalysis(diff)
	if err != nil {
		return nil, fmt.Errorf("impact analysis failed: %w", err)
	}

	// Calculate similarity
	diff.Similarity = d.calculateSimilarity(oldNodes, newNodes)

	// Determine if structural changes occurred
	diff.StructuralChanges = d.hasStructuralChanges(diff)

	// Generate hash
	diff.Hash = d.generateDiffHash(diff)

	// Set computation time
	diff.ComputationTime = time.Since(start)

	// Cache result
	if d.config.UseMemoization {
		d.cache[cacheKey] = diff
	}

	return diff, nil
}

// TrackSymbolChanges tracks changes at the symbol level
func (d *ASTDiffer) TrackSymbolChanges(diff *ASTDiff) (*SymbolChangeSet, error) {
	changeSet := &SymbolChangeSet{
		Added:    make(map[types.SymbolId]*types.Symbol),
		Removed:  make(map[types.SymbolId]*types.Symbol),
		Modified: make(map[types.SymbolId]*SymbolModification),
		Renamed:  make(map[types.SymbolId]*RenameInfo),
	}

	// Analyze additions for new symbols
	for _, addition := range diff.Additions {
		if symbol := d.extractSymbolFromNode(addition); symbol != nil {
			changeSet.Added[symbol.Id] = symbol
		}
	}

	// Analyze deletions for removed symbols
	for _, deletion := range diff.Deletions {
		if symbol := d.extractSymbolFromNode(deletion); symbol != nil {
			changeSet.Removed[symbol.Id] = symbol
		}
	}

	// Analyze modifications for changed symbols
	for _, modification := range diff.Modifications {
		if symbol := d.extractSymbolFromModification(modification); symbol != nil {
			symMod := &SymbolModification{
				Symbol:     symbol,
				Changes:    []PropertyChange{},
				Impact:     modification.Impact,
				Confidence: 0.9, // Default confidence
			}
			changeSet.Modified[symbol.Id] = symMod
		}
	}

	// Detect renames (symbols that appear removed and added with similar structure)
	d.detectRenames(changeSet)

	return changeSet, nil
}

// ComputeImpact computes the impact of symbol changes
func (d *ASTDiffer) ComputeImpact(changes *SymbolChangeSet) (*ImpactGraph, error) {
	impactGraph := &ImpactGraph{
		Nodes:     make(map[string]*ImpactNode),
		Edges:     make(map[string]*ImpactEdge),
		RiskScore: 0.0,
	}

	// Analyze impact of added symbols
	for _, symbol := range changes.Added {
		node := &ImpactNode{
			ID:     string(symbol.Id),
			Type:   "symbol_add",
			Impact: ImpactLow, // New symbols typically have low impact
			Risk:   0.2,
		}
		impactGraph.Nodes[node.ID] = node
	}

	// Analyze impact of removed symbols
	for _, symbol := range changes.Removed {
		node := &ImpactNode{
			ID:     string(symbol.Id),
			Type:   "symbol_remove",
			Impact: ImpactHigh, // Removed symbols typically have high impact
			Risk:   0.8,
		}
		impactGraph.Nodes[node.ID] = node
		impactGraph.RiskScore += 0.3 // Increase overall risk
	}

	// Analyze impact of modified symbols
	for _, modification := range changes.Modified {
		node := &ImpactNode{
			ID:     string(modification.Symbol.Id),
			Type:   "symbol_modify",
			Impact: modification.Impact,
			Risk:   d.calculateModificationRisk(modification),
		}
		impactGraph.Nodes[node.ID] = node
		impactGraph.RiskScore += node.Risk * 0.1
	}

	return impactGraph, nil
}

// ImpactGraph represents the impact graph of changes
type ImpactGraph struct {
	Nodes     map[string]*ImpactNode `json:"nodes"`
	Edges     map[string]*ImpactEdge `json:"edges"`
	RiskScore float64                `json:"risk_score"`
}

// ImpactNode represents a node in the impact graph
type ImpactNode struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Impact   ImpactLevel `json:"impact"`
	Risk     float64     `json:"risk"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ImpactEdge represents an edge in the impact graph
type ImpactEdge struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Type     string  `json:"type"`
	Strength float64 `json:"strength"`
}

// myersDiff implements the Myers diff algorithm
func (d *ASTDiffer) myersDiff(oldNodes, newNodes []ASTNode, diff *ASTDiff) error {
	// Simplified Myers algorithm implementation
	// In production, this would be a full implementation of the Myers algorithm
	
	oldSet := make(map[string]ASTNode)
	newSet := make(map[string]ASTNode)

	// Build sets for comparison
	for _, node := range oldNodes {
		oldSet[node.ID] = node
	}
	for _, node := range newNodes {
		newSet[node.ID] = node
	}

	// Find additions
	for id, node := range newSet {
		if _, exists := oldSet[id]; !exists {
			diff.Additions = append(diff.Additions, node)
		}
	}

	// Find deletions
	for id, node := range oldSet {
		if _, exists := newSet[id]; !exists {
			diff.Deletions = append(diff.Deletions, node)
		}
	}

	// Find modifications
	for id, newNode := range newSet {
		if oldNode, exists := oldSet[id]; exists {
			if d.nodesAreDifferent(oldNode, newNode) {
				modification := ASTModification{
					NodeID:   id,
					Type:     ModificationValueChange,
					OldValue: oldNode.Value,
					NewValue: newNode.Value,
					Impact:   d.calculateImpactLevel(oldNode, newNode),
				}
				diff.Modifications = append(diff.Modifications, modification)
			}
		}
	}

	return nil
}

// patienceDiff implements the Patience diff algorithm
func (d *ASTDiffer) patienceDiff(oldNodes, newNodes []ASTNode, diff *ASTDiff) error {
	// Placeholder for Patience algorithm
	// Would implement patience-specific logic here
	return d.myersDiff(oldNodes, newNodes, diff)
}

// histogramDiff implements the Histogram diff algorithm
func (d *ASTDiffer) histogramDiff(oldNodes, newNodes []ASTNode, diff *ASTDiff) error {
	// Placeholder for Histogram algorithm
	// Would implement histogram-specific logic here
	return d.myersDiff(oldNodes, newNodes, diff)
}

// Helper functions

func (d *ASTDiffer) convertAST(ast *types.AST) ([]ASTNode, error) {
	// Convert AST to internal format
	// This is a simplified implementation
	nodes := make([]ASTNode, 0)
	
	// In a real implementation, we would traverse the Tree-sitter AST
	// and convert each node to our internal format
	
	return nodes, nil
}

func (d *ASTDiffer) generateCacheKey(oldAST, newAST *types.AST) string {
	return fmt.Sprintf("%s:%s->%s", oldAST.FilePath, oldAST.Version, newAST.Version)
}

func (d *ASTDiffer) calculateSimilarity(oldNodes, newNodes []ASTNode) float64 {
	if len(oldNodes) == 0 && len(newNodes) == 0 {
		return 1.0
	}
	if len(oldNodes) == 0 || len(newNodes) == 0 {
		return 0.0
	}

	// Simple similarity calculation based on node overlap
	oldSet := make(map[string]bool)
	for _, node := range oldNodes {
		oldSet[node.ID] = true
	}

	overlap := 0
	for _, node := range newNodes {
		if oldSet[node.ID] {
			overlap++
		}
	}

	total := len(oldNodes) + len(newNodes) - overlap
	return float64(overlap) / float64(total)
}

func (d *ASTDiffer) hasStructuralChanges(diff *ASTDiff) bool {
	return len(diff.Additions) > 0 || len(diff.Deletions) > 0 || 
		   len(diff.Modifications) > 0
}

func (d *ASTDiffer) generateDiffHash(diff *ASTDiff) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s:%d:%d:%d", 
		diff.FileID, len(diff.Additions), len(diff.Deletions), len(diff.Modifications))))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func (d *ASTDiffer) computeImpactAnalysis(diff *ASTDiff) (*ImpactAnalysis, error) {
	impact := &ImpactAnalysis{
		AffectedFiles:   []string{diff.FileID},
		AffectedSymbols: make([]types.SymbolId, 0),
		RiskScore:       d.calculateRiskScore(diff),
		Recommendations: make([]string, 0),
	}

	// Build propagation tree
	impact.PropagationTree = &PropagationNode{
		ID:          diff.FileID,
		Type:        "file",
		Impact:      ImpactMedium,
		Children:    make([]*PropagationNode, 0),
		Probability: 1.0,
	}

	// Add recommendations based on changes
	if len(diff.Additions) > 0 {
		impact.Recommendations = append(impact.Recommendations, "Review new symbols for compatibility")
	}
	if len(diff.Deletions) > 0 {
		impact.Recommendations = append(impact.Recommendations, "Check for breaking changes from removed symbols")
	}
	if len(diff.Modifications) > 0 {
		impact.Recommendations = append(impact.Recommendations, "Validate modified symbols maintain contract compatibility")
	}

	return impact, nil
}

func (d *ASTDiffer) calculateRiskScore(diff *ASTDiff) float64 {
	score := 0.0
	score += float64(len(diff.Deletions)) * 0.5    // Deletions are risky
	score += float64(len(diff.Modifications)) * 0.3 // Modifications are moderately risky
	score += float64(len(diff.Additions)) * 0.1     // Additions are less risky

	return score
}

func (d *ASTDiffer) nodesAreDifferent(oldNode, newNode ASTNode) bool {
	return oldNode.Value != newNode.Value || oldNode.Type != newNode.Type
}

func (d *ASTDiffer) calculateImpactLevel(oldNode, newNode ASTNode) ImpactLevel {
	// Simple heuristic for impact level
	if oldNode.Type != newNode.Type {
		return ImpactHigh
	}
	if len(oldNode.Value) > 0 && len(newNode.Value) > 0 {
		return ImpactMedium
	}
	return ImpactLow
}

func (d *ASTDiffer) extractSymbolFromNode(node ASTNode) *types.Symbol {
	// Extract symbol information from AST node
	// This would analyze the node and create a symbol if applicable
	return nil
}

func (d *ASTDiffer) extractSymbolFromModification(mod ASTModification) *types.Symbol {
	// Extract symbol information from modification
	// This would analyze the modification and create a symbol if applicable
	return nil
}

func (d *ASTDiffer) detectRenames(changeSet *SymbolChangeSet) {
	// Detect potential renames by analyzing removed and added symbols
	// Look for symbols with similar structure but different names
	for removedId, removedSymbol := range changeSet.Removed {
		for addedId, addedSymbol := range changeSet.Added {
			if d.symbolsAreSimilar(removedSymbol, addedSymbol) {
				renameInfo := &RenameInfo{
					OldName:    removedSymbol.Name,
					NewName:    addedSymbol.Name,
					Confidence: 0.8,
				}
				changeSet.Renamed[removedId] = renameInfo
				
				// Remove from added/removed as they are now considered renamed
				delete(changeSet.Removed, removedId)
				delete(changeSet.Added, addedId)
			}
		}
	}
}

func (d *ASTDiffer) symbolsAreSimilar(sym1, sym2 *types.Symbol) bool {
	// Check if two symbols are similar (potential rename)
	return sym1.Type == sym2.Type && 
		   sym1.Location.StartLine == sym2.Location.StartLine &&
		   strings.Contains(sym1.Signature, sym2.Signature[:min(len(sym2.Signature), 10)])
}

func (d *ASTDiffer) calculateModificationRisk(mod *SymbolModification) float64 {
	risk := 0.0
	
	switch mod.Impact {
	case ImpactCritical:
		risk = 0.9
	case ImpactHigh:
		risk = 0.7
	case ImpactMedium:
		risk = 0.4
	case ImpactLow:
		risk = 0.1
	}
	
	return risk * mod.Confidence
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}