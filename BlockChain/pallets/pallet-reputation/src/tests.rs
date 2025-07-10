use crate::{mock::*, Error, Event, ReputationScore, ReasonCode};
use frame_support::{assert_noop, assert_ok};
use sp_runtime::DispatchError;

const ADMIN_ORIGIN: u64 = 0; // Assuming Root or a specific admin account for tests
const USER_A: u64 = 1;
const USER_B: u64 = 2;
const DEFAULT_SCORE: ReputationScore = 100;
const MAX_SCORE: ReputationScore = 1000;
const MIN_SCORE: ReputationScore = -100;

const REASON_TEST: ReasonCode = 1;

#[test]
fn initial_score_is_default() {
    new_test_ext().execute_with(|| {
        assert_eq!(Reputation::reputation_scores(&USER_A), DEFAULT_SCORE);
    });
}

#[test]
fn initialize_reputation_works_and_emits_event() {
    new_test_ext().execute_with(|| {
        System::set_block_number(1);
        // Set a non-default score first to see if initialize resets it
        ReputationScoresMap::<Test>::insert(&USER_A, 500);
        assert_eq!(Reputation::reputation_scores(&USER_A), 500);

        assert_ok!(Reputation::initialize_reputation(Origin::signed(ADMIN_ORIGIN), USER_A));
        assert_eq!(Reputation::reputation_scores(&USER_A), DEFAULT_SCORE);
        System::assert_last_event(Event::ReputationInitialized { who: USER_A, initial_score: DEFAULT_SCORE }.into());
    });
}

#[test]
fn update_reputation_works_positive_change() {
    new_test_ext().execute_with(|| {
        System::set_block_number(1);
        assert_ok!(Reputation::update_reputation(Origin::signed(ADMIN_ORIGIN), USER_A, 50, REASON_TEST));
        assert_eq!(Reputation::reputation_scores(&USER_A), DEFAULT_SCORE + 50);
        System::assert_last_event(Event::ReputationUpdated { who: USER_A, new_score: DEFAULT_SCORE + 50, change: 50, reason: REASON_TEST }.into());
    });
}

#[test]
fn update_reputation_works_negative_change() {
    new_test_ext().execute_with(|| {
        System::set_block_number(1);
        ReputationScoresMap::<Test>::insert(&USER_A, 200); // Start with a higher score
        assert_ok!(Reputation::update_reputation(Origin::signed(ADMIN_ORIGIN), USER_A, -50, REASON_TEST));
        assert_eq!(Reputation::reputation_scores(&USER_A), 150);
        System::assert_last_event(Event::ReputationUpdated { who: USER_A, new_score: 150, change: -50, reason: REASON_TEST }.into());
    });
}

#[test]
fn update_reputation_fails_for_non_admin_origin() {
    new_test_ext().execute_with(|| {
        assert_noop!(
            Reputation::update_reputation(Origin::signed(USER_B), USER_A, 50, REASON_TEST),
            DispatchError::BadOrigin // Or specific error if EnsureOrigin is more complex
        );
    });
}

#[test]
fn update_reputation_clamps_to_max_score() {
    new_test_ext().execute_with(|| {
        System::set_block_number(1);
        ReputationScoresMap::<Test>::insert(&USER_A, MAX_SCORE - 10);
        assert_ok!(Reputation::update_reputation(Origin::signed(ADMIN_ORIGIN), USER_A, 50, REASON_TEST)); // Tries to go over max
        assert_eq!(Reputation::reputation_scores(&USER_A), MAX_SCORE);
        System::assert_last_event(Event::ReputationUpdated { who: USER_A, new_score: MAX_SCORE, change: 50, reason: REASON_TEST }.into());
    });
}

#[test]
fn update_reputation_clamps_to_min_score() {
    new_test_ext().execute_with(|| {
        System::set_block_number(1);
        ReputationScoresMap::<Test>::insert(&USER_A, MIN_SCORE + 10);
        assert_ok!(Reputation::update_reputation(Origin::signed(ADMIN_ORIGIN), USER_A, -50, REASON_TEST)); // Tries to go below min
        assert_eq!(Reputation::reputation_scores(&USER_A), MIN_SCORE);
        System::assert_last_event(Event::ReputationUpdated { who: USER_A, new_score: MIN_SCORE, change: -50, reason: REASON_TEST }.into());
    });
}

#[test]
fn update_reputation_with_zero_change_works() {
    new_test_ext().execute_with(|| {
        System::set_block_number(1);
        assert_ok!(Reputation::update_reputation(Origin::signed(ADMIN_ORIGIN), USER_A, 0, REASON_TEST));
        assert_eq!(Reputation::reputation_scores(&USER_A), DEFAULT_SCORE);
        System::assert_last_event(Event::ReputationUpdated { who: USER_A, new_score: DEFAULT_SCORE, change: 0, reason: REASON_TEST }.into());
    });
}


#[test]
fn reset_reputation_works() {
    new_test_ext().execute_with(|| {
        System::set_block_number(1);
        ReputationScoresMap::<Test>::insert(&USER_A, 500); // Set a non-default score
        assert_ok!(Reputation::reset_reputation(Origin::signed(ADMIN_ORIGIN), USER_A));
        assert_eq!(Reputation::reputation_scores(&USER_A), DEFAULT_SCORE);
        System::assert_last_event(Event::ReputationReset { who: USER_A, new_score: DEFAULT_SCORE }.into());
    });
}

#[test]
fn reset_reputation_fails_for_non_admin_origin() {
    new_test_ext().execute_with(|| {
        assert_noop!(
            Reputation::reset_reputation(Origin::signed(USER_B), USER_A),
            DispatchError::BadOrigin
        );
    });
}

#[test]
fn get_score_helper_works() {
    new_test_ext().execute_with(|| {
        assert_eq!(Reputation::get_score(&USER_A), DEFAULT_SCORE);
        ReputationScoresMap::<Test>::insert(&USER_A, 777);
        assert_eq!(Reputation::get_score(&USER_A), 777);
    });
}

#[test]
fn internal_update_score_unsafe_works_and_clamps() {
    new_test_ext().execute_with(|| {
        System::set_block_number(1);
        // Positive update
        assert_ok!(Reputation::_update_score_unsafe(&USER_A, 50, REASON_TEST));
        assert_eq!(Reputation::get_score(&USER_A), DEFAULT_SCORE + 50);
        System::assert_last_event(Event::ReputationUpdated{who: USER_A, new_score: DEFAULT_SCORE + 50, change: 50, reason: REASON_TEST}.into());

        // Negative update
        assert_ok!(Reputation::_update_score_unsafe(&USER_A, -20, REASON_TEST));
        assert_eq!(Reputation::get_score(&USER_A), DEFAULT_SCORE + 30);
        System::assert_last_event(Event::ReputationUpdated{who: USER_A, new_score: DEFAULT_SCORE + 30, change: -20, reason: REASON_TEST}.into());

        // Clamp to max
        assert_ok!(Reputation::_update_score_unsafe(&USER_A, MAX_SCORE * 2, REASON_TEST)); // try to go way over
        assert_eq!(Reputation::get_score(&USER_A), MAX_SCORE);
        System::assert_last_event(Event::ReputationUpdated{who: USER_A, new_score: MAX_SCORE, change: MAX_SCORE * 2, reason: REASON_TEST}.into());

        // Clamp to min
        assert_ok!(Reputation::_update_score_unsafe(&USER_A, MIN_SCORE * 2, REASON_TEST)); // try to go way under
        assert_eq!(Reputation::get_score(&USER_A), MIN_SCORE);
        System::assert_last_event(Event::ReputationUpdated{who: USER_A, new_score: MIN_SCORE, change: MIN_SCORE * 2, reason: REASON_TEST}.into());
    });
}
