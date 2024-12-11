package keeper

import (
	"context"
	"errors"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// Msg: WithdrawTipLegacy, allows selectors to directly withdraw reporting rewards and stake them with a BONDED validator
func (k msgServer) WithdrawTipLegacy(goCtx context.Context, msg *types.MsgWithdrawTipLegacy) (*types.MsgWithdrawTipLegacyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	delAddr := sdk.MustAccAddressFromBech32(msg.SelectorAddress)
	shares, err := k.Keeper.SelectorTips.Get(ctx, delAddr)
	if err != nil {
		return nil, err
	}

	valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return nil, err
	}
	val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	if !val.IsBonded() {
		return nil, errors.New("chosen validator must be bonded")
	}
	amtToDelegate := shares.TruncateInt()
	if amtToDelegate.IsZero() {
		return nil, errors.New("no tips to withdraw")
	}
	_, err = k.Keeper.stakingKeeper.Delegate(ctx, delAddr, amtToDelegate, val.Status, val, false)
	if err != nil {
		return nil, err
	}

	// isolate decimals from shares
	remainder := shares.Sub(shares.TruncateDec())
	if remainder.IsZero() {
		err = k.Keeper.SelectorTips.Remove(ctx, delAddr)
		if err != nil {
			return nil, err
		}
	} else {
		err = k.Keeper.SelectorTips.Set(ctx, delAddr, remainder)
		if err != nil {
			return nil, err
		}
	}

	// send coins
	err = k.Keeper.bankKeeper.SendCoinsFromModuleToModule(ctx, types.TipsEscrowPool, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, math.NewInt(int64(amtToDelegate.Uint64())))))
	if err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"tip_withdrawn",
			sdk.NewAttribute("selector", msg.SelectorAddress),
			sdk.NewAttribute("validator", msg.ValidatorAddress),
			sdk.NewAttribute("amount", amtToDelegate.String()),
		),
	})
	return &types.MsgWithdrawTipLegacyResponse{}, nil
}
