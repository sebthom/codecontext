package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

// Tool argument structs
type GetCodebaseOverviewArgs struct {
	IncludeStats bool `json:"include_stats"`
}

type GetFileAnalysisArgs struct {
	FilePath string `json:"file_path"`
}

type GetSymbolInfoArgs struct {
	SymbolName string `json:"symbol_name"`
	FilePath   string `json:"file_path,omitempty"`
}

type SearchSymbolsArgs struct {
	Query    string `json:"query"`
	FileType string `json:"file_type,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

type GetDependenciesArgs struct {
	FilePath  string `json:"file_path,omitempty"`
	Direction string `json:"direction,omitempty"`
}

type WatchChangesArgs struct {
	Enable bool `json:"enable"`
}

// NewCodeContextMCPServer creates a new MCP server instance
func NewCodeContextMCPServer(config *MCPConfig) (*CodeContextMCPServer, error) {
	// Redirect all logging to stderr for MCP compatibility
	log.SetOutput(os.Stderr)
	log.Printf("[MCP] Creating new CodeContext MCP server with config: %+v", config)
	
	// Create server with official SDK pattern
	server := mcp.NewServer(&mcp.Implementation{
		Name:    config.Name,
		Version: config.Version,
	}, nil)
	log.Printf("[MCP] Created MCP server with name=%s, version=%s", config.Name, config.Version)
	
	s := &CodeContextMCPServer{
		server:   server,
		config:   config,
		analyzer: analyzer.NewGraphBuilder(),
	}
	log.Printf("[MCP] Created CodeContextMCPServer instance")

	// Register tools
	log.Printf("[MCP] Registering tools...")
	s.registerTools()
	log.Printf("[MCP] All tools registered successfully")
	
	return s, nil
}

// registerTools registers all MCP tools
func (s *CodeContextMCPServer) registerTools() {
	// Tool 1: Get codebase overview
	log.Printf("[MCP] Registering tool: get_codebase_overview")
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "get_codebase_overview",
		Description: "Get comprehensive overview of the entire codebase",
	}, s.getCodebaseOverview)

	// Tool 2: Get file analysis
	log.Printf("[MCP] Registering tool: get_file_analysis")
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "get_file_analysis",
		Description: "Get detailed analysis of a specific file",
	}, s.getFileAnalysis)

	// Tool 3: Get symbol information
	log.Printf("[MCP] Registering tool: get_symbol_info")
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "get_symbol_info",
		Description: "Get detailed information about a specific symbol",
	}, s.getSymbolInfo)

	// Tool 4: Search symbols
	log.Printf("[MCP] Registering tool: search_symbols")
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "search_symbols",
		Description: "Search for symbols across the codebase",
	}, s.searchSymbols)

	// Tool 5: Get dependencies
	log.Printf("[MCP] Registering tool: get_dependencies")
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "get_dependencies",
		Description: "Analyze import dependencies and relationships",
	}, s.getDependencies)

	// Tool 6: Watch changes (real-time)
	log.Printf("[MCP] Registering tool: watch_changes")
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "watch_changes",
		Description: "Enable/disable real-time change notifications",
	}, s.watchChanges)
	
	log.Printf("[MCP] Successfully registered 6 tools")
}

// Tool implementations

func (s *CodeContextMCPServer) getCodebaseOverview(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[GetCodebaseOverviewArgs]) (*mcp.CallToolResultFor[any], error) {
	args := params.Arguments
	log.Printf("[MCP] Tool called: get_codebase_overview with args: %+v", args)
	start := time.Now()
	
	// Ensure we have fresh analysis
	log.Printf("[MCP] Refreshing analysis for codebase overview...")
	if err := s.refreshAnalysis(); err != nil {
		log.Printf("[MCP] ERROR: Failed to refresh analysis: %v", err)
		return nil, fmt.Errorf("failed to refresh analysis: %w", err)
	}

	log.Printf("[MCP] Generating markdown content...")
	generator := analyzer.NewMarkdownGenerator(s.graph)
	content := generator.GenerateContextMap()
	log.Printf("[MCP] Generated markdown content (%d chars)", len(content))

	if args.IncludeStats {
		log.Printf("[MCP] Including detailed statistics...")
		stats := s.analyzer.GetFileStats()
		statsJson, _ := json.MarshalIndent(stats, "", "  ")
		content += "\n\n## Detailed Statistics\n```json\n" + string(statsJson) + "\n```"
		log.Printf("[MCP] Added statistics to content")
	}

	elapsed := time.Since(start)
	log.Printf("[MCP] Tool completed: get_codebase_overview (took %v)", elapsed)
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: content}},
	}, nil
}

func (s *CodeContextMCPServer) getFileAnalysis(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[GetFileAnalysisArgs]) (*mcp.CallToolResultFor[any], error) {
	args := params.Arguments
	log.Printf("[MCP] Tool called: get_file_analysis with args: %+v", args)
	start := time.Now()
	
	if args.FilePath == "" {
		log.Printf("[MCP] ERROR: file_path is required")
		return nil, fmt.Errorf("file_path is required")
	}

	// Ensure we have fresh analysis
	log.Printf("[MCP] Refreshing analysis for file: %s", args.FilePath)
	if err := s.refreshAnalysis(); err != nil {
		log.Printf("[MCP] ERROR: Failed to refresh analysis: %v", err)
		return nil, fmt.Errorf("failed to refresh analysis: %w", err)
	}

	// Find the file in our graph
	log.Printf("[MCP] Looking up file in graph: %s", args.FilePath)
	fileNode, exists := s.graph.Files[args.FilePath]
	if !exists {
		log.Printf("[MCP] ERROR: File not found in graph: %s (available files: %d)", args.FilePath, len(s.graph.Files))
		return nil, fmt.Errorf("file not found: %s", args.FilePath)
	}
	log.Printf("[MCP] Found file in graph: %s (language: %s, lines: %d, symbols: %d)", args.FilePath, fileNode.Language, fileNode.Lines, len(fileNode.Symbols))

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
	log.Printf("[MCP] Analyzing dependencies for file: %s", args.FilePath)
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
	log.Printf("[MCP] Found %d imports for file: %s", importCount, args.FilePath)

	elapsed := time.Since(start)
	log.Printf("[MCP] Tool completed: get_file_analysis (took %v)", elapsed)
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: analysis}},
	}, nil
}

func (s *CodeContextMCPServer) getSymbolInfo(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[GetSymbolInfoArgs]) (*mcp.CallToolResultFor[any], error) {
	args := params.Arguments
	log.Printf("[MCP] Tool called: get_symbol_info with args: %+v", args)
	start := time.Now()
	
	if args.SymbolName == "" {
		log.Printf("[MCP] ERROR: symbol_name is required")
		return nil, fmt.Errorf("symbol_name is required")
	}

	// Ensure we have fresh analysis
	log.Printf("[MCP] Refreshing analysis for symbol lookup: %s", args.SymbolName)
	if err := s.refreshAnalysis(); err != nil {
		log.Printf("[MCP] ERROR: Failed to refresh analysis: %v", err)
		return nil, fmt.Errorf("failed to refresh analysis: %w", err)
	}

	log.Printf("[MCP] Searching for symbol: %s in %d symbols", args.SymbolName, len(s.graph.Symbols))
	var foundSymbols []*types.Symbol
	for _, symbol := range s.graph.Symbols {
		if symbol.Name == args.SymbolName {
			foundSymbols = append(foundSymbols, symbol)
		}
	}

	log.Printf("[MCP] Found %d symbols matching '%s'", len(foundSymbols), args.SymbolName)
	if len(foundSymbols) == 0 {
		log.Printf("[MCP] ERROR: Symbol not found: %s", args.SymbolName)
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

	elapsed := time.Since(start)
	log.Printf("[MCP] Tool completed: get_symbol_info (took %v)", elapsed)
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil
}

func (s *CodeContextMCPServer) searchSymbols(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[SearchSymbolsArgs]) (*mcp.CallToolResultFor[any], error) {
	args := params.Arguments
	log.Printf("[MCP] Tool called: search_symbols with args: %+v", args)
	start := time.Now()
	
	if args.Query == "" {
		log.Printf("[MCP] ERROR: query is required")
		return nil, fmt.Errorf("query is required")
	}

	// Set default limit
	if args.Limit <= 0 {
		args.Limit = 20
	}
	log.Printf("[MCP] Searching symbols with query='%s', limit=%d", args.Query, args.Limit)

	// Ensure we have fresh analysis
	log.Printf("[MCP] Refreshing analysis for symbol search...")
	if err := s.refreshAnalysis(); err != nil {
		log.Printf("[MCP] ERROR: Failed to refresh analysis: %v", err)
		return nil, fmt.Errorf("failed to refresh analysis: %w", err)
	}

	var matches []*types.Symbol
	query := strings.ToLower(args.Query)
	log.Printf("[MCP] Searching through %d symbols for query: %s", len(s.graph.Symbols), query)

	for _, symbol := range s.graph.Symbols {
		if strings.Contains(strings.ToLower(symbol.Name), query) {
			matches = append(matches, symbol)
			if len(matches) >= args.Limit {
				log.Printf("[MCP] Reached limit of %d matches", args.Limit)
				break
			}
		}
	}

	if len(matches) == 0 {
		result := fmt.Sprintf("No symbols found matching '%s'", args.Query)
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, nil
	}

	result := fmt.Sprintf("# Symbol Search Results: '%s'\n\n", args.Query)
	result += fmt.Sprintf("Found %d matches:\n\n", len(matches))

	for _, symbol := range matches {
		result += fmt.Sprintf("- **%s** (%s) - Line %d\n", 
			symbol.Name, symbol.Kind, symbol.Location.StartLine)
	}

	elapsed := time.Since(start)
	log.Printf("[MCP] Tool completed: search_symbols (took %v, found %d matches)", elapsed, len(matches))
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil
}

func (s *CodeContextMCPServer) getDependencies(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[GetDependenciesArgs]) (*mcp.CallToolResultFor[any], error) {
	args := params.Arguments
	log.Printf("[MCP] Tool called: get_dependencies with args: %+v", args)
	start := time.Now()
	
	// Ensure we have fresh analysis
	log.Printf("[MCP] Refreshing analysis for dependency analysis...")
	if err := s.refreshAnalysis(); err != nil {
		log.Printf("[MCP] ERROR: Failed to refresh analysis: %v", err)
		return nil, fmt.Errorf("failed to refresh analysis: %w", err)
	}

	result := "# Dependency Analysis\n\n"
	log.Printf("[MCP] Analyzing %d edges for dependencies", len(s.graph.Edges))

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

	elapsed := time.Since(start)
	log.Printf("[MCP] Tool completed: get_dependencies (took %v)", elapsed)
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil
}

func (s *CodeContextMCPServer) watchChanges(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[WatchChangesArgs]) (*mcp.CallToolResultFor[any], error) {
	args := params.Arguments
	log.Printf("[MCP] Tool called: watch_changes with args: %+v", args)
	start := time.Now()
	
	if args.Enable {
		log.Printf("[MCP] Enabling file watching...")
		if s.watcher != nil {
			log.Printf("[MCP] File watching is already enabled")
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "File watching is already enabled"}},
			}, nil
		}
		
		// Create watcher config
		config := watcher.Config{
			TargetDir:    s.config.TargetDir,
			OutputFile:   "CLAUDE.md", // Not used in MCP mode
			DebounceTime: time.Duration(s.config.DebounceMs) * time.Millisecond,
			IncludeExts:  []string{".ts", ".tsx", ".js", ".jsx", ".go", ".py", ".java", ".cpp", ".c", ".rs"},
		}
		
		// Start file watcher
		log.Printf("[MCP] Creating file watcher with config: %+v", config)
		fileWatcher, err := watcher.NewFileWatcher(config)
		if err != nil {
			log.Printf("[MCP] ERROR: Failed to create file watcher: %v", err)
			return nil, fmt.Errorf("failed to start file watcher: %w", err)
		}
		
		s.watcher = fileWatcher
		log.Printf("[MCP] File watcher created successfully")
		
		// Start watching in a goroutine
		watchCtx := context.Background()
		log.Printf("[MCP] Starting file watcher goroutine...")
		go func() {
			if err := fileWatcher.Start(watchCtx); err != nil {
				log.Printf("[MCP] ERROR: File watcher error: %v", err)
			}
		}()
		
		elapsed := time.Since(start)
		log.Printf("[MCP] Tool completed: watch_changes (enable) (took %v)", elapsed)
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: "File watching enabled. Real-time change notifications are now active."}},
		}, nil
	} else {
		log.Printf("[MCP] Disabling file watching...")
		if s.watcher == nil {
			log.Printf("[MCP] File watching is not currently enabled")
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "File watching is not currently enabled"}},
			}, nil
		}
		
		log.Printf("[MCP] Stopping file watcher...")
		s.watcher.Stop()
		s.watcher = nil
		log.Printf("[MCP] File watcher stopped")
		
		elapsed := time.Since(start)
		log.Printf("[MCP] Tool completed: watch_changes (disable) (took %v)", elapsed)
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: "File watching disabled"}},
		}, nil
	}
}

// Helper methods

func (s *CodeContextMCPServer) refreshAnalysis() error {
	log.Printf("[MCP] Starting analysis of directory: %s", s.config.TargetDir)
	graph, err := s.analyzer.AnalyzeDirectory(s.config.TargetDir)
	if err != nil {
		log.Printf("[MCP] Analysis failed: %v", err)
		return err
	}
	log.Printf("[MCP] Analysis completed successfully - %d files, %d symbols", len(graph.Files), len(graph.Symbols))
	s.graph = graph
	return nil
}

// Run starts the MCP server
func (s *CodeContextMCPServer) Run(ctx context.Context) error {
	log.Printf("[MCP] CodeContext MCP Server starting - will analyze %s", s.config.TargetDir)
	
	// Initial analysis
	if err := s.refreshAnalysis(); err != nil {
		log.Printf("[MCP] Initial analysis failed, server will not start: %v", err)
		return fmt.Errorf("failed to perform initial analysis: %w", err)
	}
	
	log.Printf("[MCP] CodeContext MCP Server ready - analysis complete")
	
	// Run the MCP server with stdio transport
	return s.server.Run(ctx, mcp.NewStdioTransport())
}

// Stop gracefully stops the MCP server
func (s *CodeContextMCPServer) Stop() {
	log.Printf("[MCP] Stopping MCP server...")
	if s.watcher != nil {
		log.Printf("[MCP] Stopping file watcher...")
		s.watcher.Stop()
		s.watcher = nil
		log.Printf("[MCP] File watcher stopped")
	}
	log.Printf("[MCP] MCP server stopped successfully")
}