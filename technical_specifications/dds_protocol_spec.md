# DigiSocialBlock - Distributed Data Stores (DDS) Protocol Specification

**Version:** 0.1.0
**Date:** 2025-04-12
**Status:** Draft

## Introduction

This document specifies the Distributed Data Stores (DDS) protocol for DigiSocialBlock's EchoNet. DDS is responsible for the decentralized storage, replication, discovery, and retrieval of content chunks. It aims to provide content resilience, censorship resistance, and scalability, forming a critical layer for the Proof-of-Witness (PoW) mechanism and overall platform functionality.

---

## 1. Data Chunking & Content Addressing

At the core of DDS is the principle of breaking down content into manageable, verifiable pieces (chunks), each uniquely identified by a Content ID (CID). This approach ensures data integrity and facilitates efficient distributed storage and retrieval.

### 1.1. Content Chunking

*   **Fixed Chunk Size:** All content stored within the DDS will be divided into fixed-size chunks.
    *   **Specification:** Each chunk will have a maximum size of **256 KiB (256 * 1024 bytes)**.
    *   **Rationale:** A fixed size simplifies buffer management, storage allocation, and network transfer calculations. 256 KiB offers a balance between minimizing overhead for small files and not being excessively large for efficient P2P transfer and DHT advertisements.
*   **Handling Content Smaller than Chunk Size:** If a piece of content (or the final segment of a larger piece of content) is smaller than 256 KiB, it will form a chunk of its actual size. There is no padding to reach 256 KiB.
*   **Handling Empty Content:**
    *   Empty content (e.g., an empty text file) will result in a single chunk of 0 bytes. This chunk will still have a unique CID based on the hash of 0 bytes of data.
    *   A `ContentManifest` (see Section 1.3) for empty content would list the CID of this single 0-byte chunk and have an `original_content_sha256` corresponding to the hash of an empty input.

### 1.2. Content Identifiers (CIDs)

Each chunk of data is uniquely identified by a Content ID (CID). CIDs are self-describing identifiers derived from the content itself, ensuring immutability and verifiability.

*   **CID Generation Process:**
    1.  **Hashing:** The binary data of a chunk is hashed using the **SHA-256** algorithm. This produces a 32-byte hash.
    2.  **Encoding:** The resulting 32-byte SHA-256 hash is then encoded using **Base58BTC**. This encoding is commonly used in Bitcoin and IPFS for compact, human-readable (or at least, more easily transmissible) representation of binary data.
*   **CID Uniqueness & Integrity:** Since CIDs are derived from the content's hash, any modification to the chunk's data will result in a different CID. This makes content addressing inherently secure and allows for easy verification of data integrity upon retrieval.
*   **Example CID Generation (Conceptual):**
    *   Let `chunk_data` be the byte array of a 256 KiB content chunk.
    *   `sha256_hash = SHA256(chunk_data)` (produces 32 bytes)
    *   `cid_string = Base58BTCEncode(sha256_hash)`
    *   This `cid_string` is the identifier used for this specific chunk.

### 1.3. Content Manifest (`ContentManifest`)

For content that spans multiple chunks, or to provide metadata about single-chunk content (like its original full hash), a `ContentManifest` is used. The manifest itself is a piece of data and is also addressable by its own CID.

*   **Purpose:**
    *   Lists the CIDs of all constituent data chunks in their correct order.
    *   Contains the SHA-256 hash of the *original, complete content* before it was chunked. This allows for end-to-end verification after all chunks are retrieved and reassembled.
    *   May contain other metadata in future versions (e.g., original filename, MIME type), but MVP focuses on chunk list and original hash.
*   **Structure (Conceptual - Protobuf Definition in `protos/dds_messages.proto`):**
    ```protobuf
    // Conceptual structure for ContentManifest
    message ContentManifest {
      // List of CIDs for the data chunks, in order.
      repeated string chunk_cids = 1; // Base58BTC encoded SHA-256 hashes of individual chunks

      // SHA-256 hash of the original, complete content before chunking.
      // Represented as a hex-encoded string or raw bytes. For spec, let's assume raw bytes.
      bytes original_content_sha256 = 2; // 32 bytes

      // Optional: Original content size in bytes
      uint64 original_content_size_bytes = 3;
    }
    ```
*   **`manifest_cid` Generation:**
    1.  The `ContentManifest` message is first serialized into its canonical binary representation using Protobuf.
    2.  This binary representation of the manifest is then treated like any other piece of content: it's hashed using SHA-256, and the hash is Base58BTC encoded to produce the `manifest_cid`.
    *   `serialized_manifest = ProtobufSerialize(my_content_manifest)`
    *   `manifest_sha256_hash = SHA256(serialized_manifest)`
    *   `manifest_cid_string = Base58BTCEncode(manifest_sha256_hash)`
*   **Referencing Content:** In DigiSocialBlock's EchoNet (DLI records) or other services, content will typically be referenced by its `manifest_cid`. Retrieval clients will first fetch the manifest using this `manifest_cid`, then use the information within the manifest to fetch all the actual data chunks.

### 1.4. Overall Workflow Example: Publishing Content

1.  **User Action:** A user creates a piece of content (e.g., a text post, an image).
2.  **Client-Side Preparation (Originator Node):**
    a.  **Calculate Original Hash:** The client application calculates the SHA-256 hash of the complete, original content (`original_content_sha256`). Let's say the content is "Hello World". `original_content_sha256 = SHA256("Hello World")`.
    b.  **Chunk Content:** The content ("Hello World") is smaller than 256 KiB, so it forms a single chunk.
        *   `chunk1_data = "Hello World"`
    c.  **Generate Chunk CID:** The client calculates the CID for this chunk.
        *   `chunk1_cid = Base58BTCEncode(SHA256(chunk1_data))`
    d.  **Create `ContentManifest`:**
        *   `manifest.chunk_cids = [chunk1_cid]`
        *   `manifest.original_content_sha256 = original_content_sha256` (from step 2a)
        *   `manifest.original_content_size_bytes = length("Hello World")`
    e.  **Generate `manifest_cid`:** The client serializes the `ContentManifest` and calculates its CID.
        *   `manifest_cid = Base58BTCEncode(SHA256(ProtobufSerialize(manifest)))`
3.  **Storage & Advertising (Originator Node & DDS Network):**
    a.  The Originator Node stores `chunk1_data` locally, addressable by `chunk1_cid`.
    b.  The Originator Node stores the serialized `ContentManifest` data locally, addressable by `manifest_cid`.
    c.  The Originator Node advertises to the DDS network (e.g., puts provider records into the DHT) that it holds `chunk1_cid` and `manifest_cid`.
    d.  The Originator Node initiates seeding of `chunk1_cid` and `manifest_cid` to other Storage Nodes.
4.  **Reference in EchoNet:** The `manifest_cid` is then used in a `NexusContentObjectV1` record on the DLI `EchoNet` to refer to this piece of content.

This detailed specification for data chunking and content addressing ensures that content is uniquely identifiable, verifiable, and ready for decentralized storage and retrieval.
---

## 2. Node Roles & Responsibilities in DDS

The Distributed Data Stores (DDS) protocol relies on various types of nodes, each with distinct roles and responsibilities, to ensure the robust and efficient functioning of the decentralized storage network. While a single physical node may embody multiple roles, these conceptual distinctions are important for understanding the protocol.

### 2.1. Storage Nodes

*   **Primary Responsibility:** To reliably store content chunks (identified by CIDs) and make them available for retrieval by other nodes in the network.
*   **Key Functions:**
    *   **Local Storage:** Maintain a persistent local store for chunk data. This could be a dedicated key-value database (mapping `cid_string` to `chunk_data`) or a structured layout on a filesystem. The specific implementation is up to the node operator but must allow efficient CID-based lookup.
    *   **Respond to Retrieval Requests:** Listen for and respond to `RetrieveChunkRequest` messages (see Section 6) from other nodes, serving the requested chunk data if available locally.
    *   **Advertise Stored CIDs:** Periodically or upon storing new chunks, advertise their availability to Discovery Nodes by putting provider records (mapping `chunk_cid` or `manifest_cid` to their own PeerID) into the Distributed Hash Table (DHT).
    *   **Participate in Replication:** Cooperate with replication requests and instructions from other nodes or from automated replication management logic (see Section 3) to store copies of chunks for redundancy.
    *   **Data Integrity Checks (Optional but Recommended):** Periodically verify the integrity of stored chunks against their CIDs to detect local data corruption.
*   **Conceptual Requirements:**
    *   Sufficient and reliable storage capacity.
    *   Stable and adequate network bandwidth for serving chunks.
    *   High uptime to ensure data availability.
*   **Incentives (Conceptual - Linking to broader DigiSocialBlock Economics):**
    *   Storage Nodes are envisioned to be incentivized for their contributions (storing data, serving data, participating in replication) through mechanisms tied to Proof-of-Pristine (PoP) and the `pallet-rewards` system, ensuring economic viability for node operators. Details of these incentives are outside the scope of this DDS specification but are a critical dependency.

### 2.2. Originator Nodes

*   **Primary Responsibility:** The node (typically a client application acting on behalf of a user) that initially introduces new content into the DigiSocialBlock ecosystem.
*   **Key Functions:**
    *   **Content Processing:**
        *   Calculate the `original_content_sha256` of the complete content.
        *   Chunk the content according to Section 1.1.
        *   Generate `chunk_cids` for each chunk according to Section 1.2.
        *   Create and serialize the `ContentManifest` according to Section 1.3.
        *   Generate the `manifest_cid` for the `ContentManifest`.
    *   **Initial Local Storage:** Store all generated chunks and the serialized manifest locally, at least temporarily, to facilitate initial seeding.
    *   **Initial Seeding & Advertising:**
        *   Advertise the `manifest_cid` and all `chunk_cids` to the DDS network by putting provider records into the DHT, associating these CIDs with its own PeerID.
        *   Actively push (seed) the manifest and all constituent chunks to a configured number of initial Storage Nodes or Super-Hosts to ensure immediate availability and kickstart the replication process. The exact mechanism for selecting these initial seed nodes (e.g., well-known bootstrap nodes, nodes selected from a local peer list) will be defined in the implementation plan.
*   **Note:** An Originator Node also acts as a Storage Node for the content it originates, at least initially.

### 2.3. Retrieval Nodes

*   **Primary Responsibility:** Any node (e.g., a user's client application, another service within DigiSocialBlock) that needs to access and consume content stored in the DDS.
*   **Key Functions:**
    *   **Manifest Discovery:** Obtain the `manifest_cid` of the desired content. This `manifest_cid` is typically acquired from an external source, such as a `NexusContentObjectV1` record on the DLI `EchoNet` or a link shared through other means.
    *   **Manifest Retrieval:**
        *   Query Discovery Nodes (DHT) using `FindProvidersRequest` for the `manifest_cid` to get a list of PeerIDs of nodes storing the manifest.
        *   Send a `RetrieveChunkRequest` for the `manifest_cid` to one or more of these providers.
        *   Verify the integrity of the retrieved manifest data by hashing it and comparing it to the `manifest_cid`.
        *   Parse the verified `ContentManifest`.
    *   **Chunk Discovery & Retrieval:**
        *   For each `chunk_cid` listed in the parsed `ContentManifest`:
            *   Query Discovery Nodes (DHT) using `FindProvidersRequest` for the `chunk_cid`.
            *   Select suitable providers from the returned list (e.g., based on latency, reliability metrics if available).
            *   Send `RetrieveChunkRequest` for the `chunk_cid` to selected provider(s).
            *   Upon receiving `chunk_data`, immediately verify its integrity by hashing it and comparing the result to the `chunk_cid`. Discard and retry from another provider if verification fails.
    *   **Content Reassembly:** Once all verified chunks are retrieved, reassemble them in the correct order as specified in the `ContentManifest`.
    *   **Final Content Verification:** Calculate the SHA-256 hash of the fully reassembled content and compare it against the `original_content_sha256` stored in the `ContentManifest`. This provides end-to-end assurance that the retrieved content is identical to the original. If this verification fails, the content is considered corrupt or incomplete.
    *   **Caching (Optional):** May choose to cache retrieved manifests and chunks locally to speed up future access and reduce network load, subject to local cache policies (e.g., size limits, LRU eviction). If caching, the Retrieval Node may also advertise these cached CIDs to the DHT, temporarily acting as a Storage Node for that content.

### 2.4. Discovery Nodes

*   **Primary Responsibility:** To facilitate the discovery of content within the DDS network by maintaining a distributed index (DHT) that maps CIDs to the PeerIDs of Storage Nodes that hold the corresponding content.
*   **Key Functions:**
    *   **Participate in DHT:** Act as participants in the chosen Kademlia-based Distributed Hash Table (DHT) implementation (e.g., `libp2p-kad-dht`). This involves maintaining a routing table, responding to DHT queries from other peers, and storing portions of the DHT's key-value space.
    *   **Store Provider Records:** Accept and store `provider records` advertised by Storage Nodes and Originator Nodes. A provider record is essentially a mapping: `CID -> PeerID`.
    *   **Respond to `FindProviders` Queries:** When receiving a `FindProvidersRequest` for a given CID, query its local DHT segment and route the query further if necessary to find and return a list of PeerIDs of nodes that have advertised they are storing that CID.
*   **Node Types as Discovery Nodes:** Any node in the DigiSocialBlock network can potentially participate as a Discovery Node if it runs the DHT protocol. However, Super-Hosts, due to their expected higher uptime and resource availability, are prime candidates to act as robust and reliable Discovery Nodes, forming a stable backbone for the DHT.
*   **Scalability & Resilience:** The distributed nature of the DHT ensures that discovery services are scalable and resilient to individual node failures.

By clearly defining these roles, the DDS protocol aims to create a synergistic ecosystem where different nodes collaborate to ensure content is stored reliably, replicated for availability, and discoverable efficiently across the decentralized network.
---

## 3. Data Replication Strategy

Data replication is a cornerstone of the DDS protocol, designed to ensure high availability, fault tolerance, and censorship resistance for all content stored within the DigiSocialBlock network. The strategy aims for a balance between redundancy, storage efficiency, and network overhead.

### 3.1. Target Replication Factor (N)

*   **Definition:** The DDS network will aim to maintain a minimum number of distinct copies (replicas) for each unique content chunk (identified by its CID). This number is known as the Target Replication Factor (N).
*   **MVP Specification:** For the initial Minimum Viable Product (MVP), N is set to **7 (N=7)**.
    *   **Rationale:** A factor of 7 provides a reasonable degree of redundancy against simultaneous node failures or departures from the network, without imposing an excessive storage burden during early stages. This value is a network parameter and can be adjusted via governance in future iterations.
*   **Scope:** This applies to both individual data chunks and `ContentManifest` chunks.

### 3.2. Initial Seeding by Originator Nodes

*   **Responsibility:** The Originator Node (the client application that first introduces the content) is responsible for ensuring the initial N replicas of the content (both manifest and all its data chunks) are created and distributed.
*   **Process:**
    1.  After chunking content and creating the `ContentManifest` (as per Section 1), the Originator Node stores a local copy.
    2.  The Originator Node then attempts to directly transfer (push) each chunk (manifest and data chunks) to at least **N-1 other distinct Storage Nodes**.
    3.  **Seed Node Selection (Conceptual for MVP):**
        *   For MVP, Originator Nodes might select these initial N-1 seed targets from a list of known bootstrap/Super-Host nodes, or from a set of Storage Nodes discovered through the DHT that advertise sufficient capacity and reliability (if such metrics are available).
        *   Future enhancements could involve more sophisticated selection algorithms based on network topology, latency, node reputation, or load balancing.
    4.  The Originator Node should await confirmation of successful storage from these N-1 peers before considering the initial seeding complete for a given chunk. If a peer fails to store, another should be chosen.
    5.  Concurrently, the Originator Node (and the N-1 initial Storage Nodes upon successful storage) must advertise themselves as providers for these CIDs in the DHT (as per Section 2.1 and 2.2).

### 3.3. Proactive Replication by Storage Nodes

*   **Responsibility:** Storage Nodes that hold a copy of a chunk are collectively responsible for monitoring its replication level and initiating further replication if it falls below N.
*   **Mechanism (Conceptual):**
    1.  **Replication Level Check:** Periodically (e.g., every few hours, configurable), a Storage Node holding a chunk `C` will query the DHT (`FindProviders` for `C`) to determine the current number of known providers for that chunk.
    2.  **Identify Under-Replication:** If the number of unique providers is less than N, the chunk is considered under-replicated.
    3.  **Initiate Replication:**
        *   The Storage Node itself can decide to initiate replication for chunks it holds that are under-replicated.
        *   It needs to select one or more *new* target Storage Nodes (that do not already hold the chunk) to store a copy. Selection criteria could include:
            *   Network proximity (low latency).
            *   Advertised available storage capacity.
            *   Node reliability/uptime (if such metrics become available via PoP or other systems).
            *   Geographic diversity (to protect against localized outages).
            *   Anti-affinity (e.g., avoid selecting multiple nodes known to be in the same data center or under the same operator if possible).
        *   The initiating Storage Node sends a `ReplicationInstruction` message (see Section 6) to the chosen target Storage Node(s), which includes the `chunk_cid` to be replicated. The target node would then retrieve the chunk from the initiator (or any other known provider) and store it.
*   **Coordination:** To avoid redundant replication efforts (multiple nodes trying to replicate the same under-replicated chunk simultaneously), a "replication lease" or a short-term "intent-to-replicate" announcement in the DHT for a specific CID could be considered in more advanced versions. For MVP, a simpler probabilistic approach or reliance on slightly staggered check intervals might be sufficient, accepting minor potential over-replication.

### 3.4. Repair and Self-Healing

*   **Concept:** The DDS network must be able to automatically detect and repair content that becomes unavailable or whose replication factor drops significantly due to nodes going offline.
*   **Trigger for Repair:**
    *   **Retrieval Failure:** If a Retrieval Node fails to find N providers for a chunk or fails to retrieve a chunk after trying several known providers, it can signal a potential availability issue. (How it signals this – e.g., to a specific set of repair-initiating nodes or by attempting to trigger proactive replication – needs further definition, potentially post-MVP).
    *   **Proactive Replication as Repair:** The proactive replication mechanism described in Section 3.3 inherently acts as a self-healing process. As Storage Nodes detect under-replication (due to other nodes departing), they will attempt to create new replicas.
*   **Role of `ContentManifest`:** When a Retrieval Node successfully retrieves a `ContentManifest` but then struggles to retrieve one of its `chunk_cids`, this provides a clear signal that a specific, necessary part of the content is missing and needs repair.
*   **Garbage Collection (Future Consideration):** While not part of repair, related is the concept of garbage collecting truly orphaned or zero-provider chunks after an extended period, if deemed necessary by network policy. This is complex and out of MVP scope. For MVP, the focus is on maintaining N replicas of all *referenced* content (i.e., content whose manifest_cid is findable and whose chunk_cids are listed).

### 3.5. Considerations for Replication Strategy

*   **Network Overhead:** Replication traffic will consume bandwidth. The frequency of replication checks and the number of chunks replicated in each cycle must be balanced.
*   **Storage Costs:** Maintaining N replicas means total storage used is N times the unique data size. This is a direct trade-off for availability and resilience.
*   **Consistency Model:** DDS primarily offers eventual consistency for replication. There will be a delay between a node storing a chunk and that chunk being fully replicated to N peers and the DHT being updated globally. Retrieval mechanisms should be designed to handle this (e.g., by trying multiple providers).

This replication strategy, combining initial seeding with ongoing proactive replication and repair, aims to make the DDS a resilient and highly available storage layer for DigiSocialBlock content.
---

## 4. Data Discovery & Retrieval Protocols

Efficiently discovering and retrieving content chunks from a decentralized network of potentially thousands or millions of nodes is critical for the user experience and overall performance of DigiSocialBlock. This section details the protocols for these operations.

### 4.1. Discovery Mechanism: Kademlia DHT via `libp2p`

*   **Primary Mechanism:** The primary mechanism for discovering which nodes (Peers) store specific content chunks (identified by CIDs) will be a **Kademlia-based Distributed Hash Table (DHT)**, as implemented and provided by the `libp2p` networking stack.
*   **Provider Records:**
    *   Storage Nodes and Originator Nodes advertise the CIDs they hold by publishing "provider records" to the DHT.
    *   A provider record maps a `CID` to the `PeerID` of the node providing that CID.
    *   The `libp2p` DHT handles the distributed storage and lookup of these provider records.
*   **Operations:**
    *   **`Put Provider Record`:** When a node stores a new chunk (or manifest), it announces this to the DHT, effectively saying, "I, PeerID_X, am now providing CID_Y."
    *   **`Get Provider Record (FindProviders)`:** When a node needs to find a chunk, it queries the DHT for the given CID. The DHT, through its distributed lookup algorithm, returns a list of PeerIDs that have advertised they are providing that CID.

### 4.2. Content Retrieval Process

The following outlines the step-by-step process a Retrieval Node undertakes to fetch and verify content, starting from a known `manifest_cid`.

1.  **Obtain `manifest_cid`:**
    *   The Retrieval Node first needs the `manifest_cid` of the desired content. This is typically obtained from a higher-level application or service, such as:
        *   A `NexusContentObjectV1` record fetched from the DLI `EchoNet` (Module 1.3).
        *   A link or reference shared through a social feed, message, or external source.

2.  **Retrieve and Verify the `ContentManifest`:**
    a.  **Find Manifest Providers:** The Retrieval Node queries the `libp2p` DHT using a `FindProvidersRequest` (see Section 6) for the `manifest_cid`. The DHT will return a list of `PeerID`s of nodes that store the manifest chunk.
    b.  **Select Provider(s):** From the list of providers, the Retrieval Node selects one or more suitable peers. Selection criteria can include network proximity (latency), perceived reliability (if such metrics are available), or simply trying them in order.
    c.  **Request Manifest Chunk:** The Retrieval Node sends a `RetrieveChunkRequest` (see Section 6) for the `manifest_cid` to the selected provider(s).
    d.  **Verify Manifest Chunk Integrity:** Upon receiving the manifest chunk data:
        *   Calculate its SHA-256 hash.
        *   Encode this hash using Base58BTC.
        *   Compare the result with the requested `manifest_cid`.
        *   If they do not match, the manifest chunk is considered corrupt or incorrect. The Retrieval Node should discard it and may try fetching from another provider.
    e.  **Parse Manifest:** If the manifest chunk is verified, the Retrieval Node deserializes it from its Protobuf binary format into the `ContentManifest` structure (see Section 1.3).

3.  **Iterate and Retrieve Data Chunks:**
    a.  The Retrieval Node iterates through the list of `chunk_cids` contained within the verified and parsed `ContentManifest`.
    b.  For each `chunk_cid` in the manifest's `chunk_cids` list:
        i.  **Find Chunk Providers:** Perform a `FindProvidersRequest` to the DHT for the current `chunk_cid`.
        ii. **Select Provider(s):** Select suitable provider(s) from the returned list.
        iii. **Request Data Chunk:** Send a `RetrieveChunkRequest` for the `chunk_cid` to the selected provider(s).
        iv. **Verify Data Chunk Integrity:** Upon receiving the data chunk:
            *   Calculate its SHA-256 hash.
            *   Encode this hash using Base58BTC.
            *   Compare the result with the current `chunk_cid` from the manifest.
            *   If they do not match, the chunk is corrupt. Discard it and attempt to retrieve from an alternative provider for that `chunk_cid`. If all providers fail for a specific chunk, the overall content retrieval may fail or be incomplete.
        v.  Store the verified chunk temporarily in memory or on local disk, associated with its order from the manifest.

4.  **Reassemble Original Content:**
    *   Once all data chunks specified in the `ContentManifest` have been successfully retrieved and individually verified, the Retrieval Node reassembles them in the correct sequence.

5.  **Final Content Verification (End-to-End):**
    a.  Calculate the SHA-256 hash of the fully reassembled content.
    b.  Compare this hash with the `original_content_sha256` field stored within the `ContentManifest`.
    c.  **If the hashes match:** The content has been successfully and verifiably retrieved. It is identical to the content originally published.
    d.  **If the hashes do not match:** This indicates a critical failure. Either one or more chunks were corrupted in a way that passed individual CID checks but led to overall content mismatch (highly unlikely with strong hashes), chunks were missed, or the manifest itself was for a different version of the content than expected (though manifest CID check should prevent this). The content should be considered invalid/corrupt by the Retrieval Node.

### 4.3. Caching Strategy for Retrieval Nodes

To improve performance for frequently accessed content and reduce overall network load, Retrieval Nodes may implement an optional local caching mechanism.

*   **Cacheable Items:** Both `ContentManifest` chunks and individual data chunks can be cached.
*   **Cache Key:** The CID of the chunk serves as the cache key.
*   **Cache Storage:** Can be in-memory for very hot content or on-disk for larger/longer-term caching.
*   **Cache Policy (Example):**
    *   **Size Limit:** The cache will have a configurable maximum size (e.g., 1 GB).
    *   **Eviction Policy:** When the cache is full and a new item needs to be added, an eviction policy such as **Least Recently Used (LRU)** will be employed to remove older items. Other policies like LFU (Least Frequently Used) could also be considered.
*   **Advertising Cached Content (Optional):**
    *   A Retrieval Node that caches content may optionally choose to also advertise itself as a provider for those cached CIDs in the DHT. This effectively turns the Retrieval Node into a temporary Storage Node for that content, further decentralizing availability.
    *   This behavior should be configurable, as not all Retrieval Nodes (especially resource-constrained ones) may wish to take on this responsibility.
*   **Cache Invalidation:** Since content in DDS is content-addressed (immutable), cache invalidation is straightforward: if a CID is in the cache, the data is valid. There's no concept of "stale" data for a given CID.

These discovery and retrieval protocols, leveraging the robustness of `libp2p` DHT and the integrity guarantees of content addressing, form the backbone of how users and services access data within DigiSocialBlock's DDS.
---

## 5. DDS Layer Security & Access Control Approach

The security model of the Distributed Data Stores (DDS) layer is primarily focused on data integrity and availability, leveraging content addressing and decentralized replication. Higher-level concerns such as fine-grained access control, user privacy through encryption, and complex content moderation policies are generally handled by layers above DDS (e.g., the application layer, DLI EchoNet interactions, or specific User Identity & Privacy Modules).

### 5.1. Data Integrity

*   **Core Mechanism:** Data integrity within DDS is fundamentally guaranteed by the **Content ID (CID)** system (see Section 1.2).
    *   Each content chunk and manifest has a CID derived from its cryptographic hash (SHA-256).
    *   Any alteration, no matter how small, to the data of a chunk or manifest will result in a different CID.
*   **Verification:** Retrieval Nodes perform integrity checks at multiple stages:
    1.  When retrieving a chunk (data or manifest), its received data is hashed, and the resulting CID is compared against the requested CID. Mismatches indicate corruption or incorrect data, leading to the chunk being discarded.
    2.  After reassembling all data chunks for a piece of content, the hash of the reassembled content is compared against the `original_content_sha256` stored in the `ContentManifest`. This provides end-to-end integrity verification.
*   **Immutability:** By its nature, content addressed by a specific CID is immutable. A "modified" piece of content is, in fact, a new piece of content with a new CID.

### 5.2. Data Availability

*   **Core Mechanism:** Data availability is ensured through the **Data Replication Strategy** (see Section 3), which aims to maintain N distinct copies of each chunk across the network.
*   **Resilience:** The distributed nature of storage and the proactive replication/self-healing mechanisms are designed to make the system resilient to individual Storage Node failures or departures.
*   **Incentives (Conceptual):** Economic incentives for Storage Nodes (linked to PoP and rewards) are crucial for encouraging reliable storage and high uptime, thereby contributing to overall data availability.

### 5.3. Access Control & Privacy

*   **DDS Stores Opaque Blobs:** The DDS layer itself treats all content chunks as opaque binary blobs. It does not interpret the content of these chunks.
*   **Encryption at Application Layer:**
    *   If content requires privacy or restricted access (e.g., private user data, encrypted messages), the **encryption MUST occur at the application layer (client-side) *before* the content is passed to the DDS for chunking and storage.**
    *   The DDS will then store these encrypted chunks.
*   **Key Management:** The management of encryption keys, distribution of keys to authorized parties, and enforcement of decryption permissions are explicitly **out of scope for the DDS protocol itself.** These functions are the responsibility of higher-level modules within DigiSocialBlock, such as a dedicated User Identity & Privacy Module, or specific application logic.
*   **Example:** A user wishing to store a private document would first encrypt it using a chosen key, then submit the resulting ciphertext to the DDS. Only entities possessing the correct decryption key can make sense of the retrieved (encrypted) chunks.

### 5.4. Sybil Resistance

*   **DHT Vulnerability:** Like all DHT-based systems, the DDS discovery mechanism (Kademlia DHT) is potentially vulnerable to Sybil attacks, where an attacker creates a large number of fake node identities to disrupt routing, eclipse parts of the network, or poison provider records.
*   **Reliance on Nexus Protocol Layer:** The primary defense against Sybil attacks at the DDS/DHT interface is **not within the DDS protocol itself but relies on the overarching security mechanisms of the DigiSocialBlock Nexus Protocol.** This includes:
    *   Staking requirements for nodes to participate meaningfully (e.g., as Super-Hosts or recognized Storage Nodes).
    *   Reputation systems derived from Proof-of-Pristine (PoP) and other behavioral metrics.
    *   Making it economically infeasible or reputationally damaging for an entity to mount a large-scale Sybil attack against the DHT.
*   `libp2p` may offer some basic protections, but robust Sybil resistance is a cross-layer concern.

### 5.5. Denial-of-Service (DoS/DDoS) Mitigation

*   **`libp2p` Defenses:** The underlying `libp2p` stack provides several built-in mechanisms that help mitigate DoS/DDoS attacks, such as:
    *   Connection managers that limit the number of inbound/outbound connections.
    *   Stream multiplexing which can handle many logical streams over fewer connections.
    *   Resource management to prevent a single peer or a few peers from overwhelming a node.
    *   Potential for rate limiting at various levels.
*   **Distributed Nature:** The decentralized nature of DDS means an attack would need to target a significant number of nodes simultaneously to cause widespread disruption to data availability or discovery.
*   **Incentives:** Well-designed incentive mechanisms can discourage frivolous or malicious storage/retrieval requests by associating costs or reputational consequences with excessive or abusive network usage.

### 5.6. Content Moderation & Takedown

*   **Immutability vs. Moderation:** Content stored in DDS is immutable by CID. A specific chunk `X` with `CID_X` cannot be altered without changing its CID.
*   **Takedown is Not Deletion:** "Takedown" or "moderation" of content within the DDS context does **not** mean deleting the content from all Storage Nodes that might hold it (which is practically impossible in a truly decentralized system without universal backdoors).
*   **Preventing Discovery & Propagation:** Instead, content moderation is an **application-layer and DLI EchoNet-layer concern.** It involves mechanisms such as:
    *   Removing or flagging the `manifest_cid` from indexes, search results, or social feeds (e.g., in DLI records).
    *   Building reputation systems where content flagged as undesirable is deprioritized or hidden by client applications.
    *   Potentially, instructing Discovery Nodes (via a governed process) to stop resolving provider records for specific CIDs associated with illicit or harmful content, making it hard to find.
*   **Storage Node Autonomy:** Individual Storage Nodes may also have their own policies regarding the content they are willing to store or serve, but this is outside the core DDS protocol specification for content addressing and retrieval.

In summary, the DDS protocol provides strong guarantees for data integrity and aims for high availability through its design. However, it strategically defers complex access control, end-to-end encryption key management, and nuanced content moderation policies to higher layers of the DigiSocialBlock stack, allowing DDS to focus on its core competency: robust decentralized storage and retrieval of content-addressed data.
---

## 6. DDS Network Messages (Protobuf Definitions)

Communication between nodes participating in the DDS protocol is facilitated by a set of standardized messages. These messages are defined using Protocol Buffers (Protobuf) version 3 for efficiency, type safety, and cross-language compatibility. The formal definitions for these messages are located in `protos/dds_messages.proto`.

These messages are intended to be exchanged over `libp2p` network streams, utilizing appropriate `libp2p` protocols for request-response patterns or direct stream communication as determined during implementation.

### 6.1. Core Message Overview

The following provides a brief overview of the primary messages used in DDS operations:

*   **`StoreChunkRequest` / `StoreChunkResponse`:**
    *   **Purpose:** Used by a node (e.g., Originator, or a node performing replication) to request another node (Storage Node) to store a specific content chunk.
    *   `StoreChunkRequest` contains the `cid` and the `chunk_data`.
    *   `StoreChunkResponse` indicates `success` or failure, with optional `error_code` and `error_message`.

*   **`RetrieveChunkRequest` / `RetrieveChunkResponse`:**
    *   **Purpose:** Used by a node (e.g., Retrieval Node) to request a specific content chunk from another node (Storage Node) that is known to provide it.
    *   `RetrieveChunkRequest` contains the `cid` of the desired chunk.
    *   `RetrieveChunkResponse` contains the `chunk_data` if successful, or an error indication if the chunk is not found or another issue occurs.

*   **`FindProvidersRequest` / `FindProvidersResponse`:**
    *   **Purpose:** Used by any node to query the DHT (Discovery Nodes) for a list of peers that are advertising storage for a given CID (which can be a `manifest_cid` or a `chunk_cid`).
    *   `FindProvidersRequest` contains the `cid` to look up.
    *   `FindProvidersResponse` returns a list of `peer_ids` (libp2p PeerIDs as strings) of the providers, along with a success status for the query itself.

*   **`ReplicationInstruction` / `ReplicationResponse`:**
    *   **Purpose:** Used by a Storage Node (that detects under-replication for a chunk it holds) to instruct another suitable Storage Node to fetch and store a copy of that chunk.
    *   `ReplicationInstruction` contains the `cid` of the chunk to be replicated and may include a `source_peer_id_hint`.
    *   `ReplicationResponse` indicates whether the target node has accepted the instruction and will attempt replication. This is an acknowledgement of receipt and intent, not a guarantee of replication completion.

### 6.2. General Message Characteristics

*   **CIDs in Messages:** CIDs are represented as strings, assuming the Base58BTC encoding of the SHA-256 hash as specified in Section 1.2.
*   **PeerIDs in Messages:** PeerIDs are represented as strings, corresponding to the standard `libp2p` PeerID string representation (typically Base58 encoded).
*   **Error Handling:** Responses generally include a `success` boolean flag, and optionally an `error_code` (from a yet-to-be-defined enumeration of DDS-specific errors) and a human-readable `error_message` to aid in debugging and diagnostics.

The detailed field definitions, types, and any specific constraints for these messages are found in `protos/dds_messages.proto`. This specification ensures that all nodes in the DDS network can communicate effectively and unambiguously.
---
