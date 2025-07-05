# On-Chain Reputation System: Quantifying Trust in QRASL

The On-Chain Reputation System is a core component of the QRASL ecosystem, designed to quantify an account's trustworthiness, positive contributions, and standing within the community. This system directly influences a participant's governance power within the Zoologist's Guild and may unlock other privileges or responsibilities. It aims to incentivize beneficial behaviors and create a self-regulating environment where reputation has tangible on-chain value.

## 1. Core Principles

The design and operation of the On-Chain Reputation System are guided by the following core principles:

*   **Ecosystem-Beneficial Actions:** Reputation is primarily earned through actions that demonstrably benefit the QRASL ecosystem, its participants, or its core objectives, rather than through repetitive, easily gameable grinding activities.
*   **Reflection of Trustworthiness:** The system aims to reflect an account's reliability, skill in relevant areas (e.g., gameplay, governance participation), and positive social standing as evidenced by on-chain actions.
*   **Integrity through Consequences:** There must be clear, transparent, and fair mechanisms for both gaining and losing reputation. Negative actions that harm the ecosystem or its members should result in reputation loss.
*   **Dynamic and Evolving:** The reputation system itself may be subject to refinement over time via Zoologist's Guild governance, allowing the community to adapt it to new behaviors or challenges.
*   **Transparency:** While individual actions contributing to reputation changes are recorded on-chain, the exact formulas and weights for specific actions might be public or abstracted to prevent reverse-engineering for malicious exploitation, subject to Guild policy.

## 2. On-Chain Representation

The technical implementation of the reputation score needs to be robust and integrated with the account system.

*   **Storage:** An account's Reputation Score is stored as a signed integer (e.g., `i32` or `i64`, depending on the desired range and precision) within a dedicated data structure associated with their account identity on Shard 6. This could be part of a comprehensive `UserProfile` pallet/module or a specific `Reputation` pallet/module that links to the account ID.
*   **Rationale for Signed Integer:** A signed integer is crucial as it allows for the possibility of negative reputation scores. This is important for representing accounts that have engaged in severely detrimental activities, providing a broader spectrum for an account's standing beyond just neutral or positive.
*   **Initial State:** New accounts created on the QRASL network will start with a neutral reputation score (e.g., 0). Alternatively, a small positive baseline (e.g., 100) could be granted upon completion of an initial identity verification or tutorial process, if such systems are in place.
*   **Maximum/Minimum Values:** Consideration should be given to potential maximum and minimum cap values for the reputation score (e.g., `i32::MIN` to `i32::MAX`). While the conceptual range can be vast, practical limits might be implemented for storage or calculation efficiency, or to prevent runaway scores. These caps, if any, could also be Guild-adjustable parameters.

## 3. Positive Reputation Sources (Earning Trust)

Positive reputation is earned by performing actions that are verifiable on-chain and are deemed beneficial to the QRASL ecosystem or demonstrate skill, reliability, and constructive engagement. The specific point values and conditions are illustrative and would be Guild-configurable parameters.

### 3.1. Gameplay Skill & Mastery

*   **Action:** Consistently winning ranked `Critter Tactics` matches against opponents of similar or higher skill rating (MMR).
*   **Mechanism:**
    *   `+X_win_rep` reputation points per win in a ranked match.
    *   Potential bonus: `+X_bonus_mmr_diff_rep` if the opponent's MMR was significantly higher.
    *   Consideration: A daily or weekly cap on reputation earned purely from match wins to prevent excessive grinding by a small group of highly skilled players and to encourage diverse forms of positive engagement.

### 3.2. Reliability & Service (e.g., Caregiver Contracts)

*   **Action:** Successfully completing a `Caregiver` contract from the Zoologist's Lodge with demonstrably positive outcomes for the pet (e.g., significant happiness increase, skill improvement, no negative status effects incurred during care).
*   **Mechanism:**
    *   `+Y_care_contract_success_rep` reputation points per successfully completed contract.
    *   Potential bonus: `+Y_bonus_perfect_care_rep` for achieving 'perfect care' metrics (e.g., pet returned in peak condition, specific owner requests met).

### 3.3. Economic Acumen & Fair Trade

*   **Action:** Maintaining a high ratio of successful, undisputed trades on official QRASL marketplace(s) over a sustained period. (Requires a robust dispute resolution mechanism, TBD).
*   **Mechanism:**
    *   `+Z_trade_milestone_rep` reputation points upon completing `N` successful trades without a confirmed dispute.
    *   Alternatively, a periodic review (e.g., monthly) could award reputation based on maintaining a high 'successful trade ratio' (e.g., >99%) for accounts with significant trade volume.
    *   *Caution:* This area needs careful design to prevent wash trading solely for reputation gain. Focus should be on genuine economic activity.

### 3.4. Constructive Governance Participation

*   **Action:** Having a self-submitted proposal successfully pass a Zoologist's Guild vote and be enacted.
*   **Mechanism:** `+A_passed_proposal_rep` (a significant amount) awarded to the original proposer upon successful enactment of their proposal.
*   **Action:** Actively and consistently voting in Guild proposals over multiple governance epochs.
*   **Mechanism:**
    *   `+B_voting_participation_rep` reputation points per epoch for voting on `>P_voting_threshold%` of all proposals within that epoch.
    *   Potential bonus: `+B_bonus_voting_streak_rep` for maintaining such participation over several consecutive epochs.
*   **Action:** Successfully serving as an endorser for a proposal that subsequently passes.
*   **Mechanism:** `+C_successful_endorsement_rep` (a smaller amount) to each account that endorsed a proposal which then successfully passed a Guild vote. This incentivizes thoughtful endorsement.

### 3.5. Ecosystem Support & Development (Future Considerations & Discretionary Awards)

*   **Action:** Providing significant, verifiable liquidity to approved $QRASL or core ecosystem asset pools on recognized DEXs for a sustained period.
*   **Mechanism:** May require off-chain tracking or snapshotting by a Guild-approved process, or specific smart contract integrations. `+D_liquidity_provision_rep` awarded periodically.
*   **Action:** Discovering and responsibly reporting critical bugs or security vulnerabilities in core QRASL protocols or official smart contracts, which are subsequently verified and fixed by the development team.
*   **Mechanism:** Discretionary award of `+E_bug_bounty_rep` by a Guild committee or core team, based on severity and quality of the report.
*   **Action:** Creating high-quality, widely adopted community tools, educational content, or other public goods that significantly benefit the ecosystem.
*   **Mechanism:** May be awarded via `TreasurySpend` proposals that also include a reputation reward component, or through a dedicated 'Community Grants & Recognition' program managed by the Guild.

## 4. Negative Reputation Sources (Breaking Trust)

Negative reputation is incurred by performing actions that are verifiable on-chain (or through a Guild-sanctioned dispute resolution or moderation process) and are deemed harmful to the QRASL ecosystem, its participants, or the principles of fair play. Penalties should be proportionate to the severity and frequency of infractions.

### 4.1. Poor Sportsmanship / Anti-Social Behavior (Ranked Play & Community Interactions)

*   **Action:** Excessively or systematically conceding ranked `Critter Tactics` matches without genuine cause (e.g., clear evidence of attempting to manipulate rankings or grief opponents).
*   **Mechanism:** `-X_concede_abuse_rep` reputation points per confirmed incident. May require pattern detection or reporting thresholds to trigger review.
*   **Action:** Being repeatedly flagged by multiple unique players for abusive chat, harassment, or hate speech within official QRASL game environments or communication platforms, where such behavior is verified by a Guild-appointed moderation council or system (details of moderation system TBD).
*   **Mechanism:** `-Y_harassment_rep` reputation points per confirmed and severe incident. May also lead to temporary communication bans.

### 4.2. Negligence or Harm (e.g., Caregiver Contracts)

*   **Action:** Failing a `Caregiver` contract, resulting in demonstrably negative outcomes for the pet under care (e.g., significant happiness decrease, acquisition of preventable negative status effects, loss of skill due to neglect).
*   **Mechanism:** `-Z_care_contract_fail_rep` reputation points per failed contract. Severity might be scaled (e.g., minor negligence vs. gross negligence leading to severe pet deterioration).

### 4.3. Detrimental Governance Participation

*   **Action:** Submitting proposals to the Zoologist's Guild that are overwhelmingly rejected (e.g., <10% approval) AND are deemed frivolous, spammy, or clearly bad-faith by a post-vote review process or by consistently failing the endorsement phase.
*   **Mechanism:** `-A_spam_proposal_rep` reputation points per such proposal. The goal is to deter spam, not to punish genuine but unpopular ideas; hence the need for a "frivolous/spammy" determination.
*   **Action:** Consistently having proposal deposits forfeited due to failure to gain endorsement multiple times in a short period (e.g., `N` forfeited deposits within `M` epochs).
*   **Mechanism:** `-B_endorsement_failure_streak_rep` reputation points. This signals a potential lack of understanding of community sentiment or proposal quality.

### 4.4. Marketplace Misconduct & Fraud

*   **Action:** Confirmed cases of scamming, willful misrepresentation of items/assets, or engaging in other forms of malicious listings on official QRASL marketplaces. This requires a robust dispute resolution system (details TBD) capable of gathering evidence and making fair judgments.
*   **Mechanism:** `-C_marketplace_fraud_rep` (a very significant amount) per confirmed fraudulent activity. May also result in temporary or permanent marketplace bans, and forfeiture of any illicitly gained assets if technically feasible.

### 4.5. Exploiting System Vulnerabilities

*   **Action:** Verified exploitation of game bugs, smart contract vulnerabilities, or economic system loopholes for significant unfair personal gain, particularly after such exploits have been publicly acknowledged or a fix is in progress. (Does not apply to responsible disclosure of bugs).
*   **Mechanism:** `-D_exploit_abuse_rep` (a severe penalty) and potential further on-chain sanctions as determined by Guild governance or foundational protocol rules (e.g., asset seizure, account flagging).

### 4.6. Failure to Fulfill Elected/Appointed Duties (Future Consideration)

*   **Action:** If roles like 'Guardian Council' members or grants committee members are established, dereliction of duty or abuse of position by these individuals.
*   **Mechanism:** Would require a specific impeachment or penalty proposal within the Guild, leading to significant reputation loss and removal from the role.

## 5. Reputation Decay (The "Staying Active" Mechanic)

To ensure that reputation scores remain reflective of current and ongoing engagement with the QRASL ecosystem, and to prevent dormant accounts from retaining undue influence indefinitely, a reputation decay mechanism will be implemented.

*   **Purpose:**
    *   Encourage continued positive participation in the ecosystem.
    *   Prevent the accumulation of governance power by accounts that are no longer active contributors.
    *   Ensure that reputation is a dynamic measure of an account's present standing.
*   **Mechanism:**
    *   If an account does not perform any qualifying reputation-earning actions (or a specific subset of key positive actions, e.g., voting in governance, successfully completing a caregiver contract) within a defined "grace period" (e.g., `T_decay_grace_period_days` like 90 days), their reputation score will begin to decay.
    *   The decay applies only to positive reputation scores. It will gradually reduce the score towards a neutral baseline (e.g., 0 or the initial starting score like 100).
*   **Rate of Decay:**
    *   The decay rate (e.g., `R_decay_points_per_day` or `R_decay_percentage_per_week` of the current positive score) should be modest to avoid overly punishing casual but well-intentioned players.
    *   However, it must be significant enough over longer periods to differentiate truly active and engaged participants from those who have become inactive.
    *   The decay rate itself could be a Guild-configurable parameter.
*   **Decay Floor:** Reputation decay will not reduce an account's score below the designated neutral baseline (e.g., 0 or 100). It is not intended to push accounts into negative reputation solely due to inactivity. Negative reputation is reserved for explicit negative actions.
*   **Exemptions & Considerations:**
    *   **Active Staking for Governance:** Actively staking $QRASL for governance participation (i.e., to enable voting) might pause or significantly slow down reputation decay, even if the account doesn't perform other specific reputation-earning actions frequently. This acknowledges that staking is a form of active commitment.
    *   **Very High Reputation Tiers / Accolades:** The Guild could define specific, very high reputation tiers or unique, hard-earned on-chain accolades that grant partial or full immunity from decay, or a slower decay rate, as a reward for exceptional long-term positive impact. This would need careful consideration to avoid creating a permanent entrenched class.
    *   **Re-engagement:** Performing a significant reputation-earning action after a period of inactivity should reset the decay timer or halt decay.
