package integration_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *IntegrationTestSuite) oracleKeeper() (queryClient types.QueryClient, msgServer types.MsgServer) {
	types.RegisterQueryServer(s.queryHelper, s.oraclekeeper)
	types.RegisterInterfaces(s.interfaceRegistry)
	queryClient = types.NewQueryClient(s.queryHelper)
	msgServer = keeper.NewMsgServerImpl(s.oraclekeeper)
	return
}

func (s *IntegrationTestSuite) TestTipping() {
	require := s.Require()
	_, msgServer := s.oracleKeeper()
	addr, denom := s.newKeysWithTokens()
	tip := sdk.NewCoin(denom, sdk.NewInt(1000))
	twoPercent := sdk.NewCoin(denom, tip.Amount.Mul(sdk.NewInt(2)).Quo(sdk.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(s.ctx, &msg)
	require.NoError(err)
	store := s.oraclekeeper.TipStore(s.ctx)
	tips, _ := s.oraclekeeper.GetQueryTips(s.ctx, store, ethQueryData)
	require.Equal(tips.QueryData, ethQueryData)
	require.Equal(tip.Sub(twoPercent), tips.Amount)
	require.Equal(tips.TotalTips, tips.Amount)
	userTips := s.oraclekeeper.GetUserQueryTips(s.ctx, store, addr.String(), ethQueryData)
	require.Equal(userTips.Address, addr.String())
	require.Equal(userTips.Total, tips.Amount)
	userTips = s.oraclekeeper.GetUserTips(s.ctx, store, addr)
	require.Equal(userTips.Address, addr.String())
	require.Equal(userTips.Total, tips.Amount)

	// tip same query again
	_, err = msgServer.Tip(s.ctx, &msg)
	require.NoError(err)
	tips, _ = s.oraclekeeper.GetQueryTips(s.ctx, store, ethQueryData)
	require.Equal(tips.QueryData, ethQueryData)
	// tips should be 2x
	require.Equal(tip.Sub(twoPercent).Amount.Mul(sdk.NewInt(2)), tips.Amount.Amount)
	require.Equal(tips.TotalTips, tips.Amount)
	// total tips overall
	userTips = s.oraclekeeper.GetUserTips(s.ctx, store, addr)
	require.Equal(userTips.Address, addr.String())
	require.Equal(userTips.Total, tips.Amount)

	// tip different query
	_, err = msgServer.Tip(s.ctx, &types.MsgTip{QueryData: btcQueryData, Tipper: addr.String(), Amount: tip})
	require.NoError(err)
	tips, _ = s.oraclekeeper.GetQueryTips(s.ctx, store, btcQueryData)
	require.Equal(tips.QueryData, btcQueryData)
	require.Equal(tip.Sub(twoPercent), tips.Amount)
	require.Equal(tips.TotalTips, tips.Amount)
	userTips = s.oraclekeeper.GetUserQueryTips(s.ctx, store, addr.String(), btcQueryData)
	require.Equal(userTips.Address, addr.String())
	require.Equal(userTips.Total, tips.Amount)
	userTips = s.oraclekeeper.GetUserTips(s.ctx, store, addr)
	require.Equal(userTips.Address, addr.String())
	require.Equal(userTips.Total, tips.Amount.Add(tips.Amount).Add(tips.Amount))
	fmt.Println("TestTipping passed")
}

func (s *IntegrationTestSuite) TestGetCurrentTip() {
	require := s.Require()
	_, msgServer := s.oracleKeeper()
	addr, denom := s.newKeysWithTokens()
	tip := sdk.NewCoin(denom, sdk.NewInt(1000))
	twoPercent := sdk.NewCoin(denom, tip.Amount.Mul(sdk.NewInt(2)).Quo(sdk.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(s.ctx, &msg)
	require.NoError(err)

	// Get current tip
	resp, err := s.oraclekeeper.GetCurrentTip(s.ctx, &types.QueryGetCurrentTipRequest{QueryData: ethQueryData})
	require.NoError(err)
	require.Equal(resp.Tips, &types.Tips{QueryData: ethQueryData, Amount: tip.Sub(twoPercent), TotalTips: tip.Sub(twoPercent)})
}

func (s *IntegrationTestSuite) TestGetUserTipTotal() {
	require := s.Require()
	_, msgServer := s.oracleKeeper()
	addr, denom := s.newKeysWithTokens()
	tip := sdk.NewCoin(denom, sdk.NewInt(1000))
	twoPercent := sdk.NewCoin(denom, tip.Amount.Mul(sdk.NewInt(2)).Quo(sdk.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(s.ctx, &msg)
	require.NoError(err)

	// Get current tip
	resp, err := s.oraclekeeper.GetUserTipTotal(s.ctx, &types.QueryGetUserTipTotalRequest{Tipper: addr.String(), QueryData: ethQueryData})
	require.NoError(err)
	require.Equal(resp.TotalTips, &types.UserTipTotal{Address: addr.String(), Total: tip.Sub(twoPercent)})
	// Check total tips without a given query data
	resp, err = s.oraclekeeper.GetUserTipTotal(s.ctx, &types.QueryGetUserTipTotalRequest{Tipper: addr.String()})
	require.NoError(err)
	require.Equal(resp.TotalTips, &types.UserTipTotal{Address: addr.String(), Total: tip.Sub(twoPercent)})
}

// Add test tiping 10 wei

func (s *IntegrationTestSuite) TestSmallTip() {
	require := s.Require()
	_, msgServer := s.oracleKeeper()
	addr, denom := s.newKeysWithTokens()
	tip := sdk.NewCoin(denom, sdk.NewInt(10))
	twoPercent := sdk.NewCoin(denom, tip.Amount.Mul(sdk.NewInt(2)).Quo(sdk.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	accBalanceBefore := s.bankKeeper.GetBalance(s.ctx, addr, denom)
	modBalanceBefore := s.bankKeeper.GetBalance(s.ctx, authtypes.NewModuleAddress(types.ModuleName), denom)
	_, err := msgServer.Tip(s.ctx, &msg)
	require.NoError(err)
	accBalanceAfter := s.bankKeeper.GetBalance(s.ctx, addr, denom)
	modBalanceAfter := s.bankKeeper.GetBalance(s.ctx, authtypes.NewModuleAddress(types.ModuleName), denom)
	require.Equal(accBalanceBefore.Amount.Sub(tip.Amount), accBalanceAfter.Amount)
	require.Equal(modBalanceBefore.Amount.Add(tip.Amount).Sub(twoPercent.Amount), modBalanceAfter.Amount)

}
