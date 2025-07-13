package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nuthan-ms/codecontext/internal/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP (Model Context Protocol) server",
	Long: `Start a Model Context Protocol server that provides real-time codebase context 
to AI assistants. The server exposes tools for analyzing code structure, searching symbols,
tracking dependencies, and monitoring file changes.

The MCP server uses standard I/O transport and can be integrated with AI applications
like Claude Desktop, VSCode extensions, or custom MCP clients.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMCPServer()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	
	// MCP-specific flags
	mcpCmd.Flags().StringP("target", "t", ".", "target directory to analyze")
	mcpCmd.Flags().BoolP("watch", "w", true, "enable real-time file watching")
	mcpCmd.Flags().IntP("debounce", "d", 500, "debounce interval for file changes (ms)")
	mcpCmd.Flags().StringP("name", "n", "codecontext", "MCP server name")

	// Bind flags to viper
	viper.BindPFlag("mcp.target", mcpCmd.Flags().Lookup("target"))
	viper.BindPFlag("mcp.watch", mcpCmd.Flags().Lookup("watch"))
	viper.BindPFlag("mcp.debounce", mcpCmd.Flags().Lookup("debounce"))
	viper.BindPFlag("mcp.name", mcpCmd.Flags().Lookup("name"))
}

func runMCPServer() error {
	// Get configuration from flags/config
	targetDir := viper.GetString("mcp.target")
	if targetDir == "" {
		targetDir = "."
	}

	config := &mcp.MCPConfig{
		Name:        viper.GetString("mcp.name"),
		Version:     appVersion,
		TargetDir:   targetDir,
		EnableWatch: viper.GetBool("mcp.watch"),
		DebounceMs:  viper.GetInt("mcp.debounce"),
	}

	if viper.GetBool("verbose") {
		fmt.Printf("🚀 Starting CodeContext MCP Server\n")
		fmt.Printf("   Name: %s\n", config.Name)
		fmt.Printf("   Version: %s\n", config.Version)
		fmt.Printf("   Target Directory: %s\n", config.TargetDir)
		fmt.Printf("   Watch Mode: %v\n", config.EnableWatch)
		if config.EnableWatch {
			fmt.Printf("   Debounce Interval: %dms\n", config.DebounceMs)
		}
		fmt.Printf("   Transport: Standard I/O\n")
		fmt.Printf("\n")
	}

	// Create MCP server
	server, err := mcp.NewCodeContextMCPServer(config)
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		if viper.GetBool("verbose") {
			fmt.Fprintf(os.Stderr, "\n🛑 Received shutdown signal, stopping MCP server...\n")
		}
		server.Stop()
		cancel()
	}()

	// Start the MCP server
	if viper.GetBool("verbose") {
		fmt.Printf("🔌 MCP Server ready - waiting for client connections\n")
		fmt.Printf("   Available tools:\n")
		fmt.Printf("   • get_codebase_overview  - Complete repository analysis\n")
		fmt.Printf("   • get_file_analysis      - Detailed file breakdown\n")
		fmt.Printf("   • get_symbol_info        - Symbol definitions and usage\n")
		fmt.Printf("   • search_symbols         - Search symbols across codebase\n")
		fmt.Printf("   • get_dependencies       - Import/dependency analysis\n")
		fmt.Printf("   • watch_changes          - Real-time change notifications\n")
		fmt.Printf("\n")
	}

	err = server.Run(ctx)
	if err != nil {
		return fmt.Errorf("MCP server error: %w", err)
	}

	if viper.GetBool("verbose") {
		fmt.Printf("✅ MCP Server stopped gracefully\n")
	}

	return nil
}