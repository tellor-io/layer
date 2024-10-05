package integration_test

import (
	"bytes"
	"fmt"
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
	queryId := utils.QueryIDFromData(currentCycleListQuery)
	fmt.Println("currentHeight", ctx.BlockHeight())
	fmt.Println("queryId", queryId)

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 2 - eth in cycle list
	//-------------------------------------------------
	s.Equal(int64(2), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, ethQueryData))
	queryId = utils.QueryIDFromData(currentCycleListQuery)
	fmt.Println("currentHeight", ctx.BlockHeight())
	fmt.Println("queryId", queryId)

	// get query
	query, err := okpr.CurrentQuery(ctx, utils.QueryIDFromData(ethQueryData))
	s.NoError(err)
	s.Equal(math.ZeroInt(), query.Amount)
	// fmt.Println("query.Expiration", query.Expiration)
	s.True(query.Expiration > uint64(ctx.BlockHeight()))
	// expirationBeforeTip := query.Expiration

	// tip
	tipmsg, err := oserver.Tip(ctx, &types.MsgTip{Tipper: tipper.String(), QueryData: ethQueryData, Amount: sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000_000))})
	s.NotNil(tipmsg)
	s.NoError(err)

	// // get query, tipping while still active does not extend expiration
	// query, err = okpr.CurrentQuery(ctx, utils.QueryIDFromData(ethQueryData))
	// s.NoError(err)
	// s.Equal(math.NewInt(980_000), query.Amount)    // 2% burn
	// s.Equal(expirationBeforeTip, query.Expiration) //  expired

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2)
	s.NoError(err)
	//-------------------------------------------------
	// block 3 - eth still in cycle list
	//-------------------------------------------------
	// assert height is 3
	s.Equal(int64(3), ctx.BlockHeight())
	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, ethQueryData))
	queryId = utils.QueryIDFromData(currentCycleListQuery)
	fmt.Println("currentHeight", ctx.BlockHeight())
	fmt.Println("queryId", queryId)

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2)
	s.NoError(err)
	//-------------------------------------------------
	// block 4 - first block for btc, rotate at end of 6
	//-------------------------------------------------
	s.Equal(int64(4), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, btcQueryData))
	fmt.Println("currentHeight", ctx.BlockHeight())
	queryId = utils.QueryIDFromData(currentCycleListQuery)
	fmt.Println("queryId", queryId)

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // trb query data
	s.NoError(err)
	//-------------------------------------------------
	// block 5 - final block for btc
	//-------------------------------------------------
	// assert height is 5
	s.Equal(int64(5), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, btcQueryData))
	fmt.Println("currentHeight", ctx.BlockHeight())
	queryId = utils.QueryIDFromData(currentCycleListQuery)
	fmt.Println("queryId", queryId)

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // trb query data
	s.NoError(err)
	//-------------------------------------------------
	// block 6 - first block for trb
	//-------------------------------------------------
	s.Equal(int64(6), ctx.BlockHeight())
	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, trbQueryData))
	fmt.Println("currentHeight", ctx.BlockHeight())
	queryId = utils.QueryIDFromData(currentCycleListQuery)
	fmt.Println("queryId", queryId)

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // trb query data
	s.NoError(err)
	//-------------------------------------------------
	// block 7 - final block for trb
	//-------------------------------------------------
	s.Equal(int64(7), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, trbQueryData))
	fmt.Println("currentHeight", ctx.BlockHeight())
	queryId = utils.QueryIDFromData(currentCycleListQuery)
	fmt.Println("queryId", queryId)

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // trb query data
	s.NoError(err)
	//-------------------------------------------------
	// block 8 - back to eth
	//-------------------------------------------------
	s.Equal(int64(8), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, ethQueryData))
	fmt.Println("currentHeight", ctx.BlockHeight())
	queryId = utils.QueryIDFromData(currentCycleListQuery)
	fmt.Println("queryId", queryId)

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // trb query data
	s.NoError(err)
	//-------------------------------------------------
	// block 9 - final block for eth
	//-------------------------------------------------
	s.Equal(int64(9), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	fmt.Println("currentHeight", ctx.BlockHeight())
	queryId = utils.QueryIDFromData(currentCycleListQuery)
	fmt.Println("queryId", queryId)
	// s.True(bytes.Equal(currentCycleListQuery, ethQueryData))

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // btc query data
	s.NoError(err)
	//-------------------------------------------------
	// block 10 - first block for btc
	//-------------------------------------------------
	s.Equal(int64(10), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	// s.True(bytes.Equal(currentCycleListQuery, btcQueryData))
	fmt.Println("currentHeight", ctx.BlockHeight())
	queryId = utils.QueryIDFromData(currentCycleListQuery)
	fmt.Println("queryId", queryId)

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // btc query data
	s.NoError(err)
	//-------------------------------------------------
	// block 11 - final block for btc
	//-------------------------------------------------
	s.Equal(int64(11), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	// s.True(bytes.Equal(currentCycleListQuery, btcQueryData))
	fmt.Println("currentHeight", ctx.BlockHeight())
	queryId = utils.QueryIDFromData(currentCycleListQuery)
	fmt.Println("queryId", queryId)

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 12 - final block for btc
	//-------------------------------------------------
	s.Equal(int64(12), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	// s.True(bytes.Equal(currentCycleListQuery, btcQueryData))
	fmt.Println("currentHeight", ctx.BlockHeight())
	queryId = utils.QueryIDFromData(currentCycleListQuery)
	fmt.Println("queryId", queryId)

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 11 - final block for btc
	//-------------------------------------------------
	s.Equal(int64(13), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	// s.True(bytes.Equal(currentCycleListQuery, btcQueryData))
	fmt.Println("currentHeight", ctx.BlockHeight())
	queryId = utils.QueryIDFromData(currentCycleListQuery)
	fmt.Println("queryId", queryId)

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 11 - final block for btc
	//-------------------------------------------------
	s.Equal(int64(14), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	// s.True(bytes.Equal(currentCycleListQuery, btcQueryData))
	fmt.Println("currentHeight", ctx.BlockHeight())
	queryId = utils.QueryIDFromData(currentCycleListQuery)
	fmt.Println("queryId", queryId)

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 11 - final block for btc
	//-------------------------------------------------
	s.Equal(int64(15), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	// s.True(bytes.Equal(currentCycleListQuery, btcQueryData))
	fmt.Println("currentHeight", ctx.BlockHeight())
	queryId = utils.QueryIDFromData(currentCycleListQuery)
	fmt.Println("queryId", queryId)
}
