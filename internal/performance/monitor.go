package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// Monitor provides performance monitoring and memory management
type Monitor struct {
	config    *Config
	metrics   *Metrics
	callbacks []CallbackFunc
	mutex     sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	stopOnce  sync.Once
}

// Config holds configuration for performance monitoring
type Config struct {
	SampleInterval     time.Duration `json:"sample_interval"`      // How often to sample metrics
	GCThreshold        float64       `json:"gc_threshold"`         // Memory usage percentage to trigger GC
	MaxMemoryMB        int64         `json:"max_memory_mb"`        // Maximum memory usage before warnings
	EnableAutoGC       bool          `json:"enable_auto_gc"`       // Enable automatic garbage collection
	GCInterval         time.Duration `json:"gc_interval"`          // Interval for forced GC
	EnableMetrics      bool          `json:"enable_metrics"`       // Enable metrics collection
	EnableCallbacks    bool          `json:"enable_callbacks"`     // Enable performance callbacks
	CPUProfileInterval time.Duration `json:"cpu_profile_interval"` // CPU profiling interval
	MemProfileInterval time.Duration `json:"mem_profile_interval"` // Memory profiling interval
}

// Metrics holds performance metrics
type Metrics struct {
	// Memory metrics
	AllocatedMemory int64     `json:"allocated_memory"` // Currently allocated memory in bytes
	SystemMemory    int64     `json:"system_memory"`    // System memory in bytes
	GCCount         int64     `json:"gc_count"`         // Number of GC cycles
	LastGC          time.Time `json:"last_gc"`          // Last GC time
	GCPauseTotal    int64     `json:"gc_pause_total"`   // Total GC pause time in nanoseconds
	HeapInUse       int64     `json:"heap_in_use"`      // Heap memory in use
	HeapObjects     int64     `json:"heap_objects"`     // Number of objects in heap

	// CPU metrics
	CPUUsage   float64 `json:"cpu_usage"`  // CPU usage percentage
	Goroutines int     `json:"goroutines"` // Number of goroutines

	// Performance metrics
	SampleCount int64         `json:"sample_count"` // Number of samples taken
	LastSample  time.Time     `json:"last_sample"`  // Last sample time
	StartTime   time.Time     `json:"start_time"`   // Monitor start time
	Uptime      time.Duration `json:"uptime"`       // Total uptime

	// Thresholds and alerts
	MemoryWarnings   int64 `json:"memory_warnings"`    // Number of memory warnings
	GCTriggers       int64 `json:"gc_triggers"`        // Number of automatic GC triggers
	MaxMemoryReached int64 `json:"max_memory_reached"` // Maximum memory usage seen

	mutex sync.RWMutex
}

// CallbackFunc is called when performance events occur
type CallbackFunc func(event Event)

// Event represents a performance event
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
	Message   string      `json:"message"`
}

// EventType represents the type of performance event
type EventType string

const (
	EventMemoryWarning    EventType = "memory_warning"
	EventGCTriggered      EventType = "gc_triggered"
	EventHighCPU          EventType = "high_cpu"
	EventManyGoroutines   EventType = "many_goroutines"
	EventThresholdCrossed EventType = "threshold_crossed"
)

// MemoryWarningData contains data for memory warning events
type MemoryWarningData struct {
	CurrentMemoryMB int64   `json:"current_memory_mb"`
	ThresholdMB     int64   `json:"threshold_mb"`
	UsagePercent    float64 `json:"usage_percent"`
}

// GCTriggeredData contains data for GC triggered events
type GCTriggeredData struct {
	Reason        string        `json:"reason"`
	MemoryBefore  int64         `json:"memory_before"`
	MemoryAfter   int64         `json:"memory_after"`
	PauseDuration time.Duration `json:"pause_duration"`
}

// NewMonitor creates a new performance monitor
func NewMonitor(config *Config) *Monitor {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Monitor{
		config: config,
		metrics: &Metrics{
			StartTime: time.Now(),
		},
		callbacks: make([]CallbackFunc, 0),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// DefaultConfig returns default monitoring configuration
func DefaultConfig() *Config {
	return &Config{
		SampleInterval:     5 * time.Second,
		GCThreshold:        0.8, // 80% memory usage
		MaxMemoryMB:        512, // 512MB max memory
		EnableAutoGC:       true,
		GCInterval:         2 * time.Minute,
		EnableMetrics:      true,
		EnableCallbacks:    true,
		CPUProfileInterval: time.Minute,
		MemProfileInterval: time.Minute,
	}
}

// Start begins performance monitoring
func (m *Monitor) Start() error {
	if !m.config.EnableMetrics {
		return nil
	}

	// Start metrics collection
	go m.startMetricsCollection()

	// Start automatic GC if enabled
	if m.config.EnableAutoGC {
		go m.startAutoGC()
	}

	return nil
}

// Stop stops performance monitoring
func (m *Monitor) Stop() {
	m.stopOnce.Do(func() {
		m.cancel()
	})
}

// AddCallback adds a performance event callback
func (m *Monitor) AddCallback(callback CallbackFunc) {
	if !m.config.EnableCallbacks {
		return
	}

	m.mutex.Lock()
	m.callbacks = append(m.callbacks, callback)
	m.mutex.Unlock()
}

// GetMetrics returns current performance metrics
func (m *Monitor) GetMetrics() *Metrics {
	m.metrics.mutex.RLock()
	defer m.metrics.mutex.RUnlock()

	// Create a copy to avoid race conditions
	metrics := *m.metrics
	metrics.Uptime = time.Since(metrics.StartTime)

	return &metrics
}

// TriggerGC manually triggers garbage collection
func (m *Monitor) TriggerGC(reason string) {
	memBefore := m.getCurrentMemoryUsage()
	start := time.Now()

	runtime.GC()

	duration := time.Since(start)
	memAfter := m.getCurrentMemoryUsage()

	// Update metrics
	m.metrics.mutex.Lock()
	m.metrics.GCTriggers++
	m.metrics.LastGC = time.Now()
	m.metrics.mutex.Unlock()

	// Fire callback
	m.fireEvent(Event{
		Type:      EventGCTriggered,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("GC triggered: %s", reason),
		Data: GCTriggeredData{
			Reason:        reason,
			MemoryBefore:  memBefore,
			MemoryAfter:   memAfter,
			PauseDuration: duration,
		},
	})
}

// GetMemoryUsage returns current memory usage in MB
func (m *Monitor) GetMemoryUsage() int64 {
	return m.getCurrentMemoryUsage() / (1024 * 1024)
}

// GetMemoryUsagePercent returns current memory usage as percentage of max
func (m *Monitor) GetMemoryUsagePercent() float64 {
	currentMB := m.GetMemoryUsage()
	return float64(currentMB) / float64(m.config.MaxMemoryMB)
}

// IsMemoryThresholdExceeded checks if memory usage exceeds threshold
func (m *Monitor) IsMemoryThresholdExceeded() bool {
	return m.GetMemoryUsagePercent() > m.config.GCThreshold
}

// ForceMemoryCleanup forces memory cleanup
func (m *Monitor) ForceMemoryCleanup() {
	// Force GC multiple times to ensure cleanup
	runtime.GC()
	runtime.GC()

	// Note: runtime.FreeOSMemory was removed in Go 1.16

	// Update metrics
	m.metrics.mutex.Lock()
	m.metrics.GCTriggers++
	m.metrics.LastGC = time.Now()
	m.metrics.mutex.Unlock()

	m.fireEvent(Event{
		Type:      EventGCTriggered,
		Timestamp: time.Now(),
		Message:   "Forced memory cleanup",
		Data: GCTriggeredData{
			Reason: "force_cleanup",
		},
	})
}

// LogMemoryStats logs current memory statistics
func (m *Monitor) LogMemoryStats() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	fmt.Printf("Memory Stats:\n")
	fmt.Printf("  Allocated: %d KB\n", ms.Alloc/1024)
	fmt.Printf("  Total Allocated: %d KB\n", ms.TotalAlloc/1024)
	fmt.Printf("  System: %d KB\n", ms.Sys/1024)
	fmt.Printf("  Heap In Use: %d KB\n", ms.HeapInuse/1024)
	fmt.Printf("  Heap Objects: %d\n", ms.HeapObjects)
	fmt.Printf("  GC Cycles: %d\n", ms.NumGC)
	fmt.Printf("  GC Pause Total: %d ms\n", ms.PauseTotalNs/(1000*1000))
	fmt.Printf("  Next GC: %d KB\n", ms.NextGC/1024)
	fmt.Printf("  Goroutines: %d\n", runtime.NumGoroutine())
}

// Private methods

func (m *Monitor) startMetricsCollection() {
	ticker := time.NewTicker(m.config.SampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.collectMetrics()
		}
	}
}

func (m *Monitor) startAutoGC() {
	ticker := time.NewTicker(m.config.GCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkAndTriggerGC()
		}
	}
}

func (m *Monitor) collectMetrics() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	currentMB := int64(ms.Alloc) / (1024 * 1024)

	m.metrics.mutex.Lock()

	// Update memory metrics
	m.metrics.AllocatedMemory = int64(ms.Alloc)
	m.metrics.SystemMemory = int64(ms.Sys)
	m.metrics.GCCount = int64(ms.NumGC)
	m.metrics.GCPauseTotal = int64(ms.PauseTotalNs)
	m.metrics.HeapInUse = int64(ms.HeapInuse)
	m.metrics.HeapObjects = int64(ms.HeapObjects)

	// Update CPU and runtime metrics
	m.metrics.Goroutines = runtime.NumGoroutine()
	m.metrics.SampleCount++
	m.metrics.LastSample = time.Now()

	// Track maximum memory usage
	if currentMB > m.metrics.MaxMemoryReached {
		m.metrics.MaxMemoryReached = currentMB
	}

	m.metrics.mutex.Unlock()

	// Check for warnings
	m.checkMemoryWarnings(currentMB)
	m.checkGoroutineWarnings()
}

func (m *Monitor) checkAndTriggerGC() {
	if !m.config.EnableAutoGC {
		return
	}

	usagePercent := m.GetMemoryUsagePercent()

	if usagePercent > m.config.GCThreshold {
		m.TriggerGC(fmt.Sprintf("automatic (%.1f%% usage)", usagePercent*100))
	}
}

func (m *Monitor) checkMemoryWarnings(currentMB int64) {
	usagePercent := float64(currentMB) / float64(m.config.MaxMemoryMB)

	if usagePercent > 0.9 { // 90% warning threshold
		m.metrics.mutex.Lock()
		m.metrics.MemoryWarnings++
		m.metrics.mutex.Unlock()

		m.fireEvent(Event{
			Type:      EventMemoryWarning,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("High memory usage: %dMB (%.1f%%)", currentMB, usagePercent*100),
			Data: MemoryWarningData{
				CurrentMemoryMB: currentMB,
				ThresholdMB:     m.config.MaxMemoryMB,
				UsagePercent:    usagePercent,
			},
		})
	}
}

func (m *Monitor) checkGoroutineWarnings() {
	goroutines := runtime.NumGoroutine()

	if goroutines > 1000 { // Warn if too many goroutines
		m.fireEvent(Event{
			Type:      EventManyGoroutines,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("High goroutine count: %d", goroutines),
			Data:      goroutines,
		})
	}
}

func (m *Monitor) getCurrentMemoryUsage() int64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return int64(ms.Alloc)
}

func (m *Monitor) fireEvent(event Event) {
	if !m.config.EnableCallbacks {
		return
	}

	m.mutex.RLock()
	callbacks := make([]CallbackFunc, len(m.callbacks))
	copy(callbacks, m.callbacks)
	m.mutex.RUnlock()

	for _, callback := range callbacks {
		go func(cb CallbackFunc) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Performance callback panic: %v\n", r)
				}
			}()
			cb(event)
		}(callback)
	}
}

// Utility functions for external use

// GetSystemMemoryInfo returns system memory information
func GetSystemMemoryInfo() map[string]interface{} {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return map[string]interface{}{
		"allocated_mb":   ms.Alloc / (1024 * 1024),
		"total_alloc_mb": ms.TotalAlloc / (1024 * 1024),
		"system_mb":      ms.Sys / (1024 * 1024),
		"heap_inuse_mb":  ms.HeapInuse / (1024 * 1024),
		"heap_objects":   ms.HeapObjects,
		"gc_cycles":      ms.NumGC,
		"gc_pause_ms":    ms.PauseTotalNs / (1000 * 1000),
		"goroutines":     runtime.NumGoroutine(),
		"next_gc_mb":     ms.NextGC / (1024 * 1024),
	}
}

// FormatMemorySize formats memory size in human readable format
func FormatMemorySize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// PrintMemoryProfile prints a memory profile summary
func PrintMemoryProfile() {
	info := GetSystemMemoryInfo()

	fmt.Printf("=== Memory Profile ===\n")
	fmt.Printf("Allocated: %dMB\n", info["allocated_mb"])
	fmt.Printf("System: %dMB\n", info["system_mb"])
	fmt.Printf("Heap In Use: %dMB\n", info["heap_inuse_mb"])
	fmt.Printf("Heap Objects: %d\n", info["heap_objects"])
	fmt.Printf("GC Cycles: %d\n", info["gc_cycles"])
	fmt.Printf("GC Pause Total: %dms\n", info["gc_pause_ms"])
	fmt.Printf("Goroutines: %d\n", info["goroutines"])
	fmt.Printf("Next GC: %dMB\n", info["next_gc_mb"])
	fmt.Printf("=====================\n")
}
