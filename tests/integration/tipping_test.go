package integration_test

import (
	"bytes"
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// cycle list order is eth, btc, trb

// Test adding tip to a query that is already in cycle and not expired
func (s *IntegrationTestSuite) TestTipQueryInCycle() {
	ctx := s.Setup.Ctx
	app := s.Setup.App
	okpr := s.Setup.Oraclekeeper

	tipper := sample.AccAddressBytes()
	s.NoError(s.Setup.Bankkeeper.MintCoins(ctx, "mint", sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, math.NewInt(1000000000)))))
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(ctx, "mint", tipper, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, math.NewInt(1000000000)))))
	// setup reporter
	repAccs, valAddrs, _ := s.createValidatorAccs([]uint64{100, 200})
	for _, val := range valAddrs {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(s.Setup.Ctx, val.String(), []byte("not real"))
		s.NoError(err)
	}
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewSelection(repAccs[0], 1)))

	oserver := keeper.NewMsgServerImpl(okpr)
	currentHeight := ctx.BlockHeight()
	// assert height is 0
	s.Equal(int64(0), currentHeight)
	//-------------------------------------------------
	// block 1 - eth in cycle list
	//-------------------------------------------------
	ctx = ctx.WithBlockHeight(1)
	s.Equal(int64(1), ctx.BlockHeight())

	currentCycleListQuery, err := okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, ethQueryData))

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 2 - eth in cycle list
	//-------------------------------------------------
	s.Equal(int64(2), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, ethQueryData))

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2)
	s.NoError(err)
	//-------------------------------------------------
	// block 3 - eth final block in cycle list
	//-------------------------------------------------
	// assert height is 3
	s.Equal(int64(3), ctx.BlockHeight())
	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, ethQueryData))

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2)
	s.NoError(err)
	//-------------------------------------------------
	// block 4 - first block for btc
	//-------------------------------------------------
	s.Equal(int64(4), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, btcQueryData))

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // trb query data
	s.NoError(err)
	//-------------------------------------------------
	// block 5 - final block for btc, tip
	//-------------------------------------------------
	// assert height is 5
	s.Equal(int64(5), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, btcQueryData))

	// get query before tipping
	query, err := okpr.CurrentQuery(ctx, utils.QueryIDFromData(btcQueryData))
	s.NoError(err)
	s.Equal(math.ZeroInt(), query.Amount)
	s.True(query.Expiration == uint64(ctx.BlockHeight()))
	expirationBeforeTip := query.Expiration

	// tip
	tipmsg, err := oserver.Tip(ctx, &types.MsgTip{Tipper: tipper.String(), QueryData: btcQueryData, Amount: sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000_000))})
	s.NotNil(tipmsg)
	s.NoError(err)

	// get query after tipping, tipping while still in cycle list does not extend expiration
	query, err = okpr.CurrentQuery(ctx, utils.QueryIDFromData(btcQueryData))
	s.NoError(err)
	s.Equal(math.NewInt(980_000), query.Amount)    // 2% burn
	s.Equal(expirationBeforeTip, query.Expiration) // expirations should stay the same

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2))
	s.NoError(err)
	//-------------------------------------------------
	// block 6 - first block for trb
	//-------------------------------------------------
	s.Equal(int64(6), ctx.BlockHeight())
	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, trbQueryData))

	// check trb query
	query, err = okpr.CurrentQuery(ctx, utils.QueryIDFromData(trbQueryData))
	s.NoError(err)
	s.Equal(math.ZeroInt(), query.Amount)                  // amount should be zero
	s.Equal(query.Expiration, uint64(ctx.BlockHeight()+1)) // should expire next block

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2))
	s.NoError(err)
	//-------------------------------------------------
	// block 7 - final block for trb
	//-------------------------------------------------
	s.Equal(int64(7), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, trbQueryData))

	// check trb query
	query, err = okpr.CurrentQuery(ctx, utils.QueryIDFromData(trbQueryData))
	s.NoError(err)
	s.Zero(query.Amount)                                 // amount should be zero
	s.Equal(query.Expiration, uint64(ctx.BlockHeight())) // last block

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // trb query data
	s.NoError(err)
	//-------------------------------------------------
	// block 8 - first block for eth
	//-------------------------------------------------
	s.Equal(int64(8), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, ethQueryData))

	// check eth query
	query, err = okpr.CurrentQuery(ctx, utils.QueryIDFromData(ethQueryData))
	s.NoError(err)
	s.Zero(query.Amount)                                   // amount should be zero
	s.Equal(query.Expiration, uint64(ctx.BlockHeight()+1)) // should expire next block

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // trb query data
	s.NoError(err)
	//-------------------------------------------------
	// block 9 - final block for eth
	//-------------------------------------------------
	s.Equal(int64(9), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, ethQueryData))

	// check eth query
	query, err = okpr.CurrentQuery(ctx, utils.QueryIDFromData(ethQueryData))
	s.NoError(err)
	s.Zero(query.Amount)                                 // amount should be zero
	s.Equal(query.Expiration, uint64(ctx.BlockHeight())) // last block

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // btc query data
	s.NoError(err)
	//-------------------------------------------------
	// block 10 - first block for btc, should still have the tip
	//-------------------------------------------------
	s.Equal(int64(10), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, btcQueryData))

	// check btc query
	query, err = okpr.CurrentQuery(ctx, utils.QueryIDFromData(btcQueryData))
	s.NoError(err)
	s.Equal(math.NewInt(980_000), query.Amount)            // amount should still be tip
	s.Equal(query.Expiration, uint64(ctx.BlockHeight()+1)) // should expire next block

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // btc query data
	s.NoError(err)
	//-------------------------------------------------
	// block 11 - final block for btc
	//-------------------------------------------------
	s.Equal(int64(11), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, btcQueryData))

	// check btc query
	query, err = okpr.CurrentQuery(ctx, utils.QueryIDFromData(btcQueryData))
	s.NoError(err)
	s.Equal(math.NewInt(980_000), query.Amount)          // amount should be tip
	s.Equal(query.Expiration, uint64(ctx.BlockHeight())) // should expire this block

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 12 - first block for trb
	//-------------------------------------------------
	s.Equal(int64(12), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, trbQueryData))

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 13 - final block for trb
	//-------------------------------------------------
	s.Equal(int64(13), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, trbQueryData))

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 14 - first block for eth
	//-------------------------------------------------
	s.Equal(int64(14), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, ethQueryData))

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 15 - final block for eth
	//-------------------------------------------------
	s.Equal(int64(15), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, ethQueryData))
}
