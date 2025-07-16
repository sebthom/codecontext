package git

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

// GraphIntegration combines git patterns with dependency graph analysis
type GraphIntegration struct {
	semanticAnalyzer *SemanticAnalyzer
	codeGraph        *types.CodeGraph
	config           *IntegrationConfig
}

// IntegrationConfig holds configuration for graph integration
type IntegrationConfig struct {
	WeightGitPatterns     float64 `json:"weight_git_patterns"`     // Weight for git-based patterns
	WeightDependencies    float64 `json:"weight_dependencies"`     // Weight for dependency relationships
	WeightStructural      float64 `json:"weight_structural"`       // Weight for structural similarity
	MinCombinedScore      float64 `json:"min_combined_score"`      // Minimum score for neighborhood inclusion
	MaxNeighborhoodSize   int     `json:"max_neighborhood_size"`   // Maximum files per neighborhood
	IncludeWeakRelations  bool    `json:"include_weak_relations"`  // Include weak relationships
	PrioritizeRecentFiles bool    `json:"prioritize_recent_files"` // Prioritize recently changed files
}

// DefaultIntegrationConfig returns default configuration
func DefaultIntegrationConfig() *IntegrationConfig {
	return &IntegrationConfig{
		WeightGitPatterns:     0.6,
		WeightDependencies:    0.3,
		WeightStructural:      0.1,
		MinCombinedScore:      0.4,
		MaxNeighborhoodSize:   15,
		IncludeWeakRelations:  true,
		PrioritizeRecentFiles: true,
	}
}

// EnhancedNeighborhood represents a neighborhood with combined analysis
type EnhancedNeighborhood struct {
	*SemanticNeighborhood
	DependencyConnections  []DependencyConnection  `json:"dependency_connections"`
	StructuralSimilarity   []StructuralSimilarity  `json:"structural_similarity"`
	CombinedScore          float64                 `json:"combined_score"`
	ScoreBreakdown         ScoreBreakdown          `json:"score_breakdown"`
	RecommendationStrength string                  `json:"recommendation_strength"`
	UsagePatterns          []UsagePattern          `json:"usage_patterns"`
}

// DependencyConnection represents a dependency relationship
type DependencyConnection struct {
	SourceFile      string   `json:"source_file"`
	TargetFile      string   `json:"target_file"`
	ImportType      string   `json:"import_type"`      // "direct", "indirect", "circular"
	Strength        float64  `json:"strength"`         // 0.0 to 1.0
	ImportedSymbols []string `json:"imported_symbols"`
}

// StructuralSimilarity represents structural similarity between files
type StructuralSimilarity struct {
	File1           string   `json:"file1"`
	File2           string   `json:"file2"`
	SimilarityScore float64  `json:"similarity_score"`
	SharedSymbols   int      `json:"shared_symbols"`
	SharedPatterns  []string `json:"shared_patterns"`
}

// ScoreBreakdown shows how the combined score was calculated
type ScoreBreakdown struct {
	GitPatternScore     float64 `json:"git_pattern_score"`
	DependencyScore     float64 `json:"dependency_score"`
	StructuralScore     float64 `json:"structural_score"`
	WeightedTotal       float64 `json:"weighted_total"`
	NormalizationFactor float64 `json:"normalization_factor"`
}

// UsagePattern represents how files are used together
type UsagePattern struct {
	PatternType string    `json:"pattern_type"` // "always_together", "conditional", "sequential"
	Frequency   int       `json:"frequency"`
	LastSeen    time.Time `json:"last_seen"`
	Confidence  float64   `json:"confidence"`
	Description string    `json:"description"`
	Examples    []string  `json:"examples"`
}

// Clustering types for Week 3 implementation
type ClusterNode struct {
	ID           string               `json:"id"`
	Neighborhood *EnhancedNeighborhood `json:"neighborhood"`
	Connections  []ClusterConnection  `json:"connections"`
}

type ClusterConnection struct {
	TargetID string  `json:"target_id"`
	Weight   float64 `json:"weight"`
	Type     string  `json:"type"`
}

type Cluster struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Description     string                `json:"description"`
	Size            int                   `json:"size"`
	Nodes           []ClusterNode         `json:"nodes"`
	Strength        float64               `json:"strength"`
	IntraMetrics    IntraClusterMetrics   `json:"intra_metrics"`
	OptimalTasks    []string              `json:"optimal_tasks"`
	RecommendationReason string           `json:"recommendation_reason"`
}

type IntraClusterMetrics struct {
	AverageDistance float64 `json:"average_distance"`
	MinDistance     float64 `json:"min_distance"`
	MaxDistance     float64 `json:"max_distance"`
	Cohesion        float64 `json:"cohesion"`
	Density         float64 `json:"density"`
}

type ClusteredNeighborhood struct {
	Cluster        Cluster                `json:"cluster"`
	Neighborhoods  []EnhancedNeighborhood `json:"neighborhoods"`
	QualityMetrics ClusterQuality         `json:"quality_metrics"`
}

type ClusterQuality struct {
	SilhouetteScore    float64 `json:"silhouette_score"`
	DaviesBouldinIndex float64 `json:"davies_bouldin_index"`
	CalinskiHarabaszIndex float64 `json:"calinski_harabasz_index"`
}

// NewGraphIntegration creates a new graph integration instance
func NewGraphIntegration(semanticAnalyzer *SemanticAnalyzer, codeGraph *types.CodeGraph, config *IntegrationConfig) *GraphIntegration {
	if config == nil {
		config = DefaultIntegrationConfig()
	}

	return &GraphIntegration{
		semanticAnalyzer: semanticAnalyzer,
		codeGraph:        codeGraph,
		config:           config,
	}
}

// BuildEnhancedNeighborhoods combines git patterns with dependency analysis
func (gi *GraphIntegration) BuildEnhancedNeighborhoods() ([]EnhancedNeighborhood, error) {
	// Get semantic neighborhoods from git analysis
	result, err := gi.semanticAnalyzer.AnalyzeRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze repository: %w", err)
	}

	var enhancedNeighborhoods []EnhancedNeighborhood

	for _, neighborhood := range result.Neighborhoods {
		enhanced := EnhancedNeighborhood{
			SemanticNeighborhood: &neighborhood,
		}

		// Analyze dependency connections
		dependencies, err := gi.analyzeDependencyConnections(neighborhood.Files)
		if err != nil {
			log.Printf("[GraphIntegration] Warning: failed to analyze dependencies for %s: %v", neighborhood.Name, err)
		}
		enhanced.DependencyConnections = dependencies

		// Analyze structural similarity
		similarities, err := gi.analyzeStructuralSimilarity(neighborhood.Files)
		if err != nil {
			log.Printf("[GraphIntegration] Warning: failed to analyze structural similarity for %s: %v", neighborhood.Name, err)
		}
		enhanced.StructuralSimilarity = similarities

		// Calculate combined score
		enhanced.CombinedScore, enhanced.ScoreBreakdown = gi.calculateCombinedScore(&neighborhood, dependencies, similarities)

		// Classify recommendation strength
		enhanced.RecommendationStrength = gi.classifyRecommendationStrength(enhanced.CombinedScore)

		// Analyze usage patterns
		patterns, err := gi.analyzeUsagePatterns(&neighborhood)
		if err != nil {
			log.Printf("[GraphIntegration] Warning: failed to analyze usage patterns for %s: %v", neighborhood.Name, err)
		}
		enhanced.UsagePatterns = patterns

		// Only include neighborhoods that meet the minimum score threshold
		if enhanced.CombinedScore >= gi.config.MinCombinedScore {
			enhancedNeighborhoods = append(enhancedNeighborhoods, enhanced)
		}
	}

	return enhancedNeighborhoods, nil
}

// BuildClusteredNeighborhoods creates clustered neighborhoods using graph algorithms
func (gi *GraphIntegration) BuildClusteredNeighborhoods() ([]ClusteredNeighborhood, error) {
	// Get enhanced neighborhoods first
	enhancedNeighborhoods, err := gi.BuildEnhancedNeighborhoods()
	if err != nil {
		return nil, fmt.Errorf("failed to build enhanced neighborhoods: %w", err)
	}

	// Build clustering graph
	clusterGraph, err := gi.buildClusteringGraph(enhancedNeighborhoods)
	if err != nil {
		return nil, fmt.Errorf("failed to build clustering graph: %w", err)
	}

	// Apply clustering algorithms
	clusters, err := gi.applyClustering(clusterGraph, enhancedNeighborhoods)
	if err != nil {
		return nil, fmt.Errorf("failed to apply clustering: %w", err)
	}

	// Create clustered neighborhoods
	var clusteredNeighborhoods []ClusteredNeighborhood
	for _, cluster := range clusters {
		clustered, err := gi.createClusteredNeighborhood(cluster, enhancedNeighborhoods)
		if err != nil {
			log.Printf("[GraphIntegration] Warning: failed to create clustered neighborhood: %v", err)
			continue
		}
		clusteredNeighborhoods = append(clusteredNeighborhoods, clustered)
	}

	return clusteredNeighborhoods, nil
}

// analyzeDependencyConnections analyzes dependency connections between files
func (gi *GraphIntegration) analyzeDependencyConnections(files []string) ([]DependencyConnection, error) {
	if gi.codeGraph == nil {
		return []DependencyConnection{}, nil
	}

	var connections []DependencyConnection
	fileSet := make(map[string]bool)
	for _, file := range files {
		fileSet[file] = true
	}

	// Analyze edges in the code graph
	for _, edge := range gi.codeGraph.Edges {
		if edge.Type == "imports" || edge.Type == "calls" || edge.Type == "references" {
			// Get source and target file paths from nodes
			sourceNode := gi.codeGraph.Nodes[edge.From]
			targetNode := gi.codeGraph.Nodes[edge.To]
			
			if sourceNode != nil && targetNode != nil {
				sourceFile := sourceNode.FilePath
				targetFile := targetNode.FilePath

				// Check if both files are in our neighborhood
				if fileSet[sourceFile] && fileSet[targetFile] {
					connection := DependencyConnection{
						SourceFile:      sourceFile,
						TargetFile:      targetFile,
						ImportType:      "direct",
						Strength:        edge.Weight,
						ImportedSymbols: []string{string(edge.From)},
					}
					connections = append(connections, connection)
				}
			}
		}
	}

	// Find circular dependencies
	gi.markCircularDependencies(connections)

	return connections, nil
}

// analyzeStructuralSimilarity analyzes structural similarity between files
func (gi *GraphIntegration) analyzeStructuralSimilarity(files []string) ([]StructuralSimilarity, error) {
	if gi.codeGraph == nil {
		return []StructuralSimilarity{}, nil
	}

	var similarities []StructuralSimilarity

	// Compare each pair of files
	for i, file1 := range files {
		for j, file2 := range files {
			if i >= j {
				continue // Avoid duplicates and self-comparison
			}

			similarity, err := gi.calculateStructuralSimilarity(file1, file2)
			if err != nil {
				continue // Skip files we can't analyze
			}

			if similarity.SimilarityScore > 0.1 { // Only include meaningful similarities
				similarities = append(similarities, similarity)
			}
		}
	}

	return similarities, nil
}

// calculateStructuralSimilarity calculates structural similarity between two files
func (gi *GraphIntegration) calculateStructuralSimilarity(file1, file2 string) (StructuralSimilarity, error) {
	// Get symbols for both files
	symbols1 := gi.getFileSymbols(file1)
	symbols2 := gi.getFileSymbols(file2)

	if len(symbols1) == 0 || len(symbols2) == 0 {
		return StructuralSimilarity{}, fmt.Errorf("no symbols found for comparison")
	}

	// Calculate shared symbols
	sharedSymbols := gi.calculateSharedSymbols(symbols1, symbols2)
	
	// Calculate Jaccard similarity
	union := len(symbols1) + len(symbols2) - sharedSymbols
	similarityScore := float64(sharedSymbols) / float64(union)

	// Find shared patterns
	sharedPatterns := gi.findSharedPatterns(symbols1, symbols2)

	return StructuralSimilarity{
		File1:           file1,
		File2:           file2,
		SimilarityScore: similarityScore,
		SharedSymbols:   sharedSymbols,
		SharedPatterns:  sharedPatterns,
	}, nil
}

// analyzeUsagePatterns analyzes how files are used together
func (gi *GraphIntegration) analyzeUsagePatterns(neighborhood *SemanticNeighborhood) ([]UsagePattern, error) {
	var patterns []UsagePattern

	// Always together pattern
	if neighborhood.CorrelationStrength > 0.8 {
		patterns = append(patterns, UsagePattern{
			PatternType: "always_together",
			Frequency:   neighborhood.ChangeFrequency,
			LastSeen:    neighborhood.LastChanged,
			Confidence:  neighborhood.CorrelationStrength,
			Description: fmt.Sprintf("Files in %s are almost always modified together", neighborhood.Name),
			Examples:    []string{fmt.Sprintf("Last %d changes included all files", neighborhood.ChangeFrequency)},
		})
	}

	// Sequential pattern
	if neighborhood.ChangeFrequency > 10 {
		patterns = append(patterns, UsagePattern{
			PatternType: "sequential",
			Frequency:   neighborhood.ChangeFrequency,
			LastSeen:    neighborhood.LastChanged,
			Confidence:  0.7,
			Description: fmt.Sprintf("Files in %s are frequently modified in sequence", neighborhood.Name),
			Examples:    []string{"Sequential modifications detected in git history"},
		})
	}

	return patterns, nil
}

// calculateCombinedScore calculates the combined score for a neighborhood
func (gi *GraphIntegration) calculateCombinedScore(neighborhood *SemanticNeighborhood, dependencies []DependencyConnection, similarities []StructuralSimilarity) (float64, ScoreBreakdown) {
	// Git pattern score
	gitScore := neighborhood.CorrelationStrength

	// Dependency score
	depScore := gi.calculateDependencyScore(dependencies)

	// Structural score
	structScore := gi.calculateStructuralScore(similarities)

	// Calculate weighted total
	weightedTotal := (gitScore * gi.config.WeightGitPatterns) +
		(depScore * gi.config.WeightDependencies) +
		(structScore * gi.config.WeightStructural)

	// Normalize based on weights
	totalWeight := gi.config.WeightGitPatterns + gi.config.WeightDependencies + gi.config.WeightStructural
	normalizationFactor := 1.0
	if totalWeight > 0 {
		normalizationFactor = totalWeight
	}

	finalScore := weightedTotal / normalizationFactor

	breakdown := ScoreBreakdown{
		GitPatternScore:     gitScore,
		DependencyScore:     depScore,
		StructuralScore:     structScore,
		WeightedTotal:       weightedTotal,
		NormalizationFactor: normalizationFactor,
	}

	return finalScore, breakdown
}

// Week 3 Clustering Implementation

// buildClusteringGraph builds a graph for clustering neighborhoods
func (gi *GraphIntegration) buildClusteringGraph(neighborhoods []EnhancedNeighborhood) ([]ClusterNode, error) {
	var nodes []ClusterNode

	// Create nodes for each neighborhood
	for i, neighborhood := range neighborhoods {
		node := ClusterNode{
			ID:           fmt.Sprintf("node_%d", i),
			Neighborhood: &neighborhood,
			Connections:  []ClusterConnection{},
		}

		// Calculate connections to other neighborhoods
		for j, otherNeighborhood := range neighborhoods {
			if i != j {
				weight := gi.calculateClusteringWeight(&neighborhood, &otherNeighborhood)
				if weight > 0.1 { // Only include meaningful connections
					connectionType := gi.determineConnectionType(&neighborhood, &otherNeighborhood)
					connection := ClusterConnection{
						TargetID: fmt.Sprintf("node_%d", j),
						Weight:   weight,
						Type:     connectionType,
					}
					node.Connections = append(node.Connections, connection)
				}
			}
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// applyClustering applies hierarchical clustering algorithms
func (gi *GraphIntegration) applyClustering(nodes []ClusterNode, neighborhoods []EnhancedNeighborhood) ([]Cluster, error) {
	if len(nodes) == 0 {
		return []Cluster{}, nil
	}

	// Apply hierarchical clustering with Ward linkage
	clusters, err := gi.hierarchicalClustering(nodes)
	if err != nil {
		return nil, fmt.Errorf("hierarchical clustering failed: %w", err)
	}

	// Calculate metrics for each cluster
	for i := range clusters {
		clusters[i].IntraMetrics = gi.calculateIntraClusterMetrics(clusters[i], nodes)
		clusters[i].Strength = gi.calculateClusterStrength(clusters[i])
		clusters[i].OptimalTasks = gi.determineOptimalTasks(neighborhoods)
		clusters[i].RecommendationReason = gi.generateRecommendationReason(neighborhoods, clusters[i].Strength)
	}

	return clusters, nil
}

// hierarchicalClustering implements hierarchical clustering with Ward linkage
func (gi *GraphIntegration) hierarchicalClustering(nodes []ClusterNode) ([]Cluster, error) {
	if len(nodes) == 0 {
		return []Cluster{}, nil
	}

	if len(nodes) == 1 {
		var neighborhoods []EnhancedNeighborhood
		if nodes[0].Neighborhood != nil {
			neighborhoods = []EnhancedNeighborhood{*nodes[0].Neighborhood}
		}
		return []Cluster{{
			ID:   "cluster_0",
			Name: gi.generateClusterName(neighborhoods),
			Size: 1,
			Nodes: nodes,
		}}, nil
	}

	// Determine optimal number of clusters
	optimalClusters := gi.determineOptimalClusters(len(nodes))

	// Initialize each node as its own cluster
	clusters := make([]Cluster, len(nodes))
	for i, node := range nodes {
		clusters[i] = Cluster{
			ID:    fmt.Sprintf("cluster_%d", i),
			Size:  1,
			Nodes: []ClusterNode{node},
		}
	}

	// Merge clusters until we reach optimal number
	for len(clusters) > optimalClusters {
		// Find the two closest clusters
		minDistance := math.Inf(1)
		mergeI, mergeJ := -1, -1

		for i := 0; i < len(clusters); i++ {
			for j := i + 1; j < len(clusters); j++ {
				distance := gi.calculateClusterDistance(clusters[i], clusters[j])
				if distance < minDistance {
					minDistance = distance
					mergeI, mergeJ = i, j
				}
			}
		}

		if mergeI == -1 || mergeJ == -1 {
			break
		}

		// Merge clusters
		merged := gi.mergeClusters(clusters[mergeI], clusters[mergeJ])
		
		// Remove the merged clusters and add the new one
		newClusters := make([]Cluster, 0, len(clusters)-1)
		for i, cluster := range clusters {
			if i != mergeI && i != mergeJ {
				newClusters = append(newClusters, cluster)
			}
		}
		newClusters = append(newClusters, merged)
		clusters = newClusters
	}

	// Generate names and descriptions for final clusters
	for i := range clusters {
		var neighborhoodsInCluster []EnhancedNeighborhood
		for _, node := range clusters[i].Nodes {
			if node.Neighborhood != nil {
				neighborhoodsInCluster = append(neighborhoodsInCluster, *node.Neighborhood)
			}
		}
		clusters[i].Name = gi.generateClusterName(neighborhoodsInCluster)
		clusters[i].Description = gi.generateClusterDescription(neighborhoodsInCluster)
	}

	return clusters, nil
}

// createClusteredNeighborhood creates a clustered neighborhood with quality metrics
func (gi *GraphIntegration) createClusteredNeighborhood(cluster Cluster, allNeighborhoods []EnhancedNeighborhood) (ClusteredNeighborhood, error) {
	var neighborhoods []EnhancedNeighborhood
	for _, node := range cluster.Nodes {
		neighborhoods = append(neighborhoods, *node.Neighborhood)
	}

	// Calculate quality metrics
	qualityMetrics := gi.calculateClusterQuality(cluster, allNeighborhoods)

	return ClusteredNeighborhood{
		Cluster:        cluster,
		Neighborhoods:  neighborhoods,
		QualityMetrics: qualityMetrics,
	}, nil
}

// Helper functions for clustering

// calculateClusteringWeight calculates the weight between two neighborhoods for clustering
func (gi *GraphIntegration) calculateClusteringWeight(n1, n2 *EnhancedNeighborhood) float64 {
	gitSimilarity := gi.calculateGitSimilarity(n1, n2)
	depSimilarity := gi.calculateDependencySimilarity(n1, n2)
	structSimilarity := gi.calculateStructuralSimilarityBetweenNeighborhoods(n1, n2)

	// Weighted combination
	weight := (gitSimilarity * gi.config.WeightGitPatterns) +
		(depSimilarity * gi.config.WeightDependencies) +
		(structSimilarity * gi.config.WeightStructural)

	totalWeight := gi.config.WeightGitPatterns + gi.config.WeightDependencies + gi.config.WeightStructural
	if totalWeight > 0 {
		weight /= totalWeight
	}

	return weight
}

// calculateGitSimilarity calculates similarity based on git patterns
func (gi *GraphIntegration) calculateGitSimilarity(n1, n2 *EnhancedNeighborhood) float64 {
	files1 := make(map[string]bool)
	files2 := make(map[string]bool)

	for _, file := range n1.Files {
		files1[file] = true
	}
	for _, file := range n2.Files {
		files2[file] = true
	}

	// Calculate Jaccard similarity
	intersection := 0
	union := len(files1)

	for file := range files2 {
		if files1[file] {
			intersection++
		} else {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// calculateDependencySimilarity calculates similarity based on dependencies
func (gi *GraphIntegration) calculateDependencySimilarity(n1, n2 *EnhancedNeighborhood) float64 {
	deps1 := make(map[string]bool)
	deps2 := make(map[string]bool)

	for _, dep := range n1.DependencyConnections {
		deps1[dep.SourceFile+"->"+dep.TargetFile] = true
	}
	for _, dep := range n2.DependencyConnections {
		deps2[dep.SourceFile+"->"+dep.TargetFile] = true
	}

	if len(deps1) == 0 && len(deps2) == 0 {
		return 0.0
	}

	// Calculate Jaccard similarity
	intersection := 0
	union := len(deps1)

	for dep := range deps2 {
		if deps1[dep] {
			intersection++
		} else {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// calculateStructuralSimilarityBetweenNeighborhoods calculates structural similarity between neighborhoods
func (gi *GraphIntegration) calculateStructuralSimilarityBetweenNeighborhoods(n1, n2 *EnhancedNeighborhood) float64 {
	patterns1 := make(map[string]bool)
	patterns2 := make(map[string]bool)

	for _, sim := range n1.StructuralSimilarity {
		for _, pattern := range sim.SharedPatterns {
			patterns1[pattern] = true
		}
	}
	for _, sim := range n2.StructuralSimilarity {
		for _, pattern := range sim.SharedPatterns {
			patterns2[pattern] = true
		}
	}

	if len(patterns1) == 0 && len(patterns2) == 0 {
		return 0.0
	}

	// Calculate Jaccard similarity
	intersection := 0
	union := len(patterns1)

	for pattern := range patterns2 {
		if patterns1[pattern] {
			intersection++
		} else {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// determineConnectionType determines the type of connection between neighborhoods
func (gi *GraphIntegration) determineConnectionType(n1, n2 *EnhancedNeighborhood) string {
	if len(n1.DependencyConnections) > 0 || len(n2.DependencyConnections) > 0 {
		return "dependency"
	}
	if len(n1.StructuralSimilarity) > 0 || len(n2.StructuralSimilarity) > 0 {
		return "structural"
	}
	return "git_pattern"
}

// determineOptimalClusters determines optimal number of clusters using elbow method
func (gi *GraphIntegration) determineOptimalClusters(n int) int {
	if n <= 2 {
		return n
	}
	if n <= 4 {
		return 2
	}
	if n <= 8 {
		return 2
	}
	if n <= 16 {
		return 3
	}
	if n <= 32 {
		return 4
	}
	if n <= 64 {
		return 5
	}
	if n <= 100 {
		return 6
	}
	if n <= 128 {
		return 7
	}
	if n <= 256 {
		return 8
	}
	if n <= 512 {
		return 9
	}
	if n <= 1000 {
		return 10
	}
	return 10 // Maximum clusters
}

// calculateNodeDistance calculates distance between two cluster nodes
func (gi *GraphIntegration) calculateNodeDistance(node1, node2 ClusterNode) float64 {
	// Check if there's a direct connection
	for _, conn := range node1.Connections {
		if conn.TargetID == node2.ID {
			return 1.0 - conn.Weight // Convert similarity to distance
		}
	}

	// Calculate distance based on neighborhood properties
	if node1.Neighborhood == nil || node2.Neighborhood == nil {
		return 1.0 // Maximum distance for nil neighborhoods
	}
	
	combinedScore1 := node1.Neighborhood.CombinedScore
	combinedScore2 := node2.Neighborhood.CombinedScore

	// Distance based on score difference
	scoreDiff := math.Abs(combinedScore1 - combinedScore2)
	
	// Use correlation strength difference as additional factor if available
	corrDiff := 0.0
	if node1.Neighborhood.SemanticNeighborhood != nil && node2.Neighborhood.SemanticNeighborhood != nil {
		corrStrength1 := node1.Neighborhood.SemanticNeighborhood.CorrelationStrength
		corrStrength2 := node2.Neighborhood.SemanticNeighborhood.CorrelationStrength
		corrDiff = math.Abs(corrStrength1 - corrStrength2)
	}

	// Combine the differences (normalized)
	distance := (scoreDiff + corrDiff) / 2.0
	
	return distance
}

// calculateClusterDistance calculates distance between two clusters
func (gi *GraphIntegration) calculateClusterDistance(c1, c2 Cluster) float64 {
	if len(c1.Nodes) == 0 || len(c2.Nodes) == 0 {
		return math.Inf(1)
	}

	// Ward linkage: use minimum distance between any two nodes
	minDistance := math.Inf(1)
	for _, node1 := range c1.Nodes {
		for _, node2 := range c2.Nodes {
			distance := gi.calculateNodeDistance(node1, node2)
			if distance < minDistance {
				minDistance = distance
			}
		}
	}

	return minDistance
}

// mergeClusters merges two clusters
func (gi *GraphIntegration) mergeClusters(c1, c2 Cluster) Cluster {
	merged := Cluster{
		ID:    fmt.Sprintf("merged_%s_%s", c1.ID, c2.ID),
		Size:  c1.Size + c2.Size,
		Nodes: append(c1.Nodes, c2.Nodes...),
	}

	// Calculate merged cluster strength
	totalStrength := (c1.Strength * float64(c1.Size)) + (c2.Strength * float64(c2.Size))
	merged.Strength = totalStrength / float64(merged.Size)

	return merged
}

// calculateIntraClusterMetrics calculates metrics within a cluster
func (gi *GraphIntegration) calculateIntraClusterMetrics(cluster Cluster, allNodes []ClusterNode) IntraClusterMetrics {
	if len(cluster.Nodes) <= 1 {
		return IntraClusterMetrics{
			AverageDistance: 0,
			MinDistance:     0,
			MaxDistance:     0,
			Cohesion:        1.0,
			Density:         1.0,
		}
	}

	var distances []float64
	minDist := math.Inf(1)
	maxDist := 0.0

	// Calculate all pairwise distances within cluster
	for i, node1 := range cluster.Nodes {
		for j, node2 := range cluster.Nodes {
			if i < j {
				distance := gi.calculateNodeDistance(node1, node2)
				distances = append(distances, distance)
				if distance < minDist {
					minDist = distance
				}
				if distance > maxDist {
					maxDist = distance
				}
			}
		}
	}

	// Calculate average distance
	avgDist := 0.0
	if len(distances) > 0 {
		for _, dist := range distances {
			avgDist += dist
		}
		avgDist /= float64(len(distances))
	}

	// Calculate cohesion (inverse of average distance)
	cohesion := 0.0
	if avgDist > 0 {
		cohesion = 1.0 / (1.0 + avgDist)
	} else {
		cohesion = 1.0
	}

	// Calculate density
	density := gi.calculateDensityScore(cluster)

	return IntraClusterMetrics{
		AverageDistance: avgDist,
		MinDistance:     minDist,
		MaxDistance:     maxDist,
		Cohesion:        cohesion,
		Density:         density,
	}
}

// calculateDensityScore calculates density score for a cluster
func (gi *GraphIntegration) calculateDensityScore(cluster Cluster) float64 {
	if cluster.Size <= 1 {
		return 1.0
	}

	// Count actual connections within cluster
	actualConnections := 0
	nodeMap := make(map[string]bool)
	for _, node := range cluster.Nodes {
		nodeMap[node.ID] = true
	}

	for _, node := range cluster.Nodes {
		for _, conn := range node.Connections {
			if nodeMap[conn.TargetID] {
				actualConnections++
			}
		}
	}

	// Calculate possible connections (n * (n-1))
	possibleConnections := cluster.Size * (cluster.Size - 1)
	if possibleConnections == 0 {
		return 1.0
	}

	return float64(actualConnections) / float64(possibleConnections)
}

// calculateClusterStrength calculates overall strength of a cluster
func (gi *GraphIntegration) calculateClusterStrength(cluster Cluster) float64 {
	if cluster.Size == 0 {
		return 0.0
	}

	// Average combined score of neighborhoods in cluster
	totalScore := 0.0
	for _, node := range cluster.Nodes {
		totalScore += node.Neighborhood.CombinedScore
	}
	avgScore := totalScore / float64(cluster.Size)

	// Factor in cohesion
	strength := (avgScore + cluster.IntraMetrics.Cohesion) / 2.0

	return strength
}

// generateClusterName generates a name for a cluster based on neighborhoods
func (gi *GraphIntegration) generateClusterName(neighborhoods []EnhancedNeighborhood) string {
	if len(neighborhoods) == 0 {
		return "Empty Cluster"
	}

	// Extract common words from neighborhood names
	words := make(map[string]int)
	for _, neighborhood := range neighborhoods {
		if neighborhood.SemanticNeighborhood != nil && neighborhood.Name != "" {
			nameWords := strings.Fields(strings.ToLower(neighborhood.Name))
			for _, word := range nameWords {
				if !gi.isCommonWord(word) {
					words[word]++
				}
			}
		}
	}

	// Find most common meaningful word
	maxCount := 0
	commonWord := ""
	for word, count := range words {
		if count > maxCount {
			maxCount = count
			commonWord = word
		}
	}

	if commonWord != "" {
		return strings.Title(commonWord) + " Group"
	}

	return "Mixed Group"
}

// generateClusterDescription generates a description for a cluster
func (gi *GraphIntegration) generateClusterDescription(neighborhoods []EnhancedNeighborhood) string {
	if len(neighborhoods) == 0 {
		return "Empty cluster"
	}

	totalFiles := 0
	totalScore := 0.0

	for _, neighborhood := range neighborhoods {
		if neighborhood.SemanticNeighborhood != nil {
			totalFiles += len(neighborhood.Files)
		}
		totalScore += neighborhood.CombinedScore
	}

	avgScore := totalScore / float64(len(neighborhoods))

	return fmt.Sprintf("Cluster of %d neighborhoods containing %d files with %.2f average combined score",
		len(neighborhoods), totalFiles, avgScore)
}

// classifyRecommendationStrength classifies the strength of a recommendation
func (gi *GraphIntegration) classifyRecommendationStrength(score float64) string {
	if score >= 0.8 {
		return "very_strong"
	} else if score >= 0.6 {
		return "strong"
	} else if score >= 0.4 {
		return "moderate"
	} else {
		return "weak"
	}
}

// generateRecommendationReason generates a reason for the recommendation
func (gi *GraphIntegration) generateRecommendationReason(neighborhoods []EnhancedNeighborhood, strength float64) string {
	if strength >= 0.8 {
		return "Very strong cluster with high cohesion and frequent co-changes"
	} else if strength >= 0.6 {
		return "Strong cluster with good cohesion and regular co-changes"
	} else if strength >= 0.4 {
		return "Moderate cluster with some cohesion and occasional co-changes"
	} else {
		return "Weak cluster with low cohesion but some related changes"
	}
}

// determineOptimalTasks determines optimal tasks for clustered neighborhoods
func (gi *GraphIntegration) determineOptimalTasks(neighborhoods []EnhancedNeighborhood) []string {
	var tasks []string

	// Analyze file types to suggest relevant tasks
	hasTests := false
	hasConfig := false
	hasDocs := false

	for _, neighborhood := range neighborhoods {
		for _, file := range neighborhood.Files {
			lower := strings.ToLower(file)
			if strings.Contains(lower, "test") || strings.HasSuffix(lower, "_test.go") || strings.HasSuffix(lower, ".test.") {
				hasTests = true
			}
			if strings.Contains(lower, "config") || strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".toml") {
				hasConfig = true
			}
			if strings.Contains(lower, "readme") || strings.Contains(lower, "doc") || strings.HasSuffix(lower, ".md") {
				hasDocs = true
			}
		}
	}

	if hasTests {
		tasks = append(tasks, "testing", "debugging", "quality_assurance")
	}
	if hasConfig {
		tasks = append(tasks, "configuration", "setup", "deployment")
	}
	if hasDocs {
		tasks = append(tasks, "documentation", "maintenance", "onboarding")
	}

	// Default tasks
	if len(tasks) == 0 {
		tasks = []string{"development", "refactoring", "feature_implementation"}
	}

	return tasks
}

// calculateClusterQuality calculates quality metrics for a cluster
func (gi *GraphIntegration) calculateClusterQuality(cluster Cluster, allNeighborhoods []EnhancedNeighborhood) ClusterQuality {
	// For now, return basic metrics based on intra-cluster properties
	silhouetteScore := cluster.IntraMetrics.Cohesion
	daviesBouldinIndex := 1.0 - cluster.IntraMetrics.Density // Lower is better
	calinskiHarabaszIndex := cluster.Strength * float64(cluster.Size)

	return ClusterQuality{
		SilhouetteScore:       silhouetteScore,
		DaviesBouldinIndex:    daviesBouldinIndex,
		CalinskiHarabaszIndex: calinskiHarabaszIndex,
	}
}

// isCommonWord checks if a word is a common English word that should be filtered
func (gi *GraphIntegration) isCommonWord(word string) bool {
	commonWords := []string{
		"the", "and", "is", "are", "was", "were", "have", "has", "had",
		"will", "would", "could", "should", "can", "may", "might", "must",
		"do", "does", "did", "be", "been", "being", "to", "of", "in", "on",
		"at", "by", "for", "with", "from", "as", "but", "or", "if", "when",
		"where", "why", "how", "what", "which", "who", "whom", "whose",
		"this", "that", "these", "those", "a", "an", "it", "its", "they",
		"them", "their", "theirs", "we", "us", "our", "ours", "you", "your",
		"yours", "he", "him", "his", "she", "her", "hers", "i", "me", "my",
		"mine", "all", "any", "each", "every", "no", "none", "some", "many",
		"much", "few", "little", "more", "most", "less", "least", "other",
		"another", "same", "different", "new", "old", "good", "bad", "big",
		"small", "long", "short", "high", "low", "first", "last", "next",
		"previous", "before", "after", "during", "while", "since", "until",
		"now", "then", "here", "there", "where", "anywhere", "everywhere",
		"somewhere", "nowhere",
	}

	word = strings.ToLower(word)
	for _, common := range commonWords {
		if word == common {
			return true
		}
	}
	return false
}

// parseIndex parses index from node ID string or plain number string
func (gi *GraphIntegration) parseIndex(idStr string) int {
	if idStr == "" {
		return 0
	}
	
	// First try parsing as plain number
	if index, err := strconv.Atoi(idStr); err == nil {
		return index
	}
	
	// Then try parsing as node_X format
	parts := strings.Split(idStr, "_")
	if len(parts) < 2 {
		return -1
	}
	
	index, err := strconv.Atoi(parts[1])
	if err != nil {
		return -1
	}
	
	return index
}

// findNodeIndex finds the index of a node by ID
func (gi *GraphIntegration) findNodeIndex(nodeID string) int {
	if !strings.HasPrefix(nodeID, "node_") {
		return -1
	}
	
	return gi.parseIndex(nodeID)
}

// Helper functions for dependency and structural analysis

// markCircularDependencies marks circular dependencies in connections
func (gi *GraphIntegration) markCircularDependencies(connections []DependencyConnection) {
	// Build adjacency map
	adjMap := make(map[string][]string)
	for _, conn := range connections {
		adjMap[conn.SourceFile] = append(adjMap[conn.SourceFile], conn.TargetFile)
	}

	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range adjMap[node] {
			if !visited[neighbor] && hasCycle(neighbor) {
				return true
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	// Mark circular dependencies
	for i := range connections {
		if hasCycle(connections[i].SourceFile) {
			connections[i].ImportType = "circular"
		}
	}
}

// getFileSymbols gets symbols for a file from the code graph
func (gi *GraphIntegration) getFileSymbols(filePath string) []string {
	if gi.codeGraph == nil {
		return []string{}
	}

	fileNode := gi.codeGraph.Files[filePath]
	if fileNode == nil {
		return []string{}
	}

	var symbolNames []string
	for _, symbolID := range fileNode.Symbols {
		if symbol := gi.codeGraph.Symbols[symbolID]; symbol != nil {
			symbolNames = append(symbolNames, symbol.Name)
		}
	}

	return symbolNames
}

// calculateSharedSymbols calculates number of shared symbols between two files
func (gi *GraphIntegration) calculateSharedSymbols(symbols1, symbols2 []string) int {
	symbolSet := make(map[string]bool)
	for _, symbol := range symbols1 {
		symbolSet[symbol] = true
	}

	shared := 0
	for _, symbol := range symbols2 {
		if symbolSet[symbol] {
			shared++
		}
	}

	return shared
}

// findSharedPatterns finds shared patterns between symbols
func (gi *GraphIntegration) findSharedPatterns(symbols1, symbols2 []string) []string {
	patterns1 := gi.extractPatterns(symbols1)
	patterns2 := gi.extractPatterns(symbols2)

	var shared []string
	patternSet := make(map[string]bool)
	for _, pattern := range patterns1 {
		patternSet[pattern] = true
	}

	for _, pattern := range patterns2 {
		if patternSet[pattern] && !gi.contains(shared, pattern) {
			shared = append(shared, pattern)
		}
	}

	return shared
}

// extractPatterns extracts naming patterns from symbols
func (gi *GraphIntegration) extractPatterns(symbols []string) []string {
	var patterns []string
	
	for _, symbol := range symbols {
		// Extract prefixes
		if len(symbol) > 3 {
			patterns = append(patterns, "prefix_"+symbol[:3])
		}
		
		// Extract suffixes
		if len(symbol) > 3 {
			patterns = append(patterns, "suffix_"+symbol[len(symbol)-3:])
		}
		
		// Extract case patterns
		if strings.Title(symbol) == symbol {
			patterns = append(patterns, "title_case")
		}
		if strings.ToLower(symbol) == symbol {
			patterns = append(patterns, "lower_case")
		}
	}
	
	return patterns
}

// contains checks if a slice contains a string
func (gi *GraphIntegration) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// calculateDependencyScore calculates score from dependency connections
func (gi *GraphIntegration) calculateDependencyScore(dependencies []DependencyConnection) float64 {
	if len(dependencies) == 0 {
		return 0.0
	}

	totalStrength := 0.0
	for _, dep := range dependencies {
		totalStrength += dep.Strength
	}

	return totalStrength / float64(len(dependencies))
}

// calculateStructuralScore calculates score from structural similarities
func (gi *GraphIntegration) calculateStructuralScore(similarities []StructuralSimilarity) float64 {
	if len(similarities) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, sim := range similarities {
		totalScore += sim.SimilarityScore
	}

	return totalScore / float64(len(similarities))
}