package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nuthan-ms/codecontext/internal/parser"
	"github.com/nuthan-ms/codecontext/pkg/types"
)

// GraphBuilder builds code graphs from parsed files
type GraphBuilder struct {
	parser *parser.Manager
	graph  *types.CodeGraph
}

// NewGraphBuilder creates a new graph builder
func NewGraphBuilder() *GraphBuilder {
	return &GraphBuilder{
		parser: parser.NewManager(),
		graph: &types.CodeGraph{
			Nodes:       make(map[types.NodeId]*types.GraphNode),
			Edges:       make(map[types.EdgeId]*types.GraphEdge),
			Files:       make(map[string]*types.FileNode),
			Symbols:     make(map[types.SymbolId]*types.Symbol),
			Metadata:    &types.GraphMetadata{},
		},
	}
}

// AnalyzeDirectory analyzes a directory and builds a complete code graph
func (gb *GraphBuilder) AnalyzeDirectory(targetDir string) (*types.CodeGraph, error) {
	start := time.Now()
	
	// Initialize graph metadata
	gb.graph.Metadata = &types.GraphMetadata{
		Generated:   time.Now(),
		Version:     "2.0.0",
		TotalFiles:  0,
		TotalSymbols: 0,
		Languages:   make(map[string]int),
	}
	
	// Walk directory and process files
	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories and unsupported files
		if info.IsDir() || !gb.isSupportedFile(path) {
			return nil
		}
		
		// Skip certain directories
		if gb.shouldSkipPath(path) {
			return nil
		}
		
		return gb.processFile(path)
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to analyze directory: %w", err)
	}
	
	// Build relationships between files
	gb.buildFileRelationships()
	
	// Update metadata
	gb.graph.Metadata.TotalFiles = len(gb.graph.Files)
	gb.graph.Metadata.TotalSymbols = len(gb.graph.Symbols)
	gb.graph.Metadata.AnalysisTime = time.Since(start)
	
	return gb.graph, nil
}

// processFile processes a single file and extracts symbols
func (gb *GraphBuilder) processFile(filePath string) error {
	// Detect language
	classification, err := gb.parser.ClassifyFile(filePath)
	if err != nil {
		// Skip files we can't classify
		return nil
	}
	
	// Parse the file
	ast, err := gb.parser.ParseFile(filePath, classification.Language)
	if err != nil {
		return fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}
	
	// Extract symbols
	symbols, err := gb.parser.ExtractSymbols(ast)
	if err != nil {
		return fmt.Errorf("failed to extract symbols from %s: %w", filePath, err)
	}
	
	// Extract imports
	imports, err := gb.parser.ExtractImports(ast)
	if err != nil {
		return fmt.Errorf("failed to extract imports from %s: %w", filePath, err)
	}
	
	// Create file node
	fileNode := &types.FileNode{
		Path:         filePath,
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
	
	// Add symbols to graph and file
	for _, symbol := range symbols {
		gb.graph.Symbols[symbol.Id] = symbol
		fileNode.Symbols = append(fileNode.Symbols, symbol.Id)
		
		// Create symbol node
		symbolNode := &types.GraphNode{
			Id:       types.NodeId(fmt.Sprintf("symbol-%s", symbol.Id)),
			Type:     "symbol",
			Label:    symbol.Name,
			FilePath: filePath,
			Metadata: map[string]interface{}{
				"symbolType": symbol.Type,
				"language":   symbol.Language,
				"signature":  symbol.Signature,
				"line":       symbol.Location.Line,
			},
		}
		gb.graph.Nodes[symbolNode.Id] = symbolNode
	}
	
	// Add file to graph
	gb.graph.Files[filePath] = fileNode
	
	// Update language statistics
	if gb.graph.Metadata.Languages == nil {
		gb.graph.Metadata.Languages = make(map[string]int)
	}
	gb.graph.Metadata.Languages[classification.Language.Name]++
	
	return nil
}

// buildFileRelationships analyzes imports to build file-to-file relationships
func (gb *GraphBuilder) buildFileRelationships() {
	for filePath, fileNode := range gb.graph.Files {
		for _, imp := range fileNode.Imports {
			targetFile := gb.resolveImportPath(imp.Path, filePath)
			if targetFile != "" && gb.graph.Files[targetFile] != nil {
				// Create edge for file dependency
				edgeId := types.EdgeId(fmt.Sprintf("import-%s-%s", filePath, targetFile))
				edge := &types.GraphEdge{
					Id:     edgeId,
					From:   types.NodeId(fmt.Sprintf("file-%s", filePath)),
					To:     types.NodeId(fmt.Sprintf("file-%s", targetFile)),
					Type:   "imports",
					Weight: 1.0,
					Metadata: map[string]interface{}{
						"importPath": imp.Path,
						"specifiers": imp.Specifiers,
						"isDefault":  imp.IsDefault,
					},
				}
				gb.graph.Edges[edgeId] = edge
			}
		}
	}
}

// resolveImportPath attempts to resolve an import path to an actual file
func (gb *GraphBuilder) resolveImportPath(importPath, fromFile string) string {
	// Handle relative imports
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		dir := filepath.Dir(fromFile)
		resolved := filepath.Join(dir, importPath)
		
		// Try common extensions
		extensions := []string{".ts", ".tsx", ".js", ".jsx"}
		for _, ext := range extensions {
			candidate := resolved + ext
			if _, exists := gb.graph.Files[candidate]; exists {
				return candidate
			}
		}
		
		// Try with index files
		for _, ext := range extensions {
			candidate := filepath.Join(resolved, "index"+ext)
			if _, exists := gb.graph.Files[candidate]; exists {
				return candidate
			}
		}
	}
	
	// For now, we don't resolve node_modules or absolute imports
	// This could be enhanced later
	return ""
}

// isSupportedFile checks if a file is supported for parsing
func (gb *GraphBuilder) isSupportedFile(path string) bool {
	ext := filepath.Ext(path)
	supportedExtensions := []string{".ts", ".tsx", ".js", ".jsx", ".json", ".yaml", ".yml"}
	
	for _, supported := range supportedExtensions {
		if ext == supported {
			return true
		}
	}
	return false
}

// shouldSkipPath checks if a path should be skipped during analysis
func (gb *GraphBuilder) shouldSkipPath(path string) bool {
	skipDirs := []string{
		"node_modules", ".git", ".codecontext", "dist", "build", 
		"coverage", ".nyc_output", "tmp", "temp",
	}
	
	for _, skipDir := range skipDirs {
		if strings.Contains(path, skipDir) {
			return true
		}
	}
	
	return false
}

// GetSupportedLanguages returns the list of supported languages
func (gb *GraphBuilder) GetSupportedLanguages() []types.Language {
	return gb.parser.GetSupportedLanguages()
}

// GetFileStats returns statistics about the analyzed files
func (gb *GraphBuilder) GetFileStats() map[string]interface{} {
	if gb.graph.Metadata == nil {
		return map[string]interface{}{}
	}
	
	return map[string]interface{}{
		"totalFiles":   gb.graph.Metadata.TotalFiles,
		"totalSymbols": gb.graph.Metadata.TotalSymbols,
		"languages":    gb.graph.Metadata.Languages,
		"analysisTime": gb.graph.Metadata.AnalysisTime,
	}
}