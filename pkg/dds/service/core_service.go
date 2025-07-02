// Package service defines the core service layer for the Distributed Data Store (DDS).
// It orchestrates operations like content publishing (chunking, manifest generation, storage)
// and retrieval (manifest fetching, chunk fetching, reassembly, verification).
package service

import (
	"context"
	"bytes" // Added
	"crypto/sha256"
	"errors" // Already present, moved for grouping
	"fmt"
	"hash" // Added
	"io"
	"time"

	"github.com/DigiSocialBlock/EchoNet/internal/protos/dds/manifest/v1"
	"github.com/DigiSocialBlock/EchoNet/pkg/dds/chunking"
	"github.com/DigiSocialBlock/EchoNet/pkg/dds/discovery" // Added
	"github.com/DigiSocialBlock/EchoNet/pkg/dds/network"   // Added
	"github.com/DigiSocialBlock/EchoNet/pkg/dds/storage"
	"google.golang.org/protobuf/proto"
)

// DDSCoreService defines the primary operations for interacting with the DDS.
// It handles the end-to-end process of publishing and retrieving content.
type DDSCoreService interface {
	// Publish processes the given content, chunks it, generates a manifest,
	// stores all chunks and the manifest using the configured StorageProvider,
	// and (in later phases) announces/pins the content to the network.
	// It returns the manifest CID of the published content and an error if any occurred.
	// Metadata such as filename and MIME type can be passed via options or a dedicated struct.
	Publish(ctx context.Context, content io.Reader, contentSize uint64, options ...PublishOption) (manifestCID string, err error)

	// Retrieve fetches content identified by its manifest CID.
	// It retrieves the manifest, then all constituent data chunks from the
	// configured StorageProvider (and potentially the network in later phases).
	// It verifies the integrity of all data and reassembles the original content.
	// Returns an io.Reader for the reassembled content, the original content size, and an error.
	Retrieve(ctx context.Context, manifestCID string) (contentReader io.Reader, originalSize uint64, err error)

	// TODO (Post-MVP):
	// DeleteManifestAndChunks(ctx context.Context, manifestCID string) error
	// PinContent(ctx context.Context, manifestCID string) error
	// UnpinContent(ctx context.Context, manifestCID string) error
	// GetStatus(ctx context.Context, manifestCID string) (status StatusInfo, err error)
}

// ddsCoreSvc implements the DDSCoreService interface.
// It requires a Chunker (from pkg/dds/chunking) and a StorageProvider (from pkg/dds/storage).
// It will also use NetworkProvider and DiscoveryProvider for fetching remote chunks.
type ddsCoreSvc struct {
	chunker   chunking.Chunker
	store     storage.StorageProvider
	netProv   network.NetworkProvider   // Added
	discoProv discovery.DiscoveryProvider // Added
}

// PublishOption defines a function type for optional parameters to Publish.
type PublishOption func(*publishOptions)

type publishOptions struct {
	filename string
	mimeType string
	meta     map[string]string
	// add other options like custom creationTime if needed
}

// WithFilename is an option to set the filename for the manifest.
func WithFilename(name string) PublishOption {
	return func(opts *publishOptions) {
		opts.filename = name
	}
}

// WithMimeType is an option to set the MIME type for the manifest.
func WithMimeType(mime string) PublishOption {
	return func(opts *publishOptions) {
		opts.mimeType = mime
	}
}

// WithCustomMetadata is an option to set custom metadata for the manifest.
func WithCustomMetadata(meta map[string]string) PublishOption {
	return func(opts *publishOptions) {
		opts.meta = meta
	}
}


// NewDDSCoreService creates a new instance of ddsCoreSvc.
// It requires a Chunker, StorageProvider, NetworkProvider, and DiscoveryProvider.
// For MVP stages where network/discovery might not be fully implemented, stubs can be passed.
func NewDDSCoreService(
	chk chunking.Chunker,
	sp storage.StorageProvider,
	np network.NetworkProvider,
	dp discovery.DiscoveryProvider,
) (DDSCoreService, error) {
	if chk == nil {
		return nil, errors.New("chunker cannot be nil")
	}
	if sp == nil {
		return nil, errors.New("storage provider cannot be nil")
	}
	if np == nil {
		// Allow nil for early tests if only local retrieval is being tested,
		// but service should ideally always have them. Or use specific constructors.
		// For now, let's require them for the full service.
		return nil, errors.New("network provider cannot be nil")
	}
	if dp == nil {
		return nil, errors.New("discovery provider cannot be nil")
	}
	return &ddsCoreSvc{
		chunker:   chk,
		store:     sp,
		netProv:   np,
		discoProv: dp,
	}, nil
}

// Publish processes the given content, chunks it, generates a manifest,
// stores all chunks and the manifest using the configured StorageProvider.
// It returns the manifest CID of the published content and an error if any occurred.
func (s *ddsCoreSvc) Publish(ctx context.Context, content io.Reader, contentSize uint64, options ...PublishOption) (string, error) {
	opts := &publishOptions{} // Default options
	for _, opt := range options {
		opt(opts)
	}

	// 1. Calculate original content hash while reading for chunking
	// To do this efficiently, we might need a TeeReader or a way to hash as we chunk.
	// For simplicity in this first pass, let's assume the chunker might handle this
	// or we pre-hash if the content is small enough / can be read twice.
	// A more advanced chunker could return original hash.
	// For now, let's assume we need to read content to hash it, then again to chunk it.
	// This is inefficient for large files. A better approach would be to pass the reader
	// to a function that both hashes and chunks simultaneously.
	//
	// Let's simplify: Assume content can be fully read for hashing for now for MVP.
	// This is a limitation to address later for large files.
	// A better pattern:
	// contentBytes, err := io.ReadAll(content)
	// if err != nil {
	//    return "", fmt.Errorf("%w: failed to read all content for hashing: %v", ErrPublishFailed, err)
	// }
	// originalHash := sha256.Sum256(contentBytes)
	// contentSize = uint64(len(contentBytes)) // Update contentSize if ReadAll is used
	// readerForChunking := bytes.NewReader(contentBytes)

	// Simpler, but assumes content Reader can be reset or we pass it twice
	// For this implementation, we'll assume the chunker needs a resettable reader or byte slice.
	// The chunking.ChunkData function expects an io.Reader and contentSize.
	// Let's stick to the current chunking.ChunkData signature.
	// We need the original content hash *before* chunking.
	// This means we need to hash the content stream.

	// Create a TeeReader to hash while chunking.
	hashReader := newHashingReader(content) // Custom reader that calculates hash on the fly

	// 2. Chunk the data (ChunkData already generates CIDs for each chunk)
	chunks, err := s.chunker.ChunkData(ctx, hashReader, contentSize) // Added ctx
	if err != nil {
		return "", fmt.Errorf("%w: failed during chunking: %v", ErrPublishFailed, err)
	}

	actualContentSizeRead := hashReader.BytesRead()
	// It's possible contentSize provided was an estimate or max,
	// actualContentSizeRead is the true size processed.
	// For manifest, we should use the actual size of data that was hashed and chunked.
	if actualContentSizeRead != contentSize {
		// This case should ideally be handled by chunker.ChunkData returning an error
		// if it can't read contentSize bytes. Or, we trust actualContentSizeRead.
		// For now, let's use actualContentSizeRead for manifest if different,
		// but this indicates a potential discrepancy to be aware of.
		// The current chunker.ChunkData aims to read exactly contentSize.
	}
	originalContentSHA256 := hashReader.Sum()


	// 3. Store each chunk
	for _, chunk := range chunks {
		err := s.store.Store(ctx, chunk.CID, chunk.Data) // Already had ctx
		if err != nil {
			// TODO: Potential rollback or cleanup of already stored chunks? For MVP, fail fast.
			return "", fmt.Errorf("%w: failed to store chunk %s: %v", ErrPublishFailed, chunk.CID, err)
		}
	}

	// 4. Generate the manifest
	// For MVP, use current time. Filename and MimeType can be passed as options.
	creationTime := time.Now()

	var manifestProto *manifestv1.ContentManifestV1 // Explicitly declare type
	var manifestCID string

	manifestProto, manifestCID, err = s.chunker.GenerateManifest(
		ctx,
		chunks,
		originalContentSHA256,
		actualContentSizeRead, // Use actual bytes read and hashed
		creationTime,
		opts.mimeType,
		opts.filename,
		opts.meta,
	)
	if err != nil {
		return "", fmt.Errorf("%w: failed to generate manifest: %v", ErrPublishFailed, err)
	}

	// 5. Store the manifest
	serializedManifest, err := proto.Marshal(manifestProto)
	if err != nil {
		return "", fmt.Errorf("%w: failed to marshal manifest for storage: %v", ErrPublishFailed, err)
	}
	err = s.store.Store(ctx, manifestCID, serializedManifest)
	if err != nil {
		// TODO: Rollback data chunks?
		return "", fmt.Errorf("%w: failed to store manifest %s: %v", ErrPublishFailed, manifestCID, err)
	}

	// 6. TODO (Post-MVP for this service, part of network layer): Announce/Pin content to network
	// For now, Publish means it's chunked, manifested, and stored in the local StorageProvider.

	return manifestCID, nil
}

// Retrieve fetches content identified by its manifest CID.
// It retrieves the manifest, then all constituent data chunks from the
	// configured StorageProvider (and potentially the network). It verifies integrity and reassembles content.
func (s *ddsCoreSvc) Retrieve(ctx context.Context, manifestCID string) (io.Reader, uint64, error) {
	if manifestCID == "" {
		return nil, 0, fmt.Errorf("%w: manifest CID cannot be empty", ErrInvalidManifestCID)
	}

	// 1. Retrieve the manifest chunk (try local store first)
	serializedManifest, err := s.store.Retrieve(ctx, manifestCID)
	if err != nil {
		if errors.Is(err, storage.ErrChunkNotFound) {
			// Manifest not found locally, try fetching from network if providers are available
			if s.discoProv == nil || s.netProv == nil {
				return nil, 0, fmt.Errorf("%w: manifest %s not found locally and network/discovery providers not configured: %v", ErrRetrieveFailed, manifestCID, err)
			}
			peers, findErr := s.discoProv.FindPeers(ctx, manifestCID, 5) // Find up to 5 peers
			if findErr != nil {
				return nil, 0, fmt.Errorf("%w: failed to find peers for manifest %s: %v", ErrRetrieveFailed, manifestCID, findErr)
			}
			if len(peers) == 0 {
				return nil, 0, fmt.Errorf("%w: manifest %s not found locally and no network peers found: %v", ErrRetrieveFailed, manifestCID, err) // Original not found error
			}

			var fetchErr error
			for _, peerInfo := range peers {
				// TODO: Add timeout/retry logic for FetchChunk
				serializedManifest, fetchErr = s.netProv.FetchChunk(ctx, manifestCID, peerInfo.ID)
				if fetchErr == nil {
					// Successfully fetched from network, store it locally (cache-aside)
					// Log errors from this store operation but don't let it fail the retrieve if fetch was successful
					if storeErr := s.store.Store(ctx, manifestCID, serializedManifest); storeErr != nil {
						// Log this: fmt.Printf("Warning: failed to cache manifest %s fetched from peer %s: %v\n", manifestCID, peerInfo.ID, storeErr)
					}
					break // Found it
				}
				// Log fetchErr for this peer and try next
			}
			if fetchErr != nil { // Failed to fetch from all tried peers
				return nil, 0, fmt.Errorf("%w: manifest %s not found locally and failed to fetch from network peers: %v", ErrRetrieveFailed, manifestCID, fetchErr)
			}

		} else { // Other error retrieving from local store
			return nil, 0, fmt.Errorf("%w: failed to retrieve manifest %s: %v", ErrRetrieveFailed, manifestCID, err)
		}
	}

	// 2. Deserialize the manifest (unchanged)
	manifestProto := &manifestv1.ContentManifestV1{}
	if err := proto.Unmarshal(serializedManifest, manifestProto); err != nil {
		return nil, 0, fmt.Errorf("%w: failed to unmarshal manifest %s: %v", ErrRetrieveFailed, manifestCID, err)
	}

	// 3. Basic validation of the parsed manifest (unchanged)
	if len(manifestProto.ChunkCids) == 0 && manifestProto.OriginalContentSizeBytes > 0 {
		return nil, 0, fmt.Errorf("%w: manifest %s has no chunk CIDs but claims content size > 0", ErrRetrieveFailed, manifestCID)
	}
    if len(manifestProto.ChunkCids) > 0 && manifestProto.OriginalContentSizeBytes == 0 && len(manifestProto.ChunkCids[0]) > 0 {
        emptyChunkCID, _ := chunking.GenerateCID([]byte{})
        if manifestProto.ChunkCids[0] != emptyChunkCID {
             return nil, 0, fmt.Errorf("%w: manifest %s has chunk CIDs but content size is 0", ErrRetrieveFailed, manifestCID)
        }
    }

	// 4. Retrieve all data chunks
	var reassembledData bytes.Buffer
	var totalBytesRetrieved uint64

	if manifestProto.OriginalContentSizeBytes == 0 && len(manifestProto.ChunkCids) == 1 {
		// ... (0-byte content handling - largely unchanged, but Retrieve needs network fallback too)
		emptyChunkCID, _ := chunking.GenerateCID([]byte{})
		if manifestProto.ChunkCids[0] == emptyChunkCID {
			chunkData, err := s.retrieveChunkData(ctx, manifestProto.ChunkCids[0]) // Use helper
			if err != nil {
				return nil, 0, fmt.Errorf("%w: failed to retrieve empty data chunk %s: %v", ErrRetrieveFailed, manifestProto.ChunkCids[0], err)
			}
			if len(chunkData) != 0 {
				return nil, 0, fmt.Errorf("%w: empty data chunk %s was not empty", ErrRetrieveFailed, manifestProto.ChunkCids[0])
			}
		} else {
			return nil, 0, fmt.Errorf("%w: manifest %s indicates 0 size but chunk CID is not for empty data", ErrRetrieveFailed, manifestCID)
		}
	} else {
		// Regular chunk retrieval
		for i, chunkCID := range manifestProto.ChunkCids {
			chunkData, err := s.retrieveChunkData(ctx, chunkCID) // Use helper
			if err != nil {
				// Specific error message for missing chunk within the loop
				return nil, 0, fmt.Errorf("%w: data chunk %s (index %d) for manifest %s: %w", ErrRetrieveFailed, chunkCID, i, manifestCID, err) // Changed %v to %w
			}

			calculatedChunkCID, cidErr := chunking.GenerateCID(chunkData)
			if cidErr != nil {
				return nil, 0, fmt.Errorf("%w: failed to calculate CID for retrieved chunk %s (index %d): %v", ErrRetrieveFailed, chunkCID, i, cidErr)
			}
			if calculatedChunkCID != chunkCID {
				return nil, 0, fmt.Errorf("%w: integrity check failed for chunk %s (index %d): expected CID %s, got %s", ErrRetrieveFailed, chunkCID, i, chunkCID, calculatedChunkCID)
			}

			reassembledData.Write(chunkData)
			totalBytesRetrieved += uint64(len(chunkData))
		}
	}

	// 5. Verify overall content size (unchanged)
	if totalBytesRetrieved != manifestProto.OriginalContentSizeBytes {
		return nil, 0, fmt.Errorf("%w: reassembled content size %d does not match manifest original size %d for manifest %s", ErrRetrieveFailed, totalBytesRetrieved, manifestProto.OriginalContentSizeBytes, manifestCID)
	}

	// 6. Verify overall content hash
	finalHash := sha256.Sum256(reassembledData.Bytes())
	if !bytes.Equal(finalHash[:], manifestProto.OriginalContentSha256) {
		return nil, 0, fmt.Errorf("%w: reassembled content hash mismatch for manifest %s", ErrRetrieveFailed, manifestCID)
	}

	// Return an io.Reader for the reassembled data
	return bytes.NewReader(reassembledData.Bytes()), manifestProto.OriginalContentSizeBytes, nil
}

// Ensure ddsCoreSvc implements DDSCoreService
var _ DDSCoreService = (*ddsCoreSvc)(nil)

// Helper for errors, can be expanded
var (
	ErrPublishFailed      = errors.New("dds: publish operation failed")
	ErrRetrieveFailed     = errors.New("dds: retrieve operation failed")
	ErrInvalidManifestCID = errors.New("dds: invalid manifest CID")
	ErrChunkFetchFailed   = errors.New("dds: failed to fetch chunk from network")
)

// retrieveChunkData is a helper method to get a chunk, trying local store first, then network.
func (s *ddsCoreSvc) retrieveChunkData(ctx context.Context, cid string) ([]byte, error) {
	data, err := s.store.Retrieve(ctx, cid)
	if err == nil {
		return data, nil // Found locally
	}

	if errors.Is(err, storage.ErrChunkNotFound) && s.discoProv != nil && s.netProv != nil {
		// Not found locally, try network
		peers, findErr := s.discoProv.FindPeers(ctx, cid, 5) // Find up to 5 peers
		if findErr != nil {
			return nil, fmt.Errorf("discovery failed for chunk %s: %w", cid, findErr)
		}
		if len(peers) == 0 {
			return nil, fmt.Errorf("chunk %s not found locally and no network peers found: %w", cid, storage.ErrChunkNotFound)
		}

		var fetchErr error
		for _, peerInfo := range peers {
			// TODO: Add timeout/retry logic for FetchChunk
			data, fetchErr = s.netProv.FetchChunk(ctx, cid, peerInfo.ID)
			if fetchErr == nil {
				// Successfully fetched from network, store it locally (cache-aside)
				if storeErr := s.store.Store(ctx, cid, data); storeErr != nil {
					// Log this warning, but proceed with the fetched data
					// fmt.Printf("Warning: failed to cache chunk %s fetched from peer %s: %v\n", cid, peerInfo.ID, storeErr)
				}
				return data, nil // Success
			}
			// Log fetchErr for this peer and try next
		}
		return nil, fmt.Errorf("%w: chunk %s: %v", ErrChunkFetchFailed, cid, fetchErr) // Failed to fetch from all tried peers
	}
	// Other local store error, or network/discovery providers not configured
	return nil, err
}


// hashingReader is an io.Reader that calculates the SHA256 hash of the data as it's read.
type hashingReader struct {
	reader    io.Reader
	hasher    hash.Hash
	bytesRead uint64
}

func newHashingReader(r io.Reader) *hashingReader {
	return &hashingReader{
		reader: r,
		hasher: sha256.New(),
	}
}

func (hr *hashingReader) Read(p []byte) (n int, err error) {
	n, err = hr.reader.Read(p)
	if n > 0 {
		hr.hasher.Write(p[:n])
		hr.bytesRead += uint64(n)
	}
	return
}

// Sum returns the SHA256 hash of all data read so far.
func (hr *hashingReader) Sum() []byte {
	return hr.hasher.Sum(nil)
}

// BytesRead returns the total number of bytes read.
func (hr *hashingReader) BytesRead() uint64 {
	return hr.bytesRead
}
