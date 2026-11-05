package keeper_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestRegisterTwoBitReporter(t *testing.T) {
	k, _, _, _, _, ms, ctx := setupMsgServer(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	reporter := sample.AccAddressBytes()
	agent := sample.AccAddressBytes()
	other := sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "moniker")
	require.NoError(t, k.Reporters.Set(ctx, reporter.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, other.Bytes(), rep))

	_, err := ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: reporter.String(),
		TwoBitAddress:   agent.String(),
	})
	require.NoError(t, err)

	got, err := k.TwoBitReporter(ctx, agent)
	require.NoError(t, err)
	require.True(t, bytes.Equal(reporter.Bytes(), got.Bytes()))
	storedAgent, err := k.TwoBitReporterToAgent.Get(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(agent.Bytes(), storedAgent))

	// Replacing deletes the previous agent key.
	next := sample.AccAddressBytes()
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: reporter.String(),
		TwoBitAddress:   next.String(),
	})
	require.NoError(t, err)
	_, err = k.TwoBitAgentToReporter.Get(ctx, agent.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)
	got, err = k.TwoBitReporter(ctx, next)
	require.NoError(t, err)
	require.True(t, bytes.Equal(reporter.Bytes(), got.Bytes()))

	// Another reporter cannot take an agent that is already registered.
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: other.String(),
		TwoBitAddress:   next.String(),
	})
	require.ErrorIs(t, err, types.ErrTwoBitAgentTaken)

	// The agent cannot be a reporter, and cannot be the reporter's own address.
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: reporter.String(),
		TwoBitAddress:   other.String(),
	})
	require.ErrorIs(t, err, types.ErrTwoBitAgentIsReporter)
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: reporter.String(),
		TwoBitAddress:   reporter.String(),
	})
	require.ErrorIs(t, err, types.ErrInvalidReporter)

	// Empty address clears both indexes.
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: reporter.String(),
	})
	require.NoError(t, err)
	hasAgent, err := k.TwoBitAgentToReporter.Has(ctx, next.Bytes())
	require.NoError(t, err)
	require.False(t, hasAgent)
	hasReporter, err := k.TwoBitReporterToAgent.Has(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.False(t, hasReporter)

	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: sample.AccAddressBytes().String(),
		TwoBitAddress:   sample.AccAddressBytes().String(),
	})
	require.ErrorIs(t, err, types.ErrReporterDoesNotExist)

	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: "not-an-address",
	})
	require.Error(t, err)
}

func TestCreateReporterRejectsTwoBitAgent(t *testing.T) {
	k, _, _, _, _, ms, ctx := setupMsgServer(t)
	agent, reporter := sample.AccAddressBytes(), sample.AccAddressBytes()
	require.NoError(t, k.TwoBitAgentToReporter.Set(ctx, agent.Bytes(), reporter.Bytes()))

	_, err := ms.CreateReporter(ctx, &types.MsgCreateReporter{
		ReporterAddress:   agent.String(),
		CommissionRate:    types.DefaultMinCommissionRate,
		MinTokensRequired: types.DefaultMinLoya,
		Moniker:           "moniker",
	})
	require.ErrorIs(t, err, types.ErrAddressIsTwoBitAgent)
}

func TestRemoveReporterClearsTwoBitMapping(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	reporter := sample.AccAddressBytes()
	agent := sample.AccAddressBytes()
	require.NoError(t, k.Reporters.Set(ctx, reporter.Bytes(), types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "moniker")))
	require.NoError(t, k.SetTwoBitReporter(ctx, reporter, agent))

	require.NoError(t, k.RemoveReporterForTest(ctx, reporter.Bytes()))

	hasReporter, err := k.Reporters.Has(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.False(t, hasReporter)
	hasAgent, err := k.TwoBitAgentToReporter.Has(ctx, agent.Bytes())
	require.NoError(t, err)
	require.False(t, hasAgent)
	hasReverse, err := k.TwoBitReporterToAgent.Has(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.False(t, hasReverse)
}
