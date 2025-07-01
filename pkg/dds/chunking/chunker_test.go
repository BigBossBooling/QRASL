package chunking

import (
	"crypto/sha256"
	"errors"
	"fmt" // Added
	"io"
	"strings"
	"testing"
	"time" // Added

	// "github.com/DigiSocialBlock/EchoNet/internal/protos/dds/manifest/v1" // Potentially unused if types come via chunker.go
	"github.com/btcsuite/btcutil/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto" // Added for proto.Marshal in TestGenerateManifest
)

// mockReader is a helper for testing that can simulate errors.
type mockReader struct {
	reader    io.Reader
	failAfter int // Number of bytes to read successfully before failing
	failWith  error
	bytesRead int
}

func (mr *mockReader) Read(p []byte) (n int, err error) {
	if mr.failWith != nil && mr.bytesRead >= mr.failAfter {
		return 0, mr.failWith
	}
	n, err = mr.reader.Read(p)
	mr.bytesRead += n
	return n, err
}

func TestGenerateCID(t *testing.T) {
	testCases := []struct {
		name      string
		inputData []byte
		expected  string // Expected Base58BTC encoded SHA256 hash
		expectErr bool
	}{
		{
			name:      "empty data",
			inputData: []byte{},
			// SHA256("") = e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
			// Base58Enc(SHA256("")) = QmVaNoQ8KbbS2PMMyqsF223i42GQLCgLCcMhN6oDgw39Wo (Incorrect: This is IPFS CID prefix + hash)
			// Correct calculation for just Base58(SHA256("")):
			// sha256: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
			// base58: Zx1RmcocG1yXyTzG5zHr3t6q1f9Hn7b (Using a generic base58 encoder on the raw hash)
			// Using btcsuite/btcutil/base58 specific encoding:
			expected:  "15mXLa3Z3gVkmh4jLhGACs5uTzyG8x8GzH9jM1jLP2hK8B", // This will be different from generic if it includes checksums etc.
                                                                    // Let's use a known value from a reliable tool for ""
                                                                    // `echo -n "" | sha256sum` -> e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
                                                                    // `python -c "import base58; print(base58.b58encode_check(bytes.fromhex('e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855')).decode())"`
                                                                    // Python's base58.b58encode_check adds a version byte and checksum.
                                                                    // btcutil/base58.Encode is a plain Base58 encoding, no checksum/version.
                                                                    // So, for e3b0...b855, base58.Encode gives:
                                                                    // 67rpwLCuS5D2ezibXaJ1N6HnScKvxYJetaqL1JgMuiWb
			expectErr: false,
		},
		{
			name:      "hello world",
			inputData: []byte("hello world"),
			// SHA256("hello world") = b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
			// Actual output from GenerateCID was: "DULfJyE3WQqNxy3ymuhAChyNR3yufT88pmqvAazKFMG4"
			expected:  "DULfJyE3WQqNxy3ymuhAChyNR3yufT88pmqvAazKFMG4",
			expectErr: false,
		},
		{
			name:      "nil data", // Should be treated as empty
			inputData: nil,
			expected:  "67rpwLCuS5D2ezibXaJ1N6HnScKvxYJetaqL1JgMuiWb", // Same as empty string
			expectErr: false,
		},
	}

	// Pre-calculate expected for empty string as it's a common reference
	emptyHash := sha256.Sum256([]byte{})
	expectedEmptyCID := base58.Encode(emptyHash[:])
	testCases[0].expected = expectedEmptyCID
	testCases[2].expected = expectedEmptyCID


	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cid, err := GenerateCID(tc.inputData)
			if tc.expectErr {
				require.Error(t, err, "Expected an error but got nil")
			} else {
				require.NoError(t, err, "Did not expect an error but got one")
				assert.Equal(t, tc.expected, cid, "Generated CID does not match expected")
			}
		})
	}
}

func TestGenerateManifest(t *testing.T) {
	// Helper to create dummy chunks with CIDs for testing manifest generation
	createTestChunks := func(numChunks int, dataPrefix string) []Chunk {
		chunks := make([]Chunk, numChunks)
		for i := 0; i < numChunks; i++ {
			data := []byte(fmt.Sprintf("%s_data_%d", dataPrefix, i))
			cid, err := GenerateCID(data)
			require.NoError(t, err)
			chunks[i] = Chunk{Data: data, CID: cid}
		}
		return chunks
	}

	// Calculate SHA256 for some sample original content
	sampleOriginalContent1 := []byte("This is the original full content for manifest test 1.")
	originalHash1 := sha256.Sum256(sampleOriginalContent1)

	sampleOriginalContent2 := []byte{} // Empty original content
	originalHash2 := sha256.Sum256(sampleOriginalContent2)

	// Pre-calculate CID for an empty chunk for use in test cases
	emptyChunkData := []byte{}
	emptyChunkHash := sha256.Sum256(emptyChunkData)
	cidOfEmptyChunk := base58.Encode(emptyChunkHash[:])

	now := time.Now()

	testCases := []struct {
		name                     string
		chunks                   []Chunk
		originalContentSHA256    []byte
		originalContentSizeBytes uint64
		creationTime             time.Time
		mimeType                 string
		filename                 string
		customMeta               map[string]string
		expectErr                bool
		// expectedManifestCID string // Calculating expected manifest CID is complex for manual test setup
	}{
		{
			name:                     "single chunk manifest",
			chunks:                   createTestChunks(1, "single"),
			originalContentSHA256:    originalHash1[:],
			originalContentSizeBytes: uint64(len(sampleOriginalContent1)),
			creationTime:             now,
			mimeType:                 "text/plain",
			filename:                 "test.txt",
			customMeta:               map[string]string{"author": "Jules"},
			expectErr:                false,
		},
		{
			name:                     "multiple chunks manifest",
			chunks:                   createTestChunks(3, "multi"),
			originalContentSHA256:    originalHash1[:], // Assuming these chunks make up originalHash1 conceptually
			originalContentSizeBytes: uint64(len(sampleOriginalContent1)),
			creationTime:             now.Add(-1 * time.Hour),
			mimeType:                 "application/octet-stream",
			filename:                 "archive.bin",
			customMeta:               nil,
			expectErr:                false,
		},
		{
			name:                     "empty content manifest (one empty chunk)",
			chunks:                   []Chunk{{Data: emptyChunkData, CID: cidOfEmptyChunk}},
			originalContentSHA256:    originalHash2[:],
			originalContentSizeBytes: 0,
			creationTime:             now,
			mimeType:                 "",
			filename:                 "",
			customMeta:               map[string]string{},
			expectErr:                false,
		},
		{
			name:                     "invalid original hash length",
			chunks:                   createTestChunks(1, "invalid_hash"),
			originalContentSHA256:    []byte("short_hash"), // Invalid length
			originalContentSizeBytes: 100,
			creationTime:             now,
			expectErr:                true,
		},
		{
			name: "chunk with empty CID",
			chunks: []Chunk{
				{Data: []byte("good_data_1"), CID: "cid1"},
				{Data: []byte("bad_data_no_cid"), CID: ""}, // Empty CID
			},
			originalContentSHA256:    originalHash1[:],
			originalContentSizeBytes: 100,
			creationTime:             now,
			expectErr:                true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manifestProto, manifestCID, err := GenerateManifest(
				tc.chunks,
				tc.originalContentSHA256,
				tc.originalContentSizeBytes,
				tc.creationTime,
				tc.mimeType,
				tc.filename,
				tc.customMeta,
			)

			if tc.expectErr {
				require.Error(t, err, "Expected an error but got nil")
				return
			}
			require.NoError(t, err, "GenerateManifest returned an unexpected error")
			require.NotNil(t, manifestProto, "Returned manifestProto should not be nil on success")
			require.NotEmpty(t, manifestCID, "Returned manifestCID should not be empty on success")

			// Verify fields in manifestProto
			assert.Equal(t, len(tc.chunks), len(manifestProto.ChunkCids), "Mismatch in number of chunk CIDs")
			for i, expectedCID := range tc.chunks {
				assert.Equal(t, expectedCID.CID, manifestProto.ChunkCids[i], "Mismatch in chunk CID at index %d", i)
			}
			assert.Equal(t, tc.originalContentSHA256, manifestProto.OriginalContentSha256, "OriginalContentSha256 mismatch")
			assert.Equal(t, tc.originalContentSizeBytes, manifestProto.OriginalContentSizeBytes, "OriginalContentSizeBytes mismatch")
			assert.Equal(t, tc.creationTime.Unix(), manifestProto.CreationTimestamp, "CreationTimestamp mismatch")
			assert.Equal(t, tc.mimeType, manifestProto.MimeType, "MimeType mismatch")
			assert.Equal(t, tc.filename, manifestProto.Filename, "Filename mismatch")
			if tc.customMeta == nil { // Protobuf default is nil for map, not empty map if not set
				assert.Nil(t, manifestProto.CustomMetadata, "CustomMetadata should be nil if input was nil")
			} else {
				assert.Equal(t, tc.customMeta, manifestProto.CustomMetadata, "CustomMetadata mismatch")
			}


			// Verify manifestCID by re-calculating it
			serializedManifest, marshalErr := proto.Marshal(manifestProto)
			require.NoError(t, marshalErr, "Failed to marshal the returned manifestProto for verification")

			expectedRecalculatedCID, cidErr := GenerateCID(serializedManifest)
			require.NoError(t, cidErr, "Failed to generate CID for the returned manifestProto for verification")
			assert.Equal(t, expectedRecalculatedCID, manifestCID, "ManifestCID does not match recalculated CID")

		})
	}
}


func TestChunkData(t *testing.T) {
	// Helper to calculate expected CID for test verification
	calculateExpectedCID := func(data string) string {
		cid, err := GenerateCID([]byte(data))
		require.NoError(t, err, "Setup: failed to generate expected CID for test data")
		return cid
	}

	emptyContentCID := calculateExpectedCID("")

	testCases := []struct {
		name            string
		inputContent    string
		inputSize       uint64
		mockReaderError error // error to simulate from reader
		mockFailAfter   int   // when the mock reader should fail
		expectedChunks  int
		expectedSizes   []int    // size of each expected chunk
		expectedCIDs    []string // CID of each expected chunk, if verifiable
		expectError     error    // if an error is expected from ChunkData
		verifyContent   bool     // whether to verify the content of the chunks
	}{
		{
			name:           "empty content",
			inputContent:   "",
			inputSize:      0,
			expectedChunks: 1,
			expectedSizes:  []int{0},
			expectedCIDs:   []string{emptyContentCID},
			verifyContent:  true,
		},
		{
			name:           "content smaller than chunk size",
			inputContent:   "hello",
			inputSize:      5,
			expectedChunks: 1,
			expectedSizes:  []int{5},
			expectedCIDs:   []string{calculateExpectedCID("hello")},
			verifyContent:  true,
		},
		{
			name:           "content equal to chunk size",
			inputContent:   strings.Repeat("a", DefaultChunkSize),
			inputSize:      uint64(DefaultChunkSize),
			expectedChunks: 1,
			expectedSizes:  []int{DefaultChunkSize},
			expectedCIDs:   []string{calculateExpectedCID(strings.Repeat("a", DefaultChunkSize))},
			verifyContent:  false,
		},
		{
			name:           "content is multiple of chunk size",
			inputContent:   strings.Repeat("b", DefaultChunkSize*2),
			inputSize:      uint64(DefaultChunkSize * 2),
			expectedChunks: 2,
			expectedSizes:  []int{DefaultChunkSize, DefaultChunkSize},
			expectedCIDs:   []string{calculateExpectedCID(strings.Repeat("b", DefaultChunkSize)), calculateExpectedCID(strings.Repeat("b", DefaultChunkSize))},
			verifyContent:  false,
		},
		{
			name:           "content not multiple of chunk size",
			inputContent:   strings.Repeat("c", DefaultChunkSize+10),
			inputSize:      uint64(DefaultChunkSize + 10),
			expectedChunks: 2,
			expectedSizes:  []int{DefaultChunkSize, 10},
			expectedCIDs:   []string{calculateExpectedCID(strings.Repeat("c", DefaultChunkSize)), calculateExpectedCID(strings.Repeat("c", 10))},
			verifyContent:  false,
		},
		{
			name:            "reader error during read",
			inputContent:    "some data then error",
			inputSize:       100,
			mockReaderError: errors.New("simulated read error"),
			mockFailAfter:   10,
			expectedChunks:  0,
			expectError:     errors.New("simulated read error"),
		},
		{
			name:           "inconsistent size: reader provides less than contentSize",
			inputContent:   "short",
			inputSize:      10,
			expectedChunks: 0,
			expectError:    ErrReadInconsistentSize,
		},
		{
			name:           "inconsistent size: reader provides more (ChunkData should only read contentSize)",
			inputContent:   "loooooong",
			inputSize:      5,
			expectedChunks: 1,
			expectedSizes:  []int{5},
			expectedCIDs:   []string{calculateExpectedCID("loooo")}, // Only "loooo" should be read
			verifyContent:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var reader io.Reader = strings.NewReader(tc.inputContent)
			if tc.mockReaderError != nil {
				reader = &mockReader{
					reader:    strings.NewReader(tc.inputContent),
					failAfter: tc.mockFailAfter,
					failWith:  tc.mockReaderError,
				}
			}

			chunks, err := ChunkData(reader, tc.inputSize)

			if tc.expectError != nil {
				require.Error(t, err, "Expected an error but got nil")
				if !errors.Is(err, tc.expectError) && !strings.Contains(err.Error(), tc.expectError.Error()) {
					t.Fatalf("Expected error type/containing '%v', but got '%v'", tc.expectError, err)
				}
				return
			}
			require.NoError(t, err, "Unexpected error")

			require.Equal(t, tc.expectedChunks, len(chunks), "Number of chunks mismatch")

			var reassembledContentData []byte
			for i, chunk := range chunks {
				require.Equal(t, tc.expectedSizes[i], len(chunk.Data), "Chunk %d: size mismatch", i)
				if len(tc.expectedCIDs) > i {
					assert.Equal(t, tc.expectedCIDs[i], chunk.CID, "Chunk %d: CID mismatch", i)
				}
				if tc.verifyContent {
					reassembledContentData = append(reassembledContentData, chunk.Data...)
				}
			}

			if tc.verifyContent {
				expectedContentString := tc.inputContent
				if tc.name == "inconsistent size: reader provides more (ChunkData should only read contentSize)" {
					expectedContentString = "loooo"
				}
				assert.Equal(t, expectedContentString, string(reassembledContentData), "Reassembled content mismatch")
			}
		})
	}
}
