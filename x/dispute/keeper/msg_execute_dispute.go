package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/dispute/types"
)

func (k msgServer) ExecuteDispute(ctx context.Context, msg *types.MsgExecuteDispute) (*types.MsgExecuteDisputeResponse, error) {
	err := k.Keeper.ExecuteVote(ctx, msg.DisputeId)
	if err != nil {
		return nil, err
	}

	return &types.MsgExecuteDisputeResponse{}, nil
}
