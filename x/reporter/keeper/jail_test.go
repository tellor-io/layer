package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"
)

func TestJailReporter(t *testing.T) {
	k, _, _, _, ctx, _ := setupKeeper(t)
	addr := sample.AccAddressBytes()
	updatedAt := time.Now().UTC()
	reporter := types.NewReporter(types.DefaultMinCommissionRate, math.OneInt())

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
	k, _, _, _, ctx, _ := setupKeeper(t)
	addr := sample.AccAddressBytes()
	jailedAt := time.Now().UTC()
	reporter := types.NewReporter(types.DefaultMinCommissionRate, math.OneInt())
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
