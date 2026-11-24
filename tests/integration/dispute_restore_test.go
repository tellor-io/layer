package integration_test

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	setup "github.com/tellor-io/layer/tests"
	dispute "github.com/tellor-io/layer/x/dispute"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputefork "github.com/tellor-io/layer/x/dispute/migrations/fork"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reporterfork "github.com/tellor-io/layer/x/reporter/migrations/fork"
	reporter "github.com/tellor-io/layer/x/reporter/module"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// TestRestoreResizedAppealedDispute exercises the two module-state files that
// are written by the real module exporters and read by the fork migration
// readers. The restored dispute is the final (closed) round of an appeal:
// round one was quoted at 100, recovered 60, retained F=60, and the bonded
// payer's gross receipts are 70 bond plus 30 cash. The claim is deliberately left unpaid in the
// exported state and is settled only in the fresh stores.
func TestRestoreResizedAppealedDispute(t *testing.T) {
	exportDir := t.TempDir()
	t.Chdir(exportDir)

	original := &setup.SharedSetup{}
	original.SetupTest(t)
	restored := &setup.SharedSetup{}
	restored.SetupTest(t)

	blockTime := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	original.Ctx = original.Ctx.WithBlockHeight(100).WithBlockTime(blockTime)
	restored.Ctx = restored.Ctx.WithBlockHeight(100).WithBlockTime(blockTime)

	_, originalVals, _ := original.CreateValidators(1)
	_, restoredVals, _ := restored.CreateValidators(1)
	require.Equal(t, originalVals, restoredVals, "matching staking fixtures must use the same validator")

	bondPayer := fixedAddress(0x11)
	cashPayer := fixedAddress(0x22)
	reporterAddr := fixedAddress(0x33)
	hashID := []byte("restore-resized-appealed-dispute")
	prepareBondedReceipt(t, original, bondPayer, originalVals[0], hashID)
	prepareBondedStakingState(t, restored, bondPayer, restoredVals[0])
	prepareReporterEscrow(t, original, reporterAddr, originalVals[0], 60)
	prepareReporterEscrow(t, restored, reporterAddr, restoredVals[0], 60)

	const (
		openingQuote = int64(100)
		retainedFee  = int64(60)
		grossPayment = int64(70)
		principal    = int64(60)
		appealFee    = int64(6)
		burnAmount   = int64(9) // floor(60/20) + the round-two offer of 6
	)

	// The reporter's escrow is part of the actual exported reporter state. The
	// destination deliberately has no reporter claim state yet: its 60-loya
	// escrow is only bank/staking state until the reporter fork reader imports
	// the snapshot below.
	origin := &reportertypes.TokenOriginInfo{
		DelegatorAddress: reporterAddr.Bytes(),
		ValidatorAddress: originalVals[0].Bytes(),
		Amount:           math.NewInt(principal),
	}
	require.NoError(t, original.Reporterkeeper.DisputedDelegationAmounts.Set(original.Ctx, hashID, reportertypes.DelegationsAmounts{
		TokenOrigins: []*reportertypes.TokenOriginInfo{origin},
		Total:        math.NewInt(principal),
	}))
	setRestorableDispute(t, original, hashID, bondPayer, cashPayer, reporterAddr, blockTime, openingQuote, retainedFee, appealFee, burnAmount)
	prepareCashFunding(t, original, cashPayer)
	prepareCashFunding(t, restored, cashPayer)

	// SUPPORT execution burns the final round's nine-loya burn allocation. Do
	// this before exporting so the files describe an actually closed, executed
	// dispute, and mirror the same bank state in the fresh app.
	require.NoError(t, original.Disputekeeper.ExecuteVote(original.Ctx, 2))
	require.NoError(t, restored.Bankkeeper.BurnCoins(restored.Ctx, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(restored.Denom, math.NewInt(burnAmount)))))
	require.Equal(t,
		original.Bankkeeper.GetBalance(original.Ctx, original.Accountkeeper.GetModuleAddress(disputetypes.ModuleName), original.Denom).Amount,
		restored.Bankkeeper.GetBalance(restored.Ctx, restored.Accountkeeper.GetModuleAddress(disputetypes.ModuleName), restored.Denom).Amount,
		"fresh bank state must match the exported app before module restore",
	)
	originalBonded, err := original.Stakingkeeper.GetDelegatorBonded(original.Ctx, bondPayer)
	require.NoError(t, err)
	restoredBonded, err := restored.Stakingkeeper.GetDelegatorBonded(restored.Ctx, bondPayer)
	require.NoError(t, err)
	require.Equal(t, originalBonded, restoredBonded, "fresh staking state must match the exported app before module restore")

	// No dispute or reporter claim rows may already exist in the destination;
	// otherwise a no-op reader could make this test pass.
	_, err = restored.Disputekeeper.Disputes.Get(restored.Ctx, 2)
	require.Error(t, err)
	_, err = restored.Disputekeeper.DisputeFeePayer.Get(restored.Ctx, collections.Join(uint64(1), bondPayer.Bytes()))
	require.Error(t, err)
	_, err = restored.Reporterkeeper.FeePaidFromStakeByPayer(restored.Ctx, hashID, bondPayer)
	require.Error(t, err)
	_, err = restored.Reporterkeeper.DisputedDelegationAmounts.Get(restored.Ctx, hashID)
	require.Error(t, err)

	// These are the repository's real module exporters. They write the files
	// consumed by the fork readers; no hand-built JSON fixture is involved.
	dispute.ExportGenesis(original.Ctx, original.Disputekeeper)
	reporter.ExportGenesis(original.Ctx, original.Reporterkeeper)
	require.FileExists(t, filepath.Join(exportDir, "dispute_module_state.json"))
	require.FileExists(t, filepath.Join(exportDir, "reporter_module_state.json"))

	// Import both export files into the corresponding fresh module stores.
	// Keep bank and staking state outside the module files, but construct it by
	// the same real operations in both apps before importing module state.
	migrationCodec := codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
	disputeKey, ok := restored.App.UnsafeFindStoreKey(disputetypes.StoreKey).(*storetypes.KVStoreKey)
	require.True(t, ok)
	reporterKey, ok := restored.App.UnsafeFindStoreKey(reportertypes.StoreKey).(*storetypes.KVStoreKey)
	require.True(t, ok)
	disputeStore := runtime.NewKVStoreService(disputeKey)
	reporterStore := runtime.NewKVStoreService(reporterKey)
	require.NoError(t, disputefork.MigrateStore(restored.Ctx, disputeStore, migrationCodec, exportDir))
	require.NoError(t, reporterfork.MigrateFork(restored.Ctx, reporterStore, migrationCodec, exportDir))

	gotDispute, err := restored.Disputekeeper.Disputes.Get(restored.Ctx, 2)
	require.NoError(t, err)
	require.Equal(t, disputetypes.Resolved, gotDispute.DisputeStatus)
	require.False(t, gotDispute.Open)
	require.Equal(t, []uint64{1, 2}, gotDispute.PrevDisputeIds)
	require.Equal(t, math.NewInt(retainedFee), gotDispute.DisputeFee)
	require.Equal(t, math.NewInt(openingQuote), gotDispute.SlashAmount)

	gotVote, err := restored.Disputekeeper.Votes.Get(restored.Ctx, 2)
	require.NoError(t, err)
	require.Equal(t, disputetypes.VoteResult_SUPPORT, gotVote.VoteResult)
	require.True(t, gotVote.Executed)

	gotReceipt, err := restored.Disputekeeper.DisputeFeePayer.Get(restored.Ctx, collections.Join(uint64(1), bondPayer.Bytes()))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(grossPayment), gotReceipt.Amount)
	require.True(t, gotReceipt.FromBond)
	gotCashReceipt, err := restored.Disputekeeper.DisputeFeePayer.Get(restored.Ctx, collections.Join(uint64(1), cashPayer.Bytes()))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(30), gotCashReceipt.Amount)
	require.False(t, gotCashReceipt.FromBond)
	gotTracker, err := restored.Reporterkeeper.FeePaidFromStakeByPayer(restored.Ctx, hashID, bondPayer)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(grossPayment), gotTracker.Total)
	gotEscrow, err := restored.Reporterkeeper.DisputedDelegationAmounts.Get(restored.Ctx, hashID)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(principal), gotEscrow.Total)

	moduleBefore := restored.Bankkeeper.GetBalance(restored.Ctx, restored.Accountkeeper.GetModuleAddress(disputetypes.ModuleName), restored.Denom).Amount
	payerBankBefore := restored.Bankkeeper.GetBalance(restored.Ctx, bondPayer, restored.Denom).Amount
	payerStakeBefore, err := restored.Stakingkeeper.GetDelegatorBonded(restored.Ctx, bondPayer)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(157), moduleBefore)
	require.Equal(t, math.ZeroInt(), payerStakeBefore)

	_, err = disputekeeper.NewMsgServerImpl(restored.Disputekeeper).WithdrawFeeRefund(restored.Ctx, &disputetypes.MsgWithdrawFeeRefund{
		CallerAddress: bondPayer.String(),
		PayerAddress:  bondPayer.String(),
		Id:            2,
	})
	require.NoError(t, err)

	payerBankAfter := restored.Bankkeeper.GetBalance(restored.Ctx, bondPayer, restored.Denom).Amount
	payerStakeAfter, err := restored.Stakingkeeper.GetDelegatorBonded(restored.Ctx, bondPayer)
	require.NoError(t, err)
	moduleAfter := restored.Bankkeeper.GetBalance(restored.Ctx, restored.Accountkeeper.GetModuleAddress(disputetypes.ModuleName), restored.Denom).Amount
	require.Equal(t, payerBankBefore, payerBankAfter, "a bonded claim must not create a liquid balance")
	require.Equal(t, math.NewInt(109), payerStakeAfter.Sub(payerStakeBefore), "67 fee leg plus 42 SUPPORT principal leg")
	require.Equal(t, math.NewInt(48), moduleAfter, "157 escrow after execution less 109 claimed")
	require.Equal(t, math.NewInt(900000), mustGetDust(t, restored), "the 0.9-loya fee remainder is recorded once")

	_, err = restored.Disputekeeper.DisputeFeePayer.Get(restored.Ctx, collections.Join(uint64(1), bondPayer.Bytes()))
	require.Error(t, err, "the payer row is consumed by the claim")
	_, err = restored.Reporterkeeper.FeePaidFromStakeByPayer(restored.Ctx, hashID, bondPayer)
	require.Error(t, err, "the bonded receipt is consumed by the claim")
	_, err = disputekeeper.NewMsgServerImpl(restored.Disputekeeper).WithdrawFeeRefund(restored.Ctx, &disputetypes.MsgWithdrawFeeRefund{
		CallerAddress: bondPayer.String(),
		PayerAddress:  bondPayer.String(),
		Id:            2,
	})
	require.Error(t, err, "a second claim has no remaining right")
}

func prepareBondedReceipt(t *testing.T, state *setup.SharedSetup, payer sdk.AccAddress, val sdk.ValAddress, hashID []byte) {
	t.Helper()
	amount := math.NewInt(70)
	state.MintTokens(payer, amount)
	_, err := stakingkeeper.NewMsgServerImpl(state.Stakingkeeper).Delegate(state.Ctx, &stakingtypes.MsgDelegate{
		DelegatorAddress: payer.String(),
		ValidatorAddress: val.String(),
		Amount:           sdk.NewCoin(state.Denom, amount),
	})
	require.NoError(t, err)
	require.NoError(t, state.Reporterkeeper.Selectors.Set(state.Ctx, payer, reportertypes.NewSelection(payer.Bytes(), 1)))
	require.NoError(t, state.Reporterkeeper.FeefromReporterStake(state.Ctx, payer, amount, hashID, true))
}

func prepareBondedStakingState(t *testing.T, state *setup.SharedSetup, payer sdk.AccAddress, val sdk.ValAddress) {
	t.Helper()
	amount := math.NewInt(70)
	state.MintTokens(payer, amount)
	_, err := stakingkeeper.NewMsgServerImpl(state.Stakingkeeper).Delegate(state.Ctx, &stakingtypes.MsgDelegate{
		DelegatorAddress: payer.String(),
		ValidatorAddress: val.String(),
		Amount:           sdk.NewCoin(state.Denom, amount),
	})
	require.NoError(t, err)
	validator, err := state.Stakingkeeper.GetValidator(state.Ctx, val)
	require.NoError(t, err)
	shares, err := validator.SharesFromTokens(amount)
	require.NoError(t, err)
	actual, err := state.Stakingkeeper.Unbond(state.Ctx, payer, val, shares)
	require.NoError(t, err)
	require.Equal(t, amount, actual)
	require.NoError(t, state.Bankkeeper.SendCoinsFromModuleToModule(state.Ctx, stakingtypes.BondedPoolName, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(state.Denom, actual))))
}

func prepareReporterEscrow(t *testing.T, state *setup.SharedSetup, reporter sdk.AccAddress, val sdk.ValAddress, amount int64) {
	t.Helper()
	escrow := math.NewInt(amount)
	state.MintTokens(reporter, escrow)
	_, err := stakingkeeper.NewMsgServerImpl(state.Stakingkeeper).Delegate(state.Ctx, &stakingtypes.MsgDelegate{
		DelegatorAddress: reporter.String(),
		ValidatorAddress: val.String(),
		Amount:           sdk.NewCoin(state.Denom, escrow),
	})
	require.NoError(t, err)
	validator, err := state.Stakingkeeper.GetValidator(state.Ctx, val)
	require.NoError(t, err)
	shares, err := validator.SharesFromTokens(escrow)
	require.NoError(t, err)
	actual, err := state.Stakingkeeper.Unbond(state.Ctx, reporter, val, shares)
	require.NoError(t, err)
	require.Equal(t, escrow, actual)
	require.NoError(t, state.Bankkeeper.SendCoinsFromModuleToModule(state.Ctx, stakingtypes.BondedPoolName, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(state.Denom, actual))))
}

func prepareCashFunding(t *testing.T, state *setup.SharedSetup, payer sdk.AccAddress) {
	t.Helper()
	coins := sdk.NewCoins(sdk.NewCoin(state.Denom, math.NewInt(36)))
	state.MintTokens(payer, coins.AmountOf(state.Denom))
	require.NoError(t, state.Bankkeeper.SendCoinsFromAccountToModule(state.Ctx, payer, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(state.Denom, math.NewInt(30)))))
	require.NoError(t, state.Bankkeeper.SendCoinsFromAccountToModule(state.Ctx, payer, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(state.Denom, math.NewInt(6)))))
}

func fixedAddress(fill byte) sdk.AccAddress {
	return sdk.AccAddress(bytes.Repeat([]byte{fill}, 20))
}

func setRestorableDispute(t *testing.T, state *setup.SharedSetup, hashID []byte, bondPayer, cashPayer, reporter sdk.AccAddress, blockTime time.Time, quote, retainedFee, appealFee, burn int64) {
	t.Helper()
	evidence := oracletypes.MicroReport{Reporter: reporter.String(), BlockNumber: 1}
	roundOne := disputetypes.Dispute{
		HashId:          hashID,
		DisputeId:       1,
		DisputeCategory: disputetypes.Major,
		DisputeFee:      math.NewInt(retainedFee),
		DisputeStatus:   disputetypes.Unresolved,
		DisputeEndTime:  blockTime.Add(-time.Hour),
		DisputeRound:    1,
		SlashAmount:     math.NewInt(quote),
		BurnAmount:      math.NewInt(3),
		InitialEvidence: evidence,
		FeeTotal:        math.NewInt(quote),
		PrevDisputeIds:  []uint64{1},
		BlockNumber:     1,
		Open:            false,
	}
	finalRound := disputetypes.Dispute{
		HashId:          hashID,
		DisputeId:       2,
		DisputeCategory: disputetypes.Major,
		DisputeFee:      math.NewInt(retainedFee),
		DisputeStatus:   disputetypes.Resolved,
		DisputeEndTime:  blockTime.Add(-time.Hour),
		DisputeRound:    2,
		SlashAmount:     math.NewInt(quote),
		BurnAmount:      math.NewInt(burn),
		InitialEvidence: evidence,
		FeeTotal:        math.NewInt(quote + appealFee),
		PrevDisputeIds:  []uint64{1, 2},
		BlockNumber:     1,
		Open:            false,
	}
	require.NoError(t, state.Disputekeeper.Disputes.Set(state.Ctx, 1, roundOne))
	require.NoError(t, state.Disputekeeper.Disputes.Set(state.Ctx, 2, finalRound))
	require.NoError(t, state.Disputekeeper.Votes.Set(state.Ctx, 2, disputetypes.Vote{
		Id:         2,
		VoteStart:  blockTime.Add(-2 * time.Hour),
		VoteEnd:    blockTime.Add(-time.Hour),
		VoteResult: disputetypes.VoteResult_SUPPORT,
	}))
	require.NoError(t, state.Disputekeeper.DisputeFeePayer.Set(state.Ctx, collections.Join(uint64(1), bondPayer.Bytes()), disputetypes.PayerInfo{
		Amount:   math.NewInt(70),
		FromBond: true,
	}))
	require.NoError(t, state.Disputekeeper.DisputeFeePayer.Set(state.Ctx, collections.Join(uint64(1), cashPayer.Bytes()), disputetypes.PayerInfo{
		Amount:   math.NewInt(30),
		FromBond: false,
	}))
	require.NoError(t, state.Disputekeeper.Dust.Set(state.Ctx, math.ZeroInt()))
}

func mustGetDust(t *testing.T, state *setup.SharedSetup) math.Int {
	t.Helper()
	dust, err := state.Disputekeeper.Dust.Get(state.Ctx)
	require.NoError(t, err)
	return dust
}
