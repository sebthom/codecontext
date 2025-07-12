package analyzer

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

// RelationshipAnalyzer analyzes various types of relationships between code elements
type RelationshipAnalyzer struct {
	graph *types.CodeGraph
}

// NewRelationshipAnalyzer creates a new relationship analyzer
func NewRelationshipAnalyzer(graph *types.CodeGraph) *RelationshipAnalyzer {
	return &RelationshipAnalyzer{
		graph: graph,
	}
}

// RelationshipType represents different types of relationships
type RelationshipType string

const (
	RelationshipImport     RelationshipType = "imports"
	RelationshipCalls      RelationshipType = "calls"
	RelationshipExtends    RelationshipType = "extends"
	RelationshipImplements RelationshipType = "implements"
	RelationshipReferences RelationshipType = "references"
	RelationshipContains   RelationshipType = "contains"
	RelationshipUses       RelationshipType = "uses"
	RelationshipDepends    RelationshipType = "depends"
)

// RelationshipMetrics holds metrics about relationships
type RelationshipMetrics struct {
	TotalRelationships int                      `json:"total_relationships"`
	ByType             map[RelationshipType]int `json:"by_type"`
	FileToFile         int                      `json:"file_to_file"`
	SymbolToSymbol     int                      `json:"symbol_to_symbol"`
	CrossFileRefs      int                      `json:"cross_file_refs"`
	CircularDeps       []CircularDependency     `json:"circular_deps"`
	HotspotFiles       []FileHotspot            `json:"hotspot_files"`
	IsolatedFiles      []string                 `json:"isolated_files"`
}

// CircularDependency represents a circular dependency between files
type CircularDependency struct {
	Files []string `json:"files"`
	Path  []string `json:"path"`
	Type  string   `json:"type"`
}

// FileHotspot represents a file with high dependency activity
type FileHotspot struct {
	FilePath       string  `json:"file_path"`
	ImportCount    int     `json:"import_count"`
	ReferenceCount int     `json:"reference_count"`
	Score          float64 `json:"score"`
}

// AnalyzeAllRelationships performs comprehensive relationship analysis
func (ra *RelationshipAnalyzer) AnalyzeAllRelationships() (*RelationshipMetrics, error) {
	metrics := &RelationshipMetrics{
		ByType:        make(map[RelationshipType]int),
		CircularDeps:  make([]CircularDependency, 0),
		HotspotFiles:  make([]FileHotspot, 0),
		IsolatedFiles: make([]string, 0),
	}

	// Analyze import relationships
	ra.analyzeImportRelationships(metrics)

	// Analyze symbol usage relationships
	ra.analyzeSymbolUsageRelationships(metrics)

	// Analyze call relationships
	ra.analyzeCallRelationships(metrics)

	// Detect circular dependencies
	ra.detectCircularDependencies(metrics)

	// Identify hotspot files
	ra.identifyHotspotFiles(metrics)

	// Find isolated files
	ra.findIsolatedFiles(metrics)

	// Calculate totals
	for _, count := range metrics.ByType {
		metrics.TotalRelationships += count
	}

	return metrics, nil
}

// analyzeImportRelationships analyzes import-based relationships
func (ra *RelationshipAnalyzer) analyzeImportRelationships(metrics *RelationshipMetrics) {
	importCount := 0

	for filePath, fileNode := range ra.graph.Files {
		for _, imp := range fileNode.Imports {
			targetFile := ra.resolveImportPath(imp.Path, filePath)

			if targetFile != "" {
				// Create or update import relationship
				edgeId := types.EdgeId(fmt.Sprintf("import-%s-%s", filePath, targetFile))

				if _, exists := ra.graph.Edges[edgeId]; !exists {
					edge := &types.GraphEdge{
						Id:     edgeId,
						From:   types.NodeId(fmt.Sprintf("file-%s", filePath)),
						To:     types.NodeId(fmt.Sprintf("file-%s", targetFile)),
						Type:   string(RelationshipImport),
						Weight: 1.0,
						Metadata: map[string]interface{}{
							"import_path":   imp.Path,
							"specifiers":    imp.Specifiers,
							"is_default":    imp.IsDefault,
							"resolved_path": targetFile,
						},
					}
					ra.graph.Edges[edgeId] = edge
				}

				importCount++
			} else {
				// External import
				edgeId := types.EdgeId(fmt.Sprintf("external-import-%s-%s", filePath, imp.Path))
				edge := &types.GraphEdge{
					Id:     edgeId,
					From:   types.NodeId(fmt.Sprintf("file-%s", filePath)),
					To:     types.NodeId(fmt.Sprintf("external-%s", imp.Path)),
					Type:   string(RelationshipImport),
					Weight: 0.5, // Lower weight for external imports
					Metadata: map[string]interface{}{
						"import_path": imp.Path,
						"specifiers":  imp.Specifiers,
						"is_default":  imp.IsDefault,
						"is_external": true,
					},
				}
				ra.graph.Edges[edgeId] = edge
				importCount++
			}
		}
	}

	metrics.ByType[RelationshipImport] = importCount
	metrics.FileToFile += importCount
}

// analyzeSymbolUsageRelationships analyzes symbol-to-symbol relationships
func (ra *RelationshipAnalyzer) analyzeSymbolUsageRelationships(metrics *RelationshipMetrics) {
	usageCount := 0
	referenceCount := 0

	// Analyze symbol usage within files
	for filePath, fileNode := range ra.graph.Files {
		for _, symbolId := range fileNode.Symbols {
			symbol := ra.graph.Symbols[symbolId]
			if symbol == nil {
				continue
			}

			// Analyze references in symbol signatures and documentation
			references := ra.extractSymbolReferences(symbol)

			for _, ref := range references {
				targetSymbol := ra.findSymbolByName(ref.Name, ref.Context)
				if targetSymbol != nil {
					// Create reference relationship
					edgeId := types.EdgeId(fmt.Sprintf("ref-%s-%s", symbol.Id, targetSymbol.Id))
					edge := &types.GraphEdge{
						Id:     edgeId,
						From:   types.NodeId(fmt.Sprintf("symbol-%s", symbol.Id)),
						To:     types.NodeId(fmt.Sprintf("symbol-%s", targetSymbol.Id)),
						Type:   string(RelationshipReferences),
						Weight: ref.Weight,
						Metadata: map[string]interface{}{
							"reference_type": ref.Type,
							"context":        ref.Context,
							"source_file":    filePath,
							"target_file":    targetSymbol.FullyQualifiedName,
						},
					}
					ra.graph.Edges[edgeId] = edge

					if symbol.FullyQualifiedName != targetSymbol.FullyQualifiedName {
						referenceCount++
					}
					usageCount++
				}
			}
		}
	}

	metrics.ByType[RelationshipReferences] = usageCount
	metrics.ByType[RelationshipUses] = usageCount
	metrics.SymbolToSymbol += usageCount
	metrics.CrossFileRefs += referenceCount
}

// analyzeCallRelationships analyzes function/method call relationships
func (ra *RelationshipAnalyzer) analyzeCallRelationships(metrics *RelationshipMetrics) {
	callCount := 0

	// For now, this is a basic implementation
	// In a full implementation, we would parse function bodies to extract calls
	// This requires more sophisticated AST analysis

	for _, symbol := range ra.graph.Symbols {
		if symbol.Type == types.SymbolTypeFunction || symbol.Type == types.SymbolTypeMethod {
			// Analyze function signature for parameter types (simplified)
			if strings.Contains(symbol.Signature, "=>") || strings.Contains(symbol.Signature, "return") {
				// This is a placeholder for actual call analysis
				callCount++
			}
		}
	}

	metrics.ByType[RelationshipCalls] = callCount
}

// detectCircularDependencies detects circular import dependencies
func (ra *RelationshipAnalyzer) detectCircularDependencies(metrics *RelationshipMetrics) {
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	for filePath := range ra.graph.Files {
		if !visited[filePath] {
			if cycle := ra.detectCycleDFS(filePath, visited, recursionStack, make([]string, 0)); cycle != nil {
				metrics.CircularDeps = append(metrics.CircularDeps, CircularDependency{
					Files: cycle,
					Path:  cycle,
					Type:  "import",
				})
			}
		}
	}
}

// detectCycleDFS performs DFS to detect cycles in the dependency graph
func (ra *RelationshipAnalyzer) detectCycleDFS(filePath string, visited, recursionStack map[string]bool, path []string) []string {
	visited[filePath] = true
	recursionStack[filePath] = true
	path = append(path, filePath)

	fileNode := ra.graph.Files[filePath]
	if fileNode == nil {
		return nil
	}

	for _, imp := range fileNode.Imports {
		targetFile := ra.resolveImportPath(imp.Path, filePath)
		if targetFile == "" {
			continue
		}

		if !visited[targetFile] {
			if cycle := ra.detectCycleDFS(targetFile, visited, recursionStack, path); cycle != nil {
				return cycle
			}
		} else if recursionStack[targetFile] {
			// Found a cycle
			cycleStart := -1
			for i, p := range path {
				if p == targetFile {
					cycleStart = i
					break
				}
			}
			if cycleStart != -1 {
				return append(path[cycleStart:], targetFile)
			}
		}
	}

	recursionStack[filePath] = false
	return nil
}

// identifyHotspotFiles identifies files with high dependency activity
func (ra *RelationshipAnalyzer) identifyHotspotFiles(metrics *RelationshipMetrics) {
	fileScores := make(map[string]*FileHotspot)

	// Initialize hotspot data for each file
	for filePath := range ra.graph.Files {
		fileScores[filePath] = &FileHotspot{
			FilePath:       filePath,
			ImportCount:    0,
			ReferenceCount: 0,
			Score:          0.0,
		}
	}

	// Count incoming and outgoing dependencies
	for _, edge := range ra.graph.Edges {
		if edge.Type == string(RelationshipImport) {
			fromFile := ra.extractFileFromNodeId(edge.From)
			toFile := ra.extractFileFromNodeId(edge.To)

			if fromFile != "" && fileScores[fromFile] != nil {
				fileScores[fromFile].ImportCount++
			}
			if toFile != "" && fileScores[toFile] != nil {
				fileScores[toFile].ReferenceCount++
			}
		}
	}

	// Calculate scores and identify top hotspots
	for _, hotspot := range fileScores {
		// Simple scoring: imports + references * 2 (being referenced is more important)
		hotspot.Score = float64(hotspot.ImportCount) + float64(hotspot.ReferenceCount)*2.0

		// Only include files with significant activity
		if hotspot.Score >= 2.0 {
			metrics.HotspotFiles = append(metrics.HotspotFiles, *hotspot)
		}
	}
}

// findIsolatedFiles finds files with no dependencies
func (ra *RelationshipAnalyzer) findIsolatedFiles(metrics *RelationshipMetrics) {
	connectedFiles := make(map[string]bool)

	// Mark files that have any edges
	for _, edge := range ra.graph.Edges {
		if edge.Type == string(RelationshipImport) {
			fromFile := ra.extractFileFromNodeId(edge.From)
			toFile := ra.extractFileFromNodeId(edge.To)

			if fromFile != "" {
				connectedFiles[fromFile] = true
			}
			if toFile != "" {
				connectedFiles[toFile] = true
			}
		}
	}

	// Find files with no connections
	for filePath := range ra.graph.Files {
		if !connectedFiles[filePath] {
			metrics.IsolatedFiles = append(metrics.IsolatedFiles, filePath)
		}
	}
}

// Helper types and functions

// SymbolReference represents a reference to a symbol
type SymbolReference struct {
	Name    string
	Type    string
	Context string
	Weight  float64
}

// extractSymbolReferences extracts symbol references from signatures and documentation
func (ra *RelationshipAnalyzer) extractSymbolReferences(symbol *types.Symbol) []SymbolReference {
	references := make([]SymbolReference, 0)

	// Simple pattern matching for TypeScript/JavaScript types
	if symbol.Language == "typescript" || symbol.Language == "javascript" {
		// Extract type references from signature
		if symbol.Signature != "" {
			typeRefs := ra.extractTypeReferences(symbol.Signature)
			for _, typeRef := range typeRefs {
				references = append(references, SymbolReference{
					Name:    typeRef,
					Type:    "type_reference",
					Context: "signature",
					Weight:  1.0,
				})
			}
		}
	}

	return references
}

// extractTypeReferences extracts type references from a signature
func (ra *RelationshipAnalyzer) extractTypeReferences(signature string) []string {
	// Simple pattern matching - could be enhanced with proper parsing
	types := make([]string, 0)

	// Look for TypeScript type annotations
	if strings.Contains(signature, ":") {
		parts := strings.Split(signature, ":")
		for _, part := range parts[1:] {
			// Extract type name (simplified)
			typeName := strings.TrimSpace(part)

			// Remove common separators and tokens
			if idx := strings.Index(typeName, " "); idx != -1 {
				typeName = typeName[:idx]
			}
			if idx := strings.Index(typeName, ")"); idx != -1 {
				typeName = typeName[:idx]
			}
			if idx := strings.Index(typeName, ","); idx != -1 {
				typeName = typeName[:idx]
			}
			if idx := strings.Index(typeName, ";"); idx != -1 {
				typeName = typeName[:idx]
			}

			// Clean up the type name
			typeName = strings.TrimSpace(typeName)
			if typeName != "" && !ra.isBuiltinType(typeName) {
				types = append(types, typeName)
			}
		}
	}

	return types
}

// isBuiltinType checks if a type is a built-in type
func (ra *RelationshipAnalyzer) isBuiltinType(typeName string) bool {
	builtins := []string{
		"string", "number", "boolean", "object", "undefined", "null",
		"void", "any", "unknown", "never", "bigint", "symbol",
		"Array", "Promise", "Date", "RegExp", "Error",
	}

	for _, builtin := range builtins {
		if typeName == builtin {
			return true
		}
	}

	return false
}

// findSymbolByName finds a symbol by name within a context
func (ra *RelationshipAnalyzer) findSymbolByName(name, context string) *types.Symbol {
	for _, symbol := range ra.graph.Symbols {
		if symbol.Name == name {
			return symbol
		}
	}
	return nil
}

// extractFileFromNodeId extracts the file path from a node ID
func (ra *RelationshipAnalyzer) extractFileFromNodeId(nodeId types.NodeId) string {
	nodeIdStr := string(nodeId)
	if strings.HasPrefix(nodeIdStr, "file-") {
		return nodeIdStr[5:] // Remove "file-" prefix
	}
	return ""
}

// resolveImportPath resolves an import path to an actual file path
func (ra *RelationshipAnalyzer) resolveImportPath(importPath, fromFile string) string {
	// Handle relative imports
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		dir := filepath.Dir(fromFile)
		resolved := filepath.Join(dir, importPath)

		// Try common extensions
		extensions := []string{".ts", ".tsx", ".js", ".jsx"}
		for _, ext := range extensions {
			candidate := resolved + ext
			if _, exists := ra.graph.Files[candidate]; exists {
				return candidate
			}
		}

		// Try with index files
		for _, ext := range extensions {
			candidate := filepath.Join(resolved, "index"+ext)
			if _, exists := ra.graph.Files[candidate]; exists {
				return candidate
			}
		}
	}

	return ""
}
