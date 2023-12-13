package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestTransfer() {
	require := s.Require()
	require.NotNil(s.msgServer)
	require.NotNil(s.ctx)

	//addr1 := sdk.AccAddress([]byte("addr1"))
	//transferResult := s.oracleKeeper.transfer(s.ctx, account1, sdk.NewInt64Coin("ETH", 100))
}

// func (s *KeeperTestSuite) TestSetTip() {
// 	require := s.Require()
// 	require.NotNil(s.msgServer)
// 	require.NotNil(s.ctx)

// 	addr1 := sdk.AccAddress([]byte("addr1"))
// 	queryData := "testQuery"
// 	tip := sdk.NewInt64Coin("ETH", 100)

// 	// Set up the test environment
// 	tipStore := s.oracleKeeper.TipStore(s.ctx)
// 	s.oracleKeeper.SetTotalTips(s.ctx, tipStore, tip)
// 	s.oracleKeeper.SetQueryTips(s.ctx, tipStore, queryData, tip)
// 	s.oracleKeeper.SetTipperTipsForQuery(s.ctx, tipStore, addr1.String(), queryData, tip)
// 	s.oracleKeeper.SetTipperTotalTips(s.ctx, tipStore, addr1, tip)

// 	// Retrieve and verify the stored values
// 	totalTips := s.oracleKeeper.GetTotalTips(s.ctx)
// 	require.Equal(s, tip, totalTips)

// 	queryTips, err := s.oracleKeeper.GetQueryTips(s.ctx, tipStore, queryData)
// 	require.Equal(s, tip, queryTips)
// 	require.Nil(err)

// 	tipperTips := s.oracleKeeper.GetTipperTipsForQuery(s.ctx, tipStore, addr1.String(), queryData)
// 	require.Equal(s, tip, tipperTips)

// 	tipperTotalTips := s.oracleKeeper.GetTotalTips(s.ctx)
// 	require.Equal(s, tip, tipperTotalTips)
// }

func (s *KeeperTestSuite) TestSetQueryTips() {
	require := s.Require()
	require.NotNil(s.msgServer)
	require.NotNil(s.ctx)

	queryData := "testQuery"
	tip := sdk.NewInt64Coin("ETH", 100)

	// Set up the test environment
	tipStore := s.oracleKeeper.TipStore(s.ctx)
	s.oracleKeeper.SetQueryTips(s.ctx, tipStore, queryData, tip)

	// Retrieve and verify the stored values
	queryTips, err := s.oracleKeeper.GetQueryTips(s.ctx, tipStore, queryData)
	require.Equal(s, tip, queryTips)
	require.Nil(err)
}
