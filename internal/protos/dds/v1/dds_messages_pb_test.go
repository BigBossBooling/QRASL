package ddsv1

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestStoreChunkRequest_Serialization(t *testing.T) {
	original := &StoreChunkRequest{
		Cid:       "cid_for_store_request_data_chunk_123",
		ChunkData: []byte("This is some chunk data to be stored."),
	}

	marshaledData, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal StoreChunkRequest: %v", err)
	}

	unmarshaled := &StoreChunkRequest{}
	if err := proto.Unmarshal(marshaledData, unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal StoreChunkRequest: %v", err)
	}

	if !proto.Equal(original, unmarshaled) {
		t.Errorf("proto.Equal reported a mismatch for StoreChunkRequest.\nOriginal: %v\nUnmarshaled: %v", original, unmarshaled)
	}
	if original.Cid != unmarshaled.Cid {
		t.Errorf("Cid mismatch: got %s, want %s", unmarshaled.Cid, original.Cid)
	}
	if !bytes.Equal(original.ChunkData, unmarshaled.ChunkData) {
		t.Errorf("ChunkData mismatch: got %s, want %s", string(unmarshaled.ChunkData), string(original.ChunkData))
	}
}

func TestStoreChunkResponse_Serialization(t *testing.T) {
	testCases := []struct {
		name     string
		original *StoreChunkResponse
	}{
		{
			name: "success",
			original: &StoreChunkResponse{
				Success:      true,
				ErrorCode:    0,
				ErrorMessage: "",
			},
		},
		{
			name: "failure",
			original: &StoreChunkResponse{
				Success:      false,
				ErrorCode:    101, // Example error code
				ErrorMessage: "Failed to store chunk: disk full",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			marshaledData, err := proto.Marshal(tc.original)
			if err != nil {
				t.Fatalf("Failed to marshal StoreChunkResponse: %v", err)
			}

			unmarshaled := &StoreChunkResponse{}
			if err := proto.Unmarshal(marshaledData, unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal StoreChunkResponse: %v", err)
			}

			if !proto.Equal(tc.original, unmarshaled) {
				t.Errorf("proto.Equal reported a mismatch for StoreChunkResponse.\nOriginal: %v\nUnmarshaled: %v", tc.original, unmarshaled)
			}
		})
	}
}

func TestRetrieveChunkRequest_Serialization(t *testing.T) {
	original := &RetrieveChunkRequest{
		Cid: "cid_for_retrieve_request_data_chunk_456",
	}

	marshaledData, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal RetrieveChunkRequest: %v", err)
	}

	unmarshaled := &RetrieveChunkRequest{}
	if err := proto.Unmarshal(marshaledData, unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal RetrieveChunkRequest: %v", err)
	}

	if !proto.Equal(original, unmarshaled) {
		t.Errorf("proto.Equal reported a mismatch for RetrieveChunkRequest.\nOriginal: %v\nUnmarshaled: %v", original, unmarshaled)
	}
}

func TestRetrieveChunkResponse_Serialization(t *testing.T) {
	testCases := []struct {
		name     string
		original *RetrieveChunkResponse
	}{
		{
			name: "success with data",
			original: &RetrieveChunkResponse{
				ChunkData:    []byte("Here is the retrieved chunk data."),
				Success:      true,
				ErrorCode:    0,
				ErrorMessage: "",
			},
		},
		{
			name: "failure chunk not found",
			original: &RetrieveChunkResponse{
				ChunkData:    nil,
				Success:      false,
				ErrorCode:    404, // Example error code for Not Found
				ErrorMessage: "Chunk not found",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			marshaledData, err := proto.Marshal(tc.original)
			if err != nil {
				t.Fatalf("Failed to marshal RetrieveChunkResponse: %v", err)
			}

			unmarshaled := &RetrieveChunkResponse{}
			if err := proto.Unmarshal(marshaledData, unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal RetrieveChunkResponse: %v", err)
			}

			if !proto.Equal(tc.original, unmarshaled) {
				t.Errorf("proto.Equal reported a mismatch for RetrieveChunkResponse.\nOriginal: %v\nUnmarshaled: %v", tc.original, unmarshaled)
			}
		})
	}
}

func TestFindProvidersRequest_Serialization(t *testing.T) {
    original := &FindProvidersRequest{
        Cid: "manifest_cid_to_find_providers_for_789",
    }
    marshaledData, err := proto.Marshal(original)
    if err != nil {
        t.Fatalf("Failed to marshal FindProvidersRequest: %v", err)
    }
    unmarshaled := &FindProvidersRequest{}
    if err := proto.Unmarshal(marshaledData, unmarshaled); err != nil {
        t.Fatalf("Failed to unmarshal FindProvidersRequest: %v", err)
    }
    if !proto.Equal(original, unmarshaled) {
        t.Errorf("proto.Equal reported a mismatch for FindProvidersRequest.\nOriginal: %v\nUnmarshaled: %v", original, unmarshaled)
    }
}

func TestFindProvidersResponse_Serialization(t *testing.T) {
    original := &FindProvidersResponse{
        PeerIds:      []string{"peerID1_abc", "peerID2_def", "peerID3_ghi"},
        Success:      true,
        ErrorCode:    0,
        ErrorMessage: "",
    }
    marshaledData, err := proto.Marshal(original)
    if err != nil {
        t.Fatalf("Failed to marshal FindProvidersResponse: %v", err)
    }
    unmarshaled := &FindProvidersResponse{}
    if err := proto.Unmarshal(marshaledData, unmarshaled); err != nil {
        t.Fatalf("Failed to unmarshal FindProvidersResponse: %v", err)
    }
    if !proto.Equal(original, unmarshaled) {
        t.Errorf("proto.Equal reported a mismatch for FindProvidersResponse.\nOriginal: %v\nUnmarshaled: %v", original, unmarshaled)
    }
}
// Minimal tests for ReplicationInstruction and ReplicationResponse
func TestReplicationInstruction_Serialization(t *testing.T) {
    original := &ReplicationInstruction{Cid: "cid_to_replicate", SourcePeerIdHint: "source_peer_id"}
    marshaledData, err := proto.Marshal(original)
    if err != nil { t.Fatalf("Marshal ReplicationInstruction failed: %v", err) }
    unmarshaled := &ReplicationInstruction{}
    if err := proto.Unmarshal(marshaledData, unmarshaled); err != nil { t.Fatalf("Unmarshal ReplicationInstruction failed: %v", err) }
    if !proto.Equal(original, unmarshaled) { t.Errorf("ReplicationInstruction mismatch: got %v, want %v", unmarshaled, original) }
}

func TestReplicationResponse_Serialization(t *testing.T) {
    original := &ReplicationResponse{Success: true}
    marshaledData, err := proto.Marshal(original)
    if err != nil { t.Fatalf("Marshal ReplicationResponse failed: %v", err) }
    unmarshaled := &ReplicationResponse{}
    if err := proto.Unmarshal(marshaledData, unmarshaled); err != nil { t.Fatalf("Unmarshal ReplicationResponse failed: %v", err) }
    if !proto.Equal(original, unmarshaled) { t.Errorf("ReplicationResponse mismatch: got %v, want %v", unmarshaled, original) }
}
