package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/nuthan-ms/codecontext/internal/analyzer"
	"github.com/nuthan-ms/codecontext/internal/compact"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var compactCmd = &cobra.Command{
	Use:   "compact",
	Short: "Optimize context map with compaction strategies",
	Long: `Apply compaction strategies to optimize the context map for specific tasks.
This command provides interactive context optimization with different levels
and task-specific strategies.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeCompaction(cmd)
	},
}

func init() {
	rootCmd.AddCommand(compactCmd)
	compactCmd.Flags().StringP("level", "l", "balanced", "compaction level (minimal, balanced, aggressive)")
	compactCmd.Flags().StringP("task", "t", "", "task-specific optimization (debugging, refactoring, documentation)")
	compactCmd.Flags().IntP("tokens", "n", 0, "target token limit")
	compactCmd.Flags().BoolP("preview", "p", false, "preview compaction without applying")
	compactCmd.Flags().StringSliceP("focus", "f", []string{}, "focus on specific files or directories")
}

func executeCompaction(cmd *cobra.Command) error {
	level, _ := cmd.Flags().GetString("level")
	task, _ := cmd.Flags().GetString("task")
	tokens, _ := cmd.Flags().GetInt("tokens")
	preview, _ := cmd.Flags().GetBool("preview")

	if viper.GetBool("verbose") {
		fmt.Println("🔧 Starting context compaction...")
		fmt.Printf("   Level: %s\n", level)
		if task != "" {
			fmt.Printf("   Task: %s\n", task)
		}
		if tokens > 0 {
			fmt.Printf("   Token limit: %d\n", tokens)
		}
		if preview {
			fmt.Println("   Mode: Preview only")
		}
	}

	// Read existing context map to get the graph
	inputFile := viper.GetString("output")
	if inputFile == "" {
		inputFile = "CLAUDE.md"
	}

	// Check if context map exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("context map not found: %s. Run 'generate' first", inputFile)
	}

	// Get target directory for analysis
	targetDir := viper.GetString("target")
	if targetDir == "" {
		targetDir = "."
	}

	// Build graph from directory
	builder := analyzer.NewGraphBuilder()
	graph, err := builder.AnalyzeDirectory(targetDir)
	if err != nil {
		return fmt.Errorf("failed to analyze directory: %w", err)
	}

	// Create compaction controller
	controller := compact.NewCompactController(compact.DefaultCompactConfig())

	// Map CLI level to strategy
	strategy := mapLevelToStrategy(level)
	if task != "" {
		strategy = mapTaskToStrategy(task, strategy)
	}

	// Create compaction request
	request := &compact.CompactRequest{
		Graph:    graph,
		Strategy: strategy,
		MaxSize:  tokens,
		Requirements: &compact.CompactRequirements{
			PreserveFiles: getFocusFiles(cmd),
		},
	}

	// Execute compaction
	ctx := context.Background()
	result, err := controller.Compact(ctx, request)
	if err != nil {
		return fmt.Errorf("compaction failed: %w", err)
	}

	// Calculate metrics
	originalTokens := result.OriginalSize
	compactedTokens := result.CompactedSize
	reductionPercent := (1.0 - result.CompressionRatio) * 100

	if preview {
		fmt.Printf("📊 Compaction Preview:\n")
		fmt.Printf("   Original tokens: %d\n", originalTokens)
		fmt.Printf("   Compacted tokens: %d\n", compactedTokens)
		fmt.Printf("   Reduction: %.1f%%\n", reductionPercent)
		fmt.Printf("   Strategy: %s\n", result.Strategy)
		fmt.Printf("   Processing time: %v\n", result.ExecutionTime)
		fmt.Printf("   Files removed: %d\n", len(result.RemovedItems.Files))
		fmt.Printf("   Symbols removed: %d\n", len(result.RemovedItems.Symbols))
		fmt.Println("   Run without --preview to apply changes")
	} else {
		// Generate and write compacted context map
		generator := analyzer.NewMarkdownGenerator(result.CompactedGraph)
		compactedContent := generator.GenerateContextMap()
		
		// Write to output file
		outputFile := inputFile
		if err := os.WriteFile(outputFile, []byte(compactedContent), 0644); err != nil {
			return fmt.Errorf("failed to write compacted context map: %w", err)
		}
		
		fmt.Printf("✅ Context compaction completed in %v\n", result.ExecutionTime)
		fmt.Printf("   Token reduction: %.1f%% (%d → %d)\n", reductionPercent, originalTokens, compactedTokens)
		fmt.Printf("   Strategy: %s\n", result.Strategy)
		fmt.Printf("   Files removed: %d\n", len(result.RemovedItems.Files))
		fmt.Printf("   Symbols removed: %d\n", len(result.RemovedItems.Symbols))
		fmt.Printf("   Output file: %s\n", outputFile)
	}

	return nil
}

func mapLevelToStrategy(level string) string {
	switch level {
	case "minimal":
		return "relevance"
	case "balanced":
		return "hybrid"
	case "aggressive":
		return "size"
	default:
		return "hybrid"
	}
}

func mapTaskToStrategy(task, defaultStrategy string) string {
	switch task {
	case "debugging":
		return "dependency"
	case "refactoring":
		return "hybrid"
	case "documentation":
		return "relevance"
	default:
		return defaultStrategy
	}
}

func getFocusFiles(cmd *cobra.Command) []string {
	focus, _ := cmd.Flags().GetStringSlice("focus")
	return focus
}

// Test helper functions - kept for backward compatibility with tests
func getReductionFactor(level string) float64 {
	switch level {
	case "minimal":
		return 0.3
	case "balanced":
		return 0.6
	case "aggressive":
		return 0.15
	default:
		return 0.6
	}
}

func getQualityScore(level string) float64 {
	switch level {
	case "minimal":
		return 0.95
	case "balanced":
		return 0.85
	case "aggressive":
		return 0.70
	default:
		return 0.85
	}
}
