package keeper

import (
	"bytes"
	"context"

	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
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
	currentAccBytes := currentAcc.Bytes()
	newAccBytes := newAcc.Bytes()
	// if the team has voted on a dispute, transfer vote to the new address
	iter, err := k.Disputes.Indexes.OpenDisputes.MatchExact(ctx, true)
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key, err := iter.PrimaryKey()
		if err != nil {
			return nil, err
		}
		dispute, err := k.Disputes.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		// if dispute is open, check if team has voted
		if dispute.Open {
			id := dispute.DisputeId
			teamVoteExists, err := k.Voter.Has(ctx, collections.Join(id, currentAccBytes))
			if err != nil {
				return nil, err
			}
			// if team has voted, remove previous team vote and set again with new address
			if teamVoteExists {
				vote, err := k.Voter.Get(ctx, collections.Join(id, currentAccBytes))
				if err != nil {
					return nil, err
				}
				err = k.Voter.Remove(ctx, collections.Join(id, currentAccBytes))
				if err != nil {
					return nil, err
				}
				err = k.Voter.Set(ctx, collections.Join(id, newAccBytes), vote)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// emit event
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"team_address_updated",
			sdk.NewAttribute("new_team_address", msg.NewTeamAddress),
			sdk.NewAttribute("old_team_address", msg.CurrentTeamAddress),
		),
	})

	return &types.MsgUpdateTeamResponse{}, nil
}
