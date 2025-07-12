package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitializeProject(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() string
		cleanupFunc func(string)
		wantErr     bool
	}{
		{
			name: "successful initialization",
			setupFunc: func() string {
				tmpDir := t.TempDir()
				if err := os.Chdir(tmpDir); err != nil {
					t.Fatalf("Failed to change directory: %v", err)
				}
				return tmpDir
			},
			cleanupFunc: func(dir string) {
				// Cleanup is handled by t.TempDir()
			},
			wantErr: false,
		},
		{
			name: "initialization in existing project",
			setupFunc: func() string {
				tmpDir := t.TempDir()
				if err := os.Chdir(tmpDir); err != nil {
					t.Fatalf("Failed to change directory: %v", err)
				}

				// Create existing config
				configDir := ".codecontext"
				if err := os.MkdirAll(configDir, 0755); err != nil {
					t.Fatalf("Failed to create config directory: %v", err)
				}

				configFile := filepath.Join(configDir, "config.yaml")
				if err := os.WriteFile(configFile, []byte("existing config"), 0644); err != nil {
					t.Fatalf("Failed to write existing config: %v", err)
				}

				return tmpDir
			},
			cleanupFunc: func(dir string) {
				// Cleanup is handled by t.TempDir()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := tt.setupFunc()
			defer tt.cleanupFunc(tmpDir)

			err := initializeProject()
			if (err != nil) != tt.wantErr {
				t.Errorf("initializeProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Check if config file was created
				configFile := filepath.Join(".codecontext", "config.yaml")
				if _, err := os.Stat(configFile); os.IsNotExist(err) {
					t.Errorf("Config file was not created: %s", configFile)
				}

				// Check if gitignore was created or updated
				gitignoreFile := ".gitignore"
				if _, err := os.Stat(gitignoreFile); os.IsNotExist(err) {
					t.Errorf("Gitignore file was not created: %s", gitignoreFile)
				}
			}
		})
	}
}

func TestInitializeProjectWithForce(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Create existing config
	configDir := ".codecontext"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte("existing config"), 0644); err != nil {
		t.Fatalf("Failed to write existing config: %v", err)
	}

	// TODO: Test force flag functionality
	// This would require refactoring initializeProject to accept parameters
	// For now, we'll test the basic functionality

	// Verify existing config
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Errorf("Existing config file not found: %s", configFile)
	}
}
