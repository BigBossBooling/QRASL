package filestore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"fmt"

	"github.com/DigiSocialBlock/EchoNet/pkg/dds/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestFileStore(t *testing.T, dirLevels int, prefixLenPerLevel int) (fs *FileStore, cleanupFunc func()) {
	t.Helper()
	basePath, err := os.MkdirTemp("", "filestore_test_")
	require.NoError(t, err, "Failed to create temp base path for FileStore test")

	fs, err = NewFileStore(basePath, dirLevels, prefixLenPerLevel)
	require.NoError(t, err, "NewFileStore failed")

	cleanupFunc = func() {
		os.RemoveAll(basePath)
	}
	return fs, cleanupFunc
}

func TestNewFileStore(t *testing.T) {
	t.Run("valid creation", func(t *testing.T) {
		basePath := t.TempDir()
		fs, err := NewFileStore(basePath, 2, 2)
		require.NoError(t, err)
		assert.NotNil(t, fs)
		assert.Equal(t, basePath, fs.basePath)
		assert.Equal(t, 2, fs.dirLevels)
		assert.Equal(t, 2, fs.prefixLenPerLevel)
	})

	t.Run("empty base path", func(t *testing.T) {
		_, err := NewFileStore("", 2, 2)
		require.Error(t, err)
	})

	t.Run("invalid dirLevels", func(t *testing.T) {
		basePath := t.TempDir()
		_, err := NewFileStore(basePath, -1, 2)
		require.Error(t, err)
	})

	t.Run("invalid prefixLenPerLevel", func(t *testing.T) {
		basePath := t.TempDir()
		_, err := NewFileStore(basePath, 2, 0)
		require.Error(t, err)
		_, err = NewFileStore(basePath, 1, -1)
		require.Error(t, err)
	})
}

func TestFileStore_GetPathForCID(t *testing.T) {
	testCases := []struct {
		name              string
		dirLevels         int
		prefixLenPerLevel int
		cid               string
		expectedPathSuffix string // Suffix after base path
		expectError       bool
	}{
		{"no sharding", 0, 0, "mycid123", "mycid123", false},
		{"1 level sharding", 1, 2, "abcdef123", filepath.Join("ab", "abcdef123"), false},
		{"2 levels sharding", 2, 2, "abcdef123", filepath.Join("ab", "cd", "abcdef123"), false},
		{"2 levels, 1 prefix char", 2, 1, "abcdef123", filepath.Join("a", "b", "abcdef123"), false},
		{"CID too short for sharding", 2, 2, "abc", "", true}, // "abc" is shorter than 2*2=4
		{"empty CID", 1, 2, "", "", true},
		{"exact length for sharding", 2, 2, "abcd", filepath.Join("ab", "cd", "abcd"), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs, cleanup := setupTestFileStore(t, tc.dirLevels, tc.prefixLenPerLevel)
			defer cleanup()

			fullPath, err := fs.getPathForCID(tc.cid)
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			expectedFullPath := filepath.Join(fs.basePath, tc.expectedPathSuffix)
			assert.Equal(t, expectedFullPath, fullPath)
		})
	}
}


func TestFileStore_StoreRetrieveHasDelete(t *testing.T) {
	fs, cleanup := setupTestFileStore(t, 2, 2) // Use 2 levels, 2 chars per level for sharding
	defer cleanup()

	ctx := context.Background()
	cid1 := "abcdef1234567890" // Path: basePath/ab/cd/abcdef1234567890
	data1 := []byte("hello world")

	cid2 := "uvwxyzABCDEFGHIJ" // Path: basePath/uv/wx/uvwxyzABCDEFGHIJ
	data2 := []byte("another chunk")

	nonExistentCID := "0000000000000000"

	// Test Store
	err := fs.Store(ctx, cid1, data1)
	require.NoError(t, err, "Store cid1 failed")
	err = fs.Store(ctx, cid2, data2)
	require.NoError(t, err, "Store cid2 failed")

	// Verify file creation via getPathForCID (internal detail, but useful for test confidence)
	path1, _ := fs.getPathForCID(cid1)
	_, err = os.Stat(path1)
	require.NoError(t, err, "File for cid1 should exist after store")

	// Test Has
	has, err := fs.Has(ctx, cid1)
	require.NoError(t, err, "Has cid1 failed")
	assert.True(t, has, "Expected Has(cid1) to be true")

	has, err = fs.Has(ctx, cid2)
	require.NoError(t, err, "Has cid2 failed")
	assert.True(t, has, "Expected Has(cid2) to be true")

	has, err = fs.Has(ctx, nonExistentCID)
	require.NoError(t, err, "Has nonExistentCID failed") // Has should not error for non-existent if path is valid
	assert.False(t, has, "Expected Has(nonExistentCID) to be false")

	// Test Retrieve
	retrievedData1, err := fs.Retrieve(ctx, cid1)
	require.NoError(t, err, "Retrieve cid1 failed")
	assert.Equal(t, data1, retrievedData1, "Retrieved data1 does not match original")

	retrievedData2, err := fs.Retrieve(ctx, cid2)
	require.NoError(t, err, "Retrieve cid2 failed")
	assert.Equal(t, data2, retrievedData2, "Retrieved data2 does not match original")

	_, err = fs.Retrieve(ctx, nonExistentCID)
	require.Error(t, err, "Expected error when retrieving nonExistentCID")
	assert.ErrorIs(t, err, storage.ErrChunkNotFound, "Expected ErrChunkNotFound for nonExistentCID")

	// Test Delete
	err = fs.Delete(ctx, cid1)
	require.NoError(t, err, "Delete cid1 failed")

	has, err = fs.Has(ctx, cid1)
	require.NoError(t, err, "Has cid1 after delete failed")
	assert.False(t, has, "Expected Has(cid1) to be false after delete")

	_, err = os.Stat(path1) // Check file system directly
	assert.True(t, os.IsNotExist(err), "File for cid1 should not exist after delete")


	// Test idempotent delete
	err = fs.Delete(ctx, cid1) // Delete again
	require.NoError(t, err, "Idempotent delete for cid1 failed")

	// Delete cid2 and test directory cleanup (basic check, full cleanup is complex)
	path1Dir := filepath.Dir(path1) // Path for .../ab/cd (directory containing cid1 file)
	// path2BaseDir := filepath.Dir(path1Dir) // Path for .../ab - This variable was unused.

	path2, _ := fs.getPathForCID(cid2) // Get path for cid2 to check its parent directory
	path2ParentDir := filepath.Dir(path2) // Path for .../uv/wx (directory containing cid2 file)


	err = fs.Delete(ctx, cid2)
	require.NoError(t, err, "Delete cid2 failed")

	// Note: The current FileStore.Delete does not clean up empty directories.
	// So, these directories should still exist.
	// If directory cleanup were implemented, these checks would need to change.
	_, err = os.Stat(path2ParentDir) // Check if basePath/uv/wx exists
	assert.NoError(t, err, "Parent directory of cid2's shard should still exist after cid2 delete")
	_, err = os.Stat(path1Dir) // Check if basePath/ab/cd exists (example from cid1)
	assert.NoError(t, err, "Parent directory of cid1's shard should still exist")


	// Test error cases for Store
	err = fs.Store(ctx, "", data1) // Empty CID
	assert.ErrorIs(t, err, storage.ErrInvalidCIDFormat, "Store with empty CID should error")

	// Test error cases for Retrieve
	_, err = fs.Retrieve(ctx, "") // Empty CID
	assert.ErrorIs(t, err, storage.ErrInvalidCIDFormat, "Retrieve with empty CID should error")

	// Test error cases for Has
	_, err = fs.Has(ctx, "")
	assert.ErrorIs(t, err, storage.ErrInvalidCIDFormat, "Has with empty CID should error")

	cidTooShort := "a" // Assuming prefixLenPerLevel=2, dirLevels=2
	_, err = fs.Has(ctx, cidTooShort)
	assert.Error(t, err, "Has with CID too short for sharding should error on path generation")
	fmt.Println(err) // To see the actual error

	// Test error cases for Delete
	err = fs.Delete(ctx, "")
	assert.ErrorIs(t, err, storage.ErrInvalidCIDFormat, "Delete with empty CID should error")
}

// TODO: Add tests for concurrent access if the FileStore is intended to be concurrency-safe
// (current implementation uses standard os calls which are generally safe, but explicit tests would be good).
// TODO: Add tests for disk full errors (requires mocking os functions, complex).
// TODO: Add tests for permission errors (requires mocking os functions or specific test setup).

// Ensure a final newline
