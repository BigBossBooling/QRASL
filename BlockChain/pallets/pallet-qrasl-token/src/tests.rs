use crate::{mock::*, Error, Event};
use frame_support::{assert_noop, assert_ok};
use sp_runtime::DispatchError;

#[test]
fn initial_total_supply_should_be_zero() {
    new_test_ext().execute_with(|| {
        assert_eq!(QraslToken::total_supply(), 0);
    });
}

#[test]
fn mint_works() {
    new_test_ext().execute_with(|| {
        System::set_block_number(1); // Needed for events
        assert_ok!(QraslToken::mint(Origin::root(), 1, 100));
        assert_eq!(QraslToken::account_balance(&1), 100);
        assert_eq!(QraslToken::total_supply(), 100);
        System::assert_last_event(Event::Minted { who: 1, amount: 100 }.into());

        assert_ok!(QraslToken::mint(Origin::root(), 2, 50));
        assert_eq!(QraslToken::account_balance(&2), 50);
        assert_eq!(QraslToken::total_supply(), 150);
        System::assert_last_event(Event::Minted { who: 2, amount: 50 }.into());
    });
}

#[test]
fn mint_fails_for_non_root_origin() {
    new_test_ext().execute_with(|| {
        assert_noop!(
            QraslToken::mint(Origin::signed(1), 1, 100),
            DispatchError::BadOrigin
        );
    });
}

#[test]
fn mint_fails_for_zero_amount() {
    new_test_ext().execute_with(|| {
        assert_noop!(
            QraslToken::mint(Origin::root(), 1, 0),
            Error::<Test>::AmountMustBePositive
        );
    });
}

#[test]
fn mint_fails_on_total_supply_overflow() {
    new_test_ext().execute_with(|| {
        QraslToken::mint(Origin::root(), 1, Balance::MAX).unwrap();
        assert_noop!(
            QraslToken::mint(Origin::root(), 2, 1),
            Error::<Test>::Overflow
        );
    });
}

#[test]
fn mint_fails_on_balance_overflow() {
    new_test_ext().execute_with(|| {
        QraslToken::mint(Origin::root(), 1, Balance::MAX).unwrap();
         assert_noop!(
            QraslToken::mint(Origin::root(), 1, 1), // Minting again to same account
            Error::<Test>::Overflow
        );
    });
}


#[test]
fn transfer_works() {
    new_test_ext().execute_with(|| {
        System::set_block_number(1);
        assert_ok!(QraslToken::mint(Origin::root(), 1, 100));
        assert_ok!(QraslToken::transfer(Origin::signed(1), 2, 30));
        assert_eq!(QraslToken::account_balance(&1), 70);
        assert_eq!(QraslToken::account_balance(&2), 30);
        assert_eq!(QraslToken::total_supply(), 100); // Total supply should not change
        System::assert_last_event(Event::Transferred { from: 1, to: 2, amount: 30 }.into());
    });
}

#[test]
fn transfer_fails_on_insufficient_balance() {
    new_test_ext().execute_with(|| {
        assert_ok!(QraslToken::mint(Origin::root(), 1, 20));
        assert_noop!(
            QraslToken::transfer(Origin::signed(1), 2, 30),
            Error::<Test>::InsufficientBalance
        );
    });
}

#[test]
fn transfer_fails_for_zero_amount() {
    new_test_ext().execute_with(|| {
        assert_ok!(QraslToken::mint(Origin::root(), 1, 100));
        assert_noop!(
            QraslToken::transfer(Origin::signed(1), 2, 0),
            Error::<Test>::AmountMustBePositive
        );
    });
}

#[test]
fn transfer_to_self_works() {
     new_test_ext().execute_with(|| {
        System::set_block_number(1);
        assert_ok!(QraslToken::mint(Origin::root(), 1, 100));
        assert_ok!(QraslToken::transfer(Origin::signed(1), 1, 30));
        assert_eq!(QraslToken::account_balance(&1), 100); // Balance should not change
        assert_eq!(QraslToken::total_supply(), 100);
        System::assert_last_event(Event::Transferred { from: 1, to: 1, amount: 30 }.into());
    });
}

#[test]
fn burn_works() {
    new_test_ext().execute_with(|| {
        System::set_block_number(1);
        assert_ok!(QraslToken::mint(Origin::root(), 1, 100));
        assert_ok!(QraslToken::burn(Origin::signed(1), 40));
        assert_eq!(QraslToken::account_balance(&1), 60);
        assert_eq!(QraslToken::total_supply(), 60);
        System::assert_last_event(Event::Burned { who: 1, amount: 40 }.into());
    });
}

#[test]
fn burn_fails_on_insufficient_balance() {
    new_test_ext().execute_with(|| {
        assert_ok!(QraslToken::mint(Origin::root(), 1, 30));
        assert_noop!(
            QraslToken::burn(Origin::signed(1), 40),
            Error::<Test>::InsufficientBalance
        );
    });
}

#[test]
fn burn_fails_for_zero_amount() {
    new_test_ext().execute_with(|| {
        assert_ok!(QraslToken::mint(Origin::root(), 1, 100));
        assert_noop!(
            QraslToken::burn(Origin::signed(1), 0),
            Error::<Test>::AmountMustBePositive
        );
    });
}

#[test]
fn burn_all_tokens_works() {
    new_test_ext().execute_with(|| {
        System::set_block_number(1);
        assert_ok!(QraslToken::mint(Origin::root(), 1, 100));
        assert_ok!(QraslToken::burn(Origin::signed(1), 100));
        assert_eq!(QraslToken::account_balance(&1), 0);
        assert_eq!(QraslToken::total_supply(), 0);
        System::assert_last_event(Event::Burned { who: 1, amount: 100 }.into());
    });
}

#[test]
fn burn_fails_on_total_supply_underflow_if_burning_more_than_total_supply_from_account_with_enough_balance() {
    // This scenario is tricky. If an account has X tokens, and total supply is X,
    // burning X from the account is fine.
    // If an account has X tokens, and total supply is Y < X (due to some inconsistency or direct storage manipulation not possible via extrinsics),
    // then burning X from account would make total_supply negative.
    // Our current burn logic subtracts from account first, then total_supply.
    // The check `who_balance >= amount` ensures we don't burn more than the account has.
    // The check `current_total_supply.checked_sub(&amount)` ensures total supply doesn't underflow.
    // So, this specific underflow on total_supply while account balance is sufficient should be prevented
    // by the InsufficientBalance check on the *account* if amount > who_balance,
    // or by the total_supply underflow check if who_balance >= amount but amount > total_supply.
    // Let's test the latter.
    new_test_ext().execute_with(|| {
        // Directly manipulate storage to create inconsistent state for testing underflow
        Balances::<Test>::insert(1, 200); // Account 1 has 200
        TotalSupply::<Test>::put(100);     // Total supply is only 100

        // Attempt to burn 150 from account 1. Account has enough.
        // But this would make total supply negative.
        assert_noop!(
            QraslToken::burn(Origin::signed(1), 150),
            Error::<Test>::Underflow // Should fail on total supply underflow
        );
    });
}
