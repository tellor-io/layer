package integration_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	setup "github.com/tellor-io/layer/tests"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// TestBondedClaimFullQueueRollsBackAndRetries covers a bonded receipt whose
// origins belong to two different selector owners. The selected reporter is
// deliberately capped so every fee leg must overflow to its original
// validator. The second owner's unbonding queue is full, so a cached SUPPORT
// claim first attempts the first owner, then fails on the second owner and
// leaves every operation untouched. A cash payer settles independently;
// freeing one queue slot then lets the exact same bonded claim message
// complete both SUPPORT legs.
func TestBondedClaimFullQueueRollsBackAndRetries(t *testing.T) {
	state := &setup.SharedSetup{}
	state.SetupTest(t)
	state.Ctx = state.Ctx.WithBlockHeight(5).WithBlockTime(time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC))
	_, vals, _ := state.CreateValidators(3)
	smallVal := vals[2]

	params, err := state.Reporterkeeper.Params.Get(state.Ctx)
	require.NoError(t, err)
	params.MaxValidatorPowerShare = math.LegacyOneDec()
	// The two selector owners choose the bonded payer as their reporter. Make
	// that reporter's cap smaller than its remaining ten-loya stake so all of
	// its fee returns must use their original validator/queue pair. The cash
	// payer has no selection and can use the ordinary validator for its
	// principal because the validator cap is disabled.
	params.MaxReporterPowerShare = math.LegacyNewDecWithPrec(1, 12)
	require.NoError(t, state.Reporterkeeper.Params.Set(state.Ctx, params))

	bondPayer := retryAddress(0x44)
	ownerOne := retryAddress(0x11)
	ownerTwo := retryAddress(0x22)
	cashPayer := retryAddress(0x33)
	reporter := retryAddress(0x55)
	hashID := []byte("bonded-claim-full-queue")

	// Two independent selector owners each have forty loya on a different
	// bonded validator. The 70-loya fee is therefore recorded as two 35-loya
	// receipt origins.
	prepareRetrySelector(t, state, ownerOne, bondPayer, vals[0], 40)
	prepareRetrySelector(t, state, ownerTwo, bondPayer, vals[1], 40)
	require.NoError(t, state.Reporterkeeper.FeefromReporterStake(state.Ctx, bondPayer, math.NewInt(70), hashID, true))

	// The independent cash payer has a real small-validator delegation so its
	// SUPPORT principal leg can be paid while the large validators are capped.
	state.MintTokens(cashPayer, math.NewInt(40))
	_, err = stakingkeeper.NewMsgServerImpl(state.Stakingkeeper).Delegate(state.Ctx, &stakingtypes.MsgDelegate{
		DelegatorAddress: cashPayer.String(),
		ValidatorAddress: smallVal.String(),
		Amount:           sdk.NewCoin(state.Denom, math.NewInt(10)),
	})
	require.NoError(t, err)
	require.NoError(t, state.Bankkeeper.SendCoinsFromAccountToModule(state.Ctx, cashPayer, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(state.Denom, math.NewInt(30)))))

	// Escrow sixty loya from a distinct reporter's real delegation. The
	// dispute module then holds 70 bond + 30 cash + 60 escrow = 160.
	prepareRetryReporterEscrow(t, state, reporter, vals[2], hashID, 60)
	require.NoError(t, state.Reporterkeeper.DisputedDelegationAmounts.Set(state.Ctx, hashID, reportertypes.DelegationsAmounts{
		TokenOrigins: []*reportertypes.TokenOriginInfo{{
			DelegatorAddress: reporter.Bytes(),
			ValidatorAddress: vals[2].Bytes(),
			Amount:           math.NewInt(60),
		}},
		Total: math.NewInt(60),
	}))

	report := oracletypes.MicroReport{Reporter: reporter.String(), BlockNumber: uint64(state.Ctx.BlockHeight()), Timestamp: state.Ctx.BlockTime().Add(-time.Hour)}
	dispute := disputetypes.Dispute{
		HashId:           hashID,
		DisputeId:        1,
		DisputeCategory:  disputetypes.Major,
		DisputeFee:       math.NewInt(60),
		DisputeStatus:    disputetypes.Resolved,
		DisputeStartTime: state.Ctx.BlockTime().Add(-2 * time.Hour),
		DisputeEndTime:   state.Ctx.BlockTime().Add(-time.Hour),
		DisputeRound:     1,
		SlashAmount:      math.NewInt(100),
		BurnAmount:       math.NewInt(3),
		InitialEvidence:  report,
		FeeTotal:         math.NewInt(100),
		PrevDisputeIds:   []uint64{1},
		Open:             false,
		BlockNumber:      uint64(state.Ctx.BlockHeight()),
	}
	require.NoError(t, state.Disputekeeper.Disputes.Set(state.Ctx, 1, dispute))
	require.NoError(t, state.Disputekeeper.Votes.Set(state.Ctx, 1, disputetypes.Vote{
		Id:         1,
		VoteStart:  dispute.DisputeStartTime,
		VoteEnd:    dispute.DisputeEndTime,
		VoteResult: disputetypes.VoteResult_SUPPORT,
		Executed:   true,
	}))
	require.NoError(t, state.Disputekeeper.DisputeFeePayer.Set(state.Ctx, collections.Join(uint64(1), bondPayer.Bytes()), disputetypes.PayerInfo{Amount: math.NewInt(70), FromBond: true}))
	require.NoError(t, state.Disputekeeper.DisputeFeePayer.Set(state.Ctx, collections.Join(uint64(1), cashPayer.Bytes()), disputetypes.PayerInfo{Amount: math.NewInt(30)}))
	require.NoError(t, state.Disputekeeper.Dust.Set(state.Ctx, math.ZeroInt()))
	require.NoError(t, state.Bankkeeper.BurnCoins(state.Ctx, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(state.Denom, math.NewInt(3)))))
	require.Equal(t, math.NewInt(157), retryModuleBalance(state))

	trackerBefore, err := state.Reporterkeeper.FeePaidFromStakeByPayer(state.Ctx, hashID, bondPayer)
	require.NoError(t, err)
	require.Len(t, trackerBefore.TokenOrigins, 2)
	require.Equal(t, ownerOne.Bytes(), trackerBefore.TokenOrigins[0].DelegatorAddress)
	require.Equal(t, ownerTwo.Bytes(), trackerBefore.TokenOrigins[1].DelegatorAddress)
	require.Equal(t, math.NewInt(35), trackerBefore.TokenOrigins[0].Amount)
	require.Equal(t, math.NewInt(35), trackerBefore.TokenOrigins[1].Amount)

	maxEntries, err := state.Stakingkeeper.MaxEntries(state.Ctx)
	require.NoError(t, err)
	// Leave the first origin writable so the cached attempt performs real
	// work before the second origin hits its full queue.
	fillRetryQueue(t, state, ownerTwo, vals[1], maxEntries)
	ownerOneUBDBefore := retryUBD(t, state, ownerOne, vals[0])
	ownerTwoUBDBefore := retryUBD(t, state, ownerTwo, vals[1])
	ownerOneStakeBefore := retryStake(t, state, ownerOne)
	ownerTwoStakeBefore := retryStake(t, state, ownerTwo)
	bondStakeBefore := retryStake(t, state, bondPayer)
	moduleBefore := retryModuleBalance(state)
	dustBefore, err := state.Disputekeeper.Dust.Get(state.Ctx)
	require.NoError(t, err)
	rowBefore, err := state.Disputekeeper.DisputeFeePayer.Get(state.Ctx, collections.Join(uint64(1), bondPayer.Bytes()))
	require.NoError(t, err)

	claim := &disputetypes.MsgWithdrawFeeRefund{CallerAddress: bondPayer.String(), PayerAddress: bondPayer.String(), Id: 1}
	claimBefore := *claim
	cacheCtx, _ := state.Ctx.CacheContext()
	_, err = disputekeeper.NewMsgServerImpl(state.Disputekeeper).WithdrawFeeRefund(cacheCtx, claim)
	require.ErrorIs(t, err, stakingtypes.ErrMaxUnbondingDelegationEntries)
	attempted, err := state.Stakingkeeper.GetUnbondingDelegation(cacheCtx, ownerOne, vals[0])
	require.NoError(t, err)
	require.Len(t, attempted.Entries, 1)
	require.Equal(t, math.NewInt(34), attempted.Entries[0].Balance)

	require.Equal(t, claimBefore, *claim, "the failed cached attempt must not rewrite the message")

	// The cached claim returned the first owner's fee before the second owner's
	// full queue failed, but committed nothing: no delegation, queue entry,
	// receipt, module transfer, dust, or later-owner work may leak into the
	// parent context.
	require.Equal(t, moduleBefore, retryModuleBalance(state))
	require.Equal(t, dustBefore, mustRetryDust(t, state))
	require.Equal(t, rowBefore, mustRetryPayer(t, state, bondPayer))
	require.Equal(t, trackerBefore, mustRetryTracker(t, state, hashID, bondPayer))
	require.Equal(t, ownerOneStakeBefore, retryStake(t, state, ownerOne))
	require.Equal(t, ownerTwoStakeBefore, retryStake(t, state, ownerTwo))
	require.Equal(t, ownerOneUBDBefore, retryUBD(t, state, ownerOne, vals[0]))
	require.Equal(t, ownerTwoUBDBefore, retryUBD(t, state, ownerTwo, vals[1]))
	require.Equal(t, bondStakeBefore, retryStake(t, state, bondPayer))

	// A different payer is independent: its cash fee and SUPPORT principal can
	// settle while the bonded payer's queue remains blocked.
	cashBalanceBefore := state.Bankkeeper.GetBalance(state.Ctx, cashPayer, state.Denom).Amount
	cashStakeBefore := retryStake(t, state, cashPayer)
	_, err = disputekeeper.NewMsgServerImpl(state.Disputekeeper).WithdrawFeeRefund(state.Ctx, &disputetypes.MsgWithdrawFeeRefund{CallerAddress: cashPayer.String(), PayerAddress: cashPayer.String(), Id: 1})
	require.NoError(t, err)
	require.Equal(t, math.NewInt(29), state.Bankkeeper.GetBalance(state.Ctx, cashPayer, state.Denom).Amount.Sub(cashBalanceBefore))
	require.Equal(t, math.NewInt(18), retryStake(t, state, cashPayer).Sub(cashStakeBefore))
	require.Error(t, mustRetryPayerErr(state, cashPayer), "the independent payer's row is consumed")
	require.Equal(t, moduleBefore.Sub(math.NewInt(47)), retryModuleBalance(state))

	// Remove exactly one old entry from the blocked owner's pair, preserving the
	// same receipt and message for the retry.
	ownerTwoUBD := retryUBD(t, state, ownerTwo, vals[1])
	require.NotNil(t, ownerTwoUBD)
	require.Len(t, ownerTwoUBD.Entries, int(maxEntries))
	ownerTwoUBD.Entries = ownerTwoUBD.Entries[1:]
	require.NoError(t, state.Stakingkeeper.SetUnbondingDelegation(state.Ctx, *ownerTwoUBD))
	_, err = disputekeeper.NewMsgServerImpl(state.Disputekeeper).WithdrawFeeRefund(state.Ctx, claim)
	require.NoError(t, err)
	require.Equal(t, claimBefore, *claim, "retry uses the exact same message")
	// The named payer receives only the 42-loya SUPPORT principal. The 67-loya
	// fee is split 34/33 back to the two selector owners; the extra unit goes
	// to the first origin by the stable largest-remainder tie break.
	require.Equal(t, math.NewInt(42), retryStake(t, state, bondPayer).Sub(bondStakeBefore), "only SUPPORT principal returns to the named payer")
	require.Equal(t, math.NewInt(34), retryStake(t, state, ownerOne).Sub(ownerOneStakeBefore), "34 fee return to the first owner")
	require.Equal(t, math.NewInt(32), retryStake(t, state, ownerTwo).Sub(ownerTwoStakeBefore), "33 fee return minus the old queue entry removed")
	require.Equal(t, math.NewInt(34), retryNewUnbondingAmount(t, state, ownerOne, vals[0], math.NewInt(34)), "first fee origin receives 34")
	require.Equal(t, math.NewInt(33), retryNewUnbondingAmount(t, state, ownerTwo, vals[1], math.NewInt(33)), "second fee origin receives 33")
	require.Equal(t, math.ZeroInt(), retryModuleBalance(state))
	require.Equal(t, math.ZeroInt(), mustRetryDust(t, state))
	require.Error(t, mustRetryPayerErr(state, bondPayer), "the bonded payer's row is consumed")
	_, err = state.Reporterkeeper.FeePaidFromStakeByPayer(state.Ctx, hashID, bondPayer)
	require.Error(t, err, "the bonded receipt is consumed")
	_, err = disputekeeper.NewMsgServerImpl(state.Disputekeeper).WithdrawFeeRefund(state.Ctx, claim)
	require.Error(t, err, "a second bonded claim has no right")
}

func retryAddress(fill byte) sdk.AccAddress {
	return sdk.AccAddress(bytes.Repeat([]byte{fill}, 20))
}

func prepareRetrySelector(t *testing.T, state *setup.SharedSetup, owner, reporter sdk.AccAddress, val sdk.ValAddress, amount int64) {
	t.Helper()
	stake := math.NewInt(amount)
	state.MintTokens(owner, stake)
	_, err := stakingkeeper.NewMsgServerImpl(state.Stakingkeeper).Delegate(state.Ctx, &stakingtypes.MsgDelegate{
		DelegatorAddress: owner.String(),
		ValidatorAddress: val.String(),
		Amount:           sdk.NewCoin(state.Denom, stake),
	})
	require.NoError(t, err)
	require.NoError(t, state.Reporterkeeper.Selectors.Set(state.Ctx, owner, reportertypes.NewSelection(reporter.Bytes(), 1)))
}

func prepareRetryReporterEscrow(t *testing.T, state *setup.SharedSetup, reporter sdk.AccAddress, val sdk.ValAddress, hashID []byte, amount int64) {
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

func fillRetryQueue(t *testing.T, state *setup.SharedSetup, owner sdk.AccAddress, val sdk.ValAddress, maxEntries uint32) {
	t.Helper()
	for i := uint32(0); i < maxEntries; i++ {
		_, err := state.Stakingkeeper.SetUnbondingDelegationEntry(state.Ctx, owner, val, int64(i+1), state.Ctx.BlockTime().Add(time.Duration(i+1)*time.Hour), math.OneInt())
		require.NoError(t, err)
	}
}

func retryUBD(t *testing.T, state *setup.SharedSetup, owner sdk.AccAddress, val sdk.ValAddress) *stakingtypes.UnbondingDelegation {
	t.Helper()
	ubd, err := state.Stakingkeeper.GetUnbondingDelegation(state.Ctx, owner, val)
	if errors.Is(err, stakingtypes.ErrNoUnbondingDelegation) {
		return nil
	}
	require.NoError(t, err)
	return &ubd
}

func retryStake(t *testing.T, state *setup.SharedSetup, owner sdk.AccAddress) math.Int {
	t.Helper()
	stake, err := state.Stakingkeeper.GetDelegatorBonded(state.Ctx, owner)
	require.NoError(t, err)
	entries, err := state.Stakingkeeper.GetAllUnbondingDelegations(state.Ctx, owner)
	require.NoError(t, err)
	for _, ubd := range entries {
		for _, entry := range ubd.Entries {
			stake = stake.Add(entry.Balance)
		}
	}
	return stake
}

func retryNewUnbondingAmount(t *testing.T, state *setup.SharedSetup, owner sdk.AccAddress, val sdk.ValAddress, returned math.Int) math.Int {
	t.Helper()
	ubd := retryUBD(t, state, owner, val)
	if ubd == nil {
		return math.ZeroInt()
	}
	amount := math.ZeroInt()
	for _, entry := range ubd.Entries {
		if entry.Balance.Equal(returned) {
			amount = amount.Add(entry.Balance)
		}
	}
	return amount
}

func retryModuleBalance(state *setup.SharedSetup) math.Int {
	return state.Bankkeeper.GetBalance(state.Ctx, state.Accountkeeper.GetModuleAddress(disputetypes.ModuleName), state.Denom).Amount
}

func mustRetryDust(t *testing.T, state *setup.SharedSetup) math.Int {
	t.Helper()
	dust, err := state.Disputekeeper.Dust.Get(state.Ctx)
	require.NoError(t, err)
	return dust
}

func mustRetryPayer(t *testing.T, state *setup.SharedSetup, payer sdk.AccAddress) disputetypes.PayerInfo {
	t.Helper()
	payerInfo, err := state.Disputekeeper.DisputeFeePayer.Get(state.Ctx, collections.Join(uint64(1), payer.Bytes()))
	require.NoError(t, err)
	return payerInfo
}

func mustRetryPayerErr(state *setup.SharedSetup, payer sdk.AccAddress) error {
	_, err := state.Disputekeeper.DisputeFeePayer.Get(state.Ctx, collections.Join(uint64(1), payer.Bytes()))
	return err
}

func mustRetryTracker(t *testing.T, state *setup.SharedSetup, hashID []byte, payer sdk.AccAddress) reportertypes.DelegationsAmounts {
	t.Helper()
	tracker, err := state.Reporterkeeper.FeePaidFromStakeByPayer(state.Ctx, hashID, payer)
	require.NoError(t, err)
	return tracker
}
