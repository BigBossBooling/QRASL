// Package chunking provides a robust toolkit for dividing content into
// standardized chunks, generating unique Content Identifiers (CIDs) for them,
// and creating manifests to describe collections of these chunks.
// It forms a foundational part of the Distributed Data Stores (DDS) protocol
// within DigiSocialBlock's EchoNet, ensuring data integrity and enabling
// content-addressable storage.
package chunking

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"time" // Added

	"github.com/DigiSocialBlock/EchoNet/internal/protos/dds/manifest/v1" // Added
	"github.com/btcsuite/btcutil/base58"
	"google.golang.org/protobuf/proto" // Added
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

// ChunkData reads content from an io.Reader and divides it into Chunks of DefaultChunkSize.
// Each chunk includes its generated CID.
// The contentSize parameter is used to determine how many bytes to read in total.
func ChunkData(content io.Reader, contentSize uint64) ([]Chunk, error) {
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

// GenerateManifest creates a ContentManifestV1 protobuf message from a list of chunks
// and other metadata. It also calculates the CID for this manifest.
func GenerateManifest(
	chunks []Chunk,
	originalContentSHA256 []byte, // Raw SHA256 hash
	originalContentSizeBytes uint64,
	creationTime time.Time, // Go time.Time object
	mimeType string,
	filename string,
	customMeta map[string]string,
) (*manifestv1.ContentManifestV1, string, error) {

	if len(originalContentSHA256) != sha256.Size {
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
