package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/nuthan-ms/codecontext/internal/analyzer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate initial context map",
	Long: `Generate a comprehensive context map of the codebase.
This command analyzes the entire repository and creates an intelligent
context map optimized for AI-powered development tools.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateContextMap(cmd)
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringP("target", "t", ".", "target directory to analyze")
	generateCmd.Flags().BoolP("watch", "w", false, "enable watch mode for continuous updates")
	generateCmd.Flags().StringP("format", "f", "markdown", "output format (markdown, json, yaml)")

	// Bind flags to viper with error handling
	if err := viper.BindPFlag("target", generateCmd.Flags().Lookup("target")); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to bind target flag: %v\n", err)
	}
	if err := viper.BindPFlag("watch", generateCmd.Flags().Lookup("watch")); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to bind watch flag: %v\n", err)
	}
	if err := viper.BindPFlag("format", generateCmd.Flags().Lookup("format")); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to bind format flag: %v\n", err)
	}
}

func generateContextMap(cmd *cobra.Command) error {
	start := time.Now()

	if viper.GetBool("verbose") {
		fmt.Println("🔍 Starting context map generation...")
	}

	// Get target directory from flags - try direct flag first, then viper fallback
	targetDir, err := cmd.Flags().GetString("target")
	if err != nil || targetDir == "" {
		targetDir = viper.GetString("target")
		if targetDir == "" {
			targetDir = "."
		}
	}

	outputFile := viper.GetString("output")
	if outputFile == "" {
		outputFile = "CLAUDE.md"
	}

	if viper.GetBool("verbose") {
		fmt.Printf("📁 Analyzing directory: %s\n", targetDir)
		fmt.Printf("📄 Output file: %s\n", outputFile)
		fmt.Printf("🔍 Target from flag: %s\n", targetDir)
		fmt.Printf("🔍 Target from viper: %s\n", viper.GetString("target"))
	}

	// Create graph builder and analyze directory
	builder := analyzer.NewGraphBuilder()
	graph, err := builder.AnalyzeDirectory(targetDir)
	if err != nil {
		return fmt.Errorf("failed to analyze directory: %w", err)
	}

	if viper.GetBool("verbose") {
		stats := builder.GetFileStats()
		fmt.Printf("📊 Analysis complete: %d files, %d symbols\n",
			stats["totalFiles"], stats["totalSymbols"])
	}

	// Generate markdown content from real data
	generator := analyzer.NewMarkdownGenerator(graph)
	content := generator.GenerateContextMap()

	// Write real content
	if err := writeOutputFile(outputFile, content); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	duration := time.Since(start)
	fmt.Printf("✅ Context map generated successfully in %v\n", duration)
	fmt.Printf("   Output file: %s\n", outputFile)

	return nil
}

func writeOutputFile(filename, content string) error {
	fmt.Printf("📝 Writing to %s...\n", filename)
	return os.WriteFile(filename, []byte(content), 0644)
}
