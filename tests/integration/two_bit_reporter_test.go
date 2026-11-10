package integration_test

import (
	"encoding/hex"

	"github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	registrykeeper "github.com/tellor-io/layer/x/registry/keeper"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// A 2bit agent reports only after accepting, and only for 2bitAgentic queries.
// The report, its power, and the tip all belong to the reporter.
func (s *IntegrationTestSuite) TestTwoBitAgentReportCreditsReporter() {
	require := s.Require()
	ctx := s.Setup.Ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	s.Setup.Ctx = ctx

	reporterMsg := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	oracleMsg := oraclekeeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	registryMsg := registrykeeper.NewMsgServerImpl(s.Setup.Registrykeeper)

	reporters, _, _ := s.createValidatorAccs([]uint64{1000})
	reporter := reporters[0]
	commission := math.LegacyNewDecWithPrec(1, 1)
	_, err := reporterMsg.CreateReporter(ctx, &reportertypes.MsgCreateReporter{
		ReporterAddress:   reporter.String(),
		CommissionRate:    commission,
		MinTokensRequired: reportertypes.DefaultMinLoya,
		Moniker:           "two-bit-owner",
	})
	require.NoError(err)

	agent := sample.AccAddressBytes()
	dataspec := registrytypes.DataSpec{
		DocumentHash:      "2bit-agentic",
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		AbiComponents: []*registrytypes.ABIComponent{{
			Name:            "payload",
			FieldType:       "string",
			NestedComponent: []*registrytypes.ABIComponent{},
		}},
		Registrar:         reporter.String(),
		QueryType:         "2bitAgentic",
		ReportBlockWindow: 2,
	}
	_, err = registryMsg.RegisterSpec(ctx, &registrytypes.MsgRegisterSpec{
		Registrar: reporter.String(),
		QueryType: "2bitAgentic",
		Spec:      dataspec,
	})
	require.NoError(err)

	queryData, err := dataspec.EncodeData("2bitAgentic", `["report"]`)
	require.NoError(err)
	queryID := utils.QueryIDFromData(queryData)
	value := testutil.EncodeValue(123.45)
	// Delegation spent the validator's liquid balance, so mint the tip separately.
	s.Setup.MintTokens(reporter, math.NewInt(1_000_000))
	tip := sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000_000))
	_, err = oracleMsg.Tip(ctx, &oracletypes.MsgTip{
		Tipper:    reporter.String(),
		QueryData: queryData,
		Amount:    tip,
	})
	require.NoError(err)
	query, err := s.Setup.Oraclekeeper.CurrentQuery(ctx, queryID)
	require.NoError(err)

	_, err = reporterMsg.RegisterTwoBitReporter(ctx, &reportertypes.MsgRegisterTwoBitReporter{
		ReporterAddress: reporter.String(),
		TwoBitAddress:   agent.String(),
	})
	require.NoError(err)

	reporterQuery := reporterkeeper.NewQuerier(s.Setup.Reporterkeeper)
	expiry := uint64(ctx.BlockHeight()) + reportertypes.TwoBitPendingRequestExpiryBlocks
	pending := &reportertypes.TwoBitPendingRequestInfo{
		Reporter:      reporter.String(),
		TwoBitAddress: agent.String(),
		ExpiryHeight:  expiry,
	}
	fromReporter, err := reporterQuery.TwoBitPendingRequests(ctx, &reportertypes.QueryTwoBitPendingRequestsRequest{
		ReporterAddress: reporter.String(),
	})
	require.NoError(err)
	require.Equal([]*reportertypes.TwoBitPendingRequestInfo{pending}, fromReporter.Requests)
	toAgent, err := reporterQuery.TwoBitPendingRequests(ctx, &reportertypes.QueryTwoBitPendingRequestsRequest{
		TwoBitAddress: agent.String(),
	})
	require.NoError(err)
	require.Equal([]*reportertypes.TwoBitPendingRequestInfo{pending}, toAgent.Requests)
	beforeAccept, err := reporterQuery.Reporter(ctx, &reportertypes.QueryReporterRequest{ReporterAddress: reporter.String()})
	require.NoError(err)
	require.Empty(beforeAccept.Reporter.TwoBitAddress)

	submit := &oracletypes.MsgSubmitValue{Creator: agent.String(), QueryData: queryData, Value: value}
	_, err = oracleMsg.SubmitValue(ctx, submit)
	require.ErrorIs(err, collections.ErrNotFound)

	_, err = oracleMsg.SubmitValue(ctx, &oracletypes.MsgSubmitValue{Creator: agent.String(), QueryData: ethQueryData, Value: value})
	require.ErrorIs(err, collections.ErrNotFound)

	_, err = reporterMsg.AcceptTwoBitRegistration(ctx, &reportertypes.MsgAcceptTwoBitRegistration{
		TwoBitAddress:   agent.String(),
		ReporterAddress: reporter.String(),
	})
	require.NoError(err)

	accepted, err := reporterQuery.Reporter(ctx, &reportertypes.QueryReporterRequest{ReporterAddress: reporter.String()})
	require.NoError(err)
	require.Equal(agent.String(), accepted.Reporter.TwoBitAddress)
	listed, err := reporterQuery.Reporters(ctx, &reportertypes.QueryReportersRequest{})
	require.NoError(err)
	require.Len(listed.Reporters, 1)
	require.Equal(reporter.String(), listed.Reporters[0].Address)
	require.Equal(agent.String(), listed.Reporters[0].TwoBitAddress)
	afterAccept, err := reporterQuery.TwoBitPendingRequests(ctx, &reportertypes.QueryTwoBitPendingRequestsRequest{
		ReporterAddress: reporter.String(),
	})
	require.NoError(err)
	require.Empty(afterAccept.Requests)

	// Accepting does not let the agent report any other query type.
	_, err = oracleMsg.SubmitValue(ctx, &oracletypes.MsgSubmitValue{Creator: agent.String(), QueryData: ethQueryData, Value: value})
	require.ErrorIs(err, collections.ErrNotFound)

	stake, err := s.Setup.Reporterkeeper.ReporterStake(ctx, reporter, queryID)
	require.NoError(err)
	_, err = oracleMsg.SubmitValue(ctx, submit)
	require.NoError(err)

	report, err := s.Setup.Oraclekeeper.Reports.Get(ctx, collections.Join3(queryID, reporter.Bytes(), query.Id))
	require.NoError(err)
	require.Equal(reporter.String(), report.Reporter)
	require.Equal("2bitAgentic", report.QueryType)
	require.Equal(value, report.Value)
	require.Equal(stake.Quo(layertypes.PowerReduction).Uint64(), report.Power)
	hasAgentReport, err := s.Setup.Oraclekeeper.Reports.Has(ctx, collections.Join3(queryID, agent.Bytes(), query.Id))
	require.NoError(err)
	require.False(hasAgentReport)

	aggregateCtx := ctx.WithBlockHeight(int64(query.Expiration)).WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	require.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(aggregateCtx))
	aggregate, err := oraclekeeper.NewQuerier(s.Setup.Oraclekeeper).GetCurrentAggregateReport(aggregateCtx, &oracletypes.QueryGetCurrentAggregateReportRequest{
		QueryId: hex.EncodeToString(queryID),
	})
	require.NoError(err)
	require.Equal(reporter.String(), aggregate.Aggregate.AggregateReporter)

	expectedCommission := query.Amount.ToLegacyDec().Mul(commission)
	reporterTips, err := s.Setup.Reporterkeeper.SelectorTips.Get(aggregateCtx, reporter.Bytes())
	require.NoError(err)
	require.True(expectedCommission.Equal(reporterTips))
	_, err = s.Setup.Reporterkeeper.SelectorTips.Get(aggregateCtx, agent.Bytes())
	require.ErrorIs(err, collections.ErrNotFound)
	period, err := s.Setup.Reporterkeeper.ReporterPeriodData.Get(aggregateCtx, reporter)
	require.NoError(err)
	require.True(query.Amount.ToLegacyDec().Sub(expectedCommission).Equal(period.RewardAmount))
	_, err = s.Setup.Reporterkeeper.ReporterPeriodData.Get(aggregateCtx, agent)
	require.ErrorIs(err, collections.ErrNotFound)

	_, err = reporterMsg.RegisterTwoBitReporter(aggregateCtx, &reportertypes.MsgRegisterTwoBitReporter{
		ReporterAddress: reporter.String(),
		TwoBitAddress:   reporter.String(),
	})
	require.NoError(err)
	cleared, err := reporterQuery.Reporter(aggregateCtx, &reportertypes.QueryReporterRequest{ReporterAddress: reporter.String()})
	require.NoError(err)
	require.Empty(cleared.Reporter.TwoBitAddress)
	clearedPending, err := reporterQuery.TwoBitPendingRequests(aggregateCtx, &reportertypes.QueryTwoBitPendingRequestsRequest{
		ReporterAddress: reporter.String(),
	})
	require.NoError(err)
	require.Empty(clearedPending.Requests)
	_, err = oracleMsg.SubmitValue(aggregateCtx, submit)
	require.ErrorIs(err, collections.ErrNotFound)
}
