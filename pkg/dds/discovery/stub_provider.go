// Package discovery (continued) - This file provides a stub implementation
// of the DiscoveryProvider interface for testing purposes.
package discovery

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// StubDiscoveryProvider is a mock/stub implementation of the DiscoveryProvider interface.
// It allows simulating responses for finding and providing CIDs.
type StubDiscoveryProvider struct {
	mu sync.RWMutex
	// providers stores a map of CID to a list of PeerIDs that "provide" it.
	providers map[string][]peer.AddrInfo
	// provideErrors allows configuring an error for a specific Provide call. Key: cid
	provideErrors map[string]error
	// findPeersErrors allows configuring an error for a specific FindPeers call. Key: cid
	findPeersErrors map[string]error
	// defaultProvideError is returned if no specific error is set for Provide.
	defaultProvideError error
	// defaultFindPeersError is returned if no specific error is set for FindPeers.
	defaultFindPeersError error
}

// NewStubDiscoveryProvider creates a new StubDiscoveryProvider.
func NewStubDiscoveryProvider() *StubDiscoveryProvider {
	return &StubDiscoveryProvider{
		providers:       make(map[string][]peer.AddrInfo),
		provideErrors:   make(map[string]error),
		findPeersErrors: make(map[string]error),
	}
}

// AddProviderRecord allows pre-populating the stub with a provider for a CID.
func (s *StubDiscoveryProvider) AddProviderRecord(cid string, pi peer.AddrInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[cid] = append(s.providers[cid], pi)
}

// SetProvideError configures a specific error to be returned when Provide is called for a given CID.
func (s *StubDiscoveryProvider) SetProvideError(cid string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.provideErrors[cid] = err
}

// SetFindPeersError configures a specific error to be returned when FindPeers is called for a given CID.
func (s *StubDiscoveryProvider) SetFindPeersError(cid string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.findPeersErrors[cid] = err
}

// SetDefaultProvideError sets a default error for Provide calls.
func (s *StubDiscoveryProvider) SetDefaultProvideError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultProvideError = err
}

// SetDefaultFindPeersError sets a default error for FindPeers calls.
func (s *StubDiscoveryProvider) SetDefaultFindPeersError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultFindPeersError = err
}

// ClearAll clears all configured records and errors.
func (s *StubDiscoveryProvider) ClearAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers = make(map[string][]peer.AddrInfo)
	s.provideErrors = make(map[string]error)
	s.findPeersErrors = make(map[string]error)
	s.defaultProvideError = nil
	s.defaultFindPeersError = nil
}

// Provide implements the DiscoveryProvider interface.
func (s *StubDiscoveryProvider) Provide(ctx context.Context, cid string) error {
	s.mu.Lock() // Lock for write, as we might add self as provider (though stub doesn't explicitly)
	defer s.mu.Unlock()

	if err, ok := s.provideErrors[cid]; ok {
		return err
	}
	if s.defaultProvideError != nil {
		return s.defaultProvideError
	}
	// In a real scenario, this would interact with a DHT.
	// For a stub, we might just log it or assume success if no error is configured.
	// To make it somewhat useful for testing, we can add the "current node" (if known)
	// or just return nil. Let's assume for now it just returns configured errors or nil.
	// fmt.Printf("StubDiscoveryProvider: Provide called for CID %s\n", cid)
	return nil
}

// FindPeers implements the DiscoveryProvider interface.
func (s *StubDiscoveryProvider) FindPeers(ctx context.Context, cid string, count int) ([]peer.AddrInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err, ok := s.findPeersErrors[cid]; ok {
		return nil, err
	}
	if s.defaultFindPeersError != nil {
		return nil, s.defaultFindPeersError
	}

	if providers, ok := s.providers[cid]; ok {
		if count > 0 && len(providers) > count {
			// Return a copy of the sub-slice
			result := make([]peer.AddrInfo, count)
			copy(result, providers[:count])
			return result, nil
		}
		// Return a copy of the slice
		result := make([]peer.AddrInfo, len(providers))
		copy(result, providers)
		return result, nil
	}
	return []peer.AddrInfo{}, nil // No providers found, return empty slice and no error
}

// Ensure StubDiscoveryProvider implements DiscoveryProvider
var _ DiscoveryProvider = (*StubDiscoveryProvider)(nil)
