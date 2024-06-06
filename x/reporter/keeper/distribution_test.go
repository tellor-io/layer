package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestDivvyingTips(t *testing.T) {
	k, _, _, ctx := keepertest.ReporterKeeper(t)
	height := int64(10)
	val1Address := sample.AccAddressBytes()
	vals := simtestutil.ConvertAddrsToValAddrs([]sdk.AccAddress{val1Address})
	val1 := vals[0]
	addr := sample.AccAddressBytes()
	addr2 := sample.AccAddressBytes()
	updatedAt := time.Now().UTC()
	commission := types.NewCommissionWithTime(types.DefaultMinCommissionRate, types.DefaultMinCommissionRate.MulInt(math.NewInt(2)), types.DefaultMinCommissionRate, updatedAt)
	reporter1 := types.NewOracleReporter(addr.String(), math.NewInt(2000*1e6), &commission, 1)
	ctx = ctx.WithBlockHeight(height)

	err := k.Reporters.Set(ctx, addr, reporter1)
	require.NoError(t, err)

	tokenOrigin1 := &types.TokenOriginInfo{
		DelegatorAddress: addr.Bytes(),
		ValidatorAddress: val1.Bytes(),
		Amount:           math.NewInt(1000 * 1e6),
	}

	tokenOrigin2 := &types.TokenOriginInfo{
		DelegatorAddress: addr2.Bytes(),
		ValidatorAddress: val1.Bytes(),
		Amount:           math.NewInt(1000 * 1e6),
	}
	tokenOrigins := []*types.TokenOriginInfo{tokenOrigin1, tokenOrigin2}
	total := tokenOrigin1.Amount.Add(tokenOrigin2.Amount)

	delegationAmounts := types.DelegationsAmounts{TokenOrigins: tokenOrigins, Total: total}

	err = k.Report.Set(ctx, collections.Join(addr.Bytes(), height), delegationAmounts)
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(12)
	err = k.DivvyingTips(ctx, addr, math.NewInt(10*1e6), 10)
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(13)
	del1, err := k.DelegatorTips.Get(ctx, addr.Bytes())
	require.NoError(t, err)
	del2, err := k.DelegatorTips.Get(ctx, addr2.Bytes())

	fmt.Printf("delegator1: %v, delegator2: %v\r", del1, del2)
	require.Equal(t, math.NewInt(5*1e6), del1)

	require.NoError(t, err)
	require.Equal(t, math.NewInt(5*1e6), del2)
}

// func TestReturnSlashedTokens(t *testing.T) {
// 	k, sk, bk, ctx := keepertest.ReporterKeeper(t)

// 	val1Address := sample.AccAddressBytes()
// 	vals := simtestutil.ConvertAddrsToValAddrs([]sdk.AccAddress{val1Address})
// 	val1 := vals[0]
// 	addr := sample.AccAddressBytes()
// 	addr2 := sample.AccAddressBytes()
// 	updatedAt := time.Now().UTC()
// 	commission := types.NewCommissionWithTime(types.DefaultMinCommissionRate, types.DefaultMinCommissionRate.MulInt(math.NewInt(2)), types.DefaultMinCommissionRate, updatedAt)
// 	reporter1 := types.NewOracleReporter(addr.String(), math.NewInt(2000*1e6), &commission)
// 	//reporter2 := types.NewOracleReporter(addr2.String(), math.NewInt(1000*1e6), &commission)

// 	err := k.Reporters.Set(ctx, addr, reporter1)
// 	require.NoError(t, err)

// 	tokenOrigin1 := &types.TokenOriginInfo{
// 		DelegatorAddress: addr.Bytes(),
// 		ValidatorAddress: val1.Bytes(),
// 		Amount:           math.NewInt(1000 * 1e6),
// 	}

// 	tokenOrigin2 := &types.TokenOriginInfo{
// 		DelegatorAddress: addr2.Bytes(),
// 		ValidatorAddress: val1.Bytes(),
// 		Amount:           math.NewInt(1000 * 1e6),
// 	}
// 	tokenOrigins := []*types.TokenOriginInfo{tokenOrigin1, tokenOrigin2}
// 	total := tokenOrigin1.Amount.Add(tokenOrigin2.Amount)

// 	delegationAmounts := types.DelegationsAmounts{TokenOrigins: tokenOrigins, Total: total}
// }
