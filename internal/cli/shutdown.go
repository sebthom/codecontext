package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ShutdownManager handles graceful shutdown for CLI applications
type ShutdownManager struct {
	handlers     []ShutdownHandler
	timeout      time.Duration
	mutex        sync.RWMutex
	shutdownChan chan os.Signal
	ctx          context.Context
	cancel       context.CancelFunc
	shutdownOnce sync.Once
	completed    chan struct{}
}

// ShutdownHandler represents a function that should be called during shutdown
type ShutdownHandler struct {
	Name     string
	Priority int // Lower numbers run first
	Handler  func() error
	Timeout  time.Duration
}

// ShutdownConfig holds configuration for shutdown management
type ShutdownConfig struct {
	GracefulTimeout time.Duration `json:"graceful_timeout"` // Total time to wait for graceful shutdown
	HandlerTimeout  time.Duration `json:"handler_timeout"`  // Timeout for individual handlers
	Signals         []os.Signal   `json:"signals"`          // Signals to listen for
	EnableLogging   bool          `json:"enable_logging"`   // Enable shutdown logging
}

// NewShutdownManager creates a new shutdown manager
func NewShutdownManager(config *ShutdownConfig) *ShutdownManager {
	if config == nil {
		config = &ShutdownConfig{
			GracefulTimeout: 30 * time.Second,
			HandlerTimeout:  10 * time.Second,
			Signals:         []os.Signal{syscall.SIGINT, syscall.SIGTERM},
			EnableLogging:   true,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	sm := &ShutdownManager{
		handlers:     make([]ShutdownHandler, 0),
		timeout:      config.GracefulTimeout,
		shutdownChan: make(chan os.Signal, 1),
		ctx:          ctx,
		cancel:       cancel,
		completed:    make(chan struct{}),
	}

	// Register signal handlers
	signal.Notify(sm.shutdownChan, config.Signals...)

	// Start shutdown listener
	go sm.listenForShutdown(config.EnableLogging)

	return sm
}

// RegisterHandler registers a shutdown handler
func (sm *ShutdownManager) RegisterHandler(name string, priority int, timeout time.Duration, handler func() error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.handlers = append(sm.handlers, ShutdownHandler{
		Name:     name,
		Priority: priority,
		Handler:  handler,
		Timeout:  timeout,
	})

	// Sort handlers by priority
	sm.sortHandlers()
}

// RegisterSimpleHandler registers a simple shutdown handler with default priority and timeout
func (sm *ShutdownManager) RegisterSimpleHandler(name string, handler func() error) {
	sm.RegisterHandler(name, 50, 5*time.Second, handler)
}

// TriggerShutdown manually triggers the shutdown process
func (sm *ShutdownManager) TriggerShutdown() {
	select {
	case sm.shutdownChan <- syscall.SIGTERM:
	default:
		// Channel full or shutdown already in progress
	}
}

// Wait waits for the shutdown process to complete
func (sm *ShutdownManager) Wait() {
	<-sm.completed
}

// GetContext returns the shutdown context
func (sm *ShutdownManager) GetContext() context.Context {
	return sm.ctx
}

// IsShuttingDown returns true if shutdown is in progress
func (sm *ShutdownManager) IsShuttingDown() bool {
	select {
	case <-sm.ctx.Done():
		return true
	default:
		return false
	}
}

// Private methods

func (sm *ShutdownManager) listenForShutdown(enableLogging bool) {
	defer close(sm.completed)

	sig := <-sm.shutdownChan

	if enableLogging {
		fmt.Printf("\n🛑 Received signal %v, initiating graceful shutdown...\n", sig)
	}

	sm.shutdownOnce.Do(func() {
		sm.performShutdown(enableLogging)
	})
}

func (sm *ShutdownManager) performShutdown(enableLogging bool) {
	// Cancel the context to signal all operations to stop
	sm.cancel()

	// Create timeout context for the entire shutdown process
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), sm.timeout)
	defer shutdownCancel()

	if enableLogging {
		fmt.Printf("⏳ Running %d shutdown handlers (timeout: %v)...\n", len(sm.handlers), sm.timeout)
	}

	// Execute shutdown handlers
	sm.mutex.RLock()
	handlers := make([]ShutdownHandler, len(sm.handlers))
	copy(handlers, sm.handlers)
	sm.mutex.RUnlock()

	for i, handler := range handlers {
		select {
		case <-shutdownCtx.Done():
			if enableLogging {
				fmt.Printf("⚠️  Shutdown timeout reached, %d handlers remaining\n", len(handlers)-i)
			}
			return
		default:
			sm.executeHandler(handler, enableLogging)
		}
	}

	if enableLogging {
		fmt.Println("✅ Graceful shutdown completed")
	}
}

func (sm *ShutdownManager) executeHandler(handler ShutdownHandler, enableLogging bool) {
	if enableLogging {
		fmt.Printf("🔄 Executing shutdown handler: %s\n", handler.Name)
	}

	// Create timeout context for this handler
	ctx, cancel := context.WithTimeout(context.Background(), handler.Timeout)
	defer cancel()

	// Execute handler in goroutine to respect timeout
	done := make(chan error, 1)
	go func() {
		done <- handler.Handler()
	}()

	select {
	case err := <-done:
		if err != nil {
			if enableLogging {
				fmt.Printf("⚠️  Handler %s failed: %v\n", handler.Name, err)
			}
		} else {
			if enableLogging {
				fmt.Printf("✅ Handler %s completed\n", handler.Name)
			}
		}
	case <-ctx.Done():
		if enableLogging {
			fmt.Printf("⚠️  Handler %s timed out after %v\n", handler.Name, handler.Timeout)
		}
	}
}

func (sm *ShutdownManager) sortHandlers() {
	// Sort by priority (lower numbers first)
	for i := 0; i < len(sm.handlers)-1; i++ {
		for j := 0; j < len(sm.handlers)-i-1; j++ {
			if sm.handlers[j].Priority > sm.handlers[j+1].Priority {
				sm.handlers[j], sm.handlers[j+1] = sm.handlers[j+1], sm.handlers[j]
			}
		}
	}
}

// WatchModeShutdown provides specific shutdown handling for watch mode
type WatchModeShutdown struct {
	manager      *ShutdownManager
	watchManager *WatchManager
}

// NewWatchModeShutdown creates a shutdown handler specifically for watch mode
func NewWatchModeShutdown(watchManager *WatchManager) *WatchModeShutdown {
	config := &ShutdownConfig{
		GracefulTimeout: 15 * time.Second, // Shorter timeout for CLI tools
		HandlerTimeout:  5 * time.Second,
		Signals:         []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP},
		EnableLogging:   true,
	}

	ws := &WatchModeShutdown{
		manager:      NewShutdownManager(config),
		watchManager: watchManager,
	}

	// Register watch mode specific handlers
	ws.registerHandlers()

	return ws
}

// Wait waits for shutdown to complete
func (ws *WatchModeShutdown) Wait() {
	ws.manager.Wait()
}

// GetContext returns the shutdown context
func (ws *WatchModeShutdown) GetContext() context.Context {
	return ws.manager.GetContext()
}

// IsShuttingDown returns true if shutdown is in progress
func (ws *WatchModeShutdown) IsShuttingDown() bool {
	return ws.manager.IsShuttingDown()
}

// RegisterCustomHandler allows registering additional shutdown handlers
func (ws *WatchModeShutdown) RegisterCustomHandler(name string, handler func() error) {
	ws.manager.RegisterSimpleHandler(name, handler)
}

// Private methods for WatchModeShutdown

func (ws *WatchModeShutdown) registerHandlers() {
	// Stop file watcher (highest priority)
	ws.manager.RegisterHandler("file_watcher", 10, 3*time.Second, func() error {
		if ws.watchManager.watcher != nil {
			ws.watchManager.watcher.StopWatching()
		}
		return nil
	})

	// Process pending changes
	ws.manager.RegisterHandler("pending_changes", 20, 5*time.Second, func() error {
		if ws.watchManager.analyzer != nil {
			// Process any pending changes
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			// This would process any pending changes in the analyzer
			// For now, we'll just ensure the analyzer is in a clean state
			_ = ctx // Use the context in real implementation
		}
		return nil
	})

	// Generate final output
	ws.manager.RegisterHandler("final_output", 30, 3*time.Second, func() error {
		return ws.watchManager.generateOutput()
	})

	// Save cache state
	ws.manager.RegisterHandler("cache_save", 40, 2*time.Second, func() error {
		if ws.watchManager.cache != nil {
			return ws.watchManager.cache.Close()
		}
		return nil
	})

	// Print final statistics
	ws.manager.RegisterHandler("final_stats", 90, 1*time.Second, func() error {
		ws.watchManager.PrintStats()
		return nil
	})

	// Cleanup resources (lowest priority)
	ws.manager.RegisterHandler("cleanup", 100, 1*time.Second, func() error {
		ws.watchManager.Cleanup()
		return nil
	})
}

// EmergencyShutdown performs an emergency shutdown without waiting for handlers
func EmergencyShutdown(reason string) {
	fmt.Printf("\n💥 EMERGENCY SHUTDOWN: %s\n", reason)
	fmt.Println("⚠️  Forcing immediate exit without cleanup")
	os.Exit(1)
}

// SetupPanicHandler sets up a global panic handler for emergency shutdown
func SetupPanicHandler() {
	originalHandler := recover()
	if originalHandler != nil {
		fmt.Printf("\n💥 PANIC DETECTED: %v\n", originalHandler)
		EmergencyShutdown("Unrecoverable panic occurred")
	}
}

// ShutdownAware is an interface for components that need shutdown notifications
type ShutdownAware interface {
	Shutdown(ctx context.Context) error
}

// RegisterShutdownAware registers a shutdown-aware component
func (sm *ShutdownManager) RegisterShutdownAware(name string, priority int, component ShutdownAware) {
	sm.RegisterHandler(name, priority, 10*time.Second, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		return component.Shutdown(ctx)
	})
}

// HealthChecker can be used to verify system state during shutdown
type HealthChecker struct {
	checks map[string]func() error
	mutex  sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]func() error),
	}
}

// AddCheck adds a health check
func (hc *HealthChecker) AddCheck(name string, check func() error) {
	hc.mutex.Lock()
	hc.checks[name] = check
	hc.mutex.Unlock()
}

// RunChecks runs all health checks and returns any errors
func (hc *HealthChecker) RunChecks() map[string]error {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	results := make(map[string]error)
	for name, check := range hc.checks {
		results[name] = check()
	}

	return results
}

// Example usage functions

// SetupGracefulShutdown sets up graceful shutdown for a typical CLI application
func SetupGracefulShutdown() *ShutdownManager {
	config := &ShutdownConfig{
		GracefulTimeout: 30 * time.Second,
		HandlerTimeout:  10 * time.Second,
		Signals:         []os.Signal{syscall.SIGINT, syscall.SIGTERM},
		EnableLogging:   true,
	}

	sm := NewShutdownManager(config)

	// Add common handlers
	sm.RegisterHandler("save_state", 10, 5*time.Second, func() error {
		// Save application state
		return nil
	})

	sm.RegisterHandler("close_connections", 20, 3*time.Second, func() error {
		// Close database connections, etc.
		return nil
	})

	sm.RegisterHandler("cleanup_temp", 90, 2*time.Second, func() error {
		// Clean up temporary files
		return nil
	})

	return sm
}
