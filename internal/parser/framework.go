package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// FrameworkDetector handles framework detection based on multiple strategies
type FrameworkDetector struct {
	projectRoot    string
	packageCache   map[string]*PackageInfo
	frameworkCache map[string]string
}

// PackageInfo represents parsed package.json information
type PackageInfo struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Scripts         map[string]string `json:"scripts"`
}

// NewFrameworkDetector creates a new framework detector
func NewFrameworkDetector(projectRoot string) *FrameworkDetector {
	return &FrameworkDetector{
		projectRoot:    projectRoot,
		packageCache:   make(map[string]*PackageInfo),
		frameworkCache: make(map[string]string),
	}
}

// DetectFramework detects the framework for a given file
func (fd *FrameworkDetector) DetectFramework(filePath, language, content string) string {
	// Check cache first
	if framework, exists := fd.frameworkCache[filePath]; exists {
		return framework
	}

	var framework string

	// Strategy 1: File extension based detection
	framework = fd.detectByFileExtension(filePath)
	if framework != "" {
		fd.frameworkCache[filePath] = framework
		return framework
	}

	// Strategy 2: Import/require statements analysis (for JS/TS files)
	if language == "javascript" || language == "typescript" {
		framework = fd.detectByImports(content)
		if framework != "" {
			fd.frameworkCache[filePath] = framework
			return framework
		}
	}

	// Strategy 3: Package.json dependencies analysis
	framework = fd.detectByPackageJson(filePath)
	if framework != "" {
		fd.frameworkCache[filePath] = framework
		return framework
	}

	// Strategy 4: Python framework detection
	if language == "python" {
		framework = fd.detectPythonFramework(content)
		if framework != "" {
			fd.frameworkCache[filePath] = framework
			return framework
		}
	}

	// Strategy 5: Java framework detection
	if language == "java" {
		framework = fd.detectJavaFramework(content)
		if framework != "" {
			fd.frameworkCache[filePath] = framework
			return framework
		}
	}

	// No framework detected
	fd.frameworkCache[filePath] = ""
	return ""
}

// detectByFileExtension detects framework based on file extensions
func (fd *FrameworkDetector) detectByFileExtension(filePath string) string {
	ext := filepath.Ext(filePath)
	base := filepath.Base(filePath)

	switch ext {
	case ".vue":
		return "Vue"
	case ".svelte":
		return "Svelte"
	case ".astro":
		return "Astro"
	}

	// Check for framework-specific file patterns
	if strings.Contains(base, ".component.") {
		if strings.Contains(base, ".ts") || strings.Contains(base, ".js") {
			return "Angular"
		}
	}

	return ""
}

// detectByImports analyzes import statements to detect frameworks
func (fd *FrameworkDetector) detectByImports(content string) string {
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// React detection
		if strings.Contains(line, "from 'react'") || 
		   strings.Contains(line, "from \"react\"") ||
		   strings.Contains(line, "import React") {
			return "React"
		}
		
		// Next.js detection
		if strings.Contains(line, "from 'next/") || 
		   strings.Contains(line, "from \"next/") ||
		   strings.Contains(line, "from 'next'") ||
		   strings.Contains(line, "from \"next\"") {
			return "Next.js"
		}
		
		// Vue detection
		if strings.Contains(line, "from 'vue'") || 
		   strings.Contains(line, "from \"vue\"") {
			return "Vue"
		}
		
		// Nuxt detection
		if strings.Contains(line, "from '#app'") || 
		   strings.Contains(line, "from \"#app\"") ||
		   strings.Contains(line, "from 'nuxt/") {
			return "Nuxt"
		}
		
		// Angular detection
		if strings.Contains(line, "@angular/core") || 
		   strings.Contains(line, "@angular/common") {
			return "Angular"
		}
		
		// Svelte detection
		if strings.Contains(line, "from 'svelte") || 
		   strings.Contains(line, "from \"svelte") {
			return "Svelte"
		}
		
		// SvelteKit detection
		if strings.Contains(line, "$app/") || 
		   strings.Contains(line, "@sveltejs/kit") {
			return "SvelteKit"
		}
		
		// Astro detection
		if strings.Contains(line, "astro:") {
			return "Astro"
		}
	}
	
	return ""
}

// detectByPackageJson analyzes package.json to detect frameworks
func (fd *FrameworkDetector) detectByPackageJson(filePath string) string {
	packageJson := fd.findPackageJson(filePath)
	if packageJson == "" {
		return ""
	}

	packageInfo := fd.getPackageInfo(packageJson)
	if packageInfo == nil {
		return ""
	}

	// Check dependencies for framework markers
	allDeps := make(map[string]string)
	for k, v := range packageInfo.Dependencies {
		allDeps[k] = v
	}
	for k, v := range packageInfo.DevDependencies {
		allDeps[k] = v
	}

	// Priority order matters - check more specific frameworks first
	if _, exists := allDeps["next"]; exists {
		return "Next.js"
	}
	if _, exists := allDeps["nuxt"]; exists {
		return "Nuxt"
	}
	if _, exists := allDeps["@sveltejs/kit"]; exists {
		return "SvelteKit"
	}
	if _, exists := allDeps["astro"]; exists {
		return "Astro"
	}
	if _, exists := allDeps["react"]; exists {
		return "React"
	}
	if _, exists := allDeps["vue"]; exists {
		return "Vue"
	}
	if _, exists := allDeps["svelte"]; exists {
		return "Svelte"
	}
	if _, exists := allDeps["@angular/core"]; exists {
		return "Angular"
	}

	return ""
}

// detectPythonFramework detects Python frameworks from imports
func (fd *FrameworkDetector) detectPythonFramework(content string) string {
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Django detection
		if strings.Contains(line, "from django") || 
		   strings.Contains(line, "import django") {
			return "Django"
		}
		
		// Flask detection
		if strings.Contains(line, "from flask") || 
		   strings.Contains(line, "import flask") {
			return "Flask"
		}
		
		// FastAPI detection
		if strings.Contains(line, "from fastapi") || 
		   strings.Contains(line, "import fastapi") {
			return "FastAPI"
		}
	}
	
	return ""
}

// detectJavaFramework detects Java frameworks from imports and annotations
func (fd *FrameworkDetector) detectJavaFramework(content string) string {
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Spring Boot detection
		if strings.Contains(line, "org.springframework") || 
		   strings.Contains(line, "@SpringBootApplication") ||
		   strings.Contains(line, "@RestController") ||
		   strings.Contains(line, "@Service") {
			return "Spring Boot"
		}
	}
	
	return ""
}

// findPackageJson finds the nearest package.json file
func (fd *FrameworkDetector) findPackageJson(filePath string) string {
	dir := filepath.Dir(filePath)
	
	for {
		packageJsonPath := filepath.Join(dir, "package.json")
		if _, err := os.Stat(packageJsonPath); err == nil {
			return packageJsonPath
		}
		
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}
	
	return ""
}

// getPackageInfo parses and caches package.json information
func (fd *FrameworkDetector) getPackageInfo(packageJsonPath string) *PackageInfo {
	if info, exists := fd.packageCache[packageJsonPath]; exists {
		return info
	}

	data, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return nil
	}

	var packageInfo PackageInfo
	if err := json.Unmarshal(data, &packageInfo); err != nil {
		return nil
	}

	fd.packageCache[packageJsonPath] = &packageInfo
	return &packageInfo
}