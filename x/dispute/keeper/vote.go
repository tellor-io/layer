package keeper

import (
	"bytes"
	"context"
	"errors"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k Keeper) initVoterClasses() *types.VoterClasses {
	return &types.VoterClasses{
		Reporters:    math.ZeroInt(),
		TokenHolders: math.ZeroInt(),
		Users:        math.ZeroInt(),
		Team:         math.ZeroInt(),
	}
}

// Set vote start info for a dispute
func (k Keeper) SetStartVote(ctx sdk.Context, id uint64) error {
	vote := types.Vote{
		Id:        id,
		VoteStart: ctx.BlockTime(),
		VoteEnd:   ctx.BlockTime().Add(TWO_DAYS),
	}
	return k.Votes.Set(ctx, id, vote)
}

func (k Keeper) TeamVote(ctx context.Context, id uint64) (math.Int, error) {
	teamTally := math.ZeroInt()
	voted, err := k.TeamVoter.Has(ctx, id)
	if err != nil {
		return math.Int{}, err
	}
	if voted {
		teamTally = math.OneInt()
	}

	return teamTally, nil
}
func (k Keeper) GetTeamAddress(ctx context.Context) (sdk.AccAddress, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	return sdk.AccAddressFromBech32(params.TeamAddress)
}
func (k Keeper) SetTeamVote(ctx context.Context, id uint64, voter sdk.AccAddress) (math.Int, error) {
	teamAddr, err := k.GetTeamAddress(ctx)
	if err != nil {
		return math.Int{}, err
	}

	if bytes.Equal(voter, teamAddr) {
		return math.NewInt(25000000), k.TeamVoter.Set(ctx, id, true)
	}
	return math.ZeroInt(), nil
}

func (k Keeper) GetUserTotalTips(ctx context.Context, voter sdk.AccAddress, blockNumber int64) (math.Int, error) {
	tips, err := k.oracleKeeper.GetTipsAtBlockForTipper(ctx, blockNumber, voter)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return math.Int{}, err
		}
		return math.ZeroInt(), nil
	}
	return tips, nil
}

func (k Keeper) SetVoterTips(ctx context.Context, id uint64, voter sdk.AccAddress, blockNumber int64) (math.Int, error) {
	tips, err := k.GetUserTotalTips(ctx, voter, blockNumber)
	if err != nil {
		return math.Int{}, err
	}
	if !tips.IsZero() {
		return tips, k.UsersGroup.Set(ctx, collections.Join(id, voter), tips)
	}
	return math.ZeroInt(), nil
}

func (k Keeper) SetVoterReporterStake(ctx context.Context, id uint64, voter sdk.AccAddress, blockNumber int64) (math.Int, error) {
	delegation, err := k.reporterKeeper.Delegation(ctx, voter)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return math.ZeroInt(), nil
		}
		return math.Int{}, err
	}
	reporter := sdk.MustAccAddressFromBech32(delegation.Reporter)
	// check if reporter has voted if not store voter tokens either full if reporter or delegation amount
	// this amount the amount to reduce from reporter so total amount of delegators that voted
	reporterTokensVoted, err := k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(reporter.Bytes(), id))
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return math.Int{}, err
		}
		if bytes.Equal(reporter, voter) {
			reporterTokens, err := k.reporterKeeper.GetReporterTokensAtBlock(ctx, reporter, blockNumber)
			if err != nil {
				return math.Int{}, err
			}
			return reporterTokens, k.ReportersGroup.Set(ctx, collections.Join(id, voter), reporterTokens)
		}
		amt, err := k.reporterKeeper.GetDelegatorTokensAtBlock(ctx, reporter, blockNumber)
		if err != nil {
			return math.Int{}, err
		}
		exists, err := k.ReportersGroup.Has(ctx, collections.Join(id, reporter))
		if err != nil {
			return math.Int{}, err
		}
		if exists {
			// get reporter tokens and reduce the amount
			reporterTokens, err := k.ReportersGroup.Get(ctx, collections.Join(id, reporter))
			if err != nil {
				return math.Int{}, err
			}
			reporterTokens = reporterTokens.Sub(amt)
			voterV, err := k.Voter.Get(ctx, collections.Join(id, reporter))
			if err != nil {
				return math.Int{}, err
			}
			voterV.VoterPower = voterV.VoterPower.Sub(amt)
			if err := k.Voter.Set(ctx, collections.Join(id, reporter), voterV); err != nil {
				return math.Int{}, err
			}
			if err := k.ReportersGroup.Set(ctx, collections.Join(id, reporter), reporterTokens); err != nil {
				return math.Int{}, err
			}
			return amt, k.ReportersGroup.Set(ctx, collections.Join(id, voter), amt)
		}
		if err := k.ReportersWithDelegatorsVotedBefore.Set(ctx, collections.Join(reporter.Bytes(), id), amt); err != nil {
			return math.Int{}, err
		}
		return amt, k.ReportersGroup.Set(ctx, collections.Join(id, voter), amt)
	}
	// if reporter delegators have voted before reporter, then if voter is reporter get reporter tokens at block and reduce the amount that has voted already
	if bytes.Equal(reporter, voter) {
		reporterTokens, err := k.reporterKeeper.GetReporterTokensAtBlock(ctx, reporter, blockNumber)
		if err != nil {
			return math.Int{}, err
		}
		return reporterTokens.Sub(reporterTokensVoted), k.ReportersGroup.Set(ctx, collections.Join(id, voter), reporterTokens.Sub(reporterTokensVoted))
	} else {
		amt, err := k.reporterKeeper.GetDelegatorTokensAtBlock(ctx, reporter, blockNumber)
		if err != nil {
			return math.Int{}, err
		}
		if err := k.ReportersWithDelegatorsVotedBefore.Set(ctx, collections.Join(reporter.Bytes(), id), reporterTokensVoted.Add(amt)); err != nil {
			return math.Int{}, err
		}
	}

	return math.ZeroInt(), nil
}
