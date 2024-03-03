package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestGetCurrentTip() {
	require := s.Require()

	// tip trb
	amount := sdk.NewCoin("loya", math.NewInt(1000))
	msg := types.MsgTip{
		Tipper:    Addr.String(),
		QueryData: trbQueryData,
		Amount:    amount,
	}
	_, err := s.msgServer.Tip(s.ctx, &msg)
	require.Nil(err)

	// get trb tips
	tipRequest := &types.QueryGetCurrentTipRequest{
		QueryData: trbQueryData,
	}
	trbTips, err := s.oracleKeeper.GetCurrentTip(s.ctx, tipRequest)
	require.Nil(err)
	require.Equal(trbQueryData, trbTips.Tips.QueryData)
	twoPercent := sdk.NewCoin("loya", amount.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	require.Equal(amount.Sub(twoPercent), trbTips.Tips.Amount)

	// get btc tips (none)
	btcTips, err := s.oracleKeeper.GetCurrentTip(s.ctx, &types.QueryGetCurrentTipRequest{QueryData: btcQueryData})
	require.Nil(err)
	require.Equal(btcQueryData, btcTips.Tips.QueryData)
	zeroAmount := sdk.NewCoin("loya", math.NewInt(0))
	require.Equal(zeroAmount, btcTips.Tips.Amount)
	require.Equal(btcTips.Tips.Amount.Denom, "loya")

	//tip trb again
	amount = sdk.NewCoin("loya", math.NewInt(10000))
	msg = types.MsgTip{
		Tipper:    Addr.String(),
		QueryData: trbQueryData,
		Amount:    amount,
	}
	_, err = s.msgServer.Tip(s.ctx, &msg)
	require.Nil(err)
	trbTips2, err := s.oracleKeeper.GetCurrentTip(s.ctx, tipRequest)
	require.Nil(err)
	trbTipTotal := sdk.NewCoin("loya", (math.NewInt(10780)))
	require.Equal(trbTipTotal, trbTips2.Tips.Amount)
	require.Equal(trbQueryData, trbTips2.Tips.QueryData)
	require.Equal(trbTips2.Tips.Amount.Denom, "loya")

}

func (s *KeeperTestSuite) TestGetCurrentTipInvalidRequest() {
	_, err := s.oracleKeeper.GetCurrentTip(s.ctx, nil)
	s.ErrorContains(err, "invalid request")
}
