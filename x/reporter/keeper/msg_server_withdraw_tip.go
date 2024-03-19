package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tellor-io/layer/x/reporter/types"

	layertypes "github.com/tellor-io/layer/types"
)

func (k msgServer) WithdrawTip(goCtx context.Context, msg *types.MsgWithdrawTip) (*types.MsgWithdrawTipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	delAddr := sdk.MustAccAddressFromBech32(msg.DelegatorAddress)
	shares, err := k.Keeper.DelegatorTips.Get(ctx, delAddr)
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
	_, err = k.Keeper.stakingKeeper.Delegate(ctx, delAddr, shares, stakingtypes.Unbonding, val, false)
	if err != nil {
		return nil, err
	}

	err = k.Keeper.DelegatorTips.Remove(ctx, delAddr)
	if err != nil {
		return nil, err
	}

	// send coins
	err = k.Keeper.bankKeeper.SendCoinsFromModuleToModule(ctx, types.TipsEscrowPool, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, shares)))
	if err != nil {
		return nil, err
	}

	return &types.MsgWithdrawTipResponse{}, nil
}
