package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

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
	ok.On("IsAggregateReport", s.ctx, addtlReport).Return(false, nil).Once()
	ok.On("FlagAggregateReport", s.ctx, addtlReport).Return(nil).Once()
	_, err = s.msgServer.AddEvidence(s.ctx, &types.MsgAddEvidence{
		DisputeId: dispute.DisputeId,
		Reports:   evidence,
	})
	require.NoError(err)
	// check dispute to see if report is in additional evidence
	dispute, err = s.disputeKeeper.Disputes.Get(s.ctx, dispute.DisputeId)
	require.NoError(err)
	require.Equal(len(dispute.AdditionalEvidence), 1)

	// warning disputes cannot add aggregate reports as evidence
	aggReport := oracletypes.MicroReport{
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
	ok.On("IsAggregateReport", s.ctx, aggReport).Return(true, nil).Once()
	_, err = s.msgServer.AddEvidence(s.ctx, &types.MsgAddEvidence{
		DisputeId: dispute.DisputeId,
		Reports:   []*oracletypes.MicroReport{&aggReport},
	})
	require.ErrorIs(err, types.ErrWarningAggregateEvidence)
	dispute, err = s.disputeKeeper.Disputes.Get(s.ctx, dispute.DisputeId)
	require.NoError(err)
	require.Equal(len(dispute.AdditionalEvidence), 1) // unchanged
}

func (s *KeeperTestSuite) TestAddEvidence_MinorAggregateLimit() {
	require := s.Require()
	ok := s.oracleKeeper
	s.ctx = s.ctx.WithBlockTime(time.Now())
	s.ctx = s.ctx.WithBlockHeight(10)

	dispute := s.dispute(s.ctx)
	dispute.DisputeCategory = types.Minor
	s.NoError(s.disputeKeeper.Disputes.Set(s.ctx, dispute.DisputeId, dispute))
	dispute, err := s.disputeKeeper.Disputes.Get(s.ctx, dispute.DisputeId)
	require.NoError(err)
	disputedReporter := dispute.InitialEvidence.Reporter

	aggReport := oracletypes.MicroReport{
		Reporter:    disputedReporter,
		Power:       100,
		QueryType:   dispute.InitialEvidence.QueryType,
		QueryId:     dispute.InitialEvidence.QueryId,
		Value:       "200",
		Timestamp:   s.ctx.BlockTime(),
		Cyclelist:   dispute.InitialEvidence.Cyclelist,
		BlockNumber: uint64(s.ctx.BlockHeight()),
	}
	nonAggReport := oracletypes.MicroReport{
		Reporter:    disputedReporter,
		Power:       100,
		QueryType:   dispute.InitialEvidence.QueryType,
		QueryId:     dispute.InitialEvidence.QueryId,
		Value:       "100",
		Timestamp:   s.ctx.BlockTime().Add(-1 * 24 * time.Hour),
		Cyclelist:   dispute.InitialEvidence.Cyclelist,
		BlockNumber: uint64(s.ctx.BlockHeight()) + 1,
	}

	// initial evidence is not an aggregate; first aggregate evidence is allowed
	ok.On("IsAggregateReport", mock.Anything, dispute.InitialEvidence).Return(false, nil).Once()
	ok.On("IsAggregateReport", mock.Anything, aggReport).Return(true, nil).Once()
	ok.On("FlagAggregateReport", mock.Anything, aggReport).Return(nil).Once()
	_, err = s.msgServer.AddEvidence(s.ctx, &types.MsgAddEvidence{
		DisputeId: dispute.DisputeId,
		Reports:   []*oracletypes.MicroReport{&aggReport},
	})
	require.NoError(err)

	dispute, err = s.disputeKeeper.Disputes.Get(s.ctx, dispute.DisputeId)
	require.NoError(err)
	require.Equal(1, len(dispute.AdditionalEvidence))

	// second aggregate evidence is rejected
	secondAgg := aggReport
	secondAgg.Value = "300"
	secondAgg.BlockNumber = uint64(s.ctx.BlockHeight()) + 2
	ok.On("IsAggregateReport", mock.Anything, dispute.InitialEvidence).Return(false, nil).Once()
	ok.On("IsAggregateReport", mock.Anything, *dispute.AdditionalEvidence[0]).Return(true, nil).Once()
	ok.On("IsAggregateReport", mock.Anything, secondAgg).Return(true, nil).Once()
	_, err = s.msgServer.AddEvidence(s.ctx, &types.MsgAddEvidence{
		DisputeId: dispute.DisputeId,
		Reports:   []*oracletypes.MicroReport{&secondAgg},
	})
	require.ErrorIs(err, types.ErrMinorAggregateEvidenceLimit)

	// non-aggregate evidence is still allowed
	ok.On("IsAggregateReport", mock.Anything, dispute.InitialEvidence).Return(false, nil).Once()
	ok.On("IsAggregateReport", mock.Anything, *dispute.AdditionalEvidence[0]).Return(true, nil).Once()
	ok.On("IsAggregateReport", mock.Anything, nonAggReport).Return(false, nil).Once()
	ok.On("FlagAggregateReport", mock.Anything, nonAggReport).Return(nil).Once()
	_, err = s.msgServer.AddEvidence(s.ctx, &types.MsgAddEvidence{
		DisputeId: dispute.DisputeId,
		Reports:   []*oracletypes.MicroReport{&nonAggReport},
	})
	require.NoError(err)
}

func (s *KeeperTestSuite) TestAddEvidence_MinorInitialEvidenceIsAggregate() {
	require := s.Require()
	ok := s.oracleKeeper
	s.ctx = s.ctx.WithBlockTime(time.Now())
	s.ctx = s.ctx.WithBlockHeight(10)

	dispute := s.dispute(s.ctx)
	dispute.DisputeCategory = types.Minor
	s.NoError(s.disputeKeeper.Disputes.Set(s.ctx, dispute.DisputeId, dispute))
	dispute, err := s.disputeKeeper.Disputes.Get(s.ctx, dispute.DisputeId)
	require.NoError(err)

	aggReport := oracletypes.MicroReport{
		Reporter:    dispute.InitialEvidence.Reporter,
		Power:       100,
		QueryType:   dispute.InitialEvidence.QueryType,
		QueryId:     dispute.InitialEvidence.QueryId,
		Value:       "200",
		Timestamp:   s.ctx.BlockTime(),
		Cyclelist:   dispute.InitialEvidence.Cyclelist,
		BlockNumber: uint64(s.ctx.BlockHeight()),
	}

	// initial evidence already counts as the one allowed aggregate
	ok.On("IsAggregateReport", mock.Anything, dispute.InitialEvidence).Return(true, nil).Once()
	ok.On("IsAggregateReport", mock.Anything, aggReport).Return(true, nil).Once()
	_, err = s.msgServer.AddEvidence(s.ctx, &types.MsgAddEvidence{
		DisputeId: dispute.DisputeId,
		Reports:   []*oracletypes.MicroReport{&aggReport},
	})
	require.ErrorIs(err, types.ErrMinorAggregateEvidenceLimit)
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
		ok.On("IsAggregateReport", mock.Anything, addtlReport).Return(false, nil)
		ok.On("FlagAggregateReport", mock.Anything, addtlReport).Return(nil)
		_, err := msgServer.AddEvidence(ctx, &types.MsgAddEvidence{
			DisputeId: dispute.DisputeId,
			Reports:   evidence,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
