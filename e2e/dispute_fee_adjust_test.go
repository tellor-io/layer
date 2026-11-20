package e2e_test

import (
	"context"
	"encoding/json"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	layerutil "github.com/tellor-io/layer/testutil"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TestCollectedSlashResizesRoundOneFee is the live CLI check for a round-1 major
// fee resized to the reporter stake a prior dispute already escrowed.
//
// The reporter bonds 1000 TRB and reports with power 1000. The disputer bonds
// 2000 TRB, more than a 50 TRB minor fee plus the 1000 TRB major quote, because
// that quote is taken from bond before the excess is returned. The minor dispute
// keeps its quote. The major dispute keeps only the reporter stake it can escrow.
func TestCollectedSlashResizesRoundOneFee(t *testing.T) {
	require := require.New(t)
	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := e2e.SetupChain(t, 2, 0)
	defer ic.Close()

	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	val := validators[0]

	const reportPower uint64 = 1000
	minorFee := minorDisputeFeeFromReportPower(reportPower)
	majorQuote := disputeFeeFromReportPower(reportPower, 100)
	require.True(minorFee.Equal(math.NewInt(50_000_000)))
	require.True(majorQuote.Equal(math.NewInt(1_000_000_000)))

	fundAmt := math.NewInt(10_000 * 1e6)
	reporterStake := sdk.NewCoin("loya", math.NewInt(1_000*1e6))
	// Bond must cover the minor fee and the full major quote before the excess returns.
	disputerStake := sdk.NewCoin("loya", math.NewInt(2_000*1e6))
	require.True(disputerStake.Amount.GT(minorFee.Add(majorQuote)))

	reporter := interchaintest.GetAndFundTestUsers(t, ctx, "fee_adjust_reporter", fundAmt, chain)[0]
	disputer := interchaintest.GetAndFundTestUsers(t, ctx, "fee_adjust_disputer", fundAmt, chain)[0]
	reporterAddr := reporter.FormattedAddress()
	disputerAddr := disputer.FormattedAddress()

	_, err = val.Node.ExecTx(ctx, reporterAddr, "staking", "delegate", val.ValAddr, reporterStake.String(), "--keyring-dir", val.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)
	_, err = val.Node.ExecTx(ctx, disputerAddr, "staking", "delegate", val.ValAddr, disputerStake.String(), "--keyring-dir", val.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)

	require.NoError(e2e.TurnOnMinting(ctx, chain, val.Node))
	require.NoError(testutil.WaitForBlocks(ctx, 5, val.Node))

	for _, user := range []struct{ addr, moniker string }{
		{reporterAddr, "fee_adjust_reporter"},
		{disputerAddr, "fee_adjust_disputer"},
	} {
		_, err = val.Node.ExecTx(ctx, user.addr, "reporter", "create-reporter", commissRate, "1000000", user.moniker, "--keyring-dir", val.Node.HomeDir())
		require.NoError(err)
	}

	tip := sdk.NewCoin("loya", math.NewInt(1_000_000))
	_, _, err = val.Node.Exec(ctx, val.Node.TxCommand(disputerAddr, "oracle", "tip", bchQData, tip.String(), "--keyring-dir", val.Node.HomeDir()), val.Node.Chain.Config().Env)
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 1, val.Node))

	value := layerutil.EncodeValue(10000000.99)
	_, err = val.Node.ExecTx(ctx, reporterAddr, "oracle", "submit-value", bchQData, value, "--keyring-dir", val.Node.HomeDir())
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 1, val.Node))
	require.NoError(testutil.WaitForBlocks(ctx, 2, val.Node))

	res, _, err := e2e.QueryWithTimeout(ctx, val.Node, "oracle", "get-reportsby-reporter", reporterAddr, "--page-limit", "1")
	require.NoError(err)
	var reports e2e.QueryMicroReportsResponse
	require.NoError(json.Unmarshal(res, &reports))
	require.NotEmpty(reports.MicroReports)
	report := reports.MicroReports[0]
	require.Equal(reporterAddr, report.Reporter)
	require.Equal("1000", report.Power)
	require.Equal("SpotPrice", report.QueryType)

	reporterBefore := delegationTokens(t, ctx, chain, reporterAddr, val.ValAddr)
	disputerBefore := delegationTokens(t, ctx, chain, disputerAddr, val.ValAddr)
	require.Equal(reporterStake.Amount.String(), reporterBefore.String())
	require.Equal(disputerStake.Amount.String(), disputerBefore.String())

	_, err = val.Node.ExecTx(ctx, disputerAddr, "dispute", "propose-dispute",
		report.Reporter, report.MetaId, report.QueryID, "minor", sdk.NewCoin("loya", minorFee).String(), "true",
		"--keyring-dir", val.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)

	minor := disputeByCategory(t, queryDisputes(t, ctx, val.Node), "DISPUTE_CATEGORY_MINOR")
	require.Equal("DISPUTE_STATUS_VOTING", minor.DisputeStatus)
	require.Equal("1", minor.DisputeRound)
	require.Equal(minorFee.String(), minor.FeeTotal)
	require.Equal(minor.FeeTotal, minor.DisputeFee)
	require.Equal(minor.FeeTotal, minor.SlashAmount)

	reporterAfterMinor := delegationTokens(t, ctx, chain, reporterAddr, val.ValAddr)
	require.True(reporterAfterMinor.IsPositive())
	require.True(reporterAfterMinor.LT(reporterBefore), "minor dispute should escrow part of the reporter stake")

	_, err = val.Node.ExecTx(ctx, disputerAddr, "dispute", "propose-dispute",
		report.Reporter, report.MetaId, report.QueryID, "major", sdk.NewCoin("loya", majorQuote).String(), "true",
		"--keyring-dir", val.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)

	disputes := queryDisputes(t, ctx, val.Node)
	minor = disputeByCategory(t, disputes, "DISPUTE_CATEGORY_MINOR")
	require.Equal("DISPUTE_STATUS_VOTING", minor.DisputeStatus)
	require.Equal(minorFee.String(), minor.FeeTotal)
	require.Equal(minor.FeeTotal, minor.DisputeFee)
	require.Equal(minor.FeeTotal, minor.SlashAmount)

	major := disputeByCategory(t, disputes, "DISPUTE_CATEGORY_MAJOR")
	require.Equal("DISPUTE_STATUS_VOTING", major.DisputeStatus)
	require.Equal("1", major.DisputeRound)
	require.Equal(major.FeeTotal, major.DisputeFee)
	require.Equal(major.FeeTotal, major.SlashAmount)

	retained := mustParseLoya(t, major.FeeTotal)
	removed := reporterAfterMinor.Sub(delegationTokens(t, ctx, chain, reporterAddr, val.ValAddr))
	require.True(retained.IsPositive())
	require.True(retained.LT(majorQuote), "major fee %s should be below the quote %s", retained, majorQuote)
	require.Equal(removed.String(), retained.String(), "major fee should equal reporter stake removed by this dispute")

	burn := math.LegacyNewDecFromInt(retained).Quo(math.LegacyNewDec(20)).TruncateInt()
	require.Equal(burn.String(), major.BurnAmount)

	// The excess major fee is back on the disputer's delegation in this same transaction.
	dropped := disputerBefore.Sub(delegationTokens(t, ctx, chain, disputerAddr, val.ValAddr))
	want := minorFee.Add(retained)
	require.True(dropped.Sub(want).Abs().LTE(math.OneInt()),
		"disputer delegation fell %s, want minor fee plus retained major fee %s", dropped, want)
	require.True(dropped.LT(minorFee.Add(majorQuote)),
		"disputer delegation fell %s; minor fee plus the full major quote is %s", dropped, minorFee.Add(majorQuote))
}

// TestAdjustedMajorFeeSupportWithdraw settles the resized major dispute.
// Team and the disputer vote support, the end blocker executes, and the disputer
// withdraws the kept fee minus its 5% burn plus the escrowed reporter stake.
func TestAdjustedMajorFeeSupportWithdraw(t *testing.T) {
	require := require.New(t)
	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := e2e.SetupChain(t, 2, 0)
	defer ic.Close()

	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	val := validators[0]

	const reportPower uint64 = 1000
	minorFee := minorDisputeFeeFromReportPower(reportPower)
	majorQuote := disputeFeeFromReportPower(reportPower, 100)

	fundAmt := math.NewInt(10_000 * 1e6)
	reporterStake := sdk.NewCoin("loya", math.NewInt(1_000*1e6))
	disputerStake := sdk.NewCoin("loya", math.NewInt(2_000*1e6))

	reporter := interchaintest.GetAndFundTestUsers(t, ctx, "fee_settle_reporter", fundAmt, chain)[0]
	disputer := interchaintest.GetAndFundTestUsers(t, ctx, "fee_settle_disputer", fundAmt, chain)[0]
	reporterAddr := reporter.FormattedAddress()
	disputerAddr := disputer.FormattedAddress()

	_, err = val.Node.ExecTx(ctx, reporterAddr, "staking", "delegate", val.ValAddr, reporterStake.String(), "--keyring-dir", val.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)
	_, err = val.Node.ExecTx(ctx, disputerAddr, "staking", "delegate", val.ValAddr, disputerStake.String(), "--keyring-dir", val.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)

	require.NoError(e2e.TurnOnMinting(ctx, chain, val.Node))
	require.NoError(testutil.WaitForBlocks(ctx, 5, val.Node))

	for _, user := range []struct{ addr, moniker string }{
		{reporterAddr, "fee_settle_reporter"},
		{disputerAddr, "fee_settle_disputer"},
	} {
		_, err = val.Node.ExecTx(ctx, user.addr, "reporter", "create-reporter", commissRate, "1000000", user.moniker, "--keyring-dir", val.Node.HomeDir())
		require.NoError(err)
	}

	tip := sdk.NewCoin("loya", math.NewInt(1_000_000))
	_, _, err = val.Node.Exec(ctx, val.Node.TxCommand(disputerAddr, "oracle", "tip", bchQData, tip.String(), "--keyring-dir", val.Node.HomeDir()), val.Node.Chain.Config().Env)
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 1, val.Node))

	value := layerutil.EncodeValue(10000000.99)
	_, err = val.Node.ExecTx(ctx, reporterAddr, "oracle", "submit-value", bchQData, value, "--keyring-dir", val.Node.HomeDir())
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 1, val.Node))
	require.NoError(testutil.WaitForBlocks(ctx, 2, val.Node))

	res, _, err := e2e.QueryWithTimeout(ctx, val.Node, "oracle", "get-reportsby-reporter", reporterAddr, "--page-limit", "1")
	require.NoError(err)
	var reports e2e.QueryMicroReportsResponse
	require.NoError(json.Unmarshal(res, &reports))
	require.NotEmpty(reports.MicroReports)
	report := reports.MicroReports[0]

	disputerBefore := delegationTokens(t, ctx, chain, disputerAddr, val.ValAddr)
	require.Equal(disputerStake.Amount.String(), disputerBefore.String())

	_, err = val.Node.ExecTx(ctx, disputerAddr, "dispute", "propose-dispute",
		report.Reporter, report.MetaId, report.QueryID, "minor", sdk.NewCoin("loya", minorFee).String(), "true",
		"--keyring-dir", val.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)
	reporterAfterMinor := delegationTokens(t, ctx, chain, reporterAddr, val.ValAddr)

	_, err = val.Node.ExecTx(ctx, disputerAddr, "dispute", "propose-dispute",
		report.Reporter, report.MetaId, report.QueryID, "major", sdk.NewCoin("loya", majorQuote).String(), "true",
		"--keyring-dir", val.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)

	major := disputeByCategory(t, queryDisputes(t, ctx, val.Node), "DISPUTE_CATEGORY_MAJOR")
	require.Equal("DISPUTE_STATUS_VOTING", major.DisputeStatus)
	retained := mustParseLoya(t, major.FeeTotal)
	removed := reporterAfterMinor.Sub(delegationTokens(t, ctx, chain, reporterAddr, val.ValAddr))
	require.True(retained.IsPositive())
	require.True(retained.LT(majorQuote))
	require.Equal(removed.String(), retained.String())
	reporterAfterMajor := delegationTokens(t, ctx, chain, reporterAddr, val.ValAddr)

	_, err = val.Node.ExecTx(ctx, disputerAddr, "dispute", "vote", major.DisputeId, "vote-support",
		"--keyring-dir", val.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)
	_, err = val.Node.ExecTx(ctx, "team", "dispute", "vote", major.DisputeId, "vote-support",
		"--keyring-dir", val.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 2, val.Node))

	settled := disputeByCategory(t, queryDisputes(t, ctx, val.Node), "DISPUTE_CATEGORY_MAJOR")
	require.Equal("DISPUTE_STATUS_RESOLVED", settled.DisputeStatus)
	require.False(settled.PendingExecution)

	_, err = val.Node.ExecTx(ctx, disputerAddr, "dispute", "withdraw-fee-refund", disputerAddr, major.DisputeId,
		"--keyring-dir", val.Node.HomeDir(), "--gas", "500000", "--fees", "50loya")
	require.NoError(err)

	// Support returns the kept fee minus its 5% burn, plus the escrowed reporter stake.
	// The minor fee stays escrowed. start - minor + retained - burn.
	burn := math.LegacyNewDecFromInt(retained).Quo(math.LegacyNewDec(20)).TruncateInt()
	want := disputerBefore.Sub(minorFee).Add(retained).Sub(burn)
	got := delegationTokens(t, ctx, chain, disputerAddr, val.ValAddr).Add(unbondingTokens(t, ctx, chain, disputerAddr))
	require.True(got.Sub(want).Abs().LTE(math.NewInt(5)),
		"disputer stake %s, want %s (start %s, minor %s, retained %s, burn %s)", got, want, disputerBefore, minorFee, retained, burn)
	require.True(delegationTokens(t, ctx, chain, reporterAddr, val.ValAddr).Sub(reporterAfterMajor).Abs().LTE(math.OneInt()),
		"reporter stake should stay escrowed after a support result")
}

func unbondingTokens(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, delegator string) math.Int {
	t.Helper()
	ubds, err := chain.StakingQueryUnbondingDelegations(ctx, delegator)
	if err != nil {
		return math.ZeroInt()
	}
	total := math.ZeroInt()
	for _, ubd := range ubds {
		for _, entry := range ubd.Entries {
			total = total.Add(entry.Balance)
		}
	}
	return total
}

func delegationTokens(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, delegator, valAddr string) math.Int {
	t.Helper()
	dels, err := chain.StakingQueryDelegations(ctx, delegator)
	require.NoError(t, err)
	total := math.ZeroInt()
	for _, del := range dels {
		if del.Delegation.ValidatorAddress != valAddr {
			continue
		}
		total = total.Add(del.Balance.Amount)
	}
	return total
}

func disputeByCategory(t *testing.T, disputes e2e.Disputes2, category string) e2e.MetaData2 {
	t.Helper()
	var found *e2e.MetaData2
	for i := range disputes.Disputes {
		if disputes.Disputes[i].Metadata.DisputeCategory != category {
			continue
		}
		require.Nil(t, found, "more than one dispute in category %s", category)
		meta := disputes.Disputes[i].Metadata
		found = &meta
	}
	require.NotNil(t, found, "no dispute in category %s", category)
	return *found
}

func mustParseLoya(t *testing.T, amount string) math.Int {
	t.Helper()
	n, ok := math.NewIntFromString(amount)
	require.True(t, ok, "loya amount %q", amount)
	return n
}
