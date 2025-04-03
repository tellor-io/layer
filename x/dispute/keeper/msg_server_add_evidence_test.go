package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
)

func (s *KeeperTestSuite) TestAddEvidence() {
	require := s.Require()
	require.NotNil(s.msgServer)
	ok := s.oracleKeeper
	s.ctx = s.ctx.WithBlockTime(time.Now())
	s.ctx = s.ctx.WithBlockHeight(10)

	// empty message
	_, err := s.msgServer.AddEvidence(s.ctx, &types.MsgAddEvidence{})
	require.Error(err)

	// open dispute
	dispute := s.dispute(s.ctx)
	s.NoError(s.disputeKeeper.Disputes.Set(s.ctx, dispute.DisputeId, dispute))
	disputedReporter := dispute.InitialEvidence.Reporter

	// add evidence, evidence is not an aggregate
	addtlReport := oracletypes.MicroReport{
		Reporter:        disputedReporter,
		Power:           100,
		QueryType:       dispute.InitialEvidence.QueryType,
		QueryId:         dispute.InitialEvidence.QueryId,
		AggregateMethod: dispute.InitialEvidence.AggregateMethod,
		Value:           "100",
		Timestamp:       s.ctx.BlockTime().Add(-1 * 24 * time.Hour), // 1 day old
		Cyclelist:       dispute.InitialEvidence.Cyclelist,
		BlockNumber:     uint64(s.ctx.BlockHeight()),
	}
	evidence := []*oracletypes.MicroReport{&addtlReport}
	ok.On("FlagAggregateReport", s.ctx, addtlReport).Return(collections.ErrNotFound).Once()
	_, err = s.msgServer.AddEvidence(s.ctx, &types.MsgAddEvidence{
		DisputeId: dispute.DisputeId,
		Reports:   evidence,
	})
	require.NoError(err)
	// check dispute to see if report is in additional evidence
	dispute, err = s.disputeKeeper.Disputes.Get(s.ctx, dispute.DisputeId)
	require.NoError(err)
	require.Equal(len(dispute.AdditionalEvidence), 1)

	// add evidence that is an aggregate
	addtlReport = oracletypes.MicroReport{
		Reporter:        disputedReporter,
		Power:           100,
		QueryType:       dispute.InitialEvidence.QueryType,
		QueryId:         dispute.InitialEvidence.QueryId,
		AggregateMethod: dispute.InitialEvidence.AggregateMethod,
		Value:           "200",
		Timestamp:       s.ctx.BlockTime(),
		Cyclelist:       dispute.InitialEvidence.Cyclelist,
		BlockNumber:     uint64(s.ctx.BlockHeight()),
	}
	evidence = append(evidence, &addtlReport)
	ok.On("FlagAggregateReport", s.ctx, *evidence[0]).Return(collections.ErrNotFound).Once()
	ok.On("FlagAggregateReport", s.ctx, *evidence[1]).Return(nil).Once()
	_, err = s.msgServer.AddEvidence(s.ctx, &types.MsgAddEvidence{
		DisputeId: dispute.DisputeId,
		Reports:   evidence,
	})
	require.NoError(err)
	// check dispute to see if report is in additional evidence
	dispute, err = s.disputeKeeper.Disputes.Get(s.ctx, dispute.DisputeId)
	require.NoError(err)
	require.Equal(len(dispute.AdditionalEvidence), 3) // first + 2 additional
}

func BenchmarkAddEvidence(b *testing.B) {
	require := require.New(b)
	k, ok, _, _, _, ctx := keepertest.DisputeKeeper(b)
	msgServer := keeper.NewMsgServerImpl(k)
	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(10)

	// create initial dispute and set it
	dispute := types.Dispute{
		DisputeId: 1,
		InitialEvidence: oracletypes.MicroReport{
			Reporter:    "reporter1",
			Power:       100,
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
		DisputeFee:        math.NewInt(100),
		DisputeStatus:     types.Voting,
		DisputeStartTime:  ctx.BlockTime().Add(-1 * time.Hour),
		DisputeEndTime:    ctx.BlockTime().Add(1 * time.Hour),
		DisputeStartBlock: uint64(ctx.BlockHeight()),
		DisputeRound:      1,
		SlashAmount:       math.NewInt(100),
		PendingExecution:  false,
		BurnAmount:        math.NewInt(100),
		FeeTotal:          math.NewInt(100),
	}
	require.NoError(k.Disputes.Set(ctx, dispute.DisputeId, dispute))

	// prepare test data
	addtlReport := oracletypes.MicroReport{
		Reporter:    dispute.InitialEvidence.Reporter,
		Power:       100,
		QueryType:   dispute.InitialEvidence.QueryType,
		QueryId:     dispute.InitialEvidence.QueryId,
		Value:       "100",
		Timestamp:   ctx.BlockTime().Add(-1 * 24 * time.Hour),
		Cyclelist:   dispute.InitialEvidence.Cyclelist,
		BlockNumber: uint64(ctx.BlockHeight()),
	}
	evidence := []*oracletypes.MicroReport{&addtlReport}

	// reset timer before the benchmark loop
	b.ResetTimer()

	// run the benchmark
	for i := 0; i < b.N; i++ {
		ok.On("FlagAggregateReport", ctx, addtlReport).Return(nil)
		_, err := msgServer.AddEvidence(ctx, &types.MsgAddEvidence{
			DisputeId: dispute.DisputeId,
			Reports:   evidence,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
