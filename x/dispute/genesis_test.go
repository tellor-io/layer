package dispute_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/nullify"
	"github.com/tellor-io/layer/x/dispute"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/math"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:                          types.DefaultParams(),
		Disputes:                        []*types.DisputeStateEntry{},
		Votes:                           []*types.VotesStateEntry{},
		Voter:                           []*types.VoterStateEntry{},
		ReportersWithDelegatorsWhoVoted: []*types.ReportersWithDelegatorsWhoVotedStateEntry{},
		BlockInfo:                       []*types.BlockInfoStateEntry{},
		DisputeFeePayer:                 []*types.DisputeFeePayerStateEntry{},
		Dust:                            math.ZeroInt(),
		VoteCountsByGroup:               []*types.VoteCountsByGroupStateEntry{},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, _, _, _, _, ctx := keepertest.DisputeKeeper(t)
	require.NotPanics(t, func() { dispute.InitGenesis(ctx, k, genesisState) })
	got := dispute.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	genesisState.Params = types.Params{TeamAddress: []byte("test team address")}
	genesisState.Disputes = append(genesisState.Disputes, &types.DisputeStateEntry{DisputeId: 1, Dispute: &types.Dispute{DisputeId: 1}})
	genesisState.Votes = append(genesisState.Votes, &types.VotesStateEntry{DisputeId: 1, Vote: &types.Vote{Id: 1, VoteStart: time.Now()}})
	genesisState.ReportersWithDelegatorsWhoVoted = append(genesisState.ReportersWithDelegatorsWhoVoted, &types.ReportersWithDelegatorsWhoVotedStateEntry{ReporterAddress: []byte("reporter"), VotedAmount: math.NewInt(1_000_000)})
	genesisState.BlockInfo = append(genesisState.BlockInfo, &types.BlockInfoStateEntry{HashId: []byte("hash id"), BlockInfo: &types.BlockInfo{TotalReporterPower: math.NewInt(5_000_000), TotalUserTips: math.NewInt(10_000)}})
	genesisState.DisputeFeePayer = append(genesisState.DisputeFeePayer, &types.DisputeFeePayerStateEntry{DisputeId: 1, Payer: []byte("payer"), PayerInfo: &types.PayerInfo{Amount: math.NewInt(10_000), FromBond: false}})
	genesisState.Dust = math.NewInt(1000)
	genesisState.VoteCountsByGroup = append(genesisState.VoteCountsByGroup, &types.VoteCountsByGroupStateEntry{DisputeId: 1, Users: &types.VoteCounts{Support: 10, Against: 10, Invalid: 10}, Reporters: &types.VoteCounts{Support: 10, Against: 10, Invalid: 10}, Team: &types.VoteCounts{Support: 10, Against: 10, Invalid: 10}})

	k, _, _, _, _, ctx = keepertest.DisputeKeeper(t)
	require.NotPanics(t, func() { dispute.InitGenesis(ctx, k, genesisState) })
	got = dispute.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.Equal(t, genesisState.Params, got.Params)
	require.Equal(t, genesisState.Disputes, got.Disputes)
	require.Equal(t, genesisState.Votes, got.Votes)
	require.Equal(t, genesisState.ReportersWithDelegatorsWhoVoted, got.ReportersWithDelegatorsWhoVoted)
	require.Equal(t, genesisState.BlockInfo, got.BlockInfo)
	require.Equal(t, genesisState.DisputeFeePayer, got.DisputeFeePayer)
	require.Equal(t, genesisState.Dust, got.Dust)
	require.Equal(t, genesisState.VoteCountsByGroup, got.VoteCountsByGroup)
	// Clean up the exported state file
	err := os.Remove("dispute_module_state.json")
	require.NoError(t, err)

	// this line is used by starport scaffolding # genesis/test/assert
}
