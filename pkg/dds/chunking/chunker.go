// Package chunking provides a robust toolkit for dividing content into
// standardized chunks, generating unique Content Identifiers (CIDs) for them,
// and creating manifests to describe collections of these chunks.
// It forms a foundational part of the Distributed Data Stores (DDS) protocol
// within DigiSocialBlock's EchoNet, ensuring data integrity and enabling
// content-addressable storage.
package chunking

import (
	"context" // Added
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/DigiSocialBlock/EchoNet/internal/protos/dds/manifest/v1"
	"github.com/btcsuite/btcutil/base58"
	"google.golang.org/protobuf/proto"
)

// DefaultChunkSize is the standard size for content chunks in bytes.
const DefaultChunkSize = 256 * 1024 // 256 KiB

// ErrInvalidContentSize indicates that the provided contentSize is negative.
// Note: uint64 naturally prevents negative, but this error is for conceptual completeness
// or if the type were to change in the future.
var ErrInvalidContentSize = errors.New("content size cannot be negative")

// ErrReadInconsistentSize indicates that the number of bytes read from io.Reader
// does not match the provided contentSize, or more bytes were read than expected.
var ErrReadInconsistentSize = errors.New("number of bytes read from reader inconsistent with contentSize")

// Chunker defines the operations for content chunking and manifest generation.
// Implementations of this interface handle the logic of breaking down content,
// generating CIDs for chunks, and creating a manifest to describe the content.
type Chunker interface {
	// ChunkData reads content from an io.Reader and divides it into Chunks.
	// Each Chunk includes its data and its generated CID.
	// contentSize is used to determine how many bytes to read in total.
	ChunkData(ctx context.Context, content io.Reader, contentSize uint64) ([]Chunk, error)

	// GenerateManifest creates a ContentManifestV1 protobuf message from a list of chunks
	// and other metadata. It also calculates and returns the CID for this manifest.
	GenerateManifest(
		ctx context.Context,
		chunks []Chunk,
		originalContentSHA256 []byte, // Raw SHA256 hash
		originalContentSizeBytes uint64,
		creationTime time.Time, // Go time.Time object
		mimeType string,
		filename string,
		customMeta map[string]string,
	) (*manifestv1.ContentManifestV1, string, error)
}

// defaultChunker is a concrete implementation of the Chunker interface.
type defaultChunker struct{}

// NewDefaultChunker creates a new instance of the default Chunker implementation.
func NewDefaultChunker() Chunker {
	return &defaultChunker{}
}

// ChunkData implements the Chunker interface by calling the package-level ChunkData function.
func (dc *defaultChunker) ChunkData(ctx context.Context, content io.Reader, contentSize uint64) ([]Chunk, error) {
	return PackageChunkData(ctx, content, contentSize) // Renamed to avoid collision
}

// GenerateManifest implements the Chunker interface by calling the package-level GenerateManifest function.
func (dc *defaultChunker) GenerateManifest(
	ctx context.Context,
	chunks []Chunk,
	originalContentSHA256 []byte,
	originalContentSizeBytes uint64,
	creationTime time.Time,
	mimeType string,
	filename string,
	customMeta map[string]string,
) (*manifestv1.ContentManifestV1, string, error) {
	return PackageGenerateManifest(ctx, chunks, originalContentSHA256, originalContentSizeBytes, creationTime, mimeType, filename, customMeta) // Renamed
}


// Chunk represents a piece of content data.
// The CID field will be populated in a subsequent task.
type Chunk struct {
	Data []byte
	CID  string // Base58BTC encoded SHA-256 hash of Data
}

// GenerateCID creates a Content ID for a given byte slice.
// It computes the SHA-256 hash of the data and then Base58BTC encodes the hash.
func GenerateCID(data []byte) (string, error) {
	if data == nil {
		data = []byte{}
	}
	hasher := sha256.New()
	_, err := hasher.Write(data)
	if err != nil {
		return "", fmt.Errorf("failed to write data to hasher: %w", err)
	}
	hashBytes := hasher.Sum(nil)
	cid := base58.Encode(hashBytes)
	return cid, nil
}

// PackageChunkData reads content from an io.Reader and divides it into Chunks of DefaultChunkSize.
// Each chunk includes its generated CID.
// The contentSize parameter is used to determine how many bytes to read in total.
// Renamed from ChunkData to PackageChunkData to avoid conflict with interface method.
func PackageChunkData(ctx context.Context, content io.Reader, contentSize uint64) ([]Chunk, error) {
	// contentSize being uint64 handles the negative case implicitly.

	if contentSize == 0 {
		emptyData := []byte{}
		cid, err := GenerateCID(emptyData)
		if err != nil {
			return nil, fmt.Errorf("failed to generate CID for empty chunk: %w", err)
		}
		return []Chunk{{Data: emptyData, CID: cid}}, nil
	}

	var chunks []Chunk
	var totalBytesRead uint64

	for totalBytesRead < contentSize {
		bytesToRead := uint64(DefaultChunkSize)
		remainingBytes := contentSize - totalBytesRead

		if remainingBytes < bytesToRead {
			bytesToRead = remainingBytes
		}

		buffer := make([]byte, bytesToRead)
		n, err := io.ReadFull(content, buffer)
		totalBytesRead += uint64(n)

		if err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				return nil, err // A genuine read error
			}
			if n == 0 && totalBytesRead < contentSize && err == io.EOF {
				return nil, ErrReadInconsistentSize
			}
		}

		if n > 0 {
			chunkData := make([]byte, n)
			copy(chunkData, buffer[:n])

			cid, cidErr := GenerateCID(chunkData)
			if cidErr != nil {
				return nil, fmt.Errorf("failed to generate CID for chunk: %w", cidErr)
			}
			chunks = append(chunks, Chunk{Data: chunkData, CID: cid})
		}

		if totalBytesRead >= contentSize {
			break
		}
	}

	if totalBytesRead != contentSize {
		return nil, ErrReadInconsistentSize
	}

	return chunks, nil
}

// PackageGenerateManifest creates a ContentManifestV1 protobuf message from a list of chunks
// and other metadata. It also calculates the CID for this manifest.
// Renamed from GenerateManifest to PackageGenerateManifest.
func PackageGenerateManifest(
	ctx context.Context,
	chunks []Chunk,
	originalContentSHA256 []byte, // Raw SHA256 hash
	originalContentSizeBytes uint64,
	creationTime time.Time, // Go time.Time object
	mimeType string,
	filename string,
	customMeta map[string]string,
) (*manifestv1.ContentManifestV1, string, error) {

	if len(originalContentSHA256) != sha256.Size {
		// Consider using ctx for logging/tracing here in future
		return nil, "", fmt.Errorf("originalContentSHA256 must be %d bytes, got %d", sha256.Size, len(originalContentSHA256))
	}

	chunkCIDs := make([]string, len(chunks))
	for i, chk := range chunks {
		if chk.CID == "" {
			// This should ideally not happen if chunks come from ChunkData which now calculates CIDs
			return nil, "", fmt.Errorf("chunk %d has empty CID", i)
		}
		chunkCIDs[i] = chk.CID
	}

	manifestProto := &manifestv1.ContentManifestV1{
		ChunkCids:                chunkCIDs,
		OriginalContentSha256:    originalContentSHA256,
		OriginalContentSizeBytes: originalContentSizeBytes,
		CreationTimestamp:        creationTime.Unix(), // Store as Unix timestamp (seconds)
		MimeType:                 mimeType,
		Filename:                 filename,
		CustomMetadata:           customMeta,
	}

	serializedManifest, err := proto.Marshal(manifestProto)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal ContentManifestV1: %w", err)
	}

	manifestCID, err := GenerateCID(serializedManifest)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate CID for manifest: %w", err)
	}

	return manifestProto, manifestCID, nil
}
