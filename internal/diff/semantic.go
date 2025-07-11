package diff

import (
	"context"
	"fmt"
	"strings"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

// SemanticDiffer performs semantic-level diff analysis
type SemanticDiffer struct {
	config *Config
}

// SemanticChange represents a semantic-level change
type SemanticChange struct {
	Type        SemanticChangeType `json:"type"`
	Description string             `json:"description"`
	Impact      ImpactLevel        `json:"impact"`
	Reason      string             `json:"reason"`
	Context     SemanticContext    `json:"context"`
	Confidence  float64            `json:"confidence"`
}

// SemanticChangeType represents types of semantic changes
type SemanticChangeType string

const (
	SemanticChangeBreaking      SemanticChangeType = "breaking"
	SemanticChangeNonBreaking   SemanticChangeType = "non_breaking"
	SemanticChangeRefactoring   SemanticChangeType = "refactoring"
	SemanticChangeBehavioral    SemanticChangeType = "behavioral"
	SemanticChangePerformance   SemanticChangeType = "performance"
	SemanticChangeDocumentation SemanticChangeType = "documentation"
)

// SemanticContext provides context for semantic changes
type SemanticContext struct {
	AffectedAPIs     []string `json:"affected_apis"`
	AffectedClients  []string `json:"affected_clients"`
	MigrationPath    string   `json:"migration_path"`
	RiskLevel        string   `json:"risk_level"`
	TestsRequired    bool     `json:"tests_required"`
}

// NewSemanticDiffer creates a new semantic differ
func NewSemanticDiffer(config *Config) *SemanticDiffer {
	return &SemanticDiffer{
		config: config,
	}
}

// Compare performs semantic diff analysis between two files
func (sd *SemanticDiffer) Compare(ctx context.Context, oldFile, newFile *types.FileInfo) ([]Change, error) {
	var changes []Change

	// Analyze symbol-level semantic changes
	symbolChanges, err := sd.compareSymbols(ctx, oldFile, newFile)
	if err != nil {
		return nil, fmt.Errorf("symbol comparison failed: %w", err)
	}
	changes = append(changes, symbolChanges...)

	// Analyze API contract changes
	apiChanges, err := sd.compareAPIContracts(ctx, oldFile, newFile)
	if err != nil {
		return nil, fmt.Errorf("API contract comparison failed: %w", err)
	}
	changes = append(changes, apiChanges...)

	// Analyze behavioral changes
	behaviorChanges, err := sd.compareBehavior(ctx, oldFile, newFile)
	if err != nil {
		return nil, fmt.Errorf("behavior comparison failed: %w", err)
	}
	changes = append(changes, behaviorChanges...)

	return changes, nil
}

// compareSymbols analyzes semantic changes at the symbol level
func (sd *SemanticDiffer) compareSymbols(ctx context.Context, oldFile, newFile *types.FileInfo) ([]Change, error) {
	var changes []Change

	// Create symbol maps for efficient lookup
	oldSymbols := make(map[string]*types.Symbol)
	newSymbols := make(map[string]*types.Symbol)

	for _, symbol := range oldFile.Symbols {
		oldSymbols[symbol.FullyQualifiedName] = symbol
	}

	for _, symbol := range newFile.Symbols {
		newSymbols[symbol.FullyQualifiedName] = symbol
	}

	// Find added symbols
	for name, newSymbol := range newSymbols {
		if _, exists := oldSymbols[name]; !exists {
			change := Change{
				Type:     ChangeTypeAdd,
				Path:     name,
				NewValue: newSymbol,
				Position: Position{
					Line:   newSymbol.Location.StartLine,
					Column: newSymbol.Location.StartColumn,
				},
				Impact: sd.assessAdditionImpact(newSymbol),
				Context: sd.buildSymbolContext(newSymbol, newFile),
				Metadata: map[string]interface{}{
					"semantic_type": "symbol_addition",
					"symbol_kind":   newSymbol.Kind,
					"visibility":    newSymbol.Visibility,
				},
			}
			changes = append(changes, change)
		}
	}

	// Find deleted symbols
	for name, oldSymbol := range oldSymbols {
		if _, exists := newSymbols[name]; !exists {
			change := Change{
				Type:     ChangeTypeDelete,
				Path:     name,
				OldValue: oldSymbol,
				Position: Position{
					Line:   oldSymbol.Location.StartLine,
					Column: oldSymbol.Location.StartColumn,
				},
				Impact: sd.assessDeletionImpact(oldSymbol),
				Context: sd.buildSymbolContext(oldSymbol, oldFile),
				Metadata: map[string]interface{}{
					"semantic_type": "symbol_deletion",
					"symbol_kind":   oldSymbol.Kind,
					"visibility":    oldSymbol.Visibility,
				},
			}
			changes = append(changes, change)
		}
	}

	// Find modified symbols
	for name, newSymbol := range newSymbols {
		if oldSymbol, exists := oldSymbols[name]; exists {
			symbolChanges := sd.compareSymbolSemantics(oldSymbol, newSymbol)
			changes = append(changes, symbolChanges...)
		}
	}

	return changes, nil
}

// compareAPIContracts analyzes changes in API contracts
func (sd *SemanticDiffer) compareAPIContracts(ctx context.Context, oldFile, newFile *types.FileInfo) ([]Change, error) {
	var changes []Change

	// Analyze public interface changes
	publicChanges := sd.analyzePublicInterfaceChanges(oldFile, newFile)
	changes = append(changes, publicChanges...)

	// Analyze function signature changes
	signatureChanges := sd.analyzeFunctionSignatureChanges(oldFile, newFile)
	changes = append(changes, signatureChanges...)

	// Analyze type definition changes
	typeChanges := sd.analyzeTypeDefinitionChanges(oldFile, newFile)
	changes = append(changes, typeChanges...)

	return changes, nil
}

// compareBehavior analyzes potential behavioral changes
func (sd *SemanticDiffer) compareBehavior(ctx context.Context, oldFile, newFile *types.FileInfo) ([]Change, error) {
	var changes []Change

	// Analyze control flow changes
	controlFlowChanges := sd.analyzeControlFlowChanges(oldFile, newFile)
	changes = append(changes, controlFlowChanges...)

	// Analyze error handling changes
	errorHandlingChanges := sd.analyzeErrorHandlingChanges(oldFile, newFile)
	changes = append(changes, errorHandlingChanges...)

	// Analyze performance implications
	performanceChanges := sd.analyzePerformanceImplications(oldFile, newFile)
	changes = append(changes, performanceChanges...)

	return changes, nil
}

// compareSymbolSemantics compares the semantic aspects of two symbols
func (sd *SemanticDiffer) compareSymbolSemantics(oldSymbol, newSymbol *types.Symbol) []Change {
	var changes []Change

	// Compare signatures semantically
	if oldSymbol.Signature != newSymbol.Signature {
		impact := sd.assessSignatureChangeImpact(oldSymbol, newSymbol)
		change := Change{
			Type:     ChangeTypeModify,
			Path:     newSymbol.FullyQualifiedName + ".signature",
			OldValue: oldSymbol.Signature,
			NewValue: newSymbol.Signature,
			Position: Position{
				Line:   newSymbol.Location.StartLine,
				Column: newSymbol.Location.StartColumn,
			},
			Impact: impact,
			Context: sd.buildSignatureChangeContext(oldSymbol, newSymbol),
			Metadata: map[string]interface{}{
				"semantic_type":    "signature_change",
				"breaking_change":  sd.isBreakingSignatureChange(oldSymbol, newSymbol),
				"parameter_count":  sd.countParameters(newSymbol.Signature),
				"return_type":      sd.extractReturnType(newSymbol.Signature),
			},
		}
		changes = append(changes, change)
	}

	// Compare visibility changes
	if oldSymbol.Visibility != newSymbol.Visibility {
		impact := sd.assessVisibilityChangeImpact(oldSymbol, newSymbol)
		change := Change{
			Type:     ChangeTypeModify,
			Path:     newSymbol.FullyQualifiedName + ".visibility",
			OldValue: oldSymbol.Visibility,
			NewValue: newSymbol.Visibility,
			Position: Position{
				Line:   newSymbol.Location.StartLine,
				Column: newSymbol.Location.StartColumn,
			},
			Impact: impact,
			Context: sd.buildVisibilityChangeContext(oldSymbol, newSymbol),
			Metadata: map[string]interface{}{
				"semantic_type":   "visibility_change",
				"breaking_change": sd.isBreakingVisibilityChange(oldSymbol, newSymbol),
			},
		}
		changes = append(changes, change)
	}

	// Compare documentation changes (low impact but important for API understanding)
	if oldSymbol.Documentation != newSymbol.Documentation {
		change := Change{
			Type:     ChangeTypeModify,
			Path:     newSymbol.FullyQualifiedName + ".documentation",
			OldValue: oldSymbol.Documentation,
			NewValue: newSymbol.Documentation,
			Position: Position{
				Line:   newSymbol.Location.StartLine,
				Column: newSymbol.Location.StartColumn,
			},
			Impact: ImpactLow,
			Context: sd.buildDocumentationChangeContext(oldSymbol, newSymbol),
			Metadata: map[string]interface{}{
				"semantic_type": "documentation_change",
				"doc_quality":   sd.assessDocumentationQuality(newSymbol.Documentation),
			},
		}
		changes = append(changes, change)
	}

	return changes
}

// assessAdditionImpact assesses the impact of adding a new symbol
func (sd *SemanticDiffer) assessAdditionImpact(symbol *types.Symbol) ImpactLevel {
	if symbol.Visibility == "public" {
		switch symbol.Kind {
		case "function", "method":
			return ImpactMedium
		case "class", "interface", "type":
			return ImpactHigh
		default:
			return ImpactLow
		}
	}
	return ImpactLow
}

// assessDeletionImpact assesses the impact of deleting a symbol
func (sd *SemanticDiffer) assessDeletionImpact(symbol *types.Symbol) ImpactLevel {
	if symbol.Visibility == "public" {
		return ImpactCritical
	}
	if symbol.Visibility == "protected" {
		return ImpactHigh
	}
	return ImpactMedium
}

// assessSignatureChangeImpact assesses the impact of signature changes
func (sd *SemanticDiffer) assessSignatureChangeImpact(oldSymbol, newSymbol *types.Symbol) ImpactLevel {
	if newSymbol.Visibility == "public" || newSymbol.Visibility == "protected" {
		if sd.isBreakingSignatureChange(oldSymbol, newSymbol) {
			return ImpactCritical
		}
		return ImpactHigh
	}
	return ImpactMedium
}

// assessVisibilityChangeImpact assesses the impact of visibility changes
func (sd *SemanticDiffer) assessVisibilityChangeImpact(oldSymbol, newSymbol *types.Symbol) ImpactLevel {
	if sd.isBreakingVisibilityChange(oldSymbol, newSymbol) {
		return ImpactCritical
	}
	
	if oldSymbol.Visibility == "private" && newSymbol.Visibility == "public" {
		return ImpactMedium // Exposing internals
	}
	
	return ImpactLow
}

// isBreakingSignatureChange determines if a signature change is breaking
func (sd *SemanticDiffer) isBreakingSignatureChange(oldSymbol, newSymbol *types.Symbol) bool {
	oldParamCount := sd.countParameters(oldSymbol.Signature)
	newParamCount := sd.countParameters(newSymbol.Signature)
	
	// Removing parameters is always breaking
	if newParamCount < oldParamCount {
		return true
	}
	
	// Adding required parameters is breaking
	if newParamCount > oldParamCount && !sd.hasDefaultParameters(newSymbol.Signature) {
		return true
	}
	
	// Return type changes can be breaking
	oldReturnType := sd.extractReturnType(oldSymbol.Signature)
	newReturnType := sd.extractReturnType(newSymbol.Signature)
	if oldReturnType != newReturnType {
		return true
	}
	
	return false
}

// isBreakingVisibilityChange determines if a visibility change is breaking
func (sd *SemanticDiffer) isBreakingVisibilityChange(oldSymbol, newSymbol *types.Symbol) bool {
	visibilityOrder := map[string]int{
		"private":   0,
		"protected": 1,
		"public":    2,
	}
	
	oldLevel, okOld := visibilityOrder[oldSymbol.Visibility]
	newLevel, okNew := visibilityOrder[newSymbol.Visibility]
	
	if !okOld || !okNew {
		return false
	}
	
	// Reducing visibility is breaking
	return newLevel < oldLevel
}

// Helper methods for building context

func (sd *SemanticDiffer) buildSymbolContext(symbol *types.Symbol, file *types.FileInfo) ChangeContext {
	return ChangeContext{
		Function: sd.findContainingFunction(symbol, file),
		Class:    sd.findContainingClass(symbol, file),
		Module:   file.Path,
		Scope:    symbol.Visibility,
		Tags:     []string{symbol.Kind, symbol.Visibility},
	}
}

func (sd *SemanticDiffer) buildSignatureChangeContext(oldSymbol, newSymbol *types.Symbol) ChangeContext {
	context := ChangeContext{
		Function: newSymbol.Name,
		Scope:    newSymbol.Visibility,
		Tags:     []string{"signature_change"},
	}
	
	if sd.isBreakingSignatureChange(oldSymbol, newSymbol) {
		context.Tags = append(context.Tags, "breaking_change")
	}
	
	return context
}

func (sd *SemanticDiffer) buildVisibilityChangeContext(oldSymbol, newSymbol *types.Symbol) ChangeContext {
	context := ChangeContext{
		Function: newSymbol.Name,
		Scope:    fmt.Sprintf("%s->%s", oldSymbol.Visibility, newSymbol.Visibility),
		Tags:     []string{"visibility_change"},
	}
	
	if sd.isBreakingVisibilityChange(oldSymbol, newSymbol) {
		context.Tags = append(context.Tags, "breaking_change")
	}
	
	return context
}

func (sd *SemanticDiffer) buildDocumentationChangeContext(oldSymbol, newSymbol *types.Symbol) ChangeContext {
	return ChangeContext{
		Function: newSymbol.Name,
		Scope:    newSymbol.Visibility,
		Tags:     []string{"documentation_change"},
	}
}

// Helper methods for signature analysis

func (sd *SemanticDiffer) countParameters(signature string) int {
	// Simple parameter counting (language-agnostic)
	if !strings.Contains(signature, "(") {
		return 0
	}
	
	paramSection := strings.Split(signature, "(")[1]
	if !strings.Contains(paramSection, ")") {
		return 0
	}
	
	paramSection = strings.Split(paramSection, ")")[0]
	if strings.TrimSpace(paramSection) == "" {
		return 0
	}
	
	return strings.Count(paramSection, ",") + 1
}

func (sd *SemanticDiffer) extractReturnType(signature string) string {
	// Simple return type extraction (would need language-specific logic)
	if strings.Contains(signature, "->") {
		parts := strings.Split(signature, "->")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	
	if strings.Contains(signature, ":") {
		parts := strings.Split(signature, ":")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	
	return "unknown"
}

func (sd *SemanticDiffer) hasDefaultParameters(signature string) bool {
	// Simple check for default parameters
	return strings.Contains(signature, "=") || strings.Contains(signature, "?")
}

func (sd *SemanticDiffer) assessDocumentationQuality(doc string) string {
	if doc == "" {
		return "none"
	}
	
	wordCount := len(strings.Fields(doc))
	if wordCount < 5 {
		return "minimal"
	} else if wordCount < 20 {
		return "basic"
	} else {
		return "comprehensive"
	}
}

func (sd *SemanticDiffer) findContainingFunction(symbol *types.Symbol, file *types.FileInfo) string {
	// Find the function that contains this symbol
	for _, s := range file.Symbols {
		if s.Kind == "function" || s.Kind == "method" {
			if symbol.Location.StartLine >= s.Location.StartLine &&
				symbol.Location.EndLine <= s.Location.EndLine {
				return s.Name
			}
		}
	}
	return ""
}

func (sd *SemanticDiffer) findContainingClass(symbol *types.Symbol, file *types.FileInfo) string {
	// Find the class that contains this symbol
	for _, s := range file.Symbols {
		if s.Kind == "class" || s.Kind == "interface" {
			if symbol.Location.StartLine >= s.Location.StartLine &&
				symbol.Location.EndLine <= s.Location.EndLine {
				return s.Name
			}
		}
	}
	return ""
}

// Placeholder methods for advanced analysis (would need more sophisticated implementation)

func (sd *SemanticDiffer) analyzePublicInterfaceChanges(oldFile, newFile *types.FileInfo) []Change {
	// Placeholder for public interface analysis
	return []Change{}
}

func (sd *SemanticDiffer) analyzeFunctionSignatureChanges(oldFile, newFile *types.FileInfo) []Change {
	// Placeholder for function signature analysis
	return []Change{}
}

func (sd *SemanticDiffer) analyzeTypeDefinitionChanges(oldFile, newFile *types.FileInfo) []Change {
	// Placeholder for type definition analysis
	return []Change{}
}

func (sd *SemanticDiffer) analyzeControlFlowChanges(oldFile, newFile *types.FileInfo) []Change {
	// Placeholder for control flow analysis
	return []Change{}
}

func (sd *SemanticDiffer) analyzeErrorHandlingChanges(oldFile, newFile *types.FileInfo) []Change {
	// Placeholder for error handling analysis
	return []Change{}
}

func (sd *SemanticDiffer) analyzePerformanceImplications(oldFile, newFile *types.FileInfo) []Change {
	// Placeholder for performance analysis
	return []Change{}
}