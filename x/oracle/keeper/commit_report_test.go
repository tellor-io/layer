package keeper_test

// import (
// 	"fmt"
// 	"testing"

// 	sdk "github.com/cosmos/cosmos-sdk/types"
// 	"github.com/stretchr/testify/require"

// 	"github.com/tellor-io/layer/x/oracle/keeper"
// 	"github.com/tellor-io/layer/x/oracle/types"
// )

// func TestSetCommitReport(t *testing.T) {

// 	// Create a new context
// 	ctx := sdk.Context{}

// 	// Create test data
// 	reporter := sdk.AccAddress([]byte("reporter"))
// 	queryID := []byte("queryID")
// 	commit := &types.CommitReport{
// 		Report: &types.Commit{
// 			Creator: "A",
// 			QueryId: queryID,
// 			//Signature: ,
// 		},
// 		Block: 1,
// 	}

// 	// Call the function under test
// 	oracleKeeper.SetCommitReport(ctx, reporter, commit)

// 	// Retrieve the commit report from the store
// 	store := oracleKeeper.CommitStore(ctx)
// 	retrievedCommit := store.Get(append(reporter, queryID...))
// 	fmt.Println("retreived commit:", retrievedCommit)

// 	// Assert that the retrieved commit report matches the expected commit report
// 	//require.Equal(t, commit, cdc.MustUnmarshal(retrievedCommit))

// 	// Retrieve the block reports from the store
// 	blockKey := types.BlockKey(commit.Block)
// 	blockReports := store.Get(blockKey)
// 	fmt.Println("block reports:", blockReports)

// 	// Unmarshal the block reports
// 	var retrievedBlockReports types.CommitsByHeight
// 	//oracleKeeper.Cdc.MustUnmarshal(blockReports, &retrievedBlockReports)

// 	// Assert that the commit report is appended to the block reports
// 	require.Contains(t, retrievedBlockReports.Commits, commit.Report)

// 	// Retrieve the last block reports from the store
// 	lastBlockReports := store.Get(types.BlockKey(commit.Block - 1))

// 	// Assert that the last block reports are deleted
// 	require.Empty(t, lastBlockReports)
// }
