package diff

import (
	"context"
	"fmt"
	"strings"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

// ASTDiffer performs structural AST-level diff analysis
type ASTDiffer struct {
	config           *Config
	languageHandlers map[string]LanguageHandler
}

// LanguageHandler provides language-specific AST diffing
type LanguageHandler interface {
	DiffAST(ctx context.Context, oldAST, newAST interface{}) ([]ASTChange, error)
	GetNodeType(node interface{}) string
	GetNodeChildren(node interface{}) []interface{}
	GetNodeText(node interface{}) string
	GetNodePosition(node interface{}) Position
}

// ASTChange represents a change in the AST structure
type ASTChange struct {
	Type        ASTChangeType `json:"type"`
	NodeType    string        `json:"node_type"`
	Path        string        `json:"path"`
	OldNode     interface{}   `json:"old_node,omitempty"`
	NewNode     interface{}   `json:"new_node,omitempty"`
	Position    Position      `json:"position"`
	Impact      ImpactLevel   `json:"impact"`
	Description string        `json:"description"`
	Context     ASTContext    `json:"context"`
}

// ASTChangeType represents types of AST changes
type ASTChangeType string

const (
	ASTChangeAdd    ASTChangeType = "add"
	ASTChangeDelete ASTChangeType = "delete"
	ASTChangeModify ASTChangeType = "modify"
	ASTChangeMove   ASTChangeType = "move"
)

// ASTContext provides context for AST changes
type ASTContext struct {
	ParentType     string   `json:"parent_type"`
	SiblingTypes   []string `json:"sibling_types"`
	Depth          int      `json:"depth"`
	PathToRoot     []string `json:"path_to_root"`
	AffectedScopes []string `json:"affected_scopes"`
}

// NewASTDiffer creates a new AST differ
func NewASTDiffer(config *Config) *ASTDiffer {
	differ := &ASTDiffer{
		config:           config,
		languageHandlers: make(map[string]LanguageHandler),
	}

	// Register language handlers
	differ.registerLanguageHandlers()

	return differ
}

// Compare performs structural diff analysis between two files
func (ad *ASTDiffer) Compare(ctx context.Context, oldFile, newFile *types.FileInfo) ([]Change, error) {
	// Get language handler
	handler, exists := ad.languageHandlers[newFile.Language]
	if !exists {
		return ad.genericStructuralDiff(ctx, oldFile, newFile)
	}

	// Perform language-specific AST diff
	astChanges, err := handler.DiffAST(ctx, oldFile.AST, newFile.AST)
	if err != nil {
		return nil, fmt.Errorf("language-specific AST diff failed: %w", err)
	}

	// Convert AST changes to generic changes
	var changes []Change
	for _, astChange := range astChanges {
		change := ad.convertASTChange(astChange)
		changes = append(changes, change)
	}

	return changes, nil
}

// registerLanguageHandlers registers handlers for different languages
func (ad *ASTDiffer) registerLanguageHandlers() {
	ad.languageHandlers["javascript"] = NewJavaScriptHandler()
	ad.languageHandlers["typescript"] = NewTypeScriptHandler()
	ad.languageHandlers["go"] = NewGoHandler()
	ad.languageHandlers["python"] = NewPythonHandler()
}

// genericStructuralDiff performs generic structural diffing when no language handler is available
func (ad *ASTDiffer) genericStructuralDiff(ctx context.Context, oldFile, newFile *types.FileInfo) ([]Change, error) {
	var changes []Change

	// Compare symbols structurally
	symbolChanges := ad.compareSymbolsStructurally(oldFile, newFile)
	changes = append(changes, symbolChanges...)

	// Compare imports/dependencies
	importChanges := ad.compareImports(oldFile, newFile)
	changes = append(changes, importChanges...)

	// Compare file structure metrics
	structureChanges := ad.compareFileStructure(oldFile, newFile)
	changes = append(changes, structureChanges...)

	return changes, nil
}

// compareSymbolsStructurally compares symbols from a structural perspective
func (ad *ASTDiffer) compareSymbolsStructurally(oldFile, newFile *types.FileInfo) []Change {
	var changes []Change

	// Create maps for efficient lookup
	oldSymbolMap := make(map[string]*types.Symbol)
	newSymbolMap := make(map[string]*types.Symbol)

	for _, symbol := range oldFile.Symbols {
		oldSymbolMap[symbol.FullyQualifiedName] = symbol
	}

	for _, symbol := range newFile.Symbols {
		newSymbolMap[symbol.FullyQualifiedName] = symbol
	}

	// Find structural additions
	for name, newSymbol := range newSymbolMap {
		if _, exists := oldSymbolMap[name]; !exists {
			change := Change{
				Type:     ChangeTypeAdd,
				Path:     name,
				NewValue: newSymbol,
				Position: Position{
					Line:   newSymbol.Location.StartLine,
					Column: newSymbol.Location.StartColumn,
				},
				Impact:  ad.assessStructuralImpact(ChangeTypeAdd, newSymbol),
				Context: ad.buildStructuralContext(newSymbol, newFile),
				Metadata: map[string]interface{}{
					"diff_type":   "structural",
					"symbol_kind": newSymbol.Kind,
					"line_count":  newSymbol.Location.EndLine - newSymbol.Location.StartLine + 1,
					"complexity":  ad.calculateSymbolComplexity(newSymbol),
				},
			}
			changes = append(changes, change)
		}
	}

	// Find structural deletions
	for name, oldSymbol := range oldSymbolMap {
		if _, exists := newSymbolMap[name]; !exists {
			change := Change{
				Type:     ChangeTypeDelete,
				Path:     name,
				OldValue: oldSymbol,
				Position: Position{
					Line:   oldSymbol.Location.StartLine,
					Column: oldSymbol.Location.StartColumn,
				},
				Impact:  ad.assessStructuralImpact(ChangeTypeDelete, oldSymbol),
				Context: ad.buildStructuralContext(oldSymbol, oldFile),
				Metadata: map[string]interface{}{
					"diff_type":   "structural",
					"symbol_kind": oldSymbol.Kind,
					"line_count":  oldSymbol.Location.EndLine - oldSymbol.Location.StartLine + 1,
					"complexity":  ad.calculateSymbolComplexity(oldSymbol),
				},
			}
			changes = append(changes, change)
		}
	}

	// Find structural modifications
	for name, newSymbol := range newSymbolMap {
		if oldSymbol, exists := oldSymbolMap[name]; exists {
			symbolChanges := ad.compareSymbolStructure(oldSymbol, newSymbol)
			changes = append(changes, symbolChanges...)
		}
	}

	return changes
}

// compareSymbolStructure compares the structure of two symbols
func (ad *ASTDiffer) compareSymbolStructure(oldSymbol, newSymbol *types.Symbol) []Change {
	var changes []Change

	// Compare location (symbol moved)
	if oldSymbol.Location.StartLine != newSymbol.Location.StartLine {
		change := Change{
			Type: ChangeTypeMove,
			Path: newSymbol.FullyQualifiedName + ".location",
			OldValue: Position{
				Line:   oldSymbol.Location.StartLine,
				Column: oldSymbol.Location.StartColumn,
			},
			NewValue: Position{
				Line:   newSymbol.Location.StartLine,
				Column: newSymbol.Location.StartColumn,
			},
			Position: Position{
				Line:   newSymbol.Location.StartLine,
				Column: newSymbol.Location.StartColumn,
			},
			Impact:  ImpactLow,
			Context: ad.buildStructuralContext(newSymbol, nil),
			Metadata: map[string]interface{}{
				"diff_type":   "structural",
				"change_type": "location_change",
				"line_delta":  newSymbol.Location.StartLine - oldSymbol.Location.StartLine,
			},
		}
		changes = append(changes, change)
	}

	// Compare size (lines of code)
	oldSize := oldSymbol.Location.EndLine - oldSymbol.Location.StartLine + 1
	newSize := newSymbol.Location.EndLine - newSymbol.Location.StartLine + 1

	if oldSize != newSize {
		sizeDelta := newSize - oldSize
		impact := ad.assessSizeChangeImpact(sizeDelta)

		change := Change{
			Type:     ChangeTypeModify,
			Path:     newSymbol.FullyQualifiedName + ".size",
			OldValue: oldSize,
			NewValue: newSize,
			Position: Position{
				Line:   newSymbol.Location.StartLine,
				Column: newSymbol.Location.StartColumn,
			},
			Impact:  impact,
			Context: ad.buildStructuralContext(newSymbol, nil),
			Metadata: map[string]interface{}{
				"diff_type":       "structural",
				"change_type":     "size_change",
				"size_delta":      sizeDelta,
				"relative_change": float64(sizeDelta) / float64(oldSize),
			},
		}
		changes = append(changes, change)
	}

	// Compare structural complexity
	oldComplexity := ad.calculateSymbolComplexity(oldSymbol)
	newComplexity := ad.calculateSymbolComplexity(newSymbol)

	if oldComplexity != newComplexity {
		complexityDelta := newComplexity - oldComplexity
		impact := ad.assessComplexityChangeImpact(complexityDelta)

		change := Change{
			Type:     ChangeTypeModify,
			Path:     newSymbol.FullyQualifiedName + ".complexity",
			OldValue: oldComplexity,
			NewValue: newComplexity,
			Position: Position{
				Line:   newSymbol.Location.StartLine,
				Column: newSymbol.Location.StartColumn,
			},
			Impact:  impact,
			Context: ad.buildStructuralContext(newSymbol, nil),
			Metadata: map[string]interface{}{
				"diff_type":        "structural",
				"change_type":      "complexity_change",
				"complexity_delta": complexityDelta,
			},
		}
		changes = append(changes, change)
	}

	return changes
}

// compareImports compares import/dependency structures
func (ad *ASTDiffer) compareImports(oldFile, newFile *types.FileInfo) []Change {
	var changes []Change

	// Simple import comparison (would be enhanced for language-specific analysis)
	oldImports := ad.extractImports(oldFile)
	newImports := ad.extractImports(newFile)

	// Find added imports
	for _, newImport := range newImports {
		if !ad.containsImport(oldImports, newImport) {
			change := Change{
				Type:     ChangeTypeAdd,
				Path:     "imports." + newImport,
				NewValue: newImport,
				Position: Position{Line: 1}, // Imports typically at top
				Impact:   ImpactLow,
				Context: ChangeContext{
					Module: newFile.Path,
					Tags:   []string{"import", "dependency"},
				},
				Metadata: map[string]interface{}{
					"diff_type":   "structural",
					"change_type": "import_added",
					"import_path": newImport,
				},
			}
			changes = append(changes, change)
		}
	}

	// Find removed imports
	for _, oldImport := range oldImports {
		if !ad.containsImport(newImports, oldImport) {
			change := Change{
				Type:     ChangeTypeDelete,
				Path:     "imports." + oldImport,
				OldValue: oldImport,
				Position: Position{Line: 1},
				Impact:   ImpactLow,
				Context: ChangeContext{
					Module: oldFile.Path,
					Tags:   []string{"import", "dependency"},
				},
				Metadata: map[string]interface{}{
					"diff_type":   "structural",
					"change_type": "import_removed",
					"import_path": oldImport,
				},
			}
			changes = append(changes, change)
		}
	}

	return changes
}

// compareFileStructure compares overall file structure metrics
func (ad *ASTDiffer) compareFileStructure(oldFile, newFile *types.FileInfo) []Change {
	var changes []Change

	// Compare symbol counts by type
	oldCounts := ad.countSymbolsByType(oldFile)
	newCounts := ad.countSymbolsByType(newFile)

	for symbolType, newCount := range newCounts {
		oldCount := oldCounts[symbolType]
		if oldCount != newCount {
			delta := newCount - oldCount
			change := Change{
				Type:     ChangeTypeModify,
				Path:     fmt.Sprintf("structure.%s_count", symbolType),
				OldValue: oldCount,
				NewValue: newCount,
				Position: Position{Line: 1},
				Impact:   ad.assessCountChangeImpact(symbolType, delta),
				Context: ChangeContext{
					Module: newFile.Path,
					Tags:   []string{"structure", symbolType},
				},
				Metadata: map[string]interface{}{
					"diff_type":   "structural",
					"change_type": "symbol_count_change",
					"symbol_type": symbolType,
					"count_delta": delta,
				},
			}
			changes = append(changes, change)
		}
	}

	return changes
}

// Helper methods

func (ad *ASTDiffer) convertASTChange(astChange ASTChange) Change {
	return Change{
		Type:     ChangeType(astChange.Type),
		Path:     astChange.Path,
		OldValue: astChange.OldNode,
		NewValue: astChange.NewNode,
		Position: astChange.Position,
		Impact:   astChange.Impact,
		Context: ChangeContext{
			Tags: []string{"ast", astChange.NodeType},
		},
		Metadata: map[string]interface{}{
			"diff_type":   "ast",
			"node_type":   astChange.NodeType,
			"ast_depth":   astChange.Context.Depth,
			"parent_type": astChange.Context.ParentType,
		},
	}
}

func (ad *ASTDiffer) assessStructuralImpact(changeType ChangeType, symbol *types.Symbol) ImpactLevel {
	// Base impact on symbol visibility and type
	if symbol.Visibility == "public" {
		switch symbol.Kind {
		case "class", "interface", "type":
			return ImpactHigh
		case "function", "method":
			return ImpactMedium
		default:
			return ImpactLow
		}
	}

	if symbol.Visibility == "protected" {
		return ImpactMedium
	}

	return ImpactLow
}

func (ad *ASTDiffer) assessSizeChangeImpact(sizeDelta int) ImpactLevel {
	absDelta := sizeDelta
	if absDelta < 0 {
		absDelta = -sizeDelta
	}

	if absDelta > 100 {
		return ImpactHigh
	} else if absDelta > 20 {
		return ImpactMedium
	}
	return ImpactLow
}

func (ad *ASTDiffer) assessComplexityChangeImpact(complexityDelta int) ImpactLevel {
	if complexityDelta > 10 {
		return ImpactHigh
	} else if complexityDelta > 5 {
		return ImpactMedium
	}
	return ImpactLow
}

func (ad *ASTDiffer) assessCountChangeImpact(symbolType string, delta int) ImpactLevel {
	absDelta := delta
	if absDelta < 0 {
		absDelta = -delta
	}

	switch symbolType {
	case "class", "interface":
		if absDelta > 0 {
			return ImpactMedium
		}
	case "function", "method":
		if absDelta > 5 {
			return ImpactMedium
		}
	}
	return ImpactLow
}

func (ad *ASTDiffer) buildStructuralContext(symbol *types.Symbol, file *types.FileInfo) ChangeContext {
	context := ChangeContext{
		Function: symbol.Name,
		Scope:    symbol.Visibility,
		Tags:     []string{"structural", symbol.Kind},
	}

	if file != nil {
		context.Module = file.Path
	}

	return context
}

func (ad *ASTDiffer) calculateSymbolComplexity(symbol *types.Symbol) int {
	// Simple complexity calculation based on signature and documentation
	complexity := 1

	// Add complexity for parameters
	paramCount := strings.Count(symbol.Signature, ",")
	complexity += paramCount

	// Add complexity for nested structures (rough estimate)
	lineCount := symbol.Location.EndLine - symbol.Location.StartLine + 1
	complexity += lineCount / 10

	return complexity
}

func (ad *ASTDiffer) extractImports(file *types.FileInfo) []string {
	var imports []string

	// Simple heuristic: look for symbols that might be imports
	for _, symbol := range file.Symbols {
		if symbol.Kind == "import" || strings.HasPrefix(symbol.Name, "import ") {
			imports = append(imports, symbol.Name)
		}
	}

	return imports
}

func (ad *ASTDiffer) containsImport(imports []string, target string) bool {
	for _, imp := range imports {
		if imp == target {
			return true
		}
	}
	return false
}

func (ad *ASTDiffer) countSymbolsByType(file *types.FileInfo) map[string]int {
	counts := make(map[string]int)

	for _, symbol := range file.Symbols {
		counts[symbol.Kind]++
	}

	return counts
}

// Language-specific handlers (placeholder implementations)

type JavaScriptHandler struct{}

func NewJavaScriptHandler() *JavaScriptHandler {
	return &JavaScriptHandler{}
}

func (jh *JavaScriptHandler) DiffAST(ctx context.Context, oldAST, newAST interface{}) ([]ASTChange, error) {
	// Placeholder for JavaScript-specific AST diffing
	return []ASTChange{}, nil
}

func (jh *JavaScriptHandler) GetNodeType(node interface{}) string {
	// Placeholder
	return "unknown"
}

func (jh *JavaScriptHandler) GetNodeChildren(node interface{}) []interface{} {
	// Placeholder
	return []interface{}{}
}

func (jh *JavaScriptHandler) GetNodeText(node interface{}) string {
	// Placeholder
	return ""
}

func (jh *JavaScriptHandler) GetNodePosition(node interface{}) Position {
	// Placeholder
	return Position{}
}

type TypeScriptHandler struct{}

func NewTypeScriptHandler() *TypeScriptHandler {
	return &TypeScriptHandler{}
}

func (th *TypeScriptHandler) DiffAST(ctx context.Context, oldAST, newAST interface{}) ([]ASTChange, error) {
	// Placeholder for TypeScript-specific AST diffing
	return []ASTChange{}, nil
}

func (th *TypeScriptHandler) GetNodeType(node interface{}) string {
	return "unknown"
}

func (th *TypeScriptHandler) GetNodeChildren(node interface{}) []interface{} {
	return []interface{}{}
}

func (th *TypeScriptHandler) GetNodeText(node interface{}) string {
	return ""
}

func (th *TypeScriptHandler) GetNodePosition(node interface{}) Position {
	return Position{}
}

type GoHandler struct{}

func NewGoHandler() *GoHandler {
	return &GoHandler{}
}

func (gh *GoHandler) DiffAST(ctx context.Context, oldAST, newAST interface{}) ([]ASTChange, error) {
	// Placeholder for Go-specific AST diffing
	return []ASTChange{}, nil
}

func (gh *GoHandler) GetNodeType(node interface{}) string {
	return "unknown"
}

func (gh *GoHandler) GetNodeChildren(node interface{}) []interface{} {
	return []interface{}{}
}

func (gh *GoHandler) GetNodeText(node interface{}) string {
	return ""
}

func (gh *GoHandler) GetNodePosition(node interface{}) Position {
	return Position{}
}

type PythonHandler struct{}

func NewPythonHandler() *PythonHandler {
	return &PythonHandler{}
}

func (ph *PythonHandler) DiffAST(ctx context.Context, oldAST, newAST interface{}) ([]ASTChange, error) {
	// Placeholder for Python-specific AST diffing
	return []ASTChange{}, nil
}

func (ph *PythonHandler) GetNodeType(node interface{}) string {
	return "unknown"
}

func (ph *PythonHandler) GetNodeChildren(node interface{}) []interface{} {
	return []interface{}{}
}

func (ph *PythonHandler) GetNodeText(node interface{}) string {
	return ""
}

func (ph *PythonHandler) GetNodePosition(node interface{}) Position {
	return Position{}
}
