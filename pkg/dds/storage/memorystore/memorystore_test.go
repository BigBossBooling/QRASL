package memorystore

import (
	"context"
	"fmt" // Added
	"sync"
	"testing"

	"github.com/DigiSocialBlock/EchoNet/pkg/dds/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryStore(t *testing.T) {
	ms := NewMemoryStore()
	require.NotNil(t, ms, "NewMemoryStore should not return nil")
	require.NotNil(t, ms.data, "MemoryStore data map should be initialized")
}

func TestMemoryStore_StoreRetrieveHasDelete(t *testing.T) {
	ms := NewMemoryStore()
	ctx := context.Background()

	cid1 := "testCID1"
	data1 := []byte("hello world")
	cid2 := "testCID2"
	data2 := []byte("another chunk of data")
	nonExistentCID := "cidThatDoesNotExist"

	// Test Store
	err := ms.Store(ctx, cid1, data1)
	require.NoError(t, err, "Store cid1 failed")
	err = ms.Store(ctx, cid2, data2)
	require.NoError(t, err, "Store cid2 failed")

	// Test internal state (optional, but good for this simple map)
	assert.Len(t, ms.data, 2, "Expected 2 items in store after two Stores")

	// Test Has
	has, err := ms.Has(ctx, cid1)
	require.NoError(t, err, "Has cid1 failed")
	assert.True(t, has, "Expected Has(cid1) to be true")

	has, err = ms.Has(ctx, nonExistentCID)
	require.NoError(t, err, "Has nonExistentCID should not error")
	assert.False(t, has, "Expected Has(nonExistentCID) to be false")

	// Test Retrieve
	retrievedData1, err := ms.Retrieve(ctx, cid1)
	require.NoError(t, err, "Retrieve cid1 failed")
	assert.Equal(t, data1, retrievedData1, "Retrieved data1 does not match original")

	// Ensure Retrieve returns a copy
	retrievedData1[0] = 'J' // Modify the retrieved slice
	originalStoredData1, _ := ms.data[cid1] // Access internal for verification
	assert.NotEqual(t, retrievedData1[0], originalStoredData1[0], "Retrieve should return a copy, internal data was modified")
	// Reset data1 for further tests if needed, or use a fresh variable
	data1 = []byte("hello world") // Reset data1 to original state for next assertion if any.

	_, err = ms.Retrieve(ctx, nonExistentCID)
	require.Error(t, err, "Expected error when retrieving nonExistentCID")
	assert.ErrorIs(t, err, storage.ErrChunkNotFound, "Expected ErrChunkNotFound for nonExistentCID")

	// Test Delete
	err = ms.Delete(ctx, cid1)
	require.NoError(t, err, "Delete cid1 failed")
	assert.Len(t, ms.data, 1, "Expected 1 item in store after deleting cid1")


	has, err = ms.Has(ctx, cid1)
	require.NoError(t, err, "Has cid1 after delete failed")
	assert.False(t, has, "Expected Has(cid1) to be false after delete")

	// Test idempotent delete
	err = ms.Delete(ctx, cid1) // Delete again
	require.NoError(t, err, "Idempotent delete for cid1 failed")
	assert.Len(t, ms.data, 1, "Store size should remain 1 after idempotent delete")


	// Test error cases for empty CIDs
	err = ms.Store(ctx, "", data1)
	assert.ErrorIs(t, err, storage.ErrInvalidCIDFormat, "Store with empty CID should error")

	_, err = ms.Retrieve(ctx, "")
	assert.ErrorIs(t, err, storage.ErrInvalidCIDFormat, "Retrieve with empty CID should error")

	_, err = ms.Has(ctx, "")
	assert.ErrorIs(t, err, storage.ErrInvalidCIDFormat, "Has with empty CID should error")

	err = ms.Delete(ctx, "")
	assert.ErrorIs(t, err, storage.ErrInvalidCIDFormat, "Delete with empty CID should error")
}

func TestMemoryStore_StoreNilData(t *testing.T) {
	ms := NewMemoryStore()
	ctx := context.Background()
	cid := "cidForNilData"

	err := ms.Store(ctx, cid, nil)
	require.NoError(t, err, "Storing nil data should not error")

	retrieved, err := ms.Retrieve(ctx, cid)
	require.NoError(t, err, "Retrieving nil data (stored as empty slice) should not error")
	assert.NotNil(t, retrieved, "Retrieved data for nil store should be non-nil (empty slice)")
	assert.Len(t, retrieved, 0, "Retrieved data for nil store should be an empty slice")
}


func TestMemoryStore_Concurrency(t *testing.T) {
	ms := NewMemoryStore()
	ctx := context.Background()
	numGoroutines := 100
	numOperationsPerGoroutine := 100

	var wg sync.WaitGroup

	// Concurrent Stores
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperationsPerGoroutine; j++ {
				cid := fmt.Sprintf("concurrent_cid_%d_%d", goroutineID, j)
				data := []byte(fmt.Sprintf("data_for_%s", cid))
				err := ms.Store(ctx, cid, data)
				assert.NoError(t, err, "Concurrent Store failed for %s", cid)
			}
		}(i)
	}
	wg.Wait()
	expectedTotalItems := numGoroutines * numOperationsPerGoroutine
	assert.Equal(t, expectedTotalItems, len(ms.data), "Mismatch in store size after concurrent Stores")

	// Concurrent Has and Retrieve
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperationsPerGoroutine; j++ {
				cid := fmt.Sprintf("concurrent_cid_%d_%d", goroutineID, j)
				expectedData := []byte(fmt.Sprintf("data_for_%s", cid))

				has, err := ms.Has(ctx, cid)
				assert.NoError(t, err, "Concurrent Has failed for %s", cid)
				assert.True(t, has, "Concurrent Has returned false for existing CID %s", cid)

				retrieved, err := ms.Retrieve(ctx, cid)
				assert.NoError(t, err, "Concurrent Retrieve failed for %s", cid)
				assert.Equal(t, expectedData, retrieved, "Concurrent Retrieve data mismatch for CID %s", cid)
			}
		}(i)
	}
	wg.Wait()

	// Concurrent Deletes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			// Each goroutine deletes a subset of CIDs to avoid race on map iteration for deletion
			// For simplicity, each deletes its own previously created CIDs.
			for j := 0; j < numOperationsPerGoroutine; j++ {
				cid := fmt.Sprintf("concurrent_cid_%d_%d", goroutineID, j)
				err := ms.Delete(ctx, cid)
				assert.NoError(t, err, "Concurrent Delete failed for %s", cid)
			}
		}(i)
	}
	wg.Wait()
	assert.Equal(t, 0, len(ms.data), "Store should be empty after concurrent Deletes")
}
