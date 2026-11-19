package keeper_test

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestRegisterTwoBitReporter(t *testing.T) {
	k, sk, _, _, _, ms, ctx := setupMsgServer(t)
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
	pending, err := k.TwoBitPendingRequests.Get(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(agent.Bytes(), pending.TwoBitReporter))
	require.Equal(t, uint64(ctx.BlockHeight())+types.TwoBitPendingRequestExpiryBlocks, pending.ExpiryHeight)
	// A request does not register the address, so it cannot block CreateReporter.
	_, err = k.TwoBitReporter(ctx, agent)
	require.ErrorIs(t, err, collections.ErrNotFound)
	requireEventAttr(t, ctx, "two_bit_registration_requested", "two_bit_address", agent.String())
	requireEventAttr(t, ctx, "two_bit_registration_requested", "expiry_height", strconv.FormatUint(pending.ExpiryHeight, 10))

	sk.On("IterateDelegatorDelegations", ctx, agent, mock.Anything).Return(nil)
	_, err = ms.CreateReporter(ctx, &types.MsgCreateReporter{
		ReporterAddress:   agent.String(),
		CommissionRate:    types.DefaultMinCommissionRate,
		MinTokensRequired: types.DefaultMinLoya,
		Moniker:           "moniker",
	})
	require.Error(t, err)
	require.NotErrorIs(t, err, types.ErrAddressIsTwoBitAgent)

	// Replacing the request leaves the previous address free and drops its ability to accept.
	next := sample.AccAddressBytes()
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: reporter.String(),
		TwoBitAddress:   next.String(),
	})
	require.NoError(t, err)
	pending, err = k.TwoBitPendingRequests.Get(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(next.Bytes(), pending.TwoBitReporter))
	_, err = ms.AcceptTwoBitRegistration(ctx, &types.MsgAcceptTwoBitRegistration{
		TwoBitAddress:   agent.String(),
		ReporterAddress: reporter.String(),
	})
	require.ErrorIs(t, err, types.ErrTwoBitRegistrationNotFound)

	_, err = ms.AcceptTwoBitRegistration(ctx, &types.MsgAcceptTwoBitRegistration{
		TwoBitAddress:   next.String(),
		ReporterAddress: reporter.String(),
	})
	require.NoError(t, err)
	got, err := k.TwoBitReporter(ctx, next)
	require.NoError(t, err)
	require.True(t, bytes.Equal(reporter.Bytes(), got.Bytes()))
	_, err = k.TwoBitPendingRequests.Get(ctx, reporter.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)
	requireEventAttr(t, ctx, "two_bit_reporter_registered", "reporter", reporter.String())
	requireEventAttr(t, ctx, "two_bit_reporter_registered", "two_bit_address", next.String())
	storedAgent, err := k.TwoBitReporterToAgent.Get(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(next.Bytes(), storedAgent))

	// Another reporter cannot request an agent that is already registered.
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: other.String(),
		TwoBitAddress:   next.String(),
	})
	require.ErrorIs(t, err, types.ErrTwoBitAgentTaken)

	// The agent cannot be a reporter. An empty address is rejected.
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: reporter.String(),
		TwoBitAddress:   other.String(),
	})
	require.ErrorIs(t, err, types.ErrTwoBitAgentIsReporter)
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: reporter.String(),
	})
	require.Error(t, err)
	hasAgent, err := k.TwoBitAgentToReporter.Has(ctx, next.Bytes())
	require.NoError(t, err)
	require.True(t, hasAgent)

	// Accepting a new agent replaces the previous one. The old agent keeps the slot until then.
	replacement := sample.AccAddressBytes()
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: reporter.String(),
		TwoBitAddress:   replacement.String(),
	})
	require.NoError(t, err)
	got, err = k.TwoBitReporter(ctx, next)
	require.NoError(t, err)
	require.True(t, bytes.Equal(reporter.Bytes(), got.Bytes()))
	_, err = ms.AcceptTwoBitRegistration(ctx, &types.MsgAcceptTwoBitRegistration{
		TwoBitAddress:   replacement.String(),
		ReporterAddress: reporter.String(),
	})
	require.NoError(t, err)
	_, err = k.TwoBitAgentToReporter.Get(ctx, next.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)
	got, err = k.TwoBitReporter(ctx, replacement)
	require.NoError(t, err)
	require.True(t, bytes.Equal(reporter.Bytes(), got.Bytes()))

	// The reporter's own address clears the active registration and any pending request.
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: reporter.String(),
		TwoBitAddress:   agent.String(),
	})
	require.NoError(t, err)
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: reporter.String(),
		TwoBitAddress:   reporter.String(),
	})
	require.NoError(t, err)
	foundClear := false
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == "two_bit_reporter_cleared" {
			foundClear = true
		}
	}
	require.True(t, foundClear)
	hasAgent, err = k.TwoBitAgentToReporter.Has(ctx, replacement.Bytes())
	require.NoError(t, err)
	require.False(t, hasAgent)
	hasReporter, err := k.TwoBitReporterToAgent.Has(ctx, reporter.Bytes())
	require.NoError(t, err)
	require.False(t, hasReporter)
	_, err = k.TwoBitPendingRequests.Get(ctx, reporter.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)
	_, err = ms.AcceptTwoBitRegistration(ctx, &types.MsgAcceptTwoBitRegistration{
		TwoBitAddress:   agent.String(),
		ReporterAddress: reporter.String(),
	})
	require.ErrorIs(t, err, types.ErrTwoBitRegistrationNotFound)

	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: sample.AccAddressBytes().String(),
		TwoBitAddress:   sample.AccAddressBytes().String(),
	})
	require.ErrorIs(t, err, types.ErrReporterDoesNotExist)

	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: "not-an-address",
	})
	require.Error(t, err)
	_, err = ms.AcceptTwoBitRegistration(ctx, &types.MsgAcceptTwoBitRegistration{
		TwoBitAddress:   "not-an-address",
		ReporterAddress: reporter.String(),
	})
	require.Error(t, err)
	_, err = ms.AcceptTwoBitRegistration(ctx, &types.MsgAcceptTwoBitRegistration{
		TwoBitAddress:   agent.String(),
		ReporterAddress: "not-an-address",
	})
	require.Error(t, err)
}

func TestAcceptTwoBitRegistrationRejectsTakenAndReporterAgent(t *testing.T) {
	k, _, _, _, _, ms, ctx := setupMsgServer(t)
	reporter := sample.AccAddressBytes()
	other := sample.AccAddressBytes()
	agent := sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "moniker")
	require.NoError(t, k.Reporters.Set(ctx, reporter.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, other.Bytes(), rep))

	// Two reporters can request the same free address. The first accept locks it.
	_, err := ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: reporter.String(),
		TwoBitAddress:   agent.String(),
	})
	require.NoError(t, err)
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: other.String(),
		TwoBitAddress:   agent.String(),
	})
	require.NoError(t, err)
	_, err = ms.AcceptTwoBitRegistration(ctx, &types.MsgAcceptTwoBitRegistration{
		TwoBitAddress:   agent.String(),
		ReporterAddress: reporter.String(),
	})
	require.NoError(t, err)
	_, err = ms.AcceptTwoBitRegistration(ctx, &types.MsgAcceptTwoBitRegistration{
		TwoBitAddress:   agent.String(),
		ReporterAddress: other.String(),
	})
	require.ErrorIs(t, err, types.ErrTwoBitAgentTaken)
	pending, err := k.TwoBitPendingRequests.Get(ctx, other.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(agent.Bytes(), pending.TwoBitReporter))

	// An address that became a reporter after the request cannot accept it.
	whale := sample.AccAddressBytes()
	require.NoError(t, k.Reporters.Set(ctx, whale.Bytes(), rep))
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: other.String(),
		TwoBitAddress:   whale.String(),
	})
	require.ErrorIs(t, err, types.ErrTwoBitAgentIsReporter)

	fresh := sample.AccAddressBytes()
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: other.String(),
		TwoBitAddress:   fresh.String(),
	})
	require.NoError(t, err)
	require.NoError(t, k.Reporters.Set(ctx, fresh.Bytes(), rep))
	_, err = ms.AcceptTwoBitRegistration(ctx, &types.MsgAcceptTwoBitRegistration{
		TwoBitAddress:   fresh.String(),
		ReporterAddress: other.String(),
	})
	require.ErrorIs(t, err, types.ErrTwoBitAgentIsReporter)
	_, err = k.TwoBitReporter(ctx, fresh)
	require.ErrorIs(t, err, collections.ErrNotFound)

	_, err = ms.AcceptTwoBitRegistration(ctx, &types.MsgAcceptTwoBitRegistration{
		TwoBitAddress:   agent.String(),
		ReporterAddress: sample.AccAddressBytes().String(),
	})
	require.ErrorIs(t, err, types.ErrReporterDoesNotExist)
	require.Error(t, k.AcceptTwoBitRegistration(ctx, nil, agent))
	require.Error(t, k.AcceptTwoBitRegistration(ctx, reporter, nil))
}

func TestTwoBitRequestRejectsNegativeHeight(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockHeight(-1)
	reporter := sample.AccAddressBytes()
	require.NoError(t, k.Reporters.Set(ctx, reporter.Bytes(), types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "moniker")))

	err := k.SetTwoBitReporter(ctx, reporter, sample.AccAddressBytes())
	require.Error(t, err)
	_, err = k.TwoBitPendingRequests.Get(ctx, reporter.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func requireEventAttr(t *testing.T, ctx sdk.Context, eventType, key, value string) {
	t.Helper()
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type != eventType {
			continue
		}
		for _, attr := range ev.Attributes {
			if attr.Key == key && attr.Value == value {
				return
			}
		}
	}
	t.Fatalf("event %s with %s=%s not found", eventType, key, value)
}

func TestTwoBitPendingRequestExpiresAndReplacementDeletesIt(t *testing.T) {
	k, _, _, _, _, ms, ctx := setupMsgServer(t)
	ctx = ctx.WithBlockHeight(100)
	reporter := sample.AccAddressBytes()
	other := sample.AccAddressBytes()
	agent := sample.AccAddressBytes()
	rep := types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "moniker")
	require.NoError(t, k.Reporters.Set(ctx, reporter.Bytes(), rep))
	require.NoError(t, k.Reporters.Set(ctx, other.Bytes(), rep))

	_, err := ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: reporter.String(),
		TwoBitAddress:   agent.String(),
	})
	require.NoError(t, err)

	// The request block and the following 2000 blocks can still accept.
	deadline := ctx.WithBlockHeight(100 + int64(types.TwoBitPendingRequestExpiryBlocks))
	_, err = ms.AcceptTwoBitRegistration(deadline, &types.MsgAcceptTwoBitRegistration{
		TwoBitAddress:   agent.String(),
		ReporterAddress: reporter.String(),
	})
	require.NoError(t, err)
	got, err := k.TwoBitReporter(deadline, agent)
	require.NoError(t, err)
	require.True(t, bytes.Equal(reporter.Bytes(), got.Bytes()))

	// One block past the deadline cannot be accepted, and it does not create a mapping.
	late := sample.AccAddressBytes()
	_, err = ms.RegisterTwoBitReporter(ctx, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: other.String(),
		TwoBitAddress:   late.String(),
	})
	require.NoError(t, err)
	expired := ctx.WithBlockHeight(100 + int64(types.TwoBitPendingRequestExpiryBlocks) + 1)
	_, err = ms.AcceptTwoBitRegistration(expired, &types.MsgAcceptTwoBitRegistration{
		TwoBitAddress:   late.String(),
		ReporterAddress: other.String(),
	})
	require.ErrorIs(t, err, types.ErrTwoBitRegistrationExpired)
	_, err = k.TwoBitReporter(expired, late)
	require.ErrorIs(t, err, collections.ErrNotFound)
	hasLate, err := k.TwoBitReporterToAgent.Has(expired, other.Bytes())
	require.NoError(t, err)
	require.False(t, hasLate)

	// Replacing the invite deletes it. The previous address cannot accept inside the old window.
	first := sample.AccAddressBytes()
	second := sample.AccAddressBytes()
	replacedAt := ctx.WithBlockHeight(150)
	_, err = ms.RegisterTwoBitReporter(replacedAt, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: other.String(),
		TwoBitAddress:   first.String(),
	})
	require.NoError(t, err)
	_, err = ms.RegisterTwoBitReporter(replacedAt, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: other.String(),
		TwoBitAddress:   second.String(),
	})
	require.NoError(t, err)
	pending, err := k.TwoBitPendingRequests.Get(replacedAt, other.Bytes())
	require.NoError(t, err)
	require.True(t, bytes.Equal(second.Bytes(), pending.TwoBitReporter))
	require.Equal(t, uint64(150)+types.TwoBitPendingRequestExpiryBlocks, pending.ExpiryHeight)
	_, err = ms.AcceptTwoBitRegistration(replacedAt, &types.MsgAcceptTwoBitRegistration{
		TwoBitAddress:   first.String(),
		ReporterAddress: other.String(),
	})
	require.ErrorIs(t, err, types.ErrTwoBitRegistrationNotFound)
	_, err = k.TwoBitReporter(replacedAt, first)
	require.ErrorIs(t, err, collections.ErrNotFound)

	// Canceling with the reporter's own address deletes the invite.
	_, err = ms.RegisterTwoBitReporter(replacedAt, &types.MsgRegisterTwoBitReporter{
		ReporterAddress: other.String(),
		TwoBitAddress:   other.String(),
	})
	require.NoError(t, err)
	_, err = k.TwoBitPendingRequests.Get(replacedAt, other.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)
	_, err = ms.AcceptTwoBitRegistration(replacedAt, &types.MsgAcceptTwoBitRegistration{
		TwoBitAddress:   second.String(),
		ReporterAddress: other.String(),
	})
	require.ErrorIs(t, err, types.ErrTwoBitRegistrationNotFound)
	_, err = k.TwoBitReporter(replacedAt, second)
	require.ErrorIs(t, err, collections.ErrNotFound)
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
	pending := sample.AccAddressBytes()
	require.NoError(t, k.Reporters.Set(ctx, reporter.Bytes(), types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "moniker")))
	require.NoError(t, k.SetTwoBitMapping(ctx, reporter.Bytes(), agent.Bytes()))
	require.NoError(t, k.SetTwoBitReporter(ctx, reporter, pending))

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
	_, err = k.TwoBitPendingRequests.Get(ctx, reporter.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)
}
