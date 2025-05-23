package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/bridge/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Claim deposits made by the Ethereum token bridge contract into Layer.
func (k msgServer) ClaimDeposits(goCtx context.Context, msg *types.MsgClaimDepositsRequest) (*types.MsgClaimDepositsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	if len(msg.DepositIds) != len(msg.Timestamps) {
		return nil, types.ErrInvalidDepositIdsAndIndicesLength
	}
	for i, depositId := range msg.DepositIds {
		timestamp := msg.Timestamps[i]
		if err := k.Keeper.ClaimDeposit(sdkCtx, depositId, timestamp); err != nil {
			return nil, err
		}
	}

	return &types.MsgClaimDepositsResponse{}, nil
}
