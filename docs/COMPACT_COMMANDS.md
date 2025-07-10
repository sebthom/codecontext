# Compact Commands Specification

**Version:** 2.0  
**Component:** Compact Controller  
**Status:** Design Complete, Implementation in Progress

## Overview

The Compact Commands system provides interactive context optimization through the `/compact` command interface. This allows users to dynamically reduce context size based on task requirements while maintaining code semantics and relationships.

## Command Interface

### Basic Syntax
```bash
codecontext compact [flags]
```

### Available Flags
```bash
--level, -l       Compaction level (minimal, balanced, aggressive)
--task, -t        Task-specific optimization (debugging, refactoring, documentation, review, testing)
--tokens, -n      Target token limit
--preview, -p     Preview compaction without applying
--focus, -f       Focus on specific files or directories
--preserve        Additional patterns to preserve
--remove          Additional patterns to remove
--strategy        Custom strategy name
--undo           Undo last compaction
--history        Show compaction history
```

## Compaction Levels

### 1. Minimal (30% reduction, 95% quality)
**Target Use Case**: Critical debugging, production issues

**Preservation Strategy**:
- Core API definitions
- Error handling code
- Critical data structures
- Main execution paths

**Removal Strategy**:
- All test files
- Example code
- Documentation comments
- Generated code
- Build artifacts

```bash
codecontext compact --level minimal
```

**Expected Output**:
```
📊 Compaction Preview:
   Strategy: Minimal
   Original tokens: 150,000
   Compacted tokens: 45,000
   Reduction: 70.0%
   Quality score: 0.95
   
   Preserved:
   ✓ Core APIs (src/api/*.ts)
   ✓ Error handlers (src/errors/*.ts)
   ✓ Main logic (src/core/*.ts)
   
   Removed:
   ✗ Test files (2,341 files)
   ✗ Examples (src/examples/*)
   ✗ Generated code (dist/*, build/*)
```

### 2. Balanced (40% reduction, 85% quality)
**Target Use Case**: General development, code review

**Preservation Strategy**:
- Core and peripheral APIs
- Type definitions
- Interface declarations
- Important utility functions
- Configuration files

**Removal Strategy**:
- Test files
- Example code
- Some generated content

```bash
codecontext compact --level balanced
```

### 3. Aggressive (85% reduction, 70% quality)
**Target Use Case**: Quick overview, architecture understanding

**Preservation Strategy**:
- Core APIs only
- Essential type definitions
- Critical interfaces

**Removal Strategy**:
- All test code
- All examples
- All comments
- Implementation details
- Utility functions

```bash
codecontext compact --level aggressive
```

## Task-Specific Compaction

### 1. Debugging Focus
**Command**: `codecontext compact --task debugging`

**Optimization Strategy**:
```yaml
debugging:
  preserve_patterns:
    - ".*[Ee]rror.*"
    - ".*[Ee]xception.*"
    - ".*[Ll]og.*"
    - ".*[Dd]ebug.*"
    - ".*[Tt]race.*"
    - "try|catch|finally"
    - "console\.(log|error|warn|debug)"
  
  expand_context:
    - error_call_stacks
    - exception_handlers
    - logging_statements
    - state_variables
    - debugging_utilities
  
  remove_patterns:
    - ".*[Tt]est.*"
    - ".*[Ss]pec.*"
    - ".*[Mm]ock.*"
    - documentation_comments
    - example_code
```

**Example Output**:
```typescript
// Preserved: Error handling and logging
class UserService {
  async getUser(id: string): Promise<User> {
    try {
      console.log(`Fetching user with ID: ${id}`);
      const user = await this.repository.findById(id);
      if (!user) {
        throw new UserNotFoundError(`User ${id} not found`);
      }
      return user;
    } catch (error) {
      console.error(`Error fetching user ${id}:`, error);
      throw error;
    }
  }
}

class UserNotFoundError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'UserNotFoundError';
  }
}
```

### 2. Refactoring Focus
**Command**: `codecontext compact --task refactoring`

**Optimization Strategy**:
```yaml
refactoring:
  preserve_patterns:
    - class_declarations
    - interface_definitions
    - function_signatures
    - type_definitions
    - inheritance_relationships
    - dependency_imports
  
  expand_context:
    - class_hierarchies
    - method_signatures
    - property_definitions
    - constructor_parameters
  
  remove_patterns:
    - method_implementations
    - private_methods
    - test_files
    - example_code
```

### 3. Documentation Focus
**Command**: `codecontext compact --task documentation`

**Optimization Strategy**:
```yaml
documentation:
  preserve_patterns:
    - public_interfaces
    - exported_functions
    - type_definitions
    - jsdoc_comments
    - readme_files
    - api_documentation
  
  expand_context:
    - parameter_types
    - return_types
    - usage_examples
    - configuration_options
  
  remove_patterns:
    - private_implementations
    - internal_utilities
    - test_implementations
```

## Custom Strategies

### Creating Custom Strategies
```bash
# Save current settings as a custom strategy
codecontext compact --level balanced --task debugging --save-strategy "debug-balanced"

# Use custom strategy
codecontext compact --strategy "debug-balanced"
```

### Strategy Configuration File
```yaml
# .codecontext/strategies/debug-balanced.yaml
name: "debug-balanced"
description: "Balanced compaction optimized for debugging"
token_target: 0.5
quality_threshold: 0.8

preserve_rules:
  - name: "core_apis"
    pattern: "src/api/**/*.ts"
    priority: 1
  - name: "error_handling"
    pattern: ".*[Ee]rror.*|.*[Ee]xception.*"
    priority: 1
  - name: "logging"
    pattern: ".*[Ll]og.*|console\\."
    priority: 2

remove_rules:
  - name: "tests"
    pattern: "**/*.test.*|**/*.spec.*"
    priority: 1
  - name: "examples"
    pattern: "**/examples/**"
    priority: 2

quality_weights:
  symbol_coverage: 0.4
  relationship_preservation: 0.3
  context_coherence: 0.3
```

## Token-Limited Compaction

### Usage
```bash
# Target specific token count
codecontext compact --tokens 50000

# Combine with other constraints
codecontext compact --tokens 30000 --preserve "src/core/**" --remove "**/*.test.*"
```

### Algorithm
1. **Initial Assessment**: Calculate current token count
2. **Strategy Selection**: Choose base strategy to reach target
3. **Iterative Refinement**: Remove low-priority content until target is met
4. **Quality Validation**: Ensure minimum quality threshold

```go
type TokenTargetStrategy struct {
    MaxTokens        int
    QualityThreshold float64
    PreservePriority []string
    RemovePriority   []string
}

func (s *TokenTargetStrategy) Optimize(graph *CodeGraph) *CompactResult {
    current := s.countTokens(graph)
    if current <= s.MaxTokens {
        return &CompactResult{Success: true, Changes: nil}
    }
    
    // Iteratively remove lowest priority content
    for current > s.MaxTokens {
        removed := s.removeLowestPriority(graph)
        if removed == 0 {
            break // Can't reduce further
        }
        current = s.countTokens(graph)
    }
    
    return s.validateQuality(graph)
}
```

## Focus Mode

### File-Specific Focus
```bash
# Focus on specific files
codecontext compact --focus "src/auth.ts,src/user.ts"

# Focus on directories
codecontext compact --focus "src/core/**"

# Combine with compaction level
codecontext compact --level minimal --focus "src/api/**"
```

### Context Expansion
When focusing on specific files, the system automatically includes:
- Direct dependencies
- Immediate dependents  
- Shared types and interfaces
- Related error handlers

```yaml
focus_expansion:
  include_dependencies: true
  include_dependents: true
  include_types: true
  include_errors: true
  max_depth: 2
```

## Preview Mode

### Preview Output
```bash
codecontext compact --level minimal --preview
```

**Sample Preview**:
```
📊 Compaction Preview - Minimal Level

Token Analysis:
├── Original: 147,532 tokens
├── Estimated: 44,260 tokens  
├── Reduction: 70.0%
└── Quality Score: 0.95

File Impact:
├── Preserved: 142 files
├── Removed: 1,847 files
├── Modified: 23 files
└── Unchanged: 89 files

Symbol Impact:
├── Functions: 1,247 → 374 (70% reduction)
├── Classes: 156 → 89 (43% reduction)  
├── Interfaces: 78 → 67 (14% reduction)
└── Types: 234 → 198 (15% reduction)

Quality Metrics:
├── Symbol Coverage: 0.94
├── Relationship Preservation: 0.96
├── Context Coherence: 0.95
└── Overall Score: 0.95

⚠️  Warnings:
• Some error handling code will be removed
• 12 utility functions may lose context

✓ Ready to apply - run without --preview to execute
```

## History and Undo

### Compaction History
```bash
codecontext compact --history
```

**Output**:
```
Compaction History:
1. 2025-01-15 14:30:22 | minimal     | 70% reduction | ✓ Active
2. 2025-01-15 13:15:11 | balanced    | 40% reduction | 
3. 2025-01-15 12:00:05 | debugging   | 55% reduction |
4. 2025-01-15 11:30:44 | aggressive  | 85% reduction |
```

### Undo Compaction
```bash
# Undo last compaction
codecontext compact --undo

# Undo to specific point
codecontext compact --undo --to-id "compact-20250115-133011"
```

## Quality Scoring

### Quality Metrics
```go
type QualityScore struct {
    SymbolCoverage          float64  // 0-1, important symbols preserved
    RelationshipPreservation float64  // 0-1, dependencies maintained
    ContextCoherence        float64  // 0-1, logical grouping preserved
    SemanticIntegrity       float64  // 0-1, meaning preserved
    Overall                 float64  // Weighted average
}
```

### Quality Calculation
```go
func CalculateQuality(original, compacted *CodeGraph) QualityScore {
    return QualityScore{
        SymbolCoverage:          calculateSymbolCoverage(original, compacted),
        RelationshipPreservation: calculateRelationshipPreservation(original, compacted),
        ContextCoherence:        calculateContextCoherence(original, compacted),
        SemanticIntegrity:       calculateSemanticIntegrity(original, compacted),
        Overall:                 calculateWeightedAverage(metrics),
    }
}
```

## Configuration

### Global Settings
```yaml
compact:
  default_level: "balanced"
  default_preview: true
  quality_threshold: 0.8
  max_history: 50
  
  auto_save_strategies: true
  strategy_sharing: false
  
  warnings:
    show_impact: true
    show_removed_symbols: true
    confirm_aggressive: true
```

### Per-Project Settings
```yaml
project:
  preserve_always:
    - "src/types/**"
    - "src/interfaces/**"
    - "README.md"
  
  remove_always:
    - "**/*.test.*"
    - "**/*.spec.*"
    - "dist/**"
    - "build/**"
    - "node_modules/**"
```

## API Integration

### Programmatic Access
```go
// Go API
compactController := compact.NewController(config)
result, err := compactController.CompactMinimal()

// REST API
POST /api/v1/projects/{id}/compact
{
  "level": "minimal",
  "preview": true,
  "focus": ["src/core/**"]
}
```

## Error Handling

### Common Errors
1. **Insufficient Quality**: Result quality below threshold
2. **Cannot Reach Target**: Unable to reduce to token limit
3. **Dependency Conflicts**: Required symbols would be removed
4. **File System Errors**: Cannot write compacted output

### Recovery Actions
```bash
# Reset to original state
codecontext compact --reset

# Force compaction (ignore quality threshold)  
codecontext compact --level aggressive --force

# Diagnose compaction issues
codecontext compact --diagnose
```

## Testing

### Test Commands
```bash
# Test compaction strategies
codecontext compact --test-strategy minimal

# Benchmark compaction performance
codecontext compact --benchmark

# Validate compaction results
codecontext compact --validate
```

---

*This specification defines the complete Compact Commands system and should be referenced for all compaction-related development.*