package testutils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// MockFileSystem provides a mock file system for testing
type MockFileSystem struct {
	files       map[string]*MockFile
	directories map[string]*MockDirectory
	mutex       sync.RWMutex
	watches     map[string]*MockWatch
	watchMutex  sync.RWMutex
}

// MockFile represents a file in the mock file system
type MockFile struct {
	Path    string
	Content []byte
	Mode    os.FileMode
	ModTime time.Time
	Size    int64
	IsDir   bool
	mutex   sync.RWMutex
}

// MockDirectory represents a directory in the mock file system
type MockDirectory struct {
	Path     string
	Mode     os.FileMode
	ModTime  time.Time
	Children []string
	mutex    sync.RWMutex
}

// MockWatch represents a file watcher in the mock file system
type MockWatch struct {
	Path     string
	Events   chan MockFileEvent
	Errors   chan error
	Active   bool
	Patterns []string
	mutex    sync.RWMutex
	mfs      *MockFileSystem // Reference to parent for cleanup
	key      string          // Unique key for this watch
}

// MockFileEvent represents a file system event
type MockFileEvent struct {
	Path      string
	Operation EventOperation
	Time      time.Time
	NewPath   string // For rename operations
}

// EventOperation represents the type of file system operation
type EventOperation int

const (
	OpCreate EventOperation = iota
	OpModify
	OpDelete
	OpRename
	OpChmod
)

func (op EventOperation) String() string {
	switch op {
	case OpCreate:
		return "CREATE"
	case OpModify:
		return "MODIFY"
	case OpDelete:
		return "DELETE"
	case OpRename:
		return "RENAME"
	case OpChmod:
		return "CHMOD"
	default:
		return "UNKNOWN"
	}
}

// NewMockFileSystem creates a new mock file system
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files:       make(map[string]*MockFile),
		directories: make(map[string]*MockDirectory),
		watches:     make(map[string]*MockWatch),
	}
}

// CreateFile creates a file in the mock file system
func (mfs *MockFileSystem) CreateFile(path string, content []byte) error {
	return mfs.CreateFileWithMode(path, content, 0644)
}

// CreateFileWithMode creates a file with specific mode
func (mfs *MockFileSystem) CreateFileWithMode(path string, content []byte, mode os.FileMode) error {
	mfs.mutex.Lock()
	defer mfs.mutex.Unlock()

	// Normalize path
	path = filepath.Clean(path)

	// Ensure root directory exists
	if _, exists := mfs.directories["."]; !exists {
		mfs.createDirectoryInternal(".", 0755)
	}

	// Create parent directories if they don't exist
	parentDir := filepath.Dir(path)
	if parentDir != "." && parentDir != "/" {
		if err := mfs.createDirectoryInternal(parentDir, 0755); err != nil {
			return err
		}
	}

	// Create the file
	file := &MockFile{
		Path:    path,
		Content: make([]byte, len(content)),
		Mode:    mode,
		ModTime: time.Now(),
		Size:    int64(len(content)),
		IsDir:   false,
	}
	copy(file.Content, content)

	mfs.files[path] = file

	// Add to parent directory
	if dir, exists := mfs.directories[parentDir]; exists {
		dir.mutex.Lock()
		// Check if child already exists to avoid duplicates
		childName := filepath.Base(path)
		found := false
		for _, child := range dir.Children {
			if child == childName {
				found = true
				break
			}
		}
		if !found {
			dir.Children = append(dir.Children, childName)
		}
		dir.mutex.Unlock()
	}

	// Fire watch events
	mfs.fireEvent(MockFileEvent{
		Path:      path,
		Operation: OpCreate,
		Time:      time.Now(),
	})

	return nil
}

// CreateDirectory creates a directory in the mock file system
func (mfs *MockFileSystem) CreateDirectory(path string) error {
	return mfs.CreateDirectoryWithMode(path, 0755)
}

// CreateDirectoryWithMode creates a directory with specific mode
func (mfs *MockFileSystem) CreateDirectoryWithMode(path string, mode os.FileMode) error {
	mfs.mutex.Lock()
	defer mfs.mutex.Unlock()

	return mfs.createDirectoryInternal(path, mode)
}

func (mfs *MockFileSystem) createDirectoryInternal(path string, mode os.FileMode) error {
	// Normalize path
	path = filepath.Clean(path)

	// Check if already exists
	if _, exists := mfs.directories[path]; exists {
		return nil // Directory already exists
	}

	// Create root directory if needed
	if path == "." || path == "/" {
		dir := &MockDirectory{
			Path:     path,
			Mode:     mode | os.ModeDir,
			ModTime:  time.Now(),
			Children: make([]string, 0),
		}
		mfs.directories[path] = dir
		return nil
	}

	// Create parent directories recursively
	parentDir := filepath.Dir(path)
	if parentDir != "." && parentDir != "/" && parentDir != path {
		if err := mfs.createDirectoryInternal(parentDir, mode); err != nil {
			return err
		}
	}

	// Create the directory
	dir := &MockDirectory{
		Path:     path,
		Mode:     mode | os.ModeDir,
		ModTime:  time.Now(),
		Children: make([]string, 0),
	}

	mfs.directories[path] = dir

	// Add to parent directory
	if parentDir != path {
		// Ensure parent directory exists
		if _, exists := mfs.directories[parentDir]; !exists {
			mfs.createDirectoryInternal(parentDir, mode)
		}

		if parentDirObj, exists := mfs.directories[parentDir]; exists {
			parentDirObj.mutex.Lock()
			// Check if child already exists to avoid duplicates
			childName := filepath.Base(path)
			found := false
			for _, child := range parentDirObj.Children {
				if child == childName {
					found = true
					break
				}
			}
			if !found {
				parentDirObj.Children = append(parentDirObj.Children, childName)
			}
			parentDirObj.mutex.Unlock()
		}
	}

	// Fire watch events
	mfs.fireEvent(MockFileEvent{
		Path:      path,
		Operation: OpCreate,
		Time:      time.Now(),
	})

	return nil
}

// ReadFile reads a file from the mock file system
func (mfs *MockFileSystem) ReadFile(path string) ([]byte, error) {
	mfs.mutex.RLock()
	defer mfs.mutex.RUnlock()

	path = filepath.Clean(path)

	file, exists := mfs.files[path]
	if !exists {
		return nil, &fs.PathError{
			Op:   "read",
			Path: path,
			Err:  fs.ErrNotExist,
		}
	}

	file.mutex.RLock()
	defer file.mutex.RUnlock()

	content := make([]byte, len(file.Content))
	copy(content, file.Content)
	return content, nil
}

// WriteFile writes content to a file in the mock file system
func (mfs *MockFileSystem) WriteFile(path string, content []byte) error {
	mfs.mutex.Lock()
	defer mfs.mutex.Unlock()

	path = filepath.Clean(path)

	// Check if file exists
	if file, exists := mfs.files[path]; exists {
		// Modify existing file
		file.mutex.Lock()
		file.Content = make([]byte, len(content))
		copy(file.Content, content)
		file.Size = int64(len(content))
		file.ModTime = time.Now()
		file.mutex.Unlock()

		// Fire watch event
		mfs.fireEvent(MockFileEvent{
			Path:      path,
			Operation: OpModify,
			Time:      time.Now(),
		})
	} else {
		// Create new file
		return mfs.CreateFile(path, content)
	}

	return nil
}

// DeleteFile deletes a file from the mock file system
func (mfs *MockFileSystem) DeleteFile(path string) error {
	mfs.mutex.Lock()
	defer mfs.mutex.Unlock()

	path = filepath.Clean(path)

	if _, exists := mfs.files[path]; !exists {
		return &fs.PathError{
			Op:   "remove",
			Path: path,
			Err:  fs.ErrNotExist,
		}
	}

	delete(mfs.files, path)

	// Remove from parent directory
	parentDir := filepath.Dir(path)
	if dir, exists := mfs.directories[parentDir]; exists {
		dir.mutex.Lock()
		for i, child := range dir.Children {
			if child == filepath.Base(path) {
				dir.Children = append(dir.Children[:i], dir.Children[i+1:]...)
				break
			}
		}
		dir.mutex.Unlock()
	}

	// Fire watch event
	mfs.fireEvent(MockFileEvent{
		Path:      path,
		Operation: OpDelete,
		Time:      time.Now(),
	})

	return nil
}

// DeleteDirectory deletes a directory from the mock file system
func (mfs *MockFileSystem) DeleteDirectory(path string) error {
	mfs.mutex.Lock()
	defer mfs.mutex.Unlock()

	path = filepath.Clean(path)

	dir, exists := mfs.directories[path]
	if !exists {
		return &fs.PathError{
			Op:   "remove",
			Path: path,
			Err:  fs.ErrNotExist,
		}
	}

	// Check if directory is empty
	dir.mutex.RLock()
	hasChildren := len(dir.Children) > 0
	dir.mutex.RUnlock()

	if hasChildren {
		return &fs.PathError{
			Op:   "remove",
			Path: path,
			Err:  fmt.Errorf("directory not empty"),
		}
	}

	delete(mfs.directories, path)

	// Remove from parent directory
	parentDir := filepath.Dir(path)
	if parentDirObj, exists := mfs.directories[parentDir]; exists {
		parentDirObj.mutex.Lock()
		for i, child := range parentDirObj.Children {
			if child == filepath.Base(path) {
				parentDirObj.Children = append(parentDirObj.Children[:i], parentDirObj.Children[i+1:]...)
				break
			}
		}
		parentDirObj.mutex.Unlock()
	}

	// Fire watch event
	mfs.fireEvent(MockFileEvent{
		Path:      path,
		Operation: OpDelete,
		Time:      time.Now(),
	})

	return nil
}

// RenameFile renames a file in the mock file system
func (mfs *MockFileSystem) RenameFile(oldPath, newPath string) error {
	mfs.mutex.Lock()
	defer mfs.mutex.Unlock()

	oldPath = filepath.Clean(oldPath)
	newPath = filepath.Clean(newPath)

	file, exists := mfs.files[oldPath]
	if !exists {
		return &fs.PathError{
			Op:   "rename",
			Path: oldPath,
			Err:  fs.ErrNotExist,
		}
	}

	// Move the file
	file.mutex.Lock()
	file.Path = newPath
	file.ModTime = time.Now()
	file.mutex.Unlock()

	mfs.files[newPath] = file
	delete(mfs.files, oldPath)

	// Update parent directories
	oldParent := filepath.Dir(oldPath)
	newParent := filepath.Dir(newPath)

	// Remove from old parent
	if dir, exists := mfs.directories[oldParent]; exists {
		dir.mutex.Lock()
		for i, child := range dir.Children {
			if child == filepath.Base(oldPath) {
				dir.Children = append(dir.Children[:i], dir.Children[i+1:]...)
				break
			}
		}
		dir.mutex.Unlock()
	}

	// Add to new parent
	if newParent != oldParent {
		if dir, exists := mfs.directories[newParent]; exists {
			dir.mutex.Lock()
			dir.Children = append(dir.Children, filepath.Base(newPath))
			dir.mutex.Unlock()
		}
	}

	// Fire watch event
	mfs.fireEvent(MockFileEvent{
		Path:      oldPath,
		Operation: OpRename,
		Time:      time.Now(),
		NewPath:   newPath,
	})

	return nil
}

// Stat returns file information
func (mfs *MockFileSystem) Stat(path string) (os.FileInfo, error) {
	mfs.mutex.RLock()
	defer mfs.mutex.RUnlock()

	path = filepath.Clean(path)

	// Check files first
	if file, exists := mfs.files[path]; exists {
		return &MockFileInfo{
			name:    filepath.Base(path),
			size:    file.Size,
			mode:    file.Mode,
			modTime: file.ModTime,
			isDir:   false,
		}, nil
	}

	// Check directories
	if dir, exists := mfs.directories[path]; exists {
		return &MockFileInfo{
			name:    filepath.Base(path),
			size:    0,
			mode:    dir.Mode,
			modTime: dir.ModTime,
			isDir:   true,
		}, nil
	}

	return nil, &fs.PathError{
		Op:   "stat",
		Path: path,
		Err:  fs.ErrNotExist,
	}
}

// ListDirectory lists the contents of a directory
func (mfs *MockFileSystem) ListDirectory(path string) ([]os.FileInfo, error) {
	mfs.mutex.RLock()
	defer mfs.mutex.RUnlock()

	path = filepath.Clean(path)

	dir, exists := mfs.directories[path]
	if !exists {
		return nil, &fs.PathError{
			Op:   "readdir",
			Path: path,
			Err:  fs.ErrNotExist,
		}
	}

	dir.mutex.RLock()
	defer dir.mutex.RUnlock()

	var infos []os.FileInfo
	for _, child := range dir.Children {
		childPath := filepath.Join(path, child)

		if file, exists := mfs.files[childPath]; exists {
			infos = append(infos, &MockFileInfo{
				name:    child,
				size:    file.Size,
				mode:    file.Mode,
				modTime: file.ModTime,
				isDir:   false,
			})
		} else if subdir, exists := mfs.directories[childPath]; exists {
			infos = append(infos, &MockFileInfo{
				name:    child,
				size:    0,
				mode:    subdir.Mode,
				modTime: subdir.ModTime,
				isDir:   true,
			})
		}
	}

	// Sort by name
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name() < infos[j].Name()
	})

	return infos, nil
}

// Exists checks if a path exists in the mock file system
func (mfs *MockFileSystem) Exists(path string) bool {
	mfs.mutex.RLock()
	defer mfs.mutex.RUnlock()

	path = filepath.Clean(path)

	_, fileExists := mfs.files[path]
	_, dirExists := mfs.directories[path]

	return fileExists || dirExists
}

// IsDirectory checks if a path is a directory
func (mfs *MockFileSystem) IsDirectory(path string) bool {
	mfs.mutex.RLock()
	defer mfs.mutex.RUnlock()

	path = filepath.Clean(path)

	_, exists := mfs.directories[path]
	return exists
}

// Watch starts watching a path for changes
func (mfs *MockFileSystem) Watch(path string, patterns []string) (*MockWatch, error) {
	mfs.watchMutex.Lock()
	defer mfs.watchMutex.Unlock()

	path = filepath.Clean(path)

	watch := &MockWatch{
		Path:     path,
		Events:   make(chan MockFileEvent, 100),
		Errors:   make(chan error, 10),
		Active:   true,
		Patterns: patterns,
		mfs:      mfs,
	}

	// Use a unique key to allow multiple watches on the same path
	watchKey := fmt.Sprintf("%s_%p", path, watch)
	watch.key = watchKey
	mfs.watches[watchKey] = watch
	return watch, nil
}

// StopWatch stops watching a path
func (mfs *MockFileSystem) StopWatch(path string) {
	mfs.watchMutex.Lock()
	defer mfs.watchMutex.Unlock()

	path = filepath.Clean(path)

	if watch, exists := mfs.watches[path]; exists {
		watch.mutex.Lock()
		watch.Active = false
		close(watch.Events)
		close(watch.Errors)
		watch.mutex.Unlock()

		delete(mfs.watches, path)
	}
}

// GetAllFiles returns all files in the mock file system
func (mfs *MockFileSystem) GetAllFiles() map[string]*MockFile {
	mfs.mutex.RLock()
	defer mfs.mutex.RUnlock()

	files := make(map[string]*MockFile)
	for path, file := range mfs.files {
		files[path] = file
	}
	return files
}

// GetAllDirectories returns all directories in the mock file system
func (mfs *MockFileSystem) GetAllDirectories() map[string]*MockDirectory {
	mfs.mutex.RLock()
	defer mfs.mutex.RUnlock()

	dirs := make(map[string]*MockDirectory)
	for path, dir := range mfs.directories {
		dirs[path] = dir
	}
	return dirs
}

// Clear removes all files and directories from the mock file system
func (mfs *MockFileSystem) Clear() {
	mfs.mutex.Lock()
	defer mfs.mutex.Unlock()

	mfs.files = make(map[string]*MockFile)
	mfs.directories = make(map[string]*MockDirectory)

	// Stop all watches
	mfs.watchMutex.Lock()
	for path, watch := range mfs.watches {
		watch.mutex.Lock()
		watch.Active = false
		close(watch.Events)
		close(watch.Errors)
		watch.mutex.Unlock()
		delete(mfs.watches, path)
	}
	mfs.watchMutex.Unlock()
}

// Private helper methods

func (mfs *MockFileSystem) fireEvent(event MockFileEvent) {
	mfs.watchMutex.RLock()
	defer mfs.watchMutex.RUnlock()

	for _, watch := range mfs.watches {
		if mfs.shouldFireEvent(watch.Path, event.Path, watch.Patterns) {
			watch.mutex.RLock()
			if watch.Active {
				select {
				case watch.Events <- event:
				default:
					// Channel full, skip event
				}
			}
			watch.mutex.RUnlock()
		}
	}
}

func (mfs *MockFileSystem) shouldFireEvent(watchPath, eventPath string, patterns []string) bool {
	// Normalize paths
	watchPath = filepath.Clean(watchPath)
	eventPath = filepath.Clean(eventPath)

	// Check if event path is under watch path
	if watchPath == "." {
		// Watching root, accept all events
	} else if !strings.HasPrefix(eventPath, watchPath) {
		return false
	}

	// If no patterns, accept all
	if len(patterns) == 0 {
		return true
	}

	// Check patterns
	fileName := filepath.Base(eventPath)
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return true
		}
	}

	return false
}

// MockFileInfo implements os.FileInfo for mock files
type MockFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (mfi *MockFileInfo) Name() string       { return mfi.name }
func (mfi *MockFileInfo) Size() int64        { return mfi.size }
func (mfi *MockFileInfo) Mode() os.FileMode  { return mfi.mode }
func (mfi *MockFileInfo) ModTime() time.Time { return mfi.modTime }
func (mfi *MockFileInfo) IsDir() bool        { return mfi.isDir }
func (mfi *MockFileInfo) Sys() interface{}   { return nil }

// MockWatch methods

// GetEvents returns the events channel
func (mw *MockWatch) GetEvents() <-chan MockFileEvent {
	return mw.Events
}

// GetErrors returns the errors channel
func (mw *MockWatch) GetErrors() <-chan error {
	return mw.Errors
}

// IsActive returns whether the watch is active
func (mw *MockWatch) IsActive() bool {
	mw.mutex.RLock()
	defer mw.mutex.RUnlock()
	return mw.Active
}

// Stop stops the watch
func (mw *MockWatch) Stop() {
	mw.mutex.Lock()
	defer mw.mutex.Unlock()

	if mw.Active {
		mw.Active = false
		close(mw.Events)
		close(mw.Errors)

		// Remove from parent file system
		if mw.mfs != nil {
			mw.mfs.watchMutex.Lock()
			delete(mw.mfs.watches, mw.key)
			mw.mfs.watchMutex.Unlock()
		}
	}
}

// Helper functions for testing

// CreateTestProject creates a realistic test project structure
func CreateTestProject(mfs *MockFileSystem) error {
	files := map[string]string{
		"go.mod": `module testproject

go 1.19

require (
	github.com/gorilla/mux v1.8.0
	github.com/stretchr/testify v1.8.0
)`,
		"main.go": `package main

import (
	"fmt"
	"testproject/internal/server"
	"testproject/internal/config"
)

func main() {
	cfg := config.Load()
	srv := server.New(cfg)
	
	fmt.Println("Starting server...")
	srv.Start()
}`,
		"internal/config/config.go": `package config

import "os"

type Config struct {
	Port     string
	Database string
	Debug    bool
}

func Load() *Config {
	return &Config{
		Port:     getEnv("PORT", "8080"),
		Database: getEnv("DATABASE_URL", "postgres://localhost/test"),
		Debug:    getEnv("DEBUG", "false") == "true",
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}`,
		"internal/server/server.go": `package server

import (
	"net/http"
	"github.com/gorilla/mux"
	"testproject/internal/config"
	"testproject/internal/handlers"
)

type Server struct {
	config *config.Config
	router *mux.Router
}

func New(cfg *config.Config) *Server {
	return &Server{
		config: cfg,
		router: mux.NewRouter(),
	}
}

func (s *Server) Start() error {
	s.setupRoutes()
	return http.ListenAndServe(":"+s.config.Port, s.router)
}

func (s *Server) setupRoutes() {
	h := handlers.New()
	s.router.HandleFunc("/health", h.Health).Methods("GET")
	s.router.HandleFunc("/users", h.GetUsers).Methods("GET")
	s.router.HandleFunc("/users", h.CreateUser).Methods("POST")
}`,
		"internal/handlers/handlers.go": `package handlers

import (
	"encoding/json"
	"net/http"
	"testproject/internal/models"
)

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	users := []models.User{
		{ID: 1, Name: "John Doe", Email: "john@example.com"},
		{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	user.ID = 123 // Mock ID assignment
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}`,
		"internal/models/user.go": `package models

import "errors"

type User struct {
	ID    int    ` + "`json:\"id\"`" + `
	Name  string ` + "`json:\"name\"`" + `
	Email string ` + "`json:\"email\"`" + `
}

func (u *User) Validate() error {
	if u.Name == "" {
		return errors.New("name is required")
	}
	if u.Email == "" {
		return errors.New("email is required")
	}
	return nil
}`,
		"frontend/src/app.js": `import { UserService } from './services/userService.js';
import { Router } from './router.js';
import { UI } from './ui.js';

class App {
	constructor() {
		this.userService = new UserService();
		this.router = new Router();
		this.ui = new UI();
	}

	async init() {
		await this.userService.init();
		this.router.init();
		this.ui.render();
		
		console.log('App initialized');
	}
}

export default App;`,
		"frontend/src/services/userService.js": `export class UserService {
	constructor() {
		this.baseUrl = '/api/users';
		this.users = [];
	}

	async init() {
		await this.loadUsers();
	}

	async loadUsers() {
		try {
			const response = await fetch(this.baseUrl);
			this.users = await response.json();
			return this.users;
		} catch (error) {
			console.error('Failed to load users:', error);
			throw error;
		}
	}

	async createUser(userData) {
		const response = await fetch(this.baseUrl, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(userData)
		});
		
		if (!response.ok) {
			throw new Error('Failed to create user');
		}
		
		const user = await response.json();
		this.users.push(user);
		return user;
	}
}`,
		"README.md": `# Test Project

A sample Go web server with JavaScript frontend for testing CodeContext.

## Structure

- main.go - Application entry point
- internal/ - Internal Go packages
- frontend/ - JavaScript frontend
`,
		"package.json": `{
	"name": "test-project-frontend",
	"version": "1.0.0",
	"type": "module",
	"scripts": {
		"build": "esbuild src/app.js --bundle --outfile=dist/app.js",
		"dev": "esbuild src/app.js --bundle --outfile=dist/app.js --watch"
	},
	"devDependencies": {
		"esbuild": "^0.17.0"
	}
}`,
	}

	for path, content := range files {
		if err := mfs.CreateFile(path, []byte(content)); err != nil {
			return err
		}
	}

	return nil
}

// SimulateFileChanges simulates realistic file changes for testing
func SimulateFileChanges(mfs *MockFileSystem, watch *MockWatch, t *testing.T) {
	// Simulate editing a file
	err := mfs.WriteFile("internal/config/config.go", []byte(`package config

import "os"

type Config struct {
	Port     string
	Database string
	Debug    bool
	LogLevel string // Added new field
}

func Load() *Config {
	return &Config{
		Port:     getEnv("PORT", "8080"),
		Database: getEnv("DATABASE_URL", "postgres://localhost/test"),
		Debug:    getEnv("DEBUG", "false") == "true",
		LogLevel: getEnv("LOG_LEVEL", "info"), // Added new field
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}`))
	require.NoError(t, err)

	// Simulate creating a new file
	err = mfs.CreateFile("internal/logger/logger.go", []byte(`package logger

import "log"

type Logger struct {
	level string
}

func New(level string) *Logger {
	return &Logger{level: level}
}

func (l *Logger) Info(msg string) {
	if l.level == "info" || l.level == "debug" {
		log.Println("[INFO]", msg)
	}
}

func (l *Logger) Debug(msg string) {
	if l.level == "debug" {
		log.Println("[DEBUG]", msg)
	}
}`))
	require.NoError(t, err)

	// Simulate deleting a file
	err = mfs.DeleteFile("frontend/src/router.js") // This file doesn't exist, but that's ok for testing
	if err != nil {
		// Expected if file doesn't exist
		t.Logf("Expected error deleting non-existent file: %v", err)
	}

	// Simulate renaming a file
	err = mfs.RenameFile("internal/models/user.go", "internal/models/user_model.go")
	require.NoError(t, err)
}

// AssertEventsReceived checks that expected events were received
func AssertEventsReceived(t *testing.T, watch *MockWatch, expectedOps []EventOperation, timeout time.Duration) {
	received := make([]EventOperation, 0, len(expectedOps))
	timeoutChan := time.After(timeout)

	for len(received) < len(expectedOps) {
		select {
		case event := <-watch.GetEvents():
			received = append(received, event.Operation)
		case <-timeoutChan:
			t.Fatalf("Timeout waiting for events. Expected %v, got %v", expectedOps, received)
		}
	}

	require.Equal(t, expectedOps, received)
}
