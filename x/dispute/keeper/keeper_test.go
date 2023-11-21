package keeper_test

import (
	"context"
	"testing"
	"time"

	tmdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/mocks"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
)

var (
	PrivKey cryptotypes.PrivKey
	PubKey  cryptotypes.PubKey
	Addr    sdk.AccAddress
)

type KeeperTestSuite struct {
	suite.Suite

	ctx           sdk.Context
	goCtx         context.Context
	disputeKeeper keeper.Keeper

	accountKeeper  *mocks.AccountKeeper
	bankKeeper     *mocks.BankKeeper
	oracleKeeper   *mocks.OracleKeeper
	slashingKeeper *mocks.SlashingKeeper
	stakingKeeper  *mocks.StakingKeeper

	queryClient types.QueryClient
	msgServer   types.MsgServer
}

func (s *KeeperTestSuite) SetupTest() {
	require := s.Require()
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	paramsSubspace := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"DisputeParams",
	)
	s.accountKeeper = mocks.NewAccountKeeper(s.T())
	s.bankKeeper = mocks.NewBankKeeper(s.T())
	s.oracleKeeper = mocks.NewOracleKeeper(s.T())
	s.slashingKeeper = mocks.NewSlashingKeeper(s.T())
	s.stakingKeeper = mocks.NewStakingKeeper(s.T())

	s.disputeKeeper = *keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,
		s.accountKeeper,
		s.bankKeeper,
		s.oracleKeeper,
		s.slashingKeeper,
		s.stakingKeeper,
	)

	s.ctx = sdk.NewContext(stateStore, tmproto.Header{Time: time.Now()}, false, log.NewNopLogger())
	s.goCtx = sdk.WrapSDKContext(s.ctx)
	// Initialize params
	s.disputeKeeper.SetParams(s.ctx, types.DefaultParams())

	accountPubKeyPrefix := app.AccountAddressPrefix + "pub"
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(app.AccountAddressPrefix, accountPubKeyPrefix)
	s.msgServer = keeper.NewMsgServerImpl(s.disputeKeeper)
	KeyTestPubAddr()
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func KeyTestPubAddr() {
	PrivKey = secp256k1.GenPrivKey()
	PubKey = PrivKey.PubKey()
	Addr = sdk.AccAddress(PubKey.Address())
}
