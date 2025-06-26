# DigiSocialBlock - DDS Protocol Testing Strategy

**Version:** 0.1.0
**Date:** 2025-04-12
**Status:** Draft

## Introduction

This document outlines the testing strategy for the Distributed Data Stores (DDS) protocol of DigiSocialBlock's EchoNet. It covers unit testing for individual components, integration testing for interactions between components, and high-level considerations for performance and scalability testing in later stages. This strategy supports the technical specifications in `technical_specifications/dds_protocol_spec.md` and the implementation plan in `implementation_plans/dds_protocol_initial_impl_plan.md`.

---

## 1. Unit Testing Strategy for DDS Components

Unit tests are crucial for verifying the correctness of individual DDS components in isolation. They should be comprehensive, covering happy paths, edge cases, and error conditions. Standard Go testing practices (`testing` package, table-driven tests where appropriate) will be used.

### 1.1. `chunker` Package

*   **Objective:** Verify correct content chunking, CID generation, and `ContentManifest` creation.
*   **Test Cases for `ChunkContent()`:**
    *   Content smaller than one chunk: ensure single chunk produced, correct data, correct CID.
    *   Content exactly one chunk size: ensure single chunk, correct data, correct CID.
    *   Content multiple full chunks: ensure correct number of chunks, correct data in each, correct CIDs.
    *   Content multiple chunks with a final partial chunk: ensure correct number of chunks, correct data (especially in final partial chunk), correct CIDs.
    *   Empty content (0 bytes): ensure one 0-byte chunk is produced with its specific CID.
    *   Large content (multiple MBs): ensure correct chunking without excessive memory usage (test with `io.Reader` mocks if needed).
    *   Error handling: e.g., `io.Reader` returning an error during read.
*   **Test Cases for `GenerateManifest()`:**
    *   Manifest for single chunk content: verify `chunk_cids` list, `original_content_sha256`, `original_content_size_bytes`, and the manifest's own CID.
    *   Manifest for multi-chunk content: verify all fields and the manifest's CID.
    *   Manifest for empty content.
    *   Ensure `manifest_cid` changes if any input to `ContentManifest` (e.g., chunk order, original hash) changes.
*   **CID Generation Utilities (if any helpers exist):**
    *   Test SHA-256 hashing for known byte inputs.
    *   Test Base58BTC encoding and decoding for known hashes.

### 1.2. `storage` Package (`StorageProvider` implementations)

*   **Objective:** Verify correct local storage and retrieval of chunks for each `StorageProvider` implementation (e.g., `FileStore`, `InMemoryStore`).
*   **Common Test Cases (for each implementation):**
    *   `Store()` and `Retrieve()`:
        *   Store a chunk, retrieve it, verify data matches.
        *   Attempt to retrieve a non-existent CID (expect `ErrChunkNotFound` or similar).
        *   Store duplicate CID (should overwrite or be idempotent, depending on design).
    *   `Has()`:
        *   Check for an existing CID (expect true).
        *   Check for a non-existent CID (expect false).
    *   Error Handling:
        *   `FileStore`: Test scenarios like disk full (requires mocking filesystem operations or careful test setup), permission errors.
        *   `InMemoryStore`: Test interaction with configured size limits if applicable.
*   **`FileStore` Specifics:**
    *   Verify correct directory structure and filename conventions for storing chunks by CID.
    *   Test cleanup of temporary files if any are used during store.
*   **`InMemoryStore` Specifics:**
    *   Verify it works correctly without disk I/O.

### 1.3. `discovery` Package (`Discovery` interface implementation)

*   **Objective:** Verify interaction with the `libp2p` Kademlia DHT (likely using a mocked DHT or `libp2p` testnet for unit/local integration tests).
*   **Test Cases for `Provide(cid)`:**
    *   Call `Provide`, verify that the underlying `libp2p` DHT `Provide` method is called with the correct CID.
    *   Test error handling if the DHT operation fails.
*   **Test Cases for `FindProviders(cid, count)`:**
    *   Mock underlying DHT to return a list of `peer.AddrInfo`. Verify `FindProviders` returns them correctly.
    *   Mock DHT to return no providers.
    *   Mock DHT to return fewer providers than `count`.
    *   Test error handling if the DHT query fails.
*   **Note:** True end-to-end DHT behavior is better suited for integration tests, but unit tests can verify the `discovery` package correctly uses the `libp2p` DHT client APIs.

### 1.4. `manifest` Package (if separate logic beyond struct definition)

*   **Objective:** Verify any specific logic related to `ContentManifest` parsing, validation, or manipulation if such helpers exist outside the `chunker`.
*   **Test Cases:**
    *   Serialization and deserialization of `ContentManifest` (Protobuf).
    *   Validation of a parsed `ContentManifest` (e.g., ensure `chunk_cids` are valid CIDs, `original_content_sha256` is correct length).

### 1.5. `network` Package (`NetworkService` implementation)

*   **Objective:** Verify correct serialization/deserialization of DDS Protobuf messages and basic request/response handling logic (often requires mocking `libp2p` stream/host interactions).
*   **Test Cases for Senders (e.g., `SendRetrieveChunkRequest`):**
    *   Verify correct Protobuf message is constructed.
    *   Verify it's correctly written to a mocked `libp2p` stream.
    *   Verify response is correctly read and deserialized from the stream.
    *   Test timeout handling.
    *   Test error handling (e.g., peer disconnect, stream write error).
*   **Test Cases for Handlers (registered via `SetStreamHandler`):**
    *   Simulate incoming request on a mocked stream.
    *   Verify the handler correctly deserializes the request.
    *   Verify the handler calls the appropriate backend logic (e.g., `StorageProvider.Retrieve`).
    *   Verify the handler correctly serializes and sends the response.
*   **Protobuf Message Definitions (`protos/dds_messages.proto` generated code):**
    *   While the generated code itself isn't typically unit-tested, ensure that messages can be created, populated, serialized to bytes, and deserialized back into objects with data integrity. This is often implicitly covered by `network` package tests.

### 1.6. `replication` Package (`ReplicationManager` MVP)

*   **Objective:** For MVP, if `ReplicationManager` is very basic (e.g., only `InstructReplicate`), test that functionality.
*   **Test Cases for `InstructReplicate()`:**
    *   Verify it correctly calls `NetworkService.SendReplicationInstruction` with the right parameters.
    *   Test error handling if the network call fails.
*   **Note:** More complex replication logic testing will be part of post-MVP phases and will involve more integration-style tests.

This unit testing strategy aims to build confidence in each DDS component before they are integrated.
---

## 2. Integration Testing Strategy for DDS

Integration tests will verify that different DDS components work together correctly as a cohesive system. These tests will typically involve setting up a local `libp2p` test network with multiple peer nodes, each running parts or all of the DDS service.

### 2.1. Test Environment Setup

*   **Local `libp2p` Test Network:** Utilize `libp2p`'s testing utilities (e.g., `libp2p/go-libp2p-testing`) to create multiple in-process `libp2p` hosts that can communicate with each other.
*   **Mocking vs. Real Components:**
    *   For focused integration tests, some external systems (like a full DLI EchoNet for `manifest_cid` lookup) might be mocked initially.
    *   However, the core DDS components (`StorageProvider`, `Discovery` using a real local DHT, `NetworkService`) should be the actual implementations.
*   **Node Configuration:** Each test node will be configured with its DDS services (storage, discovery, network handlers).

### 2.2. Key Integration Test Scenarios (MVP)

1.  **Scenario: End-to-End Publish & Retrieve (Two Nodes)**
    *   **Objective:** Verify the complete lifecycle of publishing content by one node and retrieving it by another.
    *   **Setup:**
        *   Node A (Originator/Storage Node): Runs full DDS service.
        *   Node B (Retrieval Node): Runs DDS retrieval logic and DHT client.
        *   Nodes A and B are connected in a local `libp2p` test network and participate in the same DHT.
    *   **Steps:**
        a.  Node A: Calls `DDS.Publish()` with sample content.
        b.  Verify: Node A successfully chunks, creates manifest, stores locally, and provides all CIDs to the DHT.
        c.  Node B: Obtains the `manifest_cid` from Node A (out-of-band for this test, or Node B could discover it if Node A also "publishes" the manifest_cid to a known test list).
        d.  Node B: Calls `DDS.Retrieve()` with the `manifest_cid`.
        e.  Verify:
            *   Node B successfully uses DHT to find Node A as provider for manifest.
            *   Node B retrieves and verifies the manifest from Node A.
            *   Node B successfully uses DHT to find Node A as provider for each data chunk.
            *   Node B retrieves and verifies each data chunk from Node A.
            *   Node B successfully reassembles the content.
            *   Node B successfully performs final content verification against `original_content_sha256`.
            *   The retrieved content matches the original sample content.
    *   **Variations:** Test with different content sizes (single chunk, multiple chunks).

2.  **Scenario: Basic Replication & Retrieval from Replica (Three Nodes)**
    *   **Objective:** Verify that content seeded to a second storage node can be retrieved by a third node.
    *   **Setup:**
        *   Node A (Originator/Storage Node).
        *   Node C (Storage Node / Initial Seed Target).
        *   Node B (Retrieval Node).
        *   All nodes connected and in the same DHT.
    *   **Steps:**
        a.  Node A: Calls `DDS.Publish()` with sample content. In its simplified MVP seeding logic, it directly instructs/sends chunks to Node C.
        b.  Verify: Node A and Node C both store the content (manifest and chunks) and provide CIDs to DHT.
        c.  Node B: Obtains `manifest_cid`.
        d.  Node B: Calls `DDS.Retrieve()` with the `manifest_cid`.
        e.  Verify: Node B can discover *both* Node A and Node C as providers (or primarily Node C if Node A is temporarily taken offline for the test).
        f.  Verify: Node B can successfully retrieve and verify all content, potentially fetching some chunks from Node A and some from Node C, or all from Node C if Node A is made unavailable after providing.
    *   **Focus:** Tests the `StoreChunkRequest` pathway for seeding and subsequent retrieval from a replica.

3.  **Scenario: DHT Provider Record Propagation and Lookup**
    *   **Objective:** Specifically test the DHT `Provide` and `FindProviders` interactions.
    *   **Setup:** Multiple DDS nodes in a DHT.
    *   **Steps:**
        a.  Node A stores a new chunk and calls `Discovery.Provide(cid)`.
        b.  Wait briefly for DHT propagation.
        c.  Node B calls `Discovery.FindProviders(cid)`.
        d.  Verify: Node B receives Node A's `peer.AddrInfo` in the results.
        e.  Node A stops providing (e.g., simulates going offline or removing the provider record if `libp2p` supports it cleanly for tests).
        f.  Node B calls `Discovery.FindProviders(cid)` again after a timeout/expiry period.
        g.  Verify: Node A is no longer listed (or its record is considered stale, depending on DHT behavior).

### 2.3. Error Condition Testing

*   **Node Unavailability:** During retrieval, simulate a provider node going offline after providing some but not all chunks. Verify the retrieval logic can (if possible) switch to other providers or gracefully report failure.
*   **Data Corruption:**
    *   Test retrieval of a chunk where the data is intentionally corrupted (but CID remains the same on provider record). Verify the client detects CID mismatch and discards the chunk.
    *   Test retrieval where the reassembled content does not match `original_content_sha256` in the manifest.
*   **DHT Failures:** Simulate DHT query failures during `FindProviders`. Verify robust error handling.
*   **Network Issues:** Simulate network partitions or high latency between peers in the testnet, observe impact on DDS operations. (More advanced, may require specialized tooling).

Integration tests will be key to ensuring the DDS system functions correctly as a whole, beyond the correctness of its individual parts.
---

## 3. Performance and Scalability Considerations (High-Level Plan)

While comprehensive performance and scalability testing will be a focus in post-MVP phases, it's important to outline initial considerations and key metrics during the planning stage. The goal of the DDS is to handle a significant volume of user-generated content efficiently.

### 3.1. Key Performance Metrics to Track

The following metrics will be important indicators of DDS performance:

*   **Time to Store (TTS):**
    *   **Definition:** The latency from when an Originator Node initiates a `DDS.Publish()` call to when the manifest and all its constituent chunks are successfully stored on the Originator's local `StorageProvider` AND provider records for all CIDs are successfully put to the DHT.
    *   **Sub-metrics:** Time for chunking, time for local storage writes, time for DHT `Provide` operations.
*   **Time to Retrieve - First Chunk (TTR-FC):**
    *   **Definition:** The latency from when a Retrieval Node initiates a `DDS.Retrieve(manifest_cid)` call to when the first data chunk of the content is successfully retrieved and verified.
    *   **Sub-metrics:** `manifest_cid` discovery time (DHT `FindProviders`), manifest retrieval time, first `chunk_cid` discovery time, first data chunk retrieval time.
*   **Time to Retrieve - Full Content (TTR-AC):**
    *   **Definition:** The latency from when a Retrieval Node initiates a `DDS.Retrieve(manifest_cid)` call to when all data chunks are retrieved, reassembled, and the full content is verified against `original_content_sha256`.
*   **DHT Query Latency:**
    *   **Definition:** Average time taken for `FindProviders` DHT lookups to resolve.
    *   **Factors:** DHT size, network conditions, efficiency of the DHT implementation.
*   **Replication Speed (Post-MVP):**
    *   **Definition:** Time taken for a chunk to reach its target replication factor (N) after initial seeding or after a repair is triggered.
*   **Throughput:**
    *   **Storage Throughput:** Rate at which new content chunks can be ingested and stored by the network (e.g., chunks/sec or MB/sec per node / across network).
    *   **Retrieval Throughput:** Rate at which content chunks can be served by Storage Nodes (e.g., chunks/sec or MB/sec per node / across network).

### 3.2. Initial Benchmarking Approach (Post-MVP Development)

While not part of the MVP *implementation testing*, a strategy for future benchmarking should be considered:

*   **Controlled Test Environment:** Set up a dedicated test network (e.g., using Kubernetes or Docker Swarm to deploy multiple DDS nodes in a simulated network environment with configurable parameters like latency, bandwidth, and churn).
*   **Workload Generation:** Develop tools to generate realistic workloads:
    *   Simulate many Originator Nodes publishing content of various sizes and types.
    *   Simulate many Retrieval Nodes requesting popular and less popular content.
*   **Monitoring & Data Collection:** Integrate metrics collection (e.g., using Prometheus and Grafana, or `libp2p`'s built-in metrics capabilities if sufficient) to track the key performance metrics defined above.
*   **Scenario-Based Testing:**
    *   **Scale Testing:** Gradually increase the number of nodes, the amount of stored data, and the rate of requests to identify performance bottlenecks and limits.
    *   **Stress Testing:** Subject the system to peak loads or adverse conditions (e.g., high node churn) to assess its stability and recovery capabilities.
*   **Profiling:** Use Go's built-in profiling tools (pprof) to identify CPU and memory hotspots within DDS node implementations during benchmark runs.

### 3.3. Scalability Considerations in Design

Even during MVP, design choices should keep future scalability in mind:

*   **Efficient DHT Usage:** Minimize unnecessary DHT traffic. For example, batching `Provide` records where possible (if supported by `libp2p` or through an intermediary service).
*   **Connection Management:** Efficiently manage `libp2p` connections to avoid overwhelming nodes.
*   **Asynchronous Operations:** Utilize Go concurrency patterns (goroutines, channels) to handle multiple requests and P2P operations concurrently without blocking.
*   **Streamlined Data Paths:** Optimize the flow of data for storage and retrieval to minimize copying and processing overhead.

By defining these metrics and outlining a future benchmarking approach, the DDS protocol can be iteratively improved for performance and scalability as DigiSocialBlock evolves.
```
