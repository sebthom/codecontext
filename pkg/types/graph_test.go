package types

import (
	"testing"
	"time"
)

func TestSymbol(t *testing.T) {
	tests := []struct {
		name   string
		symbol Symbol
		valid  bool
	}{
		{
			name: "valid function symbol",
			symbol: Symbol{
				Id:   "test-func-1",
				Name: "testFunction",
				Type: SymbolTypeFunction,
				Location: Location{
					StartLine:   10,
					StartColumn: 5,
					EndLine:     10,
					EndColumn:   20,
				},
				Language: "typescript",
				Hash:     "abc123",
			},
			valid: true,
		},
		{
			name: "valid class symbol",
			symbol: Symbol{
				Id:   "test-class-1",
				Name: "TestClass",
				Type: SymbolTypeClass,
				Location: Location{
					StartLine:   1,
					StartColumn: 0,
					EndLine:     1,
					EndColumn:   15,
				},
				Language: "typescript",
				Hash:     "def456",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.symbol.Id == "" && tt.valid {
				t.Error("Valid symbol should have an ID")
			}
			if tt.symbol.Name == "" && tt.valid {
				t.Error("Valid symbol should have a name")
			}
			if tt.symbol.Type == "" && tt.valid {
				t.Error("Valid symbol should have a type")
			}
		})
	}
}

func TestGraphNode(t *testing.T) {
	symbol := Symbol{
		Id:   "test-func-1",
		Name: "testFunction",
		Type: SymbolTypeFunction,
		Location: Location{
			StartLine:   10,
			StartColumn: 5,
			EndLine:     10,
			EndColumn:   20,
		},
		Language: "typescript",
		Hash:     "abc123",
	}

	node := GraphNode{
		Id:              "node-1",
		Symbol:          &symbol,
		Importance:      0.85,
		Connections:     3,
		ChangeFrequency: 5,
		LastModified:    time.Now(),
		Tags:            []string{"api", "critical"},
	}

	if node.Id != "node-1" {
		t.Errorf("Expected node ID 'node-1', got %s", node.Id)
	}
	if node.Symbol.Name != "testFunction" {
		t.Errorf("Expected symbol name 'testFunction', got %s", node.Symbol.Name)
	}
	if node.Importance != 0.85 {
		t.Errorf("Expected importance 0.85, got %f", node.Importance)
	}
	if len(node.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(node.Tags))
	}
}

func TestCodeGraph(t *testing.T) {
	graph := CodeGraph{
		Nodes: make(map[NodeId]*GraphNode),
		Edges: make(map[EdgeId]*GraphEdge),
		Metadata: &GraphMetadata{
			ProjectName:    "test-project",
			ProjectPath:    "/test/path",
			TotalFiles:     10,
			TotalSymbols:   50,
			Languages:      map[string]int{"typescript": 5, "javascript": 5},
			GeneratedAt:    time.Now(),
			ProcessingTime: time.Millisecond * 100,
			TokenCount:     150000,
		},
		Version: GraphVersion{
			Major:       1,
			Minor:       0,
			Patch:       0,
			Timestamp:   time.Now(),
			ChangeCount: 0,
			Hash:        "version-hash",
		},
	}

	if graph.Metadata.ProjectName != "test-project" {
		t.Errorf("Expected project name 'test-project', got %s", graph.Metadata.ProjectName)
	}
	if graph.Metadata.TotalFiles != 10 {
		t.Errorf("Expected 10 files, got %d", graph.Metadata.TotalFiles)
	}
	if len(graph.Metadata.Languages) != 2 {
		t.Errorf("Expected 2 languages, got %d", len(graph.Metadata.Languages))
	}
	if graph.Version.Major != 1 {
		t.Errorf("Expected major version 1, got %d", graph.Version.Major)
	}
}

func TestFileLocation(t *testing.T) {
	location := FileLocation{
		FilePath:  "src/main.ts",
		Line:      10,
		Column:    5,
		EndLine:   12,
		EndColumn: 15,
	}

	if location.FilePath != "src/main.ts" {
		t.Errorf("Expected file path 'src/main.ts', got %s", location.FilePath)
	}
	if location.Line != 10 {
		t.Errorf("Expected line 10, got %d", location.Line)
	}
	if location.EndLine != 12 {
		t.Errorf("Expected end line 12, got %d", location.EndLine)
	}
}
