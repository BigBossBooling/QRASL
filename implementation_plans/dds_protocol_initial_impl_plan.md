# DigiSocialBlock - DDS Protocol Initial Implementation Plan

**Version:** 0.1.0
**Date:** 2025-04-12
**Status:** Draft

## Introduction

This document outlines the initial implementation plan for the Distributed Data Stores (DDS) protocol for DigiSocialBlock's EchoNet. It focuses on defining the Go package structure, core component interfaces, the scope for a Minimum Viable Product (MVP), and interactions with `libp2p`. This plan directly supports the technical specifications laid out in `technical_specifications/dds_protocol_spec.md`.

---

## 1. Go Package Structure & Core Interfaces

A clear package structure and well-defined interfaces are crucial for a modular, testable, and maintainable implementation of the DDS protocol.

### 1.1. Proposed Go Package Structure

The core DDS logic will reside primarily within a `pkg/dds` directory (or similar top-level package, e.g., `internal/dds` if purely internal to a larger binary).

```
pkg/dds/
├── chunker/            // Logic for chunking content and manifest creation.
│   ├── chunker.go
│   └── chunker_test.go
├── manifest/           // Defines and handles ContentManifest logic (beyond just proto).
│   ├── manifest.go
│   └── manifest_test.go
├── storage/            // Interface and implementations for local chunk storage.
│   ├── provider.go     // StorageProvider interface.
│   ├── filestore/      // Example: filesystem-based storage provider.
│   │   ├── filestore.go
│   │   └── filestore_test.go
│   └── memory/         // Example: in-memory storage provider (for testing).
│       ├── memory.go
│       └── memory_test.go
├── discovery/          // DHT interactions for finding and advertising content.
│   ├── discovery.go    // Discovery interface and libp2p DHT wrapper.
│   └── discovery_test.go
├── replication/        // Logic for managing data replication (MVP: basic).
│   ├── manager.go      // ReplicationManager interface.
│   └── manager_test.go
├── network/            // Handles sending/receiving DDS Protobuf messages over libp2p.
│   ├── service.go      // NetworkService interface, message handlers.
│   └── service_test.go
├── dds.go              // Main DDS service coordinating all components.
└── dds_test.go
```
*(Note: The `protos/dds_messages.proto` file and its generated Go code `internal/protos/dds/v1/dds_messages.pb.go` are separate but heavily used by the `network` package and others.)*

### 1.2. Core Go Interfaces (MVP)

These interfaces define the contracts for the primary components of the DDS system.

```go
package dds

import (
	"context"
	"io"

	"github.com/libp2p/go-libp2p/core/peer"
	ddsv1 "github.com/DigiSocialBlock/EchoNet/internal/protos/dds/v1" // Assuming this path for generated protos
)

// Chunk represents a single piece of content data with its CID.
type Chunk struct {
	CID  string // Base58BTC encoded SHA-256 hash
	Data []byte
}

// Manifest defines the structure of a content manifest.
// This mirrors the Protobuf definition but is a Go struct for internal use.
type Manifest struct {
	ChunkCIDs             []string
	OriginalContentSHA256 []byte // Raw 32 bytes
	OriginalContentSizeBytes uint64
	ManifestCID           string // CID of this manifest itself
}

// Chunker defines the operations for content chunking and manifest generation.
type Chunker interface {
	// Chunks content into fixed-size blocks and generates their CIDs.
	ChunkContent(ctx context.Context, reader io.Reader, contentSize uint64) ([]Chunk, error)
	// Generates a ContentManifest from a list of chunks and the original content's hash.
	// Also calculates the CID for the manifest itself.
	GenerateManifest(ctx context.Context, chunks []Chunk, originalContentSHA256 []byte, originalContentSizeBytes uint64) (Manifest, error)
}

// StorageProvider defines the contract for storing and retrieving content chunks locally.
type StorageProvider interface {
	// Store saves the given chunk data, addressable by its CID.
	// Implementations should verify if data matches CID before storing if desired,
	// though primary verification happens at retrieval time by the requester.
	Store(ctx context.Context, cid string, data []byte) error
	// Retrieve fetches the chunk data for the given CID.
	// Returns an error (e.g., ErrChunkNotFound) if the CID is not found.
	Retrieve(ctx context.Context, cid string) ([]byte, error)
	// Has checks if the storage provider has the chunk for the given CID.
	Has(ctx context.Context, cid string) (bool, error)
	// StoredCIDs returns a stream or list of CIDs currently stored by this provider.
    // Useful for advertising to the DHT or for replication checks. (May be advanced/optional for first MVP pass)
    // StoredCIDs(ctx context.Context) (<-chan string, error)
}

// Discovery defines the contract for discovering content providers on the network (e.g., via DHT).
type Discovery interface {
	// Provide announces to the network that this node can provide content for the given CID.
	Provide(ctx context.Context, cid string) error
	// FindProviders queries the network to find peers who can provide content for the given CID.
	// Returns a channel of peer.AddrInfo as they are found.
	FindProviders(ctx context.Context, cid string, count int) (<-chan peer.AddrInfo, error)
}

// ReplicationManager defines the contract for ensuring content chunks are adequately replicated.
// For MVP, this might be a simplified version focusing on initial seeding or basic checks.
type ReplicationManager interface {
	// EnsureReplication is called to check and potentially trigger replication for a given CID
	// if it's found to be under the target replication factor.
	// currentProviders might be a list of known providers from a recent DHT lookup.
	EnsureReplication(ctx context.Context, cid string, knownProviders []peer.ID) error
	// InstructReplicate explicitly tells a target peer to replicate a given CID,
	// potentially hinting at a source.
	InstructReplicate(ctx context.Context, targetPeer peer.ID, cid string, sourceHint peer.ID) error
}

// NetworkService defines the contract for sending and receiving DDS protocol messages over libp2p.
// It would register handlers for incoming DDS message types.
type NetworkService interface {
	// SendStoreChunkRequest sends a StoreChunkRequest to a target peer.
	SendStoreChunkRequest(ctx context.Context, targetPeer peer.ID, req *ddsv1.StoreChunkRequest) (*ddsv1.StoreChunkResponse, error)
	// SendRetrieveChunkRequest sends a RetrieveChunkRequest to a target peer.
	SendRetrieveChunkRequest(ctx context.Context, targetPeer peer.ID, req *ddsv1.RetrieveChunkRequest) (*ddsv1.RetrieveChunkResponse, error)
	// SendReplicationInstruction sends a ReplicationInstruction to a target peer.
    SendReplicationInstruction(ctx context.Context, targetPeer peer.ID, req *ddsv1.ReplicationInstruction) (*ddsv1.ReplicationResponse, error)

	// RegisterRequestHandler allows other components (like StorageProvider) to handle incoming requests.
	// Example: HandleStoreChunk(handler func(ctx context.Context, p peer.ID, req *ddsv1.StoreChunkRequest) (*ddsv1.StoreChunkResponse, error))
}

// DDS represents the main DDS service, coordinating all underlying components.
type DDS interface {
	// Publish ingests content, chunks it, stores it locally, and announces it to the network.
    // Returns the manifest CID.
	Publish(ctx context.Context, contentReader io.Reader, contentSize uint64) (string, error)
	// Retrieve fetches content identified by its manifest CID from the network.
    // Returns an io.Reader for the reassembled content.
	Retrieve(ctx context.Context, manifestCID string) (io.Reader, uint64, error)
}

```

This initial structure and set of interfaces provide a solid foundation for developing the DDS components in a decoupled and testable manner. The specific implementations for these interfaces will be developed according to the MVP scope and subsequent phases.
---

## 2. MVP Scope and Phased Implementation

To manage complexity and deliver value incrementally, the DDS implementation will be approached in phases. The initial focus is on a Minimum Viable Product (MVP) that establishes core functionality, followed by subsequent phases to enhance robustness, scalability, and features.

### 2.1. MVP Focus - Core Functionality

The primary goal of the DDS MVP is to enable basic decentralized storage, discovery, and retrieval of content chunks and manifests with integrity verification.

*   **Content Processing (Client-Side/Originator):**
    *   Implement `chunker.ChunkContent()`: Reliable chunking of `io.Reader` content into 256KiB blocks.
    *   Implement `chunker.GenerateManifest()`: Correct generation of `ContentManifest` (Protobuf structure from `dds_protocol_spec.md`, Section 1.3) including `chunk_cids`, `original_content_sha256`, and `original_content_size_bytes`.
    *   Accurate CID generation (SHA-256 + Base58BTC) for both chunks and manifests.
*   **Local Storage (`storage.StorageProvider`):**
    *   Implement a basic, persistent `FileStore` provider:
        *   `Store(cid, data)`: Saves chunk data to the local filesystem, organized by CID (e.g., using a directory structure derived from the CID to avoid too many files in one directory).
        *   `Retrieve(cid)`: Loads chunk data from the filesystem by CID.
        *   `Has(cid)`: Checks for chunk existence.
    *   Implement an `InMemoryStore` provider (primarily for testing purposes).
*   **Basic Discovery (`discovery.Discovery` via `libp2p` Kademlia DHT):**
    *   Implement `discovery.Provide(cid)`: Node advertises to the libp2p Kademlia DHT that it holds a given CID.
    *   Implement `discovery.FindProviders(cid, count)`: Node queries the libp2p Kademlia DHT to find a specified `count` of peers providing a given CID.
*   **Basic P2P Network Communication (`network.NetworkService`):**
    *   Implement sending and handling of `RetrieveChunkRequest` and `RetrieveChunkResponse` messages between two `libp2p` peers.
    *   Implement sending and handling of `StoreChunkRequest` and `StoreChunkResponse` (primarily for initial seeding/testing replication).
*   **Core DDS Service (`dds.DDS`):**
    *   Implement `dds.Publish(content)`:
        1.  Uses `Chunker` to process content into chunks and a manifest.
        2.  Stores all chunks and the manifest locally using `StorageProvider.Store()`.
        3.  Advertises all CIDs (chunks + manifest) to the DHT using `Discovery.Provide()`.
        4.  (MVP Simplification for Seeding): May attempt to directly `StoreChunk` the manifest and data chunks to a small, pre-configured list of bootstrap/friend nodes.
        5.  Returns the `manifest_cid`.
    *   Implement `dds.Retrieve(manifestCID)`:
        1.  Uses `Discovery.FindProviders()` to find peers holding the `manifestCID`.
        2.  Uses `NetworkService` to send `RetrieveChunkRequest` for the manifest.
        3.  Verifies manifest integrity. Parses manifest.
        4.  For each `chunk_cid` in the manifest:
            a.  Uses `Discovery.FindProviders()` for the `chunk_cid`.
            b.  Uses `NetworkService` to send `RetrieveChunkRequest` for the chunk.
            c.  Verifies chunk integrity.
        5.  Reassembles content.
        6.  Performs final verification against `original_content_sha256`.
        7.  Returns an `io.Reader` for the content and its total size.
*   **Integrity Verification:** All stages of retrieval must rigorously verify CIDs and the final original content hash.
*   **Replication (MVP Simplification):**
    *   No complex automated proactive replication or self-healing in the first MVP pass.
    *   Reliability will primarily come from the Originator Node performing initial seeding to N-1 other nodes (e.g., a few known peers or bootstrap nodes) using direct `StoreChunkRequest` messages.
    *   The `ReplicationManager` interface might have a very basic implementation or be deferred.

### 2.2. Post-MVP Phases (High-Level Outline)

Following a successful MVP, subsequent phases will focus on enhancing the DDS protocol:

*   **Phase 2.A: Robust Replication & Self-Healing:**
    *   Implement full `ReplicationManager` logic for proactive replication by Storage Nodes (monitoring, selecting peers, instructing replication).
    *   Implement mechanisms for nodes to detect and trigger repair of under-replicated content they depend on.
    *   Refine peer selection algorithms for replication (considering latency, capacity, reliability, geo-distribution).
*   **Phase 2.B: Storage Incentives & Economics:**
    *   Integrate DDS operations with the Proof-of-Pristine (PoP) mechanism.
    *   Define how Storage Nodes are rewarded for storing and serving data (interface with `pallet-rewards` or similar).
    *   Implement any necessary proof mechanisms (e.g., Proofs of Retrievability/Storage) if required by the economic model.
*   **Phase 2.C: Network Optimizations & Scalability Enhancements:**
    *   Optimize DHT usage (e.g., batching provider record announcements).
    *   Improve peer selection for retrieval (e.g., preferring closer/faster peers).
    *   Implement more sophisticated caching strategies.
    *   Performance benchmarking and profiling to identify and address bottlenecks.
*   **Phase 2.D: Advanced Security Features:**
    *   Explore and implement further DoS/DDoS resistance measures at the DDS protocol level if needed.
    *   Consider mitigations for specific DHT attacks if they prove problematic.
*   **Phase 2.E: Enhanced Tooling & Observability:**
    *   Develop tools for network diagnostics, monitoring replication status, and DDS health.

This phased approach allows for iterative development, focusing on core value first and then progressively building a more resilient, scalable, and feature-rich Distributed Data Store.
---

## 3. Interaction with `libp2p`

The DDS protocol heavily relies on `libp2p` for its underlying peer-to-peer networking capabilities, including node discovery, secure communication channels, and distributed hash table (DHT) functionality.

### 3.1. Core `libp2p` Modules Utilized

*   **Host (`libp2p.Host`):** Each DDS node will run a `libp2p` Host, which is the entry point for all P2P interactions. It manages network listeners, peer connections, and protocol dispatching.
*   **PeerStore (`peerstore.Peerstore`):** Used to store persistent information about peers, such as their addresses and supported protocols.
*   **ConnectionManager (`connmgr.ConnManager`):** To manage the number of active connections, preventing resource exhaustion.
*   **Kademlia DHT (`kad-dht`):** This is the cornerstone for content discovery (Section 4.1 of `dds_protocol_spec.md`).
    *   DDS nodes will join the `libp2p` Kademlia DHT.
    *   Used for `Provide` operations (advertising CIDs) and `FindProviders` operations (locating peers holding specific CIDs).
*   **Stream Multiplexing (e.g., Yamux, Mplex):** `libp2p` uses stream multiplexers to allow multiple logical communication streams over a single network connection, essential for concurrent DDS operations.
*   **Transport Protocols (e.g., TCP, QUIC):** `libp2p` abstracts the underlying transport, allowing DDS to operate over various standard transport protocols.
*   **Crypto Channels (e.g., Noise, TLS):** All communication between `libp2p` peers is typically secured using encrypted channels, ensuring confidentiality and integrity of DDS messages in transit. `libp2p` handles the handshake and establishment of these secure channels.
*   **Peer Routing (`routing.PeerRouting`):** While DHT provides content routing (CID -> Peers), `libp2p` also includes mechanisms for peer routing (PeerID -> Network Addresses), which is used by the DHT itself and for establishing direct connections.
*   **PubSub (Potentially, Post-MVP):** While not central to the MVP's core CID discovery via DHT, `libp2p` PubSub could be explored in later phases for:
    *   Broadcasting announcements about new, highly popular content to interested peers.
    *   Network-wide signals for events like major protocol upgrades or urgent repair needs for critical system data (if any). This is speculative and requires careful design to avoid network spam/overload.

### 3.2. Mapping DDS Messages to `libp2p` Interactions

The DDS network messages defined in `protos/dds_messages.proto` (see Section 6 of `dds_protocol_spec.md`) will be exchanged using `libp2p`'s stream-oriented communication.

*   **Protocol ID:** A unique Protocol ID string (e.g., `/digisocialblock/dds/0.1.0`) will be defined for the DDS message exchange. Nodes will use this ID to negotiate DDS protocol streams with peers.
*   **Request-Response Pattern:** Most DDS interactions follow a request-response pattern:
    *   `StoreChunkRequest` $\rightarrow$ `StoreChunkResponse`
    *   `RetrieveChunkRequest` $\rightarrow$ `RetrieveChunkResponse`
    *   `ReplicationInstruction` $\rightarrow$ `ReplicationResponse`
    *   **Implementation:** The `NetworkService` (Section 1.2) will be responsible for:
        1.  Opening a new stream to the target peer using the DDS Protocol ID.
        2.  Serializing the Protobuf request message and writing it to the stream.
        3.  Reading the Protobuf response message from the stream.
        4.  Handling timeouts and errors during communication.
        5.  `libp2p`'s `SetStreamHandler` will be used on the receiving side to route incoming DDS messages to the appropriate handlers within the `NetworkService`.
*   **DHT Interaction:**
    *   `FindProvidersRequest` and `FindProvidersResponse` are conceptual DDS messages that map to `libp2p` DHT's `FindProvidersAsync` (or similar) calls and the results they yield. The `Discovery` interface (Section 1.2) will abstract these `libp2p` DHT interactions.
    *   Similarly, `Provide` operations on the `Discovery` interface will translate to `Provide` calls on the `libp2p` DHT.

### 3.3. Peer Identification

*   **`peer.ID`:** DDS will use `libp2p`'s `peer.ID` type for uniquely identifying nodes in the network. `peer.ID` is typically derived from the node's public key.
*   **Multiaddresses:** Network addresses for peers will be represented using `multiaddr` format, as is standard in `libp2p`. The PeerStore will manage known multiaddresses for peers.

By leveraging `libp2p`, the DDS protocol can focus on the application-level logic of distributed storage, replication, and discovery, while relying on `libp2p`'s battle-tested components for the complexities of P2P networking, secure communication, and DHT operations. This promotes modularity and reduces the development burden.
---
```
