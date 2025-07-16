package git

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// SemanticAnalyzer provides high-level semantic analysis of git repositories
type SemanticAnalyzer struct {
	gitAnalyzer     *GitAnalyzer
	patternDetector *PatternDetector
	config          *SemanticConfig
}

// SemanticConfig holds configuration for semantic analysis
type SemanticConfig struct {
	AnalysisPeriodDays    int     `json:"analysis_period_days"`
	MinChangeCorrelation  float64 `json:"min_change_correlation"`
	MinPatternSupport     float64 `json:"min_pattern_support"`
	MinPatternConfidence  float64 `json:"min_pattern_confidence"`
	MaxNeighborhoodSize   int     `json:"max_neighborhood_size"`
	IncludeTestFiles      bool    `json:"include_test_files"`
	IncludeDocFiles       bool    `json:"include_doc_files"`
	IncludeConfigFiles    bool    `json:"include_config_files"`
}

// DefaultSemanticConfig returns default configuration with optimized thresholds
func DefaultSemanticConfig() *SemanticConfig {
	return &SemanticConfig{
		AnalysisPeriodDays:    30,
		MinChangeCorrelation:  0.4,    // Reduced from 0.6 to 0.4 for more flexibility
		MinPatternSupport:     0.05,   // Reduced from 0.1 to 0.05 (5% of commits)
		MinPatternConfidence:  0.3,    // Reduced from 0.6 to 0.3 for FP-Growth
		MaxNeighborhoodSize:   15,     // Increased from 10 to 15 for larger patterns
		IncludeTestFiles:      true,
		IncludeDocFiles:       false,
		IncludeConfigFiles:    false,
	}
}

// SemanticNeighborhood represents a group of files that change together
type SemanticNeighborhood struct {
	Name                string                 `json:"name"`
	Files               []string               `json:"files"`
	ChangeFrequency     int                    `json:"change_frequency"`
	LastChanged         time.Time              `json:"last_changed"`
	CommonOperations    []string               `json:"common_operations"`
	CorrelationStrength float64                `json:"correlation_strength"`
	Confidence          float64                `json:"confidence"`
	Metadata            map[string]interface{} `json:"metadata"`
}

// ContextRecommendation provides context recommendations for AI assistants
type ContextRecommendation struct {
	ForFile       string   `json:"for_file"`
	IncludeFiles  []string `json:"include_files"`
	Reason        string   `json:"reason"`
	Confidence    float64  `json:"confidence"`
	Priority      string   `json:"priority"` // "high", "medium", "low"
	ChangePattern string   `json:"change_pattern"`
}

// SemanticAnalysisResult holds the complete analysis results
type SemanticAnalysisResult struct {
	Neighborhoods        []SemanticNeighborhood  `json:"neighborhoods"`
	ContextRecommendations []ContextRecommendation `json:"context_recommendations"`
	ChangePatterns       []ChangePattern         `json:"change_patterns"`
	FileRelationships    []FileRelationship      `json:"file_relationships"`
	ModuleGroups         []ModuleGroup           `json:"module_groups"`
	AnalysisSummary      AnalysisSummary         `json:"analysis_summary"`
}

// AnalysisSummary provides overview statistics
type AnalysisSummary struct {
	TotalFiles              int                `json:"total_files"`
	ActiveFiles             int                `json:"active_files"`
	NeighborhoodsFound      int                `json:"neighborhoods_found"`
	PatternsFound           int                `json:"patterns_found"`
	StrongRelationships     int                `json:"strong_relationships"`
	ModerateRelationships   int                `json:"moderate_relationships"`
	WeakRelationships       int                `json:"weak_relationships"`
	AnalysisPeriodDays      int                `json:"analysis_period_days"`
	AnalysisDate            time.Time          `json:"analysis_date"`
	RepositoryInfo          RepositoryInfo     `json:"repository_info"`
	PerformanceMetrics      PerformanceMetrics `json:"performance_metrics"`
}

// RepositoryInfo holds basic repository information
type RepositoryInfo struct {
	CurrentBranch string `json:"current_branch"`
	RemoteURL     string `json:"remote_url"`
	IsClean       bool   `json:"is_clean"`
	CommitCount   int    `json:"commit_count"`
}

// PerformanceMetrics holds performance information
type PerformanceMetrics struct {
	AnalysisTime     time.Duration `json:"analysis_time"`
	GitCommandTime   time.Duration `json:"git_command_time"`
	ProcessingTime   time.Duration `json:"processing_time"`
	FilesProcessed   int           `json:"files_processed"`
	PatternsAnalyzed int           `json:"patterns_analyzed"`
}

// NewSemanticAnalyzer creates a new semantic analyzer
func NewSemanticAnalyzer(repoPath string, config *SemanticConfig) (*SemanticAnalyzer, error) {
	if config == nil {
		config = DefaultSemanticConfig()
	}

	gitAnalyzer, err := NewGitAnalyzer(repoPath)
	if err != nil {
		return nil, err
	}

	patternDetector := NewPatternDetector(gitAnalyzer)
	patternDetector.SetThresholds(config.MinPatternSupport, config.MinPatternConfidence)

	return &SemanticAnalyzer{
		gitAnalyzer:     gitAnalyzer,
		patternDetector: patternDetector,
		config:          config,
	}, nil
}

// AnalyzeRepository performs comprehensive semantic analysis
func (sa *SemanticAnalyzer) AnalyzeRepository() (*SemanticAnalysisResult, error) {
	startTime := time.Now()
	
	// Get repository info
	repoInfo, err := sa.getRepositoryInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository info: %w", err)
	}

	// Detect change patterns
	patterns, err := sa.patternDetector.DetectChangePatterns(sa.config.AnalysisPeriodDays)
	if err != nil {
		return nil, fmt.Errorf("failed to detect change patterns: %w", err)
	}

	// Detect file relationships
	relationships, err := sa.patternDetector.DetectFileRelationships(sa.config.AnalysisPeriodDays)
	if err != nil {
		return nil, fmt.Errorf("failed to detect file relationships: %w", err)
	}

	// Detect module groups
	moduleGroups, err := sa.patternDetector.DetectModuleGroups(sa.config.AnalysisPeriodDays)
	if err != nil {
		return nil, fmt.Errorf("failed to detect module groups: %w", err)
	}

	// Build semantic neighborhoods
	neighborhoods := sa.buildSemanticNeighborhoods(patterns, relationships, moduleGroups)

	// Generate context recommendations
	contextRecommendations := sa.generateContextRecommendations(neighborhoods, relationships)

	// Create analysis summary
	summary := sa.createAnalysisSummary(neighborhoods, patterns, relationships, repoInfo, startTime)

	return &SemanticAnalysisResult{
		Neighborhoods:          neighborhoods,
		ContextRecommendations: contextRecommendations,
		ChangePatterns:         patterns,
		FileRelationships:      relationships,
		ModuleGroups:           moduleGroups,
		AnalysisSummary:        summary,
	}, nil
}

// GetContextRecommendationsForFile provides context recommendations for a specific file
func (sa *SemanticAnalyzer) GetContextRecommendationsForFile(filePath string) ([]ContextRecommendation, error) {
	// Get neighborhoods
	result, err := sa.AnalyzeRepository()
	if err != nil {
		return nil, err
	}

	// Filter recommendations for the specific file
	var recommendations []ContextRecommendation
	for _, rec := range result.ContextRecommendations {
		if rec.ForFile == filePath {
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations, nil
}

// buildSemanticNeighborhoods creates semantic neighborhoods from patterns and relationships
func (sa *SemanticAnalyzer) buildSemanticNeighborhoods(patterns []ChangePattern, relationships []FileRelationship, groups []ModuleGroup) []SemanticNeighborhood {
	var neighborhoods []SemanticNeighborhood

	// Convert module groups to neighborhoods
	for _, group := range groups {
		neighborhood := SemanticNeighborhood{
			Name:                group.Name,
			Files:               sa.filterFiles(group.Files),
			ChangeFrequency:     group.ChangeFrequency,
			LastChanged:         group.LastChanged,
			CommonOperations:    group.CommonOperations,
			CorrelationStrength: group.CohesionScore,
			Confidence:          group.CohesionScore,
			Metadata: map[string]interface{}{
				"type":                  "module_group",
				"internal_connections":  group.InternalConnections,
				"external_connections":  group.ExternalConnections,
				"cohesion_score":        group.CohesionScore,
			},
		}
		neighborhoods = append(neighborhoods, neighborhood)
	}

	// Add pattern-based neighborhoods
	for _, pattern := range patterns {
		if len(pattern.Files) > 1 {
			neighborhood := SemanticNeighborhood{
				Name:                pattern.Name,
				Files:               sa.filterFiles(pattern.Files),
				ChangeFrequency:     pattern.Frequency,
				LastChanged:         pattern.LastOccurrence,
				CommonOperations:    sa.extractCommonOperations(pattern),
				CorrelationStrength: pattern.Confidence,
				Confidence:          pattern.Confidence,
				Metadata: map[string]interface{}{
					"type":         "change_pattern",
					"avg_interval": pattern.AvgInterval.String(),
					"pattern_metadata": pattern.Metadata,
				},
			}
			neighborhoods = append(neighborhoods, neighborhood)
		}
	}

	// Remove duplicates and merge similar neighborhoods
	neighborhoods = sa.mergeSimilarNeighborhoods(neighborhoods)

	// Sort by strength
	sort.Slice(neighborhoods, func(i, j int) bool {
		return neighborhoods[i].CorrelationStrength > neighborhoods[j].CorrelationStrength
	})

	// Limit to max neighborhood size
	if len(neighborhoods) > sa.config.MaxNeighborhoodSize {
		neighborhoods = neighborhoods[:sa.config.MaxNeighborhoodSize]
	}

	return neighborhoods
}

// generateContextRecommendations creates context recommendations for AI assistants
func (sa *SemanticAnalyzer) generateContextRecommendations(neighborhoods []SemanticNeighborhood, relationships []FileRelationship) []ContextRecommendation {
	var recommendations []ContextRecommendation

	// Create file-to-neighborhood mapping
	fileToNeighborhood := make(map[string][]SemanticNeighborhood)
	for _, neighborhood := range neighborhoods {
		for _, file := range neighborhood.Files {
			fileToNeighborhood[file] = append(fileToNeighborhood[file], neighborhood)
		}
	}

	// Generate recommendations for each file
	for file, nhoods := range fileToNeighborhood {
		if len(nhoods) == 0 {
			continue
		}

		// Find the strongest neighborhood for this file
		var strongestNeighborhood SemanticNeighborhood
		maxStrength := 0.0
		for _, nhood := range nhoods {
			if nhood.CorrelationStrength > maxStrength {
				maxStrength = nhood.CorrelationStrength
				strongestNeighborhood = nhood
			}
		}

		// Create recommendation
		var includeFiles []string
		for _, f := range strongestNeighborhood.Files {
			if f != file {
				includeFiles = append(includeFiles, f)
			}
		}

		if len(includeFiles) > 0 {
			recommendation := ContextRecommendation{
				ForFile:       file,
				IncludeFiles:  includeFiles,
				Reason:        sa.generateReasonText(strongestNeighborhood),
				Confidence:    strongestNeighborhood.Confidence,
				Priority:      sa.classifyPriority(strongestNeighborhood.Confidence),
				ChangePattern: strongestNeighborhood.Name,
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	// Sort by confidence
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Confidence > recommendations[j].Confidence
	})

	return recommendations
}

// Helper methods

func (sa *SemanticAnalyzer) filterFiles(files []string) []string {
	var filtered []string
	for _, file := range files {
		if sa.shouldIncludeFile(file) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func (sa *SemanticAnalyzer) shouldIncludeFile(file string) bool {
	// Check file type filters
	if strings.Contains(file, "_test.") && !sa.config.IncludeTestFiles {
		return false
	}
	if (strings.HasSuffix(file, ".md") || strings.HasSuffix(file, ".txt")) && !sa.config.IncludeDocFiles {
		return false
	}
	if (strings.HasSuffix(file, ".json") || strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".yml")) && !sa.config.IncludeConfigFiles {
		return false
	}
	
	// Exclude hidden files and common non-source files
	if strings.HasPrefix(file, ".") {
		return false
	}

	return true
}

func (sa *SemanticAnalyzer) extractCommonOperations(pattern ChangePattern) []string {
	// Analyze commit messages for common operations
	// This is a simplified implementation
	operations := []string{}
	
	// Extract from metadata if available
	if pattern.Metadata != nil {
		if ops, exists := pattern.Metadata["operations"]; exists {
			operations = strings.Split(ops, ",")
		}
	}
	
	// Default operations based on pattern characteristics
	if len(operations) == 0 {
		if pattern.Frequency > 10 {
			operations = append(operations, "frequent_updates")
		}
		if pattern.Confidence > 0.8 {
			operations = append(operations, "coordinated_changes")
		}
	}
	
	return operations
}

func (sa *SemanticAnalyzer) mergeSimilarNeighborhoods(neighborhoods []SemanticNeighborhood) []SemanticNeighborhood {
	// Simple duplicate removal based on file overlap
	var unique []SemanticNeighborhood
	
	for _, nhood := range neighborhoods {
		isDuplicate := false
		for _, existing := range unique {
			if sa.calculateFileOverlap(nhood.Files, existing.Files) > 0.8 {
				isDuplicate = true
				break
			}
		}
		
		if !isDuplicate {
			unique = append(unique, nhood)
		}
	}
	
	return unique
}

func (sa *SemanticAnalyzer) calculateFileOverlap(files1, files2 []string) float64 {
	if len(files1) == 0 || len(files2) == 0 {
		return 0.0
	}
	
	intersection := 0
	for _, f1 := range files1 {
		for _, f2 := range files2 {
			if f1 == f2 {
				intersection++
				break
			}
		}
	}
	
	union := len(files1) + len(files2) - intersection
	return float64(intersection) / float64(union)
}

func (sa *SemanticAnalyzer) generateReasonText(neighborhood SemanticNeighborhood) string {
	if neighborhood.ChangeFrequency > 20 {
		return fmt.Sprintf("Files in %s frequently change together (%d times)", neighborhood.Name, neighborhood.ChangeFrequency)
	} else if neighborhood.CorrelationStrength > 0.8 {
		return fmt.Sprintf("Files in %s have strong correlation (%.1f%% confidence)", neighborhood.Name, neighborhood.CorrelationStrength*100)
	} else {
		return fmt.Sprintf("Files in %s are related through change patterns", neighborhood.Name)
	}
}

func (sa *SemanticAnalyzer) classifyPriority(confidence float64) string {
	if confidence >= 0.8 {
		return "high"
	} else if confidence >= 0.6 {
		return "medium"
	}
	return "low"
}

func (sa *SemanticAnalyzer) getRepositoryInfo() (RepositoryInfo, error) {
	branch, err := sa.gitAnalyzer.GetBranchInfo()
	if err != nil {
		branch = "unknown"
	}
	
	remote, err := sa.gitAnalyzer.GetRemoteInfo()
	if err != nil {
		remote = "none"
	}
	
	commits, err := sa.gitAnalyzer.GetCommitHistory(sa.config.AnalysisPeriodDays)
	if err != nil {
		commits = []CommitInfo{}
	}
	
	return RepositoryInfo{
		CurrentBranch: branch,
		RemoteURL:     remote,
		IsClean:       true, // TODO: implement git status check
		CommitCount:   len(commits),
	}, nil
}

func (sa *SemanticAnalyzer) createAnalysisSummary(neighborhoods []SemanticNeighborhood, patterns []ChangePattern, relationships []FileRelationship, repoInfo RepositoryInfo, startTime time.Time) AnalysisSummary {
	strongRels := 0
	moderateRels := 0
	weakRels := 0
	
	for _, rel := range relationships {
		switch rel.Strength {
		case "strong":
			strongRels++
		case "moderate":
			moderateRels++
		case "weak":
			weakRels++
		}
	}
	
	// Count unique files
	fileSet := make(map[string]bool)
	for _, nhood := range neighborhoods {
		for _, file := range nhood.Files {
			fileSet[file] = true
		}
	}
	
	return AnalysisSummary{
		TotalFiles:            len(fileSet),
		ActiveFiles:           len(fileSet), // TODO: distinguish active vs inactive
		NeighborhoodsFound:    len(neighborhoods),
		PatternsFound:         len(patterns),
		StrongRelationships:   strongRels,
		ModerateRelationships: moderateRels,
		WeakRelationships:     weakRels,
		AnalysisPeriodDays:    sa.config.AnalysisPeriodDays,
		AnalysisDate:          time.Now(),
		RepositoryInfo:        repoInfo,
		PerformanceMetrics: PerformanceMetrics{
			AnalysisTime:     time.Since(startTime),
			GitCommandTime:   0, // TODO: track git command time
			ProcessingTime:   time.Since(startTime),
			FilesProcessed:   len(fileSet),
			PatternsAnalyzed: len(patterns),
		},
	}
}