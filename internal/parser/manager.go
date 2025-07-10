package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"
	javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	"github.com/nuthan-ms/codecontext/pkg/types"
)

// Manager implements the parser manager interface
type Manager struct {
	parsers   map[string]*sitter.Parser
	languages map[string]*sitter.Language
	cache     *ASTCache
	mu        sync.RWMutex
}

// NewManager creates a new parser manager
func NewManager() *Manager {
	m := &Manager{
		parsers:   make(map[string]*sitter.Parser),
		languages: make(map[string]*sitter.Language),
		cache:     NewASTCache(),
	}
	
	// Initialize supported languages
	m.initLanguages()
	
	return m
}

// initLanguages initializes the supported languages with real Tree-sitter grammars
func (m *Manager) initLanguages() {
	// JavaScript grammar using official bindings
	jsLang := sitter.NewLanguage(javascript.Language())
	m.languages["javascript"] = jsLang
	
	jsParser := sitter.NewParser()
	jsParser.SetLanguage(jsLang)
	m.parsers["javascript"] = jsParser
	
	// TypeScript - use JavaScript grammar as fallback until TypeScript bindings are fixed
	// Both JS and TS have similar syntax and this provides basic parsing capability
	tsLang := sitter.NewLanguage(javascript.Language())
	m.languages["typescript"] = tsLang
	
	tsParser := sitter.NewParser()
	tsParser.SetLanguage(tsLang)
	m.parsers["typescript"] = tsParser
	
	// For JSON and YAML, we'll use basic parsers for now
	// These can be extended with proper grammars later
	basicParser := sitter.NewParser()
	m.languages["json"] = nil
	m.languages["yaml"] = nil
	m.parsers["json"] = basicParser
	m.parsers["yaml"] = basicParser
}

// ParseFile parses a file and returns an AST
func (m *Manager) ParseFile(filePath string, language types.Language) (*types.AST, error) {
	// Read file content from disk
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	
	return m.parseContent(string(content), language, filePath)
}

// ParseFileVersioned parses a file with version information
func (m *Manager) ParseFileVersioned(filePath, content, version string) (*types.VersionedAST, error) {
	// Detect language
	lang := m.detectLanguage(filePath)
	if lang == nil {
		return nil, fmt.Errorf("unsupported file type: %s", filePath)
	}
	
	// Parse the content
	ast, err := m.parseContent(content, *lang, filePath)
	if err != nil {
		return nil, err
	}
	
	ast.Version = version
	
	versionedAST := &types.VersionedAST{
		AST:       ast,
		Version:   version,
		Hash:      calculateHash(content),
		Timestamp: time.Now(),
	}
	
	return versionedAST, nil
}

// ExtractSymbols extracts symbols from an AST
func (m *Manager) ExtractSymbols(ast *types.AST) ([]*types.Symbol, error) {
	if ast.Root == nil {
		return nil, fmt.Errorf("AST root is nil")
	}
	
	var symbols []*types.Symbol
	m.extractSymbolsRecursive(ast.Root, ast.FilePath, ast.Language, &symbols)
	
	return symbols, nil
}

// ExtractImports extracts imports from an AST
func (m *Manager) ExtractImports(ast *types.AST) ([]*types.Import, error) {
	if ast.Root == nil {
		return nil, fmt.Errorf("AST root is nil")
	}
	
	var imports []*types.Import
	m.extractImportsRecursive(ast.Root, &imports)
	
	return imports, nil
}

// GetSupportedLanguages returns the list of supported languages
func (m *Manager) GetSupportedLanguages() []types.Language {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var languages []types.Language
	for name, _ := range m.languages {
		lang := types.Language{
			Name:       name,
			Extensions: m.getExtensionsForLanguage(name),
			Parser:     fmt.Sprintf("tree-sitter-%s", name),
			Enabled:    true,
		}
		languages = append(languages, lang)
	}
	
	return languages
}

// ClassifyFile classifies a file based on its path and content
func (m *Manager) ClassifyFile(filePath string) (*types.FileClassification, error) {
	ext := filepath.Ext(filePath)
	baseName := filepath.Base(filePath)
	
	// Detect language
	lang := m.detectLanguage(filePath)
	if lang == nil {
		return nil, fmt.Errorf("unsupported file type: %s", filePath)
	}
	
	// Determine file type
	fileType := "source"
	isTest := false
	
	if strings.Contains(baseName, "test") || strings.Contains(baseName, "spec") {
		fileType = "test"
		isTest = true
	} else if strings.Contains(baseName, "config") || ext == ".json" || ext == ".yaml" || ext == ".yml" {
		fileType = "config"
	}
	
	// Check if generated
	isGenerated := strings.Contains(baseName, "generated") || 
		strings.Contains(baseName, "auto") ||
		strings.HasSuffix(baseName, ".gen.ts") ||
		strings.HasSuffix(baseName, ".generated.ts")
	
	return &types.FileClassification{
		Language:    *lang,
		FileType:    fileType,
		IsGenerated: isGenerated,
		IsTest:      isTest,
		Confidence:  0.95,
	}, nil
}

// GetASTCache returns the AST cache
func (m *Manager) GetASTCache() types.ASTCache {
	return m.cache
}

// Helper methods

func (m *Manager) detectLanguage(filePath string) *types.Language {
	ext := filepath.Ext(filePath)
	
	switch ext {
	case ".ts", ".tsx":
		return &types.Language{
			Name:       "typescript",
			Extensions: []string{".ts", ".tsx"},
			Parser:     "tree-sitter-typescript",
			Enabled:    true,
		}
	case ".js", ".jsx":
		return &types.Language{
			Name:       "javascript",
			Extensions: []string{".js", ".jsx"},
			Parser:     "tree-sitter-javascript",
			Enabled:    true,
		}
	case ".json":
		return &types.Language{
			Name:       "json",
			Extensions: []string{".json"},
			Parser:     "tree-sitter-json",
			Enabled:    true,
		}
	case ".yaml", ".yml":
		return &types.Language{
			Name:       "yaml",
			Extensions: []string{".yaml", ".yml"},
			Parser:     "tree-sitter-yaml",
			Enabled:    true,
		}
	default:
		return nil
	}
}

func (m *Manager) parseContent(content string, language types.Language, filePath ...string) (*types.AST, error) {
	m.mu.RLock()
	parser, exists := m.parsers[language.Name]
	treeSitterLang := m.languages[language.Name]
	m.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("unsupported language: %s", language.Name)
	}
	
	// For languages without real grammars (JSON, YAML), create mock AST
	if treeSitterLang == nil {
		ast := &types.AST{
			Language:       language.Name,
			Content:        content,
			Hash:           calculateHash(content),
			Version:        "1.0",
			ParsedAt:       time.Now(),
			TreeSitterTree: nil,
		}
		
		if len(filePath) > 0 {
			ast.FilePath = filePath[0]
		}
		
		// Create a basic root node for unsupported languages
		ast.Root = &types.ASTNode{
			Id:   "root",
			Type: "document",
			Location: types.FileLocation{
				FilePath: ast.FilePath,
				Line:     1,
				Column:   1,
			},
			Value: content,
		}
		
		return ast, nil
	}
	
	// Parse using real Tree-sitter grammar
	tree := parser.Parse([]byte(content), nil)
	defer tree.Close()
	
	// Create AST with real Tree-sitter data
	ast := &types.AST{
		Language:       language.Name,
		Content:        content,
		Hash:           calculateHash(content),
		Version:        "1.0",
		ParsedAt:       time.Now(),
		TreeSitterTree: tree,
	}
	
	if len(filePath) > 0 {
		ast.FilePath = filePath[0]
	}
	
	// Convert Tree-sitter root node to our AST format
	if tree.RootNode() != nil {
		ast.Root = m.convertTreeSitterNode(tree.RootNode(), content)
		if ast.Root != nil {
			ast.Root.Location.FilePath = ast.FilePath
		}
	}
	
	return ast, nil
}

// convertTreeSitterNode converts a tree-sitter node to our AST node format
func (m *Manager) convertTreeSitterNode(node *sitter.Node, content string) *types.ASTNode {
	if node == nil {
		return nil
	}
	
	startPos := node.StartPosition()
	endPos := node.EndPosition()
	
	astNode := &types.ASTNode{
		Id:   fmt.Sprintf("node-%d-%d", node.StartByte(), node.EndByte()),
		Type: node.Kind(),
		Location: types.FileLocation{
			Line:      int(startPos.Row) + 1,
			Column:    int(startPos.Column) + 1,
			EndLine:   int(endPos.Row) + 1,
			EndColumn: int(endPos.Column) + 1,
		},
	}
	
	// Extract text content for the node
	if int(node.StartByte()) < len(content) && int(node.EndByte()) <= len(content) {
		astNode.Value = content[node.StartByte():node.EndByte()]
	}
	
	// Convert children (limit depth to prevent excessive memory usage)
	childCount := int(node.ChildCount())
	if childCount > 0 && childCount < 1000 { // Reasonable limit
		for i := 0; i < childCount; i++ {
			child := node.Child(uint(i))
			if child != nil {
				if childAST := m.convertTreeSitterNode(child, content); childAST != nil {
					astNode.Children = append(astNode.Children, childAST)
				}
			}
		}
	}
	
	return astNode
}

func (m *Manager) extractSymbolsRecursive(node *types.ASTNode, filePath, language string, symbols *[]*types.Symbol) {
	if node == nil {
		return
	}
	
	// Check if this node represents a symbol
	if symbol := m.nodeToSymbol(node, filePath, language); symbol != nil {
		*symbols = append(*symbols, symbol)
	}
	
	// Recursively extract from children
	for _, child := range node.Children {
		m.extractSymbolsRecursive(child, filePath, language, symbols)
	}
}

func (m *Manager) nodeToSymbol(node *types.ASTNode, filePath, language string) *types.Symbol {
	// Enhanced symbol extraction using real Tree-sitter node types
	switch node.Type {
	case "function_declaration", "function", "function_expression", "arrow_function":
		return &types.Symbol{
			Id:       types.SymbolId(fmt.Sprintf("func-%s-%d", filePath, node.Location.Line)),
			Name:     m.extractSymbolName(node),
			Type:     types.SymbolTypeFunction,
			Location: node.Location,
			Signature: m.extractFunctionSignature(node),
			Language: language,
			Hash:     calculateHash(node.Value),
			LastModified: time.Now(),
		}
	case "class_declaration", "class", "class_expression":
		return &types.Symbol{
			Id:       types.SymbolId(fmt.Sprintf("class-%s-%d", filePath, node.Location.Line)),
			Name:     m.extractSymbolName(node),
			Type:     types.SymbolTypeClass,
			Location: node.Location,
			Language: language,
			Hash:     calculateHash(node.Value),
			LastModified: time.Now(),
		}
	case "interface_declaration", "interface":
		return &types.Symbol{
			Id:       types.SymbolId(fmt.Sprintf("interface-%s-%d", filePath, node.Location.Line)),
			Name:     m.extractSymbolName(node),
			Type:     types.SymbolTypeInterface,
			Location: node.Location,
			Language: language,
			Hash:     calculateHash(node.Value),
			LastModified: time.Now(),
		}
	case "type_alias_declaration", "type_declaration":
		return &types.Symbol{
			Id:       types.SymbolId(fmt.Sprintf("type-%s-%d", filePath, node.Location.Line)),
			Name:     m.extractSymbolName(node),
			Type:     types.SymbolTypeType,
			Location: node.Location,
			Language: language,
			Hash:     calculateHash(node.Value),
			LastModified: time.Now(),
		}
	case "variable_declaration", "lexical_declaration":
		return &types.Symbol{
			Id:       types.SymbolId(fmt.Sprintf("var-%s-%d", filePath, node.Location.Line)),
			Name:     m.extractSymbolName(node),
			Type:     types.SymbolTypeVariable,
			Location: node.Location,
			Language: language,
			Hash:     calculateHash(node.Value),
			LastModified: time.Now(),
		}
	case "method_definition", "method_signature":
		return &types.Symbol{
			Id:       types.SymbolId(fmt.Sprintf("method-%s-%d", filePath, node.Location.Line)),
			Name:     m.extractSymbolName(node),
			Type:     types.SymbolTypeMethod,
			Location: node.Location,
			Signature: m.extractFunctionSignature(node),
			Language: language,
			Hash:     calculateHash(node.Value),
			LastModified: time.Now(),
		}
	case "import_statement", "import_declaration":
		return &types.Symbol{
			Id:       types.SymbolId(fmt.Sprintf("import-%s-%d", filePath, node.Location.Line)),
			Name:     m.extractImportName(node),
			Type:     types.SymbolTypeImport,
			Location: node.Location,
			Language: language,
			Hash:     calculateHash(node.Value),
			LastModified: time.Now(),
		}
	case "export_statement", "export_declaration":
		return &types.Symbol{
			Id:       types.SymbolId(fmt.Sprintf("export-%s-%d", filePath, node.Location.Line)),
			Name:     m.extractSymbolName(node),
			Type:     types.SymbolTypeNamespace, // Using namespace for exports
			Location: node.Location,
			Language: language,
			Hash:     calculateHash(node.Value),
			LastModified: time.Now(),
		}
	default:
		return nil
	}
}

func (m *Manager) extractImportsRecursive(node *types.ASTNode, imports *[]*types.Import) {
	if node == nil {
		return
	}
	
	// Check if this node represents an import
	if imp := m.nodeToImport(node); imp != nil {
		*imports = append(*imports, imp)
	}
	
	// Recursively extract from children
	for _, child := range node.Children {
		m.extractImportsRecursive(child, imports)
	}
}

func (m *Manager) nodeToImport(node *types.ASTNode) *types.Import {
	if node.Type != "import_statement" && node.Type != "import_declaration" {
		return nil
	}
	
	imp := &types.Import{
		Location: node.Location,
	}
	
	// Extract import path and specifiers from children
	for _, child := range node.Children {
		switch child.Type {
		case "string", "string_literal":
			imp.Path = strings.Trim(child.Value, `"'`)
		case "import_specifier":
			if name := m.extractSymbolName(child); name != "unknown" {
				imp.Specifiers = append(imp.Specifiers, name)
			}
		case "namespace_import":
			if name := m.extractSymbolName(child); name != "unknown" {
				imp.Alias = name
			}
		case "identifier":
			// Default import
			imp.IsDefault = true
			if name := strings.TrimSpace(child.Value); name != "" {
				imp.Specifiers = append(imp.Specifiers, name)
			}
		}
	}
	
	return imp
}

func (m *Manager) getExtensionsForLanguage(name string) []string {
	switch name {
	case "typescript":
		return []string{".ts", ".tsx"}
	case "javascript":
		return []string{".js", ".jsx"}
	default:
		return []string{}
	}
}

// Helper functions

func calculateHash(content string) string {
	// Simple hash implementation - in production, use crypto/sha256
	return fmt.Sprintf("hash-%d", len(content))
}

// extractSymbolName extracts the name from a symbol node
func (m *Manager) extractSymbolName(node *types.ASTNode) string {
	if node == nil {
		return "unknown"
	}
	
	// Look for identifier children that represent the symbol name
	for _, child := range node.Children {
		if child.Type == "identifier" || child.Type == "type_identifier" {
			return strings.TrimSpace(child.Value)
		}
		
		// For some nodes, the name might be nested deeper
		if child.Type == "property_identifier" || child.Type == "name" {
			return strings.TrimSpace(child.Value)
		}
	}
	
	// Fallback: extract from node value using heuristics
	value := strings.TrimSpace(node.Value)
	if value != "" {
		// Try to extract first word after keywords
		lines := strings.Split(value, "\n")
		if len(lines) > 0 {
			firstLine := strings.TrimSpace(lines[0])
			words := strings.Fields(firstLine)
			
			// Look for name after common keywords
			for i, word := range words {
				if word == "function" || word == "class" || word == "interface" || 
				   word == "type" || word == "const" || word == "let" || word == "var" {
					if i+1 < len(words) {
						name := strings.TrimSuffix(words[i+1], "(")
						name = strings.TrimSuffix(name, "{")
						return name
					}
				}
			}
			
			// If no keyword found, return first identifier-like word
			for _, word := range words {
				if isValidIdentifier(word) {
					return word
				}
			}
		}
	}
	
	return "unknown"
}

// extractFunctionSignature extracts function signature information
func (m *Manager) extractFunctionSignature(node *types.ASTNode) string {
	if node == nil {
		return ""
	}
	
	// Look for parameter list and return type
	for _, child := range node.Children {
		if child.Type == "formal_parameters" || child.Type == "parameters" {
			return strings.TrimSpace(child.Value)
		}
	}
	
	// Fallback: extract first line of the node
	value := strings.TrimSpace(node.Value)
	lines := strings.Split(value, "\n")
	if len(lines) > 0 {
		signature := strings.TrimSpace(lines[0])
		// Remove opening brace if present
		if idx := strings.Index(signature, "{"); idx != -1 {
			signature = strings.TrimSpace(signature[:idx])
		}
		return signature
	}
	
	return ""
}

// extractImportName extracts name from import nodes
func (m *Manager) extractImportName(node *types.ASTNode) string {
	if node == nil {
		return "unknown"
	}
	
	// Look for import specifiers
	for _, child := range node.Children {
		if child.Type == "import_specifier" || child.Type == "namespace_import" {
			if name := m.extractSymbolName(child); name != "unknown" {
				return name
			}
		}
		if child.Type == "string" || child.Type == "string_literal" {
			// This is likely the import path
			path := strings.Trim(child.Value, `"'`)
			// Extract module name from path
			if lastSlash := strings.LastIndex(path, "/"); lastSlash != -1 {
				return path[lastSlash+1:]
			}
			return path
		}
	}
	
	return "unknown"
}

// isValidIdentifier checks if a string looks like a valid identifier
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	
	// Remove common symbols
	s = strings.TrimSuffix(s, "(")
	s = strings.TrimSuffix(s, "{")
	s = strings.TrimSuffix(s, ":")
	s = strings.TrimSuffix(s, ";")
	
	// Check if it looks like an identifier
	if len(s) == 0 {
		return false
	}
	
	// Simple heuristic: starts with letter or underscore, contains only alphanumeric and underscore
	first := s[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_' || first == '$') {
		return false
	}
	
	for _, r := range s[1:] {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '$') {
			return false
		}
	}
	
	return true
}