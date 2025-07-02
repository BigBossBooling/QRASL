package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
	"fmt" // Added for Sprintf in dummy AddrInfo
	"crypto/rand" // Added

	"github.com/DigiSocialBlock/EchoNet/internal/protos/dds/manifest/v1"
	"github.com/DigiSocialBlock/EchoNet/pkg/dds/chunking"
	"github.com/DigiSocialBlock/EchoNet/pkg/dds/discovery"
	"github.com/DigiSocialBlock/EchoNet/pkg/dds/network"
	// "github.com/DigiSocialBlock/EchoNet/pkg/dds/storage" // To be removed
	"github.com/DigiSocialBlock/EchoNet/pkg/dds/storage/memorystore"
	"github.com/libp2p/go-libp2p/core/crypto" // Added
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// MockChunker is a mock implementation of the chunking.Chunker interface.
type MockChunker struct {
	mock.Mock
}

func (m *MockChunker) ChunkData(ctx context.Context, content io.Reader, contentSize uint64) ([]chunking.Chunk, error) {
	args := m.Called(ctx, content, contentSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]chunking.Chunk), args.Error(1)
}

func (m *MockChunker) GenerateManifest(
	ctx context.Context,
	chunks []chunking.Chunk,
	originalContentSHA256 []byte,
	originalContentSizeBytes uint64,
	creationTime time.Time,
	mimeType string,
	filename string,
	customMeta map[string]string,
) (*manifestv1.ContentManifestV1, string, error) {
	args := m.Called(ctx, chunks, originalContentSHA256, originalContentSizeBytes, mock.Anything, mimeType, filename, customMeta)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
	}
	return args.Get(0).(*manifestv1.ContentManifestV1), args.String(1), args.Error(2)
}

// MockStorageProvider for testing store failures (simplified)
type MockStorageProvider struct {
	mock.Mock
}

func (m *MockStorageProvider) Store(ctx context.Context, cid string, data []byte) error {
	args := m.Called(ctx, cid, data)
	return args.Error(0)
}
func (m *MockStorageProvider) Retrieve(ctx context.Context, cid string) ([]byte, error) {
	args := m.Called(ctx, cid)
	if args.Get(0) == nil {
		if args.Error(1) != nil {
			return nil, args.Error(1)
		}
		return nil, errors.New("mock retrieve error: data nil and no error specified")
	}
	return args.Get(0).([]byte), args.Error(1)
}
func (m *MockStorageProvider) Has(ctx context.Context, cid string) (bool, error) {
	args := m.Called(ctx, cid)
	return args.Bool(0), args.Error(1)
}
func (m *MockStorageProvider) Delete(ctx context.Context, cid string) error {
	args := m.Called(ctx, cid)
	return args.Error(0)
}


// Helper to generate a predictable CID for given data
func generateTestCID(t *testing.T, data []byte) string {
	cid, err := chunking.GenerateCID(data)
	require.NoError(t, err)
	return cid
}

// Helper to create a dummy peer.AddrInfo with a new random peer.ID for testing
func newRandomDummyAddrInfo(t *testing.T) peer.AddrInfo {
	// Generate a new private key
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader) // Using Ed25519 for simplicity
	require.NoError(t, err, "Failed to generate private key for dummy peer")

	// Get PeerID from private key
	pid, err := peer.IDFromPrivateKey(priv)
	require.NoError(t, err, "Failed to get PeerID from private key")

	// Create a dummy multiaddress
	maddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/12345/p2p/%s", pid.String()))
	require.NoError(t, err, "Failed to create multiaddr for dummy peer")

	return peer.AddrInfo{ID: pid, Addrs: []multiaddr.Multiaddr{maddr}}
}


func TestDDSCoreService_Publish(t *testing.T) {
	ctx := context.Background()

	t.Run("successful publish single chunk", func(t *testing.T) {
		mockChunker := new(MockChunker)
		memStore := memorystore.NewMemoryStore()
		// For Publish tests, network/discovery stubs are not strictly needed yet by the method itself
		stubNetProv := network.NewStubNetworkProvider()
		stubDiscoProv := discovery.NewStubDiscoveryProvider()
		svc, err := NewDDSCoreService(mockChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)

		contentStr := "hello world"
		contentReader := strings.NewReader(contentStr)
		contentSize := uint64(len(contentStr))

		chunk1Data := []byte(contentStr)
		chunk1CID := generateTestCID(t, chunk1Data)
		returnedChunks := []chunking.Chunk{
			{Data: chunk1Data, CID: chunk1CID},
		}

		emptyDataHash := sha256.Sum256([]byte{})
		expectedOriginalHashForMock := emptyDataHash[:]
		expectedSizeForMock := uint64(0)

		manifestProtoForReturn := &manifestv1.ContentManifestV1{
			ChunkCids:                []string{chunk1CID},
			OriginalContentSha256:    expectedOriginalHashForMock,
			OriginalContentSizeBytes: expectedSizeForMock,
			CreationTimestamp:        time.Now().Unix(), // Actual time used in mock return
			MimeType:                 "text/plain",
			Filename:                 "hello.txt",
		}
		serializedManifestForReturn, _ := proto.Marshal(manifestProtoForReturn)
		manifestCIDForReturn := generateTestCID(t, serializedManifestForReturn)

		mockChunker.On("ChunkData", ctx, mock.AnythingOfType("*service.hashingReader"), contentSize).Return(returnedChunks, nil).Once()
		mockChunker.On(
			"GenerateManifest", ctx, returnedChunks, expectedOriginalHashForMock,
			expectedSizeForMock, mock.Anything, "text/plain", "hello.txt", (map[string]string)(nil),
		).Return(manifestProtoForReturn, manifestCIDForReturn, nil).Once()

		publishedManifestCID, err := svc.Publish(ctx, contentReader, contentSize, WithMimeType("text/plain"), WithFilename("hello.txt"))
		require.NoError(t, err)
		assert.Equal(t, manifestCIDForReturn, publishedManifestCID)

		storedChunk, err := memStore.Retrieve(ctx, chunk1CID)
		require.NoError(t, err)
		assert.Equal(t, chunk1Data, storedChunk)
		storedManifestData, err := memStore.Retrieve(ctx, manifestCIDForReturn)
		require.NoError(t, err)
		assert.Equal(t, serializedManifestForReturn, storedManifestData)

		mockChunker.AssertExpectations(t)
	})

	// ... (other Publish test cases remain the same as they passed before) ...
	t.Run("successful publish multiple chunks", func(t *testing.T) {
		mockChunker := new(MockChunker)
		memStore := memorystore.NewMemoryStore()
		stubNetProv := network.NewStubNetworkProvider()
		stubDiscoProv := discovery.NewStubDiscoveryProvider()
		svc, err := NewDDSCoreService(mockChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)

		contentPart1 := strings.Repeat("a", chunking.DefaultChunkSize)
		contentPart2 := "final part"
		fullContent := contentPart1 + contentPart2
		contentReader := strings.NewReader(fullContent)
		contentSize := uint64(len(fullContent))

		chunk1Data := []byte(contentPart1)
		chunk1CID := generateTestCID(t, chunk1Data)
		chunk2Data := []byte(contentPart2)
		chunk2CID := generateTestCID(t, chunk2Data)
		returnedChunks := []chunking.Chunk{
			{Data: chunk1Data, CID: chunk1CID},
			{Data: chunk2Data, CID: chunk2CID},
		}

		emptyDataHash := sha256.Sum256([]byte{})
		expectedOriginalHashForMock := emptyDataHash[:]
		expectedSizeForMock := uint64(0)

		manifestProtoForReturn := &manifestv1.ContentManifestV1{
			ChunkCids:                []string{chunk1CID, chunk2CID},
			OriginalContentSha256:    expectedOriginalHashForMock,
			OriginalContentSizeBytes: expectedSizeForMock,
			CreationTimestamp:        time.Now().Unix(),
		}
		serializedManifestForReturn, _ := proto.Marshal(manifestProtoForReturn)
		manifestCIDForReturn := generateTestCID(t, serializedManifestForReturn)

		mockChunker.On("ChunkData", ctx, mock.AnythingOfType("*service.hashingReader"), contentSize).Return(returnedChunks, nil).Once()
		mockChunker.On(
			"GenerateManifest", ctx, returnedChunks, expectedOriginalHashForMock,
			expectedSizeForMock, mock.Anything, "", "", (map[string]string)(nil),
		).Return(manifestProtoForReturn, manifestCIDForReturn, nil).Once()

		publishedManifestCID, err := svc.Publish(ctx, contentReader, contentSize)
		require.NoError(t, err)
		assert.Equal(t, manifestCIDForReturn, publishedManifestCID)

		s1, _ := memStore.Retrieve(ctx, chunk1CID)
		assert.Equal(t, chunk1Data, s1)
		s2, _ := memStore.Retrieve(ctx, chunk2CID)
		assert.Equal(t, chunk2Data, s2)
		sm, _ := memStore.Retrieve(ctx, manifestCIDForReturn)
		assert.Equal(t, serializedManifestForReturn, sm)

		mockChunker.AssertExpectations(t)
	})

	t.Run("chunking fails", func(t *testing.T) {
		mockChunker := new(MockChunker)
		memStore := memorystore.NewMemoryStore()
		stubNetProv := network.NewStubNetworkProvider()
		stubDiscoProv := discovery.NewStubDiscoveryProvider()
		svc, err := NewDDSCoreService(mockChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)

		contentReader := strings.NewReader("test")
		contentSize := uint64(4)
		expectedError := errors.New("chunking failed")

		mockChunker.On("ChunkData", ctx, mock.AnythingOfType("*service.hashingReader"), contentSize).Return(nil, expectedError).Once()

		_, pubErr := svc.Publish(ctx, contentReader, contentSize)
		require.Error(t, pubErr)
		assert.ErrorIs(t, pubErr, ErrPublishFailed)
		assert.Contains(t, pubErr.Error(), expectedError.Error())
		mockChunker.AssertExpectations(t)
	})

	t.Run("storing a data chunk fails", func(t *testing.T) {
		mockChunker := new(MockChunker)
		failingStore := new(MockStorageProvider)
		stubNetProv := network.NewStubNetworkProvider()
		stubDiscoProv := discovery.NewStubDiscoveryProvider()
		svc, err := NewDDSCoreService(mockChunker, failingStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)

		contentStr := "fail store chunk"
		contentReader := strings.NewReader(contentStr)
		contentSize := uint64(len(contentStr))

		chunk1Data := []byte(contentStr)
		chunk1CID := generateTestCID(t, chunk1Data)
		returnedChunks := []chunking.Chunk{{Data: chunk1Data, CID: chunk1CID}}

		expectedStoreError := errors.New("disk write error")

		mockChunker.On("ChunkData", ctx, mock.AnythingOfType("*service.hashingReader"), contentSize).Return(returnedChunks, nil).Once()
		failingStore.On("Store", ctx, chunk1CID, chunk1Data).Return(expectedStoreError).Once()

		_, pubErr := svc.Publish(ctx, contentReader, contentSize)
		require.Error(t, pubErr)
		assert.ErrorIs(t, pubErr, ErrPublishFailed)
		assert.Contains(t, pubErr.Error(), expectedStoreError.Error())

		mockChunker.AssertExpectations(t)
		failingStore.AssertExpectations(t)
	})

	t.Run("storing the manifest fails", func(t *testing.T) {
		mockChunker := new(MockChunker)
		failingStore := new(MockStorageProvider)
		stubNetProv := network.NewStubNetworkProvider()
		stubDiscoProv := discovery.NewStubDiscoveryProvider()
		svc, err := NewDDSCoreService(mockChunker, failingStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)

		contentStr := "fail manifest store"
		contentReader := strings.NewReader(contentStr)
		contentSize := uint64(len(contentStr))

		chunk1Data := []byte(contentStr)
		chunk1CID := generateTestCID(t, chunk1Data)
		returnedChunks := []chunking.Chunk{{Data: chunk1Data, CID: chunk1CID}}

		// manifestProtoForMockReturn is what the mock GenerateManifest will return.
		// Its content should be consistent with the manifestCIDPlaceholder.
		// For this test, we only need a valid proto object for proto.Marshal to succeed inside Publish.
		// The actual hash/size within this object don't affect the mock expectation for GenerateManifest call args.
		manifestProtoForMockReturn := &manifestv1.ContentManifestV1{
			ChunkCids:                []string{chunk1CID},
			// These fields will be based on the actual content if not for the mock behavior:
			// OriginalContentSha256:    sha256.Sum256([]byte(contentStr))[:],
			// OriginalContentSizeBytes: contentSize,
			CreationTimestamp:        time.Now().Unix(),
		}
		manifestCIDPlaceholder := "dummyManifestCIDFromMock"
		expectedStoreError := errors.New("manifest disk write error")

		// These are the values Publish will pass to GenerateManifest due to mocked ChunkData
		emptyHash := sha256.Sum256([]byte{})
		expectedOriginalHashArg := emptyHash[:]
		expectedSizeArg := uint64(0)

		mockChunker.On("ChunkData", ctx, mock.AnythingOfType("*service.hashingReader"), contentSize).Return(returnedChunks, nil).Once()
		failingStore.On("Store", ctx, chunk1CID, chunk1Data).Return(nil).Once()

		mockChunker.On(
			"GenerateManifest", ctx, returnedChunks, expectedOriginalHashArg,
			expectedSizeArg, mock.Anything, "", "", (map[string]string)(nil),
		).Return(manifestProtoForMockReturn, manifestCIDPlaceholder, nil).Once()

		failingStore.On("Store", ctx, manifestCIDPlaceholder, mock.AnythingOfType("[]uint8")).Return(expectedStoreError).Once()

		_, pubErr := svc.Publish(ctx, contentReader, contentSize)
		require.Error(t, pubErr)
		assert.ErrorIs(t, pubErr, ErrPublishFailed)
		assert.Contains(t, pubErr.Error(), expectedStoreError.Error())

		mockChunker.AssertExpectations(t)
		failingStore.AssertExpectations(t)
	})
}


func TestDDSCoreService_Retrieve(t *testing.T) {
	ctx := context.Background()
	realChunker := chunking.NewDefaultChunker()
	stubNetProv := network.NewStubNetworkProvider()
	stubDiscoProv := discovery.NewStubDiscoveryProvider()

	t.Run("successful retrieve single chunk content (local only)", func(t *testing.T) {
		memStore := memorystore.NewMemoryStore()
		svc, err := NewDDSCoreService(realChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)
		contentStr := "retrieve this"
		publishedManifestCID, err := svc.Publish(ctx, strings.NewReader(contentStr), uint64(len(contentStr)))
		require.NoError(t, err)
		retrievedReader, retrievedSize, err := svc.Retrieve(ctx, publishedManifestCID)
		require.NoError(t, err, "Retrieve failed")
		require.NotNil(t, retrievedReader, "Retrieved reader is nil")
		assert.Equal(t, uint64(len(contentStr)), retrievedSize, "Retrieved content size mismatch")
		retrievedBytes, err := io.ReadAll(retrievedReader)
		require.NoError(t, err, "Failed to read from retrievedReader")
		assert.Equal(t, contentStr, string(retrievedBytes), "Retrieved content data mismatch")
	})

	t.Run("successful retrieve multi-chunk content (local only)", func(t *testing.T) {
		memStore := memorystore.NewMemoryStore()
		svc, err := NewDDSCoreService(realChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)
		contentPart1 := strings.Repeat("x", chunking.DefaultChunkSize)
		contentPart2 := strings.Repeat("y", chunking.DefaultChunkSize/2)
		fullContent := contentPart1 + contentPart2
		publishedManifestCID, err := svc.Publish(ctx, strings.NewReader(fullContent), uint64(len(fullContent)))
		require.NoError(t, err)
		retrievedReader, retrievedSize, err := svc.Retrieve(ctx, publishedManifestCID)
		require.NoError(t, err)
		assert.Equal(t, uint64(len(fullContent)), retrievedSize)
		retrievedBytes, _ := io.ReadAll(retrievedReader)
		assert.Equal(t, fullContent, string(retrievedBytes))
	})

	t.Run("retrieve 0-byte content (local only)", func(t *testing.T) {
		memStore := memorystore.NewMemoryStore()
		svc, err := NewDDSCoreService(realChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)
		publishedManifestCID, err := svc.Publish(ctx, strings.NewReader(""), 0)
		require.NoError(t, err)
		retrievedReader, retrievedSize, err := svc.Retrieve(ctx, publishedManifestCID)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), retrievedSize)
		retrievedBytes, _ := io.ReadAll(retrievedReader)
		assert.Equal(t, "", string(retrievedBytes))
	})

	t.Run("retrieve non-existent manifest CID (local and network)", func(t *testing.T) {
		memStore := memorystore.NewMemoryStore()
		stubNetProv.ClearAll()
		stubDiscoProv.ClearAll()
		svc, err := NewDDSCoreService(realChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)
		_, _, err = svc.Retrieve(ctx, "nonExistentManifestCID")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrRetrieveFailed)
		assert.Contains(t, err.Error(), "not found", "Error message should indicate manifest not found")
	})

	t.Run("retrieve with empty manifest CID", func(t *testing.T) {
		memStore := memorystore.NewMemoryStore()
		svc, err := NewDDSCoreService(realChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)
		_, _, err = svc.Retrieve(ctx, "")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidManifestCID)
	})

	t.Run("retrieve with corrupted manifest (cannot unmarshal - local)", func(t *testing.T) {
		memStore := memorystore.NewMemoryStore()
		svc, err := NewDDSCoreService(realChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)
		corruptedManifestCID := "cidOfCorruptedManifest"
		err = memStore.Store(ctx, corruptedManifestCID, []byte("this is not valid protobuf data"))
		require.NoError(t, err)
		_, _, err = svc.Retrieve(ctx, corruptedManifestCID)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrRetrieveFailed)
		assert.Contains(t, err.Error(), "failed to unmarshal manifest")
	})

	t.Run("retrieve with manifest listing a missing data chunk", func(t *testing.T) {
		memStore := memorystore.NewMemoryStore()
		svc, err := NewDDSCoreService(realChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)
		contentStr := "hello"
		originalHash := sha256.Sum256([]byte(contentStr))
		chunk1Data := []byte(contentStr)
		chunk1CID := generateTestCID(t, chunk1Data)
		manifestProto := &manifestv1.ContentManifestV1{
			ChunkCids:                []string{chunk1CID, "missingChunkCID"},
			OriginalContentSha256:    originalHash[:],
			OriginalContentSizeBytes: uint64(len(contentStr) + 10),
			CreationTimestamp:        time.Now().Unix(),
		}
		serializedManifest, _ := proto.Marshal(manifestProto)
		manifestCID := generateTestCID(t, serializedManifest)
		err = memStore.Store(ctx, manifestCID, serializedManifest)
		require.NoError(t, err)
		err = memStore.Store(ctx, chunk1CID, chunk1Data)
		require.NoError(t, err)
		stubDiscoProv.ClearAll()
		_, _, err = svc.Retrieve(ctx, manifestCID)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrRetrieveFailed)
		assert.Contains(t, err.Error(), "missingChunkCID", "Error should mention missing CID")
		assert.Contains(t, err.Error(), "not found", "Error should indicate missing chunk was not found")
	})

	t.Run("retrieve with corrupted data chunk (CID mismatch - from local)", func(t *testing.T) {
		memStore := memorystore.NewMemoryStore()
		svc, err := NewDDSCoreService(realChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)
		contentStr := "original good data"
		corruptedDataStr := "corrupted data here"
		publishedManifestCID, err := svc.Publish(ctx, strings.NewReader(contentStr), uint64(len(contentStr)))
		require.NoError(t, err)
		manifestData, _ := memStore.Retrieve(ctx, publishedManifestCID)
		retrievedManifestProto := &manifestv1.ContentManifestV1{}
		_ = proto.Unmarshal(manifestData, retrievedManifestProto)
		require.GreaterOrEqual(t, len(retrievedManifestProto.ChunkCids), 1, "Need at least one chunk for this test")
		targetChunkCID := retrievedManifestProto.ChunkCids[0]
		err = memStore.Store(ctx, targetChunkCID, []byte(corruptedDataStr))
		require.NoError(t, err)
		_, _, err = svc.Retrieve(ctx, publishedManifestCID)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrRetrieveFailed)
		assert.Contains(t, err.Error(), "integrity check failed for chunk")
	})

	t.Run("retrieve with reassembled content hash mismatch (from local)", func(t *testing.T) {
		memStore := memorystore.NewMemoryStore()
		svc, err := NewDDSCoreService(realChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)
		contentStr := "good content"
		publishedManifestCID, err := svc.Publish(ctx, strings.NewReader(contentStr), uint64(len(contentStr)))
		require.NoError(t, err)
		manifestData, _ := memStore.Retrieve(ctx, publishedManifestCID)
		retrievedManifestProto := &manifestv1.ContentManifestV1{}
		_ = proto.Unmarshal(manifestData, retrievedManifestProto)
		retrievedManifestProto.OriginalContentSha256 = []byte("totally wrong hash value 1234567")
		corruptedSerializedManifest, _ := proto.Marshal(retrievedManifestProto)
		err = memStore.Store(ctx, publishedManifestCID, corruptedSerializedManifest)
		require.NoError(t, err)
		_, _, err = svc.Retrieve(ctx, publishedManifestCID)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrRetrieveFailed)
		assert.Contains(t, err.Error(), "reassembled content hash mismatch")
	})

	t.Run("retrieve with reassembled content size mismatch (from local)", func(t *testing.T) {
		memStore := memorystore.NewMemoryStore()
		svc, err := NewDDSCoreService(realChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)
		contentStr := "good content for size test"
		publishedManifestCID, err := svc.Publish(ctx, strings.NewReader(contentStr), uint64(len(contentStr)))
		require.NoError(t, err)
		manifestData, _ := memStore.Retrieve(ctx, publishedManifestCID)
		retrievedManifestProto := &manifestv1.ContentManifestV1{}
		_ = proto.Unmarshal(manifestData, retrievedManifestProto)
		retrievedManifestProto.OriginalContentSizeBytes += 10
		corruptedSerializedManifest, _ := proto.Marshal(retrievedManifestProto)
		err = memStore.Store(ctx, publishedManifestCID, corruptedSerializedManifest)
		require.NoError(t, err)
		_, _, err = svc.Retrieve(ctx, publishedManifestCID)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrRetrieveFailed)
		assert.Contains(t, err.Error(), "does not match manifest original size")
	})

	t.Run("successful retrieve manifest from network, chunks from local", func(t *testing.T) {
		memStore := memorystore.NewMemoryStore()
		stubNetProv.ClearAll()
		stubDiscoProv.ClearAll()
		svc, err := NewDDSCoreService(realChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)
		contentStr := "network manifest, local chunks"
		contentReader := strings.NewReader(contentStr)
		contentSize := uint64(len(contentStr))
		tempMemStoreForPublish := memorystore.NewMemoryStore()
		tempSvc, _ := NewDDSCoreService(realChunker, tempMemStoreForPublish, network.NewStubNetworkProvider(), discovery.NewStubDiscoveryProvider())
		publishedManifestCID, _ := tempSvc.Publish(context.Background(), contentReader, contentSize)
		publishedManifestBytes, _ := tempMemStoreForPublish.Retrieve(context.Background(), publishedManifestCID)
		publishedManifestProto := &manifestv1.ContentManifestV1{}
		_ = proto.Unmarshal(publishedManifestBytes, publishedManifestProto)
		for _, chunkCID := range publishedManifestProto.ChunkCids {
			chunkData, _ := tempMemStoreForPublish.Retrieve(context.Background(), chunkCID)
			err = memStore.Store(ctx, chunkCID, chunkData)
			require.NoError(t, err)
		}
		dummyAddrInfo := newRandomDummyAddrInfo(t) // Use new helper
		stubDiscoProv.AddProviderRecord(publishedManifestCID, dummyAddrInfo)
		stubNetProv.SetPeerResponse(dummyAddrInfo.ID, publishedManifestCID, func() ([]byte, error) {
			return publishedManifestBytes, nil
		})
		retrievedReader, retrievedSize, err := svc.Retrieve(ctx, publishedManifestCID)
		require.NoError(t, err, "Retrieve failed")
		assert.Equal(t, contentSize, retrievedSize)
		retrievedBytes, _ := io.ReadAll(retrievedReader)
		assert.Equal(t, contentStr, string(retrievedBytes))
		_, err = memStore.Retrieve(ctx, publishedManifestCID)
		assert.NoError(t, err, "Manifest should have been cached locally")
	})

	t.Run("successful retrieve manifest local, one chunk network, one chunk local", func(t *testing.T) {
		memStore := memorystore.NewMemoryStore()
		stubNetProv.ClearAll()
		stubDiscoProv.ClearAll()
		svc, err := NewDDSCoreService(realChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)
		part1 := strings.Repeat("a", chunking.DefaultChunkSize)
		part2 := "this is part 2"
		fullContent := part1 + part2
		contentReader := strings.NewReader(fullContent)
		contentSize := uint64(len(fullContent))
		tempMemStore := memorystore.NewMemoryStore()
		tempSvc, _ := NewDDSCoreService(realChunker, tempMemStore, network.NewStubNetworkProvider(), discovery.NewStubDiscoveryProvider())
		publishedManifestCID, _ := tempSvc.Publish(context.Background(), contentReader, contentSize)
		manifestBytes, _ := tempMemStore.Retrieve(context.Background(), publishedManifestCID)
		manifestProto := &manifestv1.ContentManifestV1{}
		_ = proto.Unmarshal(manifestBytes, manifestProto)
		require.Len(t, manifestProto.ChunkCids, 2, "Test setup error: expected 2 chunks")
		chunk1CID := manifestProto.ChunkCids[0]
		chunk1Data, _ := tempMemStore.Retrieve(context.Background(), chunk1CID)
		chunk2CID := manifestProto.ChunkCids[1]
		chunk2Data, _ := tempMemStore.Retrieve(context.Background(), chunk2CID)
		err = memStore.Store(ctx, publishedManifestCID, manifestBytes)
		require.NoError(t, err)
		err = memStore.Store(ctx, chunk1CID, chunk1Data)
		require.NoError(t, err)
		dummyAddrInfo := newRandomDummyAddrInfo(t) // Use new helper
		stubDiscoProv.AddProviderRecord(chunk2CID, dummyAddrInfo)
		stubNetProv.SetPeerResponse(dummyAddrInfo.ID, chunk2CID, func() ([]byte, error) {
			return chunk2Data, nil
		})
		retrievedReader, retrievedSize, err := svc.Retrieve(ctx, publishedManifestCID)
		require.NoError(t, err, "Retrieve failed")
		assert.Equal(t, contentSize, retrievedSize)
		retrievedBytes, _ := io.ReadAll(retrievedReader)
		assert.Equal(t, fullContent, string(retrievedBytes))
		_, err = memStore.Retrieve(ctx, chunk2CID)
		assert.NoError(t, err, "Chunk 2 should have been cached locally")
	})

	t.Run("retrieve fails if network peer doesn't have chunk", func(t *testing.T) {
		memStore := memorystore.NewMemoryStore()
		stubNetProv.ClearAll()
		stubDiscoProv.ClearAll()
		svc, err := NewDDSCoreService(realChunker, memStore, stubNetProv, stubDiscoProv)
		require.NoError(t, err)
		tempMemStore := memorystore.NewMemoryStore()
		tempSvc, _ := NewDDSCoreService(realChunker, tempMemStore, network.NewStubNetworkProvider(), discovery.NewStubDiscoveryProvider())
		publishedManifestCID, _ := tempSvc.Publish(context.Background(), strings.NewReader("network fail test"), 17)
		manifestBytes, _ := tempMemStore.Retrieve(context.Background(), publishedManifestCID)
		manifestProto := &manifestv1.ContentManifestV1{}
		_ = proto.Unmarshal(manifestBytes, manifestProto)
		chunkToFailCID := manifestProto.ChunkCids[0]
		err = memStore.Store(ctx, publishedManifestCID, manifestBytes)
		require.NoError(t, err)
		dummyAddrInfo := newRandomDummyAddrInfo(t) // Use new helper
		stubDiscoProv.AddProviderRecord(chunkToFailCID, dummyAddrInfo)
		stubNetProv.SetPeerResponse(dummyAddrInfo.ID, chunkToFailCID, func() ([]byte, error) {
			return nil, errors.New("simulated network fetch error for chunk")
		})
		_, _, err = svc.Retrieve(ctx, publishedManifestCID)
		require.Error(t, err)
		// Check with errors.Is directly
		isChunkFetchFailed := errors.Is(err, ErrChunkFetchFailed)
		assert.True(t, isChunkFetchFailed, "errors.Is should find ErrChunkFetchFailed in the chain. Full error: %v", err)
		// Also check for the specific underlying error from the stub and the wrapper messages
		assert.Contains(t, err.Error(), "simulated network fetch error for chunk", "Error should contain the simulated network error")
		assert.Contains(t, err.Error(), ErrChunkFetchFailed.Error(), "Error message should contain the ErrChunkFetchFailed string")
		assert.Contains(t, err.Error(), ErrRetrieveFailed.Error(), "Error message should contain the ErrRetrieveFailed string")
	})
}
