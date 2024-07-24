package app_test

import (
	"testing"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/suite"

	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/app/mocks"
	"github.com/tellor-io/layer/app/testutils"
)

type VoteExtensionTestSuite struct {
	suite.Suite
	ctx sdk.Context
}

func (s *VoteExtensionTestSuite) SetupTest() {
	s.ctx = testutils.CreateTestContext(s.T())
}

func (s *VoteExtensionTestSuite) TestVoteExtensionTestSuite(t *testing.T) {
	suite.Run(t, new(VoteExtensionTestSuite))
}

func (s *VoteExtensionTestSuite) TestNewVoteExtHandler() *app.VoteExtHandler {
	require := s.Require()
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	h := app.NewVoteExtHandler(
		log.NewNopLogger(),
		cdc,
		mocks.NewOracleKeeper(s.T()),
		mocks.NewBridgeKeeper(s.T()),
	)
	require.NotNil(h)
	return h
}

func (s *VoteExtensionTestSuite) TestExtendVoteHandler() {
	require := s.Require()
	h := s.TestNewVoteExtHandler()
	require.NotNil(h)

	res, err := h.ExtendVoteHandler(s.ctx, &abci.RequestExtendVote{})
	require.NoError(err)
	require.NotNil(res)
	require.Equal(res, &abci.ResponseExtendVote{})
}

// func SetupTest(tb testing.TB) (sdk.Context, *codec.ProtoCodec, *mocks.OracleKeeper, *mocks.BridgeKeeper, log.Logger) {
// 	tb.Helper()

// 	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
// 	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

// 	db := cosmosdb.NewMemDB()
// 	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
// 	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
// 	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
// 	require.NoError(tb, stateStore.LoadLatestVersion())

// 	registry := codectypes.NewInterfaceRegistry()
// 	cdc := codec.NewProtoCodec(registry)

// 	ok := mocks.NewOracleKeeper(tb)
// 	bk := mocks.NewBridgeKeeper(tb)
// 	logger := log.NewNopLogger()

// 	ctx := sdk.NewContext(nil, tmproto.Header{}, false, logger)

// 	return ctx, cdc, ok, bk, logger
// }

// func TestNewVoteExtHandler(t *testing.T) {
// 	require := require.New(t)

// 	_, appCodec, ok, bk, logger := SetupTest(t)
// 	voteExtHandler := NewVoteExtHandler(logger, appCodec, ok, bk)
// 	require.NotNil(voteExtHandler)
// 	require.Equal(voteExtHandler.bridgeKeeper, bk)
// 	require.Equal(voteExtHandler.oracleKeeper, ok)
// 	require.Equal(voteExtHandler.codec, appCodec)
// 	require.Equal(voteExtHandler.logger, logger)
// }

// func TestExtendVoteHandler(t *testing.T) {
// 	require := require.New(t)

// 	ctx, appCodec, ok, bk, logger := SetupTest(t)
// 	h := NewVoteExtHandler(logger, appCodec, ok, bk)

// 	req := abci.RequestExtendVote{
// 		Hash: []byte("test"),
// 	}
// 	res, err := h.ExtendVoteHandler(ctx, &req)
// 	require.NoError(err)
// 	require.Equal(res, &abci.ResponseExtendVote{})
// }

// func TestGetOperatorAddress(t *testing.T) {
// 	require := require.New(t)

// 	_, appCodec, ok, bk, logger := SetupTest(t)
// 	h := NewVoteExtHandler(logger, appCodec, ok, bk)

// 	addr, err := h.GetOperatorAddress()
// 	require.Error(err)
// 	require.Equal(addr, "")
// }
