# 🤖 Using CodeContext with Claude - Complete Guide

## 🎯 Overview

CodeContext generates intelligent, token-optimized context maps specifically designed for AI development workflows with Claude. This guide shows you how to maximize your productivity when working with Claude on coding projects.

## 🚀 Quick Start

### 1. Install CodeContext
```bash
# Via Homebrew (macOS)
brew install --HEAD --build-from-source https://raw.githubusercontent.com/nmakod/codecontext/main/Formula/codecontext.rb

# Or download from releases
curl -L https://github.com/nmakod/codecontext/releases/download/v2.0.0/codecontext-2.0.0-darwin-arm64.tar.gz | tar xz
sudo mv codecontext /usr/local/bin/
```

### 2. Initialize Your Project
```bash
cd your-project
codecontext init
```

### 3. Generate Context Map
```bash
codecontext generate
```

### 4. Use with Claude
Copy the generated `CLAUDE.md` content and paste it into your Claude conversation.

## 📋 Step-by-Step Workflow

### Phase 1: Project Setup

**1. Navigate to Your Project**
```bash
cd /path/to/your/project
```

**2. Initialize CodeContext**
```bash
codecontext init
```
This creates `.codecontext/config.yaml` with intelligent defaults.

**3. Customize Configuration (Optional)**
```yaml
# .codecontext/config.yaml
project:
  name: "my-awesome-app"
  description: "React TypeScript application with Node.js backend"

parser:
  languages: ["typescript", "javascript", "go", "json"]
  
output:
  format: "markdown"
  include_stats: true
  include_relationships: true
  max_file_size: 1048576  # 1MB

analysis:
  include_patterns:
    - "src/**/*"
    - "components/**/*"
    - "utils/**/*"
  exclude_patterns:
    - "node_modules/**"
    - "dist/**"
    - "*.test.*"
    - "*.spec.*"
```

### Phase 2: Generate Context Map

**Basic Generation**
```bash
codecontext generate
```

**Advanced Options**
```bash
# Generate with specific output file
codecontext generate --output my-context.md

# Generate with verbose output
codecontext generate --verbose

# Generate for specific paths only
codecontext generate src/ components/
```

### Phase 3: Using with Claude

#### 🎯 Best Practices for Claude Conversations

**1. Start with Context**
```
I'm working on a [project type] project. Here's the current codebase context:

[Paste CLAUDE.md content here]

I need help with [specific task/problem].
```

**2. Reference Specific Files**
```
Based on the context map, I want to modify the UserService class in src/services/user.ts:12. 
Can you help me add a new method for handling user authentication?
```

**3. Use Symbol References**
```
Looking at the symbol analysis, I see we have a processPayment function in 
src/utils/payment.ts:45. I need to refactor this to handle multiple payment methods.
```

## 🔄 Incremental Workflows

### Watch Mode for Active Development

**Start Watch Mode**
```bash
codecontext watch
```

This automatically updates the context map when files change, perfect for:
- **Live debugging sessions with Claude**
- **Iterative development workflows**
- **Real-time code reviews**

**Watch Specific Directories**
```bash
codecontext watch src/ components/
```

### Update Existing Context
```bash
# Update only changed files
codecontext update

# Force full regeneration
codecontext update --force
```

## 📊 Understanding the Context Map

### Key Sections for Claude

**1. Overview Section**
- **Files Analyzed**: Total scope of the project
- **Symbols Extracted**: Functions, classes, variables identified
- **Languages Detected**: Multi-language project support
- **Import Relationships**: Dependency understanding

**2. File Analysis Table**
```markdown
| File | Language | Lines | Symbols | Imports | Type |
|------|----------|-------|---------|---------|------|
| src/components/Header.tsx | typescript | 45 | 8 | 3 | component |
```
Use this to reference specific files with Claude.

**3. Symbol Analysis**
```markdown
| Symbol | Type | File | Line | Signature |
|--------|------|------|------|----------|
| UserService | class | src/services/user.ts | 12 | class UserService |
| authenticate | method | src/services/user.ts | 25 | authenticate(token: string) |
```
Perfect for discussing specific functions/classes with Claude.

**4. Import Relationships**
```markdown
### Import Graph
- src/app.ts → [components/Header, services/UserService, utils/logger]
- components/Header.tsx → [hooks/useAuth, utils/formatters]
```
Helps Claude understand project architecture.

## 🛠️ Advanced Usage Patterns

### 1. Code Review with Claude

**Generate focused context for specific changes**
```bash
# Generate context for recently modified files
git diff --name-only HEAD~1 | xargs codecontext generate --files

# Include in Claude conversation:
# "Here's the context for files I just modified: [paste context]
# Can you review these changes for potential issues?"
```

### 2. Architecture Planning

**Generate high-level overview**
```bash
codecontext generate --compact --focus=architecture
```

**Use with Claude for:**
- System design discussions
- Refactoring planning
- Dependency analysis
- Performance optimization

### 3. Debugging Sessions

**Generate context with error traces**
```bash
codecontext generate --include-tests --verbose

# In Claude conversation:
# "I'm getting this error: [error message]
# Here's my current codebase context: [paste context]
# The error seems related to the UserAuth module. Can you help debug?"
```

### 4. Feature Development

**Before starting new features**
```bash
codecontext generate --output feature-context.md

# Ask Claude:
# "Based on this codebase structure: [paste context]
# What's the best way to implement a new [feature description]?
# Which files should I modify and what patterns should I follow?"
```

## 🎨 Context Optimization Strategies

### 1. Use Compaction for Large Projects

```bash
# Generate compact context for large codebases
codecontext compact --level balanced

# This reduces context size while preserving important information
```

**Compaction Levels:**
- `minimal`: Essential symbols and structure only
- `balanced`: Good balance of detail and brevity (recommended)
- `aggressive`: Maximum compression for very large projects

### 2. Focus on Relevant Areas

**Use include/exclude patterns**
```yaml
# .codecontext/config.yaml
analysis:
  include_patterns:
    - "src/components/**"     # Frontend components
    - "src/services/**"       # Business logic
    - "src/types/**"          # Type definitions
  exclude_patterns:
    - "**/*.test.*"          # Exclude tests
    - "src/legacy/**"        # Exclude legacy code
    - "**/*.generated.*"     # Exclude generated files
```

### 3. Incremental Context Updates

**For ongoing Claude sessions**
```bash
# Generate delta context showing only changes
codecontext update --delta --since="1 hour ago"

# Share with Claude:
# "Here are the changes since our last discussion: [paste delta context]"
```

## 🔧 Configuration Best Practices

### For Different Project Types

**React/Next.js Frontend**
```yaml
parser:
  languages: ["typescript", "javascript", "json"]
analysis:
  include_patterns:
    - "src/**"
    - "components/**"
    - "pages/**"
    - "hooks/**"
    - "utils/**"
  exclude_patterns:
    - "node_modules/**"
    - ".next/**"
    - "**/*.test.*"
```

**Node.js Backend**
```yaml
parser:
  languages: ["typescript", "javascript", "json"]
analysis:
  include_patterns:
    - "src/**"
    - "routes/**"
    - "middleware/**"
    - "models/**"
    - "services/**"
  exclude_patterns:
    - "node_modules/**"
    - "dist/**"
    - "logs/**"
```

**Full-Stack Project**
```yaml
parser:
  languages: ["typescript", "javascript", "go", "json"]
analysis:
  include_patterns:
    - "frontend/src/**"
    - "backend/src/**"
    - "shared/**"
    - "api/**"
  exclude_patterns:
    - "**/node_modules/**"
    - "**/dist/**"
    - "**/*.test.*"
```

## 🎯 Claude Conversation Templates

### 1. Initial Project Analysis
```
I'm working on a [project description] and need your help. 

Here's my current codebase structure:
[Paste CLAUDE.md content]

Based on this context, can you:
1. Identify any architectural issues or improvements
2. Suggest best practices I should follow
3. Point out any potential performance or security concerns
```

### 2. Feature Implementation
```
I want to implement [feature description] in my project.

Current codebase context:
[Paste CLAUDE.md content]

Looking at the existing structure, what's the best approach to:
1. Add this feature without breaking existing functionality
2. Follow the current code patterns and architecture
3. Maintain good separation of concerns
```

### 3. Bug Fixing
```
I'm encountering this error: [error description/stack trace]

Here's my codebase context:
[Paste CLAUDE.md content]

The error seems to be related to [specific area]. Can you help me:
1. Identify the root cause
2. Suggest a fix that fits with the existing codebase
3. Recommend any preventive measures
```

### 4. Code Review
```
I've made some changes to implement [feature/fix]. Here's the updated context:
[Paste CLAUDE.md content]

Can you review this for:
1. Code quality and best practices
2. Potential bugs or edge cases
3. Integration with existing codebase
4. Performance implications
```

### 5. Refactoring
```
I want to refactor [specific component/module] to improve [performance/maintainability/etc].

Current codebase context:
[Paste CLAUDE.md content]

Based on the current structure and dependencies, can you:
1. Suggest a refactoring approach
2. Identify what needs to be updated
3. Help maintain backward compatibility
```

## ⚡ Performance Tips

### 1. Optimize Context Size
```bash
# For very large projects, use focused analysis
codecontext generate --max-files 100 --max-symbols 500

# Or use compaction
codecontext compact --level aggressive --output compact-context.md
```

### 2. Cache for Faster Updates
```bash
# Enable caching for faster subsequent runs
codecontext generate --cache

# Update only changed files
codecontext update --incremental
```

### 3. Parallel Processing
```yaml
# .codecontext/config.yaml
performance:
  concurrent_files: 8        # Process files in parallel
  enable_caching: true       # Cache parsed ASTs
  max_memory_mb: 512        # Memory limit
```

## 🔍 Troubleshooting

### Common Issues

**1. Context Too Large**
```bash
# Solution: Use compaction
codecontext compact --level balanced

# Or exclude unnecessary files
codecontext generate --exclude "**/*.test.*" --exclude "node_modules/**"
```

**2. Missing Languages**
```bash
# Check supported languages
codecontext generate --list-languages

# Add language support in config
# .codecontext/config.yaml
parser:
  languages: ["typescript", "javascript", "go", "python"]
```

**3. Slow Performance**
```bash
# Enable performance monitoring
codecontext generate --verbose --profile

# Optimize with caching
codecontext generate --cache --parallel
```

## 🎉 Success Stories

### Example Workflow: Building a REST API

1. **Initialize project**
   ```bash
   codecontext init
   codecontext generate
   ```

2. **Share with Claude**
   ```
   I'm building a REST API for a task management app. Here's my current structure:
   [paste context]
   
   Help me design the database schema and API endpoints.
   ```

3. **Implement Claude's suggestions**
   
4. **Update context and iterate**
   ```bash
   codecontext update
   ```

5. **Continue development with updated context**

### Result: 
- **3x faster development** with proper context sharing
- **Better code quality** with Claude's architectural guidance  
- **Fewer bugs** due to comprehensive understanding of codebase

## 🔗 Integration with Other Tools

### VS Code Extension (Future)
- Automatic context generation on save
- Direct Claude integration
- Context diff visualization

### GitHub Actions (Future)
- Automatic context updates on PR
- Context-aware code reviews
- Dependency change notifications

### IDE Plugins (Future)
- Real-time context updates
- Smart symbol navigation
- AI-powered suggestions

## 📚 Additional Resources

- **GitHub Repository**: https://github.com/nmakod/codecontext
- **Issue Tracker**: https://github.com/nmakod/codecontext/issues
- **Documentation**: https://github.com/nmakod/codecontext/blob/main/README.md
- **Release Notes**: https://github.com/nmakod/codecontext/releases

---

**Happy coding with Claude! 🚀**

*This guide will be continuously updated based on user feedback and new features.*