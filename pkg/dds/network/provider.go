// Package network defines interfaces and components related to the
// peer-to-peer network interactions for DigiSocialBlock's DDS.
// This includes abstractions for fetching data from remote peers.
package network

import (
	"context"
	"errors" // For defining custom error variables

	"github.com/libp2p/go-libp2p/core/peer" // For peer.ID
)

var (
	// ErrChunkNotAvailableFromPeer indicates the target peer does not have or cannot serve the requested chunk.
	ErrChunkNotAvailableFromPeer = errors.New("network: chunk not available from the specified peer")
	// ErrPeerUnreachable indicates the target peer could not be reached.
	ErrPeerUnreachable = errors.New("network: target peer is unreachable")
	// ErrNetworkTimeout indicates a network operation timed out.
	ErrNetworkTimeout = errors.New("network: operation timed out")
	// TODO: Add other specific network errors as they become relevant.
)

// NetworkProvider defines the contract for fetching a content chunk
// from a specific remote peer in the DDS network.
// This abstraction allows the core DDS service to request chunks from the
// network without being tightly coupled to the specific P2P message exchange logic.
type NetworkProvider interface {
	// FetchChunk attempts to retrieve a specific chunk (identified by its CID)
	// from a target peer.
	//
	// Parameters:
	//  - ctx: Context for managing the request's lifecycle, including cancellation.
	//  - cid: The Content ID of the chunk to fetch.
	//  - targetPeerID: The libp2p PeerID of the node from which to fetch the chunk.
	//
	// Returns:
	//  - []byte: The raw data of the chunk if successfully fetched.
	//  - error: An error if the fetch operation fails. This could be due to
	//           network issues, the peer not having the chunk, the peer being unresponsive,
	//           or data integrity validation failure after retrieval (though integrity
	//           is usually checked by the caller against the known CID).
	//           Should return ErrChunkNotAvailableFromPeer, ErrPeerUnreachable, ErrNetworkTimeout
	//           or a wrapped error containing one of these for specific, common failures.
	FetchChunk(ctx context.Context, cid string, targetPeerID peer.ID) ([]byte, error)

	// TODO (Post-MVP): Consider methods for:
	// - Announcing locally available chunks to peers (proactive, not just DHT).
	// - Requesting a peer to store/replicate a chunk.
	//   StoreChunkAtPeer(ctx context.Context, cid string, data []byte, targetPeerID peer.ID) error
}
