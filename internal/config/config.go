package config

// Config holds the application configuration
type Config struct {
	SourcePaths     []string `json:"source_paths"`
	OutputPath      string   `json:"output_path"`
	CacheDir        string   `json:"cache_dir"`
	IncludePatterns []string `json:"include_patterns"`
	ExcludePatterns []string `json:"exclude_patterns"`
	MaxFileSize     int64    `json:"max_file_size"`
	Concurrency     int      `json:"concurrency"`
	EnableCache     bool     `json:"enable_cache"`
	EnableProgress  bool     `json:"enable_progress"`
	EnableWatching  bool     `json:"enable_watching"`
	EnableVerbose   bool     `json:"enable_verbose"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		SourcePaths:     []string{"."},
		OutputPath:      "codecontext.md",
		CacheDir:        ".codecontext",
		IncludePatterns: []string{"*.go", "*.js", "*.ts", "*.jsx", "*.tsx"},
		ExcludePatterns: []string{"node_modules/**", ".git/**", "*.test.*"},
		MaxFileSize:     1024 * 1024, // 1MB
		Concurrency:     4,
		EnableCache:     true,
		EnableProgress:  true,
		EnableWatching:  false,
		EnableVerbose:   false,
	}
}