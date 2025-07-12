package diff

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

// CamelCaseHeuristic detects renames based on camelCase patterns
type CamelCaseHeuristic struct{}

func NewCamelCaseHeuristic() *CamelCaseHeuristic {
	return &CamelCaseHeuristic{}
}

func (cch *CamelCaseHeuristic) EvaluateRename(old, new *types.Symbol, context *RenameContext) HeuristicScore {
	oldName := old.Name
	newName := new.Name

	// Extract camelCase components
	oldComponents := cch.extractCamelCaseComponents(oldName)
	newComponents := cch.extractCamelCaseComponents(newName)

	// Check for common camelCase refactoring patterns
	score, reason := cch.analyzeCamelCasePattern(oldComponents, newComponents)

	return HeuristicScore{
		Score:      score,
		Confidence: cch.calculateConfidence(oldComponents, newComponents, score),
		Reason:     reason,
		Heuristic:  "camel_case",
	}
}

func (cch *CamelCaseHeuristic) extractCamelCaseComponents(name string) []string {
	var components []string
	var current strings.Builder

	for i, char := range name {
		if i > 0 && unicode.IsUpper(char) && !unicode.IsUpper(rune(name[i-1])) {
			if current.Len() > 0 {
				components = append(components, strings.ToLower(current.String()))
				current.Reset()
			}
		}
		current.WriteRune(char)
	}

	if current.Len() > 0 {
		components = append(components, strings.ToLower(current.String()))
	}

	return components
}

func (cch *CamelCaseHeuristic) analyzeCamelCasePattern(oldComponents, newComponents []string) (float64, string) {
	if len(oldComponents) == 0 || len(newComponents) == 0 {
		return 0.0, "No camelCase components found"
	}

	// Pattern: Added prefix/suffix
	if cch.hasCommonCore(oldComponents, newComponents) {
		if len(newComponents) > len(oldComponents) {
			return 0.8, "Added camelCase component(s)"
		} else if len(oldComponents) > len(newComponents) {
			return 0.8, "Removed camelCase component(s)"
		}
	}

	// Pattern: Component reordering
	if cch.hasSharedComponents(oldComponents, newComponents) {
		return 0.7, "Reordered camelCase components"
	}

	// Pattern: Component substitution
	substitutionScore := cch.calculateSubstitutionScore(oldComponents, newComponents)
	if substitutionScore > 0.6 {
		return substitutionScore, "Substituted camelCase component(s)"
	}

	return 0.0, "No clear camelCase pattern"
}

func (cch *CamelCaseHeuristic) hasCommonCore(oldComponents, newComponents []string) bool {
	// Check if one set is a subset of the other
	minLen := min(len(oldComponents), len(newComponents))
	if minLen == 0 {
		return false
	}

	commonCount := 0
	for _, oldComp := range oldComponents {
		for _, newComp := range newComponents {
			if oldComp == newComp {
				commonCount++
				break
			}
		}
	}

	return float64(commonCount)/float64(minLen) > 0.7
}

func (cch *CamelCaseHeuristic) hasSharedComponents(oldComponents, newComponents []string) bool {
	sharedCount := 0
	for _, oldComp := range oldComponents {
		for _, newComp := range newComponents {
			if oldComp == newComp {
				sharedCount++
				break
			}
		}
	}

	return sharedCount > 0 && float64(sharedCount) >= float64(min(len(oldComponents), len(newComponents)))*0.5
}

func (cch *CamelCaseHeuristic) calculateSubstitutionScore(oldComponents, newComponents []string) float64 {
	if len(oldComponents) != len(newComponents) {
		return 0.0
	}

	matches := 0
	for i := 0; i < len(oldComponents); i++ {
		if oldComponents[i] == newComponents[i] {
			matches++
		}
	}

	return float64(matches) / float64(len(oldComponents))
}

func (cch *CamelCaseHeuristic) calculateConfidence(oldComponents, newComponents []string, score float64) float64 {
	// Higher confidence for longer component lists
	avgLen := float64(len(oldComponents)+len(newComponents)) / 2.0
	lengthFactor := avgLen / 5.0 // Normalize
	if lengthFactor > 1.0 {
		lengthFactor = 1.0
	}

	return score*0.7 + lengthFactor*0.3
}

func (cch *CamelCaseHeuristic) GetWeight() float64 {
	return 0.8
}

func (cch *CamelCaseHeuristic) GetName() string {
	return "camel_case"
}

// PrefixSuffixHeuristic detects common prefix/suffix patterns
type PrefixSuffixHeuristic struct{}

func NewPrefixSuffixHeuristic() *PrefixSuffixHeuristic {
	return &PrefixSuffixHeuristic{}
}

func (psh *PrefixSuffixHeuristic) EvaluateRename(old, new *types.Symbol, context *RenameContext) HeuristicScore {
	oldName := old.Name
	newName := new.Name

	// Check for prefix patterns
	prefixScore, prefixReason := psh.analyzePrefixPattern(oldName, newName)

	// Check for suffix patterns
	suffixScore, suffixReason := psh.analyzeSuffixPattern(oldName, newName)

	// Take the higher score
	if prefixScore > suffixScore {
		return HeuristicScore{
			Score:      prefixScore,
			Confidence: 0.7,
			Reason:     prefixReason,
			Heuristic:  "prefix_suffix",
		}
	} else {
		return HeuristicScore{
			Score:      suffixScore,
			Confidence: 0.7,
			Reason:     suffixReason,
			Heuristic:  "prefix_suffix",
		}
	}
}

func (psh *PrefixSuffixHeuristic) analyzePrefixPattern(oldName, newName string) (float64, string) {
	// Common prefixes to check
	commonPrefixes := []string{"get", "set", "is", "has", "can", "should", "will", "new", "old", "temp", "tmp"}

	// Check if one name is the other with a prefix added/removed
	for _, prefix := range commonPrefixes {
		if strings.HasPrefix(newName, prefix) && strings.ToLower(newName[len(prefix):]) == strings.ToLower(oldName) {
			return 0.9, "Added prefix: " + prefix
		}
		if strings.HasPrefix(oldName, prefix) && strings.ToLower(oldName[len(prefix):]) == strings.ToLower(newName) {
			return 0.9, "Removed prefix: " + prefix
		}
	}

	// Check for custom prefix patterns
	if psh.hasCustomPrefixPattern(oldName, newName) {
		return 0.7, "Custom prefix pattern detected"
	}

	return 0.0, "No prefix pattern found"
}

func (psh *PrefixSuffixHeuristic) analyzeSuffixPattern(oldName, newName string) (float64, string) {
	// Common suffixes to check
	commonSuffixes := []string{"er", "ed", "ing", "ly", "tion", "sion", "ness", "ment", "able", "ible", "old", "new", "temp", "copy"}

	// Check if one name is the other with a suffix added/removed
	for _, suffix := range commonSuffixes {
		if strings.HasSuffix(newName, suffix) && strings.ToLower(newName[:len(newName)-len(suffix)]) == strings.ToLower(oldName) {
			return 0.9, "Added suffix: " + suffix
		}
		if strings.HasSuffix(oldName, suffix) && strings.ToLower(oldName[:len(oldName)-len(suffix)]) == strings.ToLower(newName) {
			return 0.9, "Removed suffix: " + suffix
		}
	}

	// Check for custom suffix patterns
	if psh.hasCustomSuffixPattern(oldName, newName) {
		return 0.7, "Custom suffix pattern detected"
	}

	return 0.0, "No suffix pattern found"
}

func (psh *PrefixSuffixHeuristic) hasCustomPrefixPattern(oldName, newName string) bool {
	// Look for patterns where a consistent prefix is added/removed
	if len(newName) > len(oldName) {
		return strings.HasSuffix(strings.ToLower(newName), strings.ToLower(oldName))
	} else if len(oldName) > len(newName) {
		return strings.HasSuffix(strings.ToLower(oldName), strings.ToLower(newName))
	}
	return false
}

func (psh *PrefixSuffixHeuristic) hasCustomSuffixPattern(oldName, newName string) bool {
	// Look for patterns where a consistent suffix is added/removed
	if len(newName) > len(oldName) {
		return strings.HasPrefix(strings.ToLower(newName), strings.ToLower(oldName))
	} else if len(oldName) > len(newName) {
		return strings.HasPrefix(strings.ToLower(oldName), strings.ToLower(newName))
	}
	return false
}

func (psh *PrefixSuffixHeuristic) GetWeight() float64 {
	return 0.7
}

func (psh *PrefixSuffixHeuristic) GetName() string {
	return "prefix_suffix"
}

// AbbreviationHeuristic detects abbreviation/expansion patterns
type AbbreviationHeuristic struct{}

func NewAbbreviationHeuristic() *AbbreviationHeuristic {
	return &AbbreviationHeuristic{}
}

func (ah *AbbreviationHeuristic) EvaluateRename(old, new *types.Symbol, context *RenameContext) HeuristicScore {
	oldName := old.Name
	newName := new.Name

	// Check for abbreviation patterns
	abbrevScore, abbrevReason := ah.analyzeAbbreviationPattern(oldName, newName)

	// Check for expansion patterns
	expansionScore, expansionReason := ah.analyzeExpansionPattern(oldName, newName)

	// Take the higher score
	if abbrevScore > expansionScore {
		return HeuristicScore{
			Score:      abbrevScore,
			Confidence: 0.8,
			Reason:     abbrevReason,
			Heuristic:  "abbreviation",
		}
	} else {
		return HeuristicScore{
			Score:      expansionScore,
			Confidence: 0.8,
			Reason:     expansionReason,
			Heuristic:  "abbreviation",
		}
	}
}

func (ah *AbbreviationHeuristic) analyzeAbbreviationPattern(oldName, newName string) (float64, string) {
	// Common abbreviation mappings
	abbreviations := map[string]string{
		"information":   "info",
		"configuration": "config",
		"initialize":    "init",
		"parameter":     "param",
		"temporary":     "temp",
		"maximum":       "max",
		"minimum":       "min",
		"calculate":     "calc",
		"generate":      "gen",
		"process":       "proc",
		"execute":       "exec",
		"document":      "doc",
		"reference":     "ref",
		"example":       "ex",
		"number":        "num",
		"string":        "str",
		"boolean":       "bool",
	}

	oldLower := strings.ToLower(oldName)
	newLower := strings.ToLower(newName)

	// Check if new name is abbreviation of old name
	for full, abbrev := range abbreviations {
		if strings.Contains(oldLower, full) && strings.Contains(newLower, abbrev) {
			return 0.9, "Abbreviated: " + full + " -> " + abbrev
		}
	}

	// Check for custom abbreviation patterns
	if ah.isCustomAbbreviation(oldName, newName) {
		return 0.7, "Custom abbreviation pattern"
	}

	return 0.0, "No abbreviation pattern found"
}

func (ah *AbbreviationHeuristic) analyzeExpansionPattern(oldName, newName string) (float64, string) {
	// Same abbreviations map but reversed
	abbreviations := map[string]string{
		"info":   "information",
		"config": "configuration",
		"init":   "initialize",
		"param":  "parameter",
		"temp":   "temporary",
		"max":    "maximum",
		"min":    "minimum",
		"calc":   "calculate",
		"gen":    "generate",
		"proc":   "process",
		"exec":   "execute",
		"doc":    "document",
		"ref":    "reference",
		"ex":     "example",
		"num":    "number",
		"str":    "string",
		"bool":   "boolean",
	}

	oldLower := strings.ToLower(oldName)
	newLower := strings.ToLower(newName)

	// Check if new name is expansion of old name
	for abbrev, full := range abbreviations {
		if strings.Contains(oldLower, abbrev) && strings.Contains(newLower, full) {
			return 0.9, "Expanded: " + abbrev + " -> " + full
		}
	}

	// Check for custom expansion patterns
	if ah.isCustomExpansion(oldName, newName) {
		return 0.7, "Custom expansion pattern"
	}

	return 0.0, "No expansion pattern found"
}

func (ah *AbbreviationHeuristic) isCustomAbbreviation(oldName, newName string) bool {
	// Simple heuristic: new name is much shorter and contains similar letters
	if float64(len(newName)) < float64(len(oldName))*0.6 && len(newName) >= 2 {
		return ah.hasSignificantLetterOverlap(oldName, newName)
	}
	return false
}

func (ah *AbbreviationHeuristic) isCustomExpansion(oldName, newName string) bool {
	// Simple heuristic: new name is much longer and contains similar letters
	if float64(len(newName)) > float64(len(oldName))*1.5 && len(oldName) >= 2 {
		return ah.hasSignificantLetterOverlap(oldName, newName)
	}
	return false
}

func (ah *AbbreviationHeuristic) hasSignificantLetterOverlap(str1, str2 string) bool {
	// Check if significant portion of letters overlap
	str1Lower := strings.ToLower(str1)
	str2Lower := strings.ToLower(str2)

	matchCount := 0
	for _, char := range str1Lower {
		if strings.ContainsRune(str2Lower, char) {
			matchCount++
		}
	}

	return float64(matchCount)/float64(len(str1)) > 0.6
}

func (ah *AbbreviationHeuristic) GetWeight() float64 {
	return 0.8
}

func (ah *AbbreviationHeuristic) GetName() string {
	return "abbreviation"
}

// RefactoringPatternHeuristic detects common refactoring patterns
type RefactoringPatternHeuristic struct{}

func NewRefactoringPatternHeuristic() *RefactoringPatternHeuristic {
	return &RefactoringPatternHeuristic{}
}

func (rph *RefactoringPatternHeuristic) EvaluateRename(old, new *types.Symbol, context *RenameContext) HeuristicScore {
	// Check for common refactoring patterns
	patterns := []func(*types.Symbol, *types.Symbol, *RenameContext) (float64, string){
		rph.detectExtractMethod,
		rph.detectRenameForClarity,
		rph.detectConventionAlignment,
		rph.detectScopeChange,
	}

	bestScore := 0.0
	bestReason := "No refactoring pattern detected"

	for _, pattern := range patterns {
		score, reason := pattern(old, new, context)
		if score > bestScore {
			bestScore = score
			bestReason = reason
		}
	}

	return HeuristicScore{
		Score:      bestScore,
		Confidence: 0.7,
		Reason:     bestReason,
		Heuristic:  "refactoring_pattern",
	}
}

func (rph *RefactoringPatternHeuristic) detectExtractMethod(old, new *types.Symbol, context *RenameContext) (float64, string) {
	// Look for patterns where a method was extracted and renamed
	if old.Kind == "function" && new.Kind == "function" {
		// Check if new method is smaller (extracted from larger method)
		oldSize := old.Location.EndLine - old.Location.StartLine
		newSize := new.Location.EndLine - new.Location.StartLine

		if newSize < int(float64(oldSize)*0.7) && newSize > 0 {
			return 0.8, "Possible extract method refactoring"
		}
	}

	return 0.0, ""
}

func (rph *RefactoringPatternHeuristic) detectRenameForClarity(old, new *types.Symbol, context *RenameContext) (float64, string) {
	// Look for patterns that suggest renaming for clarity
	clarityIndicators := []string{
		"clear", "clean", "readable", "understandable", "explicit",
		"descriptive", "meaningful", "better", "improved",
	}

	oldLower := strings.ToLower(old.Name)
	newLower := strings.ToLower(new.Name)

	// Check if new name contains clarity indicators
	for _, indicator := range clarityIndicators {
		if strings.Contains(newLower, indicator) && !strings.Contains(oldLower, indicator) {
			return 0.6, "Renamed for clarity: " + indicator
		}
	}

	// Check if new name is more descriptive (longer with meaningful words)
	if float64(len(new.Name)) > float64(len(old.Name))*1.3 && rph.hasMoreMeaningfulWords(new.Name, old.Name) {
		return 0.7, "Made name more descriptive"
	}

	return 0.0, ""
}

func (rph *RefactoringPatternHeuristic) detectConventionAlignment(old, new *types.Symbol, context *RenameContext) (float64, string) {
	// Check for alignment with naming conventions
	conventions := map[string]*regexp.Regexp{
		"camelCase":  regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`),
		"PascalCase": regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`),
		"snake_case": regexp.MustCompile(`^[a-z]+(_[a-z0-9]+)*$`),
		"CONSTANT":   regexp.MustCompile(`^[A-Z]+(_[A-Z0-9]+)*$`),
	}

	oldName := old.Name
	newName := new.Name

	// Check if new name follows conventions better than old name
	for conventionName, pattern := range conventions {
		oldMatches := pattern.MatchString(oldName)
		newMatches := pattern.MatchString(newName)

		if !oldMatches && newMatches {
			return 0.8, "Aligned with " + conventionName + " convention"
		}
	}

	return 0.0, ""
}

func (rph *RefactoringPatternHeuristic) detectScopeChange(old, new *types.Symbol, context *RenameContext) (float64, string) {
	// Check if visibility changed (scope refactoring)
	if old.Visibility != new.Visibility {
		if old.Visibility == "private" && new.Visibility == "public" {
			return 0.6, "Made public (scope expansion)"
		}
		if old.Visibility == "public" && new.Visibility == "private" {
			return 0.6, "Made private (scope restriction)"
		}
	}

	return 0.0, ""
}

func (rph *RefactoringPatternHeuristic) hasMoreMeaningfulWords(newName, oldName string) bool {
	// Simple heuristic: count capital letters (indicating word boundaries in camelCase)
	newCaps := 0
	oldCaps := 0

	for _, char := range newName {
		if unicode.IsUpper(char) {
			newCaps++
		}
	}

	for _, char := range oldName {
		if unicode.IsUpper(char) {
			oldCaps++
		}
	}

	return newCaps > oldCaps
}

func (rph *RefactoringPatternHeuristic) GetWeight() float64 {
	return 0.9
}

func (rph *RefactoringPatternHeuristic) GetName() string {
	return "refactoring_pattern"
}

// ContextualHeuristic uses file and symbol context for rename detection
type ContextualHeuristic struct{}

func NewContextualHeuristic() *ContextualHeuristic {
	return &ContextualHeuristic{}
}

func (ch *ContextualHeuristic) EvaluateRename(old, new *types.Symbol, context *RenameContext) HeuristicScore {
	// Analyze contextual clues
	contextScore := 0.0
	reasons := []string{}

	// Check location proximity
	locationScore := ch.analyzeLocationProximity(old, new)
	if locationScore > 0.5 {
		contextScore += locationScore * 0.3
		reasons = append(reasons, "nearby location")
	}

	// Check usage patterns
	usageScore := ch.analyzeUsagePatterns(old, new, context)
	if usageScore > 0.5 {
		contextScore += usageScore * 0.4
		reasons = append(reasons, "similar usage pattern")
	}

	// Check sibling symbols
	siblingScore := ch.analyzeSiblingSymbols(old, new, context)
	if siblingScore > 0.5 {
		contextScore += siblingScore * 0.3
		reasons = append(reasons, "consistent with siblings")
	}

	reason := "Contextual analysis"
	if len(reasons) > 0 {
		reason = "Contextual: " + strings.Join(reasons, ", ")
	}

	return HeuristicScore{
		Score:      contextScore,
		Confidence: 0.6,
		Reason:     reason,
		Heuristic:  "contextual",
	}
}

func (ch *ContextualHeuristic) analyzeLocationProximity(old, new *types.Symbol) float64 {
	// Symbols closer together are more likely to be renames
	lineDiff := abs(old.Location.StartLine - new.Location.StartLine)

	if lineDiff == 0 {
		return 1.0
	} else if lineDiff <= 5 {
		return 0.8
	} else if lineDiff <= 20 {
		return 0.5
	} else if lineDiff <= 50 {
		return 0.2
	} else {
		return 0.0
	}
}

func (ch *ContextualHeuristic) analyzeUsagePatterns(old, new *types.Symbol, context *RenameContext) float64 {
	// This would analyze how symbols are used in the code
	// Placeholder for more sophisticated analysis
	return 0.5
}

func (ch *ContextualHeuristic) analyzeSiblingSymbols(old, new *types.Symbol, context *RenameContext) float64 {
	// Look for patterns in how other symbols in the same file were renamed
	// Placeholder for more sophisticated analysis
	return 0.5
}

func (ch *ContextualHeuristic) GetWeight() float64 {
	return 0.6
}

func (ch *ContextualHeuristic) GetName() string {
	return "contextual"
}

// Utility functions moved to utils.go
