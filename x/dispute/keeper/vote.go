package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

// Get vote by dispute id
func (k Keeper) GetVote(ctx sdk.Context, id uint64) (vote *types.Vote) {
	store := k.voteStore(ctx)
	voteBytes := store.Get(types.DisputeIdBytes(id))
	if err := k.cdc.Unmarshal(voteBytes, vote); err != nil {
		return nil
	}
	return vote
}

// Get a voter's vote by voter & dispute id
func (k Keeper) GetVoterVote(ctx sdk.Context, voter string, id uint64) (vote *types.Vote) {
	store := k.voteStore(ctx)
	voteBytes := store.Get(types.VoterKeyPrefix(voter, id))
	if err := k.cdc.Unmarshal(voteBytes, vote); err != nil {
		return nil
	}
	return vote

}

// Set tally numbers
func (k Keeper) GetTally(ctx sdk.Context, id uint64) *types.Tally {
	store := k.voteStore(ctx)
	tallyBytes := store.Get(types.TallyKeyPrefix(id))
	var tallies types.Tally
	if err := k.cdc.Unmarshal(tallyBytes, &tallies); err != nil {
		return nil
	}

	return &tallies
}

// Set tally numbers
func (k Keeper) SetTally(ctx sdk.Context, id uint64, voteFor types.VoteEnum, voter string) error {
	tallies := k.GetTally(ctx, id)
	switch voteFor {
	case types.VoteEnum_VOTE_SUPPORT:
		tallies.ForVotes.Validators.Add(k.GetValidatorPower(ctx, voter))
		tallies.ForVotes.TokenHolders.Add(k.GetAccountBalance(ctx, voter))
		tallies.ForVotes.Users.Add(k.GetUserTips(ctx, voter))
		tallies.ForVotes.Team.Add(k.IsTeamAddress(ctx, voter))
	case types.VoteEnum_VOTE_AGAINST:
		tallies.AgainstVotes.Validators.Add(k.GetValidatorPower(ctx, voter))
		tallies.AgainstVotes.TokenHolders.Add(k.GetAccountBalance(ctx, voter))
		tallies.AgainstVotes.Users.Add(k.GetUserTips(ctx, voter))
		tallies.AgainstVotes.Team.Add(k.IsTeamAddress(ctx, voter))
	case types.VoteEnum_VOTE_INVALID:
		tallies.Invalid.Validators.Add(k.GetValidatorPower(ctx, voter))
		tallies.Invalid.TokenHolders.Add(k.GetAccountBalance(ctx, voter))
		tallies.Invalid.Users.Add(k.GetUserTips(ctx, voter))
		tallies.Invalid.Team.Add(k.IsTeamAddress(ctx, voter))
	default:
		panic("invalid vote type")
	}

	store := k.voteStore(ctx)
	store.Set(types.TallyKeyPrefix(id), k.cdc.MustMarshal(tallies))
	return nil
}

func (k Keeper) GetValidatorPower(ctx sdk.Context, voter string) math.Int {
	addr, err := sdk.AccAddressFromBech32(voter)
	if err != nil {
		panic(err)
	}
	validator, found := k.stakingKeeper.GetValidator(ctx, sdk.ValAddress(addr))
	if !found {
		return sdk.ZeroInt()
	}
	power := validator.GetConsensusPower(validator.GetBondedTokens())
	return sdk.NewInt(power)
}

func (k Keeper) GetAccountBalance(ctx sdk.Context, voter string) math.Int {
	addr, err := sdk.AccAddressFromBech32(voter)
	if err != nil {
		panic(err)
	}
	return k.bankKeeper.GetBalance(ctx, addr, sdk.DefaultBondDenom).Amount
}

func (k Keeper) GetUserTips(ctx sdk.Context, voter string) math.Int {
	return sdk.ZeroInt()
}

func (k Keeper) IsTeamAddress(ctx sdk.Context, voter string) math.Int {
	return math.NewIntFromUint64(1)
}

// Set vote by voter
func (k Keeper) SetVoterVote(ctx sdk.Context, msg types.MsgVote) {
	store := k.voteStore(ctx)
	voterKey := types.VoterKeyPrefix(msg.Voter, msg.Id)
	store.Set(voterKey, k.cdc.MustMarshal(&msg))
}

// Append voters to vote struct
func (k Keeper) AppendVoters(ctx sdk.Context, id uint64, voter string) {
	store := k.voteStore(ctx)
	vote := k.GetVote(ctx, id)
	vote.Voters = append(vote.Voters, voter)
	store.Set(types.DisputeIdBytes(id), k.cdc.MustMarshal(vote))
}
