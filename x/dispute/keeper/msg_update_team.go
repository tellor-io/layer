package keeper

import (
	"bytes"
	"context"

	"github.com/tellor-io/layer/x/dispute/types"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) UpdateTeam(ctx context.Context, msg *types.MsgUpdateTeam) (*types.MsgUpdateTeamResponse, error) {
	param, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	currentAcc, err := sdk.AccAddressFromBech32(msg.CurrentTeamAddress)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(param.TeamAddress, currentAcc.Bytes()) {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid team address; expected %s, got %s", param.TeamAddress, msg.CurrentTeamAddress)
	}
	newAcc, err := sdk.AccAddressFromBech32(msg.NewTeamAddress)
	if err != nil {
		return nil, err
	}
	param.TeamAddress = newAcc
	if err := k.Params.Set(ctx, param); err != nil {
		return nil, err
	}
	return &types.MsgUpdateTeamResponse{}, nil
}
