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

	cycleList, err := okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(cycleList, ethQueryData))

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 2 - eth in cycle list
	//-------------------------------------------------
	s.Equal(int64(2), ctx.BlockHeight())

	cycleList, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(cycleList, ethQueryData))

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2)
	s.NoError(err)
	//-------------------------------------------------
	// block 3 - last block for eth in cycle list
	//-------------------------------------------------
	// assert height is 3
	s.Equal(int64(3), ctx.BlockHeight())

	// get query
	query, err := okpr.CurrentQuery(ctx, utils.QueryIDFromData(ethQueryData))
	s.NoError(err)
	s.Equal(math.ZeroInt(), query.Amount)
	fmt.Println("query.Expiration", query.Expiration)
	s.True(query.Expiration > uint64(ctx.BlockHeight()))
	expirationBeforeTip := query.Expiration

	// tip
	tipmsg, err := oserver.Tip(ctx, &types.MsgTip{Tipper: tipper.String(), QueryData: ethQueryData, Amount: sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000_000))})
	s.NotNil(tipmsg)
	s.NoError(err)

	// get query
	query, err = okpr.CurrentQuery(ctx, utils.QueryIDFromData(btcQueryData))
	s.NoError(err)
	s.Equal(math.NewInt(980_000), query.Amount)          // 2% burn
	s.True(query.Expiration > uint64(ctx.BlockHeight())) // not expired
	s.Equal(expirationBeforeTip, query.Expiration)

	// ----------------
	// next block
	ctx, err = simtestutil.NextBlock(app, ctx, ((time.Second * 15) + 1))
	s.NoError(err)
	// assert height is 4
	s.Equal(int64(4), ctx.BlockHeight())

	// next block
	ctx, err = simtestutil.NextBlock(app, ctx, ((time.Second * 15) + 1)) // trb query data
	s.NoError(err)
	// assert height is 5
	s.Equal(int64(5), ctx.BlockHeight())
	// check query
	query, err = okpr.CurrentQuery(ctx, utils.QueryIDFromData(btcQueryData))
	s.NoError(err)
	s.Equal(math.NewInt(980_000), query.Amount) // 2% burn
	// s.True(query.Expiration.Before(ctx.BlockTime())) // expired
	currentCycleListQuery, err := okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, trbQueryData))

	// next block
	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 15)) // trb query data
	s.NoError(err)
	// assert height is 6
	s.Equal(int64(6), ctx.BlockHeight())
	// // ----------------
	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 1)) // btc query data
	s.NoError(err)
	// assert height is 7
	s.Equal(int64(7), ctx.BlockHeight())

	current, err := okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(current, btcQueryData))
	// check query
	query, err = okpr.CurrentQuery(ctx, utils.QueryIDFromData(btcQueryData))
	s.NoError(err)
	s.Equal(math.NewInt(980_000), query.Amount) // 2% burn
	// s.True(query.Expiration.After(ctx.BlockTime())) // non-expired commit time window
}
