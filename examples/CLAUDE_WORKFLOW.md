# 🚀 CodeContext + Claude Workflow Example

## Real-World Example: Building a User Authentication System

This example shows how to use CodeContext with Claude to build a complete user authentication system for a Node.js/TypeScript application.

## 📁 Project Setup

```bash
# 1. Create new project
mkdir auth-app && cd auth-app
npm init -y
npm install express typescript @types/express bcrypt jsonwebtoken
npm install -D @types/bcrypt @types/jsonwebtoken

# 2. Initialize CodeContext
codecontext init

# 3. Create basic project structure
mkdir -p src/{models,routes,middleware,utils,types}
```

## 🎯 Step 1: Initial Planning with Claude

**Generate initial context:**
```bash
codecontext generate --output planning-context.md
```

**Claude Conversation:**
```
I'm building a user authentication system for a Node.js/TypeScript API. 

Current project structure:
[Paste planning-context.md content]

I need to implement:
1. User registration with email/password
2. Login with JWT tokens
3. Password hashing with bcrypt
4. Protected routes middleware
5. User profile management

Based on this structure, what files should I create and what should be the overall architecture?
```

**Claude's Response (Example):**
```
Based on your project structure, here's a recommended architecture:

src/
├── types/
│   └── user.ts          # User interfaces and types
├── models/
│   └── user.model.ts    # User data model
├── utils/
│   ├── password.ts      # Password hashing utilities
│   └── jwt.ts          # JWT token utilities
├── middleware/
│   └── auth.ts         # Authentication middleware
├── routes/
│   └── auth.routes.ts  # Authentication endpoints
└── app.ts              # Main application setup

Let me help you implement each component...
```

## 🔧 Step 2: Implementation with Incremental Context

**Create the types first:**
```typescript
// src/types/user.ts
export interface User {
  id: string;
  email: string;
  password: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface UserRegistration {
  email: string;
  password: string;
}

export interface UserLogin {
  email: string;
  password: string;
}

export interface AuthResponse {
  user: Omit<User, 'password'>;
  token: string;
}
```

**Update context and continue with Claude:**
```bash
codecontext update
```

**Claude Conversation:**
```
I've created the user types. Here's the updated context:
[Paste updated CLAUDE.md]

Now help me implement the password utility functions in src/utils/password.ts.
```

**Continue iteratively for each component...**

## 📊 Step 3: Complete Implementation Context

After implementing all components, generate final context:

```bash
codecontext generate --output final-context.md
```

**Example Final Context Output:**
```markdown
# CodeContext Map

**Generated:** 2025-07-12T18:30:15+05:30  
**Version:** 2.0.0  
**Analysis Time:** 42ms  
**Status:** Real Tree-sitter Analysis

## 📊 Overview

- **Files Analyzed**: 8 files
- **Symbols Extracted**: 34 symbols  
- **Languages Detected**: 2 languages (TypeScript, JSON)
- **Import Relationships**: 12 file dependencies

## 📁 File Analysis

| File | Language | Lines | Symbols | Imports | Type |
|------|----------|-------|---------|---------|------|
| `src/types/user.ts` | typescript | 25 | 4 | 0 | types |
| `src/utils/password.ts` | typescript | 18 | 2 | 1 | utility |
| `src/utils/jwt.ts` | typescript | 22 | 3 | 1 | utility |
| `src/models/user.model.ts` | typescript | 35 | 5 | 2 | model |
| `src/middleware/auth.ts` | typescript | 28 | 2 | 3 | middleware |
| `src/routes/auth.routes.ts` | typescript | 45 | 4 | 4 | routes |
| `src/app.ts` | typescript | 30 | 2 | 3 | main |
| `package.json` | json | 20 | 0 | 0 | config |

## 🔍 Symbol Analysis

### Symbol Types
- 🔧 **function**: 12
- 📦 **interface**: 4
- 📁 **export**: 8
- 📥 **import**: 10

### Key Symbols
| Symbol | Type | File | Line | Signature |
|--------|------|------|------|----------|
| `User` | interface | `src/types/user.ts` | 1 | `interface User` |
| `hashPassword` | function | `src/utils/password.ts` | 5 | `hashPassword(password: string)` |
| `generateToken` | function | `src/utils/jwt.ts` | 8 | `generateToken(payload: object)` |
| `UserModel` | class | `src/models/user.model.ts` | 10 | `class UserModel` |
| `authMiddleware` | function | `src/middleware/auth.ts` | 12 | `authMiddleware(req, res, next)` |
| `registerUser` | function | `src/routes/auth.routes.ts` | 15 | `registerUser(req, res)` |

## 🔗 Import Relationships

### Import Graph
- `src/app.ts` → [`express`, `src/routes/auth.routes`, `src/middleware/auth`]
- `src/routes/auth.routes.ts` → [`express`, `src/models/user.model`, `src/utils/password`, `src/utils/jwt`]
- `src/middleware/auth.ts` → [`express`, `src/utils/jwt`, `src/types/user`]
- `src/models/user.model.ts` → [`src/types/user`, `src/utils/password`]
- `src/utils/jwt.ts` → [`jsonwebtoken`]
- `src/utils/password.ts` → [`bcrypt`]

### File Dependencies
- **Core Dependencies**: express, bcrypt, jsonwebtoken
- **Internal Dependencies**: 6 cross-file imports
- **Circular Dependencies**: None detected ✅

## 🎯 Architecture Overview

### Layer Structure
1. **Types Layer**: `src/types/user.ts` - Interface definitions
2. **Utility Layer**: `src/utils/` - Password hashing, JWT operations  
3. **Model Layer**: `src/models/user.model.ts` - Data models
4. **Middleware Layer**: `src/middleware/auth.ts` - Request processing
5. **Route Layer**: `src/routes/auth.routes.ts` - API endpoints
6. **Application Layer**: `src/app.ts` - Server setup

### Key Components
- **Authentication Flow**: Register → Hash Password → Generate JWT → Protected Routes
- **Security Features**: bcrypt hashing, JWT tokens, middleware protection
- **Type Safety**: Full TypeScript interfaces and type checking
```

## 🧪 Step 4: Testing and Validation with Claude

**Generate test context:**
```bash
# Create test files
mkdir -p tests
touch tests/auth.test.ts tests/password.test.ts

codecontext generate --include tests/ --output testing-context.md
```

**Claude Conversation:**
```
Here's my complete authentication system:
[Paste final-context.md]

Can you help me:
1. Write comprehensive tests for all components
2. Identify any security vulnerabilities  
3. Suggest improvements for error handling
4. Review the overall architecture for best practices
```

## 🎯 Step 5: Optimization and Refactoring

**Use compaction for focused discussion:**
```bash
codecontext compact --level balanced --focus security --output security-review.md
```

**Claude Conversation:**
```
Here's a focused view of my authentication system for security review:
[Paste security-review.md]

Please review for:
1. Common security vulnerabilities (OWASP Top 10)
2. JWT implementation best practices  
3. Password security compliance
4. Input validation gaps
5. Rate limiting needs
```

## 📈 Results and Benefits

### Development Speed
- **Before CodeContext**: ~6 hours to implement basic auth system
- **With CodeContext + Claude**: ~2 hours for complete, production-ready system

### Code Quality Improvements
- **Architecture**: Claude suggested better separation of concerns
- **Security**: Identified 3 potential vulnerabilities before production
- **Type Safety**: Complete TypeScript implementation with proper interfaces
- **Testing**: Comprehensive test coverage guided by Claude

### Context Benefits
- **Focused Discussions**: Each conversation had full codebase context
- **Incremental Development**: Easy to track changes and get targeted help
- **Architecture Understanding**: Claude could suggest improvements based on full structure
- **No Repetition**: Didn't need to re-explain project structure in each conversation

## 🔄 Ongoing Development Workflow

### Daily Development
```bash
# Start development session
codecontext watch --output current-context.md

# Work on features, context updates automatically

# When stuck, share current context with Claude
```

### Feature Development
```bash
# Before new feature
codecontext generate --focus [feature-area] --output feature-context.md

# After implementation  
codecontext update --delta --output changes-context.md
```

### Code Reviews
```bash
# Generate context for changed files
git diff --name-only HEAD~1 | xargs codecontext generate --files --output review-context.md

# Share with Claude for review
```

## 💡 Key Takeaways

### Best Practices Learned
1. **Start with structure**: Get architecture right with Claude before coding
2. **Incremental context**: Update context as you build, don't wait until the end
3. **Focused discussions**: Use compaction and filtering for specific reviews
4. **Security first**: Always do security review with full context
5. **Test coverage**: Use context to ensure comprehensive testing

### Common Patterns
- **Planning Phase**: Full context + high-level architectural discussion
- **Implementation Phase**: Incremental updates + specific component help
- **Review Phase**: Focused context + security/quality review
- **Optimization Phase**: Compact context + performance/refactoring discussion

### Time Savings
- **Reduced Explanation Time**: Context eliminates need to explain project structure
- **Better Suggestions**: Claude gives more relevant advice with full context
- **Faster Iterations**: Quick context updates enable rapid development cycles
- **Quality Improvements**: Comprehensive reviews catch issues early

---

**This workflow pattern can be adapted for any project type - frontend React apps, backend APIs, full-stack applications, mobile apps, etc.**

**Try it yourself and see the productivity boost! 🚀**