package keeper_test

import (
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/dispute/keeper"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/colltest"
)

func (s *KeeperTestSuite) TestIndexesList_Disputes() {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	im := keeper.NewDisputesIndex(schema)
	index := im.IndexesList()
	require.NotNil(s.T(), index)
}

func (s *KeeperTestSuite) TestNewDisputesIndex() {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)
	disputesIndex := keeper.NewDisputesIndex(schema)
	require.NotNil(s.T(), disputesIndex)
	require.NotNil(s.T(), disputesIndex.OpenDisputes)
	require.NotNil(s.T(), disputesIndex.DisputeByReporter)
}

func (s *KeeperTestSuite) TestIndexesList_Voters() {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	votersIndex := keeper.NewVotersIndex(schema)
	index := votersIndex.IndexesList()
	require.NotNil(s.T(), index)
}

// func (k *KeeperTestSuite) TestNewDisputesIndex() {
// 	sk, ctx := colltest.MockStore()
// 	fmt.Println("sk: ", sk)
// 	fmt.Println("ctx: ", ctx)
// 	fmt.Println("k.ctx: ", k.ctx)
// 	require.NotNil(k.T(), sk)
// 	ctx = sk.NewStoreContext()
// 	fmt.Println("ctx: ", ctx)
// 	schema := collections.NewSchemaBuilder(sk)
// 	im := keeper.NewDisputesIndex(schema)

// 	store := colltest.StoreService.OpenKVStore(*sk, ctx)
// 	fmt.Println("store: ", store)
// 	require.NotNil(k.T(), store)

// 	dispute := k.dispute()
// 	k.NoError(k.disputeKeeper.Disputes.Set(ctx, dispute.DisputeId, dispute))
// 	dispute, err := k.disputeKeeper.Disputes.Get(ctx, dispute.DisputeId)
// 	k.NoError(err)
// 	fmt.Println("dispute: ", dispute)

// 	iter, err := im.OpenDisputes.MatchExact(ctx, true)
// 	fmt.Println("iter: ", iter)
// 	k.NoError(err)
// 	k.True(iter.Valid())

// 	// Assert that DisputeByReporter index is created correctly
// 	expectedDisputeByReporterIndexName := "dispute_by_reporter"
// 	expectedDisputeByReporterIndexPrefix := types.DisputesByReporterIndexPrefix
// 	expectedDisputeByReporterIndex := indexes.NewMulti(
// 		schema, expectedDisputeByReporterIndexPrefix, expectedDisputeByReporterIndexName,
// 		collections.BytesKey, collections.Uint64Key,
// 		func(k uint64, dispute types.Dispute) ([]byte, error) {
// 			reporterKey := fmt.Sprintf("%s:%x", dispute.ReportEvidence.Reporter, dispute.HashId)
// 			return []byte(reporterKey), nil
// 		},
// 	)
// 	k.Equal(expectedDisputeByReporterIndex, im.DisputeByReporter)

// 	// Assert that OpenDisputes index is created correctly
// 	expectedOpenDisputesIndexName := "open_disputes"
// 	expectedOpenDisputesIndexPrefix := []byte("open_disputes")
// 	expectedOpenDisputesIndex := indexes.NewMulti(
// 		schema, expectedOpenDisputesIndexPrefix, expectedOpenDisputesIndexName,
// 		collections.BoolKey, collections.Uint64Key,
// 		func(k uint64, dispute types.Dispute) (bool, error) {
// 			return dispute.Open, nil
// 		},
// 	)
// 	k.Equal(expectedOpenDisputesIndex, im.OpenDisputes)
// }

func (s *KeeperTestSuite) TestNewVotersIndex() {
	storeService, _ := colltest.MockStore()
	schema := collections.NewSchemaBuilder(storeService)

	votersIndex := keeper.NewVotersIndex(schema)
	require.NotNil(s.T(), votersIndex)
}
