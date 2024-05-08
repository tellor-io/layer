package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
)

func (k msgServer) ClaimDeposit(goCtx context.Context, msg *types.MsgClaimDepositRequest) (*types.MsgClaimDepositResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(goCtx)

	if err := k.claimDeposit(sdkCtx, msg.DepositId, msg.Index); err != nil {
		return nil, err
	}
	return &types.MsgClaimDepositResponse{}, nil
}
