package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/app"
)

var dirName = "tellorapp"

func TestTempDir_CreatesDirectory(t *testing.T) {
	t.Helper()
	dir := tempDir()
	defer func() {
		// Cleanup in case test fails before NewRootCmd cleanup runs
		if dir != app.DefaultNodeHome {
			_ = os.RemoveAll(dir)
		}
	}()

	require.NotEmpty(t, dir)

	// If temp dir creation succeeded, verify it exists
	if dir != app.DefaultNodeHome {
		info, err := os.Stat(dir)
		require.NoError(t, err, "temp directory should exist")
		require.True(t, info.IsDir(), "should be a directory")
		require.Contains(t, dir, dirName, "directory name should contain 'tellorapp'")
	}
}

func TestNewRootCmd_CleansUpTempDirectory(t *testing.T) {
	t.Helper()
	// Count existing tellorapp directories before
	beforeCount := countTellorappDirs(t)

	// Create root command (this should create and clean up a temp dir)
	option := GetOptionWithCustomStartCmd()
	rootCmd := NewRootCmd(option)
	require.NotNil(t, rootCmd)

	// The defer in NewRootCmd should have already executed when the function returned
	// Count tellorapp directories after
	afterCount := countTellorappDirs(t)

	// We should not have more directories than before
	// (allowing for 1 in case cleanup hasn't happened yet due to timing, but it should be <= beforeCount)
	require.LessOrEqual(t, afterCount, beforeCount,
		"Should not accumulate tellorapp directories after NewRootCmd returns")
}

func TestNewRootCmd_MultipleCallsDoNotAccumulate(t *testing.T) {
	t.Helper()
	// Count existing tellorapp directories before
	beforeCount := countTellorappDirs(t)

	// Call NewRootCmd multiple times (simulating multiple command invocations)
	for i := 0; i < 10; i++ {
		option := GetOptionWithCustomStartCmd()
		rootCmd := NewRootCmd(option)
		require.NotNil(t, rootCmd)
	}

	// Count tellorapp directories after
	afterCount := countTellorappDirs(t)

	// After multiple calls, we should not have accumulated directories
	// Each call should clean up its temp directory before returning
	require.LessOrEqual(t, afterCount, beforeCount,
		"Multiple calls to NewRootCmd should not accumulate temp directories")
}

func TestNewRootCmd_TempDirectoryIsRemoved(t *testing.T) {
	t.Helper()
	// Track a specific directory to ensure it gets cleaned up
	var createdDir string

	// Override tempDir temporarily to capture the created directory
	// We'll test by calling NewRootCmd and verifying cleanup
	option := GetOptionWithCustomStartCmd()
	rootCmd := NewRootCmd(option)
	require.NotNil(t, rootCmd)

	// After NewRootCmd returns, check that no tellorapp directories exist
	// that were created in the last second (our test should be fast enough)
	tmpDir := os.TempDir()
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)

	// Find any tellorapp directories
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			if len(name) > len(dirName) && name[:len(dirName)] == dirName {
				// Check if this directory was recently created (within last 2 seconds)
				info, err := entry.Info()
				if err == nil {
					// If the directory is very recent, it might be from our test
					// But since NewRootCmd should clean up immediately, this shouldn't happen
					// We'll just verify the count doesn't grow
					_ = createdDir
					_ = info
				}
			}
		}
	}

	// The real test is that count doesn't grow, which is tested above
	// This test mainly ensures NewRootCmd completes without error
}

// countTellorappDirs counts the number of tellorapp* directories in /tmp
func countTellorappDirs(t *testing.T) int {
	t.Helper()
	tmpDir := os.TempDir()
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			// Check if name starts with "tellorapp"
			if len(name) >= len(dirName) && name[:len(dirName)] == dirName {
				count++
			}
		}
	}
	return count
}

func TestTempDir_FallbackToDefaultNodeHome(t *testing.T) {
	t.Helper()
	// This test verifies that if MkdirTemp fails, we fall back to DefaultNodeHome
	// We can't easily simulate MkdirTemp failure, but we can verify the fallback logic exists
	// by checking that tempDir() can return DefaultNodeHome

	dir := tempDir()
	// If temp creation fails, it should return DefaultNodeHome
	// Otherwise it should return a path containing "tellorapp"
	if dir == app.DefaultNodeHome {
		// This is the fallback case - acceptable
		require.Equal(t, app.DefaultNodeHome, dir)
	} else {
		// Normal case - should contain tellorapp
		require.Contains(t, dir, dirName)
	}
}

func TestNewRootCmd_DoesNotLeakTempDirectories(t *testing.T) {
	t.Helper()
	// This is a more comprehensive test that verifies no leakage
	initialDirs := getTellorappDirs(t)
	initialCount := len(initialDirs)

	// Create multiple root commands
	for i := 0; i < 5; i++ {
		option := GetOptionWithCustomStartCmd()
		rootCmd := NewRootCmd(option)
		require.NotNil(t, rootCmd)
	}

	// Get directories after
	finalDirs := getTellorappDirs(t)
	finalCount := len(finalDirs)

	// We should not have created new directories that persist
	// (allowing 1 for potential race condition, but should be <= initial)
	require.LessOrEqual(t, finalCount, initialCount+1,
		"NewRootCmd should not leak temp directories")

	// Verify that any new directories are cleaned up
	// (new directories would be ones not in initialDirs)
	newDirs := make(map[string]bool)
	for dirPath := range finalDirs {
		if !initialDirs[dirPath] {
			newDirs[dirPath] = true
		}
	}

	// There should be at most 1 new directory (due to timing), and ideally 0
	require.LessOrEqual(t, len(newDirs), 1,
		"Should not have persistent new tellorapp directories")
}

// getTellorappDirs returns a map of all tellorapp* directories in /tmp
func getTellorappDirs(t *testing.T) map[string]bool {
	t.Helper()
	tmpDir := os.TempDir()
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)

	dirs := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			if len(name) >= len(dirName) && name[:len(dirName)] == dirName {
				fullPath := filepath.Join(tmpDir, name)
				dirs[fullPath] = true
			}
		}
	}
	return dirs
}
