// Package filestore provides a file-system based implementation of the
// storage.StorageProvider interface for DigiSocialBlock's DDS.
// It stores content chunks as individual files on the local disk.
package filestore

import (
	"context"
	"fmt"
	// "io/fs" // Not used directly
	"os"
	"path/filepath"
	// "strings" // Not used directly

	"github.com/DigiSocialBlock/EchoNet/pkg/dds/storage" // Import the interface package
)

// FileStore implements the storage.StorageProvider interface using the local filesystem.
type FileStore struct {
	basePath string // The root directory where chunks will be stored.
	// dirLevels defines how many levels of subdirectories to create based on CID prefix.
	// For example, if dirLevels = 2 and prefixLenPerLevel = 2, a CID "abcdefgh..."
	// might be stored in "basePath/ab/cd/abcdefgh...".
	dirLevels         int
	prefixLenPerLevel int
}

// NewFileStore creates a new FileStore instance.
// basePath is the root directory for storage. It will be created if it doesn't exist.
// dirLevels specifies the number of subdirectory levels to use for sharding files.
// prefixLenPerLevel specifies how many characters of the CID prefix to use for each directory level.
// Example: dirLevels=2, prefixLenPerLevel=2 -> /xx/yy/CID_FILENAME
func NewFileStore(basePath string, dirLevels int, prefixLenPerLevel int) (*FileStore, error) {
	if basePath == "" {
		return nil, fmt.Errorf("basePath cannot be empty")
	}
	if dirLevels < 0 {
		return nil, fmt.Errorf("dirLevels cannot be negative")
	}
	if prefixLenPerLevel <= 0 && dirLevels > 0 {
		return nil, fmt.Errorf("prefixLenPerLevel must be positive if dirLevels > 0")
	}

	// Ensure the base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path %s: %w", basePath, err)
	}

	return &FileStore{
		basePath:          basePath,
		dirLevels:         dirLevels,
		prefixLenPerLevel: prefixLenPerLevel,
	}, nil
}

// getPathForCID generates the full file path for a given CID.
// It creates subdirectories based on the CID's prefix if dirLevels > 0.
func (fs *FileStore) getPathForCID(cid string) (string, error) {
	if cid == "" {
		return "", storage.ErrInvalidCIDFormat // Or a more specific error
	}

	pathParts := []string{fs.basePath}
	remainingCID := cid
	currentPrefixLenTotal := 0

	for i := 0; i < fs.dirLevels; i++ {
		if len(remainingCID) < fs.prefixLenPerLevel {
			// Not enough characters left in CID for this level of directory prefix.
			// This could happen with very short CIDs if not validated, or if prefixLenPerLevel is too large.
			// For simplicity, we can just use what's left, or error out if strictness is required.
			// Let's error if not enough prefix for configured levels.
			return "", fmt.Errorf("CID too short for configured directory levels and prefix length: %s", cid)
		}
		prefix := remainingCID[:fs.prefixLenPerLevel]
		pathParts = append(pathParts, prefix)
		remainingCID = remainingCID[fs.prefixLenPerLevel:]
		currentPrefixLenTotal += fs.prefixLenPerLevel
	}

	// The final part of the path is the full CID to ensure uniqueness even if prefixes collide
	// (though unlikely with good CIDs and sufficient prefix usage).
	// Or, use the remainingCID if that's preferred to shorten filenames slightly.
	// Using full CID as filename is safer.
	pathParts = append(pathParts, cid)

	return filepath.Join(pathParts...), nil
}

// Store persists the given chunk data, addressable by its CID.
func (fs *FileStore) Store(ctx context.Context, cid string, data []byte) error {
	if cid == "" {
		return storage.ErrInvalidCIDFormat
	}
	filePath, err := fs.getPathForCID(cid)
	if err != nil {
		return fmt.Errorf("failed to generate path for CID %s: %w", cid, err)
	}

	// Create intermediate directories if they don't exist
	dirPath := filepath.Dir(filePath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory structure %s: %w", dirPath, err)
	}

	// Write data to file
	// Using os.WriteFile for simplicity (Go 1.16+)
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		// Could check for specific errors like disk full, but that's OS-dependent.
		// For now, a generic storage error.
		return fmt.Errorf("failed to write chunk %s to %s: %w", cid, filePath, err)
	}
	return nil
}

// Retrieve fetches the chunk data for the given CID.
func (fs *FileStore) Retrieve(ctx context.Context, cid string) ([]byte, error) {
	if cid == "" {
		return nil, storage.ErrInvalidCIDFormat
	}
	filePath, err := fs.getPathForCID(cid)
	if err != nil {
		return nil, fmt.Errorf("failed to generate path for CID %s: %w", cid, err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storage.ErrChunkNotFound
		}
		return nil, fmt.Errorf("failed to read chunk %s from %s: %w", cid, filePath, err)
	}
	return data, nil
}

// Has checks if the storage provider currently holds the chunk for the given CID.
func (fs *FileStore) Has(ctx context.Context, cid string) (bool, error) {
	if cid == "" {
		return false, storage.ErrInvalidCIDFormat
	}
	filePath, err := fs.getPathForCID(cid)
	if err != nil {
		// If path generation fails (e.g. CID too short for config), it effectively doesn't exist.
		// We could return (false, nil) or (false, specific_error).
		// Let's assume if path can't be formed, it's not there.
		// However, getPathForCID already returns an error for this.
		// So, if getPathForCID errors, we propagate that.
		return false, fmt.Errorf("failed to generate path for CID %s: %w", cid, err)
	}

	_, err = os.Stat(filePath)
	if err == nil {
		return true, nil // File exists
	}
	if os.IsNotExist(err) {
		return false, nil // File does not exist
	}
	// Other error (e.g., permission denied)
	return false, fmt.Errorf("failed to stat chunk %s at %s: %w", cid, filePath, err)
}

// Delete removes the chunk associated with the given CID from the storage.
func (fs *FileStore) Delete(ctx context.Context, cid string) error {
	if cid == "" {
		return storage.ErrInvalidCIDFormat
	}
	filePath, err := fs.getPathForCID(cid)
	if err != nil {
		return fmt.Errorf("failed to generate path for CID %s: %w", cid, err)
	}

	err = os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Idempotent delete: if it doesn't exist, it's "deleted"
		}
		return fmt.Errorf("failed to delete chunk %s at %s: %w", cid, filePath, err)
	}

	// TODO (Optional): Clean up empty parent directories.
	// This can be complex and might require careful locking if concurrent deletes happen.
	// For now, leave empty directories. A separate GC process could handle this.
	// Example:
	// currentPath := filepath.Dir(filePath)
	// for i := 0; i < fs.dirLevels; i++ {
	// 	err := os.Remove(currentPath) // Only removes if empty
	// 	if err != nil {
	// 		break // Stop if not empty or error
	// 	}
	// 	currentPath = filepath.Dir(currentPath)
	// 	if strings.HasSuffix(currentPath, fs.basePath) || currentPath == fs.basePath {
	// 		break
	// 	}
	// }

	return nil
}

// Ensure FileStore implements StorageProvider
var _ storage.StorageProvider = (*FileStore)(nil)
