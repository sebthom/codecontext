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
			if symbols[i].Location.FilePath != symbols[j].Location.FilePath {
				return symbols[i].Location.FilePath < symbols[j].Location.FilePath
			}
			return symbols[i].Location.Line < symbols[j].Location.Line
		})
		
		for _, symbol := range symbols {
			signature := symbol.Signature
			if len(signature) > 50 {
				signature = signature[:47] + "..."
			}
			
			sb.WriteString(fmt.Sprintf("| `%s` | %s | `%s` | %d | `%s` |\n",
				symbol.Name,
				symbol.Type,
				filepath.Base(symbol.Location.FilePath),
				symbol.Location.Line,
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