package dispute

import (
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	err := k.Params.Set(ctx, genState.Params)
	if err != nil {
		panic(err)
	}
	if genState.Dust.IsNil() {
		err = k.Dust.Set(ctx, math.ZeroInt())
		if err != nil {
			panic(err)
		}
	} else {
		err = k.Dust.Set(ctx, genState.Dust)
		if err != nil {
			panic(err)
		}
	}

	for _, data := range genState.Disputes {
		if err := k.Disputes.Set(ctx, data.DisputeId, *data.Dispute); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.Votes {
		if err := k.Votes.Set(ctx, data.DisputeId, *data.Vote); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.Voter {
		if err := k.Voter.Set(ctx, collections.Join(data.DisputeId, data.VoterAddress), *data.Voter); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.ReportersWithDelegatorsWhoVoted {
		if err := k.ReportersWithDelegatorsVotedBefore.Set(ctx, collections.Join(data.ReporterAddress, data.DisputeId), data.VotedAmount); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.BlockInfo {
		if err := k.BlockInfo.Set(ctx, data.HashId, *data.BlockInfo); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.DisputeFeePayer {
		if err := k.DisputeFeePayer.Set(ctx, collections.Join(data.DisputeId, data.Payer), *data.PayerInfo); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.VoteCountsByGroup {
		if err := k.VoteCountsByGroup.Set(ctx, data.DisputeId, types.StakeholderVoteCounts{Users: *data.Users, Reporters: *data.Reporters, Team: *data.Team}); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	params, _ := k.Params.Get(ctx)
	genesis.Params = params

	iterDisputes, err := k.Disputes.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	disputes := make([]*types.DisputeStateEntry, 0)
	for ; iterDisputes.Valid(); iterDisputes.Next() {
		dispute_id, err := iterDisputes.Key()
		if err != nil {
			panic(err)
		}

		dispute, err := iterDisputes.Value()
		if err != nil {
			panic(err)
		}
		disputes = append(disputes, &types.DisputeStateEntry{DisputeId: dispute_id, Dispute: &dispute})
	}
	genesis.Disputes = disputes
	err = iterDisputes.Close()
	if err != nil {
		panic(err)
	}

	iterVotes, err := k.Votes.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	votes := make([]*types.VotesStateEntry, 0)
	for ; iterVotes.Valid(); iterVotes.Next() {
		dispute_id, err := iterVotes.Key()
		if err != nil {
			panic(err)
		}

		vote, err := iterVotes.Value()
		if err != nil {
			panic(err)
		}
		votes = append(votes, &types.VotesStateEntry{DisputeId: dispute_id, Vote: &vote})
	}
	genesis.Votes = votes
	err = iterVotes.Close()
	if err != nil {
		panic(err)
	}

	iterVoter, err := k.Voter.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	voters := make([]*types.VoterStateEntry, 0)
	for ; iterVoter.Valid(); iterVoter.Next() {
		key, err := iterVoter.Key()
		if err != nil {
			panic(err)
		}
		dispute_id := key.K1()
		voterAddr := key.K2()

		voter, err := iterVoter.Value()
		if err != nil {
			panic(err)
		}
		voters = append(voters, &types.VoterStateEntry{DisputeId: dispute_id, VoterAddress: voterAddr, Voter: &voter})
	}
	genesis.Voter = voters
	err = iterVoter.Close()
	if err != nil {
		panic(err)
	}

	iterReportersDelVoted, err := k.ReportersWithDelegatorsVotedBefore.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	reportersWithDelsWhoVoted := make([]*types.ReportersWithDelegatorsWhoVotedStateEntry, 0)
	for ; iterReportersDelVoted.Valid(); iterReportersDelVoted.Next() {
		key, err := iterReportersDelVoted.Key()
		if err != nil {
			panic(err)
		}
		reporterAddr := key.K1()
		dispute_id := key.K2()

		votedAmt, err := iterReportersDelVoted.Value()
		if err != nil {
			panic(err)
		}
		reportersWithDelsWhoVoted = append(reportersWithDelsWhoVoted, &types.ReportersWithDelegatorsWhoVotedStateEntry{ReporterAddress: reporterAddr, DisputeId: dispute_id, VotedAmount: votedAmt})
	}
	genesis.ReportersWithDelegatorsWhoVoted = reportersWithDelsWhoVoted
	err = iterReportersDelVoted.Close()
	if err != nil {
		panic(err)
	}

	iterBlockInfo, err := k.BlockInfo.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	blockInfo := make([]*types.BlockInfoStateEntry, 0)
	for ; iterBlockInfo.Valid(); iterBlockInfo.Next() {
		hash_id, err := iterBlockInfo.Key()
		if err != nil {
			panic(err)
		}

		info, err := iterBlockInfo.Value()
		if err != nil {
			panic(err)
		}
		blockInfo = append(blockInfo, &types.BlockInfoStateEntry{HashId: hash_id, BlockInfo: &info})
	}
	genesis.BlockInfo = blockInfo
	err = iterBlockInfo.Close()
	if err != nil {
		panic(err)
	}

	iterDisputeFeePayer, err := k.DisputeFeePayer.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	disputeFeePayer := make([]*types.DisputeFeePayerStateEntry, 0)
	for ; iterDisputeFeePayer.Valid(); iterDisputeFeePayer.Next() {
		keys, err := iterDisputeFeePayer.Key()
		if err != nil {
			panic(err)
		}
		dispute_id := keys.K1()
		payer := keys.K2()

		payerInfo, err := iterDisputeFeePayer.Value()
		if err != nil {
			panic(err)
		}
		disputeFeePayer = append(disputeFeePayer, &types.DisputeFeePayerStateEntry{DisputeId: dispute_id, Payer: payer, PayerInfo: &payerInfo})
	}
	genesis.DisputeFeePayer = disputeFeePayer
	err = iterDisputeFeePayer.Close()
	if err != nil {
		panic(err)
	}

	Dust, err := k.Dust.Get(ctx)
	if err != nil {
		panic(err)
	}
	genesis.Dust = Dust

	iterVoteCountsByGroup, err := k.VoteCountsByGroup.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	voteCountsByGroup := make([]*types.VoteCountsByGroupStateEntry, 0)
	for ; iterVoteCountsByGroup.Valid(); iterVoteCountsByGroup.Next() {
		dispute_id, err := iterVoteCountsByGroup.Key()
		if err != nil {
			panic(err)
		}

		voteCount, err := iterVoteCountsByGroup.Value()
		if err != nil {
			panic(err)
		}
		voteCountsByGroup = append(voteCountsByGroup, &types.VoteCountsByGroupStateEntry{DisputeId: dispute_id, Users: &voteCount.Users, Reporters: &voteCount.Reporters, Team: &voteCount.Team})
	}
	genesis.VoteCountsByGroup = voteCountsByGroup
	err = iterVoteCountsByGroup.Close()
	if err != nil {
		panic(err)
	}
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
