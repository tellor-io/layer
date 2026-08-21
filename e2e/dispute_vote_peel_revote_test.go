package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/e2e"
	layerutil "github.com/tellor-io/layer/testutil"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	peelRevoteDelegateAmt = math.NewInt(1_000 * 1e6)
	peelRevoteTipAmt      = math.NewInt(1 * 1e6)
)

const (
	peelRevoteVoteGas  = "500000"
	peelRevoteVoteFees = "50loya"
)

type reporterTallySnapshot struct {
	totalPowerVoted uint64
	supportPct      float64
	againstPct      float64
	invalidPct      float64
}

func parseTallyPercent(raw string) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	raw = strings.TrimSuffix(raw, "%")
	pct, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return pct
}

func parseTallyUint64(raw string) uint64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func fetchReporterTally(ctx context.Context, node *cosmos.ChainNode, disputeID string) (reporterTallySnapshot, error) {
	res, _, err := e2e.QueryWithTimeout(ctx, node, "dispute", "tally", disputeID)
	if err != nil {
		return reporterTallySnapshot{}, err
	}
	var tally e2e.QueryDisputesTallyResponse
	if err := json.Unmarshal(res, &tally); err != nil {
		return reporterTallySnapshot{}, err
	}
	if tally.Reporters == nil || tally.Reporters.VoteCount == nil {
		return reporterTallySnapshot{}, fmt.Errorf("missing reporters tally for dispute %s", disputeID)
	}
	return reporterTallySnapshot{
		totalPowerVoted: parseTallyUint64(tally.Reporters.TotalPowerVoted),
		supportPct:      parseTallyPercent(tally.Reporters.VoteCount.Support),
		againstPct:      parseTallyPercent(tally.Reporters.VoteCount.Against),
		invalidPct:      parseTallyPercent(tally.Reporters.VoteCount.Invalid),
	}, nil
}

func execDisputeVote(ctx context.Context, node *cosmos.ChainNode, voterAddr, disputeID, choice string) error {
	_, err := node.ExecTx(ctx, voterAddr, "dispute", "vote", disputeID, choice,
		"--keyring-dir", node.HomeDir(),
		"--gas", peelRevoteVoteGas,
		"--fees", peelRevoteVoteFees,
	)
	return err
}

func queryDelegationAmount(ctx context.Context, chain *cosmos.CosmosChain, delegator, validator string) (math.Int, error) {
	del, err := chain.StakingQueryDelegation(ctx, validator, delegator)
	if err != nil {
		return math.Int{}, err
	}
	if del == nil || !del.Balance.Amount.IsPositive() {
		return math.Int{}, fmt.Errorf("no delegation from %s to %s", delegator, validator)
	}
	return del.Balance.Amount, nil
}

func fetchUserTallyPowerVoted(ctx context.Context, node *cosmos.ChainNode, disputeID string) (uint64, error) {
	res, _, err := e2e.QueryWithTimeout(ctx, node, "dispute", "tally", disputeID)
	if err != nil {
		return 0, err
	}
	var tally e2e.QueryDisputesTallyResponse
	if err := json.Unmarshal(res, &tally); err != nil {
		return 0, err
	}
	if tally.Users == nil {
		return 0, fmt.Errorf("missing users tally for dispute %s", disputeID)
	}
	return parseTallyUint64(tally.Users.TotalPowerVoted), nil
}

func requireDisputeVoting(t *testing.T, ctx context.Context, node *cosmos.ChainNode, disputeID string) {
	t.Helper()
	res, _, err := e2e.QueryWithTimeout(ctx, node, "dispute", "disputes")
	require.NoError(t, err)
	var disputes e2e.Disputes2
	require.NoError(t, json.Unmarshal(res, &disputes))
	require.NotEmpty(t, disputes.Disputes)
	var found bool
	for _, d := range disputes.Disputes {
		if d.DisputeID == disputeID {
			require.Equal(t, "DISPUTE_STATUS_VOTING", d.Metadata.DisputeStatus)
			found = true
			break
		}
	}
	require.True(t, found, "dispute %s not found", disputeID)
}

func setupPeelRevoteValidators(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain) (e2e.ValidatorInfo, e2e.ValidatorInfo) {
	t.Helper()
	require := require.New(t)
	validators, err := e2e.GetValidators(ctx, chain)
	require.NoError(err)
	require.GreaterOrEqual(len(validators), 2)
	val0, val1 := validators[0], validators[1]

	require.NoError(e2e.TurnOnMinting(ctx, chain, val0.Node))
	require.NoError(testutil.WaitForBlocks(ctx, 7, val0.Node))

	for i, val := range []e2e.ValidatorInfo{val0, val1} {
		_, err := val.Node.ExecTx(ctx, val.AccAddr, "reporter", "create-reporter", commissRate, "1000000",
			fmt.Sprintf("peel_revote_%d", i), "--keyring-dir", val.Node.HomeDir())
		require.NoError(err)
	}
	require.NoError(testutil.WaitForBlocks(ctx, 2, val0.Node))
	return val0, val1
}

func fundSelectorOnReporter(
	t *testing.T,
	ctx context.Context,
	val0 e2e.ValidatorInfo,
) (selectorAddr string) {
	t.Helper()
	require := require.New(t)

	fundAmt := math.NewInt(5_000 * 1e6)
	delegateCoin := sdk.NewCoin("loya", peelRevoteDelegateAmt)
	selector := interchaintest.GetAndFundTestUsers(t, ctx, "peel_revote_selector", fundAmt, val0.Node.Chain)[0]
	selectorAddr = selector.FormattedAddress()

	_, err := val0.Node.ExecTx(ctx, selectorAddr, "staking", "delegate", val0.ValAddr, delegateCoin.String(),
		"--keyring-dir", val0.Node.HomeDir(), "--fees", "10loya")
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 2, val0.Node))

	_, err = val0.Node.ExecTx(ctx, selectorAddr, "reporter", "select-reporter", val0.AccAddr,
		"--keyring-dir", val0.Node.HomeDir(), "--fees", "5loya")
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 2, val0.Node))
	return selectorAddr
}

func submitSpotPriceReport(t *testing.T, ctx context.Context, val e2e.ValidatorInfo, qData string, tip sdk.Coin) {
	t.Helper()
	require := require.New(t)
	spotValue := layerutil.EncodeValue(50000.99)

	require.NoError(testutil.WaitForBlocks(ctx, 1, val.Node))
	_, _, err := val.Node.Exec(ctx,
		val.Node.TxCommand(val.AccAddr, "oracle", "tip", qData, tip.String(), "--fees", "25loya", "--keyring-dir", val.Node.HomeDir()),
		val.Node.Chain.Config().Env,
	)
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 1, val.Node))
	_, err = val.Node.ExecTx(ctx, val.AccAddr, "oracle", "submit-value", qData, spotValue,
		"--keyring-dir", val.Node.HomeDir(), "--gas", "500000", "--fees", "25loya")
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 2, val.Node))
}

func latestReporterReport(t *testing.T, ctx context.Context, node *cosmos.ChainNode, reporterAddr string) e2e.MicroReport {
	t.Helper()
	require := require.New(t)
	res, _, err := e2e.QueryWithTimeout(ctx, node, "oracle", "get-reportsby-reporter", reporterAddr, "--page-limit", "1", "--page-reverse")
	require.NoError(err)
	var reports e2e.QueryMicroReportsResponse
	require.NoError(json.Unmarshal(res, &reports))
	require.NotEmpty(reports.MicroReports)
	return reports.MicroReports[0]
}

func proposeFullDispute(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
	disputerAddr string,
	report e2e.MicroReport,
	category string,
	feeFromReportPower func(uint64) math.Int,
) string {
	t.Helper()
	require := require.New(t)

	reportPower, err := strconv.ParseUint(report.Power, 10, 64)
	require.NoError(err)
	disputeFee := feeFromReportPower(reportPower)
	disputerFund := disputeFee.Add(math.NewInt(100_000 * 1e6))

	disputer := interchaintest.GetAndFundTestUsers(t, ctx, "peel_revote_disputer", disputerFund, node.Chain)[0]
	_, err = node.ExecTx(ctx, disputer.FormattedAddress(), "dispute", "propose-dispute",
		report.Reporter, report.MetaId, report.QueryID,
		category, sdk.NewCoin("loya", disputeFee).String(), notFromBond,
		"--keyring-dir", node.HomeDir(), "--gas", peelRevoteVoteGas, "--fees", peelRevoteVoteFees,
	)
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 2, node))

	res, _, err := e2e.QueryWithTimeout(ctx, node, "dispute", "disputes")
	require.NoError(err)
	var disputes e2e.Disputes2
	require.NoError(json.Unmarshal(res, &disputes))
	require.Len(disputes.Disputes, 1)
	require.Equal("DISPUTE_STATUS_VOTING", disputes.Disputes[0].Metadata.DisputeStatus)
	return disputes.Disputes[0].DisputeID
}

func tipSelector(t *testing.T, ctx context.Context, val e2e.ValidatorInfo, selectorAddr, qData string, amount sdk.Coin) {
	t.Helper()
	require := require.New(t)
	_, _, err := val.Node.Exec(ctx,
		val.Node.TxCommand(selectorAddr, "oracle", "tip", qData, amount.String(), "--fees", "25loya", "--keyring-dir", val.Node.HomeDir()),
		val.Node.Chain.Config().Env,
	)
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 2, val.Node))
}

func finalizeSwitchToReporter(t *testing.T, ctx context.Context, val0, val1 e2e.ValidatorInfo, selectorAddr string) {
	t.Helper()
	require := require.New(t)
	tip := sdk.NewCoin("loya", peelRevoteTipAmt)

	_, err := val0.Node.ExecTx(ctx, selectorAddr, "reporter", "switch-reporter", val1.AccAddr,
		"--keyring-dir", val0.Node.HomeDir(), "--fees", "5loya")
	require.NoError(err)
	require.NoError(testutil.WaitForBlocks(ctx, 2, val0.Node))

	resPending, _, err := e2e.QueryWithTimeout(ctx, val0.Node, "reporter", "selector-reporter", selectorAddr)
	require.NoError(err)
	var selectorPending e2e.QuerySelectorReporterResponse
	require.NoError(json.Unmarshal(resPending, &selectorPending))
	require.Equal(val0.AccAddr, selectorPending.Reporter, "switch must stay pending until val1 reports")

	submitSpotPriceReport(t, ctx, val1, xrpQData, tip)

	res, _, err := e2e.QueryWithTimeout(ctx, val0.Node, "reporter", "selector-reporter", selectorAddr)
	require.NoError(err)
	var selectorRes e2e.QuerySelectorReporterResponse
	require.NoError(json.Unmarshal(res, &selectorRes))
	require.Equal(val1.AccAddr, selectorRes.Reporter, "val1 SpotPrice report must finalize the pending switch")
}

// TestDisputeVotePeelRevoteE2E runs reporter vote → selector peel → reporter revote on a live chain
// and asserts reporter-group tallies conserve token totals without wrapping buckets.
func TestDisputeVotePeelRevoteE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping peel/revote e2e in short mode")
	}

	require := require.New(t)
	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := e2e.SetupChain(t, 2, 0)
	defer ic.Close()

	val0, _ := setupPeelRevoteValidators(t, ctx, chain)
	selectorAddr := fundSelectorOnReporter(t, ctx, val0)
	tip := sdk.NewCoin("loya", peelRevoteTipAmt)

	submitSpotPriceReport(t, ctx, val0, bnbQData, tip)
	disputedReport := latestReporterReport(t, ctx, val0.Node, val0.AccAddr)
	// Warning disputes enter voting immediately but do not dispute-lock selectors
	// (minor jail is 600s and blocks selector peel/reporter-stake votes).
	disputeID := proposeFullDispute(t, ctx, val0.Node, val0.AccAddr, disputedReport, "warning", warningDisputeFeeFromReportPower)

	require.NoError(execDisputeVote(ctx, val0.Node, val0.AccAddr, disputeID, "vote-support"))
	require.NoError(testutil.WaitForBlocks(ctx, 2, val0.Node))

	afterReporter, err := fetchReporterTally(ctx, val0.Node, disputeID)
	require.NoError(err)
	require.Greater(afterReporter.totalPowerVoted, uint64(0), "reporter first vote must add reporter-group power")
	require.Greater(afterReporter.supportPct, 0.0)
	require.Zero(afterReporter.againstPct)
	require.Zero(afterReporter.invalidPct)

	require.NoError(execDisputeVote(ctx, val0.Node, selectorAddr, disputeID, "vote-against"))
	require.NoError(testutil.WaitForBlocks(ctx, 2, val0.Node))

	afterPeel, err := fetchReporterTally(ctx, val0.Node, disputeID)
	require.NoError(err)
	require.Equal(afterReporter.totalPowerVoted, afterPeel.totalPowerVoted,
		"peel must conserve reporter-group token totals")
	// Tally percentages use chain-wide bonded stake as the denominator, so a selector
	// peel can be too small to show up in the 2-decimal AGAINST bucket even when it
	// moved real reporter stake (fundSelectorOnReporter delegates peelRevoteDelegateAmt).
	require.Zero(afterPeel.invalidPct)

	require.NoError(execDisputeVote(ctx, val0.Node, val0.AccAddr, disputeID, "vote-invalid"))
	require.NoError(testutil.WaitForBlocks(ctx, 2, val0.Node))

	final, err := fetchReporterTally(ctx, val0.Node, disputeID)
	require.NoError(err)
	require.Equal(afterPeel.totalPowerVoted, final.totalPowerVoted,
		"reporter revote must conserve reporter-group token totals")
	require.Less(final.supportPct, afterReporter.supportPct,
		"reporter revote power must exclude selector stake peeled to AGAINST")
	require.Less(final.supportPct, 0.01, "revote must zero SUPPORT without bucket wrap")
	require.Greater(final.invalidPct, afterPeel.invalidPct,
		"reporter revote must move remaining reporter stake to INVALID")

	requireDisputeVoting(t, ctx, val0.Node, disputeID)
}

// TestDisputeVotePeelRevoteSwitchLockE2E covers pending switch finalization, dispute-lock on the
// selector, a locked first vote funded by tips only, unlock, then selector revote that must peel
// reporter stake instead of wrapping buckets.
func TestDisputeVotePeelRevoteSwitchLockE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping switch/lock peel/revote e2e in short mode")
	}

	require := require.New(t)
	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := e2e.SetupChain(t, 2, 0)
	defer ic.Close()

	val0, val1 := setupPeelRevoteValidators(t, ctx, chain)
	selectorAddr := fundSelectorOnReporter(t, ctx, val0)
	tip := sdk.NewCoin("loya", peelRevoteTipAmt)

	submitSpotPriceReport(t, ctx, val0, bnbQData, tip)
	disputedReport := latestReporterReport(t, ctx, val0.Node, val0.AccAddr)

	finalizeSwitchToReporter(t, ctx, val0, val1, selectorAddr)

	// Tip before opening the dispute so user-group power is available at the dispute block.
	tipSelector(t, ctx, val0, selectorAddr, bnbQData, tip)

	disputeID := proposeFullDispute(t, ctx, val0.Node, val0.AccAddr, disputedReport, "minor", minorDisputeFeeFromReportPower)

	require.NoError(execDisputeVote(ctx, val0.Node, val0.AccAddr, disputeID, "vote-against"))
	require.NoError(testutil.WaitForBlocks(ctx, 2, val0.Node))

	afterReporter, err := fetchReporterTally(ctx, val0.Node, disputeID)
	require.NoError(err)
	require.Greater(afterReporter.totalPowerVoted, uint64(0))
	require.Greater(afterReporter.againstPct, 0.0)

	userPowerBeforeLock, err := fetchUserTallyPowerVoted(ctx, val0.Node, disputeID)
	require.NoError(err)

	require.NoError(execDisputeVote(ctx, val0.Node, selectorAddr, disputeID, "vote-support"))
	require.NoError(testutil.WaitForBlocks(ctx, 2, val0.Node))

	userPowerAfterLock, err := fetchUserTallyPowerVoted(ctx, val0.Node, disputeID)
	require.NoError(err)
	lockedTipPower := userPowerAfterLock - userPowerBeforeLock
	require.Greater(lockedTipPower, uint64(0), "locked selector must still vote with tip power")
	require.Less(lockedTipPower, peelRevoteDelegateAmt.Uint64(),
		"dispute-locked selector first vote must not include reporter stake")

	afterLockedVote, err := fetchReporterTally(ctx, val0.Node, disputeID)
	require.NoError(err)
	require.Equal(afterReporter.totalPowerVoted, afterLockedVote.totalPowerVoted,
		"dispute-locked selector must not add reporter-group stake on first vote")
	require.InDelta(afterReporter.againstPct, afterLockedVote.againstPct, 0.05,
		"locked selector vote must not move reporter AGAINST bucket")

	require.Greater(userPowerAfterLock, uint64(0),
		"locked selector vote with tips must still count user-group power")

	// Minor jail / dispute lock duration is 600s of chain time; existing e2e uses wall-clock sleep.
	t.Log("waiting 601s for dispute lock expiry")
	time.Sleep(601 * time.Second)
	require.NoError(testutil.WaitForBlocks(ctx, 1, val0.Node))

	// Full minor fee escrows a proportional share (~5%) of each report token origin,
	// so remaining bonded stake is below the pre-dispute delegation amount.
	selectorDelegation, err := queryDelegationAmount(ctx, chain, selectorAddr, val0.ValAddr)
	require.NoError(err)
	require.True(selectorDelegation.IsPositive(),
		"post-unlock selector must still hold remaining delegated stake (got %s)", selectorDelegation)
	require.True(selectorDelegation.LT(peelRevoteDelegateAmt),
		"minor dispute escrow should reduce selector bonded stake below pre-dispute amount")

	require.NoError(execDisputeVote(ctx, val0.Node, selectorAddr, disputeID, "vote-invalid"))
	require.NoError(testutil.WaitForBlocks(ctx, 2, val0.Node))

	final, err := fetchReporterTally(ctx, val0.Node, disputeID)
	require.NoError(err)
	// Authoritative check: without peeling from the evidence reporter, unlock would
	// ADD current-selection stake and inflate TotalPowerVoted by ~selector tokens.
	require.Equal(afterReporter.totalPowerVoted, final.totalPowerVoted,
		"post-unlock selector revote must conserve reporter-group token totals")
	require.Less(final.supportPct, 0.01,
		"ReporterPower=0 on locked first vote must not leave phantom SUPPORT after peel revote")
	// %-display uses (choice / TotalReporterPower) * 100/3; a ~1e9 peel can round to
	// 0.00% and is covered by conservation above plus keeper unit Finding2b.

	requireDisputeVoting(t, ctx, val0.Node, disputeID)
}

// TestDisputeVoteThenSwitchThenRevoteE2E covers selector peel while on the
// evidence reporter, switch finalization to another reporter, then selector
// revote. Reporter-group totals must stay conserved with no phantom add against
// the new reporter — proving switch finalize needs no open-dispute housekeeping
// beyond the stored-power revote path.
func TestDisputeVoteThenSwitchThenRevoteE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping vote-then-switch-then-revote e2e in short mode")
	}

	require := require.New(t)
	cosmos.SetSDKConfig("tellor")

	chain, ic, ctx := e2e.SetupChain(t, 2, 0)
	defer ic.Close()

	val0, val1 := setupPeelRevoteValidators(t, ctx, chain)
	selectorAddr := fundSelectorOnReporter(t, ctx, val0)
	tip := sdk.NewCoin("loya", peelRevoteTipAmt)

	submitSpotPriceReport(t, ctx, val0, bnbQData, tip)
	disputedReport := latestReporterReport(t, ctx, val0.Node, val0.AccAddr)
	disputeID := proposeFullDispute(t, ctx, val0.Node, val0.AccAddr, disputedReport, "warning", warningDisputeFeeFromReportPower)

	require.NoError(execDisputeVote(ctx, val0.Node, val0.AccAddr, disputeID, "vote-support"))
	require.NoError(testutil.WaitForBlocks(ctx, 2, val0.Node))

	afterReporter, err := fetchReporterTally(ctx, val0.Node, disputeID)
	require.NoError(err)
	require.Greater(afterReporter.totalPowerVoted, uint64(0))

	require.NoError(execDisputeVote(ctx, val0.Node, selectorAddr, disputeID, "vote-against"))
	require.NoError(testutil.WaitForBlocks(ctx, 2, val0.Node))

	afterPeel, err := fetchReporterTally(ctx, val0.Node, disputeID)
	require.NoError(err)
	require.Equal(afterReporter.totalPowerVoted, afterPeel.totalPowerVoted,
		"selector peel must conserve reporter-group token totals")

	finalizeSwitchToReporter(t, ctx, val0, val1, selectorAddr)

	require.NoError(execDisputeVote(ctx, val0.Node, selectorAddr, disputeID, "vote-invalid"))
	require.NoError(testutil.WaitForBlocks(ctx, 2, val0.Node))

	final, err := fetchReporterTally(ctx, val0.Node, disputeID)
	require.NoError(err)
	// Authoritative check: a buggy post-switch first-vote path would ADD against
	// the new reporter and inflate TotalPowerVoted. Conservation proves the
	// stored-power revote path ran instead.
	require.Equal(afterPeel.totalPowerVoted, final.totalPowerVoted,
		"post-switch selector revote must conserve reporter-group token totals")
	require.InDelta(afterPeel.supportPct, final.supportPct, 0.05,
		"evidence reporter SUPPORT must be unchanged by selector's post-switch revote")
	// %-display uses (choice / TotalReporterPower) * 100/3; a ~1e9 peel can round
	// to 0.00% in both AGAINST and INVALID, so bucket movement is covered by
	// conservation above plus keeper TestVoteThenSwitchThenRevote / MsgVote twin.

	requireDisputeVoting(t, ctx, val0.Node, disputeID)
}
