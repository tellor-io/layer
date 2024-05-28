package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/bridge/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) ClaimDeposit(goCtx context.Context, msg *types.MsgClaimDepositRequest) (*types.MsgClaimDepositResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(goCtx)

	if err := k.Keeper.ClaimDeposit(sdkCtx, msg.DepositId, msg.Index); err != nil {
		return nil, err
	}
	return &types.MsgClaimDepositResponse{}, nil
}
