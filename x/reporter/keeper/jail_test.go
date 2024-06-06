package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"
)

func TestJailReporter(t *testing.T) {
	k, _, _, ctx := keepertest.ReporterKeeper(t)
	addr := sample.AccAddressBytes()
	updatedAt := time.Now().UTC()
	commission := types.NewCommissionWithTime(types.DefaultMinCommissionRate, types.DefaultMinCommissionRate.MulInt(math.NewInt(2)), types.DefaultMinCommissionRate, updatedAt)
	reporter := types.NewOracleReporter(addr.String(), math.NewInt(1000*1e6), &commission, 1)

	err := k.Reporters.Set(ctx, addr, reporter)
	require.NoError(t, err)

	ctx = ctx.WithBlockTime(updatedAt.Add(time.Second * 10))
	jailedDuration := int64(100)

	err = k.JailReporter(ctx, addr, jailedDuration)
	require.NoError(t, err)

	ctx = ctx.WithBlockTime(updatedAt.Add(time.Second * 15))
	updatedReporter, err := k.Reporters.Get(ctx, addr)
	require.NoError(t, err)
	require.Equal(t, true, updatedReporter.Jailed)
	require.Equal(t, updatedAt.Add(time.Second*110), updatedReporter.JailedUntil)
}

func TestUnJailReporter(t *testing.T) {
	k, _, _, ctx := keepertest.ReporterKeeper(t)
	addr := sample.AccAddressBytes()
	jailedAt := time.Now().UTC()
	commission := types.NewCommissionWithTime(types.DefaultMinCommissionRate, types.DefaultMinCommissionRate.MulInt(math.NewInt(2)), types.DefaultMinCommissionRate, jailedAt)
	reporter := types.NewOracleReporter(addr.String(), math.NewInt(1000*1e6), &commission, 1)
	reporter.Jailed = true
	reporter.JailedUntil = jailedAt.Add(time.Second * 100)
	ctx = ctx.WithBlockTime(jailedAt.Add(time.Second * 50))

	// test unjailing reporter before the JailedUntil time
	err := k.UnjailReporter(ctx, addr, reporter)
	require.Error(t, err)

	// test unjailing after enough time has passed
	ctx = ctx.WithBlockTime(jailedAt.Add(time.Second * 505))
	err = k.UnjailReporter(ctx, addr, reporter)
	require.NoError(t, err)

	updatedReporter, err := k.Reporters.Get(ctx, addr)
	require.NoError(t, err)
	require.Equal(t, false, updatedReporter.Jailed)

	err = k.UnjailReporter(ctx, addr, updatedReporter)
	require.Error(t, err)
}
