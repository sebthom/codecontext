package diff

import (
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

// NameSimilarityAlgorithm compares symbol names using various string similarity metrics
type NameSimilarityAlgorithm struct{}

func NewNameSimilarityAlgorithm() *NameSimilarityAlgorithm {
	return &NameSimilarityAlgorithm{}
}

func (nsa *NameSimilarityAlgorithm) CalculateSimilarity(old, new *types.Symbol) SimilarityScore {
	oldName := old.Name
	newName := new.Name

	// Multiple similarity metrics
	editDistance := nsa.normalizedEditDistance(oldName, newName)
	jaccardSim := nsa.jaccardSimilarity(oldName, newName)
	substringScore := nsa.substringScore(oldName, newName)
	camelCaseScore := nsa.camelCaseSimilarity(oldName, newName)

	// Weighted combination
	score := (editDistance*0.3 + jaccardSim*0.3 + substringScore*0.2 + camelCaseScore*0.2)

	evidence := ""
	if score > 0.8 {
		evidence = "High name similarity"
	} else if score > 0.6 {
		evidence = "Moderate name similarity"
	} else {
		evidence = "Low name similarity"
	}

	return SimilarityScore{
		Score:      score,
		Confidence: nsa.calculateConfidence(oldName, newName, score),
		Evidence:   evidence,
		Algorithm:  "name_similarity",
	}
}

func (nsa *NameSimilarityAlgorithm) normalizedEditDistance(s1, s2 string) float64 {
	distance := nsa.levenshteinDistance(s1, s2)
	maxLen := max(len(s1), len(s2))
	if maxLen == 0 {
		return 1.0
	}
	return 1.0 - float64(distance)/float64(maxLen)
}

func (nsa *NameSimilarityAlgorithm) jaccardSimilarity(s1, s2 string) float64 {
	// Character-level Jaccard similarity
	set1 := make(map[rune]bool)
	set2 := make(map[rune]bool)

	for _, char := range s1 {
		set1[char] = true
	}
	for _, char := range s2 {
		set2[char] = true
	}

	intersection := 0
	union := len(set1)

	for char := range set2 {
		if set1[char] {
			intersection++
		} else {
			union++
		}
	}

	if union == 0 {
		return 1.0
	}
	return float64(intersection) / float64(union)
}

func (nsa *NameSimilarityAlgorithm) substringScore(s1, s2 string) float64 {
	// Longest common substring
	lcs := nsa.longestCommonSubstring(s1, s2)
	maxLen := max(len(s1), len(s2))
	if maxLen == 0 {
		return 1.0
	}
	return float64(lcs) / float64(maxLen)
}

func (nsa *NameSimilarityAlgorithm) camelCaseSimilarity(s1, s2 string) float64 {
	// Extract camelCase components
	components1 := nsa.extractCamelCaseComponents(s1)
	components2 := nsa.extractCamelCaseComponents(s2)

	return nsa.sequenceSimilarity(components1, components2)
}

func (nsa *NameSimilarityAlgorithm) calculateConfidence(s1, s2 string, score float64) float64 {
	// Higher confidence for longer strings with high similarity
	avgLen := float64(len(s1)+len(s2)) / 2.0
	lengthFactor := math.Min(avgLen/10.0, 1.0) // Normalize to 0-1

	return score*0.7 + lengthFactor*0.3
}

func (nsa *NameSimilarityAlgorithm) GetWeight() float64 {
	return 1.0
}

func (nsa *NameSimilarityAlgorithm) GetName() string {
	return "name_similarity"
}

// SignatureSimilarityAlgorithm compares function/method signatures
type SignatureSimilarityAlgorithm struct{}

func NewSignatureSimilarityAlgorithm() *SignatureSimilarityAlgorithm {
	return &SignatureSimilarityAlgorithm{}
}

func (ssa *SignatureSimilarityAlgorithm) CalculateSimilarity(old, new *types.Symbol) SimilarityScore {
	oldSig := old.Signature
	newSig := new.Signature

	if oldSig == "" && newSig == "" {
		return SimilarityScore{
			Score:      1.0,
			Confidence: 0.5, // Low confidence due to lack of data
			Evidence:   "Both signatures empty",
			Algorithm:  "signature_similarity",
		}
	}

	if oldSig == "" || newSig == "" {
		return SimilarityScore{
			Score:      0.0,
			Confidence: 0.8,
			Evidence:   "One signature missing",
			Algorithm:  "signature_similarity",
		}
	}

	// Compare parameter counts
	oldParams := ssa.extractParameters(oldSig)
	newParams := ssa.extractParameters(newSig)
	paramScore := ssa.compareParameterLists(oldParams, newParams)

	// Compare return types
	oldReturn := ssa.extractReturnType(oldSig)
	newReturn := ssa.extractReturnType(newSig)
	returnScore := ssa.compareReturnTypes(oldReturn, newReturn)

	// Overall signature similarity
	score := paramScore*0.7 + returnScore*0.3

	evidence := fmt.Sprintf("Parameter similarity: %.2f, Return type similarity: %.2f", paramScore, returnScore)

	return SimilarityScore{
		Score:      score,
		Confidence: ssa.calculateConfidence(oldSig, newSig, score),
		Evidence:   evidence,
		Algorithm:  "signature_similarity",
	}
}

func (ssa *SignatureSimilarityAlgorithm) GetWeight() float64 {
	return 1.2 // Higher weight for signature similarity
}

func (ssa *SignatureSimilarityAlgorithm) GetName() string {
	return "signature_similarity"
}

// StructuralSimilarityAlgorithm compares structural aspects of symbols
type StructuralSimilarityAlgorithm struct{}

func NewStructuralSimilarityAlgorithm() *StructuralSimilarityAlgorithm {
	return &StructuralSimilarityAlgorithm{}
}

func (ssa *StructuralSimilarityAlgorithm) CalculateSimilarity(old, new *types.Symbol) SimilarityScore {
	// Compare symbol kinds
	kindScore := ssa.compareKinds(old.Kind, new.Kind)

	// Compare visibility
	visibilityScore := ssa.compareVisibility(old.Visibility, new.Visibility)

	// Compare sizes (lines of code)
	sizeScore := ssa.compareSizes(old, new)

	// Overall structural similarity
	score := kindScore*0.4 + visibilityScore*0.3 + sizeScore*0.3

	var evidence string
	if kindScore == 1.0 && visibilityScore == 1.0 {
		evidence = "Same kind and visibility"
	} else if kindScore == 1.0 {
		evidence = "Same kind, different visibility"
	} else {
		evidence = "Different structural properties"
	}

	return SimilarityScore{
		Score:      score,
		Confidence: ssa.calculateConfidence(old, new, score),
		Evidence:   evidence,
		Algorithm:  "structural_similarity",
	}
}

func (ssa *StructuralSimilarityAlgorithm) GetWeight() float64 {
	return 0.8
}

func (ssa *StructuralSimilarityAlgorithm) GetName() string {
	return "structural_similarity"
}

// LocationSimilarityAlgorithm compares symbol locations
type LocationSimilarityAlgorithm struct{}

func NewLocationSimilarityAlgorithm() *LocationSimilarityAlgorithm {
	return &LocationSimilarityAlgorithm{}
}

func (lsa *LocationSimilarityAlgorithm) CalculateSimilarity(old, new *types.Symbol) SimilarityScore {
	oldLine := old.Location.StartLine
	newLine := new.Location.StartLine

	lineDiff := abs(oldLine - newLine)

	// Exponential decay for line distance
	score := math.Exp(-float64(lineDiff) / 20.0)

	var evidence string
	if lineDiff == 0 {
		evidence = "Same line"
	} else if lineDiff <= 5 {
		evidence = "Very close location"
	} else if lineDiff <= 20 {
		evidence = "Close location"
	} else {
		evidence = "Distant location"
	}

	return SimilarityScore{
		Score:      score,
		Confidence: 0.6, // Location is a weak signal
		Evidence:   evidence,
		Algorithm:  "location_similarity",
	}
}

func (lsa *LocationSimilarityAlgorithm) GetWeight() float64 {
	return 0.5 // Lower weight for location
}

func (lsa *LocationSimilarityAlgorithm) GetName() string {
	return "location_similarity"
}

// DocumentationSimilarityAlgorithm compares documentation/comments
type DocumentationSimilarityAlgorithm struct{}

func NewDocumentationSimilarityAlgorithm() *DocumentationSimilarityAlgorithm {
	return &DocumentationSimilarityAlgorithm{}
}

func (dsa *DocumentationSimilarityAlgorithm) CalculateSimilarity(old, new *types.Symbol) SimilarityScore {
	oldDoc := old.Documentation
	newDoc := new.Documentation

	if oldDoc == "" && newDoc == "" {
		return SimilarityScore{
			Score:      0.5, // Neutral score for missing documentation
			Confidence: 0.3,
			Evidence:   "No documentation available",
			Algorithm:  "documentation_similarity",
		}
	}

	if oldDoc == "" || newDoc == "" {
		return SimilarityScore{
			Score:      0.0,
			Confidence: 0.5,
			Evidence:   "Documentation missing in one version",
			Algorithm:  "documentation_similarity",
		}
	}

	// Text similarity using multiple metrics
	editDistance := dsa.normalizedEditDistance(oldDoc, newDoc)
	wordSimilarity := dsa.wordSimilarity(oldDoc, newDoc)
	conceptSimilarity := dsa.conceptSimilarity(oldDoc, newDoc)

	score := (editDistance + wordSimilarity + conceptSimilarity) / 3.0

	evidence := fmt.Sprintf("Documentation similarity: %.2f", score)

	return SimilarityScore{
		Score:      score,
		Confidence: dsa.calculateConfidence(oldDoc, newDoc, score),
		Evidence:   evidence,
		Algorithm:  "documentation_similarity",
	}
}

func (dsa *DocumentationSimilarityAlgorithm) GetWeight() float64 {
	return 0.6
}

func (dsa *DocumentationSimilarityAlgorithm) GetName() string {
	return "documentation_similarity"
}

// SemanticSimilarityAlgorithm performs semantic analysis
type SemanticSimilarityAlgorithm struct{}

func NewSemanticSimilarityAlgorithm() *SemanticSimilarityAlgorithm {
	return &SemanticSimilarityAlgorithm{}
}

func (ssa *SemanticSimilarityAlgorithm) CalculateSimilarity(old, new *types.Symbol) SimilarityScore {
	// Semantic analysis based on context and relationships
	contextScore := ssa.analyzeContext(old, new)
	relationshipScore := ssa.analyzeRelationships(old, new)
	purposeScore := ssa.analyzePurpose(old, new)

	score := (contextScore + relationshipScore + purposeScore) / 3.0

	evidence := "Semantic analysis of symbol purpose and context"

	return SimilarityScore{
		Score:      score,
		Confidence: 0.7,
		Evidence:   evidence,
		Algorithm:  "semantic_similarity",
	}
}

func (ssa *SemanticSimilarityAlgorithm) GetWeight() float64 {
	return 0.9
}

func (ssa *SemanticSimilarityAlgorithm) GetName() string {
	return "semantic_similarity"
}

// Helper methods for similarity algorithms

func (nsa *NameSimilarityAlgorithm) levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}

	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func (nsa *NameSimilarityAlgorithm) longestCommonSubstring(s1, s2 string) int {
	if len(s1) == 0 || len(s2) == 0 {
		return 0
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	longest := 0
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			if s1[i-1] == s2[j-1] {
				matrix[i][j] = matrix[i-1][j-1] + 1
				if matrix[i][j] > longest {
					longest = matrix[i][j]
				}
			}
		}
	}

	return longest
}

func (nsa *NameSimilarityAlgorithm) extractCamelCaseComponents(s string) []string {
	var components []string
	var current strings.Builder

	for i, char := range s {
		if i > 0 && unicode.IsUpper(char) && !unicode.IsUpper(rune(s[i-1])) {
			if current.Len() > 0 {
				components = append(components, current.String())
				current.Reset()
			}
		}
		current.WriteRune(unicode.ToLower(char))
	}

	if current.Len() > 0 {
		components = append(components, current.String())
	}

	return components
}

func (nsa *NameSimilarityAlgorithm) sequenceSimilarity(seq1, seq2 []string) float64 {
	if len(seq1) == 0 && len(seq2) == 0 {
		return 1.0
	}
	if len(seq1) == 0 || len(seq2) == 0 {
		return 0.0
	}

	// Calculate sequence alignment score
	matches := 0
	minLen := min(len(seq1), len(seq2))

	for i := 0; i < minLen; i++ {
		if seq1[i] == seq2[i] {
			matches++
		}
	}

	return float64(matches) / float64(max(len(seq1), len(seq2)))
}

// Signature similarity helper methods

func (ssa *SignatureSimilarityAlgorithm) extractParameters(signature string) []string {
	// Simple parameter extraction (language-agnostic)
	if !strings.Contains(signature, "(") {
		return []string{}
	}

	start := strings.Index(signature, "(")
	end := strings.LastIndex(signature, ")")
	if start >= end {
		return []string{}
	}

	paramStr := strings.TrimSpace(signature[start+1 : end])
	if paramStr == "" {
		return []string{}
	}

	params := strings.Split(paramStr, ",")
	for i, param := range params {
		params[i] = strings.TrimSpace(param)
	}

	return params
}

func (ssa *SignatureSimilarityAlgorithm) extractReturnType(signature string) string {
	// Simple return type extraction
	if strings.Contains(signature, "->") {
		parts := strings.Split(signature, "->")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}

	if strings.Contains(signature, ":") && !strings.Contains(signature, "(") {
		parts := strings.Split(signature, ":")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}

	return "unknown"
}

func (ssa *SignatureSimilarityAlgorithm) compareParameterLists(params1, params2 []string) float64 {
	if len(params1) == 0 && len(params2) == 0 {
		return 1.0
	}

	if len(params1) != len(params2) {
		// Penalize different parameter counts
		return 0.5 * float64(min(len(params1), len(params2))) / float64(max(len(params1), len(params2)))
	}

	matches := 0
	for i := 0; i < len(params1); i++ {
		if ssa.compareParameterTypes(params1[i], params2[i]) {
			matches++
		}
	}

	return float64(matches) / float64(len(params1))
}

func (ssa *SignatureSimilarityAlgorithm) compareParameterTypes(param1, param2 string) bool {
	// Extract type from parameter (remove variable names)
	type1 := ssa.extractType(param1)
	type2 := ssa.extractType(param2)

	return type1 == type2
}

func (ssa *SignatureSimilarityAlgorithm) extractType(param string) string {
	// Simple type extraction (would need language-specific logic)
	parts := strings.Fields(param)
	if len(parts) > 0 {
		return parts[0] // First word is often the type
	}
	return param
}

func (ssa *SignatureSimilarityAlgorithm) compareReturnTypes(type1, type2 string) float64 {
	if type1 == type2 {
		return 1.0
	}
	if type1 == "unknown" || type2 == "unknown" {
		return 0.5
	}
	return 0.0
}

func (ssa *SignatureSimilarityAlgorithm) calculateConfidence(oldSig, newSig string, score float64) float64 {
	// Higher confidence for more detailed signatures
	avgLen := float64(len(oldSig)+len(newSig)) / 2.0
	lengthFactor := math.Min(avgLen/50.0, 1.0)

	return score*0.8 + lengthFactor*0.2
}

// Structural similarity helper methods

func (ssa *StructuralSimilarityAlgorithm) compareKinds(kind1, kind2 string) float64 {
	if kind1 == kind2 {
		return 1.0
	}

	// Some kinds are similar (e.g., function vs method)
	similarKinds := map[string][]string{
		"function":  {"method"},
		"method":    {"function"},
		"class":     {"interface"},
		"interface": {"class"},
	}

	if similar, exists := similarKinds[kind1]; exists {
		for _, s := range similar {
			if s == kind2 {
				return 0.8
			}
		}
	}

	return 0.0
}

func (ssa *StructuralSimilarityAlgorithm) compareVisibility(vis1, vis2 string) float64 {
	if vis1 == vis2 {
		return 1.0
	}

	// Visibility changes are somewhat similar
	visibilityMap := map[string]int{
		"private":   0,
		"protected": 1,
		"public":    2,
	}

	v1, ok1 := visibilityMap[vis1]
	v2, ok2 := visibilityMap[vis2]

	if ok1 && ok2 {
		diff := abs(v1 - v2)
		return 1.0 - float64(diff)/2.0
	}

	return 0.5
}

func (ssa *StructuralSimilarityAlgorithm) compareSizes(old, new *types.Symbol) float64 {
	oldSize := old.Location.EndLine - old.Location.StartLine + 1
	newSize := new.Location.EndLine - new.Location.StartLine + 1

	sizeDiff := abs(oldSize - newSize)
	maxSize := max(oldSize, newSize)

	if maxSize == 0 {
		return 1.0
	}

	return 1.0 - float64(sizeDiff)/float64(maxSize)
}

func (ssa *StructuralSimilarityAlgorithm) calculateConfidence(old, new *types.Symbol, score float64) float64 {
	// Higher confidence for symbols with more structural information
	confidence := 0.5

	if old.Kind != "" && new.Kind != "" {
		confidence += 0.2
	}
	if old.Visibility != "" && new.Visibility != "" {
		confidence += 0.2
	}
	if old.Signature != "" && new.Signature != "" {
		confidence += 0.1
	}

	return confidence
}

// Documentation similarity helper methods

func (dsa *DocumentationSimilarityAlgorithm) normalizedEditDistance(s1, s2 string) float64 {
	// Reuse the name similarity algorithm's implementation
	nsa := NewNameSimilarityAlgorithm()
	return nsa.normalizedEditDistance(s1, s2)
}

func (dsa *DocumentationSimilarityAlgorithm) wordSimilarity(doc1, doc2 string) float64 {
	words1 := strings.Fields(strings.ToLower(doc1))
	words2 := strings.Fields(strings.ToLower(doc2))

	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Word-level Jaccard similarity
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, word := range words1 {
		set1[word] = true
	}
	for _, word := range words2 {
		set2[word] = true
	}

	intersection := 0
	union := len(set1)

	for word := range set2 {
		if set1[word] {
			intersection++
		} else {
			union++
		}
	}

	return float64(intersection) / float64(union)
}

func (dsa *DocumentationSimilarityAlgorithm) conceptSimilarity(doc1, doc2 string) float64 {
	// Extract key concepts (simplified)
	concepts1 := dsa.extractConcepts(doc1)
	concepts2 := dsa.extractConcepts(doc2)

	if len(concepts1) == 0 && len(concepts2) == 0 {
		return 1.0
	}
	if len(concepts1) == 0 || len(concepts2) == 0 {
		return 0.0
	}

	matches := 0
	for _, concept := range concepts1 {
		for _, otherConcept := range concepts2 {
			if concept == otherConcept {
				matches++
				break
			}
		}
	}

	return float64(matches) / float64(max(len(concepts1), len(concepts2)))
}

func (dsa *DocumentationSimilarityAlgorithm) extractConcepts(doc string) []string {
	// Simple concept extraction (keywords)
	words := strings.Fields(strings.ToLower(doc))
	var concepts []string

	// Filter for meaningful words (length > 3, not common words)
	commonWords := map[string]bool{
		"the": true, "and": true, "that": true, "this": true,
		"with": true, "for": true, "are": true, "will": true,
	}

	for _, word := range words {
		if len(word) > 3 && !commonWords[word] {
			concepts = append(concepts, word)
		}
	}

	return concepts
}

func (dsa *DocumentationSimilarityAlgorithm) calculateConfidence(doc1, doc2 string, score float64) float64 {
	// Higher confidence for longer documentation
	avgLen := float64(len(doc1)+len(doc2)) / 2.0
	lengthFactor := math.Min(avgLen/100.0, 1.0)

	return score*0.7 + lengthFactor*0.3
}

// Semantic similarity helper methods

func (ssa *SemanticSimilarityAlgorithm) analyzeContext(old, new *types.Symbol) float64 {
	// Compare the context in which symbols appear
	// This is a placeholder for more sophisticated semantic analysis
	return 0.5
}

func (ssa *SemanticSimilarityAlgorithm) analyzeRelationships(old, new *types.Symbol) float64 {
	// Analyze relationships with other symbols
	// This is a placeholder for dependency and call graph analysis
	return 0.5
}

func (ssa *SemanticSimilarityAlgorithm) analyzePurpose(old, new *types.Symbol) float64 {
	// Analyze the purpose/functionality of symbols
	// This could involve more sophisticated NLP or pattern matching
	return 0.5
}

// Utility functions moved to utils.go
