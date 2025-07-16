package git

import (
	"fmt"
	"math"
	"testing"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

func TestDefaultIntegrationConfig(t *testing.T) {
	config := DefaultIntegrationConfig()
	
	if config.WeightGitPatterns != 0.6 {
		t.Errorf("Expected WeightGitPatterns to be 0.6, got %f", config.WeightGitPatterns)
	}
	if config.WeightDependencies != 0.3 {
		t.Errorf("Expected WeightDependencies to be 0.3, got %f", config.WeightDependencies)
	}
	if config.WeightStructural != 0.1 {
		t.Errorf("Expected WeightStructural to be 0.1, got %f", config.WeightStructural)
	}
	if config.MinCombinedScore != 0.4 {
		t.Errorf("Expected MinCombinedScore to be 0.4, got %f", config.MinCombinedScore)
	}
	if config.MaxNeighborhoodSize != 15 {
		t.Errorf("Expected MaxNeighborhoodSize to be 15, got %d", config.MaxNeighborhoodSize)
	}
	if !config.IncludeWeakRelations {
		t.Errorf("Expected IncludeWeakRelations to be true")
	}
	if !config.PrioritizeRecentFiles {
		t.Errorf("Expected PrioritizeRecentFiles to be true")
	}
}

func TestNewGraphIntegration(t *testing.T) {
	analyzer := createMockSemanticAnalyzer()
	codeGraph := createMockCodeGraph()
	
	// Test with default config
	gi := NewGraphIntegration(analyzer, codeGraph, nil)
	if gi.semanticAnalyzer != analyzer {
		t.Errorf("Expected semantic analyzer to be set")
	}
	if gi.codeGraph != codeGraph {
		t.Errorf("Expected code graph to be set")
	}
	if gi.config == nil {
		t.Errorf("Expected config to be set to default")
	}
	
	// Test with custom config
	customConfig := &IntegrationConfig{
		WeightGitPatterns: 0.5,
		WeightDependencies: 0.4,
		WeightStructural: 0.1,
		MinCombinedScore: 0.3,
		MaxNeighborhoodSize: 20,
		IncludeWeakRelations: false,
		PrioritizeRecentFiles: false,
	}
	
	gi2 := NewGraphIntegration(analyzer, codeGraph, customConfig)
	if gi2.config.WeightGitPatterns != 0.5 {
		t.Errorf("Expected custom config to be used")
	}
}

func TestCalculateClusteringWeight(t *testing.T) {
	gi := createMockGraphIntegration()
	
	// Create mock enhanced neighborhoods
	n1 := &EnhancedNeighborhood{
		SemanticNeighborhood: &SemanticNeighborhood{
			Files: []string{"file1.go", "file2.go"},
		},
		DependencyConnections: []DependencyConnection{
			{SourceFile: "file1.go", TargetFile: "file2.go"},
		},
		StructuralSimilarity: []StructuralSimilarity{
			{File1: "file1.go", File2: "file2.go", SharedPatterns: []string{"pattern1"}},
		},
	}
	
	n2 := &EnhancedNeighborhood{
		SemanticNeighborhood: &SemanticNeighborhood{
			Files: []string{"file1.go", "file3.go"},
		},
		DependencyConnections: []DependencyConnection{
			{SourceFile: "file1.go", TargetFile: "file3.go"},
		},
		StructuralSimilarity: []StructuralSimilarity{
			{File1: "file1.go", File2: "file3.go", SharedPatterns: []string{"pattern1"}},
		},
	}
	
	weight := gi.calculateClusteringWeight(n1, n2)
	if weight <= 0 {
		t.Errorf("Expected positive clustering weight, got %f", weight)
	}
}

func TestCalculateGitSimilarity(t *testing.T) {
	gi := createMockGraphIntegration()
	
	// Test with overlapping files
	n1 := &EnhancedNeighborhood{
		SemanticNeighborhood: &SemanticNeighborhood{
			Files: []string{"file1.go", "file2.go", "file3.go"},
		},
	}
	
	n2 := &EnhancedNeighborhood{
		SemanticNeighborhood: &SemanticNeighborhood{
			Files: []string{"file2.go", "file3.go", "file4.go"},
		},
	}
	
	similarity := gi.calculateGitSimilarity(n1, n2)
	// Jaccard similarity: 2 overlapping files / 4 total unique files = 0.5
	if similarity != 0.5 {
		t.Errorf("Expected git similarity to be 0.5, got %f", similarity)
	}
	
	// Test with no overlapping files
	n3 := &EnhancedNeighborhood{
		SemanticNeighborhood: &SemanticNeighborhood{
			Files: []string{"file5.go", "file6.go"},
		},
	}
	
	similarity2 := gi.calculateGitSimilarity(n1, n3)
	if similarity2 != 0.0 {
		t.Errorf("Expected git similarity to be 0.0, got %f", similarity2)
	}
}

func TestCalculateDependencySimilarity(t *testing.T) {
	gi := createMockGraphIntegration()
	
	// Test with shared dependencies
	n1 := &EnhancedNeighborhood{
		DependencyConnections: []DependencyConnection{
			{SourceFile: "file1.go", TargetFile: "file2.go"},
			{SourceFile: "file2.go", TargetFile: "file3.go"},
		},
	}
	
	n2 := &EnhancedNeighborhood{
		DependencyConnections: []DependencyConnection{
			{SourceFile: "file1.go", TargetFile: "file2.go"},
			{SourceFile: "file3.go", TargetFile: "file4.go"},
		},
	}
	
	similarity := gi.calculateDependencySimilarity(n1, n2)
	// 1 shared dependency / 3 total dependencies = 0.33333 (Jaccard: intersection/union)
	expected := 1.0 / 3.0
	if math.Abs(similarity - expected) > 0.0001 {
		t.Errorf("Expected dependency similarity to be %f, got %f", expected, similarity)
	}
	
	// Test with no dependencies
	n3 := &EnhancedNeighborhood{
		DependencyConnections: []DependencyConnection{},
	}
	
	similarity2 := gi.calculateDependencySimilarity(n1, n3)
	if similarity2 != 0.0 {
		t.Errorf("Expected dependency similarity to be 0.0, got %f", similarity2)
	}
}

func TestCalculateStructuralSimilarityMethod(t *testing.T) {
	gi := createMockGraphIntegration()
	
	// Test with shared structural patterns
	n1 := &EnhancedNeighborhood{
		StructuralSimilarity: []StructuralSimilarity{
			{SharedPatterns: []string{"pattern1", "pattern2"}},
		},
	}
	
	n2 := &EnhancedNeighborhood{
		StructuralSimilarity: []StructuralSimilarity{
			{SharedPatterns: []string{"pattern1", "pattern3"}},
		},
	}
	
	similarity := gi.calculateStructuralSimilarityBetweenNeighborhoods(n1, n2)
	// 1 shared pattern / 3 total patterns = 0.33333 (Jaccard: intersection/union)
	expected := 1.0 / 3.0
	if math.Abs(similarity - expected) > 0.0001 {
		t.Errorf("Expected structural similarity to be %f, got %f", expected, similarity)
	}
	
	// Test with no patterns
	n3 := &EnhancedNeighborhood{
		StructuralSimilarity: []StructuralSimilarity{},
	}
	
	similarity2 := gi.calculateStructuralSimilarityBetweenNeighborhoods(n1, n3)
	if similarity2 != 0.0 {
		t.Errorf("Expected structural similarity to be 0.0, got %f", similarity2)
	}
}

func TestDetermineConnectionType(t *testing.T) {
	gi := createMockGraphIntegration()
	
	// Test dependency connection type
	n1 := &EnhancedNeighborhood{
		DependencyConnections: []DependencyConnection{
			{SourceFile: "file1.go", TargetFile: "file2.go"},
		},
	}
	
	n2 := &EnhancedNeighborhood{
		DependencyConnections: []DependencyConnection{
			{SourceFile: "file2.go", TargetFile: "file3.go"},
		},
	}
	
	connType := gi.determineConnectionType(n1, n2)
	if connType != "dependency" {
		t.Errorf("Expected connection type to be 'dependency', got %s", connType)
	}
	
	// Test structural connection type
	n3 := &EnhancedNeighborhood{
		StructuralSimilarity: []StructuralSimilarity{
			{SharedPatterns: []string{"pattern1"}},
		},
	}
	
	n4 := &EnhancedNeighborhood{
		StructuralSimilarity: []StructuralSimilarity{
			{SharedPatterns: []string{"pattern2"}},
		},
	}
	
	connType2 := gi.determineConnectionType(n3, n4)
	if connType2 != "structural" {
		t.Errorf("Expected connection type to be 'structural', got %s", connType2)
	}
	
	// Test git pattern connection type (default)
	n5 := &EnhancedNeighborhood{}
	n6 := &EnhancedNeighborhood{}
	
	connType3 := gi.determineConnectionType(n5, n6)
	if connType3 != "git_pattern" {
		t.Errorf("Expected connection type to be 'git_pattern', got %s", connType3)
	}
}

func TestDetermineOptimalClusters(t *testing.T) {
	gi := createMockGraphIntegration()
	
	// Test various input sizes
	tests := []struct {
		n        int
		expected int
	}{
		{1, 1},
		{2, 2},
		{4, 2},
		{8, 2},
		{16, 3},
		{32, 4},
		{100, 6},
		{1000, 10}, // max limit
	}
	
	for _, test := range tests {
		result := gi.determineOptimalClusters(test.n)
		if result != test.expected {
			t.Errorf("For n=%d, expected %d clusters, got %d", test.n, test.expected, result)
		}
	}
}

func TestClassifyRecommendationStrength(t *testing.T) {
	gi := createMockGraphIntegration()
	
	tests := []struct {
		score    float64
		expected string
	}{
		{0.9, "very_strong"},
		{0.8, "very_strong"},
		{0.7, "strong"},
		{0.6, "strong"},
		{0.5, "moderate"},
		{0.4, "moderate"},
		{0.3, "weak"},
		{0.1, "weak"},
	}
	
	for _, test := range tests {
		result := gi.classifyRecommendationStrength(test.score)
		if result != test.expected {
			t.Errorf("For score %f, expected %s, got %s", test.score, test.expected, result)
		}
	}
}

func TestGenerateClusterName(t *testing.T) {
	gi := createMockGraphIntegration()
	
	// Test with empty neighborhoods
	neighborhoods := []EnhancedNeighborhood{}
	name := gi.generateClusterName(neighborhoods)
	if name != "Empty Cluster" {
		t.Errorf("Expected 'Empty Cluster' for empty input, got %s", name)
	}
	
	// Test with neighborhoods
	neighborhoods = []EnhancedNeighborhood{
		{SemanticNeighborhood: &SemanticNeighborhood{Name: "Service Handler"}},
		{SemanticNeighborhood: &SemanticNeighborhood{Name: "Service Utils"}},
	}
	
	name = gi.generateClusterName(neighborhoods)
	if name != "Service Group" {
		t.Errorf("Expected 'Service Group', got %s", name)
	}
}

func TestGenerateRecommendationReason(t *testing.T) {
	gi := createMockGraphIntegration()
	
	neighborhoods := []EnhancedNeighborhood{}
	
	tests := []struct {
		strength float64
		expected string
	}{
		{0.9, "Very strong cluster with high cohesion and frequent co-changes"},
		{0.7, "Strong cluster with good cohesion and regular co-changes"},
		{0.5, "Moderate cluster with some cohesion and occasional co-changes"},
		{0.3, "Weak cluster with low cohesion but some related changes"},
	}
	
	for _, test := range tests {
		result := gi.generateRecommendationReason(neighborhoods, test.strength)
		if result != test.expected {
			t.Errorf("For strength %f, expected %s, got %s", test.strength, test.expected, result)
		}
	}
}

func TestDetermineOptimalTasks(t *testing.T) {
	gi := createMockGraphIntegration()
	
	// Test with test files
	neighborhoods := []EnhancedNeighborhood{
		{SemanticNeighborhood: &SemanticNeighborhood{Files: []string{"handler_test.go", "utils_test.go"}}},
	}
	
	tasks := gi.determineOptimalTasks(neighborhoods)
	if !containsString(tasks, "testing") || !containsString(tasks, "debugging") {
		t.Errorf("Expected testing and debugging tasks for test files, got %v", tasks)
	}
	
	// Test with config files
	neighborhoods = []EnhancedNeighborhood{
		{SemanticNeighborhood: &SemanticNeighborhood{Files: []string{"config.yaml", "settings.json"}}},
	}
	
	tasks = gi.determineOptimalTasks(neighborhoods)
	if !containsString(tasks, "configuration") || !containsString(tasks, "setup") {
		t.Errorf("Expected configuration and setup tasks for config files, got %v", tasks)
	}
	
	// Test with documentation files
	neighborhoods = []EnhancedNeighborhood{
		{SemanticNeighborhood: &SemanticNeighborhood{Files: []string{"README.md", "docs.md"}}},
	}
	
	tasks = gi.determineOptimalTasks(neighborhoods)
	if !containsString(tasks, "documentation") || !containsString(tasks, "maintenance") {
		t.Errorf("Expected documentation and maintenance tasks for doc files, got %v", tasks)
	}
	
	// Test with regular files (default)
	neighborhoods = []EnhancedNeighborhood{
		{SemanticNeighborhood: &SemanticNeighborhood{Files: []string{"handler.go", "utils.go"}}},
	}
	
	tasks = gi.determineOptimalTasks(neighborhoods)
	if !containsString(tasks, "development") || !containsString(tasks, "refactoring") {
		t.Errorf("Expected development and refactoring tasks for regular files, got %v", tasks)
	}
}

func TestIsCommonWord(t *testing.T) {
	gi := createMockGraphIntegration()
	
	// Test common words
	commonWords := []string{"the", "and", "is", "are", "was", "were", "have", "has", "had", "will", "would", "could", "should"}
	for _, word := range commonWords {
		if !gi.isCommonWord(word) {
			t.Errorf("Expected '%s' to be a common word", word)
		}
	}
	
	// Test non-common words
	nonCommonWords := []string{"service", "handler", "utility", "processor", "manager", "controller"}
	for _, word := range nonCommonWords {
		if gi.isCommonWord(word) {
			t.Errorf("Expected '%s' to not be a common word", word)
		}
	}
}

func TestBuildClusteringGraph(t *testing.T) {
	gi := createMockGraphIntegration()
	
	// Create mock enhanced neighborhoods
	neighborhoods := []EnhancedNeighborhood{
		{
			SemanticNeighborhood: &SemanticNeighborhood{
				Files: []string{"file1.go", "file2.go"},
			},
			CombinedScore: 0.8,
		},
		{
			SemanticNeighborhood: &SemanticNeighborhood{
				Files: []string{"file2.go", "file3.go"},
			},
			CombinedScore: 0.7,
		},
	}
	
	nodes, err := gi.buildClusteringGraph(neighborhoods)
	if err != nil {
		t.Errorf("Unexpected error building clustering graph: %v", err)
	}
	
	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}
	
	// Check that nodes have proper IDs
	for i, node := range nodes {
		expectedID := fmt.Sprintf("node_%d", i)
		if node.ID != expectedID {
			t.Errorf("Expected node ID %s, got %s", expectedID, node.ID)
		}
	}
}

func TestHierarchicalClustering(t *testing.T) {
	gi := createMockGraphIntegration()
	
	// Test with empty nodes
	nodes := []ClusterNode{}
	clusters, err := gi.hierarchicalClustering(nodes)
	if err != nil {
		t.Errorf("Unexpected error with empty nodes: %v", err)
	}
	if len(clusters) != 0 {
		t.Errorf("Expected 0 clusters for empty input, got %d", len(clusters))
	}
	
	// Test with single node
	nodes = []ClusterNode{
		{
			ID: "node_0",
			Neighborhood: &EnhancedNeighborhood{
				SemanticNeighborhood: &SemanticNeighborhood{
					Name: "Test Neighborhood",
				},
				CombinedScore: 0.8,
			},
		},
	}
	
	clusters, err = gi.hierarchicalClustering(nodes)
	if err != nil {
		t.Errorf("Unexpected error with single node: %v", err)
	}
	if len(clusters) != 1 {
		t.Errorf("Expected 1 cluster for single node, got %d", len(clusters))
	}
	if clusters[0].Size != 1 {
		t.Errorf("Expected cluster size 1, got %d", clusters[0].Size)
	}
}

func TestCalculateNodeDistance(t *testing.T) {
	gi := createMockGraphIntegration()
	
	// Create mock nodes
	node1 := ClusterNode{
		ID: "node_0",
		Neighborhood: &EnhancedNeighborhood{
			SemanticNeighborhood: &SemanticNeighborhood{
				CorrelationStrength: 0.8,
			},
			CombinedScore: 0.8,
		},
		Connections: []ClusterConnection{
			{TargetID: "node_1", Weight: 0.7},
		},
	}
	
	node2 := ClusterNode{
		ID: "node_1",
		Neighborhood: &EnhancedNeighborhood{
			SemanticNeighborhood: &SemanticNeighborhood{
				CorrelationStrength: 0.6,
			},
			CombinedScore: 0.6,
		},
	}
	
	// Test with direct connection
	distance := gi.calculateNodeDistance(node1, node2)
	expectedDistance := 1.0 - 0.7 // 1 - connection weight
	if math.Abs(distance - expectedDistance) > 0.0001 {
		t.Errorf("Expected distance %f, got %f", expectedDistance, distance)
	}
	
	// Test without direct connection
	node3 := ClusterNode{
		ID: "node_2",
		Neighborhood: &EnhancedNeighborhood{
			CombinedScore: 0.5,
		},
	}
	
	distance2 := gi.calculateNodeDistance(node1, node3)
	if distance2 <= 0 {
		t.Errorf("Expected positive distance for unconnected nodes, got %f", distance2)
	}
}

func TestCalculateIntraClusterMetrics(t *testing.T) {
	gi := createMockGraphIntegration()
	
	// Test with single node cluster
	cluster := Cluster{
		Size: 1,
		Nodes: []ClusterNode{
			{
				ID: "node_0",
				Neighborhood: &EnhancedNeighborhood{CombinedScore: 0.8},
			},
		},
	}
	
	metrics := gi.calculateIntraClusterMetrics(cluster, []ClusterNode{})
	// Single node cluster should have default metrics
	if metrics.AverageDistance != 0 || metrics.MinDistance != 0 || metrics.MaxDistance != 0 {
		t.Errorf("Single node cluster should have zero distances")
	}
	
	// Test with multi-node cluster
	cluster.Size = 2
	cluster.Nodes = append(cluster.Nodes, ClusterNode{
		ID: "node_1",
		Neighborhood: &EnhancedNeighborhood{CombinedScore: 0.6},
	})
	
	metrics = gi.calculateIntraClusterMetrics(cluster, cluster.Nodes)
	if metrics.AverageDistance <= 0 {
		t.Errorf("Multi-node cluster should have positive average distance")
	}
	if metrics.Cohesion <= 0 {
		t.Errorf("Multi-node cluster should have positive cohesion")
	}
}

func TestCalculateDensityScore(t *testing.T) {
	gi := createMockGraphIntegration()
	
	// Test with single node
	cluster := Cluster{
		Size: 1,
		Nodes: []ClusterNode{
			{ID: "node_0"},
		},
	}
	
	density := gi.calculateDensityScore(cluster)
	if density != 1.0 {
		t.Errorf("Single node cluster should have density 1.0, got %f", density)
	}
	
	// Test with connected nodes
	cluster.Size = 2
	cluster.Nodes = []ClusterNode{
		{
			ID: "node_0",
			Connections: []ClusterConnection{
				{TargetID: "node_1", Weight: 0.8},
			},
		},
		{
			ID: "node_1",
			Connections: []ClusterConnection{
				{TargetID: "node_0", Weight: 0.8},
			},
		},
	}
	
	density = gi.calculateDensityScore(cluster)
	if density != 1.0 {
		t.Errorf("Fully connected 2-node cluster should have density 1.0, got %f", density)
	}
}

func TestCalculateClusterStrength(t *testing.T) {
	gi := createMockGraphIntegration()
	
	// Test with empty cluster
	cluster := Cluster{
		Size:  0,
		Nodes: []ClusterNode{},
	}
	
	strength := gi.calculateClusterStrength(cluster)
	if strength != 0.0 {
		t.Errorf("Empty cluster should have strength 0.0, got %f", strength)
	}
	
	// Test with nodes
	cluster.Size = 2
	cluster.Nodes = []ClusterNode{
		{Neighborhood: &EnhancedNeighborhood{CombinedScore: 0.8}},
		{Neighborhood: &EnhancedNeighborhood{CombinedScore: 0.6}},
	}
	cluster.IntraMetrics = IntraClusterMetrics{
		Cohesion: 0.7,
	}
	
	strength = gi.calculateClusterStrength(cluster)
	expectedStrength := (0.7 + 0.7) / 2.0 // (average combined score + cohesion) / 2
	if strength != expectedStrength {
		t.Errorf("Expected cluster strength %f, got %f", expectedStrength, strength)
	}
}

func TestGenerateClusterDescription(t *testing.T) {
	gi := createMockGraphIntegration()
	
	neighborhoods := []EnhancedNeighborhood{
		{
			SemanticNeighborhood: &SemanticNeighborhood{
				Files: []string{"file1.go", "file2.go"},
			},
			CombinedScore: 0.8,
		},
		{
			SemanticNeighborhood: &SemanticNeighborhood{
				Files: []string{"file3.go", "file4.go", "file5.go"},
			},
			CombinedScore: 0.6,
		},
	}
	
	description := gi.generateClusterDescription(neighborhoods)
	expected := "Cluster of 2 neighborhoods containing 5 files with 0.70 average combined score"
	if description != expected {
		t.Errorf("Expected description '%s', got '%s'", expected, description)
	}
}

func TestParseIndex(t *testing.T) {
	gi := createMockGraphIntegration()
	
	tests := []struct {
		input    string
		expected int
	}{
		{"0", 0},
		{"1", 1},
		{"123", 123},
		{"abc", -1},
		{"12a", -1},
		{"", 0},
	}
	
	for _, test := range tests {
		result := gi.parseIndex(test.input)
		if result != test.expected {
			t.Errorf("For input '%s', expected %d, got %d", test.input, test.expected, result)
		}
	}
}

func TestFindNodeIndex(t *testing.T) {
	gi := createMockGraphIntegration()
	
	tests := []struct {
		nodeID   string
		expected int
	}{
		{"node_0", 0},
		{"node_1", 1},
		{"node_123", 123},
		{"invalid_id", -1},
		{"node_abc", -1},
		{"", -1},
	}
	
	for _, test := range tests {
		result := gi.findNodeIndex(test.nodeID)
		if result != test.expected {
			t.Errorf("For nodeID '%s', expected %d, got %d", test.nodeID, test.expected, result)
		}
	}
}

// Helper functions for testing

func createMockGraphIntegration() *GraphIntegration {
	return &GraphIntegration{
		semanticAnalyzer: createMockSemanticAnalyzer(),
		codeGraph:        createMockCodeGraph(),
		config:           DefaultIntegrationConfig(),
	}
}

func createMockSemanticAnalyzer() *SemanticAnalyzer {
	return &SemanticAnalyzer{
		gitAnalyzer: &GitAnalyzer{
			repoPath: "/test/repo",
			gitPath:  "git",
		},
		patternDetector: &PatternDetector{
			minSupport:    0.1,
			minConfidence: 0.6,
		},
		config: DefaultSemanticConfig(),
	}
}

func createMockCodeGraph() *types.CodeGraph {
	return &types.CodeGraph{
		Symbols: map[types.SymbolId]*types.Symbol{
			"test_func": {
				Id:   "test_func",
				Name: "TestFunction",
				Type: "function",
				Location: types.Location{
					StartLine: 10,
				},
			},
		},
		Files: map[string]*types.FileNode{
			"test.go": {
				Path: "test.go",
				Language: "go",
				Symbols: []types.SymbolId{"test_func"},
			},
		},
		Edges: map[types.EdgeId]*types.GraphEdge{
			"edge1": {
				Id:   "edge1",
				From: "node1",
				To:   "node2",
				Type: "imports",
			},
		},
	}
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}