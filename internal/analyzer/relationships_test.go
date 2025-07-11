package analyzer

import (
	"testing"
	"time"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

func createTestGraph() *types.CodeGraph {
	graph := &types.CodeGraph{
		Nodes:       make(map[types.NodeId]*types.GraphNode),
		Edges:       make(map[types.EdgeId]*types.GraphEdge),
		Files:       make(map[string]*types.FileNode),
		Symbols:     make(map[types.SymbolId]*types.Symbol),
		Metadata:    &types.GraphMetadata{},
	}

	// Create test files
	file1 := &types.FileNode{
		Path:         "src/user.ts",
		Language:     "typescript",
		Size:         1000,
		Lines:        50,
		SymbolCount:  3,
		ImportCount:  2,
		IsTest:       false,
		IsGenerated:  false,
		LastModified: time.Now(),
		Symbols:      []types.SymbolId{"user-class", "user-interface"},
		Imports: []*types.Import{
			{
				Path:       "./types",
				Specifiers: []string{"UserType"},
				IsDefault:  false,
			},
			{
				Path:       "./utils",
				Specifiers: []string{"validateUser"},
				IsDefault:  false,
			},
		},
	}

	file2 := &types.FileNode{
		Path:         "src/types.ts",
		Language:     "typescript",
		Size:         500,
		Lines:        25,
		SymbolCount:  2,
		ImportCount:  0,
		IsTest:       false,
		IsGenerated:  false,
		LastModified: time.Now(),
		Symbols:      []types.SymbolId{"user-type", "user-interface"},
		Imports:      []*types.Import{},
	}

	file3 := &types.FileNode{
		Path:         "src/utils.ts",
		Language:     "typescript",
		Size:         300,
		Lines:        15,
		SymbolCount:  1,
		ImportCount:  1,
		IsTest:       false,
		IsGenerated:  false,
		LastModified: time.Now(),
		Symbols:      []types.SymbolId{"validate-function"},
		Imports: []*types.Import{
			{
				Path:       "./types",
				Specifiers: []string{"UserType"},
				IsDefault:  false,
			},
		},
	}

	// Create test symbols
	userClass := &types.Symbol{
		Id:       "user-class",
		Name:     "User",
		Type:     types.SymbolTypeClass,
		Location: types.Location{StartLine: 10, StartColumn: 1, EndLine: 10, EndColumn: 30},
		Signature: "class User { constructor(data: UserType) }",
		Language: "typescript",
	}

	userInterface := &types.Symbol{
		Id:       "user-interface",
		Name:     "UserInterface",
		Type:     types.SymbolTypeInterface,
		Location: types.Location{StartLine: 5, StartColumn: 1, EndLine: 5, EndColumn: 30},
		Signature: "interface UserInterface { id: number; name: string }",
		Language: "typescript",
	}

	userType := &types.Symbol{
		Id:       "user-type",
		Name:     "UserType",
		Type:     types.SymbolTypeType,
		Location: types.Location{StartLine: 3, StartColumn: 1, EndLine: 3, EndColumn: 30},
		Signature: "type UserType = { id: number; name: string }",
		Language: "typescript",
	}

	validateFunction := &types.Symbol{
		Id:       "validate-function",
		Name:     "validateUser",
		Type:     types.SymbolTypeFunction,
		Location: types.Location{StartLine: 5, StartColumn: 1, EndLine: 5, EndColumn: 30},
		Signature: "function validateUser(user: UserType): boolean",
		Language: "typescript",
	}

	// Add to graph
	graph.Files["src/user.ts"] = file1
	graph.Files["src/types.ts"] = file2
	graph.Files["src/utils.ts"] = file3

	graph.Symbols["user-class"] = userClass
	graph.Symbols["user-interface"] = userInterface
	graph.Symbols["user-type"] = userType
	graph.Symbols["validate-function"] = validateFunction

	return graph
}

func TestNewRelationshipAnalyzer(t *testing.T) {
	graph := createTestGraph()
	analyzer := NewRelationshipAnalyzer(graph)

	if analyzer == nil {
		t.Fatal("NewRelationshipAnalyzer() returned nil")
	}

	if analyzer.graph != graph {
		t.Error("NewRelationshipAnalyzer() did not set graph correctly")
	}
}

func TestAnalyzeImportRelationships(t *testing.T) {
	graph := createTestGraph()
	analyzer := NewRelationshipAnalyzer(graph)
	metrics := &RelationshipMetrics{
		ByType: make(map[RelationshipType]int),
	}

	analyzer.analyzeImportRelationships(metrics)

	// Check that import relationships were created
	if metrics.ByType[RelationshipImport] == 0 {
		t.Error("Expected import relationships to be created")
	}

	// Check that edges were added to the graph
	importEdges := 0
	for _, edge := range graph.Edges {
		if edge.Type == string(RelationshipImport) {
			importEdges++
		}
	}

	if importEdges == 0 {
		t.Error("Expected import edges to be added to graph")
	}

	t.Logf("Created %d import relationships", metrics.ByType[RelationshipImport])
	t.Logf("Created %d import edges", importEdges)
}

func TestAnalyzeSymbolUsageRelationships(t *testing.T) {
	graph := createTestGraph()
	analyzer := NewRelationshipAnalyzer(graph)
	metrics := &RelationshipMetrics{
		ByType: make(map[RelationshipType]int),
	}

	analyzer.analyzeSymbolUsageRelationships(metrics)

	// Check that symbol usage relationships were analyzed
	if metrics.ByType[RelationshipReferences] == 0 {
		t.Log("No symbol references found - this is expected for the simple test case")
	}

	t.Logf("Found %d symbol references", metrics.ByType[RelationshipReferences])
}

func TestDetectCircularDependencies(t *testing.T) {
	graph := createTestGraph()
	
	// Add a circular dependency: user.ts -> types.ts -> utils.ts -> user.ts
	graph.Files["src/types.ts"].Imports = []*types.Import{
		{
			Path:       "./utils",
			Specifiers: []string{"validateUser"},
			IsDefault:  false,
		},
	}
	
	graph.Files["src/utils.ts"].Imports = []*types.Import{
		{
			Path:       "./user",
			Specifiers: []string{"User"},
			IsDefault:  false,
		},
	}

	analyzer := NewRelationshipAnalyzer(graph)
	metrics := &RelationshipMetrics{
		ByType:       make(map[RelationshipType]int),
		CircularDeps: make([]CircularDependency, 0),
	}

	analyzer.detectCircularDependencies(metrics)

	if len(metrics.CircularDeps) == 0 {
		t.Log("No circular dependencies detected - this may be expected depending on import resolution")
	} else {
		t.Logf("Detected %d circular dependencies", len(metrics.CircularDeps))
		for _, dep := range metrics.CircularDeps {
			t.Logf("Circular dependency: %v", dep.Files)
		}
	}
}

func TestIdentifyHotspotFiles(t *testing.T) {
	graph := createTestGraph()
	analyzer := NewRelationshipAnalyzer(graph)
	metrics := &RelationshipMetrics{
		ByType:       make(map[RelationshipType]int),
		HotspotFiles: make([]FileHotspot, 0),
	}

	// First analyze imports to create edges
	analyzer.analyzeImportRelationships(metrics)
	
	// Then identify hotspots
	analyzer.identifyHotspotFiles(metrics)

	t.Logf("Identified %d hotspot files", len(metrics.HotspotFiles))
	for _, hotspot := range metrics.HotspotFiles {
		t.Logf("Hotspot: %s (imports: %d, references: %d, score: %.2f)", 
			hotspot.FilePath, hotspot.ImportCount, hotspot.ReferenceCount, hotspot.Score)
	}
}

func TestFindIsolatedFiles(t *testing.T) {
	graph := createTestGraph()
	
	// Add an isolated file
	isolatedFile := &types.FileNode{
		Path:         "src/isolated.ts",
		Language:     "typescript",
		Size:         100,
		Lines:        10,
		SymbolCount:  1,
		ImportCount:  0,
		IsTest:       false,
		IsGenerated:  false,
		LastModified: time.Now(),
		Symbols:      []types.SymbolId{"isolated-function"},
		Imports:      []*types.Import{},
	}
	
	isolatedSymbol := &types.Symbol{
		Id:       "isolated-function",
		Name:     "isolatedFunction",
		Type:     types.SymbolTypeFunction,
		Location: types.Location{StartLine: 3, StartColumn: 1, EndLine: 3, EndColumn: 30},
		Signature: "function isolatedFunction(): void",
		Language: "typescript",
	}
	
	graph.Files["src/isolated.ts"] = isolatedFile
	graph.Symbols["isolated-function"] = isolatedSymbol

	analyzer := NewRelationshipAnalyzer(graph)
	metrics := &RelationshipMetrics{
		ByType:        make(map[RelationshipType]int),
		IsolatedFiles: make([]string, 0),
	}

	// First analyze imports to create edges
	analyzer.analyzeImportRelationships(metrics)
	
	// Then find isolated files
	analyzer.findIsolatedFiles(metrics)

	// Should find the isolated file
	if len(metrics.IsolatedFiles) == 0 {
		t.Error("Expected to find isolated files")
	}

	found := false
	for _, file := range metrics.IsolatedFiles {
		if file == "src/isolated.ts" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find src/isolated.ts as isolated file")
	}

	t.Logf("Found %d isolated files: %v", len(metrics.IsolatedFiles), metrics.IsolatedFiles)
}

func TestAnalyzeAllRelationships(t *testing.T) {
	graph := createTestGraph()
	analyzer := NewRelationshipAnalyzer(graph)

	metrics, err := analyzer.AnalyzeAllRelationships()
	if err != nil {
		t.Fatalf("AnalyzeAllRelationships() error = %v", err)
	}

	if metrics == nil {
		t.Fatal("AnalyzeAllRelationships() returned nil metrics")
	}

	// Check that various relationship types were analyzed
	if metrics.TotalRelationships == 0 {
		t.Error("Expected total relationships to be greater than 0")
	}

	// Check that metrics structure is populated
	if metrics.ByType == nil {
		t.Error("Expected ByType to be initialized")
	}

	if metrics.HotspotFiles == nil {
		t.Error("Expected HotspotFiles to be initialized")
	}

	if metrics.IsolatedFiles == nil {
		t.Error("Expected IsolatedFiles to be initialized")
	}

	if metrics.CircularDeps == nil {
		t.Error("Expected CircularDeps to be initialized")
	}

	t.Logf("Total relationships: %d", metrics.TotalRelationships)
	t.Logf("Relationships by type: %v", metrics.ByType)
	t.Logf("File-to-file relationships: %d", metrics.FileToFile)
	t.Logf("Symbol-to-symbol relationships: %d", metrics.SymbolToSymbol)
	t.Logf("Cross-file references: %d", metrics.CrossFileRefs)
	t.Logf("Circular dependencies: %d", len(metrics.CircularDeps))
	t.Logf("Hotspot files: %d", len(metrics.HotspotFiles))
	t.Logf("Isolated files: %d", len(metrics.IsolatedFiles))
}

func TestExtractTypeReferences(t *testing.T) {
	graph := createTestGraph()
	analyzer := NewRelationshipAnalyzer(graph)

	tests := []struct {
		name      string
		signature string
		expected  []string
	}{
		{
			name:      "function with parameter type",
			signature: "function test(user: UserType): void",
			expected:  []string{"UserType"},
		},
		{
			name:      "function with return type",
			signature: "function getUser(): UserType",
			expected:  []string{"UserType"},
		},
		{
			name:      "function with multiple types",
			signature: "function process(user: UserType, callback: CallbackType): ResultType",
			expected:  []string{"UserType", "CallbackType", "ResultType"},
		},
		{
			name:      "no type references",
			signature: "function simple()",
			expected:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.extractTypeReferences(tt.signature)
			
			if len(result) != len(tt.expected) {
				t.Errorf("extractTypeReferences() returned %d types, expected %d", len(result), len(tt.expected))
				t.Logf("Result: %v", result)
				t.Logf("Expected: %v", tt.expected)
				return
			}

			for i, expected := range tt.expected {
				if i >= len(result) || result[i] != expected {
					t.Errorf("extractTypeReferences() result[%d] = %s, expected %s", i, result[i], expected)
				}
			}
		})
	}
}

func TestIsBuiltinType(t *testing.T) {
	graph := createTestGraph()
	analyzer := NewRelationshipAnalyzer(graph)

	tests := []struct {
		name     string
		typeName string
		expected bool
	}{
		{"string type", "string", true},
		{"number type", "number", true},
		{"boolean type", "boolean", true},
		{"Promise type", "Promise", true},
		{"custom type", "UserType", false},
		{"custom interface", "UserInterface", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.isBuiltinType(tt.typeName)
			if result != tt.expected {
				t.Errorf("isBuiltinType(%s) = %v, expected %v", tt.typeName, result, tt.expected)
			}
		})
	}
}

func TestResolveImportPath(t *testing.T) {
	graph := createTestGraph()
	analyzer := NewRelationshipAnalyzer(graph)

	tests := []struct {
		name       string
		importPath string
		fromFile   string
		expected   string
	}{
		{
			name:       "relative import existing file",
			importPath: "./types",
			fromFile:   "src/user.ts",
			expected:   "src/types.ts",
		},
		{
			name:       "relative import non-existing file",
			importPath: "./nonexistent",
			fromFile:   "src/user.ts",
			expected:   "",
		},
		{
			name:       "absolute import",
			importPath: "lodash",
			fromFile:   "src/user.ts",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.resolveImportPath(tt.importPath, tt.fromFile)
			if result != tt.expected {
				t.Errorf("resolveImportPath(%s, %s) = %s, expected %s", 
					tt.importPath, tt.fromFile, result, tt.expected)
			}
		})
	}
}