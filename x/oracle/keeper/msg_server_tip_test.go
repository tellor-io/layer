package keeper_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	// "cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestTip() {
	require := s.Require()
	ctx := s.ctx
	regK := s.registryKeeper
	bk := s.bankKeeper
	msgServer := s.msgServer

	// note: previous tests got moved to ValidateBasic

	tipper := sample.AccAddressBytes()
	// query needs initialized, expiration after block time, set first tip
	amount := sdk.NewCoin("loya", math.NewInt(10*1e6))
	regK.On("GetSpec", ctx, "SpotPrice").Return(spotSpec, nil)
	bk.On("SendCoinsFromAccountToModule", ctx, tipper, types.ModuleName, sdk.NewCoins(amount)).Return(nil).Once()
	twoPercent := amount.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100))
	burnCoin := sdk.NewCoin(amount.Denom, twoPercent)
	bk.On("BurnCoins", ctx, types.ModuleName, sdk.NewCoins(burnCoin)).Return(nil).Once()
	queryBytes, err := utils.QueryBytesFromString(queryData)
	require.NoError(err)
	tipRes, err := msgServer.Tip(ctx, &types.MsgTip{
		Amount:    amount,
		Tipper:    tipper.String(),
		QueryData: queryBytes,
	})
	require.NoError(err)
	require.NotNil(tipRes)
}
