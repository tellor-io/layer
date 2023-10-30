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
func (k Keeper) SetTally(ctx sdk.Context, id uint64, voteFor bool, voter string) error {
	tallies := k.GetTally(ctx, id)
	if voteFor {
		tallies.ForVotes.Validators.Add(k.GetValidatorTokenBalance(ctx, voter))
		tallies.ForVotes.TokenHolders.Add(k.GetAccountBalance(ctx, voter))
		tallies.ForVotes.Users.Add(k.GetUserTips(ctx, voter))
		tallies.ForVotes.Team.Add(k.IsTeamAddress(ctx, voter))
	} else {
		tallies.AgainstVotes.Validators.Add(k.GetValidatorTokenBalance(ctx, voter))
		tallies.AgainstVotes.TokenHolders.Add(k.GetAccountBalance(ctx, voter))
		tallies.AgainstVotes.Users.Add(k.GetUserTips(ctx, voter))
		tallies.AgainstVotes.Team.Add(k.IsTeamAddress(ctx, voter))
	}
	store := k.voteStore(ctx)
	store.Set(types.TallyKeyPrefix(id), k.cdc.MustMarshal(tallies))
	return nil
}

func (k Keeper) GetValidatorTokenBalance(ctx sdk.Context, voter string) math.Int {
	return sdk.ZeroInt()
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
func (k Keeper) SetVoterVote(ctx sdk.Context, voter string, id uint64, vote bool) {
	store := k.voteStore(ctx)
	voterKey := types.VoterKeyPrefix(voter, id)
	var voteBytes []byte
	if vote {
		voteBytes = []byte{0x01}
	} else {
		voteBytes = []byte{0x00}
	}
	// fix this
	store.Set(voterKey, voteBytes)
}
