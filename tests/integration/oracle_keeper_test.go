package integration_test

import (
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
	_, msgServer := s.oracleKeeper()
	addr := s.newKeysWithTokens()
	tip := sdk.NewCoin(s.denom, sdk.NewInt(1000))
	twoPercent := sdk.NewCoin(s.denom, tip.Amount.Mul(sdk.NewInt(2)).Quo(sdk.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(s.ctx, &msg)
	s.NoError(err)
	store := s.oraclekeeper.TipStore(s.ctx)
	tips, _ := s.oraclekeeper.GetQueryTips(s.ctx, store, ethQueryData)
	s.Equal(tips.QueryData, ethQueryData)
	s.Equal(tip.Sub(twoPercent), tips.Amount)
	s.Equal(tips.TotalTips, tips.Amount)
	userTips := s.oraclekeeper.GetUserQueryTips(s.ctx, addr.String(), ethQueryData)
	s.Equal(userTips.Address, addr.String())
	s.Equal(userTips.Total, tips.Amount)
	userTips = s.oraclekeeper.GetUserTips(s.ctx, addr)
	s.Equal(userTips.Address, addr.String())
	s.Equal(userTips.Total, tips.Amount)

	// tip same query again
	_, err = msgServer.Tip(s.ctx, &msg)
	s.NoError(err)
	tips, _ = s.oraclekeeper.GetQueryTips(s.ctx, store, ethQueryData)
	s.Equal(tips.QueryData, ethQueryData)
	// tips should be 2x
	s.Equal(tip.Sub(twoPercent).Amount.Mul(sdk.NewInt(2)), tips.Amount.Amount)
	s.Equal(tips.TotalTips, tips.Amount)
	// total tips overall
	userTips = s.oraclekeeper.GetUserTips(s.ctx, addr)
	s.Equal(userTips.Address, addr.String())
	s.Equal(userTips.Total, tips.Amount)

	// tip different query
	_, err = msgServer.Tip(s.ctx, &types.MsgTip{QueryData: btcQueryData, Tipper: addr.String(), Amount: tip})
	s.NoError(err)
	tips, _ = s.oraclekeeper.GetQueryTips(s.ctx, store, btcQueryData)
	s.Equal(tips.QueryData, btcQueryData)
	s.Equal(tip.Sub(twoPercent), tips.Amount)
	s.Equal(tips.TotalTips, tips.Amount)
	userTips = s.oraclekeeper.GetUserQueryTips(s.ctx, addr.String(), btcQueryData)
	s.Equal(userTips.Address, addr.String())
	s.Equal(userTips.Total, tips.Amount)
	userTips = s.oraclekeeper.GetUserTips(s.ctx, addr)
	s.Equal(userTips.Address, addr.String())
	s.Equal(userTips.Total, tips.Amount.Add(tips.Amount).Add(tips.Amount))
}

func (s *IntegrationTestSuite) TestGetCurrentTip() {
	_, msgServer := s.oracleKeeper()
	addr := s.newKeysWithTokens()
	tip := sdk.NewCoin(s.denom, sdk.NewInt(1000))
	twoPercent := sdk.NewCoin(s.denom, tip.Amount.Mul(sdk.NewInt(2)).Quo(sdk.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(s.ctx, &msg)
	s.NoError(err)

	// Get current tip
	resp, err := s.oraclekeeper.GetCurrentTip(s.ctx, &types.QueryGetCurrentTipRequest{QueryData: ethQueryData})
	s.NoError(err)
	s.Equal(resp.Tips, &types.Tips{QueryData: ethQueryData, Amount: tip.Sub(twoPercent), TotalTips: tip.Sub(twoPercent)})
}

func (s *IntegrationTestSuite) TestGetUserTipTotal() {
	_, msgServer := s.oracleKeeper()
	addr := s.newKeysWithTokens()
	tip := sdk.NewCoin(s.denom, sdk.NewInt(1000))
	twoPercent := sdk.NewCoin(s.denom, tip.Amount.Mul(sdk.NewInt(2)).Quo(sdk.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(s.ctx, &msg)
	s.NoError(err)

	// Get current tip
	resp, err := s.oraclekeeper.GetUserTipTotal(s.ctx, &types.QueryGetUserTipTotalRequest{Tipper: addr.String(), QueryData: ethQueryData})
	s.NoError(err)
	s.Equal(resp.TotalTips, &types.UserTipTotal{Address: addr.String(), Total: tip.Sub(twoPercent)})
	// Check total tips without a given query data
	resp, err = s.oraclekeeper.GetUserTipTotal(s.ctx, &types.QueryGetUserTipTotalRequest{Tipper: addr.String()})
	s.NoError(err)
	s.Equal(resp.TotalTips, &types.UserTipTotal{Address: addr.String(), Total: tip.Sub(twoPercent)})
}

func (s *IntegrationTestSuite) TestSmallTip() {
	_, msgServer := s.oracleKeeper()
	addr := s.newKeysWithTokens()
	tip := sdk.NewCoin(s.denom, sdk.NewInt(10))
	twoPercent := sdk.NewCoin(s.denom, tip.Amount.Mul(sdk.NewInt(2)).Quo(sdk.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	accBalanceBefore := s.bankKeeper.GetBalance(s.ctx, addr, s.denom)
	modBalanceBefore := s.bankKeeper.GetBalance(s.ctx, authtypes.NewModuleAddress(types.ModuleName), s.denom)
	_, err := msgServer.Tip(s.ctx, &msg)
	s.NoError(err)
	accBalanceAfter := s.bankKeeper.GetBalance(s.ctx, addr, s.denom)
	modBalanceAfter := s.bankKeeper.GetBalance(s.ctx, authtypes.NewModuleAddress(types.ModuleName), s.denom)
	s.Equal(accBalanceBefore.Amount.Sub(tip.Amount), accBalanceAfter.Amount)
	s.Equal(modBalanceBefore.Amount.Add(tip.Amount).Sub(twoPercent.Amount), modBalanceAfter.Amount)

}
