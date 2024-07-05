package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/dispute/types"
)

func (k msgServer) TallyVote(ctx context.Context, msg *types.MsgTallyVote) (*types.MsgTallyVoteResponse, error) {
	err := k.Keeper.TallyVote(ctx, msg.DisputeId)
	if err != nil {
		return nil, err
	}

	return &types.MsgTallyVoteResponse{}, nil
}
