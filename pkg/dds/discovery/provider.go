// Package discovery defines interfaces and components related to discovering
// content providers within DigiSocialBlock's DDS network.
// This typically involves interacting with a Distributed Hash Table (DHT) or similar
// peer discovery mechanisms.
package discovery

import (
	"context"
	"errors" // For defining custom error variables

	"github.com/libp2p/go-libp2p/core/peer" // For peer.AddrInfo
)

var (
	// ErrDiscoveryTimeout indicates a discovery operation timed out.
	ErrDiscoveryTimeout = errors.New("discovery: operation timed out")
	// ErrDiscoveryFailed indicates a generic failure during a discovery operation.
	// Implementations may wrap more specific errors with this.
	ErrDiscoveryFailed = errors.New("discovery: operation failed")
	// Note: ErrNoProvidersFound is intentionally omitted as per standard DHT client behavior;
	// an empty slice from FindPeers and a nil error indicates no providers were found.
	// TODO: Add other specific discovery errors as they become relevant (e.g., ErrProvideFailed).
)

// DiscoveryProvider defines the contract for components responsible for finding
// peers that can provide a specific content chunk, identified by its CID, and
// for announcing the availability of content.
type DiscoveryProvider interface {
	// FindPeers queries the discovery mechanism (e.g., DHT) to find a list of
	// peers that are advertising they hold the content for the given CID.
	//
	// Parameters:
	//  - ctx: Context for managing the request's lifecycle.
	//  - cid: The Content ID of the chunk to find providers for.
	//  - count: An optional hint for the maximum number of peer records to return.
	//           If 0 or negative, the implementation may use a default or return all found.
	//
	// Returns:
	//  - []peer.AddrInfo: A slice of AddrInfo for peers found. An empty slice with a nil
	//                     error indicates no providers were found.
	//  - error: An error if the discovery operation itself fails (e.g., network issue,
	//           timeout), such as ErrDiscoveryTimeout or ErrDiscoveryFailed.
	FindPeers(ctx context.Context, cid string, count int) ([]peer.AddrInfo, error)

	// Provide announces to the discovery mechanism that the current node
	// can provide content for the given CID.
	//
	// Parameters:
	//  - ctx: Context for managing the announcement.
	//  - cid: The Content ID being provided.
	//
	// Returns:
	//  - error: An error if the announcement fails (e.g. ErrDiscoveryFailed).
	Provide(ctx context.Context, cid string) error

	// TODO (Post-MVP): Consider methods for:
	// - More fine-grained control over Provide (e.g., TTL for provider records).
	// - Finding closest peers to a CID (relevant for DHTs).
}
