package keeper_test

func (s *KeeperTestSuite) TestQueryGetDataBefore() {
	// require := s.Require()

	// s.TestSubmitValue()
	// queryData := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	// queryDataBytes, err := hex.DecodeString(queryData)
	// fmt.Println("queryDataBytes: ", queryDataBytes)
	// require.Nil(err)
	// queryIdBytes := crypto.Keccak256(queryDataBytes)
	// fmt.Println("queryIdBytes: ", queryIdBytes)
	// queryId := hex.EncodeToString(queryIdBytes)
	// fmt.Println("queryId: ", queryId)
	// test := crypto.Keccak256([]byte(queryData))
	// fmt.Println("test: ", test)
	// queryGetDataBeforeRequest := &types.QueryGetDataBeforeRequest{
	// 	QueryId:   queryId,
	// 	Timestamp: s.ctx.HeaderInfo().Time.Unix() + 100,
	// }
	// s.ctx = testutil.WithBlockTime(s.ctx,s.ctx.HeaderInfo().Time.Add(101))
	// data, err := s.oracleKeeper.GetDataBefore(s.ctx, queryGetDataBeforeRequest)
	// require.Nil(err)
	// fmt.Println(data)
}
