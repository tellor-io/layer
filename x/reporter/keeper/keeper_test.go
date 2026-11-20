package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func setupKeeper(tb testing.TB) (keeper.Keeper, *mocks.StakingKeeper, *mocks.BankKeeper, *mocks.RegistryKeeper, *mocks.AccountKeeper, sdk.Context, store.KVStoreService) {
	tb.Helper()
	return keepertest.ReporterKeeper(tb)
}

func TestKeeper(t *testing.T) {
	k, sk, bk, _, ak, ctx, _ := keepertest.ReporterKeeper(t)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
	require.NotNil(t, sk)
	require.NotNil(t, bk)
	require.NotNil(t, ak)
}

func TestGetAuthority(t *testing.T) {
	k, _, _, _, _, _, _ := setupKeeper(t)
	authority := k.GetAuthority()
	require.NotEmpty(t, authority)

	// convert to address and check if it's valid
	_, err := sdk.AccAddressFromBech32(authority)
	require.NoError(t, err)
}

func TestLogger(t *testing.T) {
	k, _, _, _, _, _, _ := setupKeeper(t)
	require.NotNil(t, k.Logger())
}

func TestGetDelegatorTokensAtBlock(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	delAddr, val1Address, val2Address := sample.AccAddressBytes(), sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	require.NoError(t, k.Selectors.Set(ctx, delAddr, types.NewSelection(delAddr, 2)))

	tokenOrigin1 := &types.TokenOriginInfo{
		DelegatorAddress: delAddr,
		ValidatorAddress: val1Address,
		Amount:           math.NewIntWithDecimal(1000, 6),
	}
	tokenOrigin2 := &types.TokenOriginInfo{
		DelegatorAddress: delAddr,
		ValidatorAddress: val2Address,
		Amount:           math.NewIntWithDecimal(1000, 6),
	}
	tokenOrigins := []*types.TokenOriginInfo{tokenOrigin1, tokenOrigin2}
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte{}, collections.Join(delAddr.Bytes(), uint64(ctx.BlockHeight()))), types.DelegationsAmounts{TokenOrigins: tokenOrigins, Total: tokenOrigin1.Amount.Add(tokenOrigin2.Amount)}))

	tokens, err := k.GetDelegatorTokensAtBlock(ctx, delAddr, uint64(ctx.BlockHeight()))
	require.NoError(t, err)
	require.Equal(t, math.NewIntWithDecimal(2000, 6), tokens)

	otherReporter := sample.AccAddressBytes()
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte{}, collections.Join(otherReporter.Bytes(), uint64(ctx.BlockHeight()))), types.DelegationsAmounts{
		TokenOrigins: []*types.TokenOriginInfo{tokenOrigin1},
		Total:        tokenOrigin1.Amount,
	}))
	fromOther, err := k.GetDelegatorTokensFromReporterAtBlock(ctx, delAddr, otherReporter, uint64(ctx.BlockHeight()))
	require.NoError(t, err)
	require.Equal(t, math.NewIntWithDecimal(1000, 6), fromOther)
}

func TestGetReporterTokensAtBlock(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporter := sample.AccAddressBytes()
	tokens, err := k.GetReporterTokensAtBlock(ctx, reporter, uint64(ctx.BlockHeight()))
	require.NoError(t, err)
	require.Equal(t, math.ZeroInt(), tokens)

	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte{}, collections.Join(reporter.Bytes(), uint64(ctx.BlockHeight()))), types.DelegationsAmounts{Total: math.OneInt()}))

	tokens, err = k.GetReporterTokensAtBlock(ctx, reporter, uint64(ctx.BlockHeight()))
	require.NoError(t, err)
	require.Equal(t, math.OneInt(), tokens)

	tokens, err = k.GetReporterTokensAtBlock(ctx, reporter, uint64(ctx.BlockHeight()+10))
	require.NoError(t, err)
	require.Equal(t, math.OneInt(), tokens)
}

// TestGetReporterTokensAtBlockAfterCompletedSelectorSwitch covers a selector
// whose switch finished before the dispute block. The old reporter's snapshot
// still lists them until that reporter reports again. Those origins stay in the
// old total until the new reporter's snapshot lists them, then they come out.
func TestGetReporterTokensAtBlockAfterCompletedSelectorSwitch(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	oldReporter := sample.AccAddressBytes()
	newReporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	unknown := sample.AccAddressBytes()
	val1 := sdk.ValAddress(sample.AccAddressBytes())
	val2 := sdk.ValAddress(sample.AccAddressBytes())
	block := uint64(40)

	selectorOnVal1 := math.NewInt(12)
	selectorOnVal2 := math.NewInt(13)
	unknownAmount := math.NewInt(5)
	reporterOwn := math.NewInt(70)
	total := math.NewInt(100)

	require.NoError(t, k.Selectors.Set(ctx, oldReporter.Bytes(), types.NewSelection(oldReporter, 1)))
	require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), types.NewSelection(newReporter, 1)))
	require.NoError(t, k.SelectorReporterSwitches.Set(ctx, selector.Bytes(), types.SelectorReporterSwitches{
		Switches: []types.SelectorReporterSwitch{
			{OriginalReporter: oldReporter.Bytes(), BeganHeight: 10, CompletedHeight: 20},
		},
	}))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(oldReporter.Bytes(), uint64(15), []byte("q")), types.DelegationsAmounts{
		Total: total,
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: selector, ValidatorAddress: val1, Amount: selectorOnVal1},
			{DelegatorAddress: selector, ValidatorAddress: val2, Amount: selectorOnVal2},
			// Nil amount must not panic, and it is not part of Total.
			{DelegatorAddress: selector, ValidatorAddress: val1, Amount: math.Int{}},
			{DelegatorAddress: unknown, ValidatorAddress: val1, Amount: unknownAmount},
			{DelegatorAddress: oldReporter, ValidatorAddress: val1, Amount: reporterOwn},
		},
	}))

	got, err := k.GetReporterTokensAtBlock(ctx, oldReporter.Bytes(), block)
	require.NoError(t, err)
	require.Equal(t, total, got, "no snapshot on the new reporter: the old total still includes the selector")

	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(newReporter.Bytes(), uint64(30), []byte("q")), types.DelegationsAmounts{
		Total: selectorOnVal1.Add(selectorOnVal2),
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: selector, ValidatorAddress: val1, Amount: selectorOnVal1},
			{DelegatorAddress: selector, ValidatorAddress: val2, Amount: selectorOnVal2},
		},
	}))
	got, err = k.GetReporterTokensAtBlock(ctx, oldReporter.Bytes(), block)
	require.NoError(t, err)
	require.Equal(t, reporterOwn.Add(unknownAmount), got, "new snapshot lists the selector, so the old total drops both of that selector's origins")

	// A later snapshot whose delegator is still on this reporter returns that total
	// even when the other reporter's snapshot also lists them.
	still := sample.AccAddressBytes()
	stillTotal := math.NewInt(40)
	require.NoError(t, k.Selectors.Set(ctx, still.Bytes(), types.NewSelection(oldReporter, 1)))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(oldReporter.Bytes(), uint64(16), []byte("still")), types.DelegationsAmounts{
		Total: stillTotal,
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: still, ValidatorAddress: val1, Amount: stillTotal},
		},
	}))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(newReporter.Bytes(), uint64(31), []byte("still")), types.DelegationsAmounts{
		Total: stillTotal,
		TokenOrigins: []*types.TokenOriginInfo{
			{DelegatorAddress: still, ValidatorAddress: val1, Amount: stillTotal},
		},
	}))
	got, err = k.GetReporterTokensAtBlock(ctx, oldReporter.Bytes(), block)
	require.NoError(t, err)
	require.Equal(t, stillTotal, got)
}

func TestTrackStakeChange(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	expiration := ctx.BlockTime().Add(1)
	err := k.Tracker.Set(ctx, types.StakeTracker{Expiration: &expiration, Amount: math.NewInt(1000)})
	require.NoError(t, err)
	require.NoError(t, k.TrackStakeChange(ctx))

	expiration = ctx.BlockTime()
	err = k.Tracker.Set(ctx, types.StakeTracker{Expiration: &expiration, Amount: math.NewInt(1000)})
	require.NoError(t, err)
	sk.On("TotalBondedTokens", ctx).Return(math.OneInt(), nil)
	require.NoError(t, k.TrackStakeChange(ctx))

	change, err := k.Tracker.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, math.OneInt(), change.Amount)
	require.Equal(t, expiration.Add(12*time.Hour), *change.Expiration)
}

func TestReportIndexedMap(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	keys := []collections.Pair[[]byte, collections.Pair[[]byte, uint64]]{
		collections.Join([]byte("queryid1"), collections.Join([]byte("reporterA"), uint64(1))),
		collections.Join([]byte("queryid2"), collections.Join([]byte("reporterA"), uint64(1))),
		collections.Join([]byte("queryid3"), collections.Join([]byte("reporterA"), uint64(1))),
		collections.Join([]byte("queryid1"), collections.Join([]byte("reporterB"), uint64(1))),
		collections.Join([]byte("queryid1"), collections.Join([]byte("reporterC"), uint64(1))),
		collections.Join([]byte("queryid1"), collections.Join([]byte("reporterD"), uint64(1))),
		collections.Join([]byte("queryid1"), collections.Join([]byte("reporterD"), uint64(1))),
		collections.Join([]byte("queryid1"), collections.Join([]byte("reporterA"), uint64(6))),
		collections.Join([]byte("queryid2"), collections.Join([]byte("reporterA"), uint64(6))),
	}
	for _, key := range keys {
		require.NoError(t, k.Report.Set(ctx, key, types.DelegationsAmounts{}))
	}

	// first key of reporterA should be at block 6 and it should queryid2
	startFrom := collections.Join([]byte("reporterA"), uint64(0))
	endAt := collections.Join([]byte("reporterA"), uint64(7))
	kc := collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key)
	startBuffer := make([]byte, kc.Size(startFrom))
	endBuffer := make([]byte, kc.Size(endAt))

	_, err := kc.Encode(startBuffer, startFrom)
	require.NoError(t, err)
	_, err = kc.Encode(endBuffer, endAt)
	require.NoError(t, err)

	iter, err := k.Report.Indexes.BlockNumber.IterateRaw(ctx, startBuffer, endBuffer, collections.OrderDescending)
	require.NoError(t, err)
	require.True(t, iter.Valid())

	key, err := iter.Key()
	require.NoError(t, err)
	require.Equal(t, []byte("queryid2"), key.K2())
	require.Equal(t, []byte("reporterA"), key.K1().K1())
	require.Equal(t, uint64(6), key.K1().K2())

	// reporterA should have 5 keys
	repAkeys, err := iter.Keys()
	require.NoError(t, err)
	require.Equal(t, 5, len(repAkeys))
}

func TestGetDelegationsAmount(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	validator := sample.AccAddressBytes()
	reporter := sample.AccAddressBytes()
	blockNumber := uint64(10)
	ctx = ctx.WithBlockHeight(int64(blockNumber))
	ctx = ctx.WithBlockTime(time.Now())

	// 1000 delegated on block 1
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid1"), collections.Join(reporter.Bytes(), uint64(1))), types.DelegationsAmounts{
		Total: math.NewInt(1000),
		TokenOrigins: []*types.TokenOriginInfo{
			{
				DelegatorAddress: reporter,
				ValidatorAddress: validator,
				Amount:           math.NewInt(1000),
			},
		},
	}))

	delegations, err := k.GetDelegationsAmount(ctx, reporter, 1)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(1000), delegations.Total)
	require.Equal(t, 1, len(delegations.TokenOrigins))
	require.Equal(t, validator.Bytes(), delegations.TokenOrigins[0].ValidatorAddress)
	require.Equal(t, reporter.Bytes(), delegations.TokenOrigins[0].DelegatorAddress)
	require.Equal(t, math.NewInt(1000), delegations.TokenOrigins[0].Amount)

	// 4000 more to same guy delegated on block 2
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid1"), collections.Join(reporter.Bytes(), uint64(2))), types.DelegationsAmounts{
		Total: math.NewInt(5000),
		TokenOrigins: []*types.TokenOriginInfo{
			{
				DelegatorAddress: reporter,
				ValidatorAddress: validator,
				Amount:           math.NewInt(5000),
			},
		},
	}))

	// 10000 to new guy delegated on block 3
	newValidator := sample.AccAddressBytes()
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid1"), collections.Join(reporter.Bytes(), uint64(3))), types.DelegationsAmounts{
		Total: math.NewInt(10000),
		TokenOrigins: []*types.TokenOriginInfo{
			{
				DelegatorAddress: reporter,
				ValidatorAddress: newValidator,
				Amount:           math.NewInt(10000),
			},
			{
				DelegatorAddress: reporter,
				ValidatorAddress: validator,
				Amount:           math.NewInt(5000),
			},
		},
	}))

	delegations, err = k.GetDelegationsAmount(ctx, reporter, 5)
	fmt.Println(delegations)
	require.NoError(t, err)
	require.Equal(t, 2, len(delegations.TokenOrigins))
	// add up token origins
	total := math.NewInt(0)
	for _, tokenOrigin := range delegations.TokenOrigins {
		total = total.Add(tokenOrigin.Amount)
	}
	require.Equal(t, math.NewInt(15000), total)
}

func TestTransitionPathsPreferNewCollection(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporter := sample.AccAddressBytes()
	ctx = ctx.WithBlockHeight(10)

	// Legacy snapshot exists.
	require.NoError(t, k.Report.Set(
		ctx,
		collections.Join([]byte("legacy-query"), collections.Join(reporter.Bytes(), uint64(4))),
		types.DelegationsAmounts{Total: math.NewInt(100)},
	))

	// New snapshot at a later eligible block should be preferred.
	require.NoError(t, k.ReportByBlock.Set(
		ctx,
		collections.Join3(reporter.Bytes(), uint64(6), []byte("new-query")),
		types.DelegationsAmounts{Total: math.NewInt(250)},
	))

	got, err := k.GetDelegationsAmount(ctx, reporter, 10)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(250), got.Total)

	last, err := k.GetLastReportedAtBlock(ctx, reporter)
	require.NoError(t, err)
	require.Equal(t, uint64(6), last)
}

func TestTransitionPathsFallbackToOldCollection(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporter := sample.AccAddressBytes()
	ctx = ctx.WithBlockHeight(5)

	// Legacy snapshot should still be served when new collection has no eligible entry.
	require.NoError(t, k.Report.Set(
		ctx,
		collections.Join([]byte("legacy-query"), collections.Join(reporter.Bytes(), uint64(4))),
		types.DelegationsAmounts{Total: math.NewInt(123)},
	))

	// New snapshot exists but is outside the requested/current block window.
	require.NoError(t, k.ReportByBlock.Set(
		ctx,
		collections.Join3(reporter.Bytes(), uint64(8), []byte("new-query")),
		types.DelegationsAmounts{Total: math.NewInt(999)},
	))

	got, err := k.GetDelegationsAmount(ctx, reporter, 5)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(123), got.Total)

	last, err := k.GetLastReportedAtBlock(ctx, reporter)
	require.NoError(t, err)
	require.Equal(t, uint64(4), last)
}

// called in endblocker
func BenchmarkReporterTrackStakeChange(b *testing.B) {
	k, sk, _, _, _, ctx, _ := setupKeeper(b)
	ctx = ctx.WithBlockHeight(3).WithBlockTime(time.Now())
	expiration := ctx.BlockTime().Add(1)
	err := k.Tracker.Set(ctx, types.StakeTracker{Expiration: &expiration, Amount: math.NewInt(1000)})
	require.NoError(b, err)

	sk.On("TotalBondedTokens", ctx).Return(math.OneInt(), nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := k.TrackStakeChange(ctx)
		require.NoError(b, err)
	}
}

func TestGetLastReportedAtBlock(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporter := sample.AccAddressBytes()
	ctx = ctx.WithBlockHeight(4)

	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid1"), collections.Join(reporter.Bytes(), uint64(1))), types.DelegationsAmounts{}))
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid1"), collections.Join(reporter.Bytes(), uint64(2))), types.DelegationsAmounts{}))
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid1"), collections.Join(reporter.Bytes(), uint64(3))), types.DelegationsAmounts{}))
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid2"), collections.Join(reporter.Bytes(), uint64(3))), types.DelegationsAmounts{}))

	lastReportedAtBlock, err := k.GetLastReportedAtBlock(ctx, reporter)
	require.NoError(t, err)

	expectedLastReportedAtBlock := uint64(3)
	require.Equal(t, expectedLastReportedAtBlock, lastReportedAtBlock)
}

func TestPruneOldReports(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporter1 := sample.AccAddressBytes()
	reporter2 := sample.AccAddressBytes()

	// Current time: Day 100
	// Cutoff: Day 40 (100 - 60 = 40)
	// Anything with timestamp < Day 40 is old and should be deleted
	now := time.Now()
	ctx = ctx.WithBlockTime(now).WithBlockHeight(500)

	// Set up Report entries at different block heights
	// Block 100: old (timestamp Day 10)
	// Block 200: old (timestamp Day 30)
	// Block 300: recent (timestamp Day 50)
	// Block 400: recent (timestamp Day 80)
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid1"), collections.Join(reporter1.Bytes(), uint64(100))), types.DelegationsAmounts{Total: math.OneInt()}))
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid2"), collections.Join(reporter1.Bytes(), uint64(200))), types.DelegationsAmounts{Total: math.OneInt()}))
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid1"), collections.Join(reporter2.Bytes(), uint64(100))), types.DelegationsAmounts{Total: math.OneInt()})) // same block as reporter1
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid1"), collections.Join(reporter1.Bytes(), uint64(300))), types.DelegationsAmounts{Total: math.OneInt()}))
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte("queryid1"), collections.Join(reporter1.Bytes(), uint64(400))), types.DelegationsAmounts{Total: math.OneInt()}))

	// Set up mock OracleKeeper
	// GetBlockHeightFromTimestamp returns the cutoff block height.
	// Blocks 100, 200 are below cutoff (old), blocks 300, 400 are above (recent).
	oracleKeeper := mocks.NewOracleKeeper(t)
	oracleKeeper.On("GetBlockHeightFromTimestamp", mock.Anything, mock.Anything).Return(uint64(250), nil)

	k.SetOracleKeeper(oracleKeeper)

	// Prune old reports
	err := k.PruneOldReports(ctx, 100)
	require.NoError(t, err)

	// reporter1 block 100 deleted (not the most recent old entry)
	_, err = k.Report.Get(ctx, collections.Join([]byte("queryid1"), collections.Join(reporter1.Bytes(), uint64(100))))
	require.Error(t, err) // should be deleted

	// reporter1 block 200 kept (most recent old entry in batch)
	_, err = k.Report.Get(ctx, collections.Join([]byte("queryid2"), collections.Join(reporter1.Bytes(), uint64(200))))
	require.NoError(t, err) // kept as most recent old entry

	// reporter2 block 100 kept (only entry, most recent old entry)
	_, err = k.Report.Get(ctx, collections.Join([]byte("queryid1"), collections.Join(reporter2.Bytes(), uint64(100))))
	require.NoError(t, err) // kept as most recent old entry

	// Recent entries still exist (block 300, 400)
	_, err = k.Report.Get(ctx, collections.Join([]byte("queryid1"), collections.Join(reporter1.Bytes(), uint64(300))))
	require.NoError(t, err)

	_, err = k.Report.Get(ctx, collections.Join([]byte("queryid1"), collections.Join(reporter1.Bytes(), uint64(400))))
	require.NoError(t, err)
}

// Saving a new snapshot clears the reporter's own old ones but keeps its newest one before the cutoff.
func TestReporterStake_SavingSnapshotCleansUpOldOnes(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockTime(time.Now()).WithBlockHeight(500)
	reporter := sample.AccAddressBytes()
	otherReporter := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())

	require.NoError(t, k.Reporters.Set(ctx, reporter, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter")))
	require.NoError(t, k.Selectors.Set(ctx, reporter, types.NewSelection(reporter, 1)))
	validatorSet := new(mocks.ValidatorSet)
	sk.On("GetValidatorSet").Return(validatorSet)
	validatorSet.On("MaxValidators", mock.Anything).Return(uint32(10), nil)
	sk.On("GetValidator", mock.Anything, valAddr).Return(stakingtypes.Validator{
		OperatorAddress: valAddr.String(), DelegatorShares: math.LegacyNewDec(1000), Status: stakingtypes.Bonded, Tokens: math.NewInt(1000),
	}, nil)
	sk.On("IterateDelegatorDelegations", mock.Anything, reporter, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		fn(stakingtypes.Delegation{ValidatorAddress: valAddr.String(), Shares: math.LegacyNewDec(1000)})
	})

	key := func(rep sdk.AccAddress, block, version uint64) collections.Triple[[]byte, uint64, uint64] {
		return collections.Join3(rep.Bytes(), block, version)
	}
	snap := types.DelegationsAmounts{Total: math.OneInt()}
	require.NoError(t, k.StakeSnapshots.Set(ctx, key(reporter, 100, 101), snap))
	require.NoError(t, k.StakeSnapshots.Set(ctx, key(reporter, 200, 102), snap))
	require.NoError(t, k.StakeSnapshots.Set(ctx, key(reporter, 200, 103), snap))
	require.NoError(t, k.StakeSnapshots.Set(ctx, key(reporter, 300, 104), snap))
	require.NoError(t, k.StakeSnapshots.Set(ctx, key(otherReporter, 100, 105), snap))

	// Block 250 is 30 days back.
	oracleKeeper := mocks.NewOracleKeeper(t)
	oracleKeeper.On("GetBlockHeightFromTimestamp", mock.Anything, mock.Anything).Return(uint64(250), nil)
	k.SetOracleKeeper(oracleKeeper)

	require.NoError(t, k.FlagStakeRecalc(ctx, reporter))
	_, ref, err := k.ReporterStakeWithSnapshot(ctx, reporter)
	require.NoError(t, err)
	require.Equal(t, uint64(500), ref.Block)

	exists := func(rep sdk.AccAddress, block, version uint64) bool {
		has, err := k.StakeSnapshots.Has(ctx, key(rep, block, version))
		require.NoError(t, err)
		return has
	}
	// Older snapshots before the cutoff are removed.
	require.False(t, exists(reporter, 100, 101))
	require.False(t, exists(reporter, 200, 102))
	// The newest one before the cutoff stays, since reports just after the cutoff may still use it.
	require.True(t, exists(reporter, 200, 103))
	// Recent snapshots, the new one, and other reporters' snapshots are untouched.
	require.True(t, exists(reporter, 300, 104))
	require.True(t, exists(reporter, ref.Block, ref.Version))
	require.True(t, exists(otherReporter, 100, 105))
}

// Each cleanup removes at most the cap, so the work per report stays small.
func TestCleanupOldStakeSnapshots_Capped(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockTime(time.Now()).WithBlockHeight(500)
	reporter := sample.AccAddressBytes()

	oldCount := keeper.MaxStakeSnapshotCleanup + 5
	for i := 1; i <= oldCount; i++ {
		require.NoError(t, k.StakeSnapshots.Set(ctx, collections.Join3(reporter.Bytes(), uint64(i), uint64(i)), types.DelegationsAmounts{Total: math.OneInt()}))
	}
	oracleKeeper := mocks.NewOracleKeeper(t)
	oracleKeeper.On("GetBlockHeightFromTimestamp", mock.Anything, mock.Anything).Return(uint64(250), nil)
	k.SetOracleKeeper(oracleKeeper)

	remaining := func() int {
		iter, err := k.StakeSnapshots.Iterate(ctx, nil)
		require.NoError(t, err)
		keys, err := iter.Keys()
		require.NoError(t, err)
		return len(keys)
	}

	require.NoError(t, k.CleanupOldStakeSnapshotsExported(ctx, reporter.Bytes()))
	require.Equal(t, oldCount-keeper.MaxStakeSnapshotCleanup, remaining())

	// The next cleanup finishes the job but keeps the newest one before the cutoff.
	require.NoError(t, k.CleanupOldStakeSnapshotsExported(ctx, reporter.Bytes()))
	require.Equal(t, 1, remaining())
	has, err := k.StakeSnapshots.Has(ctx, collections.Join3(reporter.Bytes(), uint64(oldCount), uint64(oldCount)))
	require.NoError(t, err)
	require.True(t, has)
}

func TestPruneOldReportsMaxIterations(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporter := sample.AccAddressBytes()

	now := time.Now()
	ctx = ctx.WithBlockTime(now).WithBlockHeight(500)

	// Set up 5 old Report entries
	for i := uint64(1); i <= 5; i++ {
		require.NoError(t, k.Report.Set(ctx, collections.Join(fmt.Appendf(nil, "queryid%d", i), collections.Join(reporter.Bytes(), i)), types.DelegationsAmounts{Total: math.OneInt()}))
	}

	oracleKeeper := mocks.NewOracleKeeper(t)
	oracleKeeper.On("GetBlockHeightFromTimestamp", mock.Anything, mock.Anything).Return(uint64(400), nil)
	k.SetOracleKeeper(oracleKeeper)

	// Prune with maxBatchSize = 3 (should only delete 3 entries)
	err := k.PruneOldReports(ctx, 3)
	require.NoError(t, err)

	// Count remaining entries
	count := 0
	err = k.Report.Walk(ctx, nil, func(key collections.Pair[[]byte, collections.Pair[[]byte, uint64]], value types.DelegationsAmounts) (stop bool, err error) {
		count++
		return false, nil
	})
	require.NoError(t, err)

	// Batch collected 3 reports (blocks 1,2,3). Block 3 is kept as the
	// most recent for this reporter within the batch. Blocks 1,2 deleted.
	// Blocks 3,4,5 remain.
	require.Equal(t, 3, count)
}

func TestPruneNewCollectionBlockOrder(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporter1 := sample.AccAddressBytes()
	reporter2 := sample.AccAddressBytes()

	now := time.Now()
	ctx = ctx.WithBlockTime(now).WithBlockHeight(500)

	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporter1.Bytes(), uint64(50), []byte("q1")), types.DelegationsAmounts{Total: math.OneInt()}))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporter2.Bytes(), uint64(300), []byte("q1")), types.DelegationsAmounts{Total: math.OneInt()}))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporter1.Bytes(), uint64(10), []byte("q1")), types.DelegationsAmounts{Total: math.OneInt()}))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporter2.Bytes(), uint64(200), []byte("q1")), types.DelegationsAmounts{Total: math.OneInt()}))
	require.NoError(t, k.ReportByBlock.Set(ctx, collections.Join3(reporter1.Bytes(), uint64(400), []byte("q1")), types.DelegationsAmounts{Total: math.OneInt()}))

	// Verify Multi index iterates in ascending blockNumber order
	iter, err := k.ReportByBlock.Indexes.BlockNumber.Iterate(ctx, nil)
	require.NoError(t, err)
	defer iter.Close()

	var blockNumbers []uint64
	for ; iter.Valid(); iter.Next() {
		pk, err := iter.PrimaryKey()
		require.NoError(t, err)
		blockNumbers = append(blockNumbers, pk.K2())
	}
	require.Equal(t, []uint64{10, 50, 200, 300, 400}, blockNumbers)

	// Now prune with cutoff at 250 — should delete blocks 10, 50, 200 and stop
	oracleKeeper := mocks.NewOracleKeeper(t)
	oracleKeeper.On("GetBlockHeightFromTimestamp", mock.Anything, mock.Anything).Return(uint64(250), nil)
	k.SetOracleKeeper(oracleKeeper)

	err = k.PruneOldReports(ctx, 100)
	require.NoError(t, err)

	// Verify blocks 300, 400 remain plus old entry for reporter1 at block 50 and reporter2 at block 200
	// plus their new entries at block 300 and 400 so 4 should remain
	remaining := 0
	err = k.ReportByBlock.Walk(ctx, nil, func(_ collections.Triple[[]byte, uint64, []byte], _ types.DelegationsAmounts) (bool, error) {
		remaining++
		return false, nil
	})
	require.NoError(t, err)
	require.Equal(t, 4, remaining)

	// Verify the correct entries survived
	_, err = k.ReportByBlock.Get(ctx, collections.Join3(reporter2.Bytes(), uint64(300), []byte("q1")))
	require.NoError(t, err)
	_, err = k.ReportByBlock.Get(ctx, collections.Join3(reporter1.Bytes(), uint64(400), []byte("q1")))
	require.NoError(t, err)

	// Verify most recent old entries are still
	_, err = k.ReportByBlock.Get(ctx, collections.Join3(reporter1.Bytes(), uint64(50), []byte("q1")))
	require.NoError(t, err) // kept as most recent old entry for reporter1
	_, err = k.ReportByBlock.Get(ctx, collections.Join3(reporter2.Bytes(), uint64(200), []byte("q1")))
	require.NoError(t, err) // kept as most recent old entry for reporter2

	// Verify oldest entry is deleted
	_, err = k.ReportByBlock.Get(ctx, collections.Join3(reporter1.Bytes(), uint64(10), []byte("q1")))
	require.Error(t, err)
}

func TestStakeRecalcFlag(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporter := sample.AccAddressBytes()

	// Initially no flag
	has, err := k.StakeRecalcFlag.Has(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.False(t, has)

	// Set flag
	require.NoError(t, k.FlagStakeRecalc(ctx, reporter))

	// Now flag should exist
	has, err = k.StakeRecalcFlag.Has(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.True(t, has)

	// Remove flag
	require.NoError(t, k.StakeRecalcFlag.Remove(ctx, reporter.Bytes()))

	// Flag should be gone
	has, err = k.StakeRecalcFlag.Has(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.False(t, has)
}

func TestLastValSetUpdateHeight(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)

	// Initially not set
	_, err := k.LastValSetUpdateHeight.Get(ctx)
	require.Error(t, err)

	// Set it
	require.NoError(t, k.LastValSetUpdateHeight.Set(ctx, uint64(100)))

	// Should be retrievable
	height, err := k.LastValSetUpdateHeight.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), height)

	// Update it
	require.NoError(t, k.LastValSetUpdateHeight.Set(ctx, uint64(200)))
	height, err = k.LastValSetUpdateHeight.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(200), height)
}

func TestRecalcAtTime(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporter := sample.AccAddressBytes()

	// Initially not set
	_, err := k.RecalcAtTime.Get(ctx, reporter.Bytes())
	require.Error(t, err)

	// Set it
	lockExpiry := time.Now().Add(21 * 24 * time.Hour).Unix()
	require.NoError(t, k.RecalcAtTime.Set(ctx, reporter.Bytes(), lockExpiry))

	// Should be retrievable
	stored, err := k.RecalcAtTime.Get(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.Equal(t, lockExpiry, stored)

	// Remove it
	require.NoError(t, k.RecalcAtTime.Remove(ctx, reporter.Bytes()))
	_, err = k.RecalcAtTime.Get(ctx, reporter.Bytes())
	require.Error(t, err)
}

func TestRecalcAtTime_MinOfMultipleSwitches(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporter := sample.AccAddressBytes()

	// First switch: lock expires at T1
	t1 := time.Now().Add(21 * 24 * time.Hour).Unix()
	require.NoError(t, k.RecalcAtTime.Set(ctx, reporter.Bytes(), t1))

	// Second switch: lock expires at T2 > T1, should keep T1 (the min)
	t2 := time.Now().Add(30 * 24 * time.Hour).Unix()
	existing, err := k.RecalcAtTime.Get(ctx, reporter.Bytes())
	require.NoError(t, err)
	if t2 < existing {
		require.NoError(t, k.RecalcAtTime.Set(ctx, reporter.Bytes(), t2))
	}
	stored, err := k.RecalcAtTime.Get(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.Equal(t, t1, stored)

	// Third switch: lock expires at T0 < T1, should update to T0
	t0 := time.Now().Add(10 * 24 * time.Hour).Unix()
	existing, err = k.RecalcAtTime.Get(ctx, reporter.Bytes())
	require.NoError(t, err)
	if t0 < existing {
		require.NoError(t, k.RecalcAtTime.Set(ctx, reporter.Bytes(), t0))
	}
	stored, err = k.RecalcAtTime.Get(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.Equal(t, t0, stored)
}
