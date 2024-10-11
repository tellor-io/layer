package keeper_test

import (
	"time"

	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
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
	dispute := s.dispute()
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
		Timestamp:       s.ctx.BlockTime(),
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
	// TODO: check that aggregate is flagged in integration/e2e suite

}
