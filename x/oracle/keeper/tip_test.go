package keeper_test

import (
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Accounts struct {
	PrivateKey secp256k1.PrivKey
	Account    sdk.AccAddress
}

var (
	TRB_queryId = []byte("0x5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0")
	ETH_queryId = []byte("0x83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
)

func ReturnTestQueryMeta(tip math.Int) types.QueryMeta {
	return types.QueryMeta{
		Id:                    1,
		Amount:                tip,
		Expiration:            time.Now().Add(1 * time.Minute),
		RegistrySpecTimeframe: 1 * time.Minute,
		HasRevealedReports:    false,
		QueryId:               []byte("0x5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0"),
		QueryType:             "SpotPrice",
	}
}

func (s *KeeperTestSuite) TestGetQueryTip() {
	queryMeta := ReturnTestQueryMeta(math.NewInt(1 * 1e6))
	s.NoError(s.oracleKeeper.Query.Set(s.ctx, queryMeta.QueryId, queryMeta))

	// test with a valid queryId
	res, err := s.oracleKeeper.GetQueryTip(s.ctx, queryMeta.QueryId)
	s.NoError(err)
	s.Equal(math.NewInt(1*1e6), res)

	// test with an invalid queryId that should return 0
	res, err = s.oracleKeeper.GetQueryTip(s.ctx, []byte("test"))
	s.NoError(err)
	s.Equal(math.NewInt(0), res)
}

func (s *KeeperTestSuite) TestGetUserTips() {
	acc := sample.AccAddressBytes()

	res, err := s.oracleKeeper.GetUserTips(s.ctx, acc)
	s.NoError(err)
	s.Equal(math.ZeroInt(), res)

	query := ReturnTestQueryMeta(math.NewInt(1 * 1e6))
	s.NoError(s.oracleKeeper.TipperTotal.Set(s.ctx, collections.Join(acc.Bytes(), s.ctx.BlockHeight()), query.Amount))
	res, err = s.oracleKeeper.GetUserTips(s.ctx, acc)
	s.NoError(err)
	s.Equal(math.NewInt(1*1e6), res)

	query.QueryId = ETH_queryId
	query.Id = 2
	// adding the flow here to show how its handled in msgTip
	tipperTotal, err := s.oracleKeeper.TipperTotal.Get(s.ctx, collections.Join(acc.Bytes(), s.ctx.BlockHeight()))
	s.NoError(err)
	query.Amount = tipperTotal.Add(query.Amount)
	s.NoError(s.oracleKeeper.TipperTotal.Set(s.ctx, collections.Join(acc.Bytes(), s.ctx.BlockHeight()), query.Amount))

	res, err = s.oracleKeeper.GetUserTips(s.ctx, acc)
	s.NoError(err)
	s.Equal(math.NewInt(2*1e6), res)
}

func (s *KeeperTestSuite) TestGetTotalTips() {
	res, err := s.oracleKeeper.GetTotalTips(s.ctx)
	s.NoError(err)
	s.Equal(math.ZeroInt(), res)
	s.NoError(s.oracleKeeper.TipperTotal.Set(s.ctx, collections.Join(sample.AccAddressBytes().Bytes(), s.ctx.BlockHeight()), math.NewInt(100*1e6)))
	s.NoError(s.oracleKeeper.TotalTips.Set(s.ctx, s.ctx.BlockHeight(), math.NewInt(100*1e6)))
	res, err = s.oracleKeeper.GetTotalTips(s.ctx)
	s.NoError(err)
	s.Equal(math.NewInt(100*1e6), res)
}

func (s *KeeperTestSuite) TestAddtoTotalTips() {
	s.NoError(s.oracleKeeper.TotalTips.Set(s.ctx, s.ctx.BlockHeight(), math.NewInt(1*1e6)))
	beforeTotalTips, err := s.oracleKeeper.GetTotalTips(s.ctx)
	s.NoError(err)
	s.Equal(math.NewInt(1*1e6), beforeTotalTips)

	err = s.oracleKeeper.AddtoTotalTips(s.ctx, math.NewInt(5*1e6))
	s.NoError(err)

	totalTips, err := s.oracleKeeper.GetTotalTips(s.ctx)
	s.NoError(err)

	s.Equal(math.NewInt(5*1e6).Add(beforeTotalTips), totalTips)
}

func (s *KeeperTestSuite) TestGetTipsAtBlockForTipper() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	// try to get for unrecognized address
	tipperAddr := sample.AccAddressBytes()
	tipperTotal, err := k.TipperTotal.Get(ctx, collections.Join(tipperAddr.Bytes(), ctx.BlockHeight()))
	require.Error(err)
	require.Equal(math.Int{}, tipperTotal)

	// set tipper total and get
	require.NoError(k.TipperTotal.Set(ctx, collections.Join(tipperAddr.Bytes(), ctx.BlockHeight()), math.NewInt(100*1e6)))
	tipperTotal, err = k.GetTipsAtBlockForTipper(ctx, int64(0), tipperAddr)
	require.NoError(err)
	require.Equal(math.NewInt(100*1e6), tipperTotal)

	// set more in the future and get again
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 100)
	require.NoError(k.TipperTotal.Set(ctx, collections.Join(tipperAddr.Bytes(), ctx.BlockHeight()), math.NewInt(200*1e6)))
	tipperTotal, err = k.GetTipsAtBlockForTipper(ctx, int64(101), tipperAddr)
	require.NoError(err)
	require.Equal(math.NewInt(200*1e6), tipperTotal)

	// set less in the future and get again
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 100)
	require.NoError(k.TipperTotal.Set(ctx, collections.Join(tipperAddr.Bytes(), ctx.BlockHeight()), math.NewInt(50*1e6)))
	tipperTotal, err = k.GetTipsAtBlockForTipper(ctx, int64(201), tipperAddr)
	require.NoError(err)
	require.Equal(math.NewInt(50*1e6), tipperTotal)
}

func (s *KeeperTestSuite) TestGetTotalTipsAtBlock() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	tips, err := k.GetTotalTipsAtBlock(ctx, int64(0))
	require.NoError(err)
	require.Equal(math.ZeroInt(), tips)

	// set tips for next block and check again
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	require.NoError(k.TotalTips.Set(ctx, ctx.BlockHeight(), math.NewInt(1*1e6)))
	tips, err = k.GetTotalTipsAtBlock(ctx, int64(1))
	require.NoError(err)
	require.Equal(math.NewInt(1*1e6), tips)

	// check older block
	tips, err = k.GetTotalTipsAtBlock(ctx, int64(0))
	require.NoError(err)
	require.Equal(math.ZeroInt(), tips)
}

func (s *KeeperTestSuite) TestAddToTipperTotal() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	tipper := sample.AccAddressBytes()
	amt := math.NewInt(1 * 1e6)

	require.NoError(k.AddToTipperTotal(ctx, tipper, amt))
	tipperTotal, err := k.TipperTotal.Get(ctx, collections.Join(tipper.Bytes(), ctx.BlockHeight()))
	require.NoError(err)
	require.Equal(amt, tipperTotal)

	// add more
	require.NoError(k.AddToTipperTotal(ctx, tipper, amt))
	tipperTotal, err = k.TipperTotal.Get(ctx, collections.Join(tipper.Bytes(), ctx.BlockHeight()))
	require.NoError(err)
	require.Equal(amt.Add(amt), tipperTotal)

	// add 0
	require.NoError(k.AddToTipperTotal(ctx, tipper, math.ZeroInt()))
	tipperTotal, err = k.TipperTotal.Get(ctx, collections.Join(tipper.Bytes(), ctx.BlockHeight()))
	require.NoError(err)
	require.Equal(amt.Add(amt), tipperTotal)

	// try with bad addr
	require.Error(k.AddToTipperTotal(ctx, []byte("bad"), amt))
	tipperTotal, err = k.TipperTotal.Get(ctx, collections.Join(tipper.Bytes(), ctx.BlockHeight()))
	require.NoError(err)
	require.Equal(amt.Add(amt), tipperTotal)
}