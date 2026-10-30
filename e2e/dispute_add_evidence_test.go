package e2e_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	layerutil "github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/utils"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupAddEvidenceChain(t *testing.T) (*cosmos.CosmosChain, *interchaintest.Interchain, context.Context, []e2e.ValidatorInfo) {
	t.Helper()
	cosmos.SetSDKConfig("tellor")
	chain, ic, ctx := e2e.SetupChain(t, 2, 0)
	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(t, err)
	require.Len(t, validators, 2)
	require.NoError(t, e2e.TurnOnMinting(ctx, chain, validators[0].Node))
	require.NoError(t, testutil.WaitForBlocks(ctx, 6, validators[0].Node))
	for i, val := range validators {
		_, err := val.Node.ExecTx(ctx, val.AccAddr,
			"reporter", "create-reporter", "0.1", "1000000", fmt.Sprintf("add_evidence_%d", i),
			"--keyring-dir", val.Node.HomeDir(),
			"--gas", "500000", "--fees", "50loya",
		)
		require.NoError(t, err)
	}
	return chain, ic, ctx, validators
}

func tipAndSubmit(t *testing.T, ctx context.Context, tipper e2e.ValidatorInfo, reporters []e2e.ValidatorInfo, queryData string) {
	t.Helper()
	// SpotPrice ReportBlockWindow is 2; sequential ExecTx submits can straddle aggregation
	// and hit ErrNotTokenDeposit. Tip with ExecTx, then broadcast multi-reporter submits via
	// Exec so they land in the same window (same pattern as dispute_test). Single-reporter
	// submits use ExecTx so on-chain failures surface instead of looking like empty reports.
	// Prefer non-cyclelist queryData (not eth/btc/trb): tipping an active cyclelist query
	// does not refresh Expiration, so submits can miss the short window.
	tip := sdk.NewCoin("loya", math.NewInt(1_000_000))
	_, err := tipper.Node.ExecTx(ctx, tipper.AccAddr, "oracle", "tip", queryData, tip.String(),
		"--keyring-dir", tipper.Node.HomeDir())
	require.NoError(t, err)

	value := layerutil.EncodeValue(10000000.99)
	if len(reporters) == 1 {
		_, err = reporters[0].Node.ExecTx(ctx, reporters[0].AccAddr, "oracle", "submit-value", queryData, value,
			"--keyring-dir", reporters[0].Node.HomeDir())
		require.NoError(t, err)
	} else {
		for _, reporter := range reporters {
			_, _, err = reporter.Node.Exec(ctx,
				reporter.Node.TxCommand(reporter.AccAddr, "oracle", "submit-value", queryData, value,
					"--keyring-dir", reporter.Node.HomeDir()),
				reporter.Node.Chain.Config().Env,
			)
			require.NoError(t, err)
		}
	}
	require.NoError(t, testutil.WaitForBlocks(ctx, 3, tipper.Node))
}

func latestReportForReporter(t *testing.T, ctx context.Context, node *cosmos.ChainNode, reporterAddr, queryData string) e2e.MicroReport {
	t.Helper()
	queryIDBytes, err := utils.QueryIDFromDataString(queryData)
	require.NoError(t, err)
	queryID := hex.EncodeToString(queryIDBytes)
	res, _, err := e2e.QueryWithTimeout(ctx, node, "oracle", "get-reportsby-reporter-qid", reporterAddr, queryID, "--page-limit", "1", "--page-reverse")
	require.NoError(t, err)
	var reports e2e.QueryMicroReportsResponse
	require.NoError(t, json.Unmarshal(res, &reports))
	require.NotEmpty(t, reports.MicroReports)
	return reports.MicroReports[0]
}

func currentAggregateReporter(t *testing.T, ctx context.Context, node *cosmos.ChainNode, queryData string) string {
	t.Helper()
	queryIDBytes, err := utils.QueryIDFromDataString(queryData)
	require.NoError(t, err)
	res, _, err := e2e.QueryWithTimeout(ctx, node, "oracle", "get-current-aggregate-report", hex.EncodeToString(queryIDBytes))
	require.NoError(t, err)
	var agg e2e.QueryGetCurrentAggregateReportResponse
	require.NoError(t, json.Unmarshal(res, &agg))
	require.NotNil(t, agg.Aggregate)
	return agg.Aggregate.AggregateReporter
}

func microReportFromQuery(t *testing.T, report e2e.MicroReport) oracletypes.MicroReport {
	t.Helper()
	queryID, err := hex.DecodeString(report.QueryID)
	require.NoError(t, err)
	power, err := strconv.ParseUint(report.Power, 10, 64)
	require.NoError(t, err)
	blockNumber, err := strconv.ParseUint(report.BlockNumber, 10, 64)
	require.NoError(t, err)
	metaID, err := strconv.ParseUint(report.MetaId, 10, 64)
	require.NoError(t, err)
	tsMillis, err := strconv.ParseInt(report.Timestamp, 10, 64)
	require.NoError(t, err)
	return oracletypes.MicroReport{
		Reporter:        report.Reporter,
		Power:           power,
		QueryType:       report.QueryType,
		QueryId:         queryID,
		AggregateMethod: report.AggregateMethod,
		Value:           report.Value,
		Timestamp:       time.UnixMilli(tsMillis).UTC(),
		BlockNumber:     blockNumber,
		MetaId:          metaID,
		Cyclelist:       true,
	}
}

func broadcastAddEvidence(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, disputeID uint64, report oracletypes.MicroReport) error {
	t.Helper()
	b := cosmos.NewBroadcaster(t, chain)
	b.ConfigureFactoryOptions(func(f tx.Factory) tx.Factory {
		return f.WithGas(1_000_000).WithGasAdjustment(1.5)
	})
	resp, err := cosmos.BroadcastTx(ctx, b, user, &disputetypes.MsgAddEvidence{
		CallerAddress: user.FormattedAddress(),
		DisputeId:     disputeID,
		Reports:       []*oracletypes.MicroReport{&report},
	})
	if err != nil {
		return err
	}
	// BroadcastTx returns nil err even when DeliverTx fails; surface on-chain failures.
	if resp.Code != 0 {
		return fmt.Errorf("add evidence tx failed (code %d): %s", resp.Code, resp.RawLog)
	}
	return nil
}

func queryDisputes(t *testing.T, ctx context.Context, node *cosmos.ChainNode) e2e.Disputes2 {
	t.Helper()
	res, _, err := e2e.QueryWithTimeout(ctx, node, "dispute", "disputes")
	require.NoError(t, err)
	var disputes e2e.Disputes2
	require.NoError(t, json.Unmarshal(res, &disputes))
	return disputes
}

func TestAddEvidence_WarningRejectsAggregate(t *testing.T) {
	require := require.New(t)
	chain, ic, ctx, validators := setupAddEvidenceChain(t)
	defer ic.Close()
	val0, val1 := validators[0], validators[1]

	// both reporters submit on bch so one becomes the aggregate reporter
	tipAndSubmit(t, ctx, val0, []e2e.ValidatorInfo{val0, val1}, bchQData)
	aggReporter := currentAggregateReporter(t, ctx, val0.Node, bchQData)
	nonAggReporter := val0
	if aggReporter == val0.AccAddr {
		nonAggReporter = val1
	}
	nonAggReport := latestReportForReporter(t, ctx, val0.Node, nonAggReporter.AccAddr, bchQData)
	require.NotEqual(aggReporter, nonAggReport.Reporter)

	disputer := interchaintest.GetAndFundTestUsers(t, ctx, "add_evidence_disputer", math.NewInt(5_000_000_000), chain)[0]
	_, err := val0.Node.ExecTx(ctx, disputer.FormattedAddress(),
		"dispute", "propose-dispute", nonAggReport.Reporter, nonAggReport.MetaId, nonAggReport.QueryID,
		"warning", "1000000000loya", "false",
		"--keyring-dir", val0.Node.HomeDir(), "--gas", "1000000", "--fees", "500loya",
	)
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 1, val0.Node))

	disputes := queryDisputes(t, ctx, val0.Node)
	require.Len(disputes.Disputes, 1)
	disputeID, err := strconv.ParseUint(disputes.Disputes[0].DisputeID, 10, 64)
	require.NoError(err)

	// make the disputed reporter the aggregate reporter on a different query
	tipAndSubmit(t, ctx, val0, []e2e.ValidatorInfo{nonAggReporter}, ltcQData)
	aggEvidence := microReportFromQuery(t, latestReportForReporter(t, ctx, val0.Node, nonAggReporter.AccAddr, ltcQData))
	require.Equal(nonAggReporter.AccAddr, currentAggregateReporter(t, ctx, val0.Node, ltcQData))

	err = broadcastAddEvidence(t, ctx, chain, disputer, disputeID, aggEvidence)
	require.Error(err)
	require.Contains(err.Error(), "warning disputes cannot add aggregate reports as evidence")

	disputes = queryDisputes(t, ctx, val0.Node)
	require.Len(disputes.Disputes, 1)
	require.Empty(disputes.Disputes[0].Metadata.AdditionalEvidence)
}

func TestAddEvidence_MinorAggregateLimit(t *testing.T) {
	require := require.New(t)
	chain, ic, ctx, validators := setupAddEvidenceChain(t)
	defer ic.Close()
	val0, val1 := validators[0], validators[1]

	tipAndSubmit(t, ctx, val0, []e2e.ValidatorInfo{val0, val1}, bchQData)
	aggReporter := currentAggregateReporter(t, ctx, val0.Node, bchQData)
	nonAggReporter := val0
	if aggReporter == val0.AccAddr {
		nonAggReporter = val1
	}
	nonAggReport := latestReportForReporter(t, ctx, val0.Node, nonAggReporter.AccAddr, bchQData)

	disputer := interchaintest.GetAndFundTestUsers(t, ctx, "add_evidence_minor", math.NewInt(5_000_000_000), chain)[0]
	_, err := val0.Node.ExecTx(ctx, disputer.FormattedAddress(),
		"dispute", "propose-dispute", nonAggReport.Reporter, nonAggReport.MetaId, nonAggReport.QueryID,
		"minor", "1000000000loya", "false",
		"--keyring-dir", val0.Node.HomeDir(), "--gas", "1000000", "--fees", "500loya",
	)
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 1, val0.Node))

	disputes := queryDisputes(t, ctx, val0.Node)
	require.Len(disputes.Disputes, 1)
	disputeID, err := strconv.ParseUint(disputes.Disputes[0].DisputeID, 10, 64)
	require.NoError(err)

	// first aggregate evidence is allowed
	tipAndSubmit(t, ctx, val0, []e2e.ValidatorInfo{nonAggReporter}, ltcQData)
	firstAgg := microReportFromQuery(t, latestReportForReporter(t, ctx, val0.Node, nonAggReporter.AccAddr, ltcQData))
	err = broadcastAddEvidence(t, ctx, chain, disputer, disputeID, firstAgg)
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 1, val0.Node))

	disputes = queryDisputes(t, ctx, val0.Node)
	require.Len(disputes.Disputes[0].Metadata.AdditionalEvidence, 1)

	// second aggregate evidence is rejected (usdc is not in InitialCycleList; eth/btc/trb are)
	usdcQData := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004757364630000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	tipAndSubmit(t, ctx, val0, []e2e.ValidatorInfo{nonAggReporter}, usdcQData)
	secondAgg := microReportFromQuery(t, latestReportForReporter(t, ctx, val0.Node, nonAggReporter.AccAddr, usdcQData))
	err = broadcastAddEvidence(t, ctx, chain, disputer, disputeID, secondAgg)
	require.Error(err)
	require.Contains(err.Error(), "minor disputes can flag at most one aggregate report")

	disputes = queryDisputes(t, ctx, val0.Node)
	require.Len(disputes.Disputes[0].Metadata.AdditionalEvidence, 1)
}
