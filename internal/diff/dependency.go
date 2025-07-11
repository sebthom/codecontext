package diff

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/nuthan-ms/codecontext/pkg/types"
)

// DependencyTracker tracks changes in import dependencies and file relationships
type DependencyTracker struct {
	config             *Config
	languageDetectors  map[string]DependencyDetector
	importPatterns     map[string]*regexp.Regexp
	dependencyCache    map[string][]Dependency
}

// DependencyDetector provides language-specific dependency detection
type DependencyDetector interface {
	ExtractDependencies(file *types.FileInfo) ([]Dependency, error)
	ParseImportStatement(line string) (*Import, error)
	GetImportKeywords() []string
	IsRelativeImport(importPath string) bool
	NormalizeImportPath(importPath string) string
}

// Dependency represents a dependency relationship
type Dependency struct {
	Type         DependencyType `json:"type"`
	Source       string         `json:"source"`        // File that has the dependency
	Target       string         `json:"target"`        // Dependency target (file, module, package)
	ImportPath   string         `json:"import_path"`   // Raw import path
	Alias        string         `json:"alias"`         // Import alias if any
	IsExternal   bool           `json:"is_external"`   // External vs internal dependency
	IsRelative   bool           `json:"is_relative"`   // Relative vs absolute import
	Line         int            `json:"line"`          // Line number of import
	Kind         ImportKind     `json:"kind"`          // Type of import
	Metadata     map[string]interface{} `json:"metadata"`
}

// Import represents an import statement
type Import struct {
	Path         string            `json:"path"`
	Alias        string            `json:"alias"`
	Symbols      []string          `json:"symbols"`      // Named imports
	IsDefault    bool              `json:"is_default"`
	IsNamespace  bool              `json:"is_namespace"` // Import * as name
	IsRelative   bool              `json:"is_relative"`
	Line         int               `json:"line"`
	Column       int               `json:"column"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// DependencyChange represents a change in dependencies
type DependencyChange struct {
	Type        DependencyChangeType `json:"type"`
	Dependency  *Dependency          `json:"dependency"`
	OldDep      *Dependency          `json:"old_dependency,omitempty"`
	Impact      ImpactLevel          `json:"impact"`
	Reason      string               `json:"reason"`
	Suggestions []string             `json:"suggestions"`
	Context     DependencyContext    `json:"context"`
}

// DependencyType categorizes dependency relationships
type DependencyType string

const (
	DependencyTypeImport    DependencyType = "import"
	DependencyTypeRequire   DependencyType = "require"
	DependencyTypeInclude   DependencyType = "include"
	DependencyTypeUsing     DependencyType = "using"
	DependencyTypeFrom      DependencyType = "from"
	DependencyTypeInherit   DependencyType = "inherit"
	DependencyTypeReference DependencyType = "reference"
)

// ImportKind categorizes types of imports
type ImportKind string

const (
	ImportKindDefault   ImportKind = "default"
	ImportKindNamed     ImportKind = "named"
	ImportKindNamespace ImportKind = "namespace"
	ImportKindSideEffect ImportKind = "side_effect"
	ImportKindDynamic   ImportKind = "dynamic"
)

// DependencyChangeType represents types of dependency changes
type DependencyChangeType string

const (
	DependencyChangeAdd    DependencyChangeType = "add"
	DependencyChangeRemove DependencyChangeType = "remove"
	DependencyChangeModify DependencyChangeType = "modify"
	DependencyChangeMove   DependencyChangeType = "move"
)

// DependencyContext provides context for dependency changes
type DependencyContext struct {
	AffectedFiles    []string `json:"affected_files"`
	CircularRefs     []string `json:"circular_refs"`
	UnusedImports    []string `json:"unused_imports"`
	MissingImports   []string `json:"missing_imports"`
	ExternalPackages []string `json:"external_packages"`
	SecurityRisks    []string `json:"security_risks"`
}

// NewDependencyTracker creates a new dependency tracker
func NewDependencyTracker(config *Config) *DependencyTracker {
	tracker := &DependencyTracker{
		config:            config,
		languageDetectors: make(map[string]DependencyDetector),
		importPatterns:    make(map[string]*regexp.Regexp),
		dependencyCache:   make(map[string][]Dependency),
	}

	// Register language-specific detectors
	tracker.registerLanguageDetectors()
	
	// Initialize import patterns
	tracker.initializeImportPatterns()

	return tracker
}

// TrackDependencyChanges analyzes dependency changes between two files
func (dt *DependencyTracker) TrackDependencyChanges(ctx context.Context, oldFile, newFile *types.FileInfo) ([]Change, error) {
	var changes []Change

	// Extract dependencies from both files
	oldDeps, err := dt.extractDependencies(oldFile)
	if err != nil {
		return nil, fmt.Errorf("failed to extract old dependencies: %w", err)
	}

	newDeps, err := dt.extractDependencies(newFile)
	if err != nil {
		return nil, fmt.Errorf("failed to extract new dependencies: %w", err)
	}

	// Analyze dependency changes
	depChanges := dt.analyzeDependencyChanges(oldDeps, newDeps)

	// Convert to generic changes
	for _, depChange := range depChanges {
		change := dt.convertDependencyChange(depChange, newFile)
		changes = append(changes, change)
	}

	// Analyze import order changes
	orderChanges := dt.analyzeImportOrderChanges(oldDeps, newDeps)
	changes = append(changes, orderChanges...)

	// Analyze circular dependency risks
	circularChanges := dt.analyzeCircularDependencies(newDeps, newFile)
	changes = append(changes, circularChanges...)

	return changes, nil
}

// registerLanguageDetectors registers language-specific dependency detectors
func (dt *DependencyTracker) registerLanguageDetectors() {
	dt.languageDetectors["javascript"] = NewJavaScriptDependencyDetector()
	dt.languageDetectors["typescript"] = NewTypeScriptDependencyDetector()
	dt.languageDetectors["go"] = NewGoDependencyDetector()
	dt.languageDetectors["python"] = NewPythonDependencyDetector()
	dt.languageDetectors["java"] = NewJavaDependencyDetector()
	dt.languageDetectors["csharp"] = NewCSharpDependencyDetector()
}

// initializeImportPatterns initializes regex patterns for import detection
func (dt *DependencyTracker) initializeImportPatterns() {
	dt.importPatterns["javascript"] = regexp.MustCompile(`(?m)^(?:import|export)(?:\s+(?:\{[^}]*\}|\*\s+as\s+\w+|\w+)(?:\s*,\s*(?:\{[^}]*\}|\*\s+as\s+\w+|\w+))*)?\s*from\s*['"]([^'"]+)['"]|^(?:const|let|var)\s+.*?=\s*require\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	dt.importPatterns["python"] = regexp.MustCompile(`(?m)^(?:from\s+(\S+)\s+import\s+.*|import\s+(\S+)(?:\s+as\s+\w+)?)`)
	dt.importPatterns["go"] = regexp.MustCompile(`(?m)^import\s*(?:\(\s*((?:[^)]*\n?)*)\s*\)|"([^"]+)"|(\w+)\s+"([^"]+)")`)
	dt.importPatterns["java"] = regexp.MustCompile(`(?m)^import\s+(?:static\s+)?([^;]+);`)
	dt.importPatterns["csharp"] = regexp.MustCompile(`(?m)^using\s+(?:static\s+)?([^;]+);`)
}

// extractDependencies extracts dependencies from a file
func (dt *DependencyTracker) extractDependencies(file *types.FileInfo) ([]Dependency, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%d", file.Path, file.ModTime.Unix())
	if cached, exists := dt.dependencyCache[cacheKey]; exists {
		return cached, nil
	}

	var dependencies []Dependency

	// Use language-specific detector if available
	if detector, exists := dt.languageDetectors[file.Language]; exists {
		deps, err := detector.ExtractDependencies(file)
		if err != nil {
			return nil, fmt.Errorf("language-specific extraction failed: %w", err)
		}
		dependencies = deps
	} else {
		// Fall back to generic pattern matching
		deps := dt.extractDependenciesGeneric(file)
		dependencies = deps
	}

	// Cache the result
	dt.dependencyCache[cacheKey] = dependencies

	return dependencies, nil
}

// extractDependenciesGeneric performs generic dependency extraction
func (dt *DependencyTracker) extractDependenciesGeneric(file *types.FileInfo) []Dependency {
	var dependencies []Dependency

	// Get pattern for this language
	pattern, exists := dt.importPatterns[file.Language]
	if !exists {
		return dependencies
	}

	// Find all import matches
	matches := pattern.FindAllStringSubmatch(file.Content, -1)
	lineNumber := 1

	for _, match := range matches {
		if len(match) > 1 {
			for i, capture := range match[1:] {
				if capture != "" {
					dep := Dependency{
						Type:       dt.getDefaultDependencyType(file.Language),
						Source:     file.Path,
						Target:     capture,
						ImportPath: capture,
						IsExternal: dt.isExternalDependency(capture, file.Language),
						IsRelative: dt.isRelativeImport(capture),
						Line:       lineNumber,
						Kind:       dt.inferImportKind(match[0]),
						Metadata: map[string]interface{}{
							"detection_method": "regex",
							"pattern_group":    i,
						},
					}
					dependencies = append(dependencies, dep)
				}
			}
		}
		lineNumber++
	}

	return dependencies
}

// analyzeDependencyChanges compares dependency lists and identifies changes
func (dt *DependencyTracker) analyzeDependencyChanges(oldDeps, newDeps []Dependency) []DependencyChange {
	var changes []DependencyChange

	// Create maps for efficient lookup
	oldDepMap := make(map[string]*Dependency)
	newDepMap := make(map[string]*Dependency)

	for i, dep := range oldDeps {
		key := dt.getDependencyKey(&dep)
		oldDepMap[key] = &oldDeps[i]
	}

	for i, dep := range newDeps {
		key := dt.getDependencyKey(&dep)
		newDepMap[key] = &newDeps[i]
	}

	// Find added dependencies
	for key, newDep := range newDepMap {
		if _, exists := oldDepMap[key]; !exists {
			change := DependencyChange{
				Type:       DependencyChangeAdd,
				Dependency: newDep,
				Impact:     dt.assessDependencyAdditionImpact(newDep),
				Reason:     "New dependency added",
				Context:    dt.buildDependencyContext([]*Dependency{newDep}),
			}
			changes = append(changes, change)
		}
	}

	// Find removed dependencies
	for key, oldDep := range oldDepMap {
		if _, exists := newDepMap[key]; !exists {
			change := DependencyChange{
				Type:       DependencyChangeRemove,
				Dependency: oldDep,
				Impact:     dt.assessDependencyRemovalImpact(oldDep),
				Reason:     "Dependency removed",
				Context:    dt.buildDependencyContext([]*Dependency{oldDep}),
			}
			changes = append(changes, change)
		}
	}

	// Find modified dependencies
	for key, newDep := range newDepMap {
		if oldDep, exists := oldDepMap[key]; exists {
			if dt.dependenciesAreDifferent(oldDep, newDep) {
				change := DependencyChange{
					Type:       DependencyChangeModify,
					Dependency: newDep,
					OldDep:     oldDep,
					Impact:     dt.assessDependencyModificationImpact(oldDep, newDep),
					Reason:     dt.getDependencyChangeReason(oldDep, newDep),
					Context:    dt.buildDependencyContext([]*Dependency{oldDep, newDep}),
				}
				changes = append(changes, change)
			}
		}
	}

	return changes
}

// analyzeImportOrderChanges analyzes changes in import order
func (dt *DependencyTracker) analyzeImportOrderChanges(oldDeps, newDeps []Dependency) []Change {
	var changes []Change

	// Check if import order matters for this language
	if !dt.importOrderMatters(oldDeps, newDeps) {
		return changes
	}

	// Compare import order
	if dt.hasImportOrderChanged(oldDeps, newDeps) {
		change := Change{
			Type:     ChangeTypeModify,
			Path:     "imports.order",
			OldValue: dt.getImportOrder(oldDeps),
			NewValue: dt.getImportOrder(newDeps),
			Position: Position{Line: 1},
			Impact:   ImpactLow,
			Context: ChangeContext{
				Tags: []string{"import_order", "style"},
			},
			Metadata: map[string]interface{}{
				"change_type":   "import_order",
				"old_order":     dt.getImportOrder(oldDeps),
				"new_order":     dt.getImportOrder(newDeps),
				"order_matters": true,
			},
		}
		changes = append(changes, change)
	}

	return changes
}

// analyzeCircularDependencies analyzes potential circular dependency risks
func (dt *DependencyTracker) analyzeCircularDependencies(deps []Dependency, file *types.FileInfo) []Change {
	var changes []Change

	// Build dependency graph
	graph := dt.buildDependencyGraph(deps, file)

	// Detect circular dependencies
	cycles := dt.detectCycles(graph)

	for _, cycle := range cycles {
		change := Change{
			Type:     ChangeTypeAdd,
			Path:     "dependencies.circular_risk",
			NewValue: cycle,
			Position: Position{Line: 1},
			Impact:   ImpactHigh,
			Context: ChangeContext{
				Tags: []string{"circular_dependency", "risk"},
			},
			Metadata: map[string]interface{}{
				"change_type":      "circular_dependency_risk",
				"cycle_length":     len(cycle),
				"affected_files":   cycle,
				"risk_level":       "high",
			},
		}
		changes = append(changes, change)
	}

	return changes
}

// Helper methods

func (dt *DependencyTracker) convertDependencyChange(depChange DependencyChange, file *types.FileInfo) Change {
	return Change{
		Type:     ChangeType(depChange.Type),
		Path:     fmt.Sprintf("dependencies.%s", depChange.Dependency.Target),
		OldValue: depChange.OldDep,
		NewValue: depChange.Dependency,
		Position: Position{
			Line:   depChange.Dependency.Line,
			Column: 1,
		},
		Impact: depChange.Impact,
		Context: ChangeContext{
			Module: file.Path,
			Tags:   []string{"dependency", string(depChange.Dependency.Type)},
		},
		Metadata: map[string]interface{}{
			"dependency_type":   string(depChange.Dependency.Type),
			"is_external":       depChange.Dependency.IsExternal,
			"is_relative":       depChange.Dependency.IsRelative,
			"import_kind":       string(depChange.Dependency.Kind),
			"change_reason":     depChange.Reason,
			"suggestions":       depChange.Suggestions,
		},
	}
}

func (dt *DependencyTracker) getDependencyKey(dep *Dependency) string {
	return fmt.Sprintf("%s:%s:%s", dep.Type, dep.Target, dep.ImportPath)
}

func (dt *DependencyTracker) dependenciesAreDifferent(old, new *Dependency) bool {
	return old.Alias != new.Alias ||
		old.Line != new.Line ||
		old.Kind != new.Kind ||
		old.IsRelative != new.IsRelative
}

func (dt *DependencyTracker) getDefaultDependencyType(language string) DependencyType {
	switch language {
	case "javascript", "typescript":
		return DependencyTypeImport
	case "python":
		return DependencyTypeFrom
	case "go":
		return DependencyTypeImport
	case "java":
		return DependencyTypeImport
	case "csharp":
		return DependencyTypeUsing
	default:
		return DependencyTypeImport
	}
}

func (dt *DependencyTracker) isExternalDependency(importPath, language string) bool {
	switch language {
	case "javascript", "typescript":
		return !strings.HasPrefix(importPath, ".") && !strings.HasPrefix(importPath, "/")
	case "python":
		return !strings.HasPrefix(importPath, ".")
	case "go":
		return !strings.HasPrefix(importPath, ".")
	default:
		return false
	}
}

func (dt *DependencyTracker) isRelativeImport(importPath string) bool {
	return strings.HasPrefix(importPath, ".") || strings.HasPrefix(importPath, "/")
}

func (dt *DependencyTracker) inferImportKind(importStatement string) ImportKind {
	if strings.Contains(importStatement, "*") {
		return ImportKindNamespace
	}
	if strings.Contains(importStatement, "{") {
		return ImportKindNamed
	}
	if strings.Contains(importStatement, "require(") {
		return ImportKindDefault
	}
	return ImportKindDefault
}

func (dt *DependencyTracker) assessDependencyAdditionImpact(dep *Dependency) ImpactLevel {
	if dep.IsExternal {
		return ImpactMedium
	}
	return ImpactLow
}

func (dt *DependencyTracker) assessDependencyRemovalImpact(dep *Dependency) ImpactLevel {
	if dep.IsExternal {
		return ImpactHigh
	}
	return ImpactMedium
}

func (dt *DependencyTracker) assessDependencyModificationImpact(old, new *Dependency) ImpactLevel {
	if old.IsExternal != new.IsExternal {
		return ImpactHigh
	}
	return ImpactLow
}

func (dt *DependencyTracker) getDependencyChangeReason(old, new *Dependency) string {
	reasons := []string{}

	if old.Alias != new.Alias {
		reasons = append(reasons, "alias changed")
	}
	if old.Line != new.Line {
		reasons = append(reasons, "location moved")
	}
	if old.Kind != new.Kind {
		reasons = append(reasons, "import style changed")
	}
	if old.IsRelative != new.IsRelative {
		reasons = append(reasons, "import type changed")
	}

	if len(reasons) == 0 {
		return "dependency modified"
	}

	return strings.Join(reasons, ", ")
}

func (dt *DependencyTracker) buildDependencyContext(deps []*Dependency) DependencyContext {
	context := DependencyContext{}

	for _, dep := range deps {
		if dep.IsExternal {
			context.ExternalPackages = append(context.ExternalPackages, dep.Target)
		}
	}

	return context
}

func (dt *DependencyTracker) importOrderMatters(oldDeps, newDeps []Dependency) bool {
	// Check if any of the dependencies are order-sensitive
	for _, dep := range append(oldDeps, newDeps...) {
		if dep.Kind == ImportKindSideEffect {
			return true
		}
	}
	return false
}

func (dt *DependencyTracker) hasImportOrderChanged(oldDeps, newDeps []Dependency) bool {
	oldOrder := dt.getImportOrder(oldDeps)
	newOrder := dt.getImportOrder(newDeps)
	return !dt.slicesEqual(oldOrder, newOrder)
}

func (dt *DependencyTracker) getImportOrder(deps []Dependency) []string {
	var order []string
	for _, dep := range deps {
		order = append(order, dep.Target)
	}
	return order
}

func (dt *DependencyTracker) slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (dt *DependencyTracker) buildDependencyGraph(deps []Dependency, file *types.FileInfo) map[string][]string {
	graph := make(map[string][]string)
	
	for _, dep := range deps {
		if !dep.IsExternal {
			graph[file.Path] = append(graph[file.Path], dep.Target)
		}
	}
	
	return graph
}

func (dt *DependencyTracker) detectCycles(graph map[string][]string) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	
	for node := range graph {
		if !visited[node] {
			if cycle := dt.dfsDetectCycle(graph, node, visited, recStack, []string{}); cycle != nil {
				cycles = append(cycles, cycle)
			}
		}
	}
	
	return cycles
}

func (dt *DependencyTracker) dfsDetectCycle(graph map[string][]string, node string, visited, recStack map[string]bool, path []string) []string {
	visited[node] = true
	recStack[node] = true
	path = append(path, node)
	
	for _, neighbor := range graph[node] {
		if !visited[neighbor] {
			if cycle := dt.dfsDetectCycle(graph, neighbor, visited, recStack, path); cycle != nil {
				return cycle
			}
		} else if recStack[neighbor] {
			// Found cycle
			cycleStart := -1
			for i, p := range path {
				if p == neighbor {
					cycleStart = i
					break
				}
			}
			if cycleStart != -1 {
				return append(path[cycleStart:], neighbor)
			}
		}
	}
	
	recStack[node] = false
	return nil
}

// Language-specific dependency detectors (placeholder implementations)

type JavaScriptDependencyDetector struct{}

func NewJavaScriptDependencyDetector() *JavaScriptDependencyDetector {
	return &JavaScriptDependencyDetector{}
}

func (jd *JavaScriptDependencyDetector) ExtractDependencies(file *types.FileInfo) ([]Dependency, error) {
	// Placeholder for JavaScript-specific dependency extraction
	return []Dependency{}, nil
}

func (jd *JavaScriptDependencyDetector) ParseImportStatement(line string) (*Import, error) {
	// Placeholder
	return &Import{}, nil
}

func (jd *JavaScriptDependencyDetector) GetImportKeywords() []string {
	return []string{"import", "export", "require"}
}

func (jd *JavaScriptDependencyDetector) IsRelativeImport(importPath string) bool {
	return strings.HasPrefix(importPath, ".") || strings.HasPrefix(importPath, "/")
}

func (jd *JavaScriptDependencyDetector) NormalizeImportPath(importPath string) string {
	return strings.Trim(importPath, "\"'")
}

type TypeScriptDependencyDetector struct{}

func NewTypeScriptDependencyDetector() *TypeScriptDependencyDetector {
	return &TypeScriptDependencyDetector{}
}

func (td *TypeScriptDependencyDetector) ExtractDependencies(file *types.FileInfo) ([]Dependency, error) {
	return []Dependency{}, nil
}

func (td *TypeScriptDependencyDetector) ParseImportStatement(line string) (*Import, error) {
	return &Import{}, nil
}

func (td *TypeScriptDependencyDetector) GetImportKeywords() []string {
	return []string{"import", "export", "require"}
}

func (td *TypeScriptDependencyDetector) IsRelativeImport(importPath string) bool {
	return strings.HasPrefix(importPath, ".") || strings.HasPrefix(importPath, "/")
}

func (td *TypeScriptDependencyDetector) NormalizeImportPath(importPath string) string {
	return strings.Trim(importPath, "\"'")
}

type GoDependencyDetector struct{}

func NewGoDependencyDetector() *GoDependencyDetector {
	return &GoDependencyDetector{}
}

func (gd *GoDependencyDetector) ExtractDependencies(file *types.FileInfo) ([]Dependency, error) {
	return []Dependency{}, nil
}

func (gd *GoDependencyDetector) ParseImportStatement(line string) (*Import, error) {
	return &Import{}, nil
}

func (gd *GoDependencyDetector) GetImportKeywords() []string {
	return []string{"import"}
}

func (gd *GoDependencyDetector) IsRelativeImport(importPath string) bool {
	return strings.HasPrefix(importPath, ".")
}

func (gd *GoDependencyDetector) NormalizeImportPath(importPath string) string {
	return strings.Trim(importPath, "\"")
}

type PythonDependencyDetector struct{}

func NewPythonDependencyDetector() *PythonDependencyDetector {
	return &PythonDependencyDetector{}
}

func (pd *PythonDependencyDetector) ExtractDependencies(file *types.FileInfo) ([]Dependency, error) {
	return []Dependency{}, nil
}

func (pd *PythonDependencyDetector) ParseImportStatement(line string) (*Import, error) {
	return &Import{}, nil
}

func (pd *PythonDependencyDetector) GetImportKeywords() []string {
	return []string{"import", "from"}
}

func (pd *PythonDependencyDetector) IsRelativeImport(importPath string) bool {
	return strings.HasPrefix(importPath, ".")
}

func (pd *PythonDependencyDetector) NormalizeImportPath(importPath string) string {
	return importPath
}

type JavaDependencyDetector struct{}

func NewJavaDependencyDetector() *JavaDependencyDetector {
	return &JavaDependencyDetector{}
}

func (jd *JavaDependencyDetector) ExtractDependencies(file *types.FileInfo) ([]Dependency, error) {
	return []Dependency{}, nil
}

func (jd *JavaDependencyDetector) ParseImportStatement(line string) (*Import, error) {
	return &Import{}, nil
}

func (jd *JavaDependencyDetector) GetImportKeywords() []string {
	return []string{"import", "package"}
}

func (jd *JavaDependencyDetector) IsRelativeImport(importPath string) bool {
	return false // Java doesn't have relative imports in the same sense
}

func (jd *JavaDependencyDetector) NormalizeImportPath(importPath string) string {
	return strings.TrimSuffix(importPath, ";")
}

type CSharpDependencyDetector struct{}

func NewCSharpDependencyDetector() *CSharpDependencyDetector {
	return &CSharpDependencyDetector{}
}

func (cd *CSharpDependencyDetector) ExtractDependencies(file *types.FileInfo) ([]Dependency, error) {
	return []Dependency{}, nil
}

func (cd *CSharpDependencyDetector) ParseImportStatement(line string) (*Import, error) {
	return &Import{}, nil
}

func (cd *CSharpDependencyDetector) GetImportKeywords() []string {
	return []string{"using"}
}

func (cd *CSharpDependencyDetector) IsRelativeImport(importPath string) bool {
	return false // C# doesn't have relative imports
}

func (cd *CSharpDependencyDetector) NormalizeImportPath(importPath string) string {
	return strings.TrimSuffix(importPath, ";")
}