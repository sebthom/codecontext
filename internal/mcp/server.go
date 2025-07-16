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
	"github.com/nuthan-ms/codecontext/internal/git"
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

type GetSemanticNeighborhoodsArgs struct {
	FilePath     string `json:"file_path,omitempty"`
	IncludeBasic bool   `json:"include_basic,omitempty"`
	IncludeQuality bool `json:"include_quality,omitempty"`
	MaxResults   int    `json:"max_results,omitempty"`
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

	// Tool 7: Get semantic neighborhoods
	log.Printf("[MCP] Registering tool: get_semantic_neighborhoods")
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "get_semantic_neighborhoods",
		Description: "Get semantic code neighborhoods using git patterns and hierarchical clustering",
	}, s.getSemanticNeighborhoods)
	
	log.Printf("[MCP] Successfully registered 7 tools")
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

func (s *CodeContextMCPServer) getSemanticNeighborhoods(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[GetSemanticNeighborhoodsArgs]) (*mcp.CallToolResultFor[any], error) {
	start := time.Now()
	args := params.Arguments
	log.Printf("[MCP] Tool called: get_semantic_neighborhoods with args: %+v", args)

	// Ensure we have fresh analysis
	if s.graph == nil {
		if err := s.refreshAnalysis(); err != nil {
			log.Printf("[MCP] Failed to refresh analysis: %v", err)
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Failed to analyze codebase: " + err.Error()}},
			}, nil
		}
	}

	// Get semantic neighborhoods from metadata
	semanticData, err := s.getSemanticNeighborhoodsData()
	if err != nil {
		log.Printf("[MCP] Failed to get semantic neighborhoods: %v", err)
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: "Failed to get semantic neighborhoods: " + err.Error()}},
		}, nil
	}

	// Build response based on arguments
	response := s.buildSemanticNeighborhoodsResponse(semanticData, args)

	elapsed := time.Since(start)
	log.Printf("[MCP] Tool completed: get_semantic_neighborhoods (took %v)", elapsed)
	
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: response}},
	}, nil
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

// getSemanticNeighborhoodsData extracts semantic neighborhoods from the graph metadata
func (s *CodeContextMCPServer) getSemanticNeighborhoodsData() (*analyzer.SemanticAnalysisResult, error) {
	if s.graph == nil || s.graph.Metadata == nil || s.graph.Metadata.Configuration == nil {
		return nil, fmt.Errorf("no graph metadata available")
	}

	semanticInterface, exists := s.graph.Metadata.Configuration["semantic_neighborhoods"]
	if !exists {
		return nil, fmt.Errorf("no semantic neighborhoods data found - ensure this is a git repository")
	}

	semanticResult, ok := semanticInterface.(*analyzer.SemanticAnalysisResult)
	if !ok {
		return nil, fmt.Errorf("invalid semantic neighborhoods data format")
	}

	return semanticResult, nil
}

// buildSemanticNeighborhoodsResponse builds the response string for semantic neighborhoods
func (s *CodeContextMCPServer) buildSemanticNeighborhoodsResponse(data *analyzer.SemanticAnalysisResult, args GetSemanticNeighborhoodsArgs) string {
	var response strings.Builder
	
	response.WriteString("# Semantic Code Neighborhoods Analysis\n\n")
	
	// Check if git repository
	if !data.AnalysisMetadata.IsGitRepository {
		response.WriteString("❌ **Not a Git Repository**: This directory is not a git repository. Semantic neighborhoods require git history for pattern analysis.\n")
		return response.String()
	}
	
	// Handle errors
	if data.Error != "" {
		response.WriteString(fmt.Sprintf("⚠️ **Analysis Error**: %s\n\n", data.Error))
	}
	
	// Analysis overview
	metadata := data.AnalysisMetadata
	response.WriteString("## 📊 Analysis Overview\n\n")
	response.WriteString("**Git-based pattern analysis with hierarchical clustering:**\n\n")
	response.WriteString(fmt.Sprintf("- **Analysis Period**: %d days\n", metadata.AnalysisPeriodDays))
	response.WriteString(fmt.Sprintf("- **Files with Patterns**: %d files\n", metadata.FilesWithPatterns))
	response.WriteString(fmt.Sprintf("- **Basic Neighborhoods**: %d groups\n", metadata.TotalNeighborhoods))
	response.WriteString(fmt.Sprintf("- **Clustered Groups**: %d clusters\n", metadata.TotalClusters))
	response.WriteString(fmt.Sprintf("- **Average Cluster Size**: %.1f files\n", metadata.AverageClusterSize))
	response.WriteString(fmt.Sprintf("- **Analysis Time**: %v\n", metadata.AnalysisTime))
	
	if metadata.QualityScores.OverallQualityRating != "" {
		response.WriteString(fmt.Sprintf("- **Clustering Quality**: %s\n", metadata.QualityScores.OverallQualityRating))
	}
	response.WriteString("\n")
	
	// Context recommendations based on file path
	if args.FilePath != "" {
		response.WriteString(s.buildFileContextRecommendations(data, args.FilePath))
	}
	
	// Basic neighborhoods (if requested)
	if args.IncludeBasic && len(data.SemanticNeighborhoods) > 0 {
		response.WriteString("## 🔍 Basic Semantic Neighborhoods\n\n")
		response.WriteString(s.buildBasicNeighborhoodsResponse(data.SemanticNeighborhoods, args.MaxResults))
	}
	
	// Clustered neighborhoods (always include if available)
	if len(data.ClusteredNeighborhoods) > 0 {
		response.WriteString("## 🎯 Clustered Neighborhoods\n\n")
		response.WriteString(s.buildClusteredNeighborhoodsResponse(data.ClusteredNeighborhoods, args.MaxResults))
	}
	
	// Quality metrics (if requested)
	if args.IncludeQuality && len(data.ClusteredNeighborhoods) > 0 {
		response.WriteString("## 📈 Quality Metrics\n\n")
		response.WriteString(s.buildQualityMetricsResponse(data))
	}
	
	// No neighborhoods found
	if len(data.SemanticNeighborhoods) == 0 && len(data.ClusteredNeighborhoods) == 0 {
		response.WriteString("## 🏷️ No Neighborhoods Found\n\n")
		response.WriteString("No semantic neighborhoods were detected. This could mean:\n")
		response.WriteString("- Files don't frequently change together\n")
		response.WriteString("- Insufficient git history (need at least a few commits)\n")
		response.WriteString("- Repository primarily contains single-purpose files\n")
		response.WriteString("- Analysis period too short (default: 30 days)\n")
	}
	
	return response.String()
}

// buildFileContextRecommendations builds context recommendations for a specific file
func (s *CodeContextMCPServer) buildFileContextRecommendations(data *analyzer.SemanticAnalysisResult, filePath string) string {
	var response strings.Builder
	
	response.WriteString(fmt.Sprintf("## 🎯 Context Recommendations for `%s`\n\n", filePath))
	
	// Find neighborhoods containing this file
	relatedNeighborhoods := []string{}
	relatedClusters := []string{}
	
	// Check basic neighborhoods
	for _, neighborhood := range data.SemanticNeighborhoods {
		for _, file := range neighborhood.Files {
			if strings.Contains(file, filePath) || strings.Contains(filePath, file) {
				relatedNeighborhoods = append(relatedNeighborhoods, neighborhood.Name)
				break
			}
		}
	}
	
	// Check clustered neighborhoods
	for i, clustered := range data.ClusteredNeighborhoods {
		for _, neighborhood := range clustered.Neighborhoods {
			for _, file := range neighborhood.Files {
				if strings.Contains(file, filePath) || strings.Contains(filePath, file) {
					relatedClusters = append(relatedClusters, fmt.Sprintf("Cluster %d: %s", i+1, clustered.Cluster.Name))
					break
				}
			}
		}
	}
	
	if len(relatedNeighborhoods) > 0 {
		response.WriteString("**Related Neighborhoods:**\n")
		for _, neighborhood := range relatedNeighborhoods {
			response.WriteString(fmt.Sprintf("- %s\n", neighborhood))
		}
		response.WriteString("\n")
	}
	
	if len(relatedClusters) > 0 {
		response.WriteString("**Related Clusters:**\n")
		for _, cluster := range relatedClusters {
			response.WriteString(fmt.Sprintf("- %s\n", cluster))
		}
		response.WriteString("\n")
	}
	
	if len(relatedNeighborhoods) == 0 && len(relatedClusters) == 0 {
		response.WriteString("**No direct relationships found.** This file may be independent or have weak patterns with other files.\n\n")
	}
	
	return response.String()
}

// buildBasicNeighborhoodsResponse builds the basic neighborhoods response
func (s *CodeContextMCPServer) buildBasicNeighborhoodsResponse(neighborhoods []git.SemanticNeighborhood, maxResults int) string {
	var response strings.Builder
	
	// Sort by correlation strength
	sortedNeighborhoods := make([]git.SemanticNeighborhood, len(neighborhoods))
	copy(sortedNeighborhoods, neighborhoods)
	
	limit := len(sortedNeighborhoods)
	if maxResults > 0 && maxResults < limit {
		limit = maxResults
	}
	
	for i := 0; i < limit; i++ {
		neighborhood := sortedNeighborhoods[i]
		response.WriteString(fmt.Sprintf("### %s\n\n", neighborhood.Name))
		response.WriteString(fmt.Sprintf("- **Correlation**: %.2f\n", neighborhood.CorrelationStrength))
		response.WriteString(fmt.Sprintf("- **Changes**: %d\n", neighborhood.ChangeFrequency))
		response.WriteString(fmt.Sprintf("- **Files**: %d\n", len(neighborhood.Files)))
		response.WriteString(fmt.Sprintf("- **Last Changed**: %s\n", neighborhood.LastChanged.Format("2006-01-02")))
		
		if len(neighborhood.Files) > 0 {
			response.WriteString("\n**Files:**\n")
			for _, file := range neighborhood.Files {
				response.WriteString(fmt.Sprintf("- `%s`\n", file))
			}
		}
		response.WriteString("\n")
	}
	
	return response.String()
}

// buildClusteredNeighborhoodsResponse builds the clustered neighborhoods response
func (s *CodeContextMCPServer) buildClusteredNeighborhoodsResponse(clusteredNeighborhoods []git.ClusteredNeighborhood, maxResults int) string {
	var response strings.Builder
	
	limit := len(clusteredNeighborhoods)
	if maxResults > 0 && maxResults < limit {
		limit = maxResults
	}
	
	for i := 0; i < limit; i++ {
		clustered := clusteredNeighborhoods[i]
		cluster := clustered.Cluster
		
		response.WriteString(fmt.Sprintf("### Cluster %d: %s\n\n", i+1, cluster.Name))
		response.WriteString(fmt.Sprintf("- **Description**: %s\n", cluster.Description))
		response.WriteString(fmt.Sprintf("- **Size**: %d files\n", cluster.Size))
		response.WriteString(fmt.Sprintf("- **Strength**: %.3f\n", cluster.Strength))
		response.WriteString(fmt.Sprintf("- **Silhouette Score**: %.3f\n", clustered.QualityMetrics.SilhouetteScore))
		response.WriteString(fmt.Sprintf("- **Cohesion**: %.3f\n", cluster.IntraMetrics.Cohesion))
		
		if len(cluster.OptimalTasks) > 0 {
			response.WriteString("\n**Recommended Tasks:**\n")
			for _, task := range cluster.OptimalTasks {
				response.WriteString(fmt.Sprintf("- %s\n", task))
			}
		}
		
		if cluster.RecommendationReason != "" {
			response.WriteString(fmt.Sprintf("\n**Why**: %s\n", cluster.RecommendationReason))
		}
		
		response.WriteString("\n")
	}
	
	return response.String()
}

// buildQualityMetricsResponse builds the quality metrics response
func (s *CodeContextMCPServer) buildQualityMetricsResponse(data *analyzer.SemanticAnalysisResult) string {
	var response strings.Builder
	
	scores := data.AnalysisMetadata.QualityScores
	
	response.WriteString("**Overall Clustering Performance:**\n\n")
	response.WriteString(fmt.Sprintf("- **Average Silhouette Score**: %.3f\n", scores.AverageSilhouetteScore))
	response.WriteString(fmt.Sprintf("- **Average Davies-Bouldin Index**: %.3f\n", scores.AverageDaviesBouldinIndex))
	response.WriteString(fmt.Sprintf("- **Quality Rating**: %s\n\n", scores.OverallQualityRating))
	
	response.WriteString("**Interpretation:**\n")
	response.WriteString("- **Silhouette Score**: 0.7+ Excellent, 0.5+ Good, 0.25+ Fair, <0.25 Poor\n")
	response.WriteString("- **Davies-Bouldin**: Lower values indicate better clustering\n")
	response.WriteString("- **Algorithm**: Hierarchical clustering with Ward linkage\n")
	
	return response.String()
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