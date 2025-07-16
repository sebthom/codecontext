package git

import (
	"sort"
	"strings"
	"time"
)

// SimplePatternsDetector provides a simplified pattern detection approach
type SimplePatternsDetector struct {
	minSupport    float64
	minConfidence float64
	filterFunc    func(string) bool
}

// NewSimplePatternsDetector creates a new simple patterns detector
func NewSimplePatternsDetector(minSupport, minConfidence float64) *SimplePatternsDetector {
	return &SimplePatternsDetector{
		minSupport:    minSupport,
		minConfidence: minConfidence,
		filterFunc:    func(string) bool { return true },
	}
}

// SetFileFilter sets the file filter function
func (spd *SimplePatternsDetector) SetFileFilter(filter func(string) bool) {
	spd.filterFunc = filter
}

// FileCoOccurrence represents files that occur together
type FileCoOccurrence struct {
	Files       []string
	Count       int
	Commits     []string
	LastSeen    time.Time
	Confidence  float64
}

// FrequentItemset represents a discovered frequent pattern
type FrequentItemset struct {
	Items       []string  // Files that change together
	Support     int       // Absolute support count
	Confidence  float64   // Confidence score
	Frequency   int       // How often this pattern occurs
	LastSeen    time.Time // Last time this pattern was seen
}

// MineSimplePatterns mines patterns using a simplified approach
func (spd *SimplePatternsDetector) MineSimplePatterns(commits []CommitInfo) ([]FrequentItemset, error) {
	// Step 1: Filter commits and files
	var filteredCommits []CommitInfo
	for _, commit := range commits {
		var filteredFiles []string
		for _, file := range commit.Files {
			if spd.filterFunc(file) {
				filteredFiles = append(filteredFiles, file)
			}
		}
		
		// Only include commits with 2+ files
		if len(filteredFiles) >= 2 {
			filteredCommits = append(filteredCommits, CommitInfo{
				Hash:      commit.Hash,
				Files:     filteredFiles,
				Timestamp: commit.Timestamp,
				Author:    commit.Author,
				Message:   commit.Message,
			})
		}
	}
	
	// Step 2: Find all unique file pairs using optimized approach
	pairCounts := make(map[string]*FileCoOccurrence)
	
	for _, commit := range filteredCommits {
		// Pre-sort files for consistent key generation
		files := make([]string, len(commit.Files))
		copy(files, commit.Files)
		sort.Strings(files)
		
		// Generate all pairs in this commit
		for i := 0; i < len(files); i++ {
			for j := i + 1; j < len(files); j++ {
				file1, file2 := files[i], files[j]
				key := file1 + "|" + file2
				
				if cooc, exists := pairCounts[key]; exists {
					cooc.Count++
					// Only store first few commit hashes to save memory
					if len(cooc.Commits) < 10 {
						cooc.Commits = append(cooc.Commits, commit.Hash)
					}
					if commit.Timestamp.After(cooc.LastSeen) {
						cooc.LastSeen = commit.Timestamp
					}
				} else {
					pairCounts[key] = &FileCoOccurrence{
						Files:    []string{file1, file2},
						Count:    1,
						Commits:  []string{commit.Hash},
						LastSeen: commit.Timestamp,
					}
				}
			}
		}
	}
	
	// Step 3: Filter by minimum support
	minSupportCount := int(spd.minSupport * float64(len(filteredCommits)))
	if minSupportCount < 1 {
		minSupportCount = 1
	}
	
	var validPairs []*FileCoOccurrence
	for _, cooc := range pairCounts {
		if cooc.Count >= minSupportCount {
			// Calculate confidence
			cooc.Confidence = spd.calculatePairConfidence(cooc.Files, filteredCommits)
			
			if cooc.Confidence >= spd.minConfidence {
				validPairs = append(validPairs, cooc)
			}
		}
	}
	
	// Step 4: Convert to FrequentItemset format
	var itemsets []FrequentItemset
	for _, pair := range validPairs {
		itemsets = append(itemsets, FrequentItemset{
			Items:       pair.Files,
			Support:     pair.Count,
			Confidence:  pair.Confidence,
			Frequency:   pair.Count,
			LastSeen:    pair.LastSeen,
		})
	}
	
	// Sort by frequency (descending)
	sort.Slice(itemsets, func(i, j int) bool {
		return itemsets[i].Frequency > itemsets[j].Frequency
	})
	
	return itemsets, nil
}

// calculatePairConfidence calculates confidence for a file pair using cached counts
func (spd *SimplePatternsDetector) calculatePairConfidence(files []string, commits []CommitInfo) float64 {
	if len(files) != 2 {
		return 0.0
	}
	
	file1, file2 := files[0], files[1]
	
	// Build file occurrence map for efficient lookup
	fileOccurrences := make(map[string][]bool)
	fileOccurrences[file1] = make([]bool, len(commits))
	fileOccurrences[file2] = make([]bool, len(commits))
	
	for i, commit := range commits {
		for _, file := range commit.Files {
			if file == file1 {
				fileOccurrences[file1][i] = true
			}
			if file == file2 {
				fileOccurrences[file2][i] = true
			}
		}
	}
	
	// Count occurrences efficiently
	file1Count := 0
	file2Count := 0
	bothCount := 0
	
	for i := 0; i < len(commits); i++ {
		if fileOccurrences[file1][i] {
			file1Count++
		}
		if fileOccurrences[file2][i] {
			file2Count++
		}
		if fileOccurrences[file1][i] && fileOccurrences[file2][i] {
			bothCount++
		}
	}
	
	// Confidence = P(file2|file1) = count(file1 AND file2) / count(file1)
	if file1Count > file2Count {
		return float64(bothCount) / float64(file1Count)
	} else {
		return float64(bothCount) / float64(file2Count)
	}
}

// SimplePatternName generates a simple name for a pattern
func SimplePatternName(files []string) string {
	if len(files) == 0 {
		return "Empty Pattern"
	}
	
	// Extract base file names
	var names []string
	for _, file := range files {
		parts := strings.Split(file, "/")
		fileName := parts[len(parts)-1]
		
		// Remove extension
		if dotIdx := strings.LastIndex(fileName, "."); dotIdx > 0 {
			fileName = fileName[:dotIdx]
		}
		
		names = append(names, fileName)
	}
	
	sort.Strings(names)
	return strings.Join(names, " + ")
}