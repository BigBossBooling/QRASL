// Package storage defines interfaces and implementations for persisting and
// retrieving content chunks within the Distributed Data Stores (DDS) of DigiSocialBlock.
// It abstracts the underlying storage mechanism, allowing for different backends
// like local file systems, in-memory stores (for testing), or potentially
// other decentralized storage networks in the future.
package storage

import (
	"context"
	"errors"
	// "io" // Potentially for Retrieve returning io.ReadCloser
)

var (
	// ErrChunkNotFound indicates that a requested chunk CID was not found in the storage.
	ErrChunkNotFound = errors.New("chunk not found")
	// ErrStorageFull indicates that the storage provider cannot store more data.
	ErrStorageFull = errors.New("storage is full")
	// ErrInvalidCIDFormat indicates that the provided CID string is not in a valid format.
	ErrInvalidCIDFormat = errors.New("invalid CID format")
)

// StorageProvider defines the contract for components responsible for the local
// persistence and retrieval of content chunks based on their Content IDs (CIDs).
// Implementations of this interface will handle the specifics of how and where
// chunks are stored (e.g., local filesystem, in-memory map).
type StorageProvider interface {
	// Store persists the given chunk data, addressable by its CID.
	// If the CID already exists, implementations may choose to overwrite or return an error,
	// though content-addressable storage typically implies idempotency (storing the same
	// data with the same CID multiple times has no new effect).
	// Returns an error if the storage operation fails (e.g., ErrStorageFull).
	Store(ctx context.Context, cid string, data []byte) error

	// Retrieve fetches the chunk data for the given CID.
	// Returns ErrChunkNotFound if the CID is not found in the storage.
	// Other errors may be returned for different retrieval issues.
	// For chunks of DefaultChunkSize (256KiB), returning []byte is acceptable.
	// For potentially larger objects, an io.ReadCloser might be more appropriate.
	Retrieve(ctx context.Context, cid string) ([]byte, error)

	// Has checks if the storage provider currently holds the chunk for the given CID.
	// Returns true if the chunk exists, false otherwise, and an error if the check fails.
	Has(ctx context.Context, cid string) (bool, error)

	// Delete removes the chunk associated with the given CID from the storage.
	// If the chunk does not exist, it may return nil (idempotent delete) or ErrChunkNotFound
	// depending on the desired strictness of the implementation.
	// Returns an error if the deletion operation fails.
	Delete(ctx context.Context, cid string) error

	// TODO (Post-MVP Consideration):
	// Pin ensures that a chunk with the given CID is marked as important and
	// should not be garbage collected if a GC process is implemented.
	// Pin(ctx context.Context, cid string) error

	// TODO (Post-MVP Consideration):
	// Unpin removes the important marker from a chunk, making it eligible for GC.
	// Unpin(ctx context.Context, cid string) error

	// TODO (Post-MVP Consideration or for specific implementations):
	// StoredCIDs returns a stream or list of CIDs currently stored by this provider.
	// This could be useful for advertising to a DHT or for local replication checks.
	// StoredCIDs(ctx context.Context) (<-chan string, error)
}
