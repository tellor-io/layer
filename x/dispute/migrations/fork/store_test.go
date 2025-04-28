package fork_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdkStore "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/mocks"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
)

type StreamModuleStateData struct {
	Disputes                        []*disputetypes.DisputeStateEntry                         `json:"disputes,omitempty"`
	Votes                           []*disputetypes.VotesStateEntry                           `json:"votes,omitempty"`
	Voters                          []*disputetypes.VoterStateEntry                           `json:"voters,omitempty"`
	ReportersWithDelegatorsWhoVoted []*disputetypes.ReportersWithDelegatorsWhoVotedStateEntry `json:"reporters_with_delegators_who_voted,omitempty"`
	BlockInfo                       []*disputetypes.BlockInfoStateEntry                       `json:"block_info,omitempty"`
	DisputeFeePayer                 []*disputetypes.DisputeFeePayerStateEntry                 `json:"dispute_fee_payer,omitempty"`
	Dust                            *math.Int                                                 `json:"dust,omitempty"`
	VoteCountsByGroup               []*disputetypes.VoteCountsByGroupStateEntry               `json:"vote_counts_by_group,omitempty"`
}

func setupTest(t *testing.T) (context.Context, store.KVStoreService, codec.Codec, keeper.Keeper) {
	t.Helper()
	// Create in-memory store
	storeKey := storetypes.NewKVStoreKey(disputetypes.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(disputetypes.MemStoreKey)
	db := cosmosdb.NewMemDB()

	stateStore := sdkStore.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	storeService := runtime.NewKVStoreService(storeKey)
	cdc := codec.NewProtoCodec(types.NewInterfaceRegistry())

	accountKeeper := new(mocks.AccountKeeper)
	bankKeeper := new(mocks.BankKeeper)
	oracleKeeper := new(mocks.OracleKeeper)
	reporterKeeper := new(mocks.ReporterKeeper)

	k := keeper.NewKeeper(cdc, storeService, accountKeeper, bankKeeper, oracleKeeper, reporterKeeper)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	return ctx, storeService, cdc, k
}

func TestMigrateStore(t *testing.T) {
	ctx, _, _, k := setupTest(t)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	m := keeper.NewMigrator(k)

	err := m.MigrateFork(sdkCtx)
	require.NoError(t, err)

	// Read the test data file
	file, err := os.Open("dispute_module_state.json")
	require.NoError(t, err)
	defer file.Close()

	// Read the file contents
	fileData, err := io.ReadAll(file)
	require.NoError(t, err)
	fmt.Println("Length of file data: ", len(fileData))

	// Parse the JSON data
	var moduleState StreamModuleStateData
	err = json.Unmarshal(fileData, &moduleState)
	require.NoError(t, err)

	// Verify disputes were migrated correctly
	fmt.Println("Length of disputes: ", len(moduleState.Disputes))
	for _, entry := range moduleState.Disputes {
		dispute, err := k.Disputes.Get(ctx, entry.DisputeId)
		require.NoError(t, err)
		// Verify the dispute matches the test data
		require.Equal(t, *entry.Dispute, dispute)
	}

	// Verify votes were migrated correctly
	fmt.Println("Length of votes: ", len(moduleState.Votes))
	for _, entry := range moduleState.Votes {
		vote, err := k.Votes.Get(ctx, entry.DisputeId)
		require.NoError(t, err)
		require.Equal(t, *entry.Vote, vote)
	}

	// Verify voters were migrated correctly
	fmt.Println("Length of voters: ", len(moduleState.Voters))
	for _, entry := range moduleState.Voters {
		voter, err := k.Voter.Get(ctx, collections.Join(entry.DisputeId, entry.VoterAddress))
		require.NoError(t, err)
		require.Equal(t, *entry.Voter, voter)
	}

	// Verify reporters with delegators who voted were migrated correctly
	fmt.Println("Length of reporters with delegators who voted: ", len(moduleState.ReportersWithDelegatorsWhoVoted))
	for _, entry := range moduleState.ReportersWithDelegatorsWhoVoted {
		powerVoted, err := k.ReportersWithDelegatorsVotedBefore.Get(ctx, collections.Join(entry.ReporterAddress, entry.DisputeId))
		require.NoError(t, err)
		require.Equal(t, entry.VotedAmount, powerVoted)
	}

	// Verify block info was migrated correctly
	fmt.Println("Length of block info: ", len(moduleState.BlockInfo))
	for _, entry := range moduleState.BlockInfo {
		blockInfo, err := k.BlockInfo.Get(ctx, entry.HashId)
		require.NoError(t, err)
		require.Equal(t, *entry.BlockInfo, blockInfo)
	}

	// Verify dispute fee payer was migrated correctly
	fmt.Println("Length of dispute fee payer: ", len(moduleState.DisputeFeePayer))
	for _, entry := range moduleState.DisputeFeePayer {
		disputeFeePayer, err := k.DisputeFeePayer.Get(ctx, collections.Join(entry.DisputeId, entry.Payer))
		require.NoError(t, err)
		require.Equal(t, *entry.PayerInfo, disputeFeePayer)
	}

	// dust, err := k.Dust.Get(ctx)
	// require.NoError(t, err)
	// require.Equal(t, *moduleState.Dust, dust)

	// Verify vote counts were migrated correctly
	fmt.Println("Length of vote counts: ", len(moduleState.VoteCountsByGroup))
	for _, entry := range moduleState.VoteCountsByGroup {
		voteCounts, err := k.VoteCountsByGroup.Get(ctx, entry.DisputeId)
		require.NoError(t, err)
		require.Equal(t, disputetypes.StakeholderVoteCounts{Users: *entry.Users, Reporters: *entry.Reporters, Team: *entry.Team}, voteCounts)
	}
}
