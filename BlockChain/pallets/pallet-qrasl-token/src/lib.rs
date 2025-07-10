#![cfg_attr(not(feature = "std"), no_std)]

/// Learn more about FRAME and Substrate runtime development here:
/// https://docs.substrate.io/reference/รูntime-development/
pub use pallet::*;

#[cfg(test)]
mod mock;

#[cfg(test)]
mod tests;

#[cfg(feature = "runtime-benchmarks")]
mod benchmarking;
pub mod weights;
pub use weights::*;

#[frame_support::pallet]
pub mod pallet {
    use super::*;
    // Removed duplicate imports:
    // use frame_support::pallet_prelude::*;
    // use frame_system::pallet_prelude::*;

    use frame_support::{
        pallet_prelude::*, // This one is fine at the top of the mod
        traits::{Currency, ExistenceRequirement, WithdrawReasons}, // Currency might be removed if not strictly used by this self-contained version
    };
    use frame_system::pallet_prelude::*; // This one is fine at the top of the mod
    use sp_runtime::traits::{CheckedAdd, CheckedSub, Zero};

    // Define Balance type. If T::Currency is not used directly for storage,
    // we might need a concrete type or a generic Balance from frame_support::traits::tokens::Balance.
    // For now, BalanceOf<T> is tied to T::Currency, which is fine.
    type BalanceOf<T> = <<T as Config>::Currency as Currency<<T as frame_system::Config>::AccountId>>::Balance;

    #[pallet::pallet]
    #[pallet::generate_store(pub(super) trait Store)]
    pub struct Pallet<T>(_);

    // Pallet configuration trait
    #[pallet::config]
    pub trait Config: frame_system::Config {
        /// Because this pallet emits events, it depends on the runtime's definition of an event.
        type Event: From<Event<Self>> + IsType<<Self as frame_system::Config>::Event>;

        /// The currency type for managing balances. This will be $QRASL.
        /// Using frame_support::traits::Currency allows this pallet to potentially work with any
        /// Substrate currency system, but it will be configured to use the native $QRASL token.
        type Currency: Currency<Self::AccountId>;

        /// Information on runtime weights for extrinsics in this pallet.
        type WeightInfo: WeightInfo;

        // New: Define an origin that can mint/burn tokens.
        // This would typically be Root or a governance-controlled origin.
        type MintOrigin: EnsureOrigin<Self::Origin>;
    }

    // The pallet's runtime storage items.
    // TotalSupply and Balances are managed by the `Currency` trait implementation (e.g., pallet-balances),
    // so we don't need to define them explicitly here if we are using pallet-balances.
    // If this pallet were to manage a custom token *independently* of pallet-balances,
    // then TotalSupply and Balances storage items would be defined here.
    // For now, assuming this pallet *orchestrates* actions on a token managed by pallet-balances or similar.
    // Decision: For this initial implementation, we will define TotalSupply and Balances
    // within this pallet to make it a self-contained, lean token pallet.
    // This provides clarity for the planned extrinsics (mint, transfer, burn).
    // Integration with a more comprehensive `Currency` trait implementation like `pallet-balances`
    // can be considered as a future refinement or if more advanced currency features are needed.

    #[pallet::storage]
    #[pallet::getter(fn total_supply)]
    /// The total issuance of the $QRASL token.
    pub type TotalSupply<T: Config> = StorageValue<_, BalanceOf<T>, ValueQuery>;

    #[pallet::storage]
    #[pallet::getter(fn account_balance)] // Renamed getter to avoid conflict if Currency trait is used directly later
    /// The balance of a specific account.
    pub type Balances<T: Config> = StorageMap<_, Blake2_128Concat, T::AccountId, BalanceOf<T>, ValueQuery>;

    // Pallets use events to inform users when important changes are made.
    #[pallet::event]
    #[pallet::generate_deposit(pub(super) fn deposit_event)]
    pub enum Event<T: Config> {
        /// Tokens were minted for an account. [who, amount]
        Minted { who: T::AccountId, amount: BalanceOf<T> },
        /// Tokens were transferred from one account to another. [from, to, amount]
        Transferred { from: T::AccountId, to: T::AccountId, amount: BalanceOf<T> },
        /// Tokens were burned from an account. [who, amount]
        Burned { who: T::AccountId, amount: BalanceOf<T> },
    }

    // Errors inform users that something went wrong.
    #[pallet::error]
    pub enum Error<T> {
        /// The account has an insufficient balance to complete the operation.
        InsufficientBalance,
        /// The amount specified must be greater than zero.
        AmountMustBePositive,
        /// An arithmetic operation resulted in an overflow.
        Overflow,
        /// An arithmetic operation resulted in an underflow (e.g. total supply).
        Underflow,
    }

    #[pallet::call]
    impl<T: Config> Pallet<T> {
        /// Mint new tokens.
        ///
        /// Can only be called by `T::MintOrigin` (e.g., Root or a governance-controlled account).
        /// Increases the total supply and the balance of the `to` account.
        ///
        /// Parameters:
        /// - `origin`: The origin of the call, must satisfy `T::MintOrigin`.
        /// - `to`: The account to mint tokens to.
        /// - `amount`: The amount of tokens to mint.
        ///
        /// Emits `Minted` event on success.
        #[pallet::weight(T::WeightInfo::mint())]
        pub fn mint(origin: OriginFor<T>, to: T::AccountId, amount: BalanceOf<T>) -> DispatchResult {
            T::MintOrigin::ensure_origin(origin)?;
            ensure!(amount > Zero::zero(), Error::<T>::AmountMustBePositive);

            let new_total_supply = TotalSupply::<T>::get().checked_add(&amount).ok_or(Error::<T>::Overflow)?;
            let to_balance = Balances::<T>::get(&to);
            let new_to_balance = to_balance.checked_add(&amount).ok_or(Error::<T>::Overflow)?;

            TotalSupply::<T>::put(new_total_supply);
            Balances::<T>::insert(&to, new_to_balance);

            Self::deposit_event(Event::Minted { who: to, amount });
            Ok(())
        }

        /// Transfer tokens from the caller's account to another account.
        ///
        /// Parameters:
        /// - `origin`: The origin of the call, must be a signed account.
        /// - `to`: The recipient account.
        /// - `amount`: The amount of tokens to transfer.
        ///
        /// Emits `Transferred` event on success.
        #[pallet::weight(T::WeightInfo::transfer())]
        pub fn transfer(origin: OriginFor<T>, to: T::AccountId, amount: BalanceOf<T>) -> DispatchResult {
            let sender = ensure_signed(origin)?;
            ensure!(amount > Zero::zero(), Error::<T>::AmountMustBePositive);

            let sender_balance = Balances::<T>::get(&sender);
            ensure!(sender_balance >= amount, Error::<T>::InsufficientBalance);

            let new_sender_balance = sender_balance.checked_sub(&amount).ok_or(Error::<T>::Underflow)?; // Should not happen due to check above
            let receiver_balance = Balances::<T>::get(&to);
            let new_receiver_balance = receiver_balance.checked_add(&amount).ok_or(Error::<T>::Overflow)?;

            Balances::<T>::insert(&sender, new_sender_balance);
            Balances::<T>::insert(&to, new_receiver_balance);

            // Note: TotalSupply does not change on transfer.

            Self::deposit_event(Event::Transferred { from: sender, to, amount });
            Ok(())
        }

        /// Burn tokens from the caller's account.
        ///
        /// Decreases the total supply and the balance of the caller.
        ///
        /// Parameters:
        /// - `origin`: The origin of the call, must be a signed account.
        /// - `amount`: The amount of tokens to burn.
        ///
        /// Emits `Burned` event on success.
        #[pallet::weight(T::WeightInfo::burn())]
        pub fn burn(origin: OriginFor<T>, amount: BalanceOf<T>) -> DispatchResult {
            let who = ensure_signed(origin)?;
            ensure!(amount > Zero::zero(), Error::<T>::AmountMustBePositive);

            let who_balance = Balances::<T>::get(&who);
            ensure!(who_balance >= amount, Error::<T>::InsufficientBalance);

            let new_who_balance = who_balance.checked_sub(&amount).ok_or(Error::<T>::Underflow)?; // Should not happen
            let current_total_supply = TotalSupply::<T>::get();
            let new_total_supply = current_total_supply.checked_sub(&amount).ok_or(Error::<T>::Underflow)?;


            Balances::<T>::insert(&who, new_who_balance);
            TotalSupply::<T>::put(new_total_supply);

            Self::deposit_event(Event::Burned { who, amount });
            Ok(())
        }
    }
}
