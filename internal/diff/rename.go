package diff

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

// RenameDetector detects symbol renames using advanced similarity algorithms
type RenameDetector struct {
	config      *Config
	algorithms  []SimilarityAlgorithm
	heuristics  []RenameHeuristic
	cache       map[string]*RenameCandidate
}

// SimilarityAlgorithm defines an interface for similarity calculation
type SimilarityAlgorithm interface {
	CalculateSimilarity(old, new *types.Symbol) SimilarityScore
	GetWeight() float64
	GetName() string
}

// RenameHeuristic defines rules for detecting renames
type RenameHeuristic interface {
	EvaluateRename(old, new *types.Symbol, context *RenameContext) HeuristicScore
	GetWeight() float64
	GetName() string
}

// SimilarityScore represents a similarity score from an algorithm
type SimilarityScore struct {
	Score       float64 `json:"score"`        // 0.0 to 1.0
	Confidence  float64 `json:"confidence"`   // 0.0 to 1.0
	Evidence    string  `json:"evidence"`     // Description of evidence
	Algorithm   string  `json:"algorithm"`    // Algorithm that produced this score
}

// HeuristicScore represents a score from a heuristic rule
type HeuristicScore struct {
	Score      float64 `json:"score"`       // 0.0 to 1.0
	Confidence float64 `json:"confidence"`  // 0.0 to 1.0
	Reason     string  `json:"reason"`      // Reason for this score
	Heuristic  string  `json:"heuristic"`   // Heuristic that produced this score
}

// RenameCandidate represents a potential rename match
type RenameCandidate struct {
	OldSymbol          *types.Symbol      `json:"old_symbol"`
	NewSymbol          *types.Symbol      `json:"new_symbol"`
	OverallScore       float64            `json:"overall_score"`
	Confidence         float64            `json:"confidence"`
	SimilarityScores   []SimilarityScore  `json:"similarity_scores"`
	HeuristicScores    []HeuristicScore   `json:"heuristic_scores"`
	Evidence           []string           `json:"evidence"`
	RenameType         RenameType         `json:"rename_type"`
	Risk               RiskLevel          `json:"risk"`
}

// RenameContext provides context for rename detection
type RenameContext struct {
	OldFile       *types.FileInfo
	NewFile       *types.FileInfo
	DeletedSymbols []*types.Symbol
	AddedSymbols   []*types.Symbol
	ModifiedSymbols []*types.Symbol
}

// RenameType categorizes the type of rename
type RenameType string

const (
	RenameTypeSimple       RenameType = "simple"        // Just name changed
	RenameTypeRefactoring  RenameType = "refactoring"   // Name + structure changed
	RenameTypeSignature    RenameType = "signature"     // Name + signature changed
	RenameTypeMove         RenameType = "move"          // Moved to different scope
	RenameTypeComplex      RenameType = "complex"       // Multiple changes
)

// RiskLevel represents the risk of incorrectly identifying a rename
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// NewRenameDetector creates a new rename detector
func NewRenameDetector(config *Config) *RenameDetector {
	detector := &RenameDetector{
		config: config,
		cache:  make(map[string]*RenameCandidate),
	}

	// Initialize similarity algorithms
	detector.algorithms = []SimilarityAlgorithm{
		NewNameSimilarityAlgorithm(),
		NewSignatureSimilarityAlgorithm(),
		NewStructuralSimilarityAlgorithm(),
		NewLocationSimilarityAlgorithm(),
		NewDocumentationSimilarityAlgorithm(),
		NewSemanticSimilarityAlgorithm(),
	}

	// Initialize heuristics
	detector.heuristics = []RenameHeuristic{
		NewCamelCaseHeuristic(),
		NewPrefixSuffixHeuristic(),
		NewAbbreviationHeuristic(),
		NewRefactoringPatternHeuristic(),
		NewContextualHeuristic(),
	}

	return detector
}

// DetectRenames detects symbol renames between two files
func (rd *RenameDetector) DetectRenames(ctx context.Context, oldFile, newFile *types.FileInfo) ([]Rename, error) {
	// Build rename context
	renameCtx := rd.buildRenameContext(oldFile, newFile)

	// Find potential rename candidates
	candidates := rd.findRenameCandidates(renameCtx)

	// Score and rank candidates
	scoredCandidates := rd.scoreCandidates(candidates, renameCtx)

	// Apply threshold filtering
	filteredCandidates := rd.filterCandidates(scoredCandidates)

	// Resolve conflicts (one-to-one mapping)
	finalCandidates := rd.resolveConflicts(filteredCandidates)

	// Convert to rename results
	var renames []Rename
	for _, candidate := range finalCandidates {
		rename := rd.convertToRename(candidate)
		renames = append(renames, rename)
	}

	return renames, nil
}

// buildRenameContext builds context for rename detection
func (rd *RenameDetector) buildRenameContext(oldFile, newFile *types.FileInfo) *RenameContext {
	// Create symbol maps
	oldSymbolMap := make(map[string]*types.Symbol)
	newSymbolMap := make(map[string]*types.Symbol)

	for _, symbol := range oldFile.Symbols {
		oldSymbolMap[symbol.FullyQualifiedName] = symbol
	}

	for _, symbol := range newFile.Symbols {
		newSymbolMap[symbol.FullyQualifiedName] = symbol
	}

	// Categorize symbols
	var deletedSymbols, addedSymbols, modifiedSymbols []*types.Symbol

	// Find deleted symbols
	for name, oldSymbol := range oldSymbolMap {
		if _, exists := newSymbolMap[name]; !exists {
			deletedSymbols = append(deletedSymbols, oldSymbol)
		}
	}

	// Find added symbols
	for name, newSymbol := range newSymbolMap {
		if _, exists := oldSymbolMap[name]; !exists {
			addedSymbols = append(addedSymbols, newSymbol)
		}
	}

	// Find modified symbols
	for name, newSymbol := range newSymbolMap {
		if oldSymbol, exists := oldSymbolMap[name]; exists {
			if rd.symbolsAreDifferent(oldSymbol, newSymbol) {
				modifiedSymbols = append(modifiedSymbols, newSymbol)
			}
		}
	}

	return &RenameContext{
		OldFile:         oldFile,
		NewFile:         newFile,
		DeletedSymbols:  deletedSymbols,
		AddedSymbols:    addedSymbols,
		ModifiedSymbols: modifiedSymbols,
	}
}

// findRenameCandidates finds potential rename pairs
func (rd *RenameDetector) findRenameCandidates(context *RenameContext) []*RenameCandidate {
	var candidates []*RenameCandidate

	// Compare each deleted symbol with each added symbol
	for _, deletedSymbol := range context.DeletedSymbols {
		for _, addedSymbol := range context.AddedSymbols {
			// Only consider symbols of the same kind
			if deletedSymbol.Kind == addedSymbol.Kind {
				candidate := &RenameCandidate{
					OldSymbol: deletedSymbol,
					NewSymbol: addedSymbol,
				}
				candidates = append(candidates, candidate)
			}
		}
	}

	return candidates
}

// scoreCandidates scores all rename candidates
func (rd *RenameDetector) scoreCandidates(candidates []*RenameCandidate, context *RenameContext) []*RenameCandidate {
	for _, candidate := range candidates {
		rd.scoreCandidate(candidate, context)
	}

	// Sort by overall score (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].OverallScore > candidates[j].OverallScore
	})

	return candidates
}

// scoreCandidate calculates scores for a single candidate
func (rd *RenameDetector) scoreCandidate(candidate *RenameCandidate, context *RenameContext) {
	// Calculate similarity scores
	var totalSimilarityScore, totalSimilarityWeight float64
	for _, algorithm := range rd.algorithms {
		score := algorithm.CalculateSimilarity(candidate.OldSymbol, candidate.NewSymbol)
		candidate.SimilarityScores = append(candidate.SimilarityScores, score)
		
		weight := algorithm.GetWeight()
		totalSimilarityScore += score.Score * weight
		totalSimilarityWeight += weight
	}

	// Calculate heuristic scores
	var totalHeuristicScore, totalHeuristicWeight float64
	for _, heuristic := range rd.heuristics {
		score := heuristic.EvaluateRename(candidate.OldSymbol, candidate.NewSymbol, context)
		candidate.HeuristicScores = append(candidate.HeuristicScores, score)
		
		weight := heuristic.GetWeight()
		totalHeuristicScore += score.Score * weight
		totalHeuristicWeight += weight
	}

	// Calculate overall score (weighted average)
	similarityScore := totalSimilarityScore / totalSimilarityWeight
	heuristicScore := totalHeuristicScore / totalHeuristicWeight
	
	// Combine scores (60% similarity, 40% heuristics)
	candidate.OverallScore = similarityScore*0.6 + heuristicScore*0.4

	// Calculate confidence
	candidate.Confidence = rd.calculateConfidence(candidate)

	// Determine rename type
	candidate.RenameType = rd.determineRenameType(candidate)

	// Assess risk
	candidate.Risk = rd.assessRisk(candidate)

	// Collect evidence
	candidate.Evidence = rd.collectEvidence(candidate)
}

// filterCandidates filters candidates based on threshold
func (rd *RenameDetector) filterCandidates(candidates []*RenameCandidate) []*RenameCandidate {
	var filtered []*RenameCandidate

	for _, candidate := range candidates {
		if candidate.OverallScore >= rd.config.RenameThreshold &&
			candidate.Confidence >= 0.5 {
			filtered = append(filtered, candidate)
		}
	}

	return filtered
}

// resolveConflicts ensures one-to-one mapping between old and new symbols
func (rd *RenameDetector) resolveConflicts(candidates []*RenameCandidate) []*RenameCandidate {
	usedOldSymbols := make(map[string]bool)
	usedNewSymbols := make(map[string]bool)
	var resolved []*RenameCandidate

	// Process candidates in order of score (highest first)
	for _, candidate := range candidates {
		oldKey := candidate.OldSymbol.FullyQualifiedName
		newKey := candidate.NewSymbol.FullyQualifiedName

		// Check if either symbol is already used
		if !usedOldSymbols[oldKey] && !usedNewSymbols[newKey] {
			resolved = append(resolved, candidate)
			usedOldSymbols[oldKey] = true
			usedNewSymbols[newKey] = true
		}
	}

	return resolved
}

// convertToRename converts a candidate to a Rename result
func (rd *RenameDetector) convertToRename(candidate *RenameCandidate) Rename {
	return Rename{
		Change: Change{
			Type:     ChangeTypeRename,
			Path:     candidate.NewSymbol.FullyQualifiedName,
			OldValue: candidate.OldSymbol.Name,
			NewValue: candidate.NewSymbol.Name,
			Position: Position{
				Line:   candidate.NewSymbol.Location.StartLine,
				Column: candidate.NewSymbol.Location.StartColumn,
			},
			Impact: rd.calculateRenameImpact(candidate),
			Context: ChangeContext{
				Function: candidate.NewSymbol.Name,
				Scope:    candidate.NewSymbol.Visibility,
				Tags:     []string{"rename", string(candidate.RenameType)},
			},
			Metadata: map[string]interface{}{
				"rename_type":       string(candidate.RenameType),
				"risk_level":        string(candidate.Risk),
				"similarity_scores": candidate.SimilarityScores,
				"heuristic_scores":  candidate.HeuristicScores,
				"evidence":          candidate.Evidence,
			},
		},
		OldName:    candidate.OldSymbol.Name,
		NewName:    candidate.NewSymbol.Name,
		Symbol:     candidate.NewSymbol,
		Confidence: candidate.Confidence,
		Reason:     rd.generateRenameReason(candidate),
	}
}

// Helper methods

func (rd *RenameDetector) symbolsAreDifferent(old, new *types.Symbol) bool {
	return old.Signature != new.Signature ||
		old.Documentation != new.Documentation ||
		old.Visibility != new.Visibility ||
		old.Location.StartLine != new.Location.StartLine
}

func (rd *RenameDetector) calculateConfidence(candidate *RenameCandidate) float64 {
	// Base confidence on consistency of scores
	var scores []float64
	for _, score := range candidate.SimilarityScores {
		scores = append(scores, score.Score)
	}
	for _, score := range candidate.HeuristicScores {
		scores = append(scores, score.Score)
	}

	if len(scores) == 0 {
		return 0.0
	}

	// Calculate variance to determine consistency
	mean := rd.calculateMean(scores)
	variance := rd.calculateVariance(scores, mean)
	
	// Lower variance = higher confidence
	confidence := 1.0 - math.Min(variance, 1.0)
	
	return confidence
}

func (rd *RenameDetector) determineRenameType(candidate *RenameCandidate) RenameType {
	old := candidate.OldSymbol
	new := candidate.NewSymbol

	// Check for signature changes
	if old.Signature != new.Signature {
		return RenameTypeSignature
	}

	// Check for location changes (move)
	if old.Location.StartLine != new.Location.StartLine {
		lineDiff := abs(old.Location.StartLine - new.Location.StartLine)
		if lineDiff > 10 {
			return RenameTypeMove
		}
	}

	// Check for structural changes
	oldSize := old.Location.EndLine - old.Location.StartLine
	newSize := new.Location.EndLine - new.Location.StartLine
	if abs(oldSize-newSize) > 5 {
		return RenameTypeRefactoring
	}

	// Check for complex changes
	changeCount := 0
	if old.Documentation != new.Documentation {
		changeCount++
	}
	if old.Visibility != new.Visibility {
		changeCount++
	}
	if changeCount > 1 {
		return RenameTypeComplex
	}

	return RenameTypeSimple
}

func (rd *RenameDetector) assessRisk(candidate *RenameCandidate) RiskLevel {
	// High risk if low confidence or low overall score
	if candidate.Confidence < 0.6 || candidate.OverallScore < 0.7 {
		return RiskHigh
	}

	// Medium risk if some uncertainty
	if candidate.Confidence < 0.8 || candidate.OverallScore < 0.85 {
		return RiskMedium
	}

	return RiskLow
}

func (rd *RenameDetector) collectEvidence(candidate *RenameCandidate) []string {
	var evidence []string

	// Collect evidence from high-scoring algorithms
	for _, score := range candidate.SimilarityScores {
		if score.Score > 0.7 {
			evidence = append(evidence, fmt.Sprintf("%s: %s", score.Algorithm, score.Evidence))
		}
	}

	// Collect evidence from heuristics
	for _, score := range candidate.HeuristicScores {
		if score.Score > 0.7 {
			evidence = append(evidence, fmt.Sprintf("%s: %s", score.Heuristic, score.Reason))
		}
	}

	return evidence
}

func (rd *RenameDetector) calculateRenameImpact(candidate *RenameCandidate) ImpactLevel {
	symbol := candidate.NewSymbol

	if symbol.Visibility == "public" {
		return ImpactHigh
	}
	if symbol.Visibility == "protected" {
		return ImpactMedium
	}
	return ImpactLow
}

func (rd *RenameDetector) generateRenameReason(candidate *RenameCandidate) string {
	// Generate a human-readable reason for the rename
	reasons := []string{}

	// Check for high-scoring evidence
	for _, score := range candidate.SimilarityScores {
		if score.Score > 0.8 {
			reasons = append(reasons, strings.ToLower(score.Algorithm))
		}
	}

	if len(reasons) == 0 {
		return "similarity_match"
	}

	return strings.Join(reasons, "_")
}

// Utility functions

func (rd *RenameDetector) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, value := range values {
		sum += value
	}

	return sum / float64(len(values))
}

func (rd *RenameDetector) calculateVariance(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sumSquaredDiffs := 0.0
	for _, value := range values {
		diff := value - mean
		sumSquaredDiffs += diff * diff
	}

	return sumSquaredDiffs / float64(len(values))
}

// abs function moved to utils.go