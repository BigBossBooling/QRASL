// Package network (continued) - This file provides a stub implementation
// of the NetworkProvider interface for testing purposes.
package network

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

// StubNetworkProvider is a mock/stub implementation of the NetworkProvider interface.
// It allows simulating network responses for fetching chunks.
type StubNetworkProvider struct {
	mu sync.RWMutex
	// chunks stores data that this stub can "serve".
	// Key is CID, value is chunk data.
	chunks map[string][]byte
	// peerResponses allows configuring specific responses or errors for a peer-CID combination.
	// Key: peer.ID.String() + ":" + cid
	// Value: func() ([]byte, error) that generates the response.
	peerResponses map[string]func() ([]byte, error)
	// defaultError, if set, will be returned for any FetchChunk call not covered by peerResponses.
	defaultError error
}

// NewStubNetworkProvider creates a new StubNetworkProvider.
func NewStubNetworkProvider() *StubNetworkProvider {
	return &StubNetworkProvider{
		chunks:        make(map[string][]byte),
		peerResponses: make(map[string]func() ([]byte, error)),
	}
}

// AddChunk allows pre-populating the stub with a chunk it can "serve" generally.
// If a specific peerResponse is set for this CID and a peer, that will take precedence.
func (s *StubNetworkProvider) AddChunk(cid string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chunks[cid] = data
}

// SetPeerResponse configures a specific function to generate a response when
// FetchChunk is called for a particular targetPeerID and CID.
// The responseFunc should return the data and error to simulate.
func (s *StubNetworkProvider) SetPeerResponse(targetPeerID peer.ID, cid string, responseFunc func() ([]byte, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := targetPeerID.String() + ":" + cid
	s.peerResponses[key] = responseFunc
}

// SetDefaultError sets a default error to be returned by FetchChunk if no other
// specific response or chunk is configured.
func (s *StubNetworkProvider) SetDefaultError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultError = err
}

// ClearResponses clears all configured peer-specific responses and default error.
func (s *StubNetworkProvider) ClearResponses() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.peerResponses = make(map[string]func() ([]byte, error))
	s.defaultError = nil
}

// ClearChunks removes all generally added chunks.
func (s *StubNetworkProvider) ClearChunks() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chunks = make(map[string][]byte)
}

// ClearAll clears all configured responses, chunks, and default errors.
func (s *StubNetworkProvider) ClearAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chunks = make(map[string][]byte)
	s.peerResponses = make(map[string]func() ([]byte, error))
	s.defaultError = nil
}


// FetchChunk implements the NetworkProvider interface.
// It checks for a peer-specific response first, then for a generally added chunk,
// then returns defaultError if set, otherwise a generic "not found in stub" error.
func (s *StubNetworkProvider) FetchChunk(ctx context.Context, cid string, targetPeerID peer.ID) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check for peer-specific configured response
	key := targetPeerID.String() + ":" + cid
	if responseFunc, ok := s.peerResponses[key]; ok {
		return responseFunc()
	}

	// Check for generally available chunk (ignoring specific peer for this simple stub)
	if data, ok := s.chunks[cid]; ok {
		// Return a copy to prevent modification
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)
		return dataCopy, nil
	}

	if s.defaultError != nil {
		return nil, s.defaultError
	}

	return nil, fmt.Errorf("stubNetworkProvider: chunk %s not found for peer %s or generally", cid, targetPeerID.String())
}

// Ensure StubNetworkProvider implements NetworkProvider
var _ NetworkProvider = (*StubNetworkProvider)(nil)
