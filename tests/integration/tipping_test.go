package integration_test

import (
	"bytes"
	"fmt"
	"time"

	"github.com/tellor-io/layer/testutil"
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
	s.NoError(s.Setup.Bankkeeper.MintCoins(ctx, "mint", sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, math.NewInt(1000000000000)))))
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(ctx, "mint", tipper, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, math.NewInt(1000000000000)))))
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

	// tip BTC in block 1 so that it expires at the same height where cycle list is rotated to BTC but has no reports
	tipmsg, err := oserver.Tip(ctx, &types.MsgTip{Tipper: tipper.String(), QueryData: btcQueryData, Amount: sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000_000))})
	s.NotNil(tipmsg)
	s.NoError(err)

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
	// queryId = utils.QueryIDFromData(currentCycleListQuery)

	// get query
	query, err := okpr.CurrentQuery(ctx, utils.QueryIDFromData(ethQueryData))
	s.NoError(err)
	s.Equal(math.ZeroInt(), query.Amount)
	// fmt.Println("query.Expiration", query.Expiration)
	s.True(query.Expiration > uint64(ctx.BlockHeight()))
	// expirationBeforeTip := query.Expiration

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
	// queryId = utils.QueryIDFromData(currentCycleListQuery)

	// tip TRB in block 4 so that it has a query.Expiration of 6 which should make TRB only be in the cycle list for 1 block
	tipmsg, err = oserver.Tip(ctx, &types.MsgTip{Tipper: tipper.String(), QueryData: trbQueryData, Amount: sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000_000))})
	s.NotNil(tipmsg)
	s.NoError(err)

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // trb query data
	s.NoError(err)
	//-------------------------------------------------
	// block 5 - final block for btc, tip
	//-------------------------------------------------
	// assert height is 5
	s.Equal(int64(5), ctx.BlockHeight())

	// // get query, tipping while still active does not extend expiration
	query, err = okpr.CurrentQuery(ctx, utils.QueryIDFromData(trbQueryData))
	s.NoError(err)
	s.Equal(math.NewInt(980_000), query.Amount) // 2% burn
	s.Equal(query.Expiration, uint64(6))

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, btcQueryData))
	// queryId = utils.QueryIDFromData(currentCycleListQuery)

	// get query after tipping, tipping while still in cycle list does not extend expiration
	query, err = okpr.CurrentQuery(ctx, utils.QueryIDFromData(btcQueryData))
	s.NoError(err)
	s.Equal(math.NewInt(980_000), query.Amount) // 2% burn

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2))
	s.NoError(err)
	//-------------------------------------------------
	// block 6 - first block for trb for cycle list but should be when it expires due to the tip at block 4
	//-------------------------------------------------
	s.Equal(int64(6), ctx.BlockHeight())
	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, trbQueryData))
	// queryId = utils.QueryIDFromData(currentCycleListQuery)

	// tip for btc so that it creates a query with expiration of 8 when btc should be rotated into the cycle list
	tipmsg, err = oserver.Tip(ctx, &types.MsgTip{Tipper: tipper.String(), QueryData: btcQueryData, Amount: sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000_000))})
	s.NotNil(tipmsg)
	s.NoError(err)

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2))
	s.NoError(err)
	//-------------------------------------------------
	// block 7 - back to eth
	//-------------------------------------------------
	s.Equal(int64(7), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, ethQueryData))
	// queryId = utils.QueryIDFromData(currentCycleListQuery)

	// report for btc so that it is aggregated before the cycle list is rotated
	_, err = oserver.SubmitValue(ctx, &types.MsgSubmitValue{Creator: repAccs[0].String(), QueryData: btcQueryData, Value: testutil.EncodeValue(462926)})
	s.NoError(err)

	btcQueryId := utils.QueryIDFromData(btcQueryData)
	tippedBTCQueryMeta, err := okpr.CurrentQuery(ctx, btcQueryId)
	s.NoError(err)
	fmt.Println(tippedBTCQueryMeta)

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // trb query data
	s.NoError(err)
	//-------------------------------------------------
	// block 8 - eth last block
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
	// block 9 - btc first block
	//-------------------------------------------------
	s.Equal(int64(9), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	// queryId = utils.QueryIDFromData(currentCycleListQuery)
	s.True(bytes.Equal(currentCycleListQuery, btcQueryData))

	cycleListQueryMeta, err := okpr.CurrentQuery(ctx, btcQueryId)
	s.NoError(err)
	fmt.Println(cycleListQueryMeta)
	s.NotEqual(tippedBTCQueryMeta.Id, cycleListQueryMeta.Id)

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // btc query data
	s.NoError(err)
	//-------------------------------------------------
	// block 10 - final block for btc
	//-------------------------------------------------
	s.Equal(int64(10), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, btcQueryData))
	// queryId = utils.QueryIDFromData(currentCycleListQuery)

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // btc query data
	s.NoError(err)
	//-------------------------------------------------
	// block 11 - first block for trb
	//-------------------------------------------------
	s.Equal(int64(11), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, trbQueryData))
	// queryId = utils.QueryIDFromData(currentCycleListQuery)

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 12 - final block for trb
	//-------------------------------------------------
	s.Equal(int64(12), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, trbQueryData))
	// queryId = utils.QueryIDFromData(currentCycleListQuery)

	// ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	// s.NoError(err)
	// //-------------------------------------------------
	// // block 11 - first block for eth
	// //-------------------------------------------------
	// s.Equal(int64(13), ctx.BlockHeight())

	// currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	// s.NoError(err)
	// s.True(bytes.Equal(currentCycleListQuery, ethQueryData))
	// // queryId = utils.QueryIDFromData(currentCycleListQuery)

	// ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	// s.NoError(err)
	// //-------------------------------------------------
	// // block 11 - final block for eth
	// //-------------------------------------------------
	// s.Equal(int64(14), ctx.BlockHeight())

	// currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	// s.NoError(err)
	// // s.True(bytes.Equal(currentCycleListQuery, btcQueryData))
	// // queryId = utils.QueryIDFromData(currentCycleListQuery)

	// ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	// s.NoError(err)
	// //-------------------------------------------------
	// // block 11 - final block for btc
	// //-------------------------------------------------
	// s.Equal(int64(15), ctx.BlockHeight())

	// currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	// s.NoError(err)
	// // s.True(bytes.Equal(currentCycleListQuery, btcQueryData))
	// // queryId = utils.QueryIDFromData(currentCycleListQuery)

}
