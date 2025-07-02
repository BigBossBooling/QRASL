// Package memorystore provides an in-memory implementation of the
// storage.StorageProvider interface for DigiSocialBlock's DDS.
// It is primarily intended for testing purposes where disk I/O is not desired.
package memorystore

import (
	"context"
	"sync"

	"github.com/DigiSocialBlock/EchoNet/pkg/dds/storage" // Import the interface package
)

// MemoryStore implements the storage.StorageProvider interface using an in-memory map.
// It is concurrency-safe.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewMemoryStore creates a new, empty MemoryStore instance.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string][]byte),
	}
}

// Store persists the given chunk data in memory, addressable by its CID.
func (ms *MemoryStore) Store(ctx context.Context, cid string, data []byte) error {
	if cid == "" {
		return storage.ErrInvalidCIDFormat
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Create a copy of the data to store, to prevent external modification
	// of the byte slice after it's been stored.
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	ms.data[cid] = dataCopy

	return nil
}

// Retrieve fetches the chunk data for the given CID from memory.
func (ms *MemoryStore) Retrieve(ctx context.Context, cid string) ([]byte, error) {
	if cid == "" {
		return nil, storage.ErrInvalidCIDFormat
	}

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	data, ok := ms.data[cid]
	if !ok {
		return nil, storage.ErrChunkNotFound
	}

	// Return a copy to prevent external modification of the stored slice.
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	return dataCopy, nil
}

// Has checks if the MemoryStore currently holds the chunk for the given CID.
func (ms *MemoryStore) Has(ctx context.Context, cid string) (bool, error) {
	if cid == "" {
		// Consistent with FileStore, Has might not error on invalid CID format,
		// but rather just return false as it definitely won't be found.
		// However, for strictness, let's return an error. Or, document that it returns false.
		// For now, returning error for consistency with other methods.
		return false, storage.ErrInvalidCIDFormat
	}

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	_, ok := ms.data[cid]
	return ok, nil
}

// Delete removes the chunk associated with the given CID from memory.
func (ms *MemoryStore) Delete(ctx context.Context, cid string) error {
	if cid == "" {
		return storage.ErrInvalidCIDFormat
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Deleting a non-existent key from a map is a no-op, which aligns with
	// idempotent delete behavior.
	delete(ms.data, cid)
	return nil
}

// Ensure MemoryStore implements StorageProvider
var _ storage.StorageProvider = (*MemoryStore)(nil)
