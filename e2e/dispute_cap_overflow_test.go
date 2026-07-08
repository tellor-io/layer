package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	layerutil "github.com/tellor-io/layer/testutil"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func setupDisputeCapOverflowChain(t *testing.T) (*cosmos.CosmosChain, *interchaintest.Interchain, context.Context) {
	t.Helper()

	config := e2e.DefaultSetupConfig()
	config.NumValidators = 2
	config.ModifyGenesis = append(e2e.CreateStandardGenesis(),
		cosmos.NewGenesisKV(e2e.MaxValidatorPowerShareGenesisKey, "0.300000000000000000"),
	)
	return e2e.SetupChainWithCustomConfig(t, config)
}

func createCapOverflowValidatorReporters(t *testing.T, ctx context.Context, validators []e2e.ValidatorInfo) {
	t.Helper()
	for i, val := range validators {
		_, err := val.Node.ExecTx(ctx, val.AccAddr,
			"reporter", "create-reporter", "0.1", "1000000", fmt.Sprintf("cap_overflow_%d", i),
			"--keyring-dir", val.Node.HomeDir(),
			"--gas", "500000", "--fees", "50loya",
		)
		require.NoError(t, err)
	}
}

func submitCapOverflowSpotReport(t *testing.T, ctx context.Context, tipper, reporter e2e.ValidatorInfo, queryData string) e2e.MicroReport {
	t.Helper()

	tip := sdk.NewCoin("loya", math.NewInt(1_000_000))
	_, _, err := tipper.Node.Exec(ctx,
		tipper.Node.TxCommand(tipper.AccAddr, "oracle", "tip", queryData, tip.String(), "--keyring-dir", tipper.Node.HomeDir()),
		tipper.Node.Chain.Config().Env,
	)
	require.NoError(t, err)
	require.NoError(t, testutil.WaitForBlocks(ctx, 1, tipper.Node))

	value := layerutil.EncodeValue(10000000.99)
	_, err = reporter.Node.ExecTx(ctx, reporter.AccAddr, "oracle", "submit-value", queryData, value, "--keyring-dir", reporter.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(t, err)
	require.NoError(t, testutil.WaitForBlocks(ctx, 3, tipper.Node))

	res, _, err := e2e.QueryWithTimeout(ctx, tipper.Node, "oracle", "get-reportsby-reporter", reporter.AccAddr, "--page-limit", "1")
	require.NoError(t, err)
	var reports e2e.QueryMicroReportsResponse
	require.NoError(t, json.Unmarshal(res, &reports))
	require.NotEmpty(t, reports.MicroReports)
	require.Equal(t, reporter.AccAddr, reports.MicroReports[0].Reporter)
	require.Equal(t, value, reports.MicroReports[0].Value)
	return reports.MicroReports[0]
}

func queryCapOverflowDisputes(t *testing.T, ctx context.Context, node *cosmos.ChainNode) e2e.Disputes2 {
	t.Helper()
	res, _, err := e2e.QueryWithTimeout(ctx, node, "dispute", "disputes")
	require.NoError(t, err)
	var disputes e2e.Disputes2
	require.NoError(t, json.Unmarshal(res, &disputes))
	return disputes
}

func sumCapOverflowUnbondingBalances(ubds []stakingtypes.UnbondingDelegation) math.Int {
	total := math.ZeroInt()
	for _, ubd := range ubds {
		for _, entry := range ubd.Entries {
			total = total.Add(entry.Balance)
		}
	}
	return total
}

func capOverflowUnbondingBalance(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, delegator string) math.Int {
	t.Helper()
	ubds, err := chain.StakingQueryUnbondingDelegations(ctx, delegator)
	if err != nil {
		return math.ZeroInt()
	}
	return sumCapOverflowUnbondingBalances(ubds)
}

func TestDisputeCapOverflowReturnsAndRefundsViaUnbonding(t *testing.T) {
	require := require.New(t)

	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := setupDisputeCapOverflowChain(t)
	defer ic.Close()

	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	require.Len(validators, 2)
	val0, val1 := validators[0], validators[1]

	require.NoError(e2e.TurnOnMinting(ctx, chain, val0.Node))
	require.NoError(testutil.WaitForBlocks(ctx, 6, val0.Node))
	createCapOverflowValidatorReporters(t, ctx, validators)

	// INVALID restoration: val1's self-delegation starts over the 30% cap.
	// The INVALID result must execute and return the slashed stake through UBD
	// overflow instead of bonding back to the over-cap original validator.
	firstReport := submitCapOverflowSpotReport(t, ctx, val0, val1, bchQData)
	reporterUbdBefore := capOverflowUnbondingBalance(t, ctx, chain, val1.AccAddr)
	_, err = val0.Node.ExecTx(ctx, val0.AccAddr,
		"dispute", "propose-dispute", firstReport.Reporter, firstReport.MetaId, firstReport.QueryID,
		"warning", "500000000000loya", "false",
		"--keyring-dir", val0.Node.HomeDir(), "--gas", "1000000", "--fees", "500loya",
	)
	require.NoError(err)
	_, err = val0.Node.ExecTx(ctx, val0.AccAddr, "dispute", "vote", "1", "vote-invalid", "--keyring-dir", val0.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)
	_, err = val0.Node.ExecTx(ctx, "team", "dispute", "vote", "1", "vote-invalid", "--keyring-dir", val0.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 1, val0.Node))

	disputes := queryCapOverflowDisputes(t, ctx, val0.Node)
	require.NotEmpty(disputes.Disputes)
	require.Equal("DISPUTE_STATUS_RESOLVED", disputes.Disputes[0].Metadata.DisputeStatus)
	require.False(disputes.Disputes[0].Metadata.PendingExecution)
	reporterUbdAfter := capOverflowUnbondingBalance(t, ctx, chain, val1.AccAddr)
	require.True(reporterUbdAfter.GT(reporterUbdBefore), "INVALID restoration overflow should create unbonding balance")

	reporterRes, _, err := e2e.QueryWithTimeout(ctx, val0.Node, "reporter", "reporter", val1.AccAddr)
	require.NoError(err)
	var reporter e2e.QueryReporterResponse
	require.NoError(json.Unmarshal(reporterRes, &reporter))
	require.NotNil(reporter.Reporter)
	require.False(reporter.Reporter.Metadata.Jailed)

	_, err = val0.Node.ExecTx(ctx, val0.AccAddr,
		"dispute", "withdraw-fee-refund", val0.AccAddr, "1",
		"--keyring-dir", val0.Node.HomeDir(), "--gas", "500000", "--fees", "50loya",
	)
	require.NoError(err)
	_, err = val0.Node.ExecTx(ctx, val0.AccAddr, "dispute", "claim-reward", "1", "--keyring-dir", val0.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)

	// SUPPORT refund: fee payer val0 is also over the 30% validator cap. The
	// withdraw path must bond only cap headroom and put overflow in the UBD queue.
	secondReport := submitCapOverflowSpotReport(t, ctx, val0, val1, ltcQData)
	_, err = val0.Node.ExecTx(ctx, val0.AccAddr,
		"dispute", "propose-dispute", secondReport.Reporter, secondReport.MetaId, secondReport.QueryID,
		"warning", "500000000000loya", "false",
		"--keyring-dir", val0.Node.HomeDir(), "--gas", "1000000", "--fees", "500loya",
	)
	require.NoError(err)
	_, err = val0.Node.ExecTx(ctx, val0.AccAddr, "dispute", "vote", "2", "vote-support", "--keyring-dir", val0.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)
	_, err = val0.Node.ExecTx(ctx, "team", "dispute", "vote", "2", "vote-support", "--keyring-dir", val0.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 1, val0.Node))

	disputes = queryCapOverflowDisputes(t, ctx, val0.Node)
	require.Len(disputes.Disputes, 2)
	require.Equal("DISPUTE_STATUS_RESOLVED", disputes.Disputes[1].Metadata.DisputeStatus)
	require.False(disputes.Disputes[1].Metadata.PendingExecution)
	feePayerUbdBefore := capOverflowUnbondingBalance(t, ctx, chain, val0.AccAddr)
	_, err = val0.Node.ExecTx(ctx, val0.AccAddr,
		"dispute", "withdraw-fee-refund", val0.AccAddr, "2",
		"--keyring-dir", val0.Node.HomeDir(), "--gas", "500000", "--fees", "50loya",
	)
	require.NoError(err)
	feePayerUbdAfter := capOverflowUnbondingBalance(t, ctx, chain, val0.AccAddr)
	require.True(feePayerUbdAfter.GT(feePayerUbdBefore), "SUPPORT fee-payer overflow should create unbonding balance")

	_, err = val0.Node.ExecTx(ctx, val0.AccAddr, "dispute", "claim-reward", "2", "--keyring-dir", val0.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)
}
