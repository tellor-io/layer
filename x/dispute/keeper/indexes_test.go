package keeper_test

// func (k *KeeperTestSuite) TestNewDisputesIndex() {
// 	sk, _ := colltest.MockStore()
// 	schema := collections.NewSchemaBuilder(sk)

//		im := keeper.NewDisputesIndex(schema)
//		im.DisputeByReporter.
//	}
func (k *KeeperTestSuite) TestNewDisputesIndex() {
	// sk, _ := colltest.MockStore()
	// schema := collections.NewSchemaBuilder(sk)
	// im := keeper.NewDisputesIndex(schema)

	// dispute := k.dispute()
	// k.NoError(k.disputeKeeper.Disputes.Set(k.ctx, dispute.DisputeId, dispute))

	// iter, err := im.OpenDisputes.MatchExact(k.ctx, true)
	// k.NoError(err)
	// k.True(iter.Valid())
	// // Assert that DisputeByReporter index is created correctly
	// expectedDisputeByReporterIndexName := "dispute_by_reporter"
	// expectedDisputeByReporterIndexPrefix := types.DisputesByReporterIndexPrefix
	// expectedDisputeByReporterIndex := indexes.NewMulti(
	// 	schema, expectedDisputeByReporterIndexPrefix, expectedDisputeByReporterIndexName,
	// 	collections.BytesKey, collections.Uint64Key,
	// 	func(k uint64, dispute types.Dispute) ([]byte, error) {
	// 		reporterKey := fmt.Sprintf("%s:%x", dispute.ReportEvidence.Reporter, dispute.HashId)
	// 		return []byte(reporterKey), nil
	// 	},
	// )
	// k.Equal(expectedDisputeByReporterIndex, im.DisputeByReporter)

	// Assert that OpenDisputes index is created correctly
	// expectedOpenDisputesIndexName := "open_disputes"
	// expectedOpenDisputesIndexPrefix := []byte("open_disputes")
	// expectedOpenDisputesIndex := indexes.NewMulti(
	// 	schema, expectedOpenDisputesIndexPrefix, expectedOpenDisputesIndexName,
	// 	collections.BoolKey, collections.Uint64Key,
	// 	func(k uint64, dispute types.Dispute) (bool, error) {
	// 		return dispute.Open, nil
	// 	},
	// )
	// k.Equal(expectedOpenDisputesIndex, im.OpenDisputes)
}
