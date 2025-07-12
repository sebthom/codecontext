package testutils

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockFileSystem_BasicOperations(t *testing.T) {
	mfs := NewMockFileSystem()

	// Test file creation
	content := []byte("Hello, World!")
	err := mfs.CreateFile("test.txt", content)
	require.NoError(t, err)

	// Test file reading
	readContent, err := mfs.ReadFile("test.txt")
	require.NoError(t, err)
	assert.Equal(t, content, readContent)

	// Test file exists
	assert.True(t, mfs.Exists("test.txt"))
	assert.False(t, mfs.IsDirectory("test.txt"))

	// Test file stat
	info, err := mfs.Stat("test.txt")
	require.NoError(t, err)
	assert.Equal(t, "test.txt", info.Name())
	assert.Equal(t, int64(len(content)), info.Size())
	assert.False(t, info.IsDir())
}

func TestMockFileSystem_DirectoryOperations(t *testing.T) {
	mfs := NewMockFileSystem()

	// Test directory creation
	err := mfs.CreateDirectory("testdir")
	require.NoError(t, err)

	// Test directory exists
	assert.True(t, mfs.Exists("testdir"))
	assert.True(t, mfs.IsDirectory("testdir"))

	// Test directory stat
	info, err := mfs.Stat("testdir")
	require.NoError(t, err)
	assert.Equal(t, "testdir", info.Name())
	assert.True(t, info.IsDir())

	// Test nested directory creation
	err = mfs.CreateDirectory("testdir/subdir")
	require.NoError(t, err)
	assert.True(t, mfs.Exists("testdir/subdir"))

	// Test file in directory
	err = mfs.CreateFile("testdir/file.txt", []byte("test content"))
	require.NoError(t, err)
	assert.True(t, mfs.Exists("testdir/file.txt"))

	// Test directory listing
	infos, err := mfs.ListDirectory("testdir")
	require.NoError(t, err)
	assert.Len(t, infos, 2) // subdir and file.txt

	names := make([]string, len(infos))
	for i, info := range infos {
		names[i] = info.Name()
	}
	assert.Contains(t, names, "subdir")
	assert.Contains(t, names, "file.txt")
}

func TestMockFileSystem_FileModification(t *testing.T) {
	mfs := NewMockFileSystem()

	// Create initial file
	initialContent := []byte("initial content")
	err := mfs.CreateFile("test.txt", initialContent)
	require.NoError(t, err)

	// Modify file
	modifiedContent := []byte("modified content")
	err = mfs.WriteFile("test.txt", modifiedContent)
	require.NoError(t, err)

	// Verify modification
	readContent, err := mfs.ReadFile("test.txt")
	require.NoError(t, err)
	assert.Equal(t, modifiedContent, readContent)

	// Check that modification time was updated
	info, err := mfs.Stat("test.txt")
	require.NoError(t, err)
	assert.True(t, time.Since(info.ModTime()) < time.Second)
}

func TestMockFileSystem_FileDeletion(t *testing.T) {
	mfs := NewMockFileSystem()

	// Create file
	err := mfs.CreateFile("test.txt", []byte("test"))
	require.NoError(t, err)
	assert.True(t, mfs.Exists("test.txt"))

	// Delete file
	err = mfs.DeleteFile("test.txt")
	require.NoError(t, err)
	assert.False(t, mfs.Exists("test.txt"))

	// Try to read deleted file
	_, err = mfs.ReadFile("test.txt")
	assert.Error(t, err)

	// Try to delete non-existent file
	err = mfs.DeleteFile("nonexistent.txt")
	assert.Error(t, err)
}

func TestMockFileSystem_DirectoryDeletion(t *testing.T) {
	mfs := NewMockFileSystem()

	// Create directory
	err := mfs.CreateDirectory("testdir")
	require.NoError(t, err)

	// Delete empty directory
	err = mfs.DeleteDirectory("testdir")
	require.NoError(t, err)
	assert.False(t, mfs.Exists("testdir"))

	// Create directory with content
	err = mfs.CreateDirectory("testdir2")
	require.NoError(t, err)
	err = mfs.CreateFile("testdir2/file.txt", []byte("test"))
	require.NoError(t, err)

	// Try to delete non-empty directory
	err = mfs.DeleteDirectory("testdir2")
	assert.Error(t, err) // Should fail because directory is not empty
	assert.True(t, mfs.Exists("testdir2"))

	// Delete file first, then directory
	err = mfs.DeleteFile("testdir2/file.txt")
	require.NoError(t, err)
	err = mfs.DeleteDirectory("testdir2")
	require.NoError(t, err)
	assert.False(t, mfs.Exists("testdir2"))
}

func TestMockFileSystem_FileRename(t *testing.T) {
	mfs := NewMockFileSystem()

	// Create file
	content := []byte("test content")
	err := mfs.CreateFile("original.txt", content)
	require.NoError(t, err)

	// Rename file
	err = mfs.RenameFile("original.txt", "renamed.txt")
	require.NoError(t, err)

	// Check old path doesn't exist
	assert.False(t, mfs.Exists("original.txt"))

	// Check new path exists with same content
	assert.True(t, mfs.Exists("renamed.txt"))
	readContent, err := mfs.ReadFile("renamed.txt")
	require.NoError(t, err)
	assert.Equal(t, content, readContent)

	// Try to rename non-existent file
	err = mfs.RenameFile("nonexistent.txt", "new.txt")
	assert.Error(t, err)
}

func TestMockFileSystem_NestedStructure(t *testing.T) {
	mfs := NewMockFileSystem()

	// Create nested structure
	files := []string{
		"root.txt",
		"dir1/file1.txt",
		"dir1/subdir1/file2.txt",
		"dir2/file3.txt",
		"dir2/subdir2/file4.txt",
	}

	for _, filePath := range files {
		err := mfs.CreateFile(filePath, []byte("content of "+filePath))
		require.NoError(t, err)
	}

	// Verify all files exist
	for _, filePath := range files {
		assert.True(t, mfs.Exists(filePath), "File should exist: %s", filePath)
	}

	// Verify directories exist
	dirs := []string{"dir1", "dir1/subdir1", "dir2", "dir2/subdir2"}
	for _, dirPath := range dirs {
		assert.True(t, mfs.Exists(dirPath), "Directory should exist: %s", dirPath)
		assert.True(t, mfs.IsDirectory(dirPath), "Should be directory: %s", dirPath)
	}

	// Test directory listings
	rootInfos, err := mfs.ListDirectory(".")
	require.NoError(t, err)
	assert.Len(t, rootInfos, 3) // root.txt, dir1, dir2

	dir1Infos, err := mfs.ListDirectory("dir1")
	require.NoError(t, err)
	assert.Len(t, dir1Infos, 2) // file1.txt, subdir1
}

func TestMockFileSystem_FileWatching(t *testing.T) {
	mfs := NewMockFileSystem()

	// Start watching
	watch, err := mfs.Watch(".", []string{"*.txt"})
	require.NoError(t, err)
	assert.True(t, watch.IsActive())

	// Create file - should trigger event
	err = mfs.CreateFile("test.txt", []byte("test"))
	require.NoError(t, err)

	// Check for event
	select {
	case event := <-watch.GetEvents():
		assert.Equal(t, "test.txt", event.Path)
		assert.Equal(t, OpCreate, event.Operation)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected create event")
	}

	// Modify file - should trigger event
	err = mfs.WriteFile("test.txt", []byte("modified"))
	require.NoError(t, err)

	select {
	case event := <-watch.GetEvents():
		assert.Equal(t, "test.txt", event.Path)
		assert.Equal(t, OpModify, event.Operation)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected modify event")
	}

	// Delete file - should trigger event
	err = mfs.DeleteFile("test.txt")
	require.NoError(t, err)

	select {
	case event := <-watch.GetEvents():
		assert.Equal(t, "test.txt", event.Path)
		assert.Equal(t, OpDelete, event.Operation)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected delete event")
	}

	// Stop watching
	watch.Stop()
	assert.False(t, watch.IsActive())
}

func TestMockFileSystem_WatchPatterns(t *testing.T) {
	mfs := NewMockFileSystem()

	// Watch only .go files
	watch, err := mfs.Watch(".", []string{"*.go"})
	require.NoError(t, err)

	// Create .go file - should trigger event
	err = mfs.CreateFile("main.go", []byte("package main"))
	require.NoError(t, err)

	select {
	case event := <-watch.GetEvents():
		assert.Equal(t, "main.go", event.Path)
		assert.Equal(t, OpCreate, event.Operation)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected create event for .go file")
	}

	// Create .txt file - should NOT trigger event
	err = mfs.CreateFile("readme.txt", []byte("readme"))
	require.NoError(t, err)

	select {
	case <-watch.GetEvents():
		t.Fatal("Should not receive event for .txt file")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event for non-matching pattern
	}
}

func TestMockFileSystem_MultipleWatches(t *testing.T) {
	mfs := NewMockFileSystem()

	// Create multiple watches on the same path with different patterns
	watch1, err := mfs.Watch(".", []string{"*.go"})
	require.NoError(t, err)

	watch2, err := mfs.Watch(".", []string{"*.js"})
	require.NoError(t, err)

	// Create .go file
	err = mfs.CreateFile("test.go", []byte("package test"))
	require.NoError(t, err)

	// Only watch1 should receive event due to pattern filtering
	var gotEvent1, gotEvent2 bool

	select {
	case event := <-watch1.GetEvents():
		assert.Equal(t, "test.go", event.Path)
		gotEvent1 = true
	case <-time.After(100 * time.Millisecond):
		// No event received in watch1
	}

	select {
	case <-watch2.GetEvents():
		gotEvent2 = true
	case <-time.After(50 * time.Millisecond):
		// No event received in watch2 (expected)
	}

	assert.True(t, gotEvent1, "watch1 should have received .go file event")
	assert.False(t, gotEvent2, "watch2 should not have received .go file event")

	// Create .js file
	err = mfs.CreateFile("test.js", []byte("console.log('test')"))
	require.NoError(t, err)

	gotEvent1 = false
	gotEvent2 = false

	select {
	case <-watch1.GetEvents():
		gotEvent1 = true
	case <-time.After(50 * time.Millisecond):
		// No event received in watch1 (expected)
	}

	select {
	case event := <-watch2.GetEvents():
		assert.Equal(t, "test.js", event.Path)
		gotEvent2 = true
	case <-time.After(100 * time.Millisecond):
		// No event received in watch2
	}

	assert.False(t, gotEvent1, "watch1 should not have received .js file event")
	assert.True(t, gotEvent2, "watch2 should have received .js file event")
}

func TestMockFileSystem_Clear(t *testing.T) {
	mfs := NewMockFileSystem()

	// Create some files and directories
	err := mfs.CreateFile("file1.txt", []byte("test"))
	require.NoError(t, err)
	err = mfs.CreateDirectory("dir1")
	require.NoError(t, err)
	err = mfs.CreateFile("dir1/file2.txt", []byte("test"))
	require.NoError(t, err)

	// Start watching
	watch, err := mfs.Watch(".", []string{"*"})
	require.NoError(t, err)

	// Verify files exist
	assert.True(t, mfs.Exists("file1.txt"))
	assert.True(t, mfs.Exists("dir1"))
	assert.True(t, mfs.Exists("dir1/file2.txt"))
	assert.True(t, watch.IsActive())

	// Clear file system
	mfs.Clear()

	// Verify everything is cleared
	assert.False(t, mfs.Exists("file1.txt"))
	assert.False(t, mfs.Exists("dir1"))
	assert.False(t, mfs.Exists("dir1/file2.txt"))
	assert.False(t, watch.IsActive())

	// Verify maps are empty
	files := mfs.GetAllFiles()
	dirs := mfs.GetAllDirectories()
	assert.Len(t, files, 0)
	assert.Len(t, dirs, 0)
}

func TestMockFileSystem_PathNormalization(t *testing.T) {
	mfs := NewMockFileSystem()

	// Test different path formats
	paths := []string{
		"./test.txt",
		"test.txt",
		"./dir/../test.txt",
	}

	// Create file with first path
	err := mfs.CreateFile(paths[0], []byte("test"))
	require.NoError(t, err)

	// All paths should refer to the same file
	for _, path := range paths {
		assert.True(t, mfs.Exists(path), "Path should exist: %s", path)

		content, err := mfs.ReadFile(path)
		require.NoError(t, err, "Should be able to read: %s", path)
		assert.Equal(t, []byte("test"), content)
	}

	// Delete using different path
	err = mfs.DeleteFile(paths[1])
	require.NoError(t, err)

	// All paths should now not exist
	for _, path := range paths {
		assert.False(t, mfs.Exists(path), "Path should not exist: %s", path)
	}
}

func TestCreateTestProject(t *testing.T) {
	mfs := NewMockFileSystem()

	err := CreateTestProject(mfs)
	require.NoError(t, err)

	// Verify key files exist
	expectedFiles := []string{
		"go.mod",
		"main.go",
		"internal/config/config.go",
		"internal/server/server.go",
		"internal/handlers/handlers.go",
		"internal/models/user.go",
		"frontend/src/app.js",
		"frontend/src/services/userService.js",
		"package.json",
		"README.md",
	}

	for _, file := range expectedFiles {
		assert.True(t, mfs.Exists(file), "File should exist: %s", file)

		content, err := mfs.ReadFile(file)
		require.NoError(t, err)
		assert.NotEmpty(t, content, "File should have content: %s", file)
	}

	// Verify directories exist
	expectedDirs := []string{
		"internal",
		"internal/config",
		"internal/server",
		"internal/handlers",
		"internal/models",
		"frontend",
		"frontend/src",
		"frontend/src/services",
	}

	for _, dir := range expectedDirs {
		assert.True(t, mfs.Exists(dir), "Directory should exist: %s", dir)
		assert.True(t, mfs.IsDirectory(dir), "Should be directory: %s", dir)
	}
}

func TestSimulateFileChanges(t *testing.T) {
	mfs := NewMockFileSystem()

	// Create test project first
	err := CreateTestProject(mfs)
	require.NoError(t, err)

	// Start watching
	watch, err := mfs.Watch(".", []string{"*.go", "*.js"})
	require.NoError(t, err)

	// Simulate changes
	SimulateFileChanges(mfs, watch, t)

	// Verify that the logger file was created
	assert.True(t, mfs.Exists("internal/logger/logger.go"))

	// Verify that the user model was renamed
	assert.False(t, mfs.Exists("internal/models/user.go"))
	assert.True(t, mfs.Exists("internal/models/user_model.go"))

	// Verify config file was modified
	content, err := mfs.ReadFile("internal/config/config.go")
	require.NoError(t, err)
	assert.Contains(t, string(content), "LogLevel")

	// Check that we received some events
	eventsReceived := 0
	timeout := time.After(200 * time.Millisecond)

	for {
		select {
		case <-watch.GetEvents():
			eventsReceived++
		case <-timeout:
			goto done
		}
	}

done:
	assert.Greater(t, eventsReceived, 0, "Should have received some file events")
}

func TestAssertEventsReceived(t *testing.T) {
	mfs := NewMockFileSystem()

	watch, err := mfs.Watch(".", []string{"*.txt"})
	require.NoError(t, err)

	// Create some files in background
	go func() {
		time.Sleep(10 * time.Millisecond)
		mfs.CreateFile("test1.txt", []byte("test"))
		time.Sleep(10 * time.Millisecond)
		mfs.WriteFile("test1.txt", []byte("modified"))
		time.Sleep(10 * time.Millisecond)
		mfs.DeleteFile("test1.txt")
	}()

	// Use the helper function to assert events
	expectedOps := []EventOperation{OpCreate, OpModify, OpDelete}
	AssertEventsReceived(t, watch, expectedOps, 500*time.Millisecond)
}

// Benchmark tests

func BenchmarkMockFileSystem_CreateFile(b *testing.B) {
	mfs := NewMockFileSystem()
	content := []byte("test content for benchmarking")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filename := fmt.Sprintf("test%d.txt", i)
		err := mfs.CreateFile(filename, content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMockFileSystem_ReadFile(b *testing.B) {
	mfs := NewMockFileSystem()
	content := []byte("test content for benchmarking")

	// Setup: create files
	for i := 0; i < 1000; i++ {
		filename := fmt.Sprintf("test%d.txt", i)
		mfs.CreateFile(filename, content)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filename := fmt.Sprintf("test%d.txt", i%1000)
		_, err := mfs.ReadFile(filename)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMockFileSystem_Exists(b *testing.B) {
	mfs := NewMockFileSystem()
	content := []byte("test content")

	// Setup: create files
	for i := 0; i < 1000; i++ {
		filename := fmt.Sprintf("test%d.txt", i)
		mfs.CreateFile(filename, content)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filename := fmt.Sprintf("test%d.txt", i%1000)
		mfs.Exists(filename)
	}
}

// Helper functions for testing

func createTempFiles(mfs *MockFileSystem, count int) error {
	for i := 0; i < count; i++ {
		filename := fmt.Sprintf("temp%d.txt", i)
		content := fmt.Sprintf("Content of file %d", i)
		if err := mfs.CreateFile(filename, []byte(content)); err != nil {
			return err
		}
	}
	return nil
}

func verifyFileContent(t *testing.T, mfs *MockFileSystem, path string, expectedContent []byte) {
	content, err := mfs.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, expectedContent, content)
}

func countEvents(watch *MockWatch, timeout time.Duration) int {
	count := 0
	timeoutChan := time.After(timeout)

	for {
		select {
		case <-watch.GetEvents():
			count++
		case <-timeoutChan:
			return count
		}
	}
}

func TestMockFileSystem_ThreadSafety(t *testing.T) {
	mfs := NewMockFileSystem()

	// Run concurrent operations
	done := make(chan bool, 10)

	// Concurrent file creation
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				filename := fmt.Sprintf("thread%d_file%d.txt", id, j)
				mfs.CreateFile(filename, []byte(fmt.Sprintf("content %d-%d", id, j)))
			}
		}(i)
	}

	// Concurrent file reading/existence checking
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				filename := fmt.Sprintf("thread%d_file%d.txt", id%5, j%100)
				mfs.Exists(filename)
				if mfs.Exists(filename) {
					mfs.ReadFile(filename)
				}
			}
		}(i + 5)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify that files were created correctly
	files := mfs.GetAllFiles()
	assert.Equal(t, 500, len(files)) // 5 threads * 100 files each
}
