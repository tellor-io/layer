package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestWithdrawTipToBalance(t *testing.T) {
	k, sk, bk, _, _, msg, ctx := setupMsgServer(t)
	selector := sample.AccAddressBytes()
	require.NoError(t, k.Selectors.Set(ctx, selector, types.NewSelection(selector, 1)))

	_, err := msg.WithdrawTipToBalance(ctx, &types.MsgWithdrawTipToBalance{SelectorAddress: selector.String()})
	require.ErrorIs(t, err, collections.ErrNotFound)

	amount := math.NewInt(1e6)
	require.NoError(t, k.SelectorTips.Set(ctx, selector, math.LegacyNewDecFromInt(amount)))

	unbondingTime := 21 * 24 * time.Hour
	blockTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(blockTime)
	sk.On("UnbondingTime", ctx).Return(unbondingTime, nil)
	bk.On("SendCoinsFromModuleToModule", ctx, types.TipsEscrowPool, types.TipsUnlockPool, sdk.NewCoins(sdk.NewCoin("loya", amount))).Return(nil)

	res, err := msg.WithdrawTipToBalance(ctx, &types.MsgWithdrawTipToBalance{SelectorAddress: selector.String()})
	require.NoError(t, err)
	require.Equal(t, uint64(0), res.UnlockId)

	_, err = k.SelectorTips.Get(ctx, selector)
	require.ErrorIs(t, err, collections.ErrNotFound)

	entry, err := k.TipUnlocks.Get(ctx, collections.Join(selector.Bytes(), res.UnlockId))
	require.NoError(t, err)
	require.True(t, entry.Amount.Equal(amount))
	require.Equal(t, blockTime.Add(unbondingTime), entry.CompletionTime)

	queueSel, err := k.TipUnlockQueue.Get(ctx, collections.Join(entry.CompletionTime.Unix(), res.UnlockId))
	require.NoError(t, err)
	require.Equal(t, selector.Bytes(), queueSel)

	_, err = msg.WithdrawTipToBalance(ctx, &types.MsgWithdrawTipToBalance{SelectorAddress: selector.String()})
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func TestWithdrawTipToBalance_ZeroTips(t *testing.T) {
	k, _, _, _, _, msg, ctx := setupMsgServer(t)
	selector := sample.AccAddressBytes()
	require.NoError(t, k.SelectorTips.Set(ctx, selector, math.LegacyMustNewDecFromStr("0.5")))

	_, err := msg.WithdrawTipToBalance(ctx, &types.MsgWithdrawTipToBalance{SelectorAddress: selector.String()})
	require.ErrorContains(t, err, "no tips to withdraw")
}

func TestCancelTipUnlock(t *testing.T) {
	k, sk, bk, _, _, msg, ctx := setupMsgServer(t)
	selector := sample.AccAddressBytes()
	amount := math.NewInt(2e6)
	require.NoError(t, k.SelectorTips.Set(ctx, selector, math.LegacyNewDecFromInt(amount)))

	unbondingTime := 7 * 24 * time.Hour
	blockTime := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(blockTime)
	sk.On("UnbondingTime", ctx).Return(unbondingTime, nil)
	bk.On("SendCoinsFromModuleToModule", ctx, types.TipsEscrowPool, types.TipsUnlockPool, sdk.NewCoins(sdk.NewCoin("loya", amount))).Return(nil)

	res, err := msg.WithdrawTipToBalance(ctx, &types.MsgWithdrawTipToBalance{SelectorAddress: selector.String()})
	require.NoError(t, err)

	bk.On("SendCoinsFromModuleToModule", ctx, types.TipsUnlockPool, types.TipsEscrowPool, sdk.NewCoins(sdk.NewCoin("loya", amount))).Return(nil)

	_, err = msg.CancelTipUnlock(ctx, &types.MsgCancelTipUnlock{
		SelectorAddress: selector.String(),
		UnlockId:        res.UnlockId,
	})
	require.NoError(t, err)

	_, err = k.TipUnlocks.Get(ctx, collections.Join(selector.Bytes(), res.UnlockId))
	require.ErrorIs(t, err, collections.ErrNotFound)

	tips, err := k.SelectorTips.Get(ctx, selector)
	require.NoError(t, err)
	require.True(t, tips.Equal(math.LegacyNewDecFromInt(amount)))

	_, err = msg.CancelTipUnlock(ctx, &types.MsgCancelTipUnlock{
		SelectorAddress: selector.String(),
		UnlockId:        res.UnlockId,
	})
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func TestProcessMatureTipUnlocks(t *testing.T) {
	k, sk, bk, _, _, msg, ctx := setupMsgServer(t)
	selector := sample.AccAddressBytes()
	amount := math.NewInt(3e6)
	require.NoError(t, k.SelectorTips.Set(ctx, selector, math.LegacyNewDecFromInt(amount)))

	unbondingTime := 21 * 24 * time.Hour
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(start)
	sk.On("UnbondingTime", ctx).Return(unbondingTime, nil)
	bk.On("SendCoinsFromModuleToModule", ctx, types.TipsEscrowPool, types.TipsUnlockPool, sdk.NewCoins(sdk.NewCoin("loya", amount))).Return(nil)

	res, err := msg.WithdrawTipToBalance(ctx, &types.MsgWithdrawTipToBalance{SelectorAddress: selector.String()})
	require.NoError(t, err)

	// before maturity: no payout
	require.NoError(t, k.ProcessMatureTipUnlocks(ctx))
	_, err = k.TipUnlocks.Get(ctx, collections.Join(selector.Bytes(), res.UnlockId))
	require.NoError(t, err)

	matureCtx := ctx.WithBlockTime(start.Add(unbondingTime))
	bk.On("SendCoinsFromModuleToAccount", matureCtx, types.TipsUnlockPool, selector, sdk.NewCoins(sdk.NewCoin("loya", amount))).Return(nil)

	require.NoError(t, k.ProcessMatureTipUnlocks(matureCtx))
	_, err = k.TipUnlocks.Get(matureCtx, collections.Join(selector.Bytes(), res.UnlockId))
	require.ErrorIs(t, err, collections.ErrNotFound)
	_, err = k.TipUnlockQueue.Get(matureCtx, collections.Join(start.Add(unbondingTime).Unix(), res.UnlockId))
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func TestTipUnlocksQuery(t *testing.T) {
	k, sk, bk, _, _, msg, ctx := setupMsgServer(t)
	q := keeper.NewQuerier(k)
	selector := sample.AccAddressBytes()

	res, err := q.TipUnlocks(ctx, &types.QueryTipUnlocksRequest{SelectorAddress: selector.String()})
	require.NoError(t, err)
	require.Empty(t, res.Unlocks)

	amount1 := math.NewInt(1e6)
	amount2 := math.NewInt(2e6)
	require.NoError(t, k.SelectorTips.Set(ctx, selector, math.LegacyNewDecFromInt(amount1)))

	unbondingTime := 21 * 24 * time.Hour
	blockTime := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(blockTime)
	sk.On("UnbondingTime", ctx).Return(unbondingTime, nil).Twice()
	bk.On("SendCoinsFromModuleToModule", ctx, types.TipsEscrowPool, types.TipsUnlockPool, sdk.NewCoins(sdk.NewCoin("loya", amount1))).Return(nil)
	bk.On("SendCoinsFromModuleToModule", ctx, types.TipsEscrowPool, types.TipsUnlockPool, sdk.NewCoins(sdk.NewCoin("loya", amount2))).Return(nil)

	r1, err := msg.WithdrawTipToBalance(ctx, &types.MsgWithdrawTipToBalance{SelectorAddress: selector.String()})
	require.NoError(t, err)

	require.NoError(t, k.SelectorTips.Set(ctx, selector, math.LegacyNewDecFromInt(amount2)))
	r2, err := msg.WithdrawTipToBalance(ctx, &types.MsgWithdrawTipToBalance{SelectorAddress: selector.String()})
	require.NoError(t, err)
	require.NotEqual(t, r1.UnlockId, r2.UnlockId)

	res, err = q.TipUnlocks(ctx, &types.QueryTipUnlocksRequest{SelectorAddress: selector.String()})
	require.NoError(t, err)
	require.Len(t, res.Unlocks, 2)
	require.Equal(t, r1.UnlockId, res.Unlocks[0].UnlockId)
	require.True(t, res.Unlocks[0].Amount.Equal(amount1))
	require.Equal(t, r2.UnlockId, res.Unlocks[1].UnlockId)
	require.True(t, res.Unlocks[1].Amount.Equal(amount2))
}
