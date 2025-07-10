package types

import (
	"time"
)

// VirtualGraphEngine represents the Virtual Graph Engine interface
type VirtualGraphEngine interface {
	// State management
	GetShadowGraph() *CodeGraph
	GetActualGraph() *CodeGraph
	GetPendingChanges() []ChangeSet
	
	// Core operations
	Diff(oldAST AST, newAST AST) *ASTDiff
	BatchChange(change Change) error
	Reconcile() (*ReconciliationPlan, error)
	Commit(plan *ReconciliationPlan) (*CodeGraph, error)
	Rollback(checkpoint *GraphCheckpoint) error
	
	// Optimization
	ShouldBatch(change Change) bool
	OptimizePlan(plan *ReconciliationPlan) (*OptimizedPlan, error)
	
	// Metrics
	GetChangeMetrics() *ChangeMetrics
}

// AST represents an Abstract Syntax Tree
type AST struct {
	Root         *ASTNode  `json:"root"`
	Language     string    `json:"language"`
	FilePath     string    `json:"file_path"`
	Content      string    `json:"content"`
	Hash         string    `json:"hash"`
	Version      string    `json:"version"`
	ParsedAt     time.Time `json:"parsed_at"`
	TreeSitterTree interface{} `json:"-"` // Internal tree-sitter tree
}

// ASTNode represents a node in the AST
type ASTNode struct {
	Id       string                 `json:"id"`
	Type     string                 `json:"type"`
	Value    string                 `json:"value,omitempty"`
	Children []*ASTNode             `json:"children,omitempty"`
	Location FileLocation           `json:"location"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ASTDiff represents differences between two ASTs
type ASTDiff struct {
	FileId            string             `json:"file_id"`
	FromVersion       string             `json:"from_version"`
	ToVersion         string             `json:"to_version"`
	Additions         []*ASTNode         `json:"additions"`
	Deletions         []*ASTNode         `json:"deletions"`
	Modifications     []*ASTModification `json:"modifications"`
	StructuralChanges bool               `json:"structural_changes"`
	ImpactRadius      *ImpactAnalysis    `json:"impact_radius"`
	ComputedAt        time.Time          `json:"computed_at"`
}

// ASTModification represents a modification to an AST node
type ASTModification struct {
	NodeId      string                 `json:"node_id"`
	Type        string                 `json:"type"` // "content", "structure", "position"
	OldValue    interface{}            `json:"old_value"`
	NewValue    interface{}            `json:"new_value"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ImpactAnalysis represents the impact of changes
type ImpactAnalysis struct {
	AffectedNodes     []NodeId          `json:"affected_nodes"`
	AffectedFiles     []string          `json:"affected_files"`
	PropagationDepth  int               `json:"propagation_depth"`
	Severity          string            `json:"severity"` // "low", "medium", "high"
	EstimatedTokens   int               `json:"estimated_tokens"`
	Dependencies      []NodeId          `json:"dependencies"`
	Dependents        []NodeId          `json:"dependents"`
}

// ChangeSet represents a set of changes
type ChangeSet struct {
	Id        string      `json:"id"`
	Changes   []Change    `json:"changes"`
	Timestamp time.Time   `json:"timestamp"`
	Source    string      `json:"source"` // "file_change", "user_edit", "refactor"
	BatchSize int         `json:"batch_size"`
}

// Change represents a single change
type Change struct {
	Type      string                 `json:"type"` // "add", "remove", "modify"
	Target    interface{}            `json:"target"` // NodeId, FileLocation, etc.
	Data      interface{}            `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ReconciliationPlan represents a plan for reconciling changes
type ReconciliationPlan struct {
	Id               string              `json:"id"`
	Patches          []GraphPatch        `json:"patches"`
	UpdateOrder      []NodeId            `json:"update_order"`
	Invalidations    []CacheInvalidation `json:"invalidations"`
	EstimatedDuration time.Duration      `json:"estimated_duration"`
	TokenImpact      TokenDelta          `json:"token_impact"`
	CreatedAt        time.Time           `json:"created_at"`
}

// OptimizedPlan represents an optimized reconciliation plan
type OptimizedPlan struct {
	OriginalPlan      *ReconciliationPlan `json:"original_plan"`
	OptimizedPatches  []GraphPatch        `json:"optimized_patches"`
	Optimizations     []string            `json:"optimizations"`
	EstimatedSpeedup  float64             `json:"estimated_speedup"`
	MemoryReduction   int                 `json:"memory_reduction"`
}

// CacheInvalidation represents a cache invalidation
type CacheInvalidation struct {
	Type    string   `json:"type"` // "ast", "diff", "symbol"
	Keys    []string `json:"keys"`
	Cascade bool     `json:"cascade"`
}

// TokenDelta represents a change in token count
type TokenDelta struct {
	Before int `json:"before"`
	After  int `json:"after"`
	Delta  int `json:"delta"`
}

// GraphCheckpoint represents a checkpoint in the graph state
type GraphCheckpoint struct {
	Id        string        `json:"id"`
	Graph     *CodeGraph    `json:"graph"`
	Timestamp time.Time     `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ChangeMetrics represents metrics about changes
type ChangeMetrics struct {
	TotalChanges      int           `json:"total_changes"`
	BatchedChanges    int           `json:"batched_changes"`
	AverageReconTime  time.Duration `json:"average_recon_time"`
	CacheHitRate      float64       `json:"cache_hit_rate"`
	MemoryUsage       int           `json:"memory_usage"`
	DiffComputeTime   time.Duration `json:"diff_compute_time"`
	LastReconciliation time.Time    `json:"last_reconciliation"`
}

// VersionedAST represents a versioned AST
type VersionedAST struct {
	AST       *AST      `json:"ast"`
	Version   string    `json:"version"`
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
}

// ASTCache represents a cache for ASTs
type ASTCache interface {
	Get(fileId string, version ...string) (*VersionedAST, error)
	Set(fileId string, ast *VersionedAST) error
	GetDiffCache(fileId string) ([]*ASTDiff, error)
	Invalidate(fileId string) error
	Clear() error
	Size() int
}