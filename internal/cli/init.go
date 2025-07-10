package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new CodeContext project",
	Long: `Initialize a new CodeContext project by creating the necessary
configuration files and directory structure.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return initializeProject()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolP("force", "f", false, "force initialization even if config exists")
}

func initializeProject() error {
	configDir := ".codecontext"
	configFile := filepath.Join(configDir, "config.yaml")
	
	// Check if already initialized
	if _, err := os.Stat(configFile); err == nil {
		if !viper.GetBool("force") {
			return fmt.Errorf("CodeContext project already initialized. Use --force to overwrite")
		}
	}
	
	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Create default configuration
	defaultConfig := `# CodeContext Configuration
version: "2.0"

# Virtual Graph Engine Settings
virtual_graph:
  enabled: true
  batch_threshold: 5
  batch_timeout: 500ms
  max_shadow_memory: 100MB
  diff_algorithm: myers

# Incremental Update Settings
incremental_update:
  enabled: true
  min_change_size: 10
  max_patch_history: 1000
  compact_patches: true

# Language Configuration
languages:
  typescript:
    extensions: [".ts", ".tsx", ".mts", ".cts"]
    parser: "tree-sitter-typescript"
  javascript:
    extensions: [".js", ".jsx", ".mjs", ".cjs"]
    parser: "tree-sitter-javascript"
  python:
    extensions: [".py", ".pyi"]
    parser: "tree-sitter-python"
  go:
    extensions: [".go"]
    parser: "tree-sitter-go"

# Compact Profiles
compact_profiles:
  minimal:
    token_target: 0.3
    preserve: ["core", "api", "critical"]
    remove: ["tests", "examples", "generated"]
  balanced:
    token_target: 0.6
    preserve: ["core", "api", "types", "interfaces"]
    remove: ["tests", "examples"]
  aggressive:
    token_target: 0.15
    preserve: ["core", "api"]
    remove: ["tests", "examples", "generated", "comments"]
  debugging:
    preserve: ["error_handling", "logging", "state"]
    expand: ["call_stack", "dependencies"]
  documentation:
    preserve: ["comments", "types", "interfaces"]
    remove: ["implementation_details", "private_methods"]

# Output Settings
output:
  format: "markdown"
  template: "default"
  include_metrics: true
  include_toc: true

# File Patterns
include_patterns:
  - "**/*.ts"
  - "**/*.tsx"
  - "**/*.js"
  - "**/*.jsx"
  - "**/*.py"
  - "**/*.go"

exclude_patterns:
  - "node_modules/**"
  - "dist/**"
  - "build/**"
  - "*.test.*"
  - "*.spec.*"
  - "__pycache__/**"
  - "vendor/**"
  - ".git/**"
`

	// Write config file
	if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	// Create gitignore entry
	gitignoreEntry := ".codecontext/cache/\n.codecontext/logs/\n"
	gitignoreFile := ".gitignore"
	
	if _, err := os.Stat(gitignoreFile); err == nil {
		// Append to existing gitignore
		f, err := os.OpenFile(gitignoreFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open .gitignore: %w", err)
		}
		defer f.Close()
		
		if _, err := f.WriteString(gitignoreEntry); err != nil {
			return fmt.Errorf("failed to write to .gitignore: %w", err)
		}
	} else {
		// Create new gitignore
		if err := os.WriteFile(gitignoreFile, []byte(gitignoreEntry), 0644); err != nil {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
	}
	
	fmt.Println("✅ CodeContext project initialized successfully!")
	fmt.Printf("   Config file: %s\n", configFile)
	fmt.Println("   Next steps:")
	fmt.Println("   1. Run 'codecontext generate' to create initial context map")
	fmt.Println("   2. Edit config.yaml to customize settings")
	
	return nil
}