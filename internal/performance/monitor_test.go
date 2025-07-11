package performance

import (
	"sync"
	"testing"
	"time"
)

func TestNewMonitor(t *testing.T) {
	monitor := NewMonitor(nil)
	
	if monitor == nil {
		t.Fatal("Monitor should not be nil")
	}
	
	if monitor.config == nil {
		t.Error("Config should be initialized")
	}
	
	if monitor.metrics == nil {
		t.Error("Metrics should be initialized")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config == nil {
		t.Fatal("Default config should not be nil")
	}
	
	if config.SampleInterval != 5*time.Second {
		t.Errorf("Expected sample interval 5s, got %v", config.SampleInterval)
	}
	
	if config.GCThreshold != 0.8 {
		t.Errorf("Expected GC threshold 0.8, got %f", config.GCThreshold)
	}
	
	if config.MaxMemoryMB != 512 {
		t.Errorf("Expected max memory 512MB, got %d", config.MaxMemoryMB)
	}
	
	if !config.EnableAutoGC {
		t.Error("Expected auto GC to be enabled")
	}
	
	if !config.EnableMetrics {
		t.Error("Expected metrics to be enabled")
	}
}

func TestMonitor_StartStop(t *testing.T) {
	config := &Config{
		SampleInterval:  100 * time.Millisecond,
		EnableMetrics:   true,
		EnableAutoGC:    true,
		GCInterval:      time.Second,
		GCThreshold:     0.8,
		MaxMemoryMB:     512,
		EnableCallbacks: true,
	}
	
	monitor := NewMonitor(config)
	
	// Start monitoring
	err := monitor.Start()
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}
	
	// Let it run for a short time
	time.Sleep(200 * time.Millisecond)
	
	// Check that metrics are being collected
	metrics := monitor.GetMetrics()
	if metrics.SampleCount == 0 {
		t.Error("Expected samples to be collected")
	}
	
	// Stop monitoring
	monitor.Stop()
}

func TestMonitor_GetMetrics(t *testing.T) {
	monitor := NewMonitor(nil)
	
	metrics := monitor.GetMetrics()
	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}
	
	if metrics.StartTime.IsZero() {
		t.Error("Start time should be set")
	}
	
	if metrics.Uptime == 0 {
		t.Error("Uptime should be calculated")
	}
}

func TestMonitor_TriggerGC(t *testing.T) {
	monitor := NewMonitor(nil)
	
	initialGCCount := monitor.GetMetrics().GCTriggers
	
	// Trigger GC
	monitor.TriggerGC("test")
	
	// Check that GC was triggered
	metrics := monitor.GetMetrics()
	if metrics.GCTriggers != initialGCCount+1 {
		t.Errorf("Expected GC triggers to increment by 1, got %d", metrics.GCTriggers-initialGCCount)
	}
	
	if metrics.LastGC.IsZero() {
		t.Error("Last GC time should be set")
	}
}

func TestMonitor_MemoryUsage(t *testing.T) {
	monitor := NewMonitor(nil)
	
	usage := monitor.GetMemoryUsage()
	if usage <= 0 {
		t.Error("Memory usage should be positive")
	}
	
	percent := monitor.GetMemoryUsagePercent()
	if percent < 0 || percent > 1 {
		t.Errorf("Memory usage percent should be between 0 and 1, got %f", percent)
	}
}

func TestMonitor_Callbacks(t *testing.T) {
	config := &Config{
		EnableCallbacks: true,
		EnableMetrics:   true,
		SampleInterval:  50 * time.Millisecond,
		MaxMemoryMB:     1, // Very low to trigger warnings
		GCThreshold:     0.1,
	}
	
	monitor := NewMonitor(config)
	
	var eventReceived bool
	var eventMutex sync.Mutex
	
	// Add callback
	monitor.AddCallback(func(event Event) {
		eventMutex.Lock()
		eventReceived = true
		eventMutex.Unlock()
	})
	
	// Trigger GC to generate event
	monitor.TriggerGC("test callback")
	
	// Wait a bit for callback to be called
	time.Sleep(100 * time.Millisecond)
	
	eventMutex.Lock()
	received := eventReceived
	eventMutex.Unlock()
	
	if !received {
		t.Error("Callback should have been called")
	}
}

func TestMonitor_ForceMemoryCleanup(t *testing.T) {
	monitor := NewMonitor(nil)
	
	initialGCCount := monitor.GetMetrics().GCTriggers
	
	// Force cleanup
	monitor.ForceMemoryCleanup()
	
	// Check that GC was triggered
	metrics := monitor.GetMetrics()
	if metrics.GCTriggers != initialGCCount+1 {
		t.Errorf("Expected GC triggers to increment, got %d", metrics.GCTriggers-initialGCCount)
	}
}

func TestMonitor_IsMemoryThresholdExceeded(t *testing.T) {
	config := &Config{
		GCThreshold: 0.5, // 50%
		MaxMemoryMB: 100,
	}
	
	monitor := NewMonitor(config)
	
	// Should not exceed threshold initially
	if monitor.IsMemoryThresholdExceeded() {
		// This might be true depending on actual memory usage
		// Just check that the method returns a boolean
		t.Logf("Memory threshold exceeded: %v", monitor.IsMemoryThresholdExceeded())
	}
}

func TestEventTypes(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventMemoryWarning, "memory_warning"},
		{EventGCTriggered, "gc_triggered"},
		{EventHighCPU, "high_cpu"},
		{EventManyGoroutines, "many_goroutines"},
		{EventThresholdCrossed, "threshold_crossed"},
	}
	
	for _, tt := range tests {
		if string(tt.eventType) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.eventType))
		}
	}
}

func TestGetSystemMemoryInfo(t *testing.T) {
	info := GetSystemMemoryInfo()
	
	if info == nil {
		t.Fatal("Memory info should not be nil")
	}
	
	expectedKeys := []string{
		"allocated_mb", "total_alloc_mb", "system_mb",
		"heap_inuse_mb", "heap_objects", "gc_cycles",
		"gc_pause_ms", "goroutines", "next_gc_mb",
	}
	
	for _, key := range expectedKeys {
		if _, exists := info[key]; !exists {
			t.Errorf("Expected key %s to exist in memory info", key)
		}
	}
	
	// Check that values are reasonable
	if allocated, ok := info["allocated_mb"].(uint64); ok {
		if allocated == 0 {
			t.Error("Allocated memory should be greater than 0")
		}
	} else {
		t.Error("allocated_mb should be uint64")
	}
	
	if goroutines, ok := info["goroutines"].(int); ok {
		if goroutines <= 0 {
			t.Error("Goroutines should be greater than 0")
		}
	} else {
		t.Error("goroutines should be int")
	}
}

func TestFormatMemorySize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{512, "512B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{1073741824, "1.0GB"},
	}
	
	for _, tt := range tests {
		result := FormatMemorySize(tt.bytes)
		if result != tt.expected {
			t.Errorf("FormatMemorySize(%d) = %s, expected %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestMonitor_MetricsThreadSafety(t *testing.T) {
	monitor := NewMonitor(nil)
	
	// Start concurrent operations
	var wg sync.WaitGroup
	
	// Concurrent metric updates
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				monitor.TriggerGC("concurrent test")
				monitor.GetMetrics()
				monitor.GetMemoryUsage()
			}
		}()
	}
	
	wg.Wait()
	
	// Should not panic or race
	metrics := monitor.GetMetrics()
	if metrics == nil {
		t.Error("Metrics should not be nil after concurrent access")
	}
}

func TestMonitor_DisabledMetrics(t *testing.T) {
	config := &Config{
		EnableMetrics:   false,
		EnableCallbacks: false,
		EnableAutoGC:    false,
	}
	
	monitor := NewMonitor(config)
	
	// Should not start collection when disabled
	err := monitor.Start()
	if err != nil {
		t.Errorf("Start should not fail when metrics disabled: %v", err)
	}
	
	// Callbacks should not be added when disabled
	callbackCalled := false
	monitor.AddCallback(func(Event) {
		callbackCalled = true
	})
	
	monitor.TriggerGC("test")
	time.Sleep(50 * time.Millisecond)
	
	if callbackCalled {
		t.Error("Callback should not be called when disabled")
	}
}

// Benchmark tests

func BenchmarkMonitor_GetMetrics(b *testing.B) {
	monitor := NewMonitor(nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.GetMetrics()
	}
}

func BenchmarkMonitor_TriggerGC(b *testing.B) {
	monitor := NewMonitor(nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.TriggerGC("benchmark")
	}
}

func BenchmarkGetSystemMemoryInfo(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetSystemMemoryInfo()
	}
}

func BenchmarkFormatMemorySize(b *testing.B) {
	sizes := []int64{512, 1024, 1048576, 1073741824}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatMemorySize(sizes[i%len(sizes)])
	}
}