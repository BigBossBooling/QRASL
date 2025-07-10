# QRASL: Initial Blockchain Pallet Implementation Planning

**Objective:** To formulate a detailed plan for the initial implementation of core Substrate pallets that underpin QRASL's unique functionalities, specifically focusing on the most foundational elements that enable on-chain governance and tokenomics.

## I. The "Why": Bringing the Blueprint to Life on Chain

This phase marks the transition from conceptual architecture to tangible on-chain implementation. Developing these foundational Substrate pallets is crucial for:

*   **Activating Core Functionality:** Implementing the essential smart contract logic that enables the $QRASL token's economic activities, basic governance structures through the Zoologist's Guild, and the initial framework for the On-Chain Reputation System and Verifiable Credentials.
*   **Validating Design Decisions:** This early implementation planning allows for practical testing of the feasibility, interoperability, and potential performance characteristics of our on-chain architectural choices as defined in the Master Blueprint.
*   **Establishing On-Chain Identity & Participation:** Linking user DIDs (from DigiSocialBlock) to on-chain accounts and providing the mechanisms for these accounts to stake, vote, earn reputation, and hold credentials.
*   **"Law of Constant Progression":** Taking a significant step forward from documentation and high-level design to the concrete planning required for actual code development, setting the stage for iterative building and testing.
*   **Strategic Alignment with Master Blueprint:** Ensuring that the on-chain logic developed adheres strictly to the principles, interdependencies, and overall vision articulated in the `QRASL_ARCHITECTURE_BLUEPRINT.md`. This plan serves as a direct bridge from that blueprint to initial development tasks.

## II. The "What": Key Pallet Implementation Areas

We will prioritize the foundational pallets that enable the core economic and governance loops, to be developed using the Substrate framework in Rust. Each pallet will be designed for modularity and clear responsibilities.

### II.1. $QRASL Token Pallet (Conceptual Configuration of `pallet-assets` or Custom)

*   **Purpose:** To manage the native $QRASL token, its issuance (at a high level, primary issuance is via L1 block rewards), transfers, and balances.
*   **Action - Configuration/Design:**
    *   **Underlying Pallet:** Evaluate using Substrate's standard `pallet-assets` if suitable for managing the primary native token, or define specifications for a lean custom `pallet-qrasl-token`.
    *   **Total Supply & Decimals:** Reference `docs/QRASL_TOKENOMICS.md` for the defined total supply (e.g., 10 Billion) and decimal places (e.g., 12 or 18).
    *   **Minting/Burning Capabilities:**
        *   Primary minting (inflation via block rewards) is managed by the EmPower1 Layer 1 consensus and staking mechanisms. This pallet would not typically handle this directly.
        *   However, controlled minting/burning functions might be required for specific economic adjustments approved by Zoologist's Guild governance (e.g., for specific reward pool top-ups if not directly from Treasury, or for explicit deflationary burns managed by the Guild). These functions would require strict, authorized origins (e.g., Guild governance pallet).
    *   **Basic Functions (Extrinsics/Dispatchables):**
        *   `transfer(destination: AccountId, amount: Balance)`: Standard token transfer.
        *   `force_transfer(source: AccountId, destination: AccountId, amount: Balance)`: Admin/governance controlled transfer (e.g., for dispute resolutions).
    *   **Queries:**
        *   `balance_of(who: AccountId) -> Balance`
        *   `total_supply() -> Balance`
*   **Synergies:**
    *   Directly implements core aspects of `docs/QRASL_TOKENOMICS.md`.
    *   Interacts with all other pallets that require $QRASL balance checks, transfers, or staking.

### II.2. Reputation System Pallet (`pallet-reputation`)

*   **Purpose:** To implement the core on-chain logic for the On-Chain Reputation System, managing reputation scores for user accounts (DIDs).
*   **Storage:**
    *   `ReputationScores: map AccountId => ScoreType` (e.g., `ScoreType` could be `i64` to allow for a wide range and negative values).
    *   `MaxReputation: Option<ScoreType>` (Configurable maximum cap, if any).
    *   `MinReputation: Option<ScoreType>` (Configurable minimum cap, if any).
    *   `DefaultReputation: ScoreType` (Starting score for new accounts, e.g., 0 or 100).
*   **Extrinsics (Dispatchable Functions):**
    *   `update_reputation(origin, target: T::AccountId, score_change: ScoreType, reason_code: ReasonCode)`:
        *   `origin`: Must be an authorized extrinsic origin, such as a call from another pallet (e.g., Critter Tactics pallet reporting a win, Moderation Council pallet applying a penalty) or a governance-approved extrinsic.
        *   `target`: The account whose reputation is being updated.
        *   `score_change`: A positive or negative value to be added to the current score.
        *   `reason_code`: An enum or u16 representing the reason for the change (e.g., `CritterTacticsWin`, `CaregiverContractSuccess`, `ModerationPenaltyMinor`, `MarketDisputeFraud`). This is crucial for auditability and transparency.
    *   `set_initial_reputation(origin, target: T::AccountId, score: ScoreType)`:
        *   `origin`: Likely a root or governance origin for bootstrapping new accounts or specific administrative adjustments.
        *   Used to set the reputation for an account, perhaps upon completing an initial onboarding process.
*   **Public Functions (Callable by other pallets without needing an extrinsic):**
    *   `get_reputation(who: &T::AccountId) -> ScoreType`: Returns the current reputation score for an account.
    *   `increment_reputation(who: &T::AccountId, amount: ScoreType, reason: ReasonCode) -> DispatchResult`: Internal function for other pallets to easily increment reputation.
    *   `decrement_reputation(who: &T::AccountId, amount: ScoreType, reason: ReasonCode) -> DispatchResult`: Internal function for other pallets to easily decrement reputation.
    *   `can_submit_proposal(who: &T::AccountId) -> bool`: Example helper that might check if reputation meets a certain threshold defined in Guild config.
*   **Events:**
    *   `ReputationUpdated { who: AccountId, new_score: ScoreType, change: ScoreType, reason: ReasonCode }`: Emitted whenever a reputation score changes.
    *   `ReputationInitialized { who: AccountId, score: ScoreType }`: Emitted when an account's reputation is first set.
*   **Types:**
    *   `ScoreType`: e.g., `i64`.
    *   `ReasonCode`: An enum defining all possible reasons for reputation changes, for clear on-chain records.
*   **Synergies:**
    *   Consumes inputs (triggers for `update_reputation`) from: Critter Tactics pallet, Caregiver Contracts system (via an oracle or app backend attestations), Moderation Council pallet, Marketplace Dispute Resolution pallet, and the VC Registry pallet.
    *   Its scores are read by: Zoologist's Guild pallet (for voting power calculation, proposal eligibility), Council formation logic (for eligibility).
    *   Detailed in `docs/REPUTATION_SYSTEM.md`.

### II.3. Basic Staking Pallet (`pallet-qrasl-staking`)

*   **Purpose:** To enable users to stake $QRASL for specific purposes like governance participation (Zoologist's Guild voting) and eligibility for Council roles (Moderation, Dispute Resolution). This is distinct from the Layer 1 validator staking which is part of EmPower1's core PoS.
*   **Storage:**
    *   `StakedBalances: map (StakePurpose, AccountId) => Balance`: Tracks the amount staked by an account for a specific purpose. `StakePurpose` is an enum (e.g., `GovernanceVote`, `ModerationCouncilDuty`, `DisputeCouncilDuty`).
    *   `PurposeMinStake: map StakePurpose => Balance`: Defines the minimum stake required for each purpose. Configurable by Guild governance.
    *   `PurposeUnbondingPeriod: map StakePurpose => BlockNumber`: Defines the duration of the unbonding period for each staking purpose. Configurable by Guild governance.
    *   `UnbondingEntries: map (StakePurpose, AccountId) => Vec<UnbondingInfo { amount: Balance, unlock_at_block: BlockNumber }>`: Tracks tokens currently in the unbonding process for an account and purpose.
*   **Extrinsics (Dispatchable Functions):**
    *   `stake(origin, purpose: StakePurpose, amount: Balance)`:
        *   The `origin` is the staker's account.
        *   Locks the specified `amount` of $QRASL from the user's free balance into the staked balance for the given `purpose`.
        *   Checks if `amount` meets `PurposeMinStake[purpose]`.
        *   Updates `StakedBalances`.
    *   `unstake_request(origin, purpose: StakePurpose, amount: Balance)`:
        *   Initiates the unbonding process for a portion or all of the staked amount.
        *   Moves tokens from `StakedBalances` to a new entry in `UnbondingEntries` with `unlock_at_block` calculated based on `PurposeUnbondingPeriod[purpose]`.
    *   `withdraw_unbonded(origin, purpose: StakePurpose)`:
        *   Allows the user to reclaim tokens from `UnbondingEntries` for the specified `purpose` once `current_block_number >= unlock_at_block`.
        *   Removes the entry from `UnbondingEntries` and adds tokens back to free balance.
    *   `slash_stake(origin, purpose: StakePurpose, target: T::AccountId, slash_amount: Balance, destination: SlashDestination)`:
        *   `origin`: Must be an authorized extrinsic origin (e.g., Moderation Council pallet, Dispute Resolution pallet, or Guild governance pallet).
        *   `target`: The account whose stake is being slashed.
        *   `slash_amount`: The amount of staked $QRASL to be slashed.
        *   `destination`: An enum indicating where slashed tokens go (e.g., `ToTreasury`, `Burn`).
        *   Reduces the `StakedBalances[purpose][target]` by `slash_amount`. The actual token transfer/burn handled by this pallet or by notifying the token/treasury pallet.
*   **Events:**
    *   `Staked { who: AccountId, purpose: StakePurpose, amount: Balance }`
    *   `UnstakeRequested { who: AccountId, purpose: StakePurpose, amount: Balance, unlock_at_block: BlockNumber }`
    *   `Unstaked { who: AccountId, purpose: StakePurpose, amount: Balance }`
    *   `StakeSlashed { who: AccountId, purpose: StakePurpose, amount_slashed: Balance, destination: SlashDestination }`
*   **Types:**
    *   `StakePurpose`: Enum { `GovernanceVote`, `ModerationCouncilDuty`, `DisputeCouncilDuty`, `ProposalBond`, `DisputeBond` /* etc. */ }.
    *   `SlashDestination`: Enum { `ToTreasury`, `Burn` }.
*   **Synergies:**
    *   Directly implements Staking Mechanisms from `docs/QRASL_TOKENOMICS.md`.
    *   Interacts with the $QRASL Token Pallet for balance locking/unlocking.
    *   Zoologist's Guild pallet reads `StakedBalances[GovernanceVote]` for calculating voting power.
    *   Moderation Council and Dispute Resolution System pallets query `StakedBalances` and `PurposeMinStake` for eligibility checks and can trigger `slash_stake`.

### II.4. Verifiable Credentials (VC) Registry Pallet (`pallet-vc-registry`)

*   **Purpose:** To record the issuance and status (e.g., active, revoked) of Verifiable Credentials, primarily those related to Ethics Training completion, linking them to user DIDs (AccountIds).
*   **Storage:**
    *   `IssuedVCs: double_map (T::AccountId, VCType) => Option<VCData>`: Stores data for each VC issued to an account for a specific type. `VCType` is an enum (e.g., `ModerationEthicsV1_0`, `MarketplaceEthicsV1_0`).
    *   `VCData`: Struct containing `vc_id: Hash` (unique hash of the VC content), `issuer_did_hash: Hash` (hash of issuer's DID for brevity), `issuance_block: BlockNumber`, `module_version: VersionString`, `is_revoked: bool`, `expiry_block: Option<BlockNumber>`.
*   **Extrinsics (Dispatchable Functions):**
    *   `issue_vc(origin, holder_account: T::AccountId, vc_type: VCType, vc_id: Hash, issuer_did_hash: Hash, module_version: VersionString, expiry_block: Option<BlockNumber>)`:
        *   `origin`: Must be an authorized origin, likely a specific account controlled by the Zoologist's Guild or an oracle system attesting to off-chain training completion and assessment success.
        *   Creates a new entry in `IssuedVCs`.
        *   Triggers a call to `pallet-reputation` to update the holder's reputation.
    *   `revoke_vc(origin, holder_account: T::AccountId, vc_type: VCType, reason_code: RevocationReasonCode)`:
        *   `origin`: Authorized origin (e.g., Guild governance, potentially a council for VCs related to their roles if a member is removed for cause).
        *   Sets `is_revoked = true` for the specified VC.
        *   May trigger a call to `pallet-reputation` to apply a negative adjustment if revocation is due to misconduct.
*   **Public Functions (Callable by other pallets):**
    *   `has_valid_vc(who: &T::AccountId, vc_type: &VCType) -> bool`: Checks if the account holds a specific `vc_type` that is not revoked and not expired (if `expiry_block` is set and passed).
*   **Events:**
    *   `VCIssued { holder: AccountId, vc_type: VCType, vc_id: Hash, issuer: Hash, expiry: Option<BlockNumber> }`
    *   `VCRevoked { holder: AccountId, vc_type: VCType, vc_id: Hash, reason: RevocationReasonCode }`
*   **Types:**
    *   `VCType`: Enum for different credential types and versions (e.g., `ModerationEthicsV1_0`, `MarketplaceEthicsV1_0`, `ModerationEthicsV1_1`).
    *   `VersionString`: e.g., `Vec<u8>` for "1.0".
    *   `RevocationReasonCode`: Enum for why a VC was revoked.
*   **Synergies:**
    *   Directly implements the VC issuance and on-chain recording aspects of the `docs/ETHICS_TRAINING_MODULES.md`.
    *   Links to DIDs (represented as `AccountId` on-chain) from DigiSocialBlock.
    *   Queried by Moderation Council and Dispute Resolution System pallets for checking member eligibility.
    *   Interacts with `pallet-reputation` to grant reputation boosts upon successful VC issuance.

## III. High-Level Implementation Strategy

The development of these foundational pallets will follow a structured approach emphasizing modularity, security, and testability.

*   **Framework:** All pallets will be developed using the **Substrate** blockchain framework, written in **Rust**, to leverage its inherent modularity, forkless upgrade capabilities, and performance characteristics suitable for the QRASL/EmPower1 Layer 1.
*   **Pallet Development (Modularity):**
    *   Each functional area ($QRASL Token, Reputation, Staking, VC Registry) will be implemented as a distinct Substrate pallet.
    *   This promotes separation of concerns, making each pallet easier to develop, test, audit, and upgrade independently.
    *   Pallets will expose well-defined public functions and interfaces for interaction with other pallets (e.g., VC Registry pallet calling the Reputation pallet).
*   **Test-Driven Development (TDD):**
    *   A rigorous TDD approach will be applied. Unit tests will be written for all significant functions and logic paths within each pallet *before* or *concurrently with* their implementation.
    *   Integration tests will be developed to verify the correct interaction between these foundational pallets (e.g., ensuring VC issuance correctly triggers a reputation update and that staking status is correctly read for governance).
*   **Benchmarking (Initial Performance Checks):**
    *   Once initial versions of the pallets are functional, benchmarking will be conducted for all key extrinsics (dispatchable functions).
    *   This involves measuring their execution weight (a Substrate concept representing computational cost) to ensure they perform efficiently and do not consume excessive block resources. This data is crucial for setting appropriate transaction fees and ensuring network stability.
*   **Documentation:**
    *   **In-Code Comments:** Comprehensive Rust documentation comments (`///`) will be maintained for all public functions, types, storage items, and significant logic blocks within each pallet, enabling auto-generated developer documentation.
    *   **Master Blueprint Updates:** Relevant sections of the `QRASL_ARCHITECTURE_BLUEPRINT.md` (particularly Section III: Key Interdependencies & Data Flows) will be reviewed and updated to reflect any refinements or specific implementation details emerging from this pallet planning phase.
    *   **Pallet-Specific READMEs:** Each pallet directory in the codebase should include a `README.md` briefly explaining its purpose, key features, and how to interact with it.
*   **Version Control:** All code and documentation will be managed under Git, with feature branches for individual pallet development and pull requests for review before merging into a main development branch.
*   **Iterative Development:** While this plan outlines the initial design, development will be iterative. Early versions will focus on core functionality, with more advanced features or optimizations potentially added in subsequent iterations based on testing, feedback, and evolving ecosystem needs.

## IV. Synergies (Pallet Implementation Focus)

The development of these initial foundational pallets is not done in isolation but is crucial for enabling and interacting with the broader QRASL and DashAI-Go ecosystems as defined in previous design documents.

*   **EmPower1 Blockchain (QRASL Layer 1):**
    *   These pallets are the building blocks of the QRASL runtime logic that will execute on the EmPower1 Layer 1. They rely on EmPower1 for consensus, security, and finality.
    *   The $QRASL Token Pallet definition is fundamentally tied to the native token of EmPower1.
*   **DigiSocialBlock (Nexus Protocol & DIDs):**
    *   The `AccountId` used throughout these pallets (Reputation, Staking, VC Registry) will directly correspond to or be derived from the Decentralized Identities (DIDs) provided by DigiSocialBlock. This ensures that on-chain actions, reputation, stakes, and credentials are all linked to a consistent, user-controlled digital identity.
*   **$QRASL Tokenomics (`docs/QRASL_TOKENOMICS.md`):**
    *   The `$QRASL Token Pallet` directly implements the basic functionalities of the native token.
    *   The `pallet-qrasl-staking` implements the core staking mechanisms (for governance and council roles) outlined in the tokenomics design.
    *   Fee structures, reward distributions, and treasury interactions, while managed by their own pallets or L1 mechanisms, will all involve the $QRASL token.
*   **Zoologist's Guild & Councils (`docs/ZOOLOGISTS_GUILD.md`, `docs/MODERATION_COUNCIL.md`, `docs/MARKETPLACE_DISPUTE_RESOLUTION.md`):**
    *   The `pallet-qrasl-staking` provides the mechanism for users to stake $QRASL to gain voting power in the Guild and to meet eligibility requirements for council roles.
    *   The `pallet-reputation` provides the reputation scores that are also critical for Guild participation and council eligibility. Council verdicts will, in turn, call `pallet-reputation` to update scores.
    *   The `pallet-vc-registry` provides the Verifiable Credentials that gatekeep eligibility for council roles.
    *   The operational logic of the Guild and Councils (e.g., proposal submission, voting, dispute resolution, moderation actions) will be implemented in their own future pallets, which will heavily interact with these foundational ones.
*   **CritterCraftUniverse, Critter Tactics & Caregiver Contracts (`docs/CRITTER_TACTICS_GAMEPLAY.md`, `docs/CAREGIVER_CONTRACTS.md`):**
    *   These gameplay systems provide the primary context for many reputation-generating (or losing) events that will feed into `pallet-reputation`.
    *   They are also key drivers of $QRASL token utility (entry fees, rewards, bonds), thus interacting with the `$QRASL Token Pallet` and economic flows defined in the tokenomics.
    *   Staking $QRASL via `pallet-qrasl-staking` might unlock access to certain game features or reward tiers in the future.
*   **Ethics Training Modules & VCs (`docs/ETHICS_TRAINING_MODULES.md`):**
    *   The `pallet-vc-registry` is the direct on-chain implementation of the VC issuance and verification mechanism described in this document.
*   **DashAIBrowser (Frontend & Wallet - DashAI-Go):**
    *   Will serve as the primary interface for users to interact with these pallets: viewing their $QRASL balance, staking tokens, checking their reputation score, and managing their VCs.

## V. Anticipated Challenges & Conceptual Solutions (Pallet Development)

The development of these core Substrate pallets, while foundational, will present certain technical challenges.

*   **Challenge: Rust/Substrate Learning Curve & Complexity**
    *   For developers new to Rust and the Substrate framework, there can be a significant learning curve due to Rust's ownership and borrowing system, and Substrate's intricate macro system and pallet architecture.
    *   **Conceptual Solution:**
        *   **Focused Initial Scope:** Begin with the simplest, most well-defined pallets or functionalities within pallets first to build familiarity and confidence.
        *   **Leverage Substrate Resources:** Utilize Substrate's extensive official documentation, tutorials (e.g., Substrate Kitties, Node Template), and active community support channels (Discord, StackExchange).
        *   **Incremental Development:** Build and test features incrementally.
        *   **Pair Programming/Code Reviews:** Encourage collaborative development and thorough code reviews, especially for developers less experienced with Rust/Substrate.
        *   **Start with Standard Pallets:** Where possible (like for the token), heavily lean on or fork existing, well-audited standard Substrate pallets (e.g., `pallet-assets`, `pallet-staking`) and customize them, rather than building everything from scratch initially.

*   **Challenge: Smart Contract Security for Foundational Pallets**
    *   These pallets will manage core economic value ($QRASL), governance rights, and identity-linked attributes (reputation, VCs). Vulnerabilities could have severe consequences.
    *   **Conceptual Solution:**
        *   **Rigorous Test-Driven Development (TDD):** As stated in the strategy, comprehensive unit and integration tests are paramount. Aim for high test coverage.
        *   **Security Best Practices:** Adhere strictly to Rust and Substrate security best practices (e.g., preventing reentrancy, integer overflow/underflow, proper access control on dispatchables).
        *   **Multiple Independent Audits (Future):** Before any mainnet deployment, these critical pallets must undergo thorough security audits by reputable third-party firms specializing in Substrate/Rust.
        *   **Formal Verification (Conceptual - Long-term Goal):** For the most critical logic (e.g., token transfers, core staking mechanics), explore the feasibility of applying formal verification techniques to mathematically prove correctness, potentially leveraging conceptual tools like those from EchoSphere AI-vCPU's `Integrity_Monitor` for static analysis or proof assistance.
        *   **Bug Bounty Program:** Establish a bug bounty program post-launch to incentivize community security researchers to find and report vulnerabilities.

*   **Challenge: State Management Efficiency & Storage Optimization**
    *   Poorly designed storage patterns or inefficient state transitions in pallets can lead to high blockchain storage costs, increased transaction fees (gas), and slower performance.
    *   **Conceptual Solution:**
        *   **Minimal On-Chain State:** Only store data on-chain that absolutely needs to be there for security, verifiability, or direct smart contract logic. Leverage off-chain storage (DSL/IPFS) for larger data blobs, storing only hashes/CIDs on-chain.
        *   **Efficient Storage Patterns:** Use Substrate's storage types (`StorageValue`, `StorageMap`, `StorageDoubleMap`, `StorageNMap`) appropriately. Understand their performance characteristics.
        *   **Bounded Data Structures:** Use bounded `Vec`s and other collections where possible to prevent unbounded storage growth that could be exploited. Define clear limits and garbage collection strategies for on-chain lists or queues if necessary.
        *   **Benchmarking:** Regularly benchmark extrinsics to monitor their impact on storage reads/writes and computational weight. Optimize functions that are identified as bottlenecks.
        *   **Data Migration Strategies:** Plan for how on-chain storage might need to be migrated or upgraded in future runtime upgrades without disrupting existing data or functionality.
