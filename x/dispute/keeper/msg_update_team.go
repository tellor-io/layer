package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"

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
			currentAccBytes := currentAcc.Bytes()
			newAccBytes := newAcc.Bytes()
			teamVoteExists, err := k.Voter.Has(ctx, collections.Join(id, currentAccBytes))
			if err != nil {
				if !errors.Is(err, collections.ErrNotFound) {
					return nil, err
				}
			}
			fmt.Println("teamVoteExists: ", teamVoteExists)
			// if team has voted, remove previous team vote and set again with new address
			if teamVoteExists {
				vote, err := k.Voter.Get(ctx, collections.Join(id, currentAccBytes))
				if err != nil {
					return nil, err
				}
				fmt.Println("vote: ", vote)
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

	return &types.MsgUpdateTeamResponse{}, nil
}
