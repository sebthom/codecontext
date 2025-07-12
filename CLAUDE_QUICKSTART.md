# 🚀 CodeContext + Claude Quick Start

## ⚡ 30-Second Setup

```bash
# Install
brew install --HEAD --build-from-source https://raw.githubusercontent.com/nmakod/codecontext/main/Formula/codecontext.rb

# Initialize in your project
cd your-project && codecontext init

# Generate context
codecontext generate

# Copy CLAUDE.md content and paste into Claude conversation
```

## 💬 Claude Conversation Templates

### 🎯 New Project Planning
```
I'm starting a [project type] project. Here's my current structure:

[Paste CLAUDE.md content]

Help me plan the architecture and implementation approach for [specific goal].
```

### 🔧 Feature Implementation  
```
I want to add [feature description] to my project.

Current codebase context:
[Paste CLAUDE.md content]

Based on the existing structure, what's the best way to implement this?
```

### 🐛 Debugging
```
I'm getting this error: [error details]

Here's my codebase context:
[Paste CLAUDE.md content]

Can you help identify the issue and suggest a fix?
```

### 🔍 Code Review
```
I've implemented [changes description]. Here's the updated context:
[Paste CLAUDE.md content]

Please review for quality, best practices, and potential issues.
```

## 📋 Essential Commands

| Command | Purpose | Example |
|---------|---------|---------|
| `codecontext init` | Initialize project | `codecontext init` |
| `codecontext generate` | Create context map | `codecontext generate` |
| `codecontext update` | Update existing context | `codecontext update` |
| `codecontext watch` | Auto-update on changes | `codecontext watch` |
| `codecontext compact` | Reduce context size | `codecontext compact --level balanced` |

## ⚙️ Key Configuration

```yaml
# .codecontext/config.yaml
analysis:
  include_patterns:
    - "src/**"           # Include source files
    - "components/**"    # Include components
  exclude_patterns:
    - "**/*.test.*"      # Exclude tests
    - "node_modules/**"  # Exclude dependencies
    - "dist/**"          # Exclude build output

output:
  max_file_size: 1048576  # 1MB limit
  include_stats: true     # Include analysis stats
```

## 🎯 Workflow Tips

### 📈 Development Flow
1. **Start**: `codecontext generate` → share with Claude for planning
2. **Build**: Code features → `codecontext update` → get help from Claude  
3. **Review**: `codecontext compact` → focused review with Claude
4. **Iterate**: Repeat step 2-3 until complete

### 💡 Pro Tips
- **Use watch mode** during active development: `codecontext watch`
- **Compact for large projects**: `codecontext compact --level balanced`
- **Focus on specific areas**: `codecontext generate src/components/`
- **Include relevant context only**: Configure include/exclude patterns

### ⚡ Speed Optimizations
- Enable caching: `codecontext generate --cache`
- Exclude test files for general development
- Use incremental updates: `codecontext update --incremental`
- Set memory limits in config for large projects

## 🔗 Links

- **Full Guide**: [docs/CLAUDE_INTEGRATION.md](docs/CLAUDE_INTEGRATION.md)
- **Example Workflow**: [examples/CLAUDE_WORKFLOW.md](examples/CLAUDE_WORKFLOW.md)
- **GitHub**: https://github.com/nmakod/codecontext
- **Releases**: https://github.com/nmakod/codecontext/releases

---

**Happy coding with Claude! 🤖✨**