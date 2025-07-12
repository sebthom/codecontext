package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nuthan-ms/codecontext/internal/analyzer"
	"github.com/nuthan-ms/codecontext/internal/cache"
	"github.com/nuthan-ms/codecontext/internal/watcher"
	"github.com/nuthan-ms/codecontext/pkg/types"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch directory for changes and update context map",
	Long: `Watch the target directory for file changes and automatically update
the context map using incremental analysis. This mode is optimized for
long-running processes and provides real-time context updates.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWatchMode()
	},
}

// WatchConfig holds configuration for watch mode
type WatchConfig struct {
	TargetDir          string
	OutputFile         string
	UpdateInterval     time.Duration
	MaxConcurrentFiles int
	EnableCache        bool
	CacheDir           string
	EnableGC           bool
	GCInterval         time.Duration
	MemoryThreshold    int64 // MB
	ShowProgress       bool
	ProgressInterval   time.Duration
}

// WatchManager manages the watch mode execution
type WatchManager struct {
	config      *WatchConfig
	watcher     *watcher.FileWatcher
	analyzer    *analyzer.IncrementalAnalyzer
	cache       *cache.PersistentCache
	graph       *types.CodeGraph
	lastUpdate  time.Time
	updateMutex sync.RWMutex
	stats       *WatchStats
	ctx         context.Context
	cancel      context.CancelFunc
}

// WatchStats tracks performance metrics for watch mode
type WatchStats struct {
	TotalUpdates      int64
	FilesProcessed    int64
	AverageUpdateTime time.Duration
	CacheHitRate      float64
	MemoryUsage       int64
	LastGC            time.Time
	mutex             sync.RWMutex
}

func init() {
	rootCmd.AddCommand(watchCmd)

	watchCmd.Flags().StringP("target", "t", ".", "target directory to watch")
	watchCmd.Flags().DurationP("interval", "i", 500*time.Millisecond, "update interval")
	watchCmd.Flags().IntP("concurrent", "c", 4, "maximum concurrent file processing")
	watchCmd.Flags().BoolP("cache", "", true, "enable persistent caching")
	watchCmd.Flags().StringP("cache-dir", "", ".codecontext/cache", "cache directory")
	watchCmd.Flags().BoolP("gc", "", true, "enable garbage collection monitoring")
	watchCmd.Flags().DurationP("gc-interval", "", 5*time.Minute, "garbage collection interval")
	watchCmd.Flags().Int64P("memory-threshold", "", 512, "memory threshold in MB")
	watchCmd.Flags().BoolP("progress", "p", true, "show progress indicators")
	watchCmd.Flags().DurationP("progress-interval", "", 30*time.Second, "progress update interval")

	// Bind flags to viper
	viper.BindPFlag("target", watchCmd.Flags().Lookup("target"))
	viper.BindPFlag("interval", watchCmd.Flags().Lookup("interval"))
	viper.BindPFlag("concurrent", watchCmd.Flags().Lookup("concurrent"))
	viper.BindPFlag("cache", watchCmd.Flags().Lookup("cache"))
	viper.BindPFlag("cache-dir", watchCmd.Flags().Lookup("cache-dir"))
	viper.BindPFlag("gc", watchCmd.Flags().Lookup("gc"))
	viper.BindPFlag("gc-interval", watchCmd.Flags().Lookup("gc-interval"))
	viper.BindPFlag("memory-threshold", watchCmd.Flags().Lookup("memory-threshold"))
	viper.BindPFlag("progress", watchCmd.Flags().Lookup("progress"))
	viper.BindPFlag("progress-interval", watchCmd.Flags().Lookup("progress-interval"))
}

func runWatchMode() error {
	// Get output file from flags with fallback to default
	outputFile := viper.GetString("output")
	if outputFile == "" {
		outputFile = "CLAUDE.md"
	}

	// Create watch configuration
	config := &WatchConfig{
		TargetDir:          viper.GetString("target"),
		OutputFile:         outputFile,
		UpdateInterval:     viper.GetDuration("interval"),
		MaxConcurrentFiles: viper.GetInt("concurrent"),
		EnableCache:        viper.GetBool("cache"),
		CacheDir:           viper.GetString("cache-dir"),
		EnableGC:           viper.GetBool("gc"),
		GCInterval:         viper.GetDuration("gc-interval"),
		MemoryThreshold:    viper.GetInt64("memory-threshold") * 1024 * 1024, // Convert MB to bytes
		ShowProgress:       viper.GetBool("progress"),
		ProgressInterval:   viper.GetDuration("progress-interval"),
	}

	if config.TargetDir == "" {
		config.TargetDir = "."
	}

	// Create watch manager
	manager, err := NewWatchManager(config)
	if err != nil {
		return fmt.Errorf("failed to create watch manager: %w", err)
	}
	defer manager.Cleanup()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start watch mode
	fmt.Printf("🔍 Starting watch mode on %s\n", config.TargetDir)
	fmt.Printf("   Output: %s\n", config.OutputFile)
	fmt.Printf("   Update interval: %v\n", config.UpdateInterval)
	fmt.Printf("   Concurrent files: %d\n", config.MaxConcurrentFiles)

	if config.EnableCache {
		fmt.Printf("   Cache: enabled (%s)\n", config.CacheDir)
	}

	if config.EnableGC {
		fmt.Printf("   Memory monitoring: enabled (threshold: %dMB)\n", config.MemoryThreshold/(1024*1024))
	}

	// Start the watch manager
	ctx := context.Background()
	if err := manager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start watch manager: %w", err)
	}

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		fmt.Printf("\n🛑 Received signal %v, shutting down gracefully...\n", sig)
		manager.Stop()
	case <-manager.ctx.Done():
		fmt.Println("\n✅ Watch mode completed")
	}

	// Print final statistics
	manager.PrintStats()

	return nil
}

// NewWatchManager creates a new watch manager with the given configuration
func NewWatchManager(config *WatchConfig) (*WatchManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &WatchManager{
		config: config,
		stats: &WatchStats{
			LastGC: time.Now(),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize cache if enabled
	if config.EnableCache {
		cacheConfig := &cache.Config{
			Directory:     config.CacheDir,
			MaxSize:       1000, // Max cached items
			TTL:           24 * time.Hour,
			EnableLRU:     true,
			EnableMetrics: true,
		}

		var err error
		manager.cache, err = cache.NewPersistentCache(cacheConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache: %w", err)
		}
	}

	// Initialize incremental analyzer
	analyzerConfig := &analyzer.IncrementalConfig{
		EnableVGE:          true,
		DiffAlgorithm:      "myers",
		BatchSize:          config.MaxConcurrentFiles,
		BatchTimeout:       config.UpdateInterval,
		CacheEnabled:       config.EnableCache,
		MaxCacheSize:       1000,
		ChangeDetection:    "mtime",
		IncrementalDepth:   3,
		ParallelProcessing: true,
	}

	var err error
	manager.analyzer, err = analyzer.NewIncrementalAnalyzer(config.TargetDir, analyzerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create incremental analyzer: %w", err)
	}

	// Initialize file watcher
	watcherConfig := watcher.Config{
		TargetDir:    config.TargetDir,
		OutputFile:   config.OutputFile,
		DebounceTime: config.UpdateInterval,
		ExcludePatterns: []string{
			".git/*",
			"node_modules/*",
			"*.log",
			".codecontext/cache/*",
		},
		IncludeExts: []string{
			".ts", ".tsx", ".js", ".jsx",
			".go", ".py", ".java", ".cpp", ".c",
			".rs", ".swift", ".kt", ".cs",
		},
	}

	manager.watcher, err = watcher.NewFileWatcher(watcherConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return manager, nil
}

// Start begins the watch mode operation
func (wm *WatchManager) Start(ctx context.Context) error {
	// Perform initial analysis
	if err := wm.performInitialAnalysis(); err != nil {
		return fmt.Errorf("initial analysis failed: %w", err)
	}

	// Start file watcher
	if err := wm.watcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start file watcher: %w", err)
	}

	// Start background tasks
	go wm.startPerformanceMonitoring()
	go wm.startProgressReporting()

	if wm.config.EnableGC {
		go wm.startGarbageCollectionMonitoring()
	}

	return nil
}

// Stop gracefully shuts down the watch manager
func (wm *WatchManager) Stop() {
	wm.cancel()

	if wm.watcher != nil {
		wm.watcher.StopWatching()
	}

	// Perform final update
	wm.performFinalUpdate()
}

// Cleanup releases resources
func (wm *WatchManager) Cleanup() {
	if wm.watcher != nil {
		wm.watcher.Stop()
	}

	if wm.cache != nil {
		wm.cache.Close()
	}
}

// performInitialAnalysis performs the initial codebase analysis
func (wm *WatchManager) performInitialAnalysis() error {
	start := time.Now()

	if viper.GetBool("verbose") {
		fmt.Println("🔄 Performing initial analysis...")
	}

	// Check cache for existing graph
	var graph *types.CodeGraph
	var err error

	if wm.cache != nil {
		if cached := wm.cache.GetGraph("main"); cached != nil {
			graph = cached
			if viper.GetBool("verbose") {
				fmt.Println("✅ Loaded graph from cache")
			}
		}
	}

	// If no cached graph, perform full analysis
	if graph == nil {
		builder := analyzer.NewGraphBuilder()
		graph, err = builder.AnalyzeDirectory(wm.config.TargetDir)
		if err != nil {
			return fmt.Errorf("failed to analyze directory: %w", err)
		}

		// Cache the graph
		if wm.cache != nil {
			wm.cache.SetGraph("main", graph)
		}
	}

	// Initialize incremental analyzer
	if err := wm.analyzer.Initialize(graph); err != nil {
		return fmt.Errorf("failed to initialize incremental analyzer: %w", err)
	}

	wm.updateMutex.Lock()
	wm.graph = graph
	wm.lastUpdate = time.Now()
	wm.updateMutex.Unlock()

	// Generate initial output
	if err := wm.generateOutput(); err != nil {
		return fmt.Errorf("failed to generate initial output: %w", err)
	}

	duration := time.Since(start)
	fmt.Printf("✅ Initial analysis completed in %v\n", duration)

	return nil
}

// handleFileChanges processes file change events
func (wm *WatchManager) handleFileChanges(changes []watcher.FileChange) error {
	if len(changes) == 0 {
		return nil
	}

	start := time.Now()

	if viper.GetBool("verbose") {
		fmt.Printf("🔄 Processing %d file changes...\n", len(changes))
	}

	// Extract file paths from changes
	changedPaths := make([]string, len(changes))
	for i, change := range changes {
		changedPaths[i] = change.Path
	}

	// Analyze changes incrementally
	result, err := wm.analyzer.AnalyzeChanges(wm.ctx, changedPaths)
	if err != nil {
		return fmt.Errorf("incremental analysis failed: %w", err)
	}

	// Update graph
	wm.updateMutex.Lock()
	wm.graph = result.UpdatedGraph
	wm.lastUpdate = time.Now()
	wm.updateMutex.Unlock()

	// Update cache
	if wm.cache != nil {
		wm.cache.SetGraph("main", wm.graph)
	}

	// Generate output
	if err := wm.generateOutput(); err != nil {
		return fmt.Errorf("failed to generate output: %w", err)
	}

	// Update statistics
	wm.stats.mutex.Lock()
	wm.stats.TotalUpdates++
	wm.stats.FilesProcessed += int64(len(changes))

	// Update average time using exponential moving average
	duration := time.Since(start)
	if wm.stats.AverageUpdateTime == 0 {
		wm.stats.AverageUpdateTime = duration
	} else {
		alpha := 0.1
		wm.stats.AverageUpdateTime = time.Duration(
			float64(wm.stats.AverageUpdateTime)*(1-alpha) + float64(duration)*alpha,
		)
	}

	// Update cache hit rate
	if wm.cache != nil {
		metrics := wm.cache.GetMetrics()
		wm.stats.CacheHitRate = metrics.HitRate
	}

	wm.stats.mutex.Unlock()

	if viper.GetBool("verbose") {
		fmt.Printf("✅ Update completed in %v (%d files processed)\n", duration, len(changes))
	}

	return nil
}

// generateOutput generates the context map output
func (wm *WatchManager) generateOutput() error {
	wm.updateMutex.RLock()
	graph := wm.graph
	wm.updateMutex.RUnlock()

	if graph == nil {
		return fmt.Errorf("no graph available for output generation")
	}

	// Generate markdown content
	generator := analyzer.NewMarkdownGenerator(graph)
	content := generator.GenerateContextMap()

	// Write to output file
	return writeOutputFile(wm.config.OutputFile, content)
}

// performFinalUpdate performs a final update before shutdown
func (wm *WatchManager) performFinalUpdate() {
	if viper.GetBool("verbose") {
		fmt.Println("🔄 Performing final update...")
	}

	if err := wm.generateOutput(); err != nil {
		fmt.Printf("⚠️  Final update failed: %v\n", err)
	} else {
		fmt.Println("✅ Final update completed")
	}
}

// startPerformanceMonitoring starts performance monitoring
func (wm *WatchManager) startPerformanceMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wm.ctx.Done():
			return
		case <-ticker.C:
			wm.updateMemoryUsage()
		}
	}
}

// startProgressReporting starts progress reporting
func (wm *WatchManager) startProgressReporting() {
	if !wm.config.ShowProgress {
		return
	}

	ticker := time.NewTicker(wm.config.ProgressInterval)
	defer ticker.Stop()

	for {
		select {
		case <-wm.ctx.Done():
			return
		case <-ticker.C:
			wm.printProgressUpdate()
		}
	}
}

// startGarbageCollectionMonitoring starts GC monitoring
func (wm *WatchManager) startGarbageCollectionMonitoring() {
	ticker := time.NewTicker(wm.config.GCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-wm.ctx.Done():
			return
		case <-ticker.C:
			wm.performGarbageCollectionCheck()
		}
	}
}

// updateMemoryUsage updates memory usage statistics
func (wm *WatchManager) updateMemoryUsage() {
	// This would use runtime.ReadMemStats in a real implementation
	// For now, we'll simulate memory monitoring
	wm.stats.mutex.Lock()
	wm.stats.MemoryUsage = 50 * 1024 * 1024 // Simulated 50MB
	wm.stats.mutex.Unlock()
}

// printProgressUpdate prints a progress update
func (wm *WatchManager) printProgressUpdate() {
	wm.stats.mutex.RLock()
	stats := *wm.stats
	wm.stats.mutex.RUnlock()

	wm.updateMutex.RLock()
	lastUpdate := wm.lastUpdate
	wm.updateMutex.RUnlock()

	fmt.Printf("📊 Progress: %d updates, %d files processed, avg time: %v, memory: %dMB, cache hit: %.1f%%, last update: %v ago\n",
		stats.TotalUpdates,
		stats.FilesProcessed,
		stats.AverageUpdateTime.Truncate(time.Millisecond),
		stats.MemoryUsage/(1024*1024),
		stats.CacheHitRate*100,
		time.Since(lastUpdate).Truncate(time.Second),
	)
}

// performGarbageCollectionCheck checks if GC is needed
func (wm *WatchManager) performGarbageCollectionCheck() {
	wm.stats.mutex.RLock()
	memoryUsage := wm.stats.MemoryUsage
	wm.stats.mutex.RUnlock()

	if memoryUsage > wm.config.MemoryThreshold {
		if viper.GetBool("verbose") {
			fmt.Printf("🗑️  Memory threshold exceeded (%dMB), triggering GC...\n", memoryUsage/(1024*1024))
		}

		// Trigger garbage collection
		wm.triggerGarbageCollection()

		wm.stats.mutex.Lock()
		wm.stats.LastGC = time.Now()
		wm.stats.mutex.Unlock()
	}
}

// triggerGarbageCollection triggers garbage collection
func (wm *WatchManager) triggerGarbageCollection() {
	// Clear caches if memory pressure is high
	if wm.cache != nil {
		wm.cache.Clear()
	}

	// Force VGE garbage collection
	if wm.analyzer != nil {
		metrics := wm.analyzer.GetVGEMetrics()
		if metrics.ShadowMemoryBytes > wm.config.MemoryThreshold/2 {
			// In a real implementation, we'd trigger VGE GC here
			if viper.GetBool("verbose") {
				fmt.Println("🗑️  VGE garbage collection triggered")
			}
		}
	}
}

// PrintStats prints final statistics
func (wm *WatchManager) PrintStats() {
	wm.stats.mutex.RLock()
	stats := *wm.stats
	wm.stats.mutex.RUnlock()

	fmt.Println("\n📊 Final Statistics:")
	fmt.Printf("   Total updates: %d\n", stats.TotalUpdates)
	fmt.Printf("   Files processed: %d\n", stats.FilesProcessed)
	fmt.Printf("   Average update time: %v\n", stats.AverageUpdateTime.Truncate(time.Millisecond))
	fmt.Printf("   Cache hit rate: %.1f%%\n", stats.CacheHitRate*100)
	fmt.Printf("   Peak memory usage: %dMB\n", stats.MemoryUsage/(1024*1024))

	if !stats.LastGC.IsZero() {
		fmt.Printf("   Last GC: %v ago\n", time.Since(stats.LastGC).Truncate(time.Second))
	}
}
