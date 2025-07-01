package manifestv1

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestContentManifestV1_Serialization(t *testing.T) {
	original := &ContentManifestV1{
		ChunkCids:                []string{"cid1", "cid2", "cid3"},
		OriginalContentSha256:    []byte("original_hash_1234567890123456"), // Should be 32 bytes for SHA256
		OriginalContentSizeBytes: 1024,
		CreationTimestamp:        1678886400, // Example timestamp
		MimeType:                 "text/plain",
		Filename:                 "example.txt",
		CustomMetadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	// Marshal the original message
	marshaledData, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal ContentManifestV1: %v", err)
	}

	// Unmarshal into a new message
	unmarshaled := &ContentManifestV1{}
	if err := proto.Unmarshal(marshaledData, unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal ContentManifestV1: %v", err)
	}

	// Basic field comparisons
	if len(original.ChunkCids) != len(unmarshaled.ChunkCids) {
		t.Errorf("ChunkCids length mismatch: got %d, want %d", len(unmarshaled.ChunkCids), len(original.ChunkCids))
	} else {
		for i, cid := range original.ChunkCids {
			if unmarshaled.ChunkCids[i] != cid {
				t.Errorf("ChunkCids[%d] mismatch: got %s, want %s", i, unmarshaled.ChunkCids[i], cid)
			}
		}
	}

	if !bytes.Equal(original.OriginalContentSha256, unmarshaled.OriginalContentSha256) {
		t.Errorf("OriginalContentSha256 mismatch: got %x, want %x", unmarshaled.OriginalContentSha256, original.OriginalContentSha256)
	}

	if original.OriginalContentSizeBytes != unmarshaled.OriginalContentSizeBytes {
		t.Errorf("OriginalContentSizeBytes mismatch: got %d, want %d", unmarshaled.OriginalContentSizeBytes, original.OriginalContentSizeBytes)
	}

	if original.CreationTimestamp != unmarshaled.CreationTimestamp {
		t.Errorf("CreationTimestamp mismatch: got %d, want %d", unmarshaled.CreationTimestamp, original.CreationTimestamp)
	}

	if original.MimeType != unmarshaled.MimeType {
		t.Errorf("MimeType mismatch: got %s, want %s", unmarshaled.MimeType, original.MimeType)
	}

	if original.Filename != unmarshaled.Filename {
		t.Errorf("Filename mismatch: got %s, want %s", unmarshaled.Filename, original.Filename)
	}

	if len(original.CustomMetadata) != len(unmarshaled.CustomMetadata) {
		t.Errorf("CustomMetadata length mismatch: got %d, want %d", len(unmarshaled.CustomMetadata), len(original.CustomMetadata))
	} else {
		for k, v := range original.CustomMetadata {
			if unmarshaled.CustomMetadata[k] != v {
				t.Errorf("CustomMetadata[%s] mismatch: got %s, want %s", k, unmarshaled.CustomMetadata[k], v)
			}
		}
	}

	// For a more robust check, especially with maps or complex types,
	// using proto.Equal is recommended if available and appropriate.
	if !proto.Equal(original, unmarshaled) {
		t.Errorf("proto.Equal reported a mismatch between original and unmarshaled message.\nOriginal: %v\nUnmarshaled: %v", original, unmarshaled)
	}
}

func TestContentManifestV1_Empty(t *testing.T) {
	original := &ContentManifestV1{} // Empty manifest

	marshaledData, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal empty ContentManifestV1: %v", err)
	}

	unmarshaled := &ContentManifestV1{}
	if err := proto.Unmarshal(marshaledData, unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal empty ContentManifestV1: %v", err)
	}

	if !proto.Equal(original, unmarshaled) {
		t.Errorf("proto.Equal reported a mismatch for empty manifest.\nOriginal: %v\nUnmarshaled: %v", original, unmarshaled)
	}

    if len(unmarshaled.ChunkCids) != 0 {
        t.Errorf("Expected empty ChunkCids, got %d", len(unmarshaled.ChunkCids))
    }
    if len(unmarshaled.OriginalContentSha256) != 0 {
         t.Errorf("Expected empty OriginalContentSha256, got %d bytes", len(unmarshaled.OriginalContentSha256))
    }
    if unmarshaled.OriginalContentSizeBytes != 0 {
        t.Errorf("Expected 0 OriginalContentSizeBytes, got %d", unmarshaled.OriginalContentSizeBytes)
    }
}
