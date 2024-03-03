package keeper_test

//	func createNCommits(keeper *Keeper, ctx sdk.Context, n int) []types.CommitReport {
//		items := make([]types.CommitReport, n)
//		for i := range items {
//			items[i].Report =
//			keeper.SetCommitReport(ctx, items[i])
//		}
//		return items
//	}
func (s *KeeperTestSuite) TestSetCommitReport() {
	// require := s.Require()

	// queryData := "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	// queryDataBytes, err := hex.DecodeString(queryData[2:])
	// require.Nil(err)
	// queryIdBytes := crypto.Keccak256(queryDataBytes)
	// queryId := hex.EncodeToString(queryIdBytes)
	// block := s.ctx.BlockHeight()
	// value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	// valueDecoded, err := hex.DecodeString(value)
	// require.Nil(err)
	// salt, err := utils.Salt(32)
	// require.Nil(err)
	// hash := utils.CalculateCommitment(string(valueDecoded), salt)

	// commit := types.CommitReport{
	// 	Report: &types.Commit{
	// 		Creator: Addr.String(),
	// 		QueryId: queryIdBytes,
	// 		Hash:    hash,
	// 	},
	// 	Block: block,
	// }

	// s.oracleKeeper.SetCommitReport(s.ctx, Addr, &commit)

	// oracle := codectypes.NewInterfaceRegistry()
	// cdc := codec.NewProtoCodec(oracle)
	// // Verify commit report is stored correctly
	// store := s.oracleKeeper.CommitStore(s.ctx)
	// key := append(Addr, queryId...)
	// bz := store.Get(key)
	// var storedCommit types.CommitReport
	// cdc.MustUnmarshal(bz, &storedCommit)
	// require.Equal(commit, storedCommit)

	// // Verify commit report is appended to block reports
	// blockKey := types.NumKey(block)
	// bz = store.Get(blockKey)
	// var blockReports types.CommitsByHeight
	// cdc.MustUnmarshal(bz, &blockReports)
	// require.Contains(s.T(), blockReports.Commits, commit.Report)

	// // Verify last block's reports are deleted
	// lastBlockKey := types.NumKey(block - 1)
	// require.Nil(s.T(), store.Get(lastBlockKey))
}
