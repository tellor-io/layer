package integration_test

import (
	"bytes"
	"time"

	"github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// cycle list order is eth, btc, trb

// Test adding tip to a query that is already in cycle and not expired
func (s *IntegrationTestSuite) TestTipQueryInCycle() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
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

	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker1")))
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
	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	s.NoError(err)
	//-------------------------------------------------
	// block 2 - eth in cycle list
	//-------------------------------------------------
	s.Equal(int64(2), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, ethQueryData))

	// get query
	query, err := okpr.CurrentQuery(ctx, utils.QueryIDFromData(ethQueryData))
	s.NoError(err)
	s.Equal(math.ZeroInt(), query.Amount)
	s.True(query.Expiration > uint64(ctx.BlockHeight()))

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2)
	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
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
	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	s.NoError(err)
	//-------------------------------------------------
	// block 4 - first block for btc
	//-------------------------------------------------
	s.Equal(int64(4), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, btcQueryData))

	// tip TRB in block 4 so that it has a query.Expiration of 6 which should make TRB only be in the cycle list for 1 block
	tipmsg, err = oserver.Tip(ctx, &types.MsgTip{Tipper: tipper.String(), QueryData: trbQueryData, Amount: sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000_000))})
	s.NotNil(tipmsg)
	s.NoError(err)

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // trb query data
	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
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

	// get query after tipping, tipping while still in cycle list does not extend expiration
	query, err = okpr.CurrentQuery(ctx, utils.QueryIDFromData(btcQueryData))
	s.NoError(err)
	s.Equal(math.NewInt(980_000), query.Amount) // 2% burn

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2))
	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	s.NoError(err)
	//-------------------------------------------------
	// block 6 - first block for trb for cycle list but should be when it expires due to the tip at block 4
	//-------------------------------------------------
	s.Equal(int64(6), ctx.BlockHeight())
	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, trbQueryData))

	// tip for btc so that it creates a query with expiration of 8 when btc should be rotated into the cycle list
	tipmsg, err = oserver.Tip(ctx, &types.MsgTip{Tipper: tipper.String(), QueryData: btcQueryData, Amount: sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000_000))})
	s.NotNil(tipmsg)
	s.NoError(err)

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2))
	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	s.NoError(err)
	//-------------------------------------------------
	// block 7 - back to eth
	//-------------------------------------------------
	s.Equal(int64(7), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, ethQueryData))

	// report for btc so that it is aggregated before the cycle list is rotated
	_, err = oserver.SubmitValue(ctx, &types.MsgSubmitValue{Creator: repAccs[0].String(), QueryData: btcQueryData, Value: testutil.EncodeValue(462926)})
	s.NoError(err)

	btcQueryId := utils.QueryIDFromData(btcQueryData)
	tippedBTCQueryMeta, err := okpr.CurrentQuery(ctx, btcQueryId)
	s.NoError(err)

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // trb query data
	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
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
	s.Equal(query.Amount, math.NewInt(0))                // amount should be zero
	s.Equal(query.Expiration, uint64(ctx.BlockHeight())) // should expire next block

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // trb query data
	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	s.NoError(err)
	//-------------------------------------------------
	// block 9 - btc first block
	//-------------------------------------------------
	s.Equal(int64(9), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, btcQueryData))

	cycleListQueryMeta, err := okpr.CurrentQuery(ctx, btcQueryId)
	s.NoError(err)
	s.NotEqual(tippedBTCQueryMeta.Id, cycleListQueryMeta.Id)

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // btc query data
	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	s.NoError(err)
	//-------------------------------------------------
	// block 10 - final block for btc
	//-------------------------------------------------
	s.Equal(int64(10), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, btcQueryData))

	ctx, err = simtestutil.NextBlock(app, ctx, (time.Second * 2)) // btc query data
	s.NoError(err)
	//-------------------------------------------------
	// block 11 - first block for trb
	//-------------------------------------------------
	s.Equal(int64(11), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, trbQueryData))

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 12 - final block for trb
	//-------------------------------------------------
	s.Equal(int64(12), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, trbQueryData))

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 11 - first block for eth
	//-------------------------------------------------
	s.Equal(int64(13), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, ethQueryData))

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 11 - final block for eth
	//-------------------------------------------------
	s.Equal(int64(14), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, ethQueryData))

	ctx, err = simtestutil.NextBlock(app, ctx, time.Second*2) // endblock1/beginblock2
	s.NoError(err)
	//-------------------------------------------------
	// block 11 - final block for btc
	//-------------------------------------------------
	s.Equal(int64(15), ctx.BlockHeight())

	currentCycleListQuery, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(currentCycleListQuery, btcQueryData))
}

// test tipping an expiring query
func (s *IntegrationTestSuite) TestTippingQuery() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	ctx := s.Setup.Ctx
	app := s.Setup.App
	okpr := s.Setup.Oraclekeeper
	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	msgServer := keeper.NewMsgServerImpl(okpr)
	repAccs, valAddrs, _ := s.createValidatorAccs([]uint64{100})
	for _, val := range valAddrs {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(ctx, val.String(), []byte("not real"))
		s.NoError(err)
	}
	addr := s.newKeysWithTokens()

	tip := sdk.NewCoin(s.Setup.Denom, math.NewInt(10_000))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}

	// tip a spot at block 1, expiration should be 3
	_, err := msgServer.Tip(ctx, &msg)
	s.NoError(err)

	queryId := utils.QueryIDFromData(ethQueryData)
	query, err := okpr.CurrentQuery(ctx, queryId)
	s.Equal(uint64(3), query.Expiration)
	s.Equal(math.NewInt(9800), query.Amount)
	s.NoError(err)

	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker1")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, repAccs[0], reportertypes.NewSelection(repAccs[0], 1)))

	value := testutil.EncodeValue(29266)
	reveal := report(repAccs[0].String(), value, ethQueryData)
	_, err = msgServer.SubmitValue(ctx, &reveal)
	s.NoError(err)
	query, err = okpr.CurrentQuery(ctx, queryId)
	s.True(query.HasRevealedReports)
	s.Equal(uint64(3), query.Expiration)
	s.NoError(err)
	// move to block 2
	ctx, err = simtestutil.NextBlock(app, ctx, time.Second)
	s.NoError(err)
	s.Equal(int64(2), ctx.BlockHeight())

	// move to block 3
	ctx, err = simtestutil.NextBlock(app, ctx, time.Second)
	s.NoError(err)
	s.Equal(int64(3), ctx.BlockHeight())

	// tipping at block 3 should not extend expiration
	_, err = msgServer.Tip(ctx, &msg)
	s.NoError(err)

	query, err = okpr.CurrentQuery(ctx, queryId)
	s.True(query.HasRevealedReports)
	s.Equal(uint64(3), query.Expiration)
	s.Equal(math.NewInt(19600), query.Amount)
	s.NoError(err)
	// move to block 4
	ctx, err = simtestutil.NextBlock(app, ctx, time.Second)
	s.NoError(err)
	s.Equal(int64(4), ctx.BlockHeight())

	// query should not exist as it should have been cleared in the previous end block
	_, err = okpr.CurrentQuery(ctx, queryId)
	s.ErrorIs(err, collections.ErrNotFound)
}

func (s *IntegrationTestSuite) TestRotateQueries() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	ctx := s.Setup.Ctx
	app := s.Setup.App
	okpr := s.Setup.Oraclekeeper
	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	msgServer := keeper.NewMsgServerImpl(okpr)
	_, valAddrs, _ := s.createValidatorAccs([]uint64{100})
	for _, val := range valAddrs {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(ctx, val.String(), []byte("not real"))
		s.NoError(err)
	}
	addr := s.newKeysWithTokens()
	// test for rotating queries going through the cycle list and updating the current query 1,2,3
	// get cycle list
	cycleList, err := okpr.GetCyclelist(ctx)
	s.NoError(err)
	s.Len(cycleList, 3)
	queryId0 := utils.QueryIDFromData(cycleList[0])
	queryId1 := utils.QueryIDFromData(cycleList[1])
	queryId2 := utils.QueryIDFromData(cycleList[2])
	// should be on the second query since the first one is expired from chain running during setup
	query1, err := okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(query1, cycleList[1]))
	query, err := okpr.CurrentQuery(ctx, queryId1)
	s.NoError(err)
	s.Equal(uint64(3), query.Expiration)
	firstTestMetaId := query.Id

	// move to block 2
	ctx, err = simtestutil.NextBlock(app, ctx, time.Second) // next block
	s.NoError(err)
	s.Equal(int64(2), ctx.BlockHeight())
	// should be a noop since the current query is not expired
	query1, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	idx, err := okpr.CyclelistSequencer.Peek(ctx)
	s.NoError(err)
	s.Equal(idx, uint64(1))
	s.True(bytes.Equal(query1, cycleList[idx]))

	// move to block 3
	ctx, err = simtestutil.NextBlock(app, ctx, time.Second) // next block
	s.NoError(err)
	s.Equal(int64(3), ctx.BlockHeight())
	// should be a noop since the current query is not expired
	query1, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(query1, cycleList[1]))

	// move to block 4 -- next query
	ctx, err = simtestutil.NextBlock(app, ctx, time.Second) // next block
	s.NoError(err)
	s.Equal(int64(4), ctx.BlockHeight())
	// cycle query 2
	query2, err := okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	idx, err = okpr.CyclelistSequencer.Peek(ctx)
	s.NoError(err)
	s.Equal(idx, uint64(2))
	s.True(bytes.Equal(query2, cycleList[idx]))

	query, err = okpr.CurrentQuery(ctx, queryId2)
	s.NoError(err)
	s.Equal(firstTestMetaId+1, query.Id)

	// move to block 5
	ctx, err = simtestutil.NextBlock(app, ctx, time.Second) // next block
	s.NoError(err)
	s.Equal(int64(5), ctx.BlockHeight())

	query2, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(query2, cycleList[2]))

	// move to block 7  -- next query
	ctx, err = simtestutil.NextBlock(app, ctx, time.Second) // next block
	s.NoError(err)
	ctx, err = simtestutil.NextBlock(app, ctx, time.Second) // next block
	s.NoError(err)
	s.Equal(int64(7), ctx.BlockHeight())

	// cycle query 3
	query0, err := okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	idx, err = okpr.CyclelistSequencer.Peek(ctx)
	s.NoError(err)
	s.Equal(idx, uint64(0)) // reset
	s.True(bytes.Equal(query0, cycleList[idx]))

	query, err = okpr.CurrentQuery(ctx, queryId0)
	s.NoError(err)
	s.Equal(firstTestMetaId+2, query.Id)

	// checks what happens to an expired query that has not been cleared
	// it would just add time and tip to the query
	// cyclelist[1] is the next upcoming query, tip it here before it is in cycle

	tip := sdk.NewCoin(s.Setup.Denom, math.NewInt(10_000))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: cycleList[1],
		Amount:    tip,
	}

	// tip a spot at block 1, expiration should be 3
	_, err = msgServer.Tip(ctx, &msg)
	s.NoError(err)

	query, err = okpr.CurrentQuery(ctx, queryId1)
	s.NoError(err)
	s.Equal(firstTestMetaId+3, query.Id)
	tippedQuery1ID := query.Id
	// expiration should be 9
	s.Equal(uint64(ctx.BlockHeight()+2), query.Expiration)
	s.Equal(math.NewInt(9800), query.Amount)

	// tip a different query from the list that isn't in cycle
	// testing for it going into cycle when expired and should be extended
	msg.QueryData = cycleList[2]
	_, err = msgServer.Tip(ctx, &msg)
	s.NoError(err)
	// checking the query was set correctly
	query, err = okpr.CurrentQuery(ctx, queryId2)
	s.NoError(err)
	s.Equal(uint64(9), query.Expiration)
	s.Equal(math.NewInt(9800), query.Amount)
	s.True(query.CycleList) // used to be false, now true because of the out-of-turn tip needing to be tracked for liveness
	s.Equal(firstTestMetaId+4, query.Id)

	// rotate the queries which should put queryId1 in cycle
	// expiration should not be extended for queryId1 only set cycle list to true
	// move to block 9  -- next query
	ctx, err = simtestutil.NextBlock(app, ctx, time.Second) // next block
	s.NoError(err)
	ctx, err = simtestutil.NextBlock(app, ctx, time.Second) // next block
	s.NoError(err)
	s.Equal(int64(9), ctx.BlockHeight())

	query, err = okpr.CurrentQuery(ctx, queryId1)
	s.NoError(err)
	s.Equal(uint64(9), query.Expiration)
	s.Equal(math.NewInt(9800), query.Amount)
	s.True(query.CycleList)
	s.Equal(tippedQuery1ID, query.Id)

	// rotate the queries which should put queryId2 in cycle
	// but since it will be expired it should be extended
	// move to block 11  -- next query
	ctx, err = simtestutil.NextBlock(app, ctx, time.Second) // next block
	s.NoError(err)
	ctx, err = simtestutil.NextBlock(app, ctx, time.Second) // next block
	s.NoError(err)
	s.Equal(int64(11), ctx.BlockHeight())

	query, err = okpr.CurrentQuery(ctx, queryId2)
	s.NoError(err)
	s.Equal(uint64(11), query.Expiration)
	s.Equal(math.NewInt(9800), query.Amount)
	s.True(query.CycleList)
	s.Equal(firstTestMetaId+5, query.Id)

	// test the clearing of old query that doesn't have a tip and has expired
	// should clear the old query.Id and generate a new query
	query, err = okpr.CurrentQuery(ctx, queryId0)
	s.Equal(uint64(2), query.Id)
	s.NoError(err)
	// expired query
	s.Equal(uint64(7), query.Expiration)

	// move to block 12  -- next query
	ctx, err = simtestutil.NextBlock(app, ctx, time.Second) // next block
	s.NoError(err)
	s.Equal(int64(12), ctx.BlockHeight())
	query, err = okpr.CurrentQuery(ctx, queryId0)
	s.Equal(uint64(6), query.Id)
	s.NoError(err)
	// expired query
	s.Equal(uint64(13), query.Expiration)
	s.Equal(firstTestMetaId+6, query.Id)

	_, err = okpr.Query.Get(ctx, collections.Join(queryId0, uint64(2)))
	s.ErrorIs(err, collections.ErrNotFound)
}
