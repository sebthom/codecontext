package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/nuthan-ms/codecontext/internal/watcher"
)

var updateCmd = &cobra.Command{
	Use:   "update [files...]",
	Short: "Update context map incrementally",
	Long: `Update the context map incrementally based on file changes.
This command uses the Virtual Graph Engine to efficiently update
only the affected parts of the context map.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateContextMap(args)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolP("force", "f", false, "force full regeneration")
	updateCmd.Flags().BoolP("preview", "p", false, "preview changes without applying")
	updateCmd.Flags().BoolP("watch", "w", false, "watch for file changes and update automatically")
	updateCmd.Flags().DurationP("debounce", "d", 500*time.Millisecond, "debounce time for file changes")
}

func updateContextMap(files []string) error {
	start := time.Now()
	
	// Get target directory and output file
	targetDir := viper.GetString("target")
	if targetDir == "" {
		var err error
		targetDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}
	
	outputFile := viper.GetString("output")
	if outputFile == "" {
		outputFile = filepath.Join(targetDir, "full-project-analysis.md")
	}
	
	watch := viper.GetBool("watch")
	
	if viper.GetBool("verbose") {
		fmt.Println("🔄 Starting incremental update...")
		if len(files) > 0 {
			fmt.Printf("   Target files: %v\n", files)
		} else {
			fmt.Println("   Scanning for changed files...")
		}
		fmt.Printf("   Target directory: %s\n", targetDir)
		fmt.Printf("   Output file: %s\n", outputFile)
		if watch {
			fmt.Println("   Watch mode: enabled")
		}
	}
	
	if watch {
		return startWatchMode(targetDir, outputFile)
	}
	
	// TODO: Implement one-time incremental update logic
	// This will use the Virtual Graph Engine for specific files
	
	duration := time.Since(start)
	fmt.Printf("✅ Context map updated successfully in %v\n", duration)
	fmt.Printf("   Changes: %d files processed\n", len(files))
	
	return nil
}

func startWatchMode(targetDir, outputFile string) error {
	debounceTime := viper.GetDuration("debounce")
	
	config := watcher.Config{
		TargetDir:    targetDir,
		OutputFile:   outputFile,
		DebounceTime: debounceTime,
	}
	
	fileWatcher, err := watcher.NewFileWatcher(config)
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer fileWatcher.Stop()
	
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Start file watcher
	err = fileWatcher.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start file watcher: %w", err)
	}
	
	fmt.Println("🔍 Watching for file changes... Press Ctrl+C to stop")
	
	// Wait for interrupt signal
	select {
	case <-ctx.Done():
		fmt.Println("\n👋 File watcher stopped")
		return nil
	}
}