package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func (k *KeeperTestSuite) TestMsgServerAddFeeToDispute() {
	k.ctx = k.ctx.WithBlockTime(time.Now())
	creator := sample.AccAddressBytes()
	msg := types.MsgAddFeeToDispute{
		Creator:     creator.String(),
		DisputeId:   1,
		Amount:      sdk.NewCoin("loya", math.NewInt(10000)),
		PayFromBond: false,
	}

	msg.Amount.Denom = "bond"
	res, err := k.msgServer.AddFeeToDispute(k.ctx, &msg)
	k.ErrorContains(err, "fee must be paid in loya")
	k.Nil(res)

	msg.Amount.Denom = "loya"
	res, err = k.msgServer.AddFeeToDispute(k.ctx, &msg)
	k.ErrorContains(err, "not found")
	k.Nil(res)

	dispute := k.dispute(k.ctx)
	dispute.FeeTotal = dispute.SlashAmount
	dispute.InitialEvidence.QueryId = []byte("query")
	fmt.Println("dispute: ", dispute)
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))

	res, err = k.msgServer.AddFeeToDispute(k.ctx, &msg)
	k.ErrorContains(err, "dispute fee already met")
	k.Nil(res)

	dispute.FeeTotal = math.ZeroInt()
	k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))
	fee := sdk.NewCoin(layer.BondDenom, math.NewInt(10000))
	k.oracleKeeper.On("FlagAggregateReport", k.ctx, mock.Anything).Return(nil)
	k.bankKeeper.On("HasBalance", k.ctx, creator, fee).Return(true)
	k.bankKeeper.On("SendCoinsFromAccountToModule", k.ctx, creator, types.ModuleName, sdk.NewCoins(fee)).Return(nil)
	k.reporterKeeper.On("EscrowReporterStake", k.ctx, sdk.MustAccAddressFromBech32(dispute.InitialEvidence.Reporter), dispute.InitialEvidence.Power, uint64(1), dispute.SlashAmount, dispute.InitialEvidence.QueryId, dispute.HashId).Return(nil)
	// jail duration is 0
	k.reporterKeeper.On("JailReporter", k.ctx, sdk.MustAccAddressFromBech32(dispute.InitialEvidence.Reporter), uint64(0)).Return(nil)

	res, err = k.msgServer.AddFeeToDispute(k.ctx, &msg)
	k.NoError(err)
	k.NotNil(res)

	// try to add again
	res, err = k.msgServer.AddFeeToDispute(k.ctx, &msg)
	k.ErrorContains(err, "dispute fee already met")
	k.Nil(res)
}

func BenchmarkAddFeeToDispute(b *testing.B) {
	require := require.New(b)
	k, ok, rk, _, bk, ctx := keepertest.DisputeKeeper(b)
	msgServer := keeper.NewMsgServerImpl(k)
	ctx = ctx.WithBlockTime(time.Now())

	// setup test data
	creator := sample.AccAddressBytes()
	reporter := sample.AccAddressBytes()
	msg := types.MsgAddFeeToDispute{
		Creator:     creator.String(),
		DisputeId:   1,
		Amount:      sdk.NewCoin("loya", math.NewInt(10000)),
		PayFromBond: false,
	}

	// create and set up dispute
	dispute := types.Dispute{
		DisputeId: 1,
		InitialEvidence: oracletypes.MicroReport{
			Reporter:    reporter.String(),
			Power:       1000,
			QueryType:   "test",
			QueryId:     []byte("test1"),
			Value:       "150",
			Timestamp:   ctx.BlockTime(),
			BlockNumber: uint64(ctx.BlockHeight()),
			Cyclelist:   true,
		},
		Open:              true,
		HashId:            []byte("hash"),
		DisputeCategory:   types.Warning,
		DisputeFee:        math.NewInt(1000),
		DisputeStatus:     types.Voting,
		DisputeStartTime:  ctx.BlockTime().Add(-1 * time.Hour),
		DisputeEndTime:    ctx.BlockTime().Add(1 * time.Hour),
		DisputeStartBlock: uint64(ctx.BlockHeight()),
		DisputeRound:      1,
		SlashAmount:       math.NewInt(1000),
		PendingExecution:  false,
		BurnAmount:        math.NewInt(1000),
		FeeTotal:          math.NewInt(0),
	}
	require.NoError(k.Disputes.Set(ctx, dispute.DisputeId, dispute))

	// setup mock expectations
	fee := sdk.NewCoin(layer.BondDenom, math.NewInt(1000))
	ok.On("FlagAggregateReport", ctx, mock.Anything).Return(nil).Maybe()
	bk.On("HasBalance", ctx, creator, fee).Return(true).Maybe()
	bk.On("SendCoinsFromAccountToModule", ctx, creator, types.ModuleName, sdk.NewCoins(fee)).Return(nil).Maybe()
	rk.On("EscrowReporterStake", ctx,
		sdk.MustAccAddressFromBech32(dispute.InitialEvidence.Reporter),
		dispute.InitialEvidence.Power,
		uint64(0),
		math.NewInt(10000000),
		dispute.InitialEvidence.QueryId,
		dispute.HashId).Return(nil).Maybe()
	rk.On("JailReporter", ctx,
		sdk.MustAccAddressFromBech32(dispute.InitialEvidence.Reporter),
		uint64(0)).Return(nil).Maybe()

	// reset timer before the benchmark loop
	b.ResetTimer()

	// run the benchmark
	for i := 0; i < b.N; i++ {
		_, err := msgServer.AddFeeToDispute(ctx, &msg)
		if err != nil {
			b.Fatal(err)
		}

		// reset the dispute fee total for next iteration
		dispute.FeeTotal = math.ZeroInt()
		if err := k.Disputes.Set(ctx, dispute.DisputeId, dispute); err != nil {
			b.Fatal(err)
		}
	}
}
