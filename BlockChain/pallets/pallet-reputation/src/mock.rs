use crate as pallet_reputation;
use frame_support::traits::{ConstU16, ConstU64, ConstI64};
use frame_system as system;
use sp_core::H256;
use sp_runtime::{
    testing::Header,
    traits::{BlakeTwo256, IdentityLookup},
};
use frame_support::parameter_types;

type UncheckedExtrinsic = frame_system::mocking::MockUncheckedExtrinsic<Test>;
type Block = frame_system::mocking::MockBlock<Test>;

// Configure a mock runtime to test the pallet.
frame_support::construct_runtime!(
    pub enum Test where
        Block = Block,
        NodeBlock = Block,
        UncheckedExtrinsic = UncheckedExtrinsic,
    {
        System: frame_system::{Pallet, Call, Config, Storage, Event<T>},
        Reputation: pallet_reputation::{Pallet, Call, Storage, Event<T>},
    }
);

impl system::Config for Test {
    type BaseCallFilter = frame_support::traits::Everything;
    type BlockWeights = ();
    type BlockLength = ();
    type DbWeight = ();
    type Origin = Origin;
    type Call = Call;
    type Index = u64;
    type BlockNumber = u64;
    type Hash = H256;
    type Hashing = BlakeTwo256;
    type AccountId = u64;
    type Lookup = IdentityLookup<Self::AccountId>;
    type Header = Header;
    type Event = Event;
    type BlockHashCount = ConstU64<250>;
    type Version = ();
    type PalletInfo = PalletInfo;
    type AccountData = ();
    type OnNewAccount = ();
    type OnKilledAccount = ();
    type SystemWeightInfo = ();
    type SS58Prefix = ConstU16<42>;
    type OnSetCode = ();
    type MaxConsumers = frame_support::traits::ConstU32<16>;
}

parameter_types! {
    pub const MaxRepScore: ReputationScore = 1000;
    pub const MinRepScore: ReputationScore = -100;
    pub const DefaultRepScore: ReputationScore = 100;
}

impl pallet_reputation::Config for Test {
    type Event = Event;
    type MaxReputationScore = MaxRepScore;
    type MinReputationScore = MinRepScore;
    type DefaultReputationScore = DefaultRepScore;
    type ReputationAdminOrigin = frame_system::EnsureSignedBy<ConstU64<0>, Self::AccountId>; // Account 0 is admin
    type WeightInfo = crate::weights::SubstrateWeight<Test>; // Use actual weights if defined
}

// Build genesis storage according to the mock runtime.
pub fn new_test_ext() -> sp_io::TestExternalities {
    let mut t = system::GenesisConfig::default().build_storage::<Test>().unwrap();
    // We don't need to configure genesis for pallet_reputation itself unless we want to set initial scores,
    // as ValueQuery with GetDefaultReputationScore handles defaults.
    let mut ext = sp_io::TestExternalities::new(t);
    ext.execute_with(|| System::set_block_number(1));
    ext
}

// Type alias for tests
pub type ReputationScore = i64;
pub type ReasonCode = u8;
