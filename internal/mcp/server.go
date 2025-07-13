package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
	"github.com/nuthan-ms/codecontext/internal/analyzer"
	"github.com/nuthan-ms/codecontext/internal/watcher"
	"github.com/nuthan-ms/codecontext/pkg/types"
)

// MCPConfig holds configuration for the MCP server
type MCPConfig struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	TargetDir   string `json:"target_dir"`
	EnableWatch bool   `json:"enable_watch"`
	DebounceMs  int    `json:"debounce_ms"`
}

// CodeContextMCPServer provides codecontext functionality via MCP
type CodeContextMCPServer struct {
	server   *mcp.Server
	config   *MCPConfig
	watcher  *watcher.FileWatcher
	graph    *types.CodeGraph
	analyzer *analyzer.GraphBuilder
}

// Tool argument structs with jsonschema tags

type GetCodebaseOverviewArgs struct {
	IncludeStats bool `json:"include_stats" jsonschema:"description=Include detailed statistics"`
}

type GetFileAnalysisArgs struct {
	FilePath string `json:"file_path" jsonschema:"description=Path to the file to analyze,required"`
}

type GetSymbolInfoArgs struct {
	SymbolName string `json:"symbol_name" jsonschema:"description=Name of the symbol to lookup,required"`
	FilePath   string `json:"file_path,omitempty" jsonschema:"description=Optional file path to scope the search"`
}

type SearchSymbolsArgs struct {
	Query    string `json:"query" jsonschema:"description=Search query for symbols,required"`
	FileType string `json:"file_type,omitempty" jsonschema:"description=Optional file type filter (e.g. 'typescript', 'javascript')"`
	Limit    int    `json:"limit,omitempty" jsonschema:"description=Maximum number of results to return (default: 20)"`
}

type GetDependenciesArgs struct {
	FilePath  string `json:"file_path,omitempty" jsonschema:"description=Optional file path to get dependencies for specific file"`
	Direction string `json:"direction,omitempty" jsonschema:"description=Direction: 'imports' (what file imports) or 'dependents' (what imports the file)"`
}

type WatchChangesArgs struct {
	Enable bool `json:"enable" jsonschema:"description=Enable or disable real-time change watching"`
}

// NewCodeContextMCPServer creates a new MCP server instance
func NewCodeContextMCPServer(config *MCPConfig) (*CodeContextMCPServer, error) {
	transport := stdio.NewStdioServerTransport()
	server := mcp.NewServer(transport, 
		mcp.WithName(config.Name),
		mcp.WithVersion(config.Version),
	)
	
	s := &CodeContextMCPServer{
		server:   server,
		config:   config,
		analyzer: analyzer.NewGraphBuilder(),
	}

	// Register tools
	s.registerTools()
	
	return s, nil
}

// registerTools registers all MCP tools
func (s *CodeContextMCPServer) registerTools() {
	// Tool 1: Get codebase overview
	s.server.RegisterTool("get_codebase_overview", "Get comprehensive overview of the entire codebase", 
		s.getCodebaseOverview)

	// Tool 2: Get file analysis
	s.server.RegisterTool("get_file_analysis", "Get detailed analysis of a specific file", 
		s.getFileAnalysis)

	// Tool 3: Get symbol information
	s.server.RegisterTool("get_symbol_info", "Get detailed information about a specific symbol", 
		s.getSymbolInfo)

	// Tool 4: Search symbols
	s.server.RegisterTool("search_symbols", "Search for symbols across the codebase", 
		s.searchSymbols)

	// Tool 5: Get dependencies
	s.server.RegisterTool("get_dependencies", "Analyze import dependencies and relationships", 
		s.getDependencies)

	// Tool 6: Watch changes (real-time)
	s.server.RegisterTool("watch_changes", "Enable/disable real-time change notifications", 
		s.watchChanges)
}

// Tool implementations

func (s *CodeContextMCPServer) getCodebaseOverview(args GetCodebaseOverviewArgs) (*mcp.ToolResponse, error) {
	// Ensure we have fresh analysis
	if err := s.refreshAnalysis(); err != nil {
		return nil, fmt.Errorf("failed to refresh analysis: %w", err)
	}

	generator := analyzer.NewMarkdownGenerator(s.graph)
	content := generator.GenerateContextMap()

	if args.IncludeStats {
		stats := s.analyzer.GetFileStats()
		statsJson, _ := json.MarshalIndent(stats, "", "  ")
		content += "\n\n## Detailed Statistics\n```json\n" + string(statsJson) + "\n```"
	}

	return mcp.NewToolResponse(mcp.NewTextContent(content)), nil
}

func (s *CodeContextMCPServer) getFileAnalysis(args GetFileAnalysisArgs) (*mcp.ToolResponse, error) {
	if args.FilePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}

	// Ensure we have fresh analysis
	if err := s.refreshAnalysis(); err != nil {
		return nil, fmt.Errorf("failed to refresh analysis: %w", err)
	}

	// Find the file in our graph
	fileNode, exists := s.graph.Files[args.FilePath]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", args.FilePath)
	}

	// Build detailed file analysis
	analysis := fmt.Sprintf("# File Analysis: %s\n\n", args.FilePath)
	analysis += fmt.Sprintf("**Language:** %s\n", fileNode.Language)
	analysis += fmt.Sprintf("**Lines:** %d\n", fileNode.Lines)
	analysis += fmt.Sprintf("**Symbols:** %d\n\n", len(fileNode.Symbols))

	// List symbols in this file
	if len(fileNode.Symbols) > 0 {
		analysis += "## Symbols\n\n"
		for _, symbolId := range fileNode.Symbols {
			if symbol, exists := s.graph.Symbols[symbolId]; exists {
				analysis += fmt.Sprintf("- **%s** (%s) - Line %d\n", 
					symbol.Name, symbol.Kind, symbol.Location.StartLine)
			}
		}
	}

	// List imports for this file
	analysis += "\n## Dependencies\n\n"
	importCount := 0
	for _, edge := range s.graph.Edges {
		if edge.Type == "imports" && edge.From == types.NodeId(args.FilePath) {
			if importCount == 0 {
				analysis += "### Imports:\n"
			}
			analysis += fmt.Sprintf("- %s\n", edge.To)
			importCount++
		}
	}
	if importCount == 0 {
		analysis += "No imports found.\n"
	}

	return mcp.NewToolResponse(mcp.NewTextContent(analysis)), nil
}

func (s *CodeContextMCPServer) getSymbolInfo(args GetSymbolInfoArgs) (*mcp.ToolResponse, error) {
	if args.SymbolName == "" {
		return nil, fmt.Errorf("symbol_name is required")
	}

	// Ensure we have fresh analysis
	if err := s.refreshAnalysis(); err != nil {
		return nil, fmt.Errorf("failed to refresh analysis: %w", err)
	}

	var foundSymbols []*types.Symbol
	for _, symbol := range s.graph.Symbols {
		if symbol.Name == args.SymbolName {
			// If file_path is specified, filter by it (Note: Location doesn't have a File field directly)
			foundSymbols = append(foundSymbols, symbol)
		}
	}

	if len(foundSymbols) == 0 {
		return nil, fmt.Errorf("symbol '%s' not found", args.SymbolName)
	}

	result := fmt.Sprintf("# Symbol Information: %s\n\n", args.SymbolName)
	
	for i, symbol := range foundSymbols {
		if i > 0 {
			result += "\n---\n\n"
		}
		result += fmt.Sprintf("**Line:** %d\n", symbol.Location.StartLine)
		result += fmt.Sprintf("**Type:** %s\n", symbol.Kind)
		if symbol.Signature != "" {
			result += fmt.Sprintf("**Signature:** `%s`\n", symbol.Signature)
		}
		if symbol.Documentation != "" {
			result += fmt.Sprintf("**Documentation:** %s\n", symbol.Documentation)
		}
	}

	return mcp.NewToolResponse(mcp.NewTextContent(result)), nil
}

func (s *CodeContextMCPServer) searchSymbols(args SearchSymbolsArgs) (*mcp.ToolResponse, error) {
	if args.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	// Set default limit
	if args.Limit <= 0 {
		args.Limit = 20
	}

	// Ensure we have fresh analysis
	if err := s.refreshAnalysis(); err != nil {
		return nil, fmt.Errorf("failed to refresh analysis: %w", err)
	}

	var matches []*types.Symbol
	query := strings.ToLower(args.Query)

	for _, symbol := range s.graph.Symbols {
		// Check if symbol name contains query
		if strings.Contains(strings.ToLower(symbol.Name), query) {
			matches = append(matches, symbol)
			if len(matches) >= args.Limit {
				break
			}
		}
	}

	if len(matches) == 0 {
		result := fmt.Sprintf("No symbols found matching '%s'", args.Query)
		return mcp.NewToolResponse(mcp.NewTextContent(result)), nil
	}

	result := fmt.Sprintf("# Symbol Search Results: '%s'\n\n", args.Query)
	result += fmt.Sprintf("Found %d matches:\n\n", len(matches))

	for _, symbol := range matches {
		result += fmt.Sprintf("- **%s** (%s) - Line %d\n", 
			symbol.Name, symbol.Kind, symbol.Location.StartLine)
	}

	return mcp.NewToolResponse(mcp.NewTextContent(result)), nil
}

func (s *CodeContextMCPServer) getDependencies(args GetDependenciesArgs) (*mcp.ToolResponse, error) {
	// Ensure we have fresh analysis
	if err := s.refreshAnalysis(); err != nil {
		return nil, fmt.Errorf("failed to refresh analysis: %w", err)
	}

	result := "# Dependency Analysis\n\n"

	if args.FilePath != "" {
		// File-specific dependencies
		result += fmt.Sprintf("## Dependencies for: %s\n\n", args.FilePath)
		
		if args.Direction == "" || args.Direction == "imports" {
			result += "### Imports:\n"
			found := false
			for _, edge := range s.graph.Edges {
				if edge.Type == "imports" && edge.From == types.NodeId(args.FilePath) {
					result += fmt.Sprintf("- %s\n", edge.To)
					found = true
				}
			}
			if !found {
				result += "No imports found.\n"
			}
		}

		if args.Direction == "" || args.Direction == "dependents" {
			result += "\n### Dependents (files that import this):\n"
			found := false
			for _, edge := range s.graph.Edges {
				if edge.Type == "imports" && edge.To == types.NodeId(args.FilePath) {
					result += fmt.Sprintf("- %s\n", edge.From)
					found = true
				}
			}
			if !found {
				result += "No dependents found.\n"
			}
		}
	} else {
		// Global dependency overview
		result += "## Global Dependency Overview\n\n"
		
		fileCount := len(s.graph.Files)
		importCount := 0
		for _, edge := range s.graph.Edges {
			if edge.Type == "imports" {
				importCount++
			}
		}
		
		result += fmt.Sprintf("- **Total Files:** %d\n", fileCount)
		result += fmt.Sprintf("- **Total Import Relationships:** %d\n", importCount)
		
		// Most imported files
		dependentCounts := make(map[string]int)
		for _, edge := range s.graph.Edges {
			if edge.Type == "imports" {
				dependentCounts[string(edge.To)]++
			}
		}
		
		if len(dependentCounts) > 0 {
			result += "\n### Most Imported Files:\n"
			// Simple top 5 most imported
			count := 0
			for file, deps := range dependentCounts {
				if count >= 5 {
					break
				}
				result += fmt.Sprintf("- %s (%d imports)\n", file, deps)
				count++
			}
		}
	}

	return mcp.NewToolResponse(mcp.NewTextContent(result)), nil
}

func (s *CodeContextMCPServer) watchChanges(args WatchChangesArgs) (*mcp.ToolResponse, error) {
	if args.Enable {
		if s.watcher != nil {
			return mcp.NewToolResponse(mcp.NewTextContent("File watching is already enabled")), nil
		}
		
		// Create watcher config
		config := watcher.Config{
			TargetDir:    s.config.TargetDir,
			OutputFile:   "CLAUDE.md", // Not used in MCP mode
			DebounceTime: time.Duration(s.config.DebounceMs) * time.Millisecond,
			IncludeExts:  []string{".ts", ".tsx", ".js", ".jsx", ".go", ".py", ".java", ".cpp", ".c", ".rs"},
		}
		
		// Start file watcher
		fileWatcher, err := watcher.NewFileWatcher(config)
		if err != nil {
			return nil, fmt.Errorf("failed to start file watcher: %w", err)
		}
		
		s.watcher = fileWatcher
		
		// Start watching in a goroutine
		ctx := context.Background()
		go func() {
			if err := fileWatcher.Start(ctx); err != nil {
				log.Printf("File watcher error: %v", err)
			}
		}()
		
		return mcp.NewToolResponse(mcp.NewTextContent("File watching enabled. Real-time change notifications are now active.")), nil
	} else {
		if s.watcher == nil {
			return mcp.NewToolResponse(mcp.NewTextContent("File watching is not currently enabled")), nil
		}
		
		s.watcher.Stop()
		s.watcher = nil
		
		return mcp.NewToolResponse(mcp.NewTextContent("File watching disabled")), nil
	}
}

// Helper methods

func (s *CodeContextMCPServer) refreshAnalysis() error {
	graph, err := s.analyzer.AnalyzeDirectory(s.config.TargetDir)
	if err != nil {
		return err
	}
	s.graph = graph
	return nil
}

// Run starts the MCP server
func (s *CodeContextMCPServer) Run(ctx context.Context) error {
	// Initial analysis
	if err := s.refreshAnalysis(); err != nil {
		return fmt.Errorf("failed to perform initial analysis: %w", err)
	}
	
	log.Printf("CodeContext MCP Server started - analyzing %s", s.config.TargetDir)
	
	// Run the MCP server
	return s.server.Serve()
}

// Stop gracefully stops the MCP server
func (s *CodeContextMCPServer) Stop() {
	if s.watcher != nil {
		s.watcher.Stop()
		s.watcher = nil
	}
}