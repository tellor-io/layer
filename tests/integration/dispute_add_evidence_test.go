package integration_test

import (
	"encoding/hex"
	"time"

	"github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// plantAggregate indexes a microreport as an aggregate reporter so IsAggregateReport/FlagAggregateReport see it.
func (s *IntegrationTestSuite) plantAggregate(report oracletypes.MicroReport) {
	s.T().Helper()
	aggKey := collections.Join(report.QueryId, uint64(report.Timestamp.UnixMilli()))
	s.NoError(s.Setup.Oraclekeeper.Aggregates.Set(s.Setup.Ctx, aggKey, oracletypes.Aggregate{
		QueryId:           report.QueryId,
		AggregateReporter: report.Reporter,
		AggregateValue:    report.Value,
		MicroHeight:       report.BlockNumber,
		Height:            report.BlockNumber,
		Flagged:           false,
	}))
}

func (s *IntegrationTestSuite) setupAddEvidenceFixture() (reporter sdk.AccAddress, nonAgg, agg1, agg2 oracletypes.MicroReport, msgServer types.MsgServer) {
	s.T().Helper()
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now()).WithBlockHeight(10)

	valAccs, _, _ := s.createValidatorAccs([]uint64{100, 200})
	reporter = valAccs[0]
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, reporter, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, reporter, reportertypes.NewSelection(reporter, 1)))

	btcQid := utils.QueryIDFromData(btcQueryData)
	ethQid := utils.QueryIDFromData(ethQueryData)
	trbQid := utils.QueryIDFromData(trbQueryData)

	stake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, reporter, btcQid)
	s.NoError(err)
	power := uint64(sdk.TokensToConsensusPower(stake, sdk.DefaultPowerReduction))

	nonAgg = oracletypes.MicroReport{
		Reporter:        reporter.String(),
		Power:           power,
		QueryId:         btcQid,
		QueryType:       "SpotPrice",
		AggregateMethod: "weighted-median",
		Value:           testutil.EncodeValue(1.00),
		Timestamp:       s.Setup.Ctx.BlockTime(),
		Cyclelist:       true,
		BlockNumber:     uint64(s.Setup.Ctx.BlockHeight()),
		MetaId:          1,
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(nonAgg.QueryId, reporter.Bytes(), nonAgg.MetaId), nonAgg))

	agg1 = oracletypes.MicroReport{
		Reporter:        reporter.String(),
		Power:           power,
		QueryId:         ethQid,
		QueryType:       "SpotPrice",
		AggregateMethod: "weighted-median",
		Value:           testutil.EncodeValue(2.00),
		Timestamp:       s.Setup.Ctx.BlockTime().Add(time.Second),
		Cyclelist:       true,
		BlockNumber:     uint64(s.Setup.Ctx.BlockHeight()) + 1,
		MetaId:          2,
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(agg1.QueryId, reporter.Bytes(), agg1.MetaId), agg1))
	s.plantAggregate(agg1)

	agg2 = oracletypes.MicroReport{
		Reporter:        reporter.String(),
		Power:           power,
		QueryId:         trbQid,
		QueryType:       "SpotPrice",
		AggregateMethod: "weighted-median",
		Value:           testutil.EncodeValue(3.00),
		Timestamp:       s.Setup.Ctx.BlockTime().Add(2 * time.Second),
		Cyclelist:       true,
		BlockNumber:     uint64(s.Setup.Ctx.BlockHeight()) + 2,
		MetaId:          3,
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(agg2.QueryId, reporter.Bytes(), agg2.MetaId), agg2))
	s.plantAggregate(agg2)

	isAgg, err := s.Setup.Oraclekeeper.IsAggregateReport(s.Setup.Ctx, nonAgg)
	s.NoError(err)
	s.False(isAgg)
	isAgg, err = s.Setup.Oraclekeeper.IsAggregateReport(s.Setup.Ctx, agg1)
	s.NoError(err)
	s.True(isAgg)
	isAgg, err = s.Setup.Oraclekeeper.IsAggregateReport(s.Setup.Ctx, agg2)
	s.NoError(err)
	s.True(isAgg)

	return reporter, nonAgg, agg1, agg2, keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
}

func (s *IntegrationTestSuite) proposeOpenDispute(category types.DisputeCategory, report oracletypes.MicroReport) uint64 {
	s.T().Helper()
	disputer := s.newKeysWithTokens()
	s.Setup.MintTokens(disputer, math.NewInt(100_000_000))
	fee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, report, category)
	s.NoError(err)

	_, err = keeper.NewMsgServerImpl(s.Setup.Disputekeeper).ProposeDispute(s.Setup.Ctx, &types.MsgProposeDispute{
		Creator:          disputer.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		Fee:              sdk.NewCoin(s.Setup.Denom, fee),
		DisputeCategory:  category,
	})
	s.NoError(err)

	dispute, err := s.Setup.Disputekeeper.GetDisputeByReporter(s.Setup.Ctx, report, category)
	s.NoError(err)
	s.True(dispute.Open)
	return dispute.DisputeId
}

func (s *IntegrationTestSuite) TestAddEvidence_WarningRejectsAggregate() {
	_, nonAgg, agg1, _, msgServer := s.setupAddEvidenceFixture()
	disputeID := s.proposeOpenDispute(types.Warning, nonAgg)

	_, err := msgServer.AddEvidence(s.Setup.Ctx, &types.MsgAddEvidence{
		CallerAddress: s.newKeysWithTokens().String(),
		DisputeId:     disputeID,
		Reports:       []*oracletypes.MicroReport{&agg1},
	})
	s.ErrorIs(err, types.ErrWarningAggregateEvidence)

	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeID)
	s.NoError(err)
	s.Empty(dispute.AdditionalEvidence)

	// non-aggregate evidence is still allowed
	_, err = msgServer.AddEvidence(s.Setup.Ctx, &types.MsgAddEvidence{
		CallerAddress: s.newKeysWithTokens().String(),
		DisputeId:     disputeID,
		Reports: []*oracletypes.MicroReport{{
			Reporter:    nonAgg.Reporter,
			Power:       nonAgg.Power,
			QueryId:     nonAgg.QueryId,
			QueryType:   nonAgg.QueryType,
			Value:       testutil.EncodeValue(9.00),
			Timestamp:   s.Setup.Ctx.BlockTime().Add(-time.Hour),
			Cyclelist:   true,
			BlockNumber: nonAgg.BlockNumber + 10,
			MetaId:      99,
		}},
	})
	s.NoError(err)
	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeID)
	s.NoError(err)
	s.Len(dispute.AdditionalEvidence, 1)
}

func (s *IntegrationTestSuite) TestAddEvidence_MinorAggregateLimit() {
	_, nonAgg, agg1, agg2, msgServer := s.setupAddEvidenceFixture()
	disputeID := s.proposeOpenDispute(types.Minor, nonAgg)

	_, err := msgServer.AddEvidence(s.Setup.Ctx, &types.MsgAddEvidence{
		CallerAddress: s.newKeysWithTokens().String(),
		DisputeId:     disputeID,
		Reports:       []*oracletypes.MicroReport{&agg1},
	})
	s.NoError(err)

	aggKey := collections.Join(agg1.QueryId, uint64(agg1.Timestamp.UnixMilli()))
	agg, err := s.Setup.Oraclekeeper.Aggregates.Get(s.Setup.Ctx, aggKey)
	s.NoError(err)
	s.True(agg.Flagged)

	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeID)
	s.NoError(err)
	s.Len(dispute.AdditionalEvidence, 1)

	_, err = msgServer.AddEvidence(s.Setup.Ctx, &types.MsgAddEvidence{
		CallerAddress: s.newKeysWithTokens().String(),
		DisputeId:     disputeID,
		Reports:       []*oracletypes.MicroReport{&agg2},
	})
	s.ErrorIs(err, types.ErrMinorAggregateEvidenceLimit)

	dispute, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeID)
	s.NoError(err)
	s.Len(dispute.AdditionalEvidence, 1)
}

func (s *IntegrationTestSuite) TestAddEvidence_MinorInitialEvidenceIsAggregate() {
	_, _, agg1, agg2, msgServer := s.setupAddEvidenceFixture()
	disputeID := s.proposeOpenDispute(types.Minor, agg1)

	aggKey := collections.Join(agg1.QueryId, uint64(agg1.Timestamp.UnixMilli()))
	agg, err := s.Setup.Oraclekeeper.Aggregates.Get(s.Setup.Ctx, aggKey)
	s.NoError(err)
	s.True(agg.Flagged, "proposing on an aggregate still flags it")

	_, err = msgServer.AddEvidence(s.Setup.Ctx, &types.MsgAddEvidence{
		CallerAddress: s.newKeysWithTokens().String(),
		DisputeId:     disputeID,
		Reports:       []*oracletypes.MicroReport{&agg2},
	})
	s.ErrorIs(err, types.ErrMinorAggregateEvidenceLimit)
}

func (s *IntegrationTestSuite) TestAddEvidence_WarningProposeOnAggregateStillAllowed() {
	_, _, agg1, _, _ := s.setupAddEvidenceFixture()
	disputeID := s.proposeOpenDispute(types.Warning, agg1)

	aggKey := collections.Join(agg1.QueryId, uint64(agg1.Timestamp.UnixMilli()))
	agg, err := s.Setup.Oraclekeeper.Aggregates.Get(s.Setup.Ctx, aggKey)
	s.NoError(err)
	s.True(agg.Flagged)

	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeID)
	s.NoError(err)
	s.Equal(types.Warning, dispute.DisputeCategory)
	s.True(dispute.Open)
}
