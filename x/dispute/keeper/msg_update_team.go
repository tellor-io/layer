package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k msgServer) UpdateTeam(ctx context.Context, msg *types.MsgUpdateTeam) (*types.MsgUpdateTeamResponse, error) {
	param, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	if param.TeamAddress != msg.CurrentTeamAddress {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid team address; expected %s, got %s", param.TeamAddress, msg.CurrentTeamAddress)
	}
	param.TeamAddress = msg.NewTeamAddress
	if err := k.Params.Set(ctx, param); err != nil {
		return nil, err
	}
	return &types.MsgUpdateTeamResponse{}, nil
}
