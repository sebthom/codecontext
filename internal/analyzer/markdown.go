package analyzer

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

// MarkdownGenerator generates rich markdown content from analyzed code graphs
type MarkdownGenerator struct {
	graph *types.CodeGraph
}

// NewMarkdownGenerator creates a new markdown generator
func NewMarkdownGenerator(graph *types.CodeGraph) *MarkdownGenerator {
	return &MarkdownGenerator{graph: graph}
}

// GenerateContextMap generates a comprehensive context map in markdown format
func (mg *MarkdownGenerator) GenerateContextMap() string {
	var sb strings.Builder
	
	// Header
	sb.WriteString(mg.generateHeader())
	sb.WriteString("\n\n")
	
	// Overview
	sb.WriteString(mg.generateOverview())
	sb.WriteString("\n\n")
	
	// File Analysis
	sb.WriteString(mg.generateFileAnalysis())
	sb.WriteString("\n\n")
	
	// Symbol Analysis
	sb.WriteString(mg.generateSymbolAnalysis())
	sb.WriteString("\n\n")
	
	// Language Statistics
	sb.WriteString(mg.generateLanguageStats())
	sb.WriteString("\n\n")
	
	// Import Analysis
	sb.WriteString(mg.generateImportAnalysis())
	sb.WriteString("\n\n")
	
	// Relationship Analysis
	sb.WriteString(mg.generateRelationshipAnalysis())
	sb.WriteString("\n\n")
	
	// Project Structure
	sb.WriteString(mg.generateProjectStructure())
	sb.WriteString("\n\n")
	
	// Footer
	sb.WriteString(mg.generateFooter())
	
	return sb.String()
}

// generateHeader creates the document header
func (mg *MarkdownGenerator) generateHeader() string {
	generated := mg.graph.Metadata.Generated.Format(time.RFC3339)
	analysisTime := mg.graph.Metadata.AnalysisTime.String()
	
	return fmt.Sprintf(`# CodeContext Map

**Generated:** %s  
**Version:** %s  
**Analysis Time:** %s  
**Status:** Real Tree-sitter Analysis`, 
		generated, mg.graph.Metadata.Version, analysisTime)
}

// generateOverview creates the overview section
func (mg *MarkdownGenerator) generateOverview() string {
	return fmt.Sprintf(`## 📊 Overview

This context map was generated using **real Tree-sitter parsing** and provides comprehensive analysis of your codebase:

- **Files Analyzed**: %d files
- **Symbols Extracted**: %d symbols  
- **Languages Detected**: %d languages
- **Import Relationships**: %d file dependencies

### 🎯 Analysis Capabilities
- ✅ **Real AST Parsing** - Tree-sitter JavaScript/TypeScript grammars
- ✅ **Symbol Extraction** - Functions, classes, methods, variables, imports
- ✅ **Dependency Analysis** - File-to-file relationship mapping
- ✅ **Multi-language Support** - TypeScript, JavaScript, JSON, YAML`,
		mg.graph.Metadata.TotalFiles,
		mg.graph.Metadata.TotalSymbols,
		len(mg.graph.Metadata.Languages),
		len(mg.graph.Edges))
}

// generateFileAnalysis creates the file analysis section
func (mg *MarkdownGenerator) generateFileAnalysis() string {
	var sb strings.Builder
	sb.WriteString("## 📁 File Analysis\n\n")
	
	if len(mg.graph.Files) == 0 {
		sb.WriteString("*No files analyzed.*\n")
		return sb.String()
	}
	
	// Sort files by path for consistent output
	files := make([]*types.FileNode, 0, len(mg.graph.Files))
	for _, file := range mg.graph.Files {
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	
	sb.WriteString("| File | Language | Lines | Symbols | Imports | Type |\n")
	sb.WriteString("|------|----------|-------|---------|---------|------|\n")
	
	for _, file := range files {
		fileType := "source"
		if file.IsTest {
			fileType = "test"
		} else if file.IsGenerated {
			fileType = "generated"
		}
		
		sb.WriteString(fmt.Sprintf("| `%s` | %s | %d | %d | %d | %s |\n",
			file.Path,
			file.Language,
			file.Lines,
			file.SymbolCount,
			file.ImportCount,
			fileType))
	}
	
	return sb.String()
}

// generateSymbolAnalysis creates the symbol analysis section
func (mg *MarkdownGenerator) generateSymbolAnalysis() string {
	var sb strings.Builder
	sb.WriteString("## 🔍 Symbol Analysis\n\n")
	
	if len(mg.graph.Symbols) == 0 {
		sb.WriteString("*No symbols extracted.*\n")
		return sb.String()
	}
	
	// Count symbols by type
	symbolCounts := make(map[types.SymbolType]int)
	for _, symbol := range mg.graph.Symbols {
		symbolCounts[symbol.Type]++
	}
	
	// Display symbol counts
	sb.WriteString("### Symbol Types\n\n")
	for symbolType, count := range symbolCounts {
		icon := mg.getSymbolIcon(symbolType)
		sb.WriteString(fmt.Sprintf("- %s **%s**: %d\n", icon, symbolType, count))
	}
	
	// Show detailed symbol list for smaller projects
	if len(mg.graph.Symbols) <= 50 {
		sb.WriteString("\n### Symbol Details\n\n")
		sb.WriteString("| Symbol | Type | File | Line | Signature |\n")
		sb.WriteString("|--------|------|------|------|----------|\n")
		
		// Sort symbols by file and line
		symbols := make([]*types.Symbol, 0, len(mg.graph.Symbols))
		for _, symbol := range mg.graph.Symbols {
			symbols = append(symbols, symbol)
		}
		sort.Slice(symbols, func(i, j int) bool {
			if symbols[i].FullyQualifiedName != symbols[j].FullyQualifiedName {
				return symbols[i].FullyQualifiedName < symbols[j].FullyQualifiedName
			}
			return symbols[i].Location.StartLine < symbols[j].Location.StartLine
		})
		
		for _, symbol := range symbols {
			signature := symbol.Signature
			if len(signature) > 50 {
				signature = signature[:47] + "..."
			}
			
			sb.WriteString(fmt.Sprintf("| `%s` | %s | `%s` | %d | `%s` |\n",
				symbol.Name,
				symbol.Type,
				filepath.Base(symbol.FullyQualifiedName),
				symbol.Location.StartLine,
				signature))
		}
	}
	
	return sb.String()
}

// generateLanguageStats creates the language statistics section
func (mg *MarkdownGenerator) generateLanguageStats() string {
	var sb strings.Builder
	sb.WriteString("## 📈 Language Statistics\n\n")
	
	if len(mg.graph.Metadata.Languages) == 0 {
		sb.WriteString("*No languages detected.*\n")
		return sb.String()
	}
	
	// Sort languages by file count
	type langStat struct {
		name  string
		count int
	}
	
	languages := make([]langStat, 0, len(mg.graph.Metadata.Languages))
	for lang, count := range mg.graph.Metadata.Languages {
		languages = append(languages, langStat{lang, count})
	}
	sort.Slice(languages, func(i, j int) bool {
		return languages[i].count > languages[j].count
	})
	
	sb.WriteString("| Language | Files | Percentage |\n")
	sb.WriteString("|----------|-------|------------|\n")
	
	total := mg.graph.Metadata.TotalFiles
	for _, lang := range languages {
		percentage := float64(lang.count) / float64(total) * 100
		sb.WriteString(fmt.Sprintf("| %s | %d | %.1f%% |\n",
			lang.name, lang.count, percentage))
	}
	
	return sb.String()
}

// generateImportAnalysis creates the import analysis section
func (mg *MarkdownGenerator) generateImportAnalysis() string {
	var sb strings.Builder
	sb.WriteString("## 🔗 Import Analysis\n\n")
	
	// Collect all import paths
	importCounts := make(map[string]int)
	internalImports := 0
	externalImports := 0
	
	for _, file := range mg.graph.Files {
		for _, imp := range file.Imports {
			importCounts[imp.Path]++
			
			if strings.HasPrefix(imp.Path, "./") || strings.HasPrefix(imp.Path, "../") {
				internalImports++
			} else {
				externalImports++
			}
		}
	}
	
	sb.WriteString(fmt.Sprintf("- **Total Import Statements**: %d\n", internalImports+externalImports))
	sb.WriteString(fmt.Sprintf("- **Internal Imports**: %d (relative paths)\n", internalImports))
	sb.WriteString(fmt.Sprintf("- **External Imports**: %d (packages/modules)\n", externalImports))
	sb.WriteString(fmt.Sprintf("- **Unique Modules**: %d\n\n", len(importCounts)))
	
	if len(importCounts) > 0 {
		// Show most imported modules
		type importStat struct {
			path  string
			count int
		}
		
		imports := make([]importStat, 0, len(importCounts))
		for path, count := range importCounts {
			imports = append(imports, importStat{path, count})
		}
		sort.Slice(imports, func(i, j int) bool {
			return imports[i].count > imports[j].count
		})
		
		sb.WriteString("### Most Imported Modules\n\n")
		sb.WriteString("| Module | Import Count |\n")
		sb.WriteString("|--------|-------------|\n")
		
		// Show top 10 or all if fewer
		limit := 10
		if len(imports) < limit {
			limit = len(imports)
		}
		
		for i := 0; i < limit; i++ {
			imp := imports[i]
			sb.WriteString(fmt.Sprintf("| `%s` | %d |\n", imp.path, imp.count))
		}
	}
	
	return sb.String()
}

// generateRelationshipAnalysis creates the relationship analysis section
func (mg *MarkdownGenerator) generateRelationshipAnalysis() string {
	var sb strings.Builder
	sb.WriteString("## 🔗 Relationship Analysis\n\n")
	
	// Check if relationship metrics are available
	if mg.graph.Metadata.Configuration == nil {
		sb.WriteString("*Relationship analysis not available.*\n")
		return sb.String()
	}
	
	metricsInterface, exists := mg.graph.Metadata.Configuration["relationship_metrics"]
	if !exists {
		sb.WriteString("*Relationship metrics not found.*\n")
		return sb.String()
	}
	
	metrics, ok := metricsInterface.(*RelationshipMetrics)
	if !ok {
		sb.WriteString("*Invalid relationship metrics format.*\n")
		return sb.String()
	}
	
	// Summary
	sb.WriteString("### 📊 Relationship Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Relationships**: %d\n", metrics.TotalRelationships))
	sb.WriteString(fmt.Sprintf("- **File-to-File**: %d\n", metrics.FileToFile))
	sb.WriteString(fmt.Sprintf("- **Symbol-to-Symbol**: %d\n", metrics.SymbolToSymbol))
	sb.WriteString(fmt.Sprintf("- **Cross-File References**: %d\n", metrics.CrossFileRefs))
	sb.WriteString("\n")
	
	// Relationships by type
	if len(metrics.ByType) > 0 {
		sb.WriteString("### 🔍 Relationship Types\n\n")
		sb.WriteString("| Type | Count | Description |\n")
		sb.WriteString("|------|-------|-------------|\n")
		
		for relType, count := range metrics.ByType {
			description := mg.getRelationshipDescription(relType)
			sb.WriteString(fmt.Sprintf("| %s | %d | %s |\n", relType, count, description))
		}
		sb.WriteString("\n")
	}
	
	// Circular dependencies
	if len(metrics.CircularDeps) > 0 {
		sb.WriteString("### ⚠️ Circular Dependencies\n\n")
		sb.WriteString(fmt.Sprintf("Found %d circular dependencies:\n\n", len(metrics.CircularDeps)))
		
		for i, dep := range metrics.CircularDeps {
			sb.WriteString(fmt.Sprintf("**Circular Dependency %d** (%s):\n", i+1, dep.Type))
			sb.WriteString("```\n")
			sb.WriteString(strings.Join(dep.Path, " → "))
			sb.WriteString("\n```\n\n")
		}
	} else {
		sb.WriteString("### ✅ No Circular Dependencies\n\n")
		sb.WriteString("No circular dependencies detected in the codebase.\n\n")
	}
	
	// Hotspot files
	if len(metrics.HotspotFiles) > 0 {
		sb.WriteString("### 🔥 Hotspot Files\n\n")
		sb.WriteString("Files with high dependency activity:\n\n")
		sb.WriteString("| File | Imports | References | Score |\n")
		sb.WriteString("|------|---------|------------|-------|\n")
		
		// Sort by score (descending)
		hotspots := make([]FileHotspot, len(metrics.HotspotFiles))
		copy(hotspots, metrics.HotspotFiles)
		sort.Slice(hotspots, func(i, j int) bool {
			return hotspots[i].Score > hotspots[j].Score
		})
		
		for _, hotspot := range hotspots {
			fileName := filepath.Base(hotspot.FilePath)
			sb.WriteString(fmt.Sprintf("| `%s` | %d | %d | %.1f |\n", 
				fileName, hotspot.ImportCount, hotspot.ReferenceCount, hotspot.Score))
		}
		sb.WriteString("\n")
	}
	
	// Isolated files
	if len(metrics.IsolatedFiles) > 0 {
		sb.WriteString("### 🏝️ Isolated Files\n\n")
		sb.WriteString("Files with no import/export relationships:\n\n")
		
		for _, filePath := range metrics.IsolatedFiles {
			fileName := filepath.Base(filePath)
			sb.WriteString(fmt.Sprintf("- `%s`\n", fileName))
		}
		sb.WriteString("\n")
	}
	
	return sb.String()
}

// getRelationshipDescription returns a description for a relationship type
func (mg *MarkdownGenerator) getRelationshipDescription(relType RelationshipType) string {
	switch relType {
	case RelationshipImport:
		return "File imports another file"
	case RelationshipCalls:
		return "Function/method calls another function/method"
	case RelationshipExtends:
		return "Class extends another class"
	case RelationshipImplements:
		return "Class implements an interface"
	case RelationshipReferences:
		return "Symbol references another symbol"
	case RelationshipContains:
		return "File contains symbols"
	case RelationshipUses:
		return "Symbol uses another symbol"
	case RelationshipDepends:
		return "Component depends on another component"
	default:
		return "Unknown relationship type"
	}
}

// generateProjectStructure creates the project structure section
func (mg *MarkdownGenerator) generateProjectStructure() string {
	var sb strings.Builder
	sb.WriteString("## 📁 Project Structure\n\n")
	
	if len(mg.graph.Files) == 0 {
		sb.WriteString("*No files to display.*\n")
		return sb.String()
	}
	
	// Build directory tree
	dirs := make(map[string][]string)
	for filePath := range mg.graph.Files {
		dir := filepath.Dir(filePath)
		if dir == "." {
			dir = ""
		}
		dirs[dir] = append(dirs[dir], filepath.Base(filePath))
	}
	
	// Sort directories
	sortedDirs := make([]string, 0, len(dirs))
	for dir := range dirs {
		sortedDirs = append(sortedDirs, dir)
	}
	sort.Strings(sortedDirs)
	
	sb.WriteString("```\n")
	for _, dir := range sortedDirs {
		files := dirs[dir]
		sort.Strings(files)
		
		if dir == "" {
			// Root files
			for _, file := range files {
				sb.WriteString(fmt.Sprintf("%s\n", file))
			}
		} else {
			sb.WriteString(fmt.Sprintf("%s/\n", dir))
			for _, file := range files {
				sb.WriteString(fmt.Sprintf("├── %s\n", file))
			}
		}
	}
	sb.WriteString("```\n")
	
	return sb.String()
}

// generateFooter creates the document footer
func (mg *MarkdownGenerator) generateFooter() string {
	return fmt.Sprintf(`---

*Generated by CodeContext v%s with real Tree-sitter parsing*  
*Analysis completed in %v*`,
		mg.graph.Metadata.Version,
		mg.graph.Metadata.AnalysisTime)
}

// getSymbolIcon returns an appropriate icon for a symbol type
func (mg *MarkdownGenerator) getSymbolIcon(symbolType types.SymbolType) string {
	switch symbolType {
	case types.SymbolTypeFunction:
		return "🔧"
	case types.SymbolTypeClass:
		return "🏗️"
	case types.SymbolTypeInterface:
		return "📋"
	case types.SymbolTypeMethod:
		return "⚙️"
	case types.SymbolTypeVariable:
		return "📦"
	case types.SymbolTypeImport:
		return "📥"
	case types.SymbolTypeNamespace:
		return "📁"
	case types.SymbolTypeType:
		return "🏷️"
	default:
		return "🔹"
	}
}