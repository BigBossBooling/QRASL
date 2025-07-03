# Zoologist's Guild: DAO Governance Protocol

The Zoologist's Guild is the decentralized autonomous organization (DAO) at the heart of the QRASL ecosystem, operating on Shard 6 (Governance & Data Bridge). It empowers $QRASL token holders and active participants to collectively manage the evolution, economic health, and community initiatives of the network. Through a structured proposal and voting system, the Guild ensures that QRASL remains adaptive, responsive, and aligned with the long-term interests of its user base. This document outlines the governance protocol, proposal lifecycle, voting mechanics, treasury management, and initial proposal types for the Zoologist's Guild.

## 1. The Proposal Lifecycle

The proposal lifecycle is designed to be transparent, inclusive, and secure, ensuring that ideas are properly vetted by the community before being put to a formal vote and potential enactment.

### 1.1. Submission

*   **Eligibility:** Any account holding a minimum of `X_submission_QRASL_stake` $QRASL (e.g., 10,000 $QRASL) and an on-chain Reputation score of at least `Y_submission_reputation_score` (e.g., 500) can submit a proposal. These thresholds are configurable Guild parameters.
*   **Mechanism:** Proposals are submitted by calling a dedicated function on the Zoologist's Guild smart contract deployed on Shard 6. The proposal content should be structured and may include a title, detailed description, rationale, and specific actions or parameter changes requested.
*   **Proposal Deposit:** A proposal submission requires a deposit of `Z_deposit_QRASL` $QRASL (e.g., 1,000 $QRASL).
    *   This deposit is returned to the proposer if the proposal successfully passes the endorsement phase and proceeds to a formal vote.
    *   The deposit is forfeited (e.g., sent to the Guild Treasury or burned) if the proposal fails to meet the endorsement criteria or is subsequently flagged and confirmed as malicious or spam by a Guild-governed moderation process (details of moderation TBD).

### 1.2. Seconding/Endorsement Phase

*   **Purpose:** To filter out proposals with low community interest or potential spam before they consume the resources of a full voting period.
*   **Mechanism:** Once submitted, a proposal enters an endorsement phase. It must be seconded (endorsed) by at least `N_endorsement_count` unique accounts (e.g., 50 accounts).
*   **Endorser Eligibility:** Each endorsing account must also meet minimum $QRASL holding `A_endorse_QRASL_stake` (e.g., 100 $QRASL) and Reputation `B_endorse_reputation_score` (e.g., 100) thresholds. These are distinct from submission thresholds and are also configurable Guild parameters.
*   **Duration:** The endorsement phase lasts for `T_endorse_hours` (e.g., 72 hours) from the time of submission.
*   **Outcome:**
    *   **Success:** If the proposal receives the required number of unique endorsements within the timeframe, it proceeds to the formal voting period. The proposer's initial deposit is returned.
    *   **Failure:** If the proposal fails to achieve the endorsement threshold, it is marked as 'Archived' or 'FailedEndorsement'. The proposer's deposit is forfeited as described above.

### 1.3. Voting Period (Aligned with Governance Epochs)

*   **Progression:** Successfully endorsed proposals are scheduled for a formal vote. Voting typically aligns with pre-defined Governance Epochs to provide a predictable rhythm for community participation.
*   **Duration:** A standard voting period for a proposal is `T_vote_days` (e.g., 7 days). Multiple proposals may be voted on concurrently within the same epoch.
*   **Quorum:** For a vote on any given proposal to be considered valid, a minimum quorum of `Q_quorum_percentage` (e.g., 10%) of the total *active voting power* registered with the Guild at the start of that proposal's voting window must participate. (Definition of "active voting power" may refer to accounts that have voted in recent epochs or have explicitly registered to participate in governance).
*   **Pass Threshold:** Assuming quorum is met, a simple majority (`>50%`) of the votes cast in favor is required for a proposal to pass.
    *   *Note:* Certain critical proposal types (e.g., changes to core Guild mechanics, emergency actions) may be defined by the Guild to require a higher supermajority (e.g., 66.7%).

### 1.4. Execution & Enactment

*   **Queueing:** Passed proposals are added to an execution queue on Shard 6.
*   **Timelock:** A mandatory timelock period of `T_timelock_hours` (e.g., 48 hours) applies after a proposal officially passes the vote and before its on-chain execution. This period serves as a final safeguard, allowing the community to review the implications and for any emergency intervention mechanisms (e.g., a 'Guardian Council' vote, details TBD) to be triggered if a critical flaw is discovered.
*   **Implementation:**
    *   **Automated Execution:** For proposals involving changes to smart contract parameters or predefined actions within the Guild's control, execution is an automated function call made by the Guild contract itself at the end of the timelock.
    *   **Manual/Coordinated Execution:** For proposals requiring actions beyond the direct automation capabilities of the Guild contract (e.g., funding external development teams, implementing complex off-chain elements of a strategy), the proposal itself must clearly define:
        *   The trusted parties, multi-sig committees, or pre-defined processes responsible for carrying out the enactment.
        *   Measurable milestones or deliverables.
        *   The mechanism for verifying successful completion.
        These aspects are implicitly approved by the Guild when the proposal is passed.

## 2. Voting Mechanics & Power

The Guild's voting system is designed to balance the influence of various forms of stake and contribution to the QRASL ecosystem, ensuring that governance power is distributed thoughtfully.

### 2.1. Calculation of Voting Power (VP)

*   **Formula:** An account's Voting Power (VP) is calculated using a weighted formula that considers multiple dimensions of engagement and investment within the QRASL ecosystem:
    `VP = (Zoologist_Level * W_level) + (Sum_of_Evolved_Pet_Levels * W_pets) + (Reputation_Score * W_reputation) + (Staked_QRASL_Amount * W_stake)`
*   **Components Defined:**
    *   `Zoologist_Level`: The account's current level within the primary QRASL game/application, reflecting overall progress and experience.
    *   `Sum_of_Evolved_Pet_Levels`: The sum of the current levels of all "Evolved" status pets owned by the account. This rewards investment in and development of significant in-game assets. (Consideration: Rarity of pets could also be a multiplier here in a future iteration).
    *   `Reputation_Score`: The account's on-chain reputation, earned through positive contributions to the ecosystem (e.g., helpful participation, successful past proposals, content creation if tracked).
    *   `Staked_QRASL_Amount`: The total amount of $QRASL tokens actively staked by the account in a designated governance staking contract on Shard 6. This represents direct financial commitment.
*   **Weights (`W_level`, `W_pets`, `W_reputation`, `W_stake`):** These weights determine the relative influence of each component on the final VP.
    *   They are critical Guild parameters, initially set to establish a balanced system (e.g., `W_level=10`, `W_pets=1` (as pet levels might be higher than Zoologist level), `W_reputation=5`, `W_stake=0.01` (if QRASL amounts are large)).
    *   These weights can be adjusted over time via Guild proposals of type `ParameterChange`, allowing the community to fine-tune the governance balance.
*   **Active Participation Multiplier (Optional Consideration):** A small bonus multiplier could be applied to VP if an account has actively participated (voted) in a certain percentage of proposals over the last `M` epochs, encouraging sustained engagement.

### 2.2. Snapshotting of Voting Power

*   **Mechanism:** To ensure fairness and prevent manipulation of voting power during an active vote, an account's VP (and the underlying components like staked QRASL, pet levels, etc.) is snapshotted at a specific point in time for each proposal.
*   **Timing:** The snapshot is typically taken at the precise moment a proposal officially moves from the endorsement phase to the voting period. All votes cast for that proposal will use the VP calculated from this snapshot.

### 2.3. Vote Delegation

*   **Purpose:** To allow token holders or participants who may not have the time or expertise to review every proposal to still have their voice heard by entrusting their VP to a trusted community member or representative.
*   **Mechanism:** Users can delegate their entire calculated Voting Power to another QRASL account (the "Delegate").
    *   Delegation is non-custodial; the delegator always retains ownership of their $QRASL tokens and other assets.
    *   An account can delegate its VP to only one Delegate at any given time.
    *   Delegation (and revocation or re-delegation to a different Delegate) can be performed at any time via a transaction on Shard 6.
    *   Changes in delegation (new delegation, revocation, re-delegation) take effect for any proposal voting periods that *begin* after the delegation transaction is confirmed. They do not affect ongoing votes for which VP has already been snapshotted.
*   **Delegate Status & Considerations:**
    *   **Registration:** Delegates may be required to register by signaling their intent to act as a representative, possibly meeting certain minimum criteria (e.g., minimum self-delegated VP, minimum reputation).
    *   **Delegate Platforms:** The ecosystem may support platforms or UIs where potential delegates can create profiles, state their positions on issues, and solicit delegation.
    *   **Liquid Democracy Aspect:** This system introduces elements of liquid democracy, allowing for a more flexible and dynamic representation model than pure direct voting.

## 3. The Guild Treasury & Economic Levers

The Zoologist's Guild Treasury is a critical component for fostering ecosystem growth, funding community initiatives, and ensuring the long-term sustainability of QRASL. The Guild also has the power to influence key economic parameters of the network.

### 3.1. Treasury Funding

The Guild Treasury is funded through a diversified set of on-chain revenue streams, ensuring continuous capital inflow. The specific percentages for these streams are Guild-configurable parameters.

*   **Sources:**
    *   **Marketplace Fees:** A percentage (`P_market_fee_to_treasury%`) of all transaction fees generated from the official QRASL NFT marketplace(s) and other approved economic platforms operating within the ecosystem.
    *   **Primary Sales Royalties (Potential):** A portion (`P_primary_sale_royalty%`) of revenue from initial sales of certain first-party QRASL assets or future game expansions, if applicable.
    *   **Network Action Fees:** A percentage (`P_network_action_fee_to_treasury%`) of fees collected from specific, value-added network actions. Examples include:
        *   Decentralized Identity (DID) registration or renewal fees.
        *   Fees for deploying or registering application-specific setups on Shard 5.
        *   Fees for utilizing specialized services on Shard 2 (Utility & Storage Shard), beyond basic transaction costs.
    *   **Forfeited Proposal Deposits:** $QRASL deposits from proposals that fail the endorsement phase or are deemed malicious may be directed to the Treasury.
    *   **Voluntary Donations:** The Treasury can accept direct, voluntary donations of $QRASL and potentially other approved assets.
*   **Parameter Adjustability:** All contribution percentages (`P_...%`) are parameters that can be adjusted over time via `ParameterChange` proposals passed by the Guild.

### 3.2. Treasury Spending & Proposals

The allocation and disbursement of funds from the Guild Treasury are strictly governed by Guild proposals, ensuring community oversight and alignment with strategic priorities.

*   **Process:** Any expenditure from the Treasury must be approved via a `TreasurySpend` proposal. Such proposals must clearly state:
    *   The recipient(s) of the funds.
    *   The total amount requested.
    *   A detailed purpose or project description.
    *   Measurable objectives, milestones, and/or deliverables.
    *   A timeline for execution and reporting (if applicable).
*   **Typical Categories for Treasury Spending:**
    *   **Ecosystem Grants & Bounties:** Funding for community developers, artists, researchers, content creators, and other contributors to build tools, create assets, conduct research, or improve the QRASL ecosystem.
    *   **Event Sponsorship & Community Initiatives:** Supporting community-organized tournaments, educational workshops, marketing campaigns, meetups, and other initiatives that foster engagement and growth.
    *   **Liquidity Incentives:** Allocating funds to incentivize liquidity provision for $QRASL and related assets on approved Decentralized Exchanges (DEXs) or other financial protocols.
    *   **Security Audits & Infrastructure:** Paying for third-party security audits of critical smart contracts, protocols, or network components. Funding the development or maintenance of essential public infrastructure.
    *   **Strategic Partnerships & Collaborations:** Providing resources to foster collaborations that bring strategic value to QRASL.

### 3.3. Managing Economic Levers

The Guild holds significant responsibility in shaping the macroeconomic landscape of the QRASL network by governing key economic parameters.

*   **Mechanism:** Adjustments to critical economic levers are enacted through specific proposal types (e.g., `BurnRateConfiguration`, `InflationParameterChange`).
*   **Adaptive $QRASL Burn Rate:**
    *   The Guild controls the proportion of various network fees (e.g., transaction fees, intent processing fees from IHDB shards, Shard 2 service fees) that are permanently burned, versus those directed to other uses (e.g., Guild Treasury, block proposers/validators, Solver incentives).
    *   This adaptive burn rate is a powerful tool for managing $QRASL token scarcity and can be adjusted by the Guild to respond to changing network conditions, transaction volumes, and overall economic health, serving as a sophisticated lever for long-term value sustainability.
*   **Inflation Control (Staking Rewards & Issuance):**
    *   While the core issuance schedule for $QRASL might be defined in the foundational protocol, the Guild may have the authority to propose adjustments to parameters influencing the distribution of staking rewards or other minor inflationary/deflationary mechanisms.
    *   For example, the Guild could vote on changing the allocation percentages of staking rewards between different validator pools or adjusting reward rates within protocol-defined bounds.
*   **Fee Structures:** The Guild may also govern certain network-wide fee structures, such as base fees for specific services on Shard 2 or standardized costs for DID registrations, ensuring they remain fair and appropriate.

## 4. Initial Proposal Types

To ensure clarity, manageability, and focused deliberation, the Zoologist's Guild will initially support a defined set of proposal categories. This list can be expanded or modified in the future via a `NewProposalType` proposal itself, allowing the governance system to adapt to evolving needs. Each proposal type may have unique validation criteria or data requirements.

### 4.1. `ParameterChange`

*   **Description:** Proposals aimed at modifying configurable parameters of the Zoologist's Guild smart contract or other network parameters explicitly designated as Guild-governable and operating on Shard 6.
*   **Scope:**
    *   Adjusting thresholds for proposal submission (e.g., `X_submission_QRASL_stake`, `Y_submission_reputation_score`).
    *   Modifying endorsement phase parameters (e.g., `N_endorsement_count`, `T_endorse_hours`).
    *   Changing voting period parameters (e.g., `T_vote_days`, `Q_quorum_percentage`, pass thresholds for specific proposal types).
    *   Updating weights used in the Voting Power calculation (e.g., `W_level`, `W_pets`, `W_reputation`, `W_stake`).
    *   Altering fee percentages directed to the Guild Treasury (e.g., `P_market_fee_to_treasury%`).
    *   Modifying parameters of other Shard 6 systems under Guild control.
*   **Payload:** Must clearly specify the parameter to be changed, its current value, and the proposed new value.

### 4.2. `TreasurySpend`

*   **Description:** Proposals seeking to allocate funds from the Guild Treasury for specific projects, initiatives, grants, or operational expenses.
*   **Scope:** As detailed in Section 3.2 (Ecosystem Grants, Event Sponsorship, Liquidity Incentives, Security Audits, etc.).
*   **Payload:** Must include recipient address(es), requested $QRASL amount(s), detailed justification, project plan/milestones, and reporting commitments. For significant or ongoing funding, phased disbursements tied to milestone completion may be proposed.

### 4.3. `EconomicPolicyAdjust` (Formerly `BurnRateConfiguration`)

*   **Description:** Proposals focused on adjusting key macroeconomic levers of the QRASL network to manage tokenomics and ensure long-term economic health. This is a more encompassing category for economic adjustments.
*   **Scope:**
    *   Modifying the adaptive burn rate of $QRASL (i.e., the proportion of various network fees that are burned vs. recycled/reallocated).
    *   Adjusting the distribution formulas for network fees not burned (e.g., allocation to proposers, validators, Guild Treasury).
    *   Proposing changes to staking reward distribution mechanisms or rates (within protocol-defined boundaries, if applicable).
    *   Adjusting parameters for network-wide fee structures under Guild control.
*   **Payload:** Must clearly define the economic parameter to be adjusted, its current setting, the proposed new setting, and a strong rationale supported by data or modeling where possible.

### 4.4. `TextProposal` (Community Signaling & Guidance)

*   **Description:** A non-binding proposal type used to gauge community sentiment, issue official statements, or provide formal guidance on topics that do not involve direct on-chain code changes executable by the Guild contract itself.
*   **Scope:**
    *   Endorsing community-developed standards or best practices.
    *   Suggesting strategic directions or priorities for ecosystem development (to be considered by core teams or grant recipients).
    *   Formally recognizing community contributions.
    *   Stating the Guild's official stance on an external event or issue relevant to QRASL.
*   **Payload:** Primarily textual content, clearly articulating the statement, sentiment, or guidance. While non-binding in terms of contract execution, a passed `TextProposal` carries significant weight as the official voice of the community.

### 4.5. `NewProposalType` (Meta-Governance)

*   **Description:** A meta-governance proposal to add a new, formally recognized category of proposal that the Guild can deliberate and vote on.
*   **Scope:** Allows the Guild to evolve its own governance capabilities. For instance, introducing a `ProtocolUpgradeSignal` type if direct on-chain upgrades are not initially feasible but community consensus on desired core changes needs to be formally established.
*   **Payload:** Must define the name of the new proposal type, its intended scope, any specific data requirements or validation logic for proposals of this new type, and the rationale for its introduction.

### 4.6. `RoleNomination` / `Ratification` (Potential Future Type)

*   **Description (Conceptual):** For managing community-elected roles or councils if such bodies are established by the Guild (e.g., a 'Guardian Council' for emergency timelock interventions, or members of a grants review committee).
*   **Scope:** Nominating candidates, voting on appointments, or ratifying selections made through other approved processes.
*   **Note:** This is a placeholder for future consideration as the Guild's operational complexity grows.
