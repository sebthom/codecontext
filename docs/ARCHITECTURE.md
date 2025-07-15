# CodeContext Architecture Documentation

**Version:** 2.2  
**Last Updated:** July 2025  
**Status:** Production Release - All Core Components Implemented

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Principles](#architecture-principles)
3. [System Overview](#system-overview)
4. [Core Components](#core-components)
5. [Data Flow](#data-flow)
6. [Technology Stack](#technology-stack)
7. [Performance Requirements](#performance-requirements)
8. [Security Considerations](#security-considerations)

## Executive Summary

CodeContext is an automated repository mapping system that generates intelligent context maps for AI-powered development tools. The system processes source code repositories to create optimized context representations while managing token constraints through advanced compaction strategies.

**Key Innovations:**
- **Real Tree-sitter Integration**: Production-ready AST parsing with JavaScript/TypeScript grammars ✅ IMPLEMENTED
- **Intelligent Code Analysis**: Symbol extraction, dependency mapping, and rich context generation ✅ IMPLEMENTED
- **Multi-Language Support**: Pluggable parser architecture using Tree-sitter ✅ IMPLEMENTED
- **Virtual Graph Architecture**: Virtual DOM-inspired approach for O(changes) complexity incremental updates ✅ IMPLEMENTED
- **Interactive Compaction**: Dynamic context optimization with `/compact` commands ✅ IMPLEMENTED
- **Token Optimization**: Intelligent context reduction while preserving code semantics ✅ IMPLEMENTED
- **MCP Server Integration**: Real-time AI integration with Claude Desktop ✅ IMPLEMENTED

## Architecture Principles

### 1. Modularity First
- Loosely coupled components with clear interfaces
- Dependency injection throughout the system
- Interface-based design enabling easy testing and mocking

### 2. Performance Oriented
- Streaming processing for large repositories
- Incremental updates via Virtual Graph Engine
- Multi-level caching (AST, diff, computed results)
- Parallel processing where possible

### 3. Developer Experience
- CLI-first design with rich interactive commands
- Clear error messages with recovery suggestions
- Comprehensive testing and documentation
- Self-documenting code with examples

### 4. Extensibility
- Plugin architecture for language parsers
- Configurable compaction strategies
- Template-based output generation
- API-first design for integration

### 5. Privacy and Local-First
- No mandatory cloud dependencies
- Local file system processing
- Optional distributed caching
- Sensitive data protection

## System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        CodeContext System                       │
├─────────────────┬─────────────────┬─────────────────────────────┤
│   Client Layer  │  Core Engine    │     Infrastructure         │
│                 │                 │                             │
│ ┌─────────────┐ │ ┌─────────────┐ │ ┌─────────────────────────┐ │
│ │ CLI Tool    │ │ │ Orchestrator│ │ │ Virtual Graph Engine    │ │
│ │ IDE Exts    │ │ │ Parser Mgr  │ │ │ ┌─────────┬───────────┐ │ │
│ │ REST API    │ │ │ Analyzer    │ │ │ │ Shadow  │ Actual    │ │ │
│ │ Interactive │ │ │ Optimizer   │ │ │ │ Graph   │ Graph     │ │ │
│ │ Commands    │ │ │ Generator   │ │ │ └─────────┴───────────┘ │ │
│ └─────────────┘ │ │ Compact Ctrl│ │ │ AST Differ │ Reconciler│ │ │
│                 │ └─────────────┘ │ └─────────────────────────┘ │
└─────────────────┴─────────────────┴─────────────────────────────┘
┌─────────────────────────────────────────────────────────────────┐
│                    Storage & Integration                        │
├─────────────┬─────────────────┬─────────────────┬─────────────────┤
│ File System │ Cache Layer     │ Vector Store    │ External APIs   │
│ - Local FS  │ - AST Cache     │ - Embeddings    │ - Git           │
│ - Git Repos │ - Diff Cache    │ - Semantic      │ - CI/CD         │
│ - Config    │ - Result Cache  │   Search        │ - AI Platforms  │
└─────────────┴─────────────────┴─────────────────┴─────────────────┘
```

## Core Components

### 1. Orchestrator
**Purpose**: Central coordination and workflow management

**Responsibilities**:
- Request routing and validation
- Workflow orchestration with timeout/cancellation
- Progress tracking and error handling
- Checkpoint-based resumable operations

**Key Features**:
- Context-aware cancellation
- Progress reporting with real-time updates
- Error recovery with rollback support
- Performance metrics collection

### 2. Virtual Graph Engine
**Purpose**: Efficient incremental updates using Virtual DOM pattern

**Architecture**:
```go
type VirtualGraphEngine struct {
    shadow   *CodeGraph      // Virtual representation
    actual   *CodeGraph      // Committed state  
    pending  []ChangeSet     // Batched changes
    differ   *ASTDiffer      // Diff computation
    reconciler *Reconciler   // Change application
}
```

**Key Operations**:
- `Diff(oldAST, newAST)`: Compute structural differences
- `BatchChange(change)`: Accumulate changes for batching
- `Reconcile()`: Generate minimal update plan
- `Commit(plan)`: Apply changes to actual graph

### 3. Parser Manager
**Purpose**: Multi-language AST generation and symbol extraction

**Supported Languages**:
- TypeScript/JavaScript (primary)
- Python (planned)
- Go (planned)
- Java (planned)

**Features**:
- Tree-sitter integration for robust parsing
- Language-agnostic symbol extraction
- Import resolution and alias handling
- AST caching with versioning

### 4. Compact Controller
**Purpose**: Interactive context optimization

**Compaction Levels**:
- **Minimal (30%)**: Keep only core APIs and critical paths
- **Balanced (60%)**: Good compromise between size and completeness
- **Aggressive (15%)**: Absolute essentials only

**Task-Specific Strategies**:
- **Debugging**: Preserve error handling, logging, state management
- **Refactoring**: Focus on class/interface structures
- **Documentation**: Emphasize public APIs and type definitions

### 5. Analyzer Graph
**Purpose**: Code relationship analysis and importance ranking

**Graph Construction**:
- Dependency relationships (imports, calls, inheritance)
- Symbol importance scoring (PageRank-based)
- Module boundary detection
- Change impact analysis

**Algorithms**:
- Incremental PageRank for importance scoring
- Community detection for module boundaries
- Shortest path analysis for dependencies

### 6. Generator
**Purpose**: Multi-format output generation

**Supported Formats**:
- **Markdown**: Primary format with interactive TOC
- **JSON**: Structured data for programmatic access
- **YAML**: Configuration and metadata
- **Custom**: Plugin-based extensions

## Data Flow

### Initial Generation Flow
```
Files → Parser → AST → Analyzer → Graph → Optimizer → Generator → Output
```

### Incremental Update Flow
```
File Change → Virtual Graph → Diff → Reconcile → Patch → Update Output
```

### Compact Command Flow
```
User Command → Compact Controller → Strategy Selection → Graph Transform → Preview/Apply
```

## Technology Stack

### Core Technologies (Current Implementation)
- **Language**: Go 1.24+ for performance and single binary distribution ✅ IMPLEMENTED
- **Parser**: Tree-sitter with official Go bindings for robust AST generation ✅ IMPLEMENTED
- **CLI**: Cobra framework for rich command-line interface ✅ IMPLEMENTED
- **Config**: Viper for flexible configuration management ✅ IMPLEMENTED
- **Analysis**: Custom analyzer package for code graph construction ✅ IMPLEMENTED

### Dependencies (Production)
```go
require (
    github.com/spf13/cobra v1.9.1                              // CLI framework
    github.com/spf13/viper v1.20.1                             // Configuration
    github.com/tree-sitter/go-tree-sitter v0.25.0             // Tree-sitter runtime
    github.com/tree-sitter/tree-sitter-javascript v0.23.1     // JavaScript grammar
)
```

### Language Support (Current)
- **TypeScript**: Real parsing via JavaScript grammar (excellent compatibility)
- **JavaScript**: Official Tree-sitter grammar with full AST support
- **JSON**: Basic parsing with metadata extraction
- **YAML**: Basic parsing with metadata extraction

### Optional Components
- **Redis**: Distributed caching for large teams
- **SQLite**: Local diff storage and history
- **OpenTelemetry**: Observability and metrics

## Performance Requirements

### Latency Targets
- Single file update: <100ms
- Module-level changes: <500ms
- Full repository scan: <1ms per file
- Compact operations: <2s for 100k+ files

### Memory Constraints
- Base memory usage: <50MB
- Per 10k LOC: <10MB additional
- Cache overhead: <20% of base graph size
- Virtual graph: <10% of actual graph size

### Scalability Limits
- Repository size: 1M+ files supported
- Concurrent operations: 10+ parallel processes
- Cache size: 10GB+ configurable
- History retention: 1000+ operations

### Benchmarks
```bash
# Performance testing commands
codecontext perf --benchmark
codecontext perf --memory-profile  
codecontext perf --incremental-stats
```

## Security Considerations

### Data Protection
- Local-first processing (no mandatory cloud uploads)
- Configurable data retention policies
- Sensitive pattern detection and filtering
- Secure temporary file handling

### Access Control
- File system permission respect
- Git ignore pattern compliance
- User-defined exclusion patterns
- API key and token protection

### Privacy Features
- Optional anonymization modes
- Configurable data sharing policies
- Local-only operation capability
- Audit logging for data access

## Configuration

### Project Configuration (.codecontext/config.yaml)
```yaml
version: "2.0"

virtual_graph:
  enabled: true
  batch_threshold: 5
  batch_timeout: 500ms

languages:
  typescript:
    extensions: [".ts", ".tsx"]
    parser: "tree-sitter-typescript"
  
compact_profiles:
  minimal:
    token_target: 0.3
    preserve: ["core", "api"]
```

### CLI Configuration
```bash
# Global settings
codecontext config set global.cache_size 1GB
codecontext config set global.parallel_workers 4

# Project-specific  
codecontext config set local.output_format markdown
codecontext config set local.include_metrics true
```

## Monitoring and Observability

### Metrics Collection
- Processing time per operation
- Memory usage trends
- Cache hit rates
- Error rates and types

### Health Checks
```json
{
  "status": "healthy",
  "components": {
    "virtual_graph": {
      "status": "ok",
      "pending_changes": 3,
      "memory_usage_mb": 45.2
    },
    "parser_cache": {
      "hit_rate": 0.94,
      "size_mb": 23.1
    }
  }
}
```

### Debugging Support
- Verbose logging with levels
- Operation tracing
- Performance profiling
- State inspection commands

## Future Enhancements

### Phase 2 Features
- Real-time collaboration support
- Advanced AI integration
- Custom parser plugins
- Visual graph representations

### Phase 3 Features
- Cloud-native deployment
- Enterprise security features
- Advanced analytics
- Multi-repository support

---

*This architecture document is maintained alongside the codebase and updated with each major release.*