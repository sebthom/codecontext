package types

import (
	"time"
)

// CompactController represents the Compact Controller interface
type CompactController interface {
	// Command execution
	ExecuteCommand(command string, context *CompactContext) (*CompactResult, error)
	
	// Predefined strategies
	CompactMinimal() (*CompactResult, error)
	CompactBalanced() (*CompactResult, error)
	CompactAggressive() (*CompactResult, error)
	
	// Task-specific compaction
	CompactForTask(task TaskType) (*CompactResult, error)
	CompactToTokenLimit(maxTokens int) (*CompactResult, error)
	
	// Interactive features
	PreviewCompaction(strategy CompactStrategy) (*CompactPreview, error)
	UndoCompaction() error
	GetCompactionHistory() ([]*CompactHistory, error)
}

// CompactCommand represents a compaction command
type CompactCommand struct {
	Type       string                 `json:"type"` // "level", "task", "tokens", "custom"
	Parameters CompactParameters      `json:"parameters"`
	Preview    bool                   `json:"preview"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// CompactParameters represents parameters for compaction
type CompactParameters struct {
	Level      string   `json:"level,omitempty"`      // "minimal", "balanced", "aggressive"
	Task       string   `json:"task,omitempty"`       // "debugging", "refactoring", etc.
	MaxTokens  int      `json:"max_tokens,omitempty"`
	FocusFiles []string `json:"focus_files,omitempty"`
	Preserve   []string `json:"preserve,omitempty"`
	Remove     []string `json:"remove,omitempty"`
}

// CompactResult represents the result of a compaction
type CompactResult struct {
	Id                string              `json:"id"`
	OriginalTokens    int                 `json:"original_tokens"`
	CompactedTokens   int                 `json:"compacted_tokens"`
	ReductionPercent  float64             `json:"reduction_percent"`
	PreservedSymbols  []*Symbol           `json:"preserved_symbols"`
	RemovedSymbols    []*Symbol           `json:"removed_symbols"`
	QualityScore      float64             `json:"quality_score"`
	Reversible        bool                `json:"reversible"`
	ExecutionTime     time.Duration       `json:"execution_time"`
	Strategy          CompactStrategy     `json:"strategy"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// CompactPreview represents a preview of compaction
type CompactPreview struct {
	EstimatedTokens   int               `json:"estimated_tokens"`
	EstimatedReduction float64          `json:"estimated_reduction"`
	EstimatedQuality  float64           `json:"estimated_quality"`
	WillPreserve      []string          `json:"will_preserve"`
	WillRemove        []string          `json:"will_remove"`
	Warnings          []string          `json:"warnings,omitempty"`
	Strategy          CompactStrategy   `json:"strategy"`
}

// CompactHistory represents a compaction operation in history
type CompactHistory struct {
	Id        string           `json:"id"`
	Command   CompactCommand   `json:"command"`
	Result    CompactResult    `json:"result"`
	Timestamp time.Time        `json:"timestamp"`
	Reversed  bool             `json:"reversed"`
}

// CompactContext represents the context for compaction
type CompactContext struct {
	Graph       *CodeGraph             `json:"graph"`
	Config      map[string]interface{} `json:"config"`
	FocusFiles  []string               `json:"focus_files,omitempty"`
	Constraints []string               `json:"constraints,omitempty"`
}

// TaskType represents different task types for compaction
type TaskType struct {
	Name              string          `json:"name"`
	PriorityPatterns  []string        `json:"priority_patterns"`
	PreserveList      []SymbolPattern `json:"preserve_list"`
	AggressiveRemoval []SymbolPattern `json:"aggressive_removal"`
	Description       string          `json:"description"`
}

// Standard task types
var (
	TaskTypeDebugging = TaskType{
		Name:             "debugging",
		PriorityPatterns: []string{"error", "exception", "log", "debug", "trace"},
		PreserveList: []SymbolPattern{
			{Pattern: ".*Error.*", Type: "class"},
			{Pattern: ".*Exception.*", Type: "class"},
			{Pattern: ".*log.*", Type: "function"},
			{Pattern: ".*debug.*", Type: "function"},
		},
		AggressiveRemoval: []SymbolPattern{
			{Pattern: ".*test.*", Type: "function"},
			{Pattern: ".*Test.*", Type: "class"},
		},
		Description: "Optimize for debugging tasks",
	}
	
	TaskTypeRefactoring = TaskType{
		Name:             "refactoring",
		PriorityPatterns: []string{"class", "interface", "function", "method"},
		PreserveList: []SymbolPattern{
			{Pattern: ".*", Type: "class"},
			{Pattern: ".*", Type: "interface"},
			{Pattern: ".*", Type: "function"},
		},
		AggressiveRemoval: []SymbolPattern{
			{Pattern: ".*test.*", Type: "function"},
			{Pattern: ".*comment.*", Type: "comment"},
		},
		Description: "Optimize for refactoring tasks",
	}
	
	TaskTypeDocumentation = TaskType{
		Name:             "documentation",
		PriorityPatterns: []string{"interface", "type", "class", "export"},
		PreserveList: []SymbolPattern{
			{Pattern: ".*", Type: "interface"},
			{Pattern: ".*", Type: "type"},
			{Pattern: ".*", Type: "class"},
			{Pattern: ".*", Type: "comment"},
		},
		AggressiveRemoval: []SymbolPattern{
			{Pattern: ".*private.*", Type: "method"},
			{Pattern: ".*internal.*", Type: "function"},
		},
		Description: "Optimize for documentation tasks",
	}
)

// SymbolPattern represents a pattern for matching symbols
type SymbolPattern struct {
	Pattern string `json:"pattern"`
	Type    string `json:"type"`
	Regex   bool   `json:"regex"`
}

// CompactStrategy represents different compaction strategies
type CompactStrategy struct {
	Name           string                 `json:"name"`
	TokenTarget    float64                `json:"token_target"`
	PreserveRules  []CompactRule          `json:"preserve_rules"`
	RemoveRules    []CompactRule          `json:"remove_rules"`
	Priority       int                    `json:"priority"`
	Description    string                 `json:"description"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// CompactRule represents a rule for compaction
type CompactRule struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // "symbol", "file", "pattern"
	Patterns    []string `json:"patterns"`
	Action      string   `json:"action"` // "keep", "remove", "compress"
	Priority    int      `json:"priority"`
	Conditions  []string `json:"conditions,omitempty"`
	Description string   `json:"description,omitempty"`
}

// Standard compaction strategies
var (
	CompactStrategyMinimal = CompactStrategy{
		Name:        "minimal",
		TokenTarget: 0.3,
		PreserveRules: []CompactRule{
			{Name: "core", Type: "pattern", Patterns: []string{"core", "api", "critical"}, Action: "keep", Priority: 1},
		},
		RemoveRules: []CompactRule{
			{Name: "tests", Type: "pattern", Patterns: []string{"test", "spec", "mock"}, Action: "remove", Priority: 1},
			{Name: "examples", Type: "pattern", Patterns: []string{"example", "demo", "sample"}, Action: "remove", Priority: 2},
			{Name: "generated", Type: "pattern", Patterns: []string{"generated", "auto", "build"}, Action: "remove", Priority: 3},
		},
		Description: "Minimal compaction - keep only essential code",
	}
	
	CompactStrategyBalanced = CompactStrategy{
		Name:        "balanced",
		TokenTarget: 0.6,
		PreserveRules: []CompactRule{
			{Name: "core", Type: "pattern", Patterns: []string{"core", "api", "types", "interfaces"}, Action: "keep", Priority: 1},
			{Name: "important", Type: "pattern", Patterns: []string{"main", "index", "app"}, Action: "keep", Priority: 2},
		},
		RemoveRules: []CompactRule{
			{Name: "tests", Type: "pattern", Patterns: []string{"test", "spec"}, Action: "remove", Priority: 1},
			{Name: "examples", Type: "pattern", Patterns: []string{"example", "demo"}, Action: "remove", Priority: 2},
		},
		Description: "Balanced compaction - good compromise between size and completeness",
	}
	
	CompactStrategyAggressive = CompactStrategy{
		Name:        "aggressive",
		TokenTarget: 0.15,
		PreserveRules: []CompactRule{
			{Name: "core", Type: "pattern", Patterns: []string{"core", "api"}, Action: "keep", Priority: 1},
		},
		RemoveRules: []CompactRule{
			{Name: "tests", Type: "pattern", Patterns: []string{"test", "spec", "mock"}, Action: "remove", Priority: 1},
			{Name: "examples", Type: "pattern", Patterns: []string{"example", "demo", "sample"}, Action: "remove", Priority: 2},
			{Name: "generated", Type: "pattern", Patterns: []string{"generated", "auto", "build"}, Action: "remove", Priority: 3},
			{Name: "comments", Type: "pattern", Patterns: []string{"comment", "doc"}, Action: "remove", Priority: 4},
		},
		Description: "Aggressive compaction - keep only absolute essentials",
	}
)