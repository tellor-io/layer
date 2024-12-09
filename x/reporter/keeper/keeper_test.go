package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupKeeper(tb testing.TB) (keeper.Keeper, *mocks.OracleKeeper, *mocks.StakingKeeper, *mocks.BankKeeper, *mocks.RegistryKeeper, sdk.Context, store.KVStoreService) {
	tb.Helper()
	return keepertest.ReporterKeeper(tb)
}

func TestKeeper(t *testing.T) {
	k, ok, sk, bk, _, ctx, _ := keepertest.ReporterKeeper(t)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
	require.NotNil(t, ok)
	require.NotNil(t, sk)
	require.NotNil(t, bk)
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

func TestTrackStakeChange(t *testing.T) {
	k, _, sk, _, _, ctx, _ := setupKeeper(t)
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

func TestRewardTip(t *testing.T) {
	k, ok, _, _, _, ctx, _ := setupKeeper(t)
	// make three reporters, each with power of 15 and total power of 45
	// make each reporter have three selectors each with power of 5 which makes the reporters power 15
	// reporter1
	reward := math.NewInt(100)
	_ = reward
	metaId := uint64(1)
	queryId := []byte("queryid")
	height := uint64(ctx.BlockHeight())
	reporter1, selector1b, selector1c := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	require.NoError(t, k.Report.Set(ctx, collections.Join(queryId, collections.Join(reporter1.Bytes(), height)), types.DelegationsAmounts{
		Total: math.NewInt(15).Mul(layertypes.PowerReduction),
		TokenOrigins: []*types.TokenOriginInfo{
			{
				DelegatorAddress: reporter1,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector1b,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector1c,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
		},
	}))

	reporter2, selector2b, selector2c := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	require.NoError(t, k.Report.Set(ctx, collections.Join(queryId, collections.Join(reporter2.Bytes(), height)), types.DelegationsAmounts{
		Total: math.NewInt(15).Mul(layertypes.PowerReduction),
		TokenOrigins: []*types.TokenOriginInfo{
			{
				DelegatorAddress: reporter2,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector2b,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector2c,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
		},
	}))

	reporter3, selector3b, selector3c := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	require.NoError(t, k.Report.Set(ctx, collections.Join(queryId, collections.Join(reporter3.Bytes(), height)), types.DelegationsAmounts{
		Total: math.NewInt(15).Mul(layertypes.PowerReduction),
		TokenOrigins: []*types.TokenOriginInfo{
			{
				DelegatorAddress: reporter3,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector3b,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector3c,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
		},
	}))

	reporters := []sdk.AccAddress{reporter1, reporter2, reporter3}
	for _, r := range reporters {
		require.NoError(t, k.Reporters.Set(ctx, r, types.OracleReporter{CommissionRate: math.LegacyMustNewDecFromStr("0.5")})) // 50%
	}
	// query id gets aggregated then AddTip, skip tbr for now
	require.NoError(t, k.AddTip(ctx, metaId, oracletypes.Reward{
		TotalPower:  45,
		Amount:      reward.ToLegacyDec(),
		CycleList:   false,
		BlockHeight: height + 1,
	}))

	// microreports
	ok.On("MicroReport", ctx, collections.Join3(queryId, reporter1.Bytes(), metaId)).Return(oracletypes.MicroReport{
		BlockNumber: height,
		Power:       uint64(15),
	}, nil)

	ok.On("MicroReport", ctx, collections.Join3(queryId, reporter2.Bytes(), metaId)).Return(oracletypes.MicroReport{
		BlockNumber: height,
		Power:       uint64(15),
	}, nil)

	ok.On("MicroReport", ctx, collections.Join3(queryId, reporter3.Bytes(), metaId)).Return(oracletypes.MicroReport{
		BlockNumber: height,
		Power:       uint64(15),
	}, nil)

	type Reporter struct {
		selectors []sdk.AccAddress
	}
	reporterA := Reporter{
		selectors: []sdk.AccAddress{reporter1, selector1b, selector1c},
	}
	reporterB := Reporter{
		selectors: []sdk.AccAddress{reporter2, selector2b, selector2c},
	}
	reporterC := Reporter{
		selectors: []sdk.AccAddress{reporter3, selector3b, selector3c},
	}
	all := append([]Reporter{reporterA}, reporterB, reporterC)
	total := math.LegacyZeroDec()
	for i, r := range all {
		for _, s := range r.selectors {
			share, err := k.RewardByReporter(ctx, s, reporters[i], metaId, queryId)
			require.NoError(t, err)
			total = total.Add(share)
		}
	}
	fmt.Println("total", total)
	require.True(t, total.Equal(reward.ToLegacyDec()))
}

func TestRewardTrb(t *testing.T) {
	k, ok, _, _, _, ctx, _ := setupKeeper(t)
	reward := math.NewInt(100)

	metaId := uint64(1)
	queryId := []byte("queryid")
	height := uint64(ctx.BlockHeight())
	reporter1, selector1b, selector1c := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	require.NoError(t, k.Report.Set(ctx, collections.Join(queryId, collections.Join(reporter1.Bytes(), height)), types.DelegationsAmounts{
		Total: math.NewInt(15).Mul(layertypes.PowerReduction),
		TokenOrigins: []*types.TokenOriginInfo{
			{
				DelegatorAddress: reporter1,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector1b,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector1c,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
		},
	}))

	reporter2, selector2b, selector2c := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	require.NoError(t, k.Report.Set(ctx, collections.Join(queryId, collections.Join(reporter2.Bytes(), height)), types.DelegationsAmounts{
		Total: math.NewInt(15).Mul(layertypes.PowerReduction),
		TokenOrigins: []*types.TokenOriginInfo{
			{
				DelegatorAddress: reporter2,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector2b,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector2c,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
		},
	}))

	reporter3, selector3b, selector3c := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	require.NoError(t, k.Report.Set(ctx, collections.Join(queryId, collections.Join(reporter3.Bytes(), height)), types.DelegationsAmounts{
		Total: math.NewInt(15).Mul(layertypes.PowerReduction),
		TokenOrigins: []*types.TokenOriginInfo{
			{
				DelegatorAddress: reporter3,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector3b,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector3c,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
		},
	}))

	reporters := []sdk.AccAddress{reporter1, reporter2, reporter3}
	for _, r := range reporters {
		require.NoError(t, k.Reporters.Set(ctx, r, types.OracleReporter{CommissionRate: math.LegacyMustNewDecFromStr("0.5")})) // 50%
	}

	require.NoError(t, k.AddTip(ctx, metaId, oracletypes.Reward{
		TotalPower:  45,
		Amount:      math.LegacyZeroDec(),
		CycleList:   true,
		BlockHeight: height + 1,
	}))

	require.NoError(t, k.AddTbr(ctx, metaId, oracletypes.Reward{
		TotalPower:  45,
		Amount:      reward.ToLegacyDec(),
		CycleList:   true,
		BlockHeight: height + 1,
	}))

	// microreports
	ok.On("MicroReport", ctx, collections.Join3(queryId, reporter1.Bytes(), metaId)).Return(oracletypes.MicroReport{
		BlockNumber: height,
		Power:       uint64(15),
	}, nil)

	ok.On("MicroReport", ctx, collections.Join3(queryId, reporter2.Bytes(), metaId)).Return(oracletypes.MicroReport{
		BlockNumber: height,
		Power:       uint64(15),
	}, nil)

	ok.On("MicroReport", ctx, collections.Join3(queryId, reporter3.Bytes(), metaId)).Return(oracletypes.MicroReport{
		BlockNumber: height,
		Power:       uint64(15),
	}, nil)

	type Reporter struct {
		selectors []sdk.AccAddress
	}
	reporterA := Reporter{
		selectors: []sdk.AccAddress{reporter1, selector1b, selector1c},
	}
	reporterB := Reporter{
		selectors: []sdk.AccAddress{reporter2, selector2b, selector2c},
	}
	reporterC := Reporter{
		selectors: []sdk.AccAddress{reporter3, selector3b, selector3c},
	}
	all := append([]Reporter{reporterA}, reporterB, reporterC)
	total := math.LegacyZeroDec()
	for i, r := range all {
		for _, s := range r.selectors {
			share, err := k.RewardByReporter(ctx, s, reporters[i], metaId, queryId)
			require.NoError(t, err)
			total = total.Add(share)
		}
	}
	require.True(t, total.Equal(reward.ToLegacyDec()))
}

func TestRewardTrbTip(t *testing.T) {
	k, ok, _, _, _, ctx, _ := setupKeeper(t)
	tip := math.NewInt(100)
	tbr := math.NewInt(50)
	reward := tip.Add(tbr)
	metaId := uint64(1)
	queryId := []byte("queryid")
	height := uint64(ctx.BlockHeight())
	reporter1, selector1b, selector1c := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	require.NoError(t, k.Report.Set(ctx, collections.Join(queryId, collections.Join(reporter1.Bytes(), height)), types.DelegationsAmounts{
		Total: math.NewInt(15).Mul(layertypes.PowerReduction),
		TokenOrigins: []*types.TokenOriginInfo{
			{
				DelegatorAddress: reporter1,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector1b,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector1c,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
		},
	}))

	reporter2, selector2b, selector2c := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	require.NoError(t, k.Report.Set(ctx, collections.Join(queryId, collections.Join(reporter2.Bytes(), height)), types.DelegationsAmounts{
		Total: math.NewInt(15).Mul(layertypes.PowerReduction),
		TokenOrigins: []*types.TokenOriginInfo{
			{
				DelegatorAddress: reporter2,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector2b,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector2c,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
		},
	}))

	reporter3, selector3b, selector3c := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	require.NoError(t, k.Report.Set(ctx, collections.Join(queryId, collections.Join(reporter3.Bytes(), height)), types.DelegationsAmounts{
		Total: math.NewInt(15).Mul(layertypes.PowerReduction),
		TokenOrigins: []*types.TokenOriginInfo{
			{
				DelegatorAddress: reporter3,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector3b,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
			{
				DelegatorAddress: selector3c,
				Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
			},
		},
	}))

	reporters := []sdk.AccAddress{reporter1, reporter2, reporter3}
	for _, r := range reporters {
		require.NoError(t, k.Reporters.Set(ctx, r, types.OracleReporter{CommissionRate: math.LegacyMustNewDecFromStr("0.5")})) // 50%
	}

	require.NoError(t, k.AddTip(ctx, metaId, oracletypes.Reward{
		TotalPower:  45,
		Amount:      tip.ToLegacyDec(),
		CycleList:   true,
		BlockHeight: height + 1,
	}))

	require.NoError(t, k.AddTbr(ctx, height+1, oracletypes.Reward{
		TotalPower:  45,
		Amount:      tbr.ToLegacyDec(),
		CycleList:   true,
		BlockHeight: height + 1,
	}))

	// microreports
	ok.On("MicroReport", ctx, collections.Join3(queryId, reporter1.Bytes(), metaId)).Return(oracletypes.MicroReport{
		BlockNumber: height,
		Power:       uint64(15),
	}, nil)

	ok.On("MicroReport", ctx, collections.Join3(queryId, reporter2.Bytes(), metaId)).Return(oracletypes.MicroReport{
		BlockNumber: height,
		Power:       uint64(15),
	}, nil)

	ok.On("MicroReport", ctx, collections.Join3(queryId, reporter3.Bytes(), metaId)).Return(oracletypes.MicroReport{
		BlockNumber: height,
		Power:       uint64(15),
	}, nil)

	type Reporter struct {
		selectors []sdk.AccAddress
	}
	reporterA := Reporter{
		selectors: []sdk.AccAddress{reporter1, selector1b, selector1c},
	}
	reporterB := Reporter{
		selectors: []sdk.AccAddress{reporter2, selector2b, selector2c},
	}
	reporterC := Reporter{
		selectors: []sdk.AccAddress{reporter3, selector3b, selector3c},
	}
	all := append([]Reporter{reporterA}, reporterB, reporterC)
	total := math.LegacyZeroDec()
	for i, r := range all {
		for _, s := range r.selectors {
			share, err := k.RewardByReporter(ctx, s, reporters[i], metaId, queryId)
			require.NoError(t, err)
			total = total.Add(share)
		}
	}
	fmt.Println(total)
	require.True(t, total.Equal(reward.ToLegacyDec()))
}

// two queryids one cyclelist only and the other both tip n tbr
func TestRewardMix(t *testing.T) {
	k, ok, _, _, _, ctx, _ := setupKeeper(t)
	tip := math.NewInt(100)
	tbr := math.NewInt(50)
	metaIds := []uint64{1, 2}
	queryIds := [][]byte{
		[]byte("queryid"),
		[]byte("queryid2"),
	}

	height := uint64(ctx.BlockHeight())
	reporter1, selector1b, selector1c := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	for _, queryId := range queryIds {
		require.NoError(t, k.Report.Set(ctx, collections.Join(queryId, collections.Join(reporter1.Bytes(), height)), types.DelegationsAmounts{
			Total: math.NewInt(15).Mul(layertypes.PowerReduction),
			TokenOrigins: []*types.TokenOriginInfo{
				{
					DelegatorAddress: reporter1,
					Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
				},
				{
					DelegatorAddress: selector1b,
					Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
				},
				{
					DelegatorAddress: selector1c,
					Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
				},
			},
		}))
	}

	reporter2, selector2b, selector2c := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	for _, queryId := range queryIds {
		require.NoError(t, k.Report.Set(ctx, collections.Join(queryId, collections.Join(reporter2.Bytes(), height)), types.DelegationsAmounts{
			Total: math.NewInt(15).Mul(layertypes.PowerReduction),
			TokenOrigins: []*types.TokenOriginInfo{
				{
					DelegatorAddress: reporter2,
					Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
				},
				{
					DelegatorAddress: selector2b,
					Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
				},
				{
					DelegatorAddress: selector2c,
					Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
				},
			},
		}))
	}
	reporter3, selector3b, selector3c := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	for _, queryId := range queryIds {
		require.NoError(t, k.Report.Set(ctx, collections.Join(queryId, collections.Join(reporter3.Bytes(), height)), types.DelegationsAmounts{
			Total: math.NewInt(15).Mul(layertypes.PowerReduction),
			TokenOrigins: []*types.TokenOriginInfo{
				{
					DelegatorAddress: reporter3,
					Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
				},
				{
					DelegatorAddress: selector3b,
					Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
				},
				{
					DelegatorAddress: selector3c,
					Amount:           math.NewInt(5).Mul(layertypes.PowerReduction),
				},
			},
		}))
	}

	reporters := []sdk.AccAddress{reporter1, reporter2, reporter3}
	for _, r := range reporters {
		require.NoError(t, k.Reporters.Set(ctx, r, types.OracleReporter{CommissionRate: math.LegacyMustNewDecFromStr("0.5")})) // 50%
	}

	require.NoError(t, k.AddTip(ctx, metaIds[0], oracletypes.Reward{
		TotalPower:  45,
		Amount:      math.LegacyZeroDec(),
		CycleList:   true,
		BlockHeight: height + 1,
	}))

	require.NoError(t, k.AddTip(ctx, metaIds[1], oracletypes.Reward{
		TotalPower:  45,
		Amount:      tip.ToLegacyDec(),
		CycleList:   true,
		BlockHeight: height + 1,
	}))

	require.NoError(t, k.AddTbr(ctx, height+1, oracletypes.Reward{
		TotalPower:  90,
		Amount:      tbr.ToLegacyDec(),
		CycleList:   true,
		BlockHeight: height + 1,
	}))

	// microreports
	for i, queryId := range queryIds {
		ok.On("MicroReport", ctx, collections.Join3(queryId, reporter1.Bytes(), metaIds[i])).Return(oracletypes.MicroReport{
			BlockNumber: height,
			Power:       uint64(15),
		}, nil)
	}
	for i, queryId := range queryIds {
		ok.On("MicroReport", ctx, collections.Join3(queryId, reporter2.Bytes(), metaIds[i])).Return(oracletypes.MicroReport{
			BlockNumber: height,
			Power:       uint64(15),
		}, nil)
	}
	for i, queryId := range queryIds {
		ok.On("MicroReport", ctx, collections.Join3(queryId, reporter3.Bytes(), metaIds[i])).Return(oracletypes.MicroReport{
			BlockNumber: height,
			Power:       uint64(15),
		}, nil)
	}
	type Reporter struct {
		selectors []sdk.AccAddress
	}
	reporterA := Reporter{
		selectors: []sdk.AccAddress{reporter1, selector1b, selector1c},
	}
	reporterB := Reporter{
		selectors: []sdk.AccAddress{reporter2, selector2b, selector2c},
	}
	reporterC := Reporter{
		selectors: []sdk.AccAddress{reporter3, selector3b, selector3c},
	}
	all := []Reporter{reporterA, reporterB, reporterC}
	total := math.LegacyZeroDec()
	for m, queryId := range queryIds {
		for i, r := range all {
			for _, s := range r.selectors {
				share, err := k.RewardByReporter(ctx, s, reporters[i], metaIds[m], queryId)
				require.NoError(t, err)
				total = total.Add(share)
			}
		}
	}
	fmt.Println(total)
	require.True(t, total.Equal(tip.Add(tbr).ToLegacyDec()))
}
