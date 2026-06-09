package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
)

func TestFinalizePendingSwitchCleansOrphanWhenSelectorMissing(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockTime(time.Now()).WithBlockHeight(100)

	outgoing, incoming, selector := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "r")
	require.NoError(t, k.Reporters.Set(ctx, outgoing.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, incoming.Bytes(), rep))

	outPK := collections.Join(outgoing.Bytes(), selector.Bytes())
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, outPK, types.PendingSwitchEntry{
		ToReporter:  incoming.Bytes(),
		UnlockBlock: 1,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(incoming.Bytes(), selector.Bytes()), outgoing.Bytes()))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, outgoing.Bytes(), types.ReporterPendingSwitchHead{
		OutgoingCount:     1,
		OutgoingMinUnlock: 1,
	}))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, incoming.Bytes(), types.ReporterPendingSwitchHead{
		IncomingCount:     1,
		IncomingMinUnlock: 1,
	}))

	_, err := k.ReporterStake(ctx, incoming, []byte{})
	require.NoError(t, err)

	hasOut, err := k.OutgoingPendingSwitches.Has(ctx, outPK)
	require.NoError(t, err)
	require.False(t, hasOut)

	hasIn, err := k.IncomingPendingSwitchIdx.Has(ctx, collections.Join(incoming.Bytes(), selector.Bytes()))
	require.NoError(t, err)
	require.False(t, hasIn)

	_, err = k.ReporterPendingSwitchHeads.Get(ctx, outgoing.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)
	_, err = k.ReporterPendingSwitchHeads.Get(ctx, incoming.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func TestGetReporterStakeFinalizesReadyPendingSwitches(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockTime(time.Now()).WithBlockHeight(100)

	outgoing, incoming, selector := sample.AccAddressBytes(), sample.AccAddressBytes(), sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "r")
	require.NoError(t, k.Reporters.Set(ctx, outgoing.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, incoming.Bytes(), rep))

	outPK := collections.Join(outgoing.Bytes(), selector.Bytes())
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, outPK, types.PendingSwitchEntry{
		ToReporter:  incoming.Bytes(),
		UnlockBlock: 1,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(incoming.Bytes(), selector.Bytes()), outgoing.Bytes()))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, outgoing.Bytes(), types.ReporterPendingSwitchHead{
		OutgoingCount:     1,
		OutgoingMinUnlock: 1,
	}))
	require.NoError(t, k.ReporterPendingSwitchHeads.Set(ctx, incoming.Bytes(), types.ReporterPendingSwitchHead{
		IncomingCount:     1,
		IncomingMinUnlock: 1,
	}))

	_, _, _, _, err := k.GetReporterStake(ctx, incoming)
	require.NoError(t, err)

	hasOut, err := k.OutgoingPendingSwitches.Has(ctx, outPK)
	require.NoError(t, err)
	require.False(t, hasOut)

	hasIn, err := k.IncomingPendingSwitchIdx.Has(ctx, collections.Join(incoming.Bytes(), selector.Bytes()))
	require.NoError(t, err)
	require.False(t, hasIn)
}
