package integration_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *IntegrationTestSuite) oracleKeeper() (queryClient types.QueryClient, msgServer types.MsgServer) {
	oracle.AppWiringSetup()
	types.RegisterQueryServer(s.queryHelper, s.oraclekeeper)
	queryClient = types.NewQueryClient(s.queryHelper)
	types.RegisterInterfaces(s.interfaceRegistry)

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
	userTips := s.oraclekeeper.GetUserTips(s.ctx, store, addr.String(), ethQueryData)
	require.Equal(userTips.Tipper, addr.String())
	require.Equal(userTips.TotalTipped, tips.Amount)

	// tip same query again
	_, err = msgServer.Tip(s.ctx, &msg)
	require.NoError(err)
	tips, _ = s.oraclekeeper.GetQueryTips(s.ctx, store, ethQueryData)
	require.Equal(tips.QueryData, ethQueryData)
	// tips should be 2x
	require.Equal(tip.Sub(twoPercent).Amount.Mul(sdk.NewInt(2)), tips.Amount.Amount)
	require.Equal(tips.TotalTips, tips.Amount)

	// tip different query
	_, err = msgServer.Tip(s.ctx, &types.MsgTip{QueryData: btcQueryData, Tipper: addr.String(), Amount: tip})
	require.NoError(err)
	tips, _ = s.oraclekeeper.GetQueryTips(s.ctx, store, btcQueryData)
	require.Equal(tips.QueryData, btcQueryData)
	require.Equal(tip.Sub(twoPercent), tips.Amount)
	require.Equal(tips.TotalTips, tips.Amount)
	userTips = s.oraclekeeper.GetUserTips(s.ctx, store, addr.String(), btcQueryData)
	require.Equal(userTips.Tipper, addr.String())
	require.Equal(userTips.TotalTipped, tips.Amount)
	fmt.Println("TestTipping passed")
}
