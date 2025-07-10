# API Interfaces Specification

**Version:** 2.0  
**Status:** Interface Definitions Complete  
**Last Updated:** January 2025

## Overview

This document defines all the core interfaces and APIs used throughout the CodeContext system. These interfaces serve as the contracts between different components and ensure modularity and testability.

## Table of Contents

1. [Core Interfaces](#core-interfaces)
2. [Virtual Graph Interfaces](#virtual-graph-interfaces)
3. [Parser Interfaces](#parser-interfaces)
4. [Compact Controller Interfaces](#compact-controller-interfaces)
5. [Storage Interfaces](#storage-interfaces)
6. [REST API Specifications](#rest-api-specifications)
7. [GraphQL Schema](#graphql-schema)
8. [CLI Interface](#cli-interface)

## Core Interfaces

### Orchestrator Interface
```go
type Orchestrator interface {
    // Main operations
    GenerateMap(ctx context.Context, config ProjectConfig) (*MapResult, error)
    UpdateMap(ctx context.Context, changes FileChanges) (*MapResult, error)
    UpdateMapIncremental(ctx context.Context, changes FileChanges) (*PatchResult, error)
    
    // Configuration
    ValidateConfig(config ProjectConfig) ValidationResult
    GetSupportedLanguages() []Language
    
    // Operation control
    CancelOperation(operationId string) error
    ResumeFromCheckpoint(checkpointId string) (*MapResult, error)
    
    // Compact operations
    ExecuteCompactCommand(ctx context.Context, command CompactCommand) (*CompactResult, error)
}

type OrchestrationContext struct {
    Context        context.Context
    Cancel         context.CancelFunc
    Checkpoint     *Checkpoint
    ProgressChan   chan Progress
    VirtualGraph   VirtualGraphEngine
    Metrics        *MetricsCollector
}

type MapResult struct {
    Graph          *CodeGraph
    ProcessingTime time.Duration
    TokenCount     int
    FilesProcessed int
    Errors         []ProcessingError
    Checkpoints    []Checkpoint
    Metadata       map[string]interface{}
}

type PatchResult struct {
    Patches        []GraphPatch
    AffectedNodes  []NodeId
    TokenDelta     int
    ProcessingTime time.Duration
    Applied        bool
    Conflicts      []Conflict
}
```

### Progress Tracking
```go
type Progress struct {
    OperationId    string
    Stage          string
    Completed      int
    Total          int
    Percentage     float64
    CurrentFile    string
    EstimatedTime  time.Duration
    Message        string
    Timestamp      time.Time
}

type ProgressReporter interface {
    ReportProgress(progress Progress)
    SetTotal(total int)
    Increment(delta int)
    UpdateMessage(message string)
    Finish()
}
```

## Virtual Graph Interfaces

### Core Virtual Graph Engine
```go
type VirtualGraphEngine interface {
    // State management
    GetShadowGraph() *CodeGraph
    GetActualGraph() *CodeGraph
    GetPendingChanges() []ChangeSet
    
    // Core operations
    Diff(oldAST AST, newAST AST) (*ASTDiff, error)
    BatchChange(change Change) error
    Reconcile() (*ReconciliationPlan, error)
    Commit(plan *ReconciliationPlan) (*CodeGraph, error)
    Rollback(checkpoint *GraphCheckpoint) error
    
    // Optimization
    ShouldBatch(change Change) bool
    OptimizePlan(plan *ReconciliationPlan) (*OptimizedPlan, error)
    
    // Metrics and monitoring
    GetChangeMetrics() *ChangeMetrics
    ValidateIntegrity() []ValidationError
    GetMemoryUsage() MemoryStats
}

type ASTDiffer interface {
    // Core diff operations
    StructuralDiff(oldAST AST, newAST AST) (*StructuralDiff, error)
    SemanticDiff(oldAST AST, newAST AST) (*SemanticDiff, error)
    
    // Symbol tracking
    TrackSymbolChanges(diff *StructuralDiff) (*SymbolChangeSet, error)
    ComputeImpact(changes *SymbolChangeSet) (*ImpactGraph, error)
    
    // Configuration
    SetDiffAlgorithm(algorithm string) error
    SetMaxDepth(depth int)
    EnableMemoization(enabled bool)
}

type Reconciler interface {
    // Plan generation
    GeneratePlan(changes []ChangeSet) (*ReconciliationPlan, error)
    OptimizePlan(plan *ReconciliationPlan) (*OptimizedPlan, error)
    ValidatePlan(plan *ReconciliationPlan) []ValidationError
    
    // Plan execution
    ExecutePlan(plan *ReconciliationPlan) (*ExecutionResult, error)
    PreviewPlan(plan *ReconciliationPlan) (*PlanPreview, error)
    
    // Rollback support
    CreateCheckpoint() (*GraphCheckpoint, error)
    RestoreCheckpoint(checkpoint *GraphCheckpoint) error
}
```

### Change Management
```go
type Change struct {
    Id        string
    Type      ChangeType    // Add, Remove, Modify, Move
    Target    interface{}   // NodeId, FileLocation, etc.
    Data      interface{}   // Change-specific data
    Metadata  map[string]interface{}
    Timestamp time.Time
    Source    ChangeSource  // FileSystem, UserEdit, Refactor
}

type ChangeSet struct {
    Id        string
    Changes   []Change
    Timestamp time.Time
    Source    string
    BatchSize int
    Metadata  map[string]interface{}
}

type ChangeSource string
const (
    ChangeSourceFileSystem ChangeSource = "filesystem"
    ChangeSourceUserEdit   ChangeSource = "user_edit"
    ChangeSourceRefactor   ChangeSource = "refactor"
    ChangeSourceImport     ChangeSource = "import"
)
```

## Parser Interfaces

### Parser Manager
```go
type ParserManager interface {
    // Core parsing
    ParseFile(filePath string, language Language) (*AST, error)
    ParseFileVersioned(filePath, content, version string) (*VersionedAST, error)
    ParseContent(content string, language Language) (*AST, error)
    
    // Symbol extraction
    ExtractSymbols(ast *AST) ([]*Symbol, error)
    ExtractImports(ast *AST) ([]*Import, error)
    ResolveImportAlias(importPath, fromFile string) (string, error)
    
    // Language support
    GetSupportedLanguages() []Language
    ClassifyFile(filePath string) (*FileClassification, error)
    DetectLanguage(filePath string) (*Language, error)
    
    // Cache management
    GetASTCache() ASTCache
    InvalidateCache(fileId string) error
    ClearCache() error
}

type LanguageParser interface {
    // Basic parsing
    Parse(content string) (*AST, error)
    ParseWithOptions(content string, options ParseOptions) (*AST, error)
    
    // Language-specific features
    ExtractSymbols(ast *AST) ([]*Symbol, error)
    ExtractComments(ast *AST) ([]*Comment, error)
    ExtractImports(ast *AST) ([]*Import, error)
    
    // Validation
    Validate(ast *AST) []ValidationError
    GetParseErrors() []ParseError
    
    // Configuration
    SetQueryPath(path string) error
    LoadQueries() error
    GetLanguageInfo() LanguageInfo
}

type ASTCache interface {
    // Basic cache operations
    Get(fileId string, version ...string) (*VersionedAST, error)
    Set(fileId string, ast *VersionedAST) error
    Invalidate(fileId string) error
    Clear() error
    
    // Diff cache
    GetDiffCache(fileId string) ([]*ASTDiff, error)
    SetDiffCache(fileId string, diffs []*ASTDiff) error
    
    // Cache management
    Size() int
    Stats() CacheStats
    SetMaxSize(size int)
    SetTTL(ttl time.Duration)
}
```

### Language Support
```go
type Language struct {
    Name         string
    Extensions   []string
    Parser       string
    TreeSitter   string
    QueryPath    string
    Enabled      bool
    Features     LanguageFeatures
    Configuration map[string]interface{}
}

type LanguageFeatures struct {
    SupportsClasses     bool
    SupportsInterfaces  bool
    SupportsGenerics    bool
    SupportsModules     bool
    SupportsDecorators  bool
    SupportsAsync       bool
    TypeSystem          TypeSystemLevel
}

type TypeSystemLevel string
const (
    TypeSystemNone     TypeSystemLevel = "none"
    TypeSystemDynamic  TypeSystemLevel = "dynamic"
    TypeSystemStatic   TypeSystemLevel = "static"
    TypeSystemGradual  TypeSystemLevel = "gradual"
)
```

## Compact Controller Interfaces

### Core Compaction
```go
type CompactController interface {
    // Command execution
    ExecuteCommand(command string, context *CompactContext) (*CompactResult, error)
    ParseCommand(command string) (*CompactCommand, error)
    
    // Predefined strategies
    CompactMinimal() (*CompactResult, error)
    CompactBalanced() (*CompactResult, error)
    CompactAggressive() (*CompactResult, error)
    
    // Task-specific compaction
    CompactForTask(task TaskType) (*CompactResult, error)
    CompactToTokenLimit(maxTokens int) (*CompactResult, error)
    CompactWithFocus(focusFiles []string) (*CompactResult, error)
    
    // Interactive features
    PreviewCompaction(strategy CompactStrategy) (*CompactPreview, error)
    UndoCompaction() error
    GetCompactionHistory() ([]*CompactHistory, error)
    
    // Strategy management
    SaveStrategy(name string, strategy CompactStrategy) error
    LoadStrategy(name string) (*CompactStrategy, error)
    ListStrategies() ([]string, error)
}

type CompactStrategy interface {
    // Core methods
    Apply(graph *CodeGraph) (*CodeGraph, error)
    Preview(graph *CodeGraph) (*CompactPreview, error)
    CalculateQuality(original, compacted *CodeGraph) (*QualityScore, error)
    
    // Configuration
    GetName() string
    GetDescription() string
    GetTokenTarget() float64
    GetRules() []CompactRule
    
    // Validation
    Validate() []ValidationError
    IsReversible() bool
}

type QualityAnalyzer interface {
    // Quality assessment
    CalculateQuality(original, compacted *CodeGraph) (*QualityScore, error)
    AnalyzeSymbolCoverage(original, compacted *CodeGraph) float64
    AnalyzeRelationshipPreservation(original, compacted *CodeGraph) float64
    AnalyzeContextCoherence(original, compacted *CodeGraph) float64
    
    // Validation
    ValidateQuality(score *QualityScore, threshold float64) []QualityWarning
    SuggestImprovements(score *QualityScore) []QualityImprovement
}
```

### Task Management
```go
type TaskType interface {
    GetName() string
    GetDescription() string
    GetPreservePatterns() []SymbolPattern
    GetRemovePatterns() []SymbolPattern
    GetPriorityWeights() map[string]float64
    
    ShouldPreserve(symbol *Symbol) bool
    ShouldRemove(symbol *Symbol) bool
    CalculatePriority(symbol *Symbol) float64
}

type SymbolPattern interface {
    Match(symbol *Symbol) bool
    GetPattern() string
    GetType() PatternType
    GetPriority() int
}

type PatternType string
const (
    PatternTypeExact  PatternType = "exact"
    PatternTypeRegex  PatternType = "regex"
    PatternTypeGlob   PatternType = "glob"
    PatternTypeCustom PatternType = "custom"
)
```

## Storage Interfaces

### Graph Storage
```go
type GraphStorage interface {
    // Graph operations
    SaveGraph(graph *CodeGraph) error
    LoadGraph(id string) (*CodeGraph, error)
    DeleteGraph(id string) error
    ListGraphs() ([]GraphMetadata, error)
    
    // Version management
    SaveVersion(graphId string, version *GraphVersion) error
    LoadVersion(graphId, versionId string) (*CodeGraph, error)
    ListVersions(graphId string) ([]GraphVersion, error)
    
    // Patch management
    SavePatch(patch *GraphPatch) error
    LoadPatches(graphId string, since time.Time) ([]GraphPatch, error)
    ApplyPatches(graphId string, patches []GraphPatch) error
}

type CacheStorage interface {
    // Cache operations
    Get(key string) ([]byte, error)
    Set(key string, value []byte, ttl time.Duration) error
    Delete(key string) error
    Clear() error
    
    // Batch operations
    GetMulti(keys []string) (map[string][]byte, error)
    SetMulti(items map[string][]byte, ttl time.Duration) error
    DeleteMulti(keys []string) error
    
    // Cache management
    Size() int
    Stats() CacheStats
    Cleanup() error
}

type ConfigStorage interface {
    // Configuration management
    LoadConfig(path string) (*Config, error)
    SaveConfig(path string, config *Config) error
    ValidateConfig(config *Config) []ValidationError
    
    // Defaults and templates
    GetDefaultConfig() *Config
    GetConfigTemplate(template string) (*Config, error)
    ListConfigTemplates() ([]string, error)
}
```

### Metrics and Monitoring
```go
type MetricsCollector interface {
    // Counter metrics
    IncrementCounter(name string, tags map[string]string)
    AddCounter(name string, value float64, tags map[string]string)
    
    // Gauge metrics
    SetGauge(name string, value float64, tags map[string]string)
    
    // Histogram metrics
    RecordHistogram(name string, value float64, tags map[string]string)
    RecordTiming(name string, duration time.Duration, tags map[string]string)
    
    // Custom metrics
    RecordCustom(metric CustomMetric)
    
    // Export
    Export() ([]byte, error)
    ExportPrometheus() ([]byte, error)
    ExportJSON() ([]byte, error)
}

type HealthChecker interface {
    // Health checks
    CheckHealth() (*HealthStatus, error)
    CheckComponent(component string) (*ComponentHealth, error)
    RegisterHealthCheck(name string, check HealthCheckFunc)
    
    // Monitoring
    GetHealthHistory() ([]HealthSnapshot, error)
    GetAlerts() ([]HealthAlert, error)
    SetHealthThreshold(component string, threshold HealthThreshold)
}
```

## REST API Specifications

### Core Endpoints
```yaml
openapi: 3.0.3
info:
  title: CodeContext API
  version: 2.0.0

paths:
  /api/v1/projects:
    post:
      summary: Create a new project
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ProjectConfig'
      responses:
        '201':
          description: Project created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'

  /api/v1/projects/{projectId}/generate:
    post:
      summary: Generate context map
      parameters:
        - name: projectId
          in: path
          required: true
          schema:
            type: string
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/GenerateRequest'
      responses:
        '200':
          description: Map generated successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/MapResult'

  /api/v1/projects/{projectId}/update:
    post:
      summary: Update context map incrementally
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateRequest'
      responses:
        '200':
          description: Map updated successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/PatchResult'

  /api/v1/projects/{projectId}/compact:
    post:
      summary: Execute compact command
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CompactRequest'
      responses:
        '200':
          description: Compaction executed
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/CompactResult'

components:
  schemas:
    ProjectConfig:
      type: object
      properties:
        name:
          type: string
        path:
          type: string
        languages:
          type: array
          items:
            $ref: '#/components/schemas/Language'
        excludePatterns:
          type: array
          items:
            type: string

    CompactRequest:
      type: object
      properties:
        level:
          type: string
          enum: [minimal, balanced, aggressive]
        task:
          type: string
          enum: [debugging, refactoring, documentation, review, testing]
        maxTokens:
          type: integer
        preview:
          type: boolean
        focusFiles:
          type: array
          items:
            type: string
```

## GraphQL Schema

### Core Schema
```graphql
type Query {
  # Project queries
  project(id: ID!): Project
  projects(filter: ProjectFilter): [Project!]!
  
  # Graph queries
  graph(projectId: ID!): CodeGraph
  symbols(projectId: ID!, filter: SymbolFilter): [Symbol!]!
  
  # Search
  searchSymbols(query: String!, limit: Int): [Symbol!]!
  searchFiles(query: String!, limit: Int): [File!]!
  
  # Analytics
  getMetrics(projectId: ID!, timeRange: TimeRange): Metrics
  getHealth(): HealthStatus
}

type Mutation {
  # Project operations
  createProject(config: ProjectConfigInput!): Project!
  updateProject(id: ID!, config: ProjectConfigInput!): Project!
  deleteProject(id: ID!): Boolean!
  
  # Map operations
  generateMap(projectId: ID!, config: GenerateConfigInput): MapResult!
  updateMap(projectId: ID!, changes: [FileChangeInput!]!): PatchResult!
  
  # Compact operations
  compactMap(projectId: ID!, command: CompactCommandInput!): CompactResult!
  undoCompaction(projectId: ID!): Boolean!
  
  # Configuration
  saveStrategy(projectId: ID!, strategy: CompactStrategyInput!): CompactStrategy!
  deleteStrategy(projectId: ID!, name: String!): Boolean!
}

type Subscription {
  # Real-time updates
  mapUpdates(projectId: ID!): MapUpdate!
  processingProgress(operationId: ID!): Progress!
  healthChanges: HealthStatus!
}

# Core types
type Project {
  id: ID!
  name: String!
  path: String!
  config: ProjectConfig!
  createdAt: DateTime!
  updatedAt: DateTime!
  status: ProjectStatus!
}

type CodeGraph {
  nodes: [GraphNode!]!
  edges: [GraphEdge!]!
  metadata: GraphMetadata!
  version: GraphVersion!
}

type Symbol {
  id: ID!
  name: String!
  type: SymbolType!
  location: FileLocation!
  signature: String
  documentation: String
  language: String!
  importance: Float!
}
```

## CLI Interface

### Command Structure
```bash
codecontext [global-flags] <command> [command-flags] [arguments]

Global Flags:
  --config, -c     Config file path
  --verbose, -v    Verbose output
  --output, -o     Output file
  --help, -h       Help
  --version        Version

Commands:
  init             Initialize project
  generate         Generate context map
  update           Update incrementally
  compact          Optimize context
  config           Manage configuration
  cache            Cache operations
  graph            Graph operations
  metrics          Show metrics
  health           Health check
```

### Command Interfaces
```go
type CLICommand interface {
    // Command metadata
    Name() string
    Description() string
    Usage() string
    Examples() []string
    
    // Flag management
    DefineFlags() []Flag
    ValidateFlags(flags map[string]interface{}) error
    
    // Execution
    Execute(ctx *CLIContext) error
    
    // Help and completion
    GetHelp() string
    GetCompletion(partial string) []string
}

type CLIContext struct {
    Command     string
    Flags       map[string]interface{}
    Arguments   []string
    Config      *Config
    Output      io.Writer
    Input       io.Reader
    Logger      Logger
    Metrics     MetricsCollector
}

type Flag struct {
    Name        string
    Short       string
    Type        FlagType
    Required    bool
    Default     interface{}
    Description string
    Validation  func(interface{}) error
}
```

---

*This API specification serves as the definitive reference for all interfaces in the CodeContext system and should be kept synchronized with the implementation.*