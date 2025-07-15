package git

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ChangePattern represents a detected change pattern
type ChangePattern struct {
	Name           string            `json:"name"`
	Files          []string          `json:"files"`
	Frequency      int               `json:"frequency"`
	Confidence     float64           `json:"confidence"`
	LastOccurrence time.Time         `json:"last_occurrence"`
	AvgInterval    time.Duration     `json:"avg_interval"`
	Metadata       map[string]string `json:"metadata"`
}

// FileRelationship represents the relationship between two files
type FileRelationship struct {
	File1       string  `json:"file1"`
	File2       string  `json:"file2"`
	Correlation float64 `json:"correlation"`
	Frequency   int     `json:"frequency"`
	Strength    string  `json:"strength"` // "strong", "moderate", "weak"
}

// ModuleGroup represents a group of related files
type ModuleGroup struct {
	Name                string            `json:"name"`
	Files               []string          `json:"files"`
	CohesionScore       float64           `json:"cohesion_score"`
	ChangeFrequency     int               `json:"change_frequency"`
	LastChanged         time.Time         `json:"last_changed"`
	CommonOperations    []string          `json:"common_operations"`
	InternalConnections int               `json:"internal_connections"`
	ExternalConnections int               `json:"external_connections"`
}

// PatternDetector analyzes git history to detect change patterns
type PatternDetector struct {
	analyzer    *GitAnalyzer
	minSupport  float64 // Minimum support threshold for patterns
	minConfidence float64 // Minimum confidence threshold for patterns
}

// NewPatternDetector creates a new pattern detector
func NewPatternDetector(analyzer *GitAnalyzer) *PatternDetector {
	return &PatternDetector{
		analyzer:      analyzer,
		minSupport:    0.1,  // 10% minimum support
		minConfidence: 0.6,  // 60% minimum confidence
	}
}

// SetThresholds sets the minimum support and confidence thresholds
func (pd *PatternDetector) SetThresholds(minSupport, minConfidence float64) {
	pd.minSupport = minSupport
	pd.minConfidence = minConfidence
}

// DetectChangePatterns finds recurring change patterns in the git history
func (pd *PatternDetector) DetectChangePatterns(days int) ([]ChangePattern, error) {
	commits, err := pd.analyzer.GetCommitHistory(days)
	if err != nil {
		return nil, err
	}

	// Group commits by similar file sets
	patterns := make(map[string]*ChangePattern)
	
	for _, commit := range commits {
		if len(commit.Files) < 2 {
			continue // Skip single-file commits
		}

		// Sort files to create a consistent pattern key
		sortedFiles := make([]string, len(commit.Files))
		copy(sortedFiles, commit.Files)
		sort.Strings(sortedFiles)
		
		// Create pattern key based on file set
		key := strings.Join(sortedFiles, "|")
		
		if pattern, exists := patterns[key]; exists {
			pattern.Frequency++
			if commit.Timestamp.After(pattern.LastOccurrence) {
				pattern.LastOccurrence = commit.Timestamp
			}
		} else {
			patterns[key] = &ChangePattern{
				Name:           generatePatternName(sortedFiles),
				Files:          sortedFiles,
				Frequency:      1,
				LastOccurrence: commit.Timestamp,
				Metadata:       make(map[string]string),
			}
		}
	}

	// Filter patterns by support and calculate confidence
	var result []ChangePattern
	totalCommits := len(commits)
	
	for _, pattern := range patterns {
		support := float64(pattern.Frequency) / float64(totalCommits)
		if support >= pd.minSupport {
			pattern.Confidence = pd.calculateConfidence(pattern, commits)
			if pattern.Confidence >= pd.minConfidence {
				pattern.AvgInterval = pd.calculateAvgInterval(pattern, commits)
				result = append(result, *pattern)
			}
		}
	}

	// Sort by frequency (most common first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Frequency > result[j].Frequency
	})

	return result, nil
}

// DetectFileRelationships finds relationships between files based on change patterns
func (pd *PatternDetector) DetectFileRelationships(days int) ([]FileRelationship, error) {
	coOccurrences, err := pd.analyzer.GetFileCoOccurrences(days)
	if err != nil {
		return nil, err
	}

	changeFreq, err := pd.analyzer.GetChangeFrequency(days)
	if err != nil {
		return nil, err
	}

	var relationships []FileRelationship
	processed := make(map[string]bool)

	for file1, partners := range coOccurrences {
		for _, file2 := range partners {
			// Avoid duplicate relationships
			key := file1 + "|" + file2
			reverseKey := file2 + "|" + file1
			if processed[key] || processed[reverseKey] {
				continue
			}
			processed[key] = true

			// Calculate correlation
			freq1 := changeFreq[file1]
			freq2 := changeFreq[file2]
			
			// Count co-occurrences
			coOccCount := 0
			if partners2, exists := coOccurrences[file2]; exists {
				for _, partner := range partners2 {
					if partner == file1 {
						coOccCount = 1
						break
					}
				}
			}

			// Calculate Jaccard similarity
			correlation := float64(coOccCount) / float64(freq1+freq2-coOccCount)
			
			relationship := FileRelationship{
				File1:       file1,
				File2:       file2,
				Correlation: correlation,
				Frequency:   coOccCount,
				Strength:    classifyStrength(correlation),
			}

			relationships = append(relationships, relationship)
		}
	}

	// Sort by correlation strength
	sort.Slice(relationships, func(i, j int) bool {
		return relationships[i].Correlation > relationships[j].Correlation
	})

	return relationships, nil
}

// DetectModuleGroups identifies cohesive groups of files that change together
func (pd *PatternDetector) DetectModuleGroups(days int) ([]ModuleGroup, error) {
	relationships, err := pd.DetectFileRelationships(days)
	if err != nil {
		return nil, err
	}

	changeFreq, err := pd.analyzer.GetChangeFrequency(days)
	if err != nil {
		return nil, err
	}

	lastModified, err := pd.analyzer.GetLastModified()
	if err != nil {
		return nil, err
	}

	// Build adjacency list from strong relationships
	adjacency := make(map[string][]string)
	for _, rel := range relationships {
		if rel.Strength == "strong" {
			adjacency[rel.File1] = append(adjacency[rel.File1], rel.File2)
			adjacency[rel.File2] = append(adjacency[rel.File2], rel.File1)
		}
	}

	// Find connected components using DFS
	visited := make(map[string]bool)
	var groups []ModuleGroup

	for file := range adjacency {
		if !visited[file] {
			group := pd.findConnectedComponent(file, adjacency, visited)
			if len(group) >= 2 { // Only consider groups with 2+ files
				moduleGroup := pd.buildModuleGroup(group, changeFreq, lastModified, relationships)
				groups = append(groups, moduleGroup)
			}
		}
	}

	// Sort by cohesion score
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].CohesionScore > groups[j].CohesionScore
	})

	return groups, nil
}

// calculateConfidence calculates the confidence score for a pattern
func (pd *PatternDetector) calculateConfidence(pattern *ChangePattern, commits []CommitInfo) float64 {
	// Count how often these files change together vs separately
	together := 0
	separate := 0

	for _, commit := range commits {
		hasAll := true
		hasAny := false

		for _, file := range pattern.Files {
			found := false
			for _, commitFile := range commit.Files {
				if file == commitFile {
					found = true
					hasAny = true
					break
				}
			}
			if !found {
				hasAll = false
			}
		}

		if hasAll {
			together++
		} else if hasAny {
			separate++
		}
	}

	if together+separate == 0 {
		return 0.0
	}

	return float64(together) / float64(together+separate)
}

// calculateAvgInterval calculates the average time between pattern occurrences
func (pd *PatternDetector) calculateAvgInterval(pattern *ChangePattern, commits []CommitInfo) time.Duration {
	var occurrences []time.Time

	for _, commit := range commits {
		hasAll := true
		for _, file := range pattern.Files {
			found := false
			for _, commitFile := range commit.Files {
				if file == commitFile {
					found = true
					break
				}
			}
			if !found {
				hasAll = false
				break
			}
		}

		if hasAll {
			occurrences = append(occurrences, commit.Timestamp)
		}
	}

	if len(occurrences) <= 1 {
		return 0
	}

	// Sort occurrences
	sort.Slice(occurrences, func(i, j int) bool {
		return occurrences[i].Before(occurrences[j])
	})

	// Calculate intervals
	var totalInterval time.Duration
	for i := 1; i < len(occurrences); i++ {
		totalInterval += occurrences[i].Sub(occurrences[i-1])
	}

	return totalInterval / time.Duration(len(occurrences)-1)
}

// findConnectedComponent finds all files connected to the given file
func (pd *PatternDetector) findConnectedComponent(file string, adjacency map[string][]string, visited map[string]bool) []string {
	if visited[file] {
		return nil
	}

	visited[file] = true
	component := []string{file}

	for _, neighbor := range adjacency[file] {
		if !visited[neighbor] {
			component = append(component, pd.findConnectedComponent(neighbor, adjacency, visited)...)
		}
	}

	return component
}

// buildModuleGroup builds a module group from a connected component
func (pd *PatternDetector) buildModuleGroup(files []string, changeFreq map[string]int, lastModified map[string]time.Time, relationships []FileRelationship) ModuleGroup {
	// Calculate total change frequency
	totalFreq := 0
	var latestChange time.Time
	
	for _, file := range files {
		totalFreq += changeFreq[file]
		if modified, exists := lastModified[file]; exists {
			if latestChange.IsZero() || modified.After(latestChange) {
				latestChange = modified
			}
		}
	}

	// Count internal vs external connections
	internal := 0
	external := 0
	
	for _, rel := range relationships {
		isFile1InGroup := contains(files, rel.File1)
		isFile2InGroup := contains(files, rel.File2)
		
		if isFile1InGroup && isFile2InGroup {
			internal++
		} else if isFile1InGroup || isFile2InGroup {
			external++
		}
	}

	// Calculate cohesion score
	cohesionScore := 0.0
	if internal+external > 0 {
		cohesionScore = float64(internal) / float64(internal+external)
	}

	return ModuleGroup{
		Name:                generateModuleName(files),
		Files:               files,
		CohesionScore:       cohesionScore,
		ChangeFrequency:     totalFreq,
		LastChanged:         latestChange,
		CommonOperations:    []string{}, // Will be populated by analyze common operations
		InternalConnections: internal,
		ExternalConnections: external,
	}
}

// Helper functions
func generatePatternName(files []string) string {
	if len(files) == 0 {
		return "empty-pattern"
	}
	
	// For single file, use file name without extension
	if len(files) == 1 {
		return strings.TrimSuffix(files[0], filepath.Ext(files[0])) + "-pattern"
	}
	
	// Try to find common prefix or directory
	commonPrefix := findCommonPrefix(files)
	if commonPrefix != "" {
		// Clean up the prefix by removing trailing slashes and replacing slashes with dashes
		commonPrefix = strings.TrimRight(commonPrefix, "/")
		return strings.ReplaceAll(commonPrefix, "/", "-") + "-group"
	}
	
	// Use first file as base
	return strings.TrimSuffix(files[0], filepath.Ext(files[0])) + "-pattern"
}

func generateModuleName(files []string) string {
	if len(files) == 0 {
		return "empty-module"
	}
	
	// Try to find common directory
	commonDir := findCommonDirectory(files)
	if commonDir != "" {
		return commonDir + "-module"
	}
	
	// Use file count as fallback
	return fmt.Sprintf("module-%d-files", len(files))
}

func findCommonPrefix(files []string) string {
	if len(files) == 0 {
		return ""
	}
	
	prefix := files[0]
	for _, file := range files[1:] {
		prefix = commonPrefix(prefix, file)
		if prefix == "" {
			break
		}
	}
	
	// For directory paths, don't truncate at the last slash - return the full common path
	return prefix
}

func findCommonDirectory(files []string) string {
	if len(files) == 0 {
		return ""
	}
	
	// Extract directories
	dirs := make([]string, len(files))
	for i, file := range files {
		if idx := strings.LastIndex(file, "/"); idx > 0 {
			dirs[i] = file[:idx]
		} else {
			dirs[i] = "" // File in root directory
		}
	}
	
	prefix := findCommonPrefix(dirs)
	
	// Return the common directory path (replacing / with -)
	if prefix != "" {
		return strings.ReplaceAll(prefix, "/", "-")
	}
	
	return ""
}

func commonPrefix(a, b string) string {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	
	for i := 0; i < minLen; i++ {
		if a[i] != b[i] {
			return a[:i]
		}
	}
	
	return a[:minLen]
}

func classifyStrength(correlation float64) string {
	if correlation >= 0.7 {
		return "strong"
	} else if correlation >= 0.4 {
		return "moderate"
	}
	return "weak"
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

