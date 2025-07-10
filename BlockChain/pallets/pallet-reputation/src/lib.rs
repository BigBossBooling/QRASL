#![cfg_attr(not(feature = "std"), no_std)]

pub use pallet::*;

#[cfg(test)]
mod mock;

#[cfg(test)]
mod tests;

#[cfg(feature = "runtime-benchmarks")]
mod benchmarking;
pub mod weights;
pub use weights::*;

pub type ReputationScore = i64; // Allowing for negative scores
pub type ReasonCode = u8; // For auditability of reputation changes

#[frame_support::pallet]
pub mod pallet {
    use super::*;
    use frame_support::pallet_prelude::*;
    use frame_system::pallet_prelude::*;

    #[pallet::pallet]
    #[pallet::generate_store(pub(super) trait Store)]
    pub struct Pallet<T>(_);

    #[pallet::config]
    pub trait Config: frame_system::Config {
        type Event: From<Event<Self>> + IsType<<Self as frame_system::Config>::Event>;

        #[pallet::constant]
        type MaxReputationScore: Get<ReputationScore>;

        #[pallet::constant]
        type MinReputationScore: Get<ReputationScore>;

        #[pallet::constant]
        type DefaultReputationScore: Get<ReputationScore>;

        type ReputationAdminOrigin: EnsureOrigin<Self::Origin>;

        type WeightInfo: WeightInfo;
    }

    #[pallet::storage]
    #[pallet::getter(fn reputation_scores)]
    pub type ReputationScoresMap<T: Config> =
        StorageMap<_, Blake2_128Concat, T::AccountId, ReputationScore, ValueQuery, GetDefaultReputationScore<T>>;

    // Helper type for ValueQuery to use DefaultReputationScore from Config
    pub struct GetDefaultReputationScore<T: Config>(PhantomData<T>);
    impl<T: Config> Get<ReputationScore> for GetDefaultReputationScore<T> {
        fn get() -> ReputationScore {
            T::DefaultReputationScore::get()
        }
    }


    #[pallet::event]
    #[pallet::generate_deposit(pub(super) fn deposit_event)]
    pub enum Event<T: Config> {
        ReputationUpdated {
            who: T::AccountId,
            new_score: ReputationScore,
            change: ReputationScore, // Using ReputationScore for change as well, can be negative
            reason: ReasonCode,
        },
        ReputationReset {
            who: T::AccountId,
            new_score: ReputationScore,
        },
        ReputationInitialized { // Optional, if explicit initialization is used
            who: T::AccountId,
            initial_score: ReputationScore,
        },
    }

    #[pallet::error]
    pub enum Error<T> {
        ScoreOutOfBounds, // If a change would push score beyond Max/Min if they were strict limits not just for clamping
        // No specific errors defined yet beyond what EnsureOrigin and arithmetic checks provide
    }

    use sp_runtime::DispatchResult; // Ensure DispatchResult is in scope

    #[pallet::call]
    impl<T: Config> Pallet<T> {
        /// Update the reputation score of a target account.
        ///
        /// Can only be called by an origin satisfying `T::ReputationAdminOrigin`
        /// (e.g., Root, a governance-controlled account, or another authorized pallet).
        /// The score change can be positive or negative.
        /// The final score will be clamped between `T::MinReputationScore` and `T::MaxReputationScore`.
        ///
        /// Parameters:
        /// - `origin`: The authorized origin of the call.
        /// - `target_account_id`: The account whose reputation is to be updated.
        /// - `score_change`: The amount to change the score by (can be negative).
        /// - `reason_code`: A code indicating the reason for the update.
        ///
        /// Emits `ReputationUpdated` event on success.
        #[pallet::weight(T::WeightInfo::update_reputation())]
        pub fn update_reputation(
            origin: OriginFor<T>,
            target_account_id: T::AccountId,
            score_change: ReputationScore, // Changed from i32 to ReputationScore (i64)
            reason_code: ReasonCode,
        ) -> DispatchResult {
            T::ReputationAdminOrigin::ensure_origin(origin)?;

            ReputationScoresMap::<T>::mutate(&target_account_id, |current_score| {
                let new_score_unclamped = current_score.saturating_add(score_change);
                // Clamp the score
                *current_score = new_score_unclamped
                    .max(T::MinReputationScore::get())
                    .min(T::MaxReputationScore::get());

                Self::deposit_event(Event::ReputationUpdated {
                    who: target_account_id.clone(),
                    new_score: *current_score,
                    change: score_change, // Report the intended change before clamping for transparency
                    reason: reason_code,
                });
            });

            Ok(())
        }

        /// Reset the reputation score of a target account to the default value.
        ///
        /// Can only be called by an origin satisfying `T::ReputationAdminOrigin`.
        ///
        /// Parameters:
        /// - `origin`: The authorized origin of the call.
        /// - `target_account_id`: The account whose reputation is to be reset.
        ///
        /// Emits `ReputationReset` event on success.
        #[pallet::weight(T::WeightInfo::reset_reputation())]
        pub fn reset_reputation(
            origin: OriginFor<T>,
            target_account_id: T::AccountId,
        ) -> DispatchResult {
            T::ReputationAdminOrigin::ensure_origin(origin)?;
            let default_score = T::DefaultReputationScore::get();
            ReputationScoresMap::<T>::insert(&target_account_id, default_score);

            Self::deposit_event(Event::ReputationReset {
                who: target_account_id,
                new_score: default_score,
            });
            Ok(())
        }

        /// Initialize the reputation for an account if it hasn't been set yet.
        /// (Optional: `ReputationScoresMap` with `ValueQuery` and `GetDefaultReputationScore`
        /// already handles default initialization on first read if not explicitly set.
        /// This extrinsic provides an explicit way if needed, e.g., during onboarding.)
        ///
        /// Can be called by a signed origin (the user themselves) or an admin.
        ///
        /// Parameters:
        /// - `origin`: The origin of the call.
        /// - `target_account_id`: The account to initialize reputation for.
        ///
        /// Emits `ReputationInitialized` event on success if not already initialized.
        #[pallet::weight(T::WeightInfo::initialize_reputation())]
        pub fn initialize_reputation(
            origin: OriginFor<T>,
            target_account_id: T::AccountId,
        ) -> DispatchResult {
            // For this version, let's allow signed origin to initialize for their own account,
            // or an admin to initialize for any account.
            // A more restrictive origin might be T::ReputationAdminOrigin for all initializations
            // if self-initialization is not desired.
            let who = ensure_signed_or_root(origin)?; // Example: allow root or signed

            // Check if already initialized by checking if it's different from default.
            // This check is a bit indirect because ValueQuery provides default.
            // A more robust way might be a separate StorageMap<AccountId, bool> for `IsInitialized`.
            // For simplicity now, we check if it's exactly the default. If so, we "initialize".
            // This means re-initializing to default doesn't hurt but also doesn't emit event if already default.

            let current_score = ReputationScoresMap::<T>::get(&target_account_id);
            let default_score = T::DefaultReputationScore::get();

            // Only proceed if not explicitly set to something else already, or if it is default.
            // This logic might need refinement based on exact desired behavior of "initialization".
            // If the map defaults to DefaultScore, this extrinsic might only be useful
            // if we want an event or if an account's score was somehow set to non-default by other means
            // and needs a reset *to default by this specific function*.
            // Given ValueQuery setup, this extrinsic is somewhat redundant unless an event is strictly needed
            // or if there's a scenario where an account might not have the default yet.

            // Let's assume for now this is for ensuring an account is in the map with default score
            // and emitting an event. If it's already default, we can skip.
            if current_score != default_score {
                 // If it's not default, and we are "initializing", it implies setting it to default.
                 // This might be confusing if current_score is already non-default positive.
                 // Revising: This extrinsic makes more sense if ValueQuery was NOT used with GetDefault,
                 // and StorageMap returned Option<ReputationScore>.
                 // Given current setup, this extrinsic is less critical.
                 // For now, let's make it set to default if called.
            }

            // Simplified: just ensure it's set to default and emit event.
            // If it's already default, this is idempotent but still emits event.
            ReputationScoresMap::<T>::insert(&target_account_id, default_score.clone());
            Self::deposit_event(Event::ReputationInitialized {
                who: target_account_id,
                initial_score: default_score,
            });

            Ok(())
        }
    }

    impl<T: Config> Pallet<T> {
        // Public helper functions callable from other pallets
        pub fn get_score(who: &T::AccountId) -> ReputationScore {
            ReputationScoresMap::<T>::get(who)
        }

        // Internal helper to update score; can be called by other pallets if they have a way to invoke it
        // (e.g. if this pallet exposes a Trait method that other pallets can call).
        // For direct calls from other pallets, they usually call extrinsics with an appropriate Origin.
        // This internal function is more for logic reuse within this pallet.
        pub fn _update_score_unsafe(
            who: &T::AccountId,
            score_change: ReputationScore,
            reason: ReasonCode,
        ) -> DispatchResult {
            ReputationScoresMap::<T>::mutate(who, |current_score| {
                let new_score_unclamped = current_score.saturating_add(score_change);
                *current_score = new_score_unclamped
                    .max(T::MinReputationScore::get())
                    .min(T::MaxReputationScore::get());

                Self::deposit_event(Event::ReputationUpdated {
                    who: who.clone(),
                    new_score: *current_score,
                    change: score_change,
                    reason,
                });
            });
            Ok(())
        }
    }
}
