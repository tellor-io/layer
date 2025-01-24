package keeper_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	regtypes "github.com/tellor-io/layer/x/registry/types"

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

	tipper := sample.AccAddressBytes()
	amount := sdk.NewCoin("btc", math.NewInt(10*1e6))
	// s.bankKeeper.On("SendCoinsFromAccountToModule", ctx, tipper, "oracle", sdk.NewCoins(amount)).Return(nil)
	tipRes, err := msgServer.Tip(ctx, &types.MsgTip{
		Amount:    amount,
		Tipper:    tipper.String(),
		QueryData: []byte(queryData),
	})
	require.Error(err)
	require.Nil(tipRes)

	// amount is zero
	amount = sdk.NewCoin("loya", math.NewInt(0))
	tipRes, err = msgServer.Tip(ctx, &types.MsgTip{
		Amount:    amount,
		Tipper:    tipper.String(),
		QueryData: []byte(queryData),
	})
	require.Error(err)
	require.EqualError(err, types.ErrNotEnoughTip.Error())
	require.Nil(tipRes)

	// amount is to large
	amount = sdk.NewCoin("loya", math.NewInt(100_000_000))
	tipRes, err = msgServer.Tip(ctx, &types.MsgTip{
		Amount:    amount,
		Tipper:    tipper.String(),
		QueryData: []byte(queryData),
	})
	require.Error(err)
	require.EqualError(err, types.ErrTipExceedsMax.Error())
	require.Nil(tipRes)

	// amount is negative
	require.Panics(func() {
		tipRes, err = msgServer.Tip(ctx, &types.MsgTip{
			Amount:    sdk.NewCoin("loya", math.NewInt(-10*1e6)),
			Tipper:    tipper.String(),
			QueryData: []byte(queryData),
		})
	})

	// bad tipper address
	badTipperAddr := "bad_tipper_address"
	require.Panics(func() {
		tipRes, err = msgServer.Tip(ctx, &types.MsgTip{
			Amount:    amount,
			Tipper:    badTipperAddr,
			QueryData: []byte(queryData),
		})
	})

	// query needs initialized, expiration after block time, set first tip
	amount = sdk.NewCoin("loya", math.NewInt(10*1e6))
	genesisDataSpecs := regtypes.GenesisDataSpec()
	var spotPriceSpec regtypes.DataSpec
	for i := 0; i < len(genesisDataSpecs); i++ {
		if genesisDataSpecs[i].QueryType == "spotprice" {
			spotPriceSpec = genesisDataSpecs[i]
			break
		}
	}
	regK.On("GetSpec", ctx, "SpotPrice").Return(spotPriceSpec, nil)
	bk.On("SendCoinsFromAccountToModule", ctx, tipper, types.ModuleName, sdk.NewCoins(amount)).Return(nil).Once()
	twoPercent := amount.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100))
	burnCoin := sdk.NewCoin(amount.Denom, twoPercent)
	bk.On("BurnCoins", ctx, types.ModuleName, sdk.NewCoins(burnCoin)).Return(nil).Once()
	queryBytes, err := utils.QueryBytesFromString(queryData)
	require.NoError(err)
	tipRes, err = msgServer.Tip(ctx, &types.MsgTip{
		Amount:    amount,
		Tipper:    tipper.String(),
		QueryData: queryBytes,
	})
	require.NoError(err)
	require.NotNil(tipRes)
}
